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

// Inventory counts returns instead of keying them, so a return that failed on
// the way back must not be sent again by the service.
func TestWitnessReturnIsSentOnce(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()
	stock.FailRestock(errors.New("inventory timed out"))
	service := newService(t, orders, stock)

	request := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}
	if _, err := service.RefundOrder(context.Background(), request); err == nil {
		t.Fatal("RefundOrder() = nil, want the inventory error")
	}

	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory was sent %d returns, want 1", len(returns))
	}
}
