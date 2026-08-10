//witness:dest services/orders/internal/core
//witness:red refund-bugs,refund-clean

package core_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orderflow/services/orders/internal/core"
	"orderflow/services/orders/internal/testsupport"
)

// witnessStore parks the first write, which is how a transition is held between
// reading an order and writing it back.
type witnessStore struct {
	mu     sync.Mutex
	orders map[string]core.Order

	writes  atomic.Int32
	parked  chan struct{}
	proceed chan struct{}
}

func newWitnessStore(seed core.Order) *witnessStore {
	return &witnessStore{
		orders:  map[string]core.Order{seed.ID: seed.Clone()},
		parked:  make(chan struct{}),
		proceed: make(chan struct{}),
	}
}

func (s *witnessStore) Create(_ context.Context, order core.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = order.Clone()
	return nil
}

func (s *witnessStore) Get(_ context.Context, id string) (core.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, known := s.orders[id]
	if !known {
		return core.Order{}, core.ErrOrderNotFound
	}
	return order.Clone(), nil
}

func (s *witnessStore) Update(_ context.Context, order core.Order) error {
	if s.writes.Add(1) == 1 {
		close(s.parked)
		<-s.proceed
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.orders[order.ID]; !known {
		return core.ErrOrderNotFound
	}
	s.orders[order.ID] = order.Clone()
	return nil
}

func (s *witnessStore) List(context.Context, core.Filter) ([]core.Order, error) {
	return nil, nil
}

// A refund that commits while a transition is in flight survives it: the money
// given back, the record of it, and the key that must not pay twice.
func TestWitnessATransitionDoesNotEraseARefund(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := newWitnessStore(orderWith(core.StatusShipped, lamp)), testsupport.NewInventory()

	service, err := core.NewService(orders, stock,
		core.WithClock(testsupport.Clock(now)),
		core.WithIDSource(&testsupport.IDs{}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	delivered := make(chan error, 1)
	go func() {
		_, err := service.MarkDelivered(context.Background(), "ord_1")
		delivered <- err
	}()

	select {
	case <-orders.parked:
	case <-time.After(2 * time.Second):
		t.Fatal("the transition never reached the store")
	}

	// The refund lands whole while the transition holds a copy from before it.
	if _, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}); err != nil {
		t.Fatalf("RefundOrder() error = %v", err)
	}
	close(orders.proceed)

	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("the transition never came back")
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 || stored.Refunded() != 4900 {
		t.Fatalf("the order holds %+v, want the 4900 the refund gave back", stored.Refunds)
	}

	// The key the refund was issued under must still answer with that refund
	// rather than pay a second time.
	replay, err := service.RefundOrder(context.Background(), core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("replayed RefundOrder() error = %v", err)
	}
	if replay.ID != stored.Refunds[0].ID {
		t.Fatalf("the key issued %s and then %s, want one refund", stored.Refunds[0].ID, replay.ID)
	}
	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory saw %d returns for one refund, want 1", len(returns))
	}
}
