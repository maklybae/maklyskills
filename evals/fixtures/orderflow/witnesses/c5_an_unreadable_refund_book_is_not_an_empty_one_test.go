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

// A refund book the store cannot read is not a book that says no refund was
// ever given: the store either reports the failure or keeps what it recorded.
func TestWitnessAnUnreadableRefundBookIsNotAnEmptyOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")
	book := witnessSeedRefundBook(t, dir, path)

	if err := os.Chmod(book, 0o000); err != nil {
		t.Fatalf("chmod %s: %v", book, err)
	}
	t.Cleanup(func() { _ = os.Chmod(book, 0o600) })

	// The rollback the book exists for: an older build rewrites the document
	// without the field it does not know.
	witnessDropRefunds(t, path)

	reopened, err := store.NewJSONFile(path)
	if err != nil {
		// Refusing to open is the honest answer to a book that cannot be read.
		return
	}
	stored, err := reopened.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 || stored.Refunded() != 4900 {
		t.Fatalf("a book that could not be read was taken for an empty one: refunds = %+v", stored.Refunds)
	}
}

// witnessSeedRefundBook writes an order carrying one refund and answers with the
// path of the book beside it, skipping when this build keeps no such file.
func witnessSeedRefundBook(t *testing.T, dir, path string) string {
	t.Helper()

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

	book := filepath.Join(dir, "orders.refunds.json")
	if _, err := os.Stat(book); err != nil {
		t.Skip("this build keeps refunds in the order document alone")
	}
	return book
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
