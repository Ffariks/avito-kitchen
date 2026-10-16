package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/farik/avito-kitchen/internal/domain"
)

type OrderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{pool: pool}
}

// CreateWithItems атомарно создаёт заказ и его позиции одной транзакцией,
// чтобы никогда не получить заказ без items или items без заказа.
func (r *OrderRepo) CreateWithItems(ctx context.Context, order domain.Order) (domain.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // ошибка отката не важна при успешном commit

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, restaurant_id, status, total_cents, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		order.UserID, order.RestaurantID, order.Status, order.TotalCents, order.Comment,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return domain.Order{}, err
	}

	for i, item := range order.Items {
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, name, quantity, price_cents_at_order)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			order.ID, item.MenuItemID, item.Name, item.Quantity, item.PriceCentsAtOrder,
		).Scan(&order.Items[i].ID)
		if err != nil {
			return domain.Order{}, err
		}
		order.Items[i].OrderID = order.ID
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (r *OrderRepo) GetByID(ctx context.Context, id int64) (domain.Order, error) {
	var o domain.Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, restaurant_id, status, total_cents, comment, created_at, updated_at
		FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.UserID, &o.RestaurantID, &o.Status, &o.TotalCents, &o.Comment, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, menu_item_id, name, quantity, price_cents_at_order
		FROM order_items WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return domain.Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.Name, &it.Quantity, &it.PriceCentsAtOrder); err != nil {
			return domain.Order{}, err
		}
		o.Items = append(o.Items, it)
	}
	return o, rows.Err()
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, restaurant_id, status, total_cents, comment, created_at, updated_at
		FROM orders WHERE user_id = $1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.RestaurantID, &o.Status, &o.TotalCents, &o.Comment, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepo) ListByRestaurant(ctx context.Context, restaurantID int64) ([]domain.Order, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, restaurant_id, status, total_cents, comment, created_at, updated_at
		FROM orders WHERE restaurant_id = $1 ORDER BY id DESC`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.RestaurantID, &o.Status, &o.TotalCents, &o.Comment, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// UpdateStatus меняет статус заказа, только если from совпадает с текущим статусом в БД —
// это защищает от гонки, если два запроса на смену статуса пришли одновременно.
func (r *OrderRepo) UpdateStatus(ctx context.Context, id int64, from, to domain.OrderStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $3, updated_at = now()
		WHERE id = $1 AND status = $2`, id, from, to)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvalidTransition
	}
	return nil
}
