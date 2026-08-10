//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"errors"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// Goods that have been paid back do not leave the building.
func TestWitnessRefundedOrdersDoNotShip(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-1",
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	shipped, err := service.MarkShipped(context.Background(), "ord_1")
	if !errors.Is(err, core.ErrNotShippable) {
		t.Fatalf("MarkShipped() = %+v, error = %v, want %v", shipped, err, core.ErrNotShippable)
	}
}
