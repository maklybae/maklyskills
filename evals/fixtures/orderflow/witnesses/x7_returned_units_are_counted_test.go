//witness:dest services/orders/internal/core
//witness:red refund-bugs

package core_test

import (
	"context"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// An order cannot send back more units than it ever held, however many refunds
// it takes to try.
func TestWitnessReturnedUnitsAreCounted(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}
	tray := core.Item{SKU: "cable-tray", Quantity: 3, UnitPrice: 900}
	orders, stock := testsupport.NewStore(orderWith(core.StatusDelivered, lamp, tray)), testsupport.NewInventory()
	service := newService(t, orders, stock)

	trays := []core.Item{{SKU: "cable-tray", Quantity: 3}}
	for attempt, key := range []string{"sup-1", "sup-2"} {
		_, err := service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_1", Key: key, Lines: trays,
		})
		if attempt == 0 && err != nil {
			t.Fatalf("first RefundOrder() error = %v", err)
		}
	}

	shelved := 0
	for _, ret := range stock.Returns() {
		for _, line := range ret.Lines {
			if line.SKU == "cable-tray" {
				shelved += line.Quantity
			}
		}
	}
	if shelved > 3 {
		t.Fatalf("inventory took back %d trays for an order that held 3", shelved)
	}
}
