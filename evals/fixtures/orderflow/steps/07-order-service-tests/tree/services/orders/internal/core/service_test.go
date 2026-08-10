package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

var now = time.Date(2024, time.May, 14, 9, 30, 0, 0, time.UTC)

func TestPlaceOrder(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}

	tests := []struct {
		name    string
		request core.PlaceRequest
		wantErr error
	}{
		{
			name:    "single item",
			request: core.PlaceRequest{CustomerID: "cust-17", Items: []core.Item{lamp}},
		},
		{
			name:    "customer is required",
			request: core.PlaceRequest{Items: []core.Item{lamp}},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "at least one item",
			request: core.PlaceRequest{CustomerID: "cust-17"},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name:    "quantity must be positive",
			request: core.PlaceRequest{CustomerID: "cust-17", Items: []core.Item{{SKU: "desk-lamp"}}},
			wantErr: core.ErrInvalidRequest,
		},
		{
			name: "a sku appears once",
			request: core.PlaceRequest{
				CustomerID: "cust-17",
				Items:      []core.Item{lamp, {SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}},
			},
			wantErr: core.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newService(t, testsupport.NewStore())

			order, err := service.PlaceOrder(context.Background(), tt.request)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("PlaceOrder() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlaceOrder() error = %v", err)
			}

			if order.Status != core.StatusPending {
				t.Errorf("status = %s, want %s", order.Status, core.StatusPending)
			}
			if !order.PlacedAt.Equal(now) {
				t.Errorf("placed at %s, want %s", order.PlacedAt, now)
			}
			if order.Total() != 9800 {
				t.Errorf("total = %d, want 9800", order.Total())
			}

			stored, err := service.Get(context.Background(), order.ID)
			if err != nil {
				t.Fatalf("Get(%s) error = %v", order.ID, err)
			}
			if stored.ID != order.ID {
				t.Errorf("stored order = %s, want %s", stored.ID, order.ID)
			}
		})
	}
}

func TestGetUnknownOrder(t *testing.T) {
	service := newService(t, testsupport.NewStore())

	_, err := service.Get(context.Background(), "ord_missing")
	if !errors.Is(err, core.ErrOrderNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, core.ErrOrderNotFound)
	}
}

func TestListRejectsNegativeLimit(t *testing.T) {
	service := newService(t, testsupport.NewStore())

	_, err := service.List(context.Background(), core.Filter{Limit: -1})
	if !errors.Is(err, core.ErrInvalidRequest) {
		t.Fatalf("List() error = %v, want %v", err, core.ErrInvalidRequest)
	}
}

func TestNewServiceNeedsAStore(t *testing.T) {
	if _, err := core.NewService(nil); err == nil {
		t.Fatal("NewService(nil) = nil, want an error")
	}
}

func newService(t *testing.T, orders *testsupport.Store) *core.Service {
	t.Helper()

	service, err := core.NewService(orders,
		core.WithClock(testsupport.Clock(now)),
		core.WithIDSource(&testsupport.IDs{}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func orderAt(status core.Status) core.Order {
	return core.Order{
		ID:         "ord_1",
		CustomerID: "cust-17",
		Items:      []core.Item{{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}},
		Status:     status,
		PlacedAt:   now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-2 * time.Hour),
	}
}
