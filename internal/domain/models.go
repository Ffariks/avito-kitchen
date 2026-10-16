package domain

import "time"

// OrderStatus описывает жизненный цикл заказа.
//
// created   -> заказ создан пользователем, ждёт реакции заведения
// accepted  -> заведение приняло заказ в работу
// cooking   -> заказ готовится
// ready     -> заказ готов (к выдаче/доставке)
// delivered -> заказ выполнен (терминальный статус)
// rejected  -> заведение отклонило заказ (например, товара нет в наличии)
// cancelled -> заказ отменён (терминальный статус)
type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusAccepted  OrderStatus = "accepted"
	OrderStatusCooking   OrderStatus = "cooking"
	OrderStatusReady     OrderStatus = "ready"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// allowedTransitions задаёт разрешённые переходы между статусами заказа.
// Это ядро бизнес-правила "заказ нельзя провести мимо жизненного цикла".
var allowedTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusCreated:  {OrderStatusAccepted, OrderStatusRejected, OrderStatusCancelled},
	OrderStatusAccepted: {OrderStatusCooking, OrderStatusCancelled},
	OrderStatusCooking:  {OrderStatusReady, OrderStatusCancelled},
	OrderStatusReady:    {OrderStatusDelivered},
}

// UserCancellable — статусы, из которых пользователь может сам отменить
// заказ: пока заведение не начало готовить. Дальше отменить может только
// заведение (переходы accepted/cooking -> cancelled в allowedTransitions).
func UserCancellable(s OrderStatus) bool {
	return s == OrderStatusCreated || s == OrderStatusAccepted
}

// CanTransition проверяет, допустим ли переход статуса заказа из from в to.
func CanTransition(from, to OrderStatus) bool {
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

type Restaurant struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// CallbackURL — адрес сервиса заведения, на который платформа
	// отправляет вебхуки о новых заказах (POST {CallbackURL}/orders).
	CallbackURL string    `json:"-"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type MenuItem struct {
	ID           int64     `json:"id"`
	RestaurantID int64     `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	PriceCents   int64     `json:"price_cents"`
	IsAvailable  bool      `json:"is_available"`
	CreatedAt    time.Time `json:"created_at"`
}

type Order struct {
	ID           int64       `json:"id"`
	UserID       int64       `json:"user_id"`
	RestaurantID int64       `json:"restaurant_id"`
	Status       OrderStatus `json:"status"`
	TotalCents   int64       `json:"total_cents"`
	Comment      string      `json:"comment,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Items        []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID                int64  `json:"id"`
	OrderID           int64  `json:"order_id"`
	MenuItemID        int64  `json:"menu_item_id"`
	Name              string `json:"name"`
	Quantity          int32  `json:"quantity"`
	PriceCentsAtOrder int64  `json:"price_cents_at_order"`
}
