package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/farik/avito-kitchen/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

type errorResponse struct {
	Error string `json:"error"`
}

// writeError маппит доменные ошибки на HTTP-статусы. Все прочие ошибки
// считаются внутренними и логируются с деталями, но наружу отдаётся
// только общий текст, чтобы не раскрывать внутренности системы.
func writeError(w http.ResponseWriter, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	case errors.Is(err, domain.ErrRestaurantInactive),
		errors.Is(err, domain.ErrMenuItemUnavailable),
		errors.Is(err, domain.ErrMenuItemForeign),
		errors.Is(err, domain.ErrMenuItemNotFound),
		errors.Is(err, domain.ErrEmptyOrder),
		errors.Is(err, domain.ErrInvalidTransition):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: err.Error()})
	default:
		log.Error("internal error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
	}
}

// pathID достаёт положительный int64 из параметра пути. При ошибке сам
// пишет 400 и возвращает ok=false.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid " + name})
		return 0, false
	}
	return id, true
}
