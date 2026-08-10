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

// One sku, one line: a request that names the same sku twice is refused rather
// than priced twice.
func TestWitnessASkuOnTwoLinesIsRefused(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))
	service := newService(t, orders, testsupport.NewInventory())

	refund, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}, {SKU: "desk-lamp", Quantity: 1}},
	})
	if !errors.Is(err, core.ErrInvalidRequest) {
		t.Fatalf("RefundOrder() = %+v, error = %v, want %v", refund, err, core.ErrInvalidRequest)
	}
}
