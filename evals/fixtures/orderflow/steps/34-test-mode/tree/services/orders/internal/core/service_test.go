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

func TestMain(m *testing.M) {
	testsupport.Main(m)
}

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
			orders, stock := testsupport.NewStore(), testsupport.NewInventory()
			service := newService(t, orders, stock)

			order, err := service.PlaceOrder(context.Background(), tt.request)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("PlaceOrder() error = %v, want %v", err, tt.wantErr)
				}
				if len(stock.Reservations()) != 0 {
					t.Fatal("a rejected order reserved stock")
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

func TestPlaceOrderHoldsStockFirst(t *testing.T) {
	orders, stock := testsupport.NewStore(), testsupport.NewInventory()
	service := newService(t, orders, stock)

	order, err := service.PlaceOrder(context.Background(), core.PlaceRequest{
		CustomerID: "cust-17",
		Items:      []core.Item{{SKU: "monitor-arm", Quantity: 1, UnitPrice: 12900}},
	})
	if err != nil {
		t.Fatalf("PlaceOrder() error = %v", err)
	}

	reservations := stock.Reservations()
	if len(reservations) != 1 {
		t.Fatalf("made %d reservations, want 1", len(reservations))
	}
	if reservations[0].OrderID != order.ID {
		t.Errorf("reserved for %s, want %s", reservations[0].OrderID, order.ID)
	}
}

func TestPlaceOrderWithoutStock(t *testing.T) {
	orders, stock := testsupport.NewStore(), testsupport.NewInventory()
	stock.FailReserve(core.ErrOutOfStock)
	service := newService(t, orders, stock)

	_, err := service.PlaceOrder(context.Background(), core.PlaceRequest{
		CustomerID: "cust-17",
		Items:      []core.Item{{SKU: "footrest", Quantity: 1, UnitPrice: 2500}},
	})
	if !errors.Is(err, core.ErrOutOfStock) {
		t.Fatalf("PlaceOrder() error = %v, want %v", err, core.ErrOutOfStock)
	}

	stored, err := service.List(context.Background(), core.Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored %d orders, want none", len(stored))
	}
}

func TestPlaceOrderReleasesStockWhenStoreFails(t *testing.T) {
	orders, stock := testsupport.NewStore(), testsupport.NewInventory()
	orders.FailCreate(errors.New("disk full"))
	service := newService(t, orders, stock)

	_, err := service.PlaceOrder(context.Background(), core.PlaceRequest{
		CustomerID: "cust-17",
		Items:      []core.Item{{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}},
	})
	if err == nil {
		t.Fatal("PlaceOrder() = nil, want the store error")
	}

	releases := stock.Releases()
	if len(releases) != 1 {
		t.Fatalf("released %d reservations, want 1", len(releases))
	}
}

func TestTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    core.Status
		move    func(*core.Service, string) (core.Order, error)
		want    core.Status
		wantErr error
	}{
		{
			name: "pending order is paid",
			from: core.StatusPending,
			move: func(s *core.Service, id string) (core.Order, error) { return s.MarkPaid(context.Background(), id) },
			want: core.StatusPaid,
		},
		{
			name: "paid order ships",
			from: core.StatusPaid,
			move: func(s *core.Service, id string) (core.Order, error) { return s.MarkShipped(context.Background(), id) },
			want: core.StatusShipped,
		},
		{
			name:    "unpaid order does not ship",
			from:    core.StatusPending,
			move:    func(s *core.Service, id string) (core.Order, error) { return s.MarkShipped(context.Background(), id) },
			wantErr: core.ErrNotShippable,
		},
		{
			name: "shipping twice changes nothing",
			from: core.StatusShipped,
			move: func(s *core.Service, id string) (core.Order, error) { return s.MarkShipped(context.Background(), id) },
			want: core.StatusShipped,
		},
		{
			name: "shipped order is delivered",
			from: core.StatusShipped,
			move: func(s *core.Service, id string) (core.Order, error) { return s.MarkDelivered(context.Background(), id) },
			want: core.StatusDelivered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := orderAt(tt.from)
			service := newService(t, testsupport.NewStore(seed), testsupport.NewInventory())

			order, err := tt.move(service, seed.ID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("transition error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("transition error = %v", err)
			}
			if order.Status != tt.want {
				t.Fatalf("status = %s, want %s", order.Status, tt.want)
			}
		})
	}
}

func TestGetUnknownOrder(t *testing.T) {
	service := newService(t, testsupport.NewStore(), testsupport.NewInventory())

	_, err := service.Get(context.Background(), "ord_missing")
	if !errors.Is(err, core.ErrOrderNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, core.ErrOrderNotFound)
	}
}

func TestListRejectsNegativeLimit(t *testing.T) {
	service := newService(t, testsupport.NewStore(), testsupport.NewInventory())

	_, err := service.List(context.Background(), core.Filter{Limit: -1})
	if !errors.Is(err, core.ErrInvalidRequest) {
		t.Fatalf("List() error = %v, want %v", err, core.ErrInvalidRequest)
	}
}

func TestNewServiceNeedsDependencies(t *testing.T) {
	if _, err := core.NewService(nil, testsupport.NewInventory()); err == nil {
		t.Error("NewService() without a store = nil, want an error")
	}
	if _, err := core.NewService(testsupport.NewStore(), nil); err == nil {
		t.Error("NewService() without inventory = nil, want an error")
	}
}

func newService(t *testing.T, orders *testsupport.Store, stock *testsupport.Inventory) *core.Service {
	t.Helper()

	service, err := core.NewService(orders, stock,
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
