//witness:dest services/orders/internal/store
//witness:red refund-clean

package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/store"
)

// What a write does to the refunds of one order does not depend on whether some
// unrelated order still has refunds of its own.
func TestWitnessARemovedRefundStaysRemoved(t *testing.T) {
	alone := refundAfterOverwrite(t, false)
	beside := refundAfterOverwrite(t, true)
	if alone != beside {
		t.Fatalf("an overwritten order kept %d refunds on its own and %d beside a refunded "+
			"neighbour, so what survives a stale write depends on unrelated orders", alone, beside)
	}
}

// refundAfterOverwrite records a refund, writes the order back the way a stale
// caller would - without it - and answers with how many refunds are left.
func refundAfterOverwrite(t *testing.T, neighbour bool) int {
	t.Helper()

	path := filepath.Join(t.TempDir(), "orders.json")
	orders, err := store.NewJSONFile(path)
	if err != nil {
		t.Fatalf("NewJSONFile() error = %v", err)
	}

	order := orderFor("ord_1", "cust-17", core.StatusDelivered, placedAt)
	order.Refunds = []core.Refund{{ID: "ord_1-r1", Key: "sup-1", Amount: 4900, IssuedAt: placedAt}}
	if err := orders.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if neighbour {
		other := orderFor("ord_2", "cust-42", core.StatusDelivered, placedAt)
		other.Refunds = []core.Refund{{ID: "ord_2-r1", Key: "sup-2", Amount: 900, IssuedAt: placedAt}}
		if err := orders.Create(context.Background(), other); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	stale := order
	stale.Refunds = nil
	if err := orders.Update(context.Background(), stale); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	stored, err := orders.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return len(stored.Refunds)
}
