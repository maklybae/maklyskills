//witness:dest services/orders/internal/core
//witness:red refund-bugs,refund-clean

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// Units the customer has been paid back for do not walk out of the warehouse:
// either the order stops at the door, or what leaves no longer counts them.
func TestWitnessRefundedUnitsDoNotLeaveTheWarehouse(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}
	if refund.Amount != 4900 {
		t.Fatalf("refund = %d, want the price of the one lamp that came back", refund.Amount)
	}

	shipped, err := service.MarkShipped(context.Background(), "ord_1")
	if err != nil {
		// The order stopped at the door, which is one of the two right answers.
		return
	}

	for _, item := range shipped.Items {
		if item.SKU == "desk-lamp" && item.Quantity > 1 {
			t.Fatalf("%d lamps left the warehouse for an order that was paid back for one",
				item.Quantity)
		}
	}
}
