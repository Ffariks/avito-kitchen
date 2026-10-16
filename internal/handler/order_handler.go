package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/farik/avito-kitchen/internal/domain"
	"github.com/farik/avito-kitchen/internal/service"
)

type OrderHandler struct {
	orders *service.OrderService
	log    *slog.Logger
}

func NewOrderHandler(orders *service.OrderService, log *slog.Logger) *OrderHandler {
	return &OrderHandler{orders: orders, log: log}
}

// Register регистрирует маршруты, связанные с заказами.
//
// Клиентское API:
//
//	POST /api/v1/orders                 — оформить заказ
//	GET  /api/v1/orders/{id}            — статус и состав заказа
//	POST /api/v1/orders/{id}/cancel     — отменить свой заказ
//	GET  /api/v1/users/{id}/orders      — история заказов пользователя
//
// API для заведений:
//
//	GET   /api/v1/restaurant/{restaurant_id}/orders             — список заказов заведения
//	PATCH /api/v1/restaurant/{restaurant_id}/orders/{id}/status — сменить статус своего заказа
func (h *OrderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orders", h.createOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.getOrder)
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", h.cancelOrder)
	mux.HandleFunc("GET /api/v1/users/{id}/orders", h.listUserOrders)

	mux.HandleFunc("GET /api/v1/restaurant/{restaurant_id}/orders", h.listRestaurantOrders)
	mux.HandleFunc("PATCH /api/v1/restaurant/{restaurant_id}/orders/{id}/status", h.updateOrderStatus)
}

type createOrderItemRequest struct {
	MenuItemID int64 `json:"menu_item_id"`
	Quantity   int32 `json:"quantity"`
}

type createOrderRequest struct {
	UserID       int64                    `json:"user_id"`
	RestaurantID int64                    `json:"restaurant_id"`
	Comment      string                   `json:"comment"`
	Items        []createOrderItemRequest `json:"items"`
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid body"})
		return
	}
	if req.UserID <= 0 || req.RestaurantID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_id and restaurant_id are required"})
		return
	}

	in := service.CreateOrderInput{
		UserID:       req.UserID,
		RestaurantID: req.RestaurantID,
		Comment:      req.Comment,
	}
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "item quantity must be positive"})
			return
		}
		in.Items = append(in.Items, service.CreateOrderItemInput{MenuItemID: it.MenuItemID, Quantity: it.Quantity})
	}

	order, err := h.orders.CreateOrder(r.Context(), in)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	order, err := h.orders.GetOrder(r.Context(), id)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) listUserOrders(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	orders, err := h.orders.ListUserOrders(r.Context(), id)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	if orders == nil {
		orders = []domain.Order{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) listRestaurantOrders(w http.ResponseWriter, r *http.Request) {
	restaurantID, ok := pathID(w, r, "restaurant_id")
	if !ok {
		return
	}
	orders, err := h.orders.ListRestaurantOrders(r.Context(), restaurantID)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	if orders == nil {
		orders = []domain.Order{}
	}
	writeJSON(w, http.StatusOK, orders)
}

type updateStatusRequest struct {
	Status domain.OrderStatus `json:"status"`
}

// updateOrderStatus вызывается сервисом заведения (restaurant-service), чтобы
// продвинуть заказ по жизненному циклу — это "API для интеграции" в
// обратную сторону, которое площадка предоставляет заведениям.
func (h *OrderHandler) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	restaurantID, ok := pathID(w, r, "restaurant_id")
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid body"})
		return
	}
	order, err := h.orders.UpdateStatus(r.Context(), restaurantID, id, req.Status)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

type cancelOrderRequest struct {
	// UserID передаётся явно, пока нет аутентификации (см. README):
	// отменить можно только свой заказ.
	UserID int64 `json:"user_id"`
}

func (h *OrderHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req cancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_id is required"})
		return
	}
	order, err := h.orders.CancelByUser(r.Context(), req.UserID, id)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}
