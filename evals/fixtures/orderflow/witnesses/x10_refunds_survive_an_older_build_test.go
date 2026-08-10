//witness:dest services/orders/internal/store
//witness:red refund-bugs

package store_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/store"
)

// A build that predates refunds writes the order document without them. What
// has been given back has to survive that, or a rollback pays every refund
// again.
func TestWitnessRefundsSurviveAnOlderBuild(t *testing.T) {
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

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read document: %v", err)
	}
	var document map[string]map[string]map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode document: %v", err)
	}
	for _, stored := range document["orders"] {
		delete(stored, "refunds")
	}
	rewritten, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode document: %v", err)
	}
	if err := os.WriteFile(path, rewritten, 0o600); err != nil {
		t.Fatalf("write document: %v", err)
	}

	reopened, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("second NewJSONFile() error = %v", err)
	}
	back, err := reopened.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(back.Refunds) != 1 || back.Refunded() != 4900 {
		t.Fatalf("refunds after the older build = %+v, want the 4900 that was given back", back.Refunds)
	}
}
