package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"roadmates/backend/internal/domain"
)

type Districts struct{ pool *pgxpool.Pool }

func NewDistricts(pool *pgxpool.Pool) *Districts { return &Districts{pool} }
func (r *Districts) List(ctx context.Context, query string) ([]domain.District, error) {
	// strpos treats user '%' and '_' literally rather than as SQL wildcards.
	rows, err := r.pool.Query(ctx, `SELECT id,name_az,slug,requires_permit FROM districts WHERE is_active AND ($1='' OR strpos(search_key,$1)>0) ORDER BY sort_order,name_az`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.District, 0)
	for rows.Next() {
		var d domain.District
		if err := rows.Scan(&d.ID, &d.NameAZ, &d.Slug, &d.RequiresPermit); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}
