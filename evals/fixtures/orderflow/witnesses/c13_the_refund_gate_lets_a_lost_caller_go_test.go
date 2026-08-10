//witness:dest services/orders/internal/core
//witness:red refund-clean

package core_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// The gate is there to decide one refund on one version of an order. It is not
// held across the call to inventory, so a stalled warehouse parks a network
// call and not every later caller of the same order.
func TestWitnessTheRefundGateLetsALostCallerGo(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders := testsupport.NewStore(orderWith(core.StatusDelivered, lamp))

	stock := &stallingStock{inflight: make(chan struct{}), release: make(chan struct{})}
	defer stock.let()

	service, err := core.NewService(orders, stock,
		core.WithClock(testsupport.Clock(now)),
		core.WithIDSource(&testsupport.IDs{}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	line := []core.Item{{SKU: "desk-lamp", Quantity: 1}}
	go func() {
		_, _ = service.RefundOrder(context.Background(), core.RefundRequest{
			OrderID: "ord_1", Key: "sup-1", Lines: line,
		})
	}()

	select {
	case <-stock.inflight:
	case <-time.After(2 * time.Second):
		t.Skip("this build does not send the return from inside the refund")
	}

	gone, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = service.RefundOrder(gone, core.RefundRequest{
			OrderID: "ord_1", Key: "sup-2", Lines: line,
		})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a caller that had already hung up waited on a gate held across the return call")
	}
}

// stallingStock holds the first return until it is let go and answers every
// later one at once.
type stallingStock struct {
	inflight chan struct{}
	release  chan struct{}
	once     sync.Once
	freed    sync.Once
	stalled  bool
	mu       sync.Mutex
}

func (s *stallingStock) Reserve(context.Context, core.StockReservation) error { return nil }

func (s *stallingStock) Release(context.Context, core.StockRelease) error { return nil }

func (s *stallingStock) Restock(context.Context, core.StockReturn) error {
	s.mu.Lock()
	first := !s.stalled
	s.stalled = true
	s.mu.Unlock()
	if !first {
		return nil
	}

	s.once.Do(func() { close(s.inflight) })
	<-s.release
	return nil
}

func (s *stallingStock) let() {
	s.freed.Do(func() { close(s.release) })
}
