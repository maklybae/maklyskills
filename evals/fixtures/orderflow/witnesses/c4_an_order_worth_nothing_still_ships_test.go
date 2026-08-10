//witness:dest services/orders/internal/core
//witness:red none

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// An order that cost nothing has nothing to give back, so nothing about refunds
// may stand between it and the carrier.
func TestWitnessAnOrderWorthNothingStillShips(t *testing.T) {
	sample := core.Item{SKU: "swatch-card", Quantity: 1, UnitPrice: 0}
	orders := testsupport.NewStore(orderWith(core.StatusPaid, sample))
	service := newService(t, orders, testsupport.NewInventory())

	shipped, err := service.MarkShipped(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("MarkShipped() error = %v", err)
	}
	if shipped.Status != core.StatusShipped {
		t.Fatalf("status = %s, want %s", shipped.Status, core.StatusShipped)
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Status != core.StatusShipped {
		t.Fatalf("stored status = %s, want %s", stored.Status, core.StatusShipped)
	}
}
