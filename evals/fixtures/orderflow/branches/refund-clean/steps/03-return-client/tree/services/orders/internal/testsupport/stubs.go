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

// Inventory records what the order flow asked of stock.
type Inventory struct {
	mu         sync.Mutex
	reserved   []core.StockReservation
	released   []core.StockRelease
	returned   []core.StockReturn
	reserveErr error
	releaseErr error
	restockErr error
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

// NewInventory returns an inventory that accepts everything.
func NewInventory() *Inventory {
	return &Inventory{}
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

func (i *Inventory) Reserve(ctx context.Context, reservation core.StockReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.reserveErr != nil {
		return i.reserveErr
	}
	i.reserved = append(i.reserved, reservation)
	return nil
}

func (i *Inventory) Release(ctx context.Context, release core.StockRelease) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.releaseErr != nil {
		return i.releaseErr
	}
	i.released = append(i.released, release)
	return nil
}

func (i *Inventory) Restock(ctx context.Context, ret core.StockReturn) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	i.returned = append(i.returned, ret)
	if i.restockErr != nil {
		return i.restockErr
	}
	return nil
}

// Reservations returns the reservations made so far.
func (i *Inventory) Reservations() []core.StockReservation {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]core.StockReservation(nil), i.reserved...)
}

// Releases returns the releases made so far.
func (i *Inventory) Releases() []core.StockRelease {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]core.StockRelease(nil), i.released...)
}

// Returns lists the stock returns inventory was asked for.
func (i *Inventory) Returns() []core.StockReturn {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]core.StockReturn(nil), i.returned...)
}

// FailRestock makes every following Restock return err after recording it,
// which is how a call that reached inventory and then failed on the way back
// is reproduced.
func (i *Inventory) FailRestock(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.restockErr = err
}

// FailReserve makes every following Reserve return err.
func (i *Inventory) FailReserve(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.reserveErr = err
}

// FailRelease makes every following Release return err.
func (i *Inventory) FailRelease(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.releaseErr = err
}

func (i *IDs) NextID() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.next++
	return "ord_" + strconv.Itoa(i.next)
}
