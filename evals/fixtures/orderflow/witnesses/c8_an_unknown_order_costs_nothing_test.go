//witness:dest services/orders/internal/core
//witness:red none

package core_test

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"testing"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// The refund endpoint takes an order id from the caller. An id the service has
// never heard of is refused and leaves nothing of itself behind, so a stream of
// invented ids cannot grow the process.
func TestWitnessAnUnknownOrderCostsNothing(t *testing.T) {
	const (
		attempts = 200_000
		budget   = 50_000
	)

	orders := testsupport.NewStore(orderWith(core.StatusDelivered,
		core.Item{SKU: "desk-lamp", Quantity: 1, UnitPrice: 4900}))
	service := newService(t, orders, testsupport.NewInventory())

	before := liveObjects()
	for attempt := range attempts {
		_, err := service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_" + strconv.Itoa(attempt) + "_unknown",
			Key:     "sup-1",
		})
		if !errors.Is(err, core.ErrOrderNotFound) {
			t.Fatalf("RefundOrder() of an unknown order = %v, want %v", err, core.ErrOrderNotFound)
		}
	}
	grew := liveObjects() - before

	if grew > budget {
		t.Fatalf("%d invented ids left %d objects behind, want under %d", attempts, grew, budget)
	}
}

func liveObjects() int64 {
	runtime.GC()
	runtime.GC()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return int64(stats.HeapObjects)
}
