package domain

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to OrderStatus
		want     bool
	}{
		{OrderStatusCreated, OrderStatusAccepted, true},
		{OrderStatusCreated, OrderStatusRejected, true},
		{OrderStatusCreated, OrderStatusCancelled, true},
		{OrderStatusAccepted, OrderStatusCooking, true},
		{OrderStatusCooking, OrderStatusReady, true},
		{OrderStatusReady, OrderStatusDelivered, true},

		{OrderStatusCreated, OrderStatusDelivered, false},
		{OrderStatusCreated, OrderStatusCooking, false},
		{OrderStatusAccepted, OrderStatusRejected, false},
		{OrderStatusReady, OrderStatusCancelled, false},
		{OrderStatusDelivered, OrderStatusCreated, false},
		{OrderStatusCancelled, OrderStatusAccepted, false},
		{OrderStatusRejected, OrderStatusAccepted, false},
		{OrderStatusCreated, OrderStatus("unknown"), false},
	}
	for _, tt := range tests {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestUserCancellable(t *testing.T) {
	for _, s := range []OrderStatus{OrderStatusCreated, OrderStatusAccepted} {
		if !UserCancellable(s) {
			t.Errorf("user should be able to cancel order in status %s", s)
		}
	}
	for _, s := range []OrderStatus{OrderStatusCooking, OrderStatusReady, OrderStatusDelivered, OrderStatusRejected, OrderStatusCancelled} {
		if UserCancellable(s) {
			t.Errorf("user should not be able to cancel order in status %s", s)
		}
	}
}
