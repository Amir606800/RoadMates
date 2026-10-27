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

type Rides struct{ pool *pgxpool.Pool }

func NewRides(pool *pgxpool.Pool) *Rides { return &Rides{pool} }

const rideSelect = `SELECT r.id::text,r.driver_id::text,u.first_name,u.last_name,
 f.id,f.name_az,f.slug,f.requires_permit,t.id,t.name_az,t.slug,t.requires_permit,
 r.departure_at,r.seats_total,r.seats_available,(r.price_azn*100)::bigint,r.notes,r.status,r.created_at
 FROM rides r JOIN users u ON u.id=r.driver_id
 JOIN districts f ON f.id=r.from_district_id JOIN districts t ON t.id=r.to_district_id `

func scanRide(row pgx.Row) (domain.Ride, error) {
	var r domain.Ride
	err := row.Scan(&r.ID, &r.DriverID, &r.DriverFirstName, &r.DriverLastName,
		&r.From.ID, &r.From.NameAZ, &r.From.Slug, &r.From.RequiresPermit, &r.To.ID, &r.To.NameAZ, &r.To.Slug, &r.To.RequiresPermit,
		&r.DepartureAt, &r.SeatsTotal, &r.SeatsAvailable, &r.PriceCents, &r.Notes, &r.Status, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = apperrors.ErrNotFound
	}
	return r, err
}
func (r *Rides) Active(ctx context.Context, driverID string) (domain.Ride, error) {
	return scanRide(r.pool.QueryRow(ctx, rideSelect+`WHERE r.driver_id=$1 AND r.status IN ('published','in_progress')`, driverID))
}
func (r *Rides) ByID(ctx context.Context, id string) (domain.Ride, error) {
	return scanRide(r.pool.QueryRow(ctx, rideSelect+`WHERE r.id=$1 AND u.is_active`, id))
}
func (r *Rides) Create(ctx context.Context, ride domain.Ride) (domain.Ride, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Ride{}, err
	}
	defer tx.Rollback(ctx)
	// Lock both active districts until insertion so a concurrent deactivation
	// cannot admit a new ride using an inactive district.
	rows, err := tx.Query(ctx, `SELECT id FROM districts WHERE id IN ($1,$2) AND is_active ORDER BY id FOR SHARE`, ride.From.ID, ride.To.ID)
	if err != nil {
		return domain.Ride{}, err
	}
	count := 0
	for rows.Next() {
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return domain.Ride{}, err
	}
	if count != 2 {
		return domain.Ride{}, apperrors.ErrInvalidInput
	}
	err = tx.QueryRow(ctx, `INSERT INTO rides(driver_id,from_district_id,to_district_id,departure_at,seats_total,seats_available,price_azn,notes)
 VALUES($1,$2,$3,$4,$5,$5,$6::bigint/100.0,$7) RETURNING id::text`, ride.DriverID, ride.From.ID, ride.To.ID, ride.DepartureAt, ride.SeatsTotal, ride.PriceCents, ride.Notes).Scan(&ride.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "rides_one_active_per_driver" {
		return domain.Ride{}, apperrors.ErrDriverHasActiveRide
	}
	if err != nil {
		return domain.Ride{}, err
	}
	result, err := scanRide(tx.QueryRow(ctx, rideSelect+`WHERE r.id=$1`, ride.ID))
	if err != nil {
		return domain.Ride{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Ride{}, err
	}
	return result, nil
}
func (r *Rides) Search(ctx context.Context, filter domain.RideSearch) ([]domain.Ride, error) {
	rows, err := r.pool.Query(ctx, rideSelect+rideSearchWhere, filter.FromID, filter.ToID, filter.Date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Ride, 0)
	for rows.Next() {
		item, err := scanRide(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const rideSearchWhere = `WHERE r.status='published' AND r.seats_available>0 AND r.departure_at>now()
 AND u.is_active AND f.is_active AND t.is_active
 AND ($1::int=0 OR f.id=$1) AND ($2::int=0 OR t.id=$2)
 AND ($3::text='' OR (r.departure_at >= (NULLIF($3,'')::date::timestamp AT TIME ZONE 'Asia/Baku')
 AND r.departure_at < ((NULLIF($3,'')::date+1)::timestamp AT TIME ZONE 'Asia/Baku')))
 ORDER BY r.departure_at,r.id`

func (r *Rides) SearchIDs(ctx context.Context, filter domain.RideSearch) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT r.id::text FROM rides r JOIN users u ON u.id=r.driver_id JOIN districts f ON f.id=r.from_district_id JOIN districts t ON t.id=r.to_district_id `+rideSearchWhere, filter.FromID, filter.ToID, filter.Date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Rides) SearchByIDs(ctx context.Context, ids []string) ([]domain.Ride, error) {
	if len(ids) == 0 {
		return []domain.Ride{}, nil
	}
	rows, err := r.pool.Query(ctx, rideSelect+`WHERE r.id=ANY($1::uuid[]) AND r.status='published' AND r.seats_available>0 AND r.departure_at>now() AND u.is_active AND f.is_active AND t.is_active ORDER BY r.departure_at,r.id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Ride, 0)
	for rows.Next() {
		item, err := scanRide(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Rides) Finish(ctx context.Context, id, driverID, status string) (domain.Ride, error) {
	if status != "cancelled" && status != "completed" {
		return domain.Ride{}, apperrors.ErrInvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Ride{}, err
	}
	defer tx.Rollback(ctx)
	var owner, current string
	err = tx.QueryRow(ctx, `SELECT driver_id::text,status FROM rides WHERE id=$1 FOR UPDATE`, id).Scan(&owner, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Ride{}, apperrors.ErrNotFound
	}
	if err != nil {
		return domain.Ride{}, err
	}
	if owner != driverID {
		return domain.Ride{}, apperrors.ErrForbidden
	}
	if current != "published" && current != "in_progress" {
		return domain.Ride{}, apperrors.ErrRideNotActive
	}
	if _, err = tx.Exec(ctx, `UPDATE rides SET status=$2 WHERE id=$1`, id, status); err != nil {
		return domain.Ride{}, err
	}
	result, err := scanRide(tx.QueryRow(ctx, rideSelect+`WHERE r.id=$1`, id))
	if err != nil {
		return domain.Ride{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Ride{}, err
	}
	return result, nil
}
