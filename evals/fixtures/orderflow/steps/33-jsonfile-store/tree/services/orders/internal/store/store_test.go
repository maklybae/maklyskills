package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/store"
)

var placedAt = time.Date(2024, time.June, 2, 12, 0, 0, 0, time.UTC)

func TestWriteAndRead(t *testing.T) {
	tests := []struct {
		name    string
		run     func(*testing.T, core.Store, core.Order)
		wantErr error
	}{
		{
			name: "an order comes back as it went in",
			run: func(t *testing.T, orders core.Store, order core.Order) {
				stored, err := orders.Get(context.Background(), order.ID)
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}
				if stored.CustomerID != order.CustomerID || stored.Total() != order.Total() {
					t.Fatalf("stored %+v, want %+v", stored, order)
				}
			},
		},
		{
			name: "a second create is refused",
			run: func(t *testing.T, orders core.Store, order core.Order) {
				err := orders.Create(context.Background(), order)
				if !errors.Is(err, core.ErrOrderExists) {
					t.Fatalf("Create() error = %v, want %v", err, core.ErrOrderExists)
				}
			},
		},
		{
			name: "an update is visible",
			run: func(t *testing.T, orders core.Store, order core.Order) {
				order.Status = core.StatusPaid
				if err := orders.Update(context.Background(), order); err != nil {
					t.Fatalf("Update() error = %v", err)
				}
				stored, err := orders.Get(context.Background(), order.ID)
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}
				if stored.Status != core.StatusPaid {
					t.Fatalf("status = %s, want %s", stored.Status, core.StatusPaid)
				}
			},
		},
		{
			name: "an unknown order is not found",
			run: func(t *testing.T, orders core.Store, _ core.Order) {
				_, err := orders.Get(context.Background(), "ord_missing")
				if !errors.Is(err, core.ErrOrderNotFound) {
					t.Fatalf("Get() error = %v, want %v", err, core.ErrOrderNotFound)
				}
			},
		},
		{
			name: "an unknown order cannot be updated",
			run: func(t *testing.T, orders core.Store, _ core.Order) {
				err := orders.Update(context.Background(), core.Order{ID: "ord_missing"})
				if !errors.Is(err, core.ErrOrderNotFound) {
					t.Fatalf("Update() error = %v, want %v", err, core.ErrOrderNotFound)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := newStore(t)
			order := orderFor("ord_1", "cust-17", core.StatusPending, placedAt)
			if err := orders.Create(context.Background(), order); err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			tt.run(t, orders, order)
		})
	}
}

func TestList(t *testing.T) {
	seed := []core.Order{
		orderFor("ord_1", "cust-17", core.StatusPending, placedAt),
		orderFor("ord_2", "cust-17", core.StatusShipped, placedAt.Add(time.Hour)),
		orderFor("ord_3", "cust-42", core.StatusPending, placedAt.Add(2*time.Hour)),
	}

	tests := []struct {
		name   string
		filter core.Filter
		want   []string
	}{
		{name: "everything, oldest first", filter: core.Filter{}, want: []string{"ord_1", "ord_2", "ord_3"}},
		{name: "by customer", filter: core.Filter{CustomerID: "cust-17"}, want: []string{"ord_1", "ord_2"}},
		{
			name:   "by status",
			filter: core.Filter{Statuses: []core.Status{core.StatusPending}},
			want:   []string{"ord_1", "ord_3"},
		},
		{name: "limited", filter: core.Filter{Limit: 2}, want: []string{"ord_1", "ord_2"}},
		{name: "nothing matches", filter: core.Filter{CustomerID: "cust-99"}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := newStore(t)
			for _, order := range seed {
				if err := orders.Create(context.Background(), order); err != nil {
					t.Fatalf("Create(%s) error = %v", order.ID, err)
				}
			}

			found, err := orders.List(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			ids := make([]string, 0, len(found))
			for _, order := range found {
				ids = append(ids, order.ID)
			}
			if len(ids) != len(tt.want) {
				t.Fatalf("List() = %v, want %v", ids, tt.want)
			}
			for i, id := range ids {
				if id != tt.want[i] {
					t.Fatalf("List() = %v, want %v", ids, tt.want)
				}
			}
		})
	}
}

// The store hands out copies; a caller that edits what it got must not change
// what is stored.
func TestStoredOrdersAreIsolated(t *testing.T) {
	orders := newStore(t)
	order := orderFor("ord_1", "cust-17", core.StatusPending, placedAt)
	if err := orders.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	order.Items[0].Quantity = 99

	stored, err := orders.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Items[0].Quantity != 1 {
		t.Fatalf("quantity = %d, want 1", stored.Items[0].Quantity)
	}
}

func TestJSONFileSurvivesAReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.json")
	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}
	if err := orders.Create(context.Background(), orderFor("ord_1", "cust-17", core.StatusPaid, placedAt)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	reopened, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("second NewJSONFile() error = %v", err)
	}
	stored, err := reopened.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Status != core.StatusPaid || stored.CustomerID != "cust-17" {
		t.Fatalf("reopened order = %+v, want the one that was written", stored)
	}
}

func newStore(t *testing.T) core.Store {
	t.Helper()

	return store.NewMemory()
}

func orderFor(id, customer string, status core.Status, placed time.Time) core.Order {
	return core.Order{
		ID:         id,
		CustomerID: customer,
		Items:      []core.Item{{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}},
		Status:     status,
		PlacedAt:   placed,
		UpdatedAt:  placed,
	}
}
