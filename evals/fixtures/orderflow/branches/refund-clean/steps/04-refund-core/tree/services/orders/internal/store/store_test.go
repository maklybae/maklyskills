package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/store"
	"orderflow/services/orders/internal/testsupport"
)

var placedAt = time.Date(2024, time.June, 2, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	testsupport.Main(m)
}

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

// The names in a stored document are a contract with every build that wrote one
// before this change.
func TestStoredDocumentKeepsItsFieldNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.json")
	document := `{
  "orders": {
    "ord_1": {
      "id": "ord_1",
      "customer_id": "cust-17",
      "items": [{"sku": "desk-lamp", "quantity": 2, "unit_price": 4900}],
      "status": "delivered",
      "placed_at": "2026-05-04T09:00:00Z",
      "updated_at": "2026-05-06T09:00:00Z"
    }
  }
}
`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatalf("write document: %v", err)
	}

	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}
	stored, err := orders.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(stored.Items) != 1 || stored.Items[0].UnitPrice != 4900 {
		t.Fatalf("items = %+v, want the price the document carries", stored.Items)
	}
	if stored.Total() != 9800 || stored.Status != core.StatusDelivered {
		t.Fatalf("order = %+v, want the document read as it was written", stored)
	}
	if stored.PlacedAt.IsZero() {
		t.Fatal("placed_at was not understood")
	}
}

// A build that predates refunds drops them from the order document. What was
// given back has to survive that.
func TestRefundsSurviveABuildThatDoesNotKnowThem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.json")
	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}

	order := orderFor("ord_1", "cust-17", core.StatusDelivered, placedAt)
	order.Refunds = []core.Refund{{
		ID:       "ord_1-r1",
		Key:      "sup-1",
		Amount:   4900,
		Lines:    []core.Item{{SKU: "desk-lamp", Quantity: 1}},
		IssuedAt: placedAt,
	}}
	if err := orders.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	dropRefunds(t, path)

	reopened, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("second NewJSONFile() error = %v", err)
	}
	stored, err := reopened.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 || stored.Refunded() != 4900 {
		t.Fatalf("refunds = %+v, want the 4900 that was given back", stored.Refunds)
	}
}

// dropRefunds rewrites the order document the way a build without the field
// would: everything it understands, nothing it does not.
func dropRefunds(t *testing.T, path string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read document: %v", err)
	}

	var document map[string]map[string]map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode document: %v", err)
	}
	for _, order := range document["orders"] {
		delete(order, "refunds")
	}

	rewritten, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode document: %v", err)
	}
	if err := os.WriteFile(path, rewritten, 0o600); err != nil {
		t.Fatalf("write document: %v", err)
	}
}

func newStore(t *testing.T) core.Store {
	t.Helper()

	switch mode := testsupport.Mode(); mode {
	case testsupport.ModeJSONFile:
		orders, err := store.NewJSONFile(filepath.Join(t.TempDir(), "orders.json"))
		if err != nil {
			t.Fatalf("NewJSONFile() error = %v", err)
		}
		return orders
	default:
		return store.NewMemory()
	}
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
