//witness:dest services/orders/internal/store
//witness:red refund-clean

package store_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/store"
)

// The refund book outlives a document that no longer names the order: an
// operator restoring an older document is exactly what it was written for.
func TestWitnessTheRefundBookKeepsOrdersTheDocumentForgot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}
	if err := orders.Create(context.Background(), witnessRefundedOrder("ord_1")); err != nil {
		t.Fatalf("Create(ord_1) error = %v", err)
	}

	book := filepath.Join(dir, "orders.refunds.json")
	if _, err := os.Stat(book); err != nil {
		t.Skip("this build keeps refunds in the order document alone")
	}

	// An operator restores a document from before ord_1 was placed.
	if err := os.WriteFile(path, []byte(`{"orders":{}}`+"\n"), 0o600); err != nil {
		t.Fatalf("restore document: %v", err)
	}

	if err := orders.Create(context.Background(), witnessRefundedOrder("ord_2")); err != nil {
		t.Fatalf("Create(ord_2) error = %v", err)
	}

	raw, err := os.ReadFile(book)
	if err != nil {
		t.Fatalf("read %s: %v", book, err)
	}
	if !strings.Contains(string(raw), "ord_1") {
		t.Fatalf("%s no longer names ord_1, so what it gave back is gone", filepath.Base(book))
	}
}

func witnessRefundedOrder(id string) core.Order {
	order := orderFor(id, "cust-17", core.StatusDelivered, placedAt)
	order.Refunds = []core.Refund{{
		ID:       id + "-r1",
		Key:      "sup-1",
		Amount:   4900,
		Lines:    []core.Item{{SKU: "desk-lamp", Quantity: 1}},
		IssuedAt: placedAt,
	}}
	return order
}
