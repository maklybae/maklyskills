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

// An order in pending was never charged for, so there is nothing to give back.
func TestWitnessPendingOrderIsNotRefundable(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPending, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
	})
	if !errors.Is(err, core.ErrNotRefundable) {
		t.Fatalf("RefundOrder() = %+v, error = %v, want %v", refund, err, core.ErrNotRefundable)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Refunded() != 0 {
		t.Fatalf("an unpaid order gave back %d", stored.Refunded())
	}
}
