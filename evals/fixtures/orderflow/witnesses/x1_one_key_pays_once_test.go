//witness:dest services/orders/internal/core
//witness:red refund-bugs

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

// witnessStore holds the first write open so two callers under one key overlap
// on purpose rather than by luck.
type witnessStore struct {
	mu     sync.Mutex
	orders map[string]core.Order

	reads   atomic.Int32
	writes  atomic.Int32
	read    chan struct{}
	parked  chan struct{}
	proceed chan struct{}
}

func newWitnessStore(seed core.Order) *witnessStore {
	return &witnessStore{
		orders:  map[string]core.Order{seed.ID: seed.Clone()},
		read:    make(chan struct{}),
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
	order, known := s.orders[id]
	s.mu.Unlock()

	// The second read is what says the other caller got to the order before the
	// first write landed.
	if s.reads.Add(1) == 2 {
		close(s.read)
	}
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

// Two callers arriving under one key while the first write is still in flight
// are paid once between them, and inventory hears about it once.
func TestWitnessOneKeyPaysOnce(t *testing.T) {
	lamp := core.Item{SKU: "desk-lamp", Quantity: 2, UnitPrice: 4900}
	orders, stock := newWitnessStore(orderWith(core.StatusDelivered, lamp)), testsupport.NewInventory()

	service, err := core.NewService(orders, stock,
		core.WithClock(testsupport.Clock(now)),
		core.WithIDSource(&testsupport.IDs{}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	request := core.RefundRequest{
		OrderID: "ord_1",
		Key:     "sup-1",
		Lines:   []core.Item{{SKU: "desk-lamp", Quantity: 1}},
	}

	first := make(chan error, 1)
	go func() {
		_, err := service.RefundOrder(context.Background(), request)
		first <- err
	}()

	select {
	case <-orders.parked:
	case <-time.After(2 * time.Second):
		t.Fatal("the first refund never reached the store")
	}

	second := make(chan error, 1)
	go func() {
		_, err := service.RefundOrder(context.Background(), request)
		second <- err
	}()

	// A service that holds the order for the whole refund never lets the second
	// caller read it, so the wait is a backstop rather than the mechanism.
	select {
	case <-orders.read:
	case <-time.After(time.Second):
	}
	close(orders.proceed)

	for _, done := range []chan error{first, second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("RefundOrder() error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a refund never came back")
		}
	}

	if returns := stock.Returns(); len(returns) != 1 {
		t.Fatalf("inventory saw %d returns for one key, want 1", len(returns))
	}

	stored, err := service.Get(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(stored.Refunds) != 1 {
		t.Fatalf("the order holds %d refunds for one key, want 1", len(stored.Refunds))
	}
	if stored.Refunded() != 4900 {
		t.Fatalf("one key gave back %d, want 4900", stored.Refunded())
	}
}
