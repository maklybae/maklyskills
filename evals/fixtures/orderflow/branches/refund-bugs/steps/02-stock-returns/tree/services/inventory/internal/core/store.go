package core

import "context"

// Store holds stock levels and the reservations made against them.
//
// Reserve is one method rather than a read followed by a write because the
// check and the update have to happen under the same lock.
type Store interface {
	Level(ctx context.Context, sku string) (Level, error)
	Levels(ctx context.Context) ([]Level, error)
	Reserve(ctx context.Context, orderID string, lines []Line) error
	Release(ctx context.Context, orderID string) (bool, error)
	Restock(ctx context.Context, sku string, quantity int) (Level, error)
	ReturnLines(ctx context.Context, lines []Line) ([]Level, error)
}
