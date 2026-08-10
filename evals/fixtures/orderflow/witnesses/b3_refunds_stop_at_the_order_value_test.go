//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"strconv"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// However a refund is refused, the refunds of an order never add up to more
// than the order was worth.
func TestWitnessRefundsStopAtTheOrderValue(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	line := []core.Item{{SKU: "desk-lamp", Quantity: 1}}
	for attempt := 1; attempt <= 3; attempt++ {
		request := core.RefundRequest{OrderID: "ord_1", Key: "sup-" + strconv.Itoa(attempt), Lines: line}
		_, _ = service.RefundOrder(context.Background(), request)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Refunded() > 9800 {
		t.Fatalf("gave back %d for an order worth 9800", stored.Refunded())
	}
}
