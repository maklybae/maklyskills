package store

import (
	"context"
	"slices"
	"strings"
	"sync"

	"orderflow/pkg/xerrors"
	"orderflow/services/orders/internal/core"
)

// Memory keeps orders in a map. It is the default backend and the one the
// tests run against; nothing survives a restart.
type Memory struct {
	mu     sync.RWMutex
	orders map[string]core.Order
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{orders: make(map[string]core.Order)}
}

func (m *Memory) Create(_ context.Context, order core.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.orders[order.ID]; exists {
		return xerrors.Wrapf(core.ErrOrderExists, "create order %s", order.ID)
	}
	order.Version = 1
	m.orders[order.ID] = order.Clone()
	return nil
}

func (m *Memory) Get(_ context.Context, id string) (core.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	order, ok := m.orders[id]
	if !ok {
		return core.Order{}, xerrors.Wrapf(core.ErrOrderNotFound, "get order %s", id)
	}
	return order.Clone(), nil
}

// Update refuses a write built on a version the store has already moved past.
// The caller loads the order again, redoes its change and comes back.
func (m *Memory) Update(_ context.Context, order core.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, exists := m.orders[order.ID]
	if !exists {
		return xerrors.Wrapf(core.ErrOrderNotFound, "update order %s", order.ID)
	}
	if stored.Version != order.Version {
		return xerrors.Wrapf(core.ErrVersionConflict, "update order %s at version %d, the store holds %d",
			order.ID, order.Version, stored.Version)
	}

	order.Version++
	m.orders[order.ID] = order.Clone()
	return nil
}

func (m *Memory) List(_ context.Context, filter core.Filter) ([]core.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	selected := make([]core.Order, 0, len(m.orders))
	for _, order := range m.orders {
		if filter.Matches(order) {
			selected = append(selected, order.Clone())
		}
	}

	slices.SortFunc(selected, func(a, b core.Order) int {
		if !a.PlacedAt.Equal(b.PlacedAt) {
			return a.PlacedAt.Compare(b.PlacedAt)
		}
		return strings.Compare(a.ID, b.ID)
	})

	if filter.Limit > 0 && len(selected) > filter.Limit {
		selected = selected[:filter.Limit]
	}
	return selected, nil
}
