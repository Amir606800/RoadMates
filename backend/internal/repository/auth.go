package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/service"
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type authQueries struct{ db database }
type Auth struct {
	authQueries
	pool *pgxpool.Pool
}
type authTransaction struct{ authQueries }

func NewAuth(pool *pgxpool.Pool) *Auth { return &Auth{authQueries{pool}, pool} }

const userColumns = `u.id::text,u.email,u.password_hash,u.first_name,u.last_name,u.phone,u.created_at,u.updated_at,u.is_active,
    COALESCE((SELECT array_agg(role ORDER BY role) FROM user_roles WHERE user_id=u.id),'{}'::text[])`

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Phone, &u.CreatedAt, &u.UpdatedAt, &u.IsActive, &u.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, apperrors.ErrNotFound
	}
	return u, err
}

func (q authQueries) UserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(q.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id=$1`, id))
}
func (q authQueries) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(q.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.email=$1`, email))
}

func (r *Auth) CreateUser(ctx context.Context, input domain.NewUser, refresh domain.RefreshToken) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,first_name,last_name,phone) VALUES($1,$2,$3,$4,$5) RETURNING id::text`, input.Email, input.PasswordHash, input.FirstName, input.LastName, input.Phone).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return domain.User{}, apperrors.ErrEmailTaken
		}
		return domain.User{}, err
	}
	q := authQueries{tx}
	refresh.UserID = id
	if err := q.InsertRefresh(ctx, refresh); err != nil {
		return domain.User{}, err
	}
	user, err := q.UserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (q authQueries) InsertRefresh(ctx context.Context, t domain.RefreshToken) error {
	_, err := q.db.Exec(ctx, `INSERT INTO refresh_tokens(id,user_id,family_id,token_hash,expires_at,user_agent,ip) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::inet)`, t.ID, t.UserID, t.FamilyID, t.TokenHash, t.ExpiresAt, t.UserAgent, t.IP)
	return err
}

func (r *Auth) Transaction(ctx context.Context, callback func(service.AuthTx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = callback(authTransaction{authQueries{tx}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (q authTransaction) LockRefresh(ctx context.Context, hash string) (domain.RefreshToken, error) {
	var family string
	err := q.db.QueryRow(ctx, `SELECT family_id::text FROM refresh_tokens WHERE token_hash=$1`, hash).Scan(&family)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, apperrors.ErrUnauthorized
	}
	if err != nil {
		return domain.RefreshToken{}, err
	}
	// A transaction-scoped lock serializes rotation/reuse/logout for the whole
	// family, including requests using different generations of its tokens.
	if _, err := q.db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text,0))`, family); err != nil {
		return domain.RefreshToken{}, err
	}
	var t domain.RefreshToken
	err = q.db.QueryRow(ctx, `SELECT id::text,user_id::text,family_id::text,token_hash,expires_at,revoked_at,user_agent,COALESCE(host(ip),'') FROM refresh_tokens WHERE token_hash=$1 FOR UPDATE`, hash).Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.UserAgent, &t.IP)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, apperrors.ErrUnauthorized
	}
	return t, err
}

func (q authTransaction) ReplaceRefresh(ctx context.Context, oldID, newID string) error {
	_, err := q.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now(),replaced_by=$2 WHERE id=$1`, oldID, newID)
	return err
}
func (q authTransaction) RevokeFamily(ctx context.Context, family string) error {
	_, err := q.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE family_id=$1`, family)
	return err
}

func (r *Auth) BootstrapAdmin(ctx context.Context, input domain.NewUser) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,first_name,last_name,phone) VALUES($1,$2,$3,$4,$5) ON CONFLICT(email) DO NOTHING RETURNING id::text`, input.Email, input.PasswordHash, input.FirstName, input.LastName, input.Phone).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,'admin')`, id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *Auth) CleanupRefresh(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < now() - interval '7 days'`)
	return err
}
