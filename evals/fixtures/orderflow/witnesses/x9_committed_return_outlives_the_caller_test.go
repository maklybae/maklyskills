//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// The money is committed before the stock moves, so the caller hanging up must
// not be what decides whether the shelf is right.
func TestWitnessCommittedReturnOutlivesTheCaller(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	refund, err := service.RefundOrder(ctx, core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}
	if refund.Amount != 4900 {
		t.Fatalf("amount = %d, want 4900", refund.Amount)
	}

	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory saw %d returns, want the committed return to go out", len(returns))
	}
}
