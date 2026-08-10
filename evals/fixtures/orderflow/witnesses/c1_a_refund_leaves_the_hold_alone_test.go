//witness:dest services/orders/internal/core
//witness:red none

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// A refund settles money and, for goods that have left, stock. The warehouse
// hold belongs to the cancel path: releasing it here would give the whole
// order's reservation away for one returned line.
func TestWitnessARefundLeavesTheHoldAlone(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}

	t.Run("one line of an order the warehouse still holds", func(t *testing.T) {
		orders, stock := testsupport.NewStore(orderWith(core.StatusPaid, lamp)), testsupport.NewInventory()
		service := newService(t, orders, stock)

		if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_1",
			Key:     "sup-1",
			Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
		}); err != nil {
			t.Fatalf("RefundOrder() error = %v", err)
		}
		if releases := stock.Releases(); len(releases) != 0 {
			t.Fatalf("a refund gave %d holds back, want the hold left to the cancel path", len(releases))
		}
	})

	t.Run("the whole of an order that was delivered", func(t *testing.T) {
		orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
		service := newService(t, orders, stock)

		if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_1",
			Key:     "sup-1",
		}); err != nil {
			t.Fatalf("RefundOrder() error = %v", err)
		}
		if releases := stock.Releases(); len(releases) != 0 {
			t.Fatalf("a refund gave %d holds back, want the hold left to the cancel path", len(releases))
		}
	})
}
