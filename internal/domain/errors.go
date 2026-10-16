package domain

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrRestaurantInactive  = errors.New("restaurant is not active")
	ErrMenuItemUnavailable = errors.New("menu item is not available")
	ErrMenuItemForeign     = errors.New("menu item belongs to another restaurant")
	ErrMenuItemNotFound    = errors.New("menu item not found")
	ErrEmptyOrder          = errors.New("order must contain at least one item")
	ErrInvalidTransition   = errors.New("invalid order status transition")
)
