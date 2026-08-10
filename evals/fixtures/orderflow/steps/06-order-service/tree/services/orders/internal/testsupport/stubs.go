// Package testsupport holds what the orders tests share: stand-ins for the
// service dependencies and the switch that picks a store backend.
package testsupport

import (
	"context"
	"strconv"
	"sync"
	"time"

	"orderflow/pkg/xerrors"
	"orderflow/services/orders/internal/core"
)

// Store is a core.Store for tests that do not care how orders are written down.
// Every method can be told to fail, which is how error paths get covered.
type Store struct {
	mu        sync.Mutex
	orders    map[string]core.Order
	createErr error
	getErr    error
	updateErr error
	listErr   error
}

// IDs hands out ord_1, ord_2, ... so tests can name the order they expect.
type IDs struct {
	mu   sync.Mutex
	next int
}

// NewStore returns a store already holding the given orders.
func NewStore(seed ...core.Order) *Store {
	orders := make(map[string]core.Order, len(seed))
	for _, order := range seed {
		orders[order.ID] = order.Clone()
	}
	return &Store{orders: orders}
}

// Clock returns a clock stuck at instant.
func Clock(instant time.Time) func() time.Time {
	return func() time.Time { return instant }
}

func (s *Store) Create(_ context.Context, order core.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.createErr != nil {
		return s.createErr
	}
	if _, exists := s.orders[order.ID]; exists {
		return xerrors.Wrapf(core.ErrOrderExists, "create order %s", order.ID)
	}
	s.orders[order.ID] = order.Clone()
	return nil
}

func (s *Store) Get(_ context.Context, id string) (core.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getErr != nil {
		return core.Order{}, s.getErr
	}
	order, ok := s.orders[id]
	if !ok {
		return core.Order{}, xerrors.Wrapf(core.ErrOrderNotFound, "get order %s", id)
	}
	return order.Clone(), nil
}

func (s *Store) Update(_ context.Context, order core.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.updateErr != nil {
		return s.updateErr
	}
	if _, exists := s.orders[order.ID]; !exists {
		return xerrors.Wrapf(core.ErrOrderNotFound, "update order %s", order.ID)
	}
	s.orders[order.ID] = order.Clone()
	return nil
}

func (s *Store) List(_ context.Context, filter core.Filter) ([]core.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listErr != nil {
		return nil, s.listErr
	}
	var selected []core.Order
	for _, order := range s.orders {
		if filter.Matches(order) {
			selected = append(selected, order.Clone())
		}
	}
	return selected, nil
}

// FailCreate makes every following Create return err.
func (s *Store) FailCreate(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createErr = err
}

// FailGet makes every following Get return err.
func (s *Store) FailGet(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getErr = err
}

// FailUpdate makes every following Update return err.
func (s *Store) FailUpdate(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateErr = err
}

// FailList makes every following List return err.
func (s *Store) FailList(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listErr = err
}

func (i *IDs) NextID() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.next++
	return "ord_" + strconv.Itoa(i.next)
}
