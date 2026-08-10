package core

import "context"

// Store holds stock levels and the reservations made against them.
type Store interface {
	Level(ctx context.Context, sku string) (Level, error)
	Levels(ctx context.Context) ([]Level, error)
	SetLevel(ctx context.Context, level Level) error
	Reservation(ctx context.Context, orderID string) ([]Line, bool, error)
	SaveReservation(ctx context.Context, orderID string, lines []Line) error
	DeleteReservation(ctx context.Context, orderID string) error
}
