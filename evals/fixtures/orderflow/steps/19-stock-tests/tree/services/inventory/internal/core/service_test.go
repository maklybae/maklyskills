package core_test

import (
	"context"
	"errors"
	"testing"

	"orderflow/services/inventory/internal/core"
	"orderflow/services/inventory/internal/store"
)

func TestReserve(t *testing.T) {
	tests := []struct {
		name    string
		request core.ReserveRequest
		wantErr error
	}{
		{
			name: "one line",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "desk-lamp", Quantity: 2}},
			},
		},
		{
			name: "two lines",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "desk-lamp", Quantity: 2}, {SKU: "monitor-arm", Quantity: 1}},
			},
		},
		{
			name:    "an order id is required",
			request: core.ReserveRequest{Lines: []core.Line{{SKU: "desk-lamp", Quantity: 1}}},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "at least one line",
			request: core.ReserveRequest{OrderID: "ord_1"},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name: "quantity must be positive",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "desk-lamp", Quantity: 0}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name: "a sku appears once",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "desk-lamp", Quantity: 1}, {SKU: "desk-lamp", Quantity: 1}},
			},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name: "unknown sku",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "hammock", Quantity: 1}},
			},
			wantErr: core.ErrUnknownSKU,
		},
		{
			name: "more than the shelf holds",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "monitor-arm", Quantity: 9}},
			},
			wantErr: core.ErrOutOfStock,
		},
		{
			name: "one impossible line refuses the whole reservation",
			request: core.ReserveRequest{
				OrderID: "ord_1",
				Lines:   []core.Line{{SKU: "desk-lamp", Quantity: 1}, {SKU: "monitor-arm", Quantity: 9}},
			},
			wantErr: core.ErrOutOfStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := newService(t)

			err := stock.Reserve(context.Background(), tt.request)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Reserve() error = %v, want %v", err, tt.wantErr)
				}
				assertUntouched(t, stock)
				return
			}
			if err != nil {
				t.Fatalf("Reserve() error = %v", err)
			}

			for _, line := range tt.request.Lines {
				level, err := stock.Level(context.Background(), line.SKU)
				if err != nil {
					t.Fatalf("Level(%s) error = %v", line.SKU, err)
				}
				if level.Reserved != line.Quantity {
					t.Errorf("%s reserved = %d, want %d", line.SKU, level.Reserved, line.Quantity)
				}
			}
		})
	}
}

func TestReserveTwiceForTheSameOrder(t *testing.T) {
	stock := newService(t)
	request := core.ReserveRequest{OrderID: "ord_1", Lines: []core.Line{{SKU: "desk-lamp", Quantity: 2}}}

	if err := stock.Reserve(context.Background(), request); err != nil {
		t.Fatalf("first Reserve() error = %v", err)
	}
	if err := stock.Reserve(context.Background(), request); err != nil {
		t.Fatalf("second Reserve() error = %v", err)
	}

	level, err := stock.Level(context.Background(), "desk-lamp")
	if err != nil {
		t.Fatalf("Level() error = %v", err)
	}
	if level.Reserved != 2 {
		t.Fatalf("reserved = %d, want 2: a repeated reservation must not hold stock twice", level.Reserved)
	}
}

func TestRelease(t *testing.T) {
	stock := newService(t)
	request := core.ReserveRequest{OrderID: "ord_1", Lines: []core.Line{{SKU: "desk-lamp", Quantity: 3}}}
	if err := stock.Reserve(context.Background(), request); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	released, err := stock.Release(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if !released {
		t.Error("Release() = false, want true")
	}

	level, err := stock.Level(context.Background(), "desk-lamp")
	if err != nil {
		t.Fatalf("Level() error = %v", err)
	}
	if level.Reserved != 0 {
		t.Errorf("reserved = %d, want 0", level.Reserved)
	}

	again, err := stock.Release(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("second Release() error = %v", err)
	}
	if again {
		t.Error("second Release() = true, want false")
	}
}

func TestRestock(t *testing.T) {
	tests := []struct {
		name     string
		sku      string
		quantity int
		want     int
		wantErr  error
	}{
		{name: "known sku", sku: "monitor-arm", quantity: 5, want: 13},
		{name: "new sku", sku: "hammock", quantity: 4, want: 4},
		{name: "sku is required", sku: " ", quantity: 4, wantErr: core.ErrInvalidRequest},
		{name: "quantity must be positive", sku: "monitor-arm", quantity: 0, wantErr: core.ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := newService(t)

			level, err := stock.Restock(context.Background(), tt.sku, tt.quantity)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Restock() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Restock() error = %v", err)
			}
			if level.OnHand != tt.want {
				t.Fatalf("on hand = %d, want %d", level.OnHand, tt.want)
			}
		})
	}
}

func TestLevelOfUnknownSKU(t *testing.T) {
	stock := newService(t)

	_, err := stock.Level(context.Background(), "hammock")
	if !errors.Is(err, core.ErrUnknownSKU) {
		t.Fatalf("Level() error = %v, want %v", err, core.ErrUnknownSKU)
	}
}

func TestNewServiceNeedsAStore(t *testing.T) {
	if _, err := core.NewService(nil); err == nil {
		t.Fatal("NewService(nil) = nil, want an error")
	}
}

func newService(t *testing.T) *core.Service {
	t.Helper()

	stock, err := core.NewService(store.NewMemory(
		core.Level{SKU: "desk-lamp", OnHand: 42},
		core.Level{SKU: "monitor-arm", OnHand: 8},
	))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return stock
}

func assertUntouched(t *testing.T, stock *core.Service) {
	t.Helper()

	levels, err := stock.Levels(context.Background())
	if err != nil {
		t.Fatalf("Levels() error = %v", err)
	}
	for _, level := range levels {
		if level.Reserved != 0 {
			t.Fatalf("%s holds %d reserved after a refused reservation", level.SKU, level.Reserved)
		}
	}
}
