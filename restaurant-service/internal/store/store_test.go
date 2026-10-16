package store

import (
	"testing"

	"github.com/farik/avito-kitchen/restaurant-service/internal/domain"
)

func TestSaveIsIdempotent(t *testing.T) {
	s := New()
	o := domain.IncomingOrder{OrderID: 1, RestaurantID: 1}
	if !s.Save(o) {
		t.Fatal("first delivery should be saved")
	}
	if s.Save(o) {
		t.Fatal("repeated delivery of the same order should be ignored")
	}
	if got := len(s.List()); got != 1 {
		t.Fatalf("stored %d orders, want 1", got)
	}
}
