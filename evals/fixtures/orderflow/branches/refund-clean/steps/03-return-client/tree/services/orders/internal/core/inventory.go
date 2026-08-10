package core

import "context"

// Inventory is the stock service as the order flow sees it.
type Inventory interface {
	Reserve(ctx context.Context, reservation StockReservation) error
	Release(ctx context.Context, release StockRelease) error
	Restock(ctx context.Context, ret StockReturn) error
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

// StockReturn puts the lines of a refund back on the shelf. Inventory counts
// returns instead of keying them, so it must be sent exactly once.
type StockReturn struct {
	OrderID string
	Lines   []Item
}

// NopInventory is what the service runs with when no inventory endpoint is
// configured, which is the usual setup for local work.
type NopInventory struct{}

func (NopInventory) Reserve(context.Context, StockReservation) error { return nil }

func (NopInventory) Release(context.Context, StockRelease) error { return nil }

func (NopInventory) Restock(context.Context, StockReturn) error { return nil }
