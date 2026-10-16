package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/farik/avito-kitchen/internal/domain"
	"github.com/farik/avito-kitchen/internal/httpclient"
)

// --- in-memory фейки репозиториев ---

type fakeRestaurants map[int64]domain.Restaurant

func (f fakeRestaurants) GetByID(_ context.Context, id int64) (domain.Restaurant, error) {
	r, ok := f[id]
	if !ok {
		return domain.Restaurant{}, domain.ErrNotFound
	}
	return r, nil
}

type fakeMenu map[int64]domain.MenuItem

func (f fakeMenu) GetByID(_ context.Context, id int64) (domain.MenuItem, error) {
	mi, ok := f[id]
	if !ok {
		return domain.MenuItem{}, domain.ErrNotFound
	}
	return mi, nil
}

type fakeOrders struct {
	mu     sync.Mutex
	nextID int64
	orders map[int64]domain.Order
}

func newFakeOrders() *fakeOrders { return &fakeOrders{orders: map[int64]domain.Order{}} }

func (f *fakeOrders) CreateWithItems(_ context.Context, o domain.Order) (domain.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	o.ID = f.nextID
	f.orders[o.ID] = o
	return o, nil
}

func (f *fakeOrders) GetByID(_ context.Context, id int64) (domain.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

func (f *fakeOrders) ListByUser(context.Context, int64) ([]domain.Order, error) { return nil, nil }
func (f *fakeOrders) ListByRestaurant(context.Context, int64) ([]domain.Order, error) {
	return nil, nil
}

func (f *fakeOrders) UpdateStatus(_ context.Context, id int64, from, to domain.OrderStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[id]
	if !ok || o.Status != from {
		return domain.ErrInvalidTransition
	}
	o.Status = to
	f.orders[id] = o
	return nil
}

type fakeNotifier struct {
	done chan httpclient.NewOrderPayload
}

func (f *fakeNotifier) NotifyNewOrder(_ context.Context, _ string, p httpclient.NewOrderPayload) error {
	f.done <- p
	return nil
}

// --- тесты ---

func newTestService(t *testing.T) (*OrderService, *fakeOrders, *fakeNotifier) {
	t.Helper()
	restaurants := fakeRestaurants{
		1: {ID: 1, Name: "Кухня", CallbackURL: "http://r1", IsActive: true},
		2: {ID: 2, Name: "Закрыто", CallbackURL: "http://r2", IsActive: false},
	}
	menu := fakeMenu{
		10: {ID: 10, RestaurantID: 1, Name: "Борщ", PriceCents: 35000, IsAvailable: true},
		11: {ID: 11, RestaurantID: 1, Name: "Морс", PriceCents: 12000, IsAvailable: false},
		20: {ID: 20, RestaurantID: 2, Name: "Чужое", PriceCents: 100, IsAvailable: true},
	}
	orders := newFakeOrders()
	notifier := &fakeNotifier{done: make(chan httpclient.NewOrderPayload, 1)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewOrderService(orders, menu, restaurants, notifier, log), orders, notifier
}

func TestCreateOrder_Success(t *testing.T) {
	svc, _, notifier := newTestService(t)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID: 7, RestaurantID: 1,
		Items: []CreateOrderItemInput{{MenuItemID: 10, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != domain.OrderStatusCreated {
		t.Errorf("status = %s, want created", order.Status)
	}
	if order.TotalCents != 70000 {
		t.Errorf("total = %d, want 70000", order.TotalCents)
	}
	if got := order.Items[0].PriceCentsAtOrder; got != 35000 {
		t.Errorf("price at order = %d, want 35000", got)
	}

	payload := <-notifier.done
	if payload.OrderID != order.ID || payload.RestaurantID != 1 {
		t.Errorf("webhook payload = %+v, want order %d of restaurant 1", payload, order.ID)
	}
}

func TestCreateOrder_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		in   CreateOrderInput
		want error
	}{
		{"empty order", CreateOrderInput{UserID: 1, RestaurantID: 1}, domain.ErrEmptyOrder},
		{"unknown restaurant", CreateOrderInput{UserID: 1, RestaurantID: 99, Items: []CreateOrderItemInput{{10, 1}}}, domain.ErrNotFound},
		{"inactive restaurant", CreateOrderInput{UserID: 1, RestaurantID: 2, Items: []CreateOrderItemInput{{20, 1}}}, domain.ErrRestaurantInactive},
		{"unknown menu item", CreateOrderInput{UserID: 1, RestaurantID: 1, Items: []CreateOrderItemInput{{999, 1}}}, domain.ErrMenuItemNotFound},
		{"foreign menu item", CreateOrderInput{UserID: 1, RestaurantID: 1, Items: []CreateOrderItemInput{{20, 1}}}, domain.ErrMenuItemForeign},
		{"unavailable item", CreateOrderInput{UserID: 1, RestaurantID: 1, Items: []CreateOrderItemInput{{11, 1}}}, domain.ErrMenuItemUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := newTestService(t)
			_, err := svc.CreateOrder(context.Background(), tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func createdOrder(t *testing.T, svc *OrderService, notifier *fakeNotifier) domain.Order {
	t.Helper()
	o, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID: 7, RestaurantID: 1,
		Items: []CreateOrderItemInput{{MenuItemID: 10, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	<-notifier.done
	return o
}

func TestUpdateStatus(t *testing.T) {
	svc, _, notifier := newTestService(t)
	o := createdOrder(t, svc, notifier)
	ctx := context.Background()

	if _, err := svc.UpdateStatus(ctx, 2, o.ID, domain.OrderStatusAccepted); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign restaurant: err = %v, want ErrNotFound", err)
	}
	if _, err := svc.UpdateStatus(ctx, 1, o.ID, domain.OrderStatusDelivered); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("created -> delivered: err = %v, want ErrInvalidTransition", err)
	}
	got, err := svc.UpdateStatus(ctx, 1, o.ID, domain.OrderStatusAccepted)
	if err != nil {
		t.Fatalf("created -> accepted: %v", err)
	}
	if got.Status != domain.OrderStatusAccepted {
		t.Errorf("status = %s, want accepted", got.Status)
	}
}

func TestCancelByUser(t *testing.T) {
	ctx := context.Background()

	t.Run("own order before cooking", func(t *testing.T) {
		svc, _, notifier := newTestService(t)
		o := createdOrder(t, svc, notifier)
		got, err := svc.CancelByUser(ctx, 7, o.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != domain.OrderStatusCancelled {
			t.Errorf("status = %s, want cancelled", got.Status)
		}
	})

	t.Run("someone else's order", func(t *testing.T) {
		svc, _, notifier := newTestService(t)
		o := createdOrder(t, svc, notifier)
		if _, err := svc.CancelByUser(ctx, 8, o.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("already cooking", func(t *testing.T) {
		svc, _, notifier := newTestService(t)
		o := createdOrder(t, svc, notifier)
		for _, s := range []domain.OrderStatus{domain.OrderStatusAccepted, domain.OrderStatusCooking} {
			if _, err := svc.UpdateStatus(ctx, 1, o.ID, s); err != nil {
				t.Fatalf("advance to %s: %v", s, err)
			}
		}
		if _, err := svc.CancelByUser(ctx, 7, o.ID); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("err = %v, want ErrInvalidTransition", err)
		}
	})
}
