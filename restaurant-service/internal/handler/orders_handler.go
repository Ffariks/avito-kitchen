package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/farik/avito-kitchen/restaurant-service/internal/client"
	"github.com/farik/avito-kitchen/restaurant-service/internal/domain"
	"github.com/farik/avito-kitchen/restaurant-service/internal/store"
)

type OrdersHandler struct {
	store    *store.Store
	platform *client.PlatformClient
	log      *slog.Logger
	// cookingDelay — насколько "долго" заведение готовит заказ на каждом
	// шаге симуляции. Вынесено в поле, чтобы тесты могли его сократить.
	cookingDelay time.Duration
}

func NewOrdersHandler(s *store.Store, platform *client.PlatformClient, log *slog.Logger, cookingDelay time.Duration) *OrdersHandler {
	return &OrdersHandler{store: s, platform: platform, log: log, cookingDelay: cookingDelay}
}

func (h *OrdersHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.receiveOrder)
	mux.HandleFunc("GET /orders", h.listOrders)
}

// receiveOrder — это тот самый "свой API" заведения из ТЗ: платформа
// вызывает его вебхуком при создании заказа. Заведение сохраняет заказ
// и запускает симуляцию цикла приготовления, продвигая статус через
// обратные вызовы в платформу: accepted -> cooking -> ready.
func (h *OrdersHandler) receiveOrder(w http.ResponseWriter, r *http.Request) {
	var order domain.IncomingOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil || order.OrderID <= 0 || order.RestaurantID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !h.store.Save(order) {
		h.log.Info("duplicate order webhook ignored", "order_id", order.OrderID)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	h.log.Info("received new order", "order_id", order.OrderID, "items", len(order.Items))

	go h.simulateCooking(context.Background(), order.RestaurantID, order.OrderID)

	w.WriteHeader(http.StatusAccepted)
}

func (h *OrdersHandler) simulateCooking(ctx context.Context, restaurantID, orderID int64) {
	steps := []string{"accepted", "cooking", "ready"}
	for _, status := range steps {
		time.Sleep(h.cookingDelay)
		if err := h.platform.UpdateOrderStatus(ctx, restaurantID, orderID, status); err != nil {
			h.log.Error("failed to update order status on platform", "order_id", orderID, "status", status, "error", err)
			return
		}
		h.log.Info("order status advanced", "order_id", orderID, "status", status)
	}
}

func (h *OrdersHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.store.List())
}
