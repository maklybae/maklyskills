//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// An order that has been refunded in full takes no more refund records, so a
// caller cannot keep appending rows that carry its own text.
func TestWitnessNothingLeftIsRefused(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1", Key: "sup-1",
	}); err != nil {
		t.Fatalf("first RefundOrder() error = %v", err)
	}

	for attempt := 2; attempt <= 4; attempt++ {
		_, err := service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_1", Key: "sup-" + strconv.Itoa(attempt),
		})
		if !errors.Is(err, core.ErrInvalidRequest) && !errors.Is(err, core.ErrRefundTooLarge) {
			t.Fatalf("refund %d of nothing = %v, want a refusal", attempt, err)
		}
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Fatalf("the order holds %d refunds, want only the one that moved money", len(stored.Refunds))
	}
}
