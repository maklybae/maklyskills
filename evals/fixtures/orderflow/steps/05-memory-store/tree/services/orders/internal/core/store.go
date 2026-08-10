package core

import (
	"context"
	"slices"
)

// Store persists orders. The implementations live in internal/store and are the
// only place that knows how an order is written down.
type Store interface {
	Create(ctx context.Context, order Order) error
	Get(ctx context.Context, id string) (Order, error)
	Update(ctx context.Context, order Order) error
	List(ctx context.Context, filter Filter) ([]Order, error)
}

// Filter narrows a listing. The zero value matches every order.
type Filter struct {
	CustomerID string
	Statuses   []Status
	Limit      int
}

// Matches is used by every store so the filter means the same thing everywhere.
func (f Filter) Matches(order Order) bool {
	if f.CustomerID != "" && f.CustomerID != order.CustomerID {
		return false
	}
	if len(f.Statuses) > 0 && !slices.Contains(f.Statuses, order.Status) {
		return false
	}
	return true
}
