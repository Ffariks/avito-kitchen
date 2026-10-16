package domain

// IncomingOrder — заказ, полученный от платформы Авито.Кухня вебхуком
// POST /orders. Это тот же order_id, что и в основном сервисе — заведение
// не хранит собственную копию бизнес-данных заказа, только то, что нужно
// для локальной "готовки".
type IncomingOrder struct {
	OrderID      int64          `json:"order_id"`
	RestaurantID int64          `json:"restaurant_id"`
	Items        []IncomingItem `json:"items"`
}

type IncomingItem struct {
	MenuItemID int64  `json:"menu_item_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
}
