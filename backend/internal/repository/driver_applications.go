package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/domain"
)

type DriverApplications struct{ pool *pgxpool.Pool }

func NewDriverApplications(pool *pgxpool.Pool) *DriverApplications { return &DriverApplications{pool} }

const applicationColumns = `id::text,user_id::text,contact_phone,first_name,last_name,pin_code_encrypted,car_make,car_model,car_year,car_plate,car_color,seats,status,reviewed_by::text,reviewed_at,reject_reason,created_at`

func scanApplication(row pgx.Row) (domain.DriverApplication, error) {
	var app domain.DriverApplication
	err := row.Scan(&app.ID, &app.UserID, &app.ContactPhone, &app.FirstName, &app.LastName, &app.PINCodeEncrypted, &app.CarMake, &app.CarModel, &app.CarYear, &app.CarPlate, &app.CarColor, &app.Seats, &app.Status, &app.ReviewedBy, &app.ReviewedAt, &app.RejectReason, &app.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return app, apperrors.ErrNotFound
	}
	return app, err
}

func (r *DriverApplications) Create(ctx context.Context, app domain.DriverApplication) (domain.DriverApplication, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.DriverApplication{}, err
	}
	defer tx.Rollback(ctx)
	// Use the same lock order as review: applicant, then application. This also
	// prevents a stale client JWT from applying after approval grants driver.
	var active bool
	err = tx.QueryRow(ctx, `SELECT is_active FROM users WHERE id=$1 FOR UPDATE`, app.UserID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !active) {
		return domain.DriverApplication{}, apperrors.ErrUnauthorized
	}
	if err != nil {
		return domain.DriverApplication{}, err
	}
	var driver, pending bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=$1 AND role='driver'),EXISTS(SELECT 1 FROM driver_applications WHERE user_id=$1 AND status='pending')`, app.UserID).Scan(&driver, &pending)
	if err != nil {
		return domain.DriverApplication{}, err
	}
	if driver {
		return domain.DriverApplication{}, apperrors.ErrAlreadyDriver
	}
	if pending {
		return domain.DriverApplication{}, apperrors.ErrApplicationPending
	}
	created, err := scanApplication(tx.QueryRow(ctx, `INSERT INTO driver_applications(id,user_id,contact_phone,first_name,last_name,pin_code_encrypted,car_make,car_model,car_year,car_plate,car_color,seats) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+applicationColumns, app.ID, app.UserID, app.ContactPhone, app.FirstName, app.LastName, app.PINCodeEncrypted, app.CarMake, app.CarModel, app.CarYear, app.CarPlate, app.CarColor, app.Seats))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "driver_applications_one_pending_per_user" {
			return domain.DriverApplication{}, apperrors.ErrApplicationPending
		}
		return domain.DriverApplication{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.DriverApplication{}, err
	}
	return created, nil
}

func (r *DriverApplications) Latest(ctx context.Context, userID string) (domain.DriverApplication, error) {
	return scanApplication(r.pool.QueryRow(ctx, `SELECT `+applicationColumns+` FROM driver_applications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, userID))
}
func (r *DriverApplications) ByID(ctx context.Context, id string) (domain.DriverApplication, error) {
	return scanApplication(r.pool.QueryRow(ctx, `SELECT `+applicationColumns+` FROM driver_applications WHERE id=$1`, id))
}
func (r *DriverApplications) List(ctx context.Context, status string, limit, offset int) ([]domain.DriverApplication, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+applicationColumns+` FROM driver_applications WHERE status=$1 ORDER BY created_at,id LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.DriverApplication, 0)
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, app)
	}
	return items, rows.Err()
}

func (r *DriverApplications) Review(ctx context.Context, id, adminID, action, reason string) (domain.DriverApplication, error) {
	if action != "approve" && action != "reject" {
		return domain.DriverApplication{}, apperrors.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.DriverApplication{}, err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `SELECT user_id::text FROM driver_applications WHERE id=$1`, id).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DriverApplication{}, apperrors.ErrNotFound
	}
	if err != nil {
		return domain.DriverApplication{}, err
	}
	var active bool
	if err = tx.QueryRow(ctx, `SELECT is_active FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&active); err != nil {
		return domain.DriverApplication{}, err
	}
	if !active && action == "approve" {
		return domain.DriverApplication{}, apperrors.ErrApplicantInactive
	}
	app, err := scanApplication(tx.QueryRow(ctx, `SELECT `+applicationColumns+` FROM driver_applications WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return domain.DriverApplication{}, err
	}
	if app.Status != "pending" {
		return domain.DriverApplication{}, apperrors.ErrApplicationReviewed
	}
	status := "rejected"
	if action == "approve" {
		status, reason = "approved", ""
		if _, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,'driver') ON CONFLICT DO NOTHING`, userID); err != nil {
			return domain.DriverApplication{}, err
		}
	}
	app, err = scanApplication(tx.QueryRow(ctx, `UPDATE driver_applications SET status=$2,reviewed_by=$3,reviewed_at=now(),reject_reason=NULLIF($4,'') WHERE id=$1 RETURNING `+applicationColumns, id, status, adminID, reason))
	if err != nil {
		return domain.DriverApplication{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.DriverApplication{}, err
	}
	return app, nil
}
