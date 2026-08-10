//witness:dest services/orders/internal/store
//witness:red refund-bugs

package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"orderflow/services/orders/internal/store"
)

// A document written by an earlier build of the service still reads back with
// its prices: the field names in it are a contract, not an implementation
// detail.
func TestWitnessStoredOrdersKeepTheirPrices(t *testing.T) {
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

	if len(stored.Items) != 1 {
		t.Fatalf("read %d items, want 1", len(stored.Items))
	}
	if stored.Items[0].UnitPrice != 4900 {
		t.Fatalf("unit price = %d, want 4900: the stored document was not understood", stored.Items[0].UnitPrice)
	}
	if stored.Total() != 9800 {
		t.Fatalf("total = %d, want 9800", stored.Total())
	}
}
