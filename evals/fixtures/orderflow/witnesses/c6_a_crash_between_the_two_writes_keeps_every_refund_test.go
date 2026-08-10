//witness:dest services/orders/internal/store
//witness:red refund-clean

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

// A crash between the order document and the refund book beside it must not
// cost a refund the rollback the book exists for.
func TestWitnessACrashBetweenTheTwoWritesKeepsEveryRefund(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}

	order := orderFor("ord_1", "cust-17", core.StatusDelivered, placedAt)
	order.Refunds = []core.Refund{{ID: "ord_1-r1", Key: "sup-1", Amount: 1000, IssuedAt: placedAt}}
	if err := orders.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	book := filepath.Join(dir, "orders.refunds.json")
	afterFirst, err := os.ReadFile(book)
	if err != nil {
		t.Skip("this build keeps refunds in the order document alone")
	}

	order.Refunds = append(order.Refunds, core.Refund{ID: "ord_1-r2", Key: "sup-2", Amount: 2000, IssuedAt: placedAt})
	if err := orders.Update(context.Background(), order); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// The crash: the order document made it, the second write did not.
	if err := os.WriteFile(book, afterFirst, 0o600); err != nil {
		t.Fatalf("restore %s: %v", book, err)
	}

	witnessDropRefunds(t, path)

	reopened, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("second NewJSONFile() error = %v", err)
	}
	stored, err := reopened.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 2 || stored.Refunded() != 3000 {
		t.Fatalf("after the crash the order gives back %d over %d refunds, want 3000 over 2",
			stored.Refunded(), len(stored.Refunds))
	}
}

// witnessDropRefunds rewrites the order document the way a build without the
// field would: everything it understands, nothing it does not.
func witnessDropRefunds(t *testing.T, path string) {
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
