//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// Refunding an order the warehouse still holds does not put units on the shelf
// that never left it.
func TestWitnessUnshippedGoodsAreNotShelved(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusPaid, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if returns := stock.Returns(); len(returns) != 0 {
		t.Fatalf("inventory shelved %d returns of goods that never left", len(returns))
	}
}
