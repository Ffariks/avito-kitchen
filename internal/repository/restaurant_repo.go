package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/farik/avito-kitchen/internal/domain"
)

type RestaurantRepo struct {
	pool *pgxpool.Pool
}

func NewRestaurantRepo(pool *pgxpool.Pool) *RestaurantRepo {
	return &RestaurantRepo{pool: pool}
}

func (r *RestaurantRepo) List(ctx context.Context) ([]domain.Restaurant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, callback_url, is_active, created_at
		FROM restaurants
		WHERE is_active = true
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Restaurant
	for rows.Next() {
		var rest domain.Restaurant
		if err := rows.Scan(&rest.ID, &rest.Name, &rest.Description, &rest.CallbackURL, &rest.IsActive, &rest.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, rest)
	}
	return result, rows.Err()
}

func (r *RestaurantRepo) GetByID(ctx context.Context, id int64) (domain.Restaurant, error) {
	var rest domain.Restaurant
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, callback_url, is_active, created_at
		FROM restaurants
		WHERE id = $1`, id).Scan(&rest.ID, &rest.Name, &rest.Description, &rest.CallbackURL, &rest.IsActive, &rest.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Restaurant{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Restaurant{}, err
	}
	return rest, nil
}
