//witness:dest services/orders/internal/core
//witness:red none

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// A line the customer was not charged for is still a line that can come back.
// Only a refund that names nothing at all can be turned away for being worth
// nothing.
func TestWitnessALineThatCostNothingIsRefundable(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	sample := core.Item{SKU: "swatch-card", Quantity: 1, UnitPrice: 0}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp, sample))
	service := newService(t, orders, testsupport.NewInventory())

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "swatch-card", Quantity: 1}},
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Fatalf("the order holds %d refunds, want the one that was issued", len(stored.Refunds))
	}
	if len(stored.Refunds[0].Lines) != 1 || stored.Refunds[0].Lines[0].SKU != "swatch-card" {
		t.Fatalf("the refund records %+v, want the line that came back", stored.Refunds[0].Lines)
	}
}
