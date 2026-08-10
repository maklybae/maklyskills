//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// A refund is worth what the returned lines cost, not a share of the order.
func TestWitnessPartialRefundPricesTheLines(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	tray := core.Item{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp, tray))
	service := newService(t, orders, testsupport.NewInventory())

	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "cable-tray", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	if refund.Amount != 900 {
		t.Fatalf("one returned tray refunded %d, want 900", refund.Amount)
	}
}
