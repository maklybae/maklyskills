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

// A store call that comes back as a failure has not written the order it was
// given: the caller undoes what it reserved on the strength of that answer.
func TestWitnessAFailedStoreCallLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}

	refunded := orderFor("ord_1", "cust-17", core.StatusDelivered, placedAt)
	refunded.Refunds = []core.Refund{{ID: "ord_1-r1", Key: "sup-1", Amount: 4900, IssuedAt: placedAt}}
	if err := orders.Create(context.Background(), refunded); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	book := filepath.Join(dir, "orders.refunds.json")
	if _, err := os.Stat(book); err != nil {
		t.Skip("this build keeps refunds in the order document alone")
	}

	// Take the second write away: a directory cannot be replaced by a file.
	if err := os.Remove(book); err != nil {
		t.Fatalf("remove %s: %v", book, err)
	}
	if err := os.Mkdir(book, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", book, err)
	}

	err = orders.Create(context.Background(), orderFor("ord_2", "cust-17", core.StatusPending, placedAt))
	if err == nil {
		// Nothing failed, so there is nothing to be half done.
		return
	}

	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read document: %v", readErr)
	}
	if strings.Contains(string(raw), "ord_2") {
		t.Fatalf("Create(ord_2) answered %v and the order is in %s anyway", err, filepath.Base(path))
	}
}
