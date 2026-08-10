package core

import "context"

// Inventory is the stock service as the order flow sees it.
type Inventory interface {
	Reserve(ctx context.Context, reservation StockReservation) error
	Release(ctx context.Context, release StockRelease) error
}

// StockReservation asks inventory to hold the items of an order.
type StockReservation struct {
	OrderID string
	Items   []Item
}

// StockRelease hands the items of an order back to inventory. Inventory keys
// reservations by order id, so sending the same release twice is harmless.
type StockRelease struct {
	OrderID string
	Reason  string
}

// NopInventory is what the service runs with when no inventory endpoint is
// configured, which is the usual setup for local work.
type NopInventory struct{}

func (NopInventory) Reserve(context.Context, StockReservation) error { return nil }

func (NopInventory) Release(context.Context, StockRelease) error { return nil }
