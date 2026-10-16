package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/farik/avito-kitchen/internal/domain"
)

type MenuRepo struct {
	pool *pgxpool.Pool
}

func NewMenuRepo(pool *pgxpool.Pool) *MenuRepo {
	return &MenuRepo{pool: pool}
}

func (r *MenuRepo) ListByRestaurant(ctx context.Context, restaurantID int64) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, restaurant_id, name, description, price_cents, is_available, created_at
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY id`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MenuItem
	for rows.Next() {
		var mi domain.MenuItem
		if err := rows.Scan(&mi.ID, &mi.RestaurantID, &mi.Name, &mi.Description, &mi.PriceCents, &mi.IsAvailable, &mi.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, mi)
	}
	return items, rows.Err()
}

func (r *MenuRepo) GetByID(ctx context.Context, id int64) (domain.MenuItem, error) {
	var mi domain.MenuItem
	err := r.pool.QueryRow(ctx, `
		SELECT id, restaurant_id, name, description, price_cents, is_available, created_at
		FROM menu_items
		WHERE id = $1`, id).Scan(&mi.ID, &mi.RestaurantID, &mi.Name, &mi.Description, &mi.PriceCents, &mi.IsAvailable, &mi.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MenuItem{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MenuItem{}, err
	}
	return mi, nil
}

// Create и SetAvailability нужны заведению, чтобы управлять своим меню через API.
func (r *MenuRepo) Create(ctx context.Context, mi domain.MenuItem) (domain.MenuItem, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO menu_items (restaurant_id, name, description, price_cents, is_available)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`, mi.RestaurantID, mi.Name, mi.Description, mi.PriceCents, mi.IsAvailable,
	).Scan(&mi.ID, &mi.CreatedAt)
	if err != nil {
		return domain.MenuItem{}, err
	}
	return mi, nil
}

// SetAvailability меняет наличие позиции только в рамках указанного заведения:
// чужую позицию заведение изменить не может (вернётся ErrNotFound).
func (r *MenuRepo) SetAvailability(ctx context.Context, restaurantID, id int64, isAvailable bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE menu_items SET is_available = $3
		WHERE id = $1 AND restaurant_id = $2`, id, restaurantID, isAvailable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
