package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/farik/avito-kitchen/internal/domain"
	"github.com/farik/avito-kitchen/internal/service"
)

type CatalogHandler struct {
	catalog *service.CatalogService
	log     *slog.Logger
}

func NewCatalogHandler(catalog *service.CatalogService, log *slog.Logger) *CatalogHandler {
	return &CatalogHandler{catalog: catalog, log: log}
}

// Register регистрирует маршруты каталога.
//
// Клиентское API (веб-версия площадки):
//
//	GET  /api/v1/restaurants            — список активных заведений
//	GET  /api/v1/restaurants/{id}       — карточка заведения
//	GET  /api/v1/restaurants/{id}/menu  — меню заведения
//
// API для заведений (закрытый список интеграторов):
//
//	POST  /api/v1/restaurant/{restaurant_id}/menu-items                   — добавить позицию меню
//	PATCH /api/v1/restaurant/{restaurant_id}/menu-items/{id}/availability — отметить наличие/отсутствие товара
//
// restaurant_id в пути — заглушка вместо идентификатора из токена заведения
// (аутентификация вне рамок MVP): все операции заведения ограничены его
// собственными данными.
func (h *CatalogHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/restaurants", h.listRestaurants)
	mux.HandleFunc("GET /api/v1/restaurants/{id}", h.getRestaurant)
	mux.HandleFunc("GET /api/v1/restaurants/{id}/menu", h.listMenu)

	mux.HandleFunc("POST /api/v1/restaurant/{restaurant_id}/menu-items", h.addMenuItem)
	mux.HandleFunc("PATCH /api/v1/restaurant/{restaurant_id}/menu-items/{id}/availability", h.setAvailability)
}

func (h *CatalogHandler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	restaurants, err := h.catalog.ListRestaurants(r.Context())
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	if restaurants == nil {
		restaurants = []domain.Restaurant{}
	}
	writeJSON(w, http.StatusOK, restaurants)
}

func (h *CatalogHandler) getRestaurant(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	rest, err := h.catalog.GetRestaurant(r.Context(), id)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, rest)
}

func (h *CatalogHandler) listMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	items, err := h.catalog.ListMenu(r.Context(), id)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	if items == nil {
		items = []domain.MenuItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

type addMenuItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
}

func (h *CatalogHandler) addMenuItem(w http.ResponseWriter, r *http.Request) {
	restaurantID, ok := pathID(w, r, "restaurant_id")
	if !ok {
		return
	}
	var req addMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid body"})
		return
	}
	if req.Name == "" || req.PriceCents <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and positive price_cents are required"})
		return
	}
	created, err := h.catalog.AddMenuItem(r.Context(), domain.MenuItem{
		RestaurantID: restaurantID,
		Name:         req.Name,
		Description:  req.Description,
		PriceCents:   req.PriceCents,
		IsAvailable:  true,
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type setAvailabilityRequest struct {
	// Указатель, чтобы отличить отсутствующее поле от явного false:
	// пустое тело {} не должно молча снимать товар с продажи.
	IsAvailable *bool `json:"is_available"`
}

func (h *CatalogHandler) setAvailability(w http.ResponseWriter, r *http.Request) {
	restaurantID, ok := pathID(w, r, "restaurant_id")
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req setAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IsAvailable == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "is_available is required"})
		return
	}
	if err := h.catalog.SetMenuItemAvailability(r.Context(), restaurantID, id, *req.IsAvailable); err != nil {
		writeError(w, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
