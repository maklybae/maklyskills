package store

import (
	"context"
	"slices"
	"strings"
	"sync"

	"orderflow/pkg/xerrors"
	"orderflow/services/inventory/internal/core"
)

// Memory holds the shelf and the reservations against it in two maps.
type Memory struct {
	mu           sync.Mutex
	levels       map[string]core.Level
	reservations map[string][]core.Line
	snapshots    *SnapshotCache
}

// NewMemory returns a store holding the given levels.
func NewMemory(levels ...core.Level) *Memory {
	store := &Memory{
		levels:       make(map[string]core.Level, len(levels)),
		reservations: make(map[string][]core.Line),
		snapshots:    NewSnapshotCache(),
	}
	for _, level := range levels {
		store.levels[level.SKU] = level
	}
	return store
}

// LoadSnapshot replaces the shelf with the warehouse export at path.
// TODO: the export carries no timestamp, so a file that stopped being
// written looks as fresh as one from this morning.
func (m *Memory) LoadSnapshot(path string) error {
	levels, err := m.snapshots.Load(path)
	if err != nil {
		return xerrors.Wrapf(err, "load stock snapshot %s", path)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.levels = make(map[string]core.Level, len(levels))
	for _, level := range levels {
		m.levels[level.SKU] = level
	}
	return nil
}

func (m *Memory) Level(_ context.Context, sku string) (core.Level, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	level, ok := m.levels[sku]
	if !ok {
		return core.Level{}, xerrors.Wrapf(core.ErrUnknownSKU, "stock of %s", sku)
	}
	return level, nil
}

func (m *Memory) Levels(_ context.Context) ([]core.Level, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	levels := make([]core.Level, 0, len(m.levels))
	for _, level := range m.levels {
		levels = append(levels, level)
	}
	slices.SortFunc(levels, func(a, b core.Level) int { return strings.Compare(a.SKU, b.SKU) })
	return levels, nil
}

func (m *Memory) SetLevel(_ context.Context, level core.Level) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.levels[level.SKU] = level
	return nil
}

func (m *Memory) Reservation(_ context.Context, orderID string) ([]core.Line, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines, held := m.reservations[orderID]
	if !held {
		return nil, false, nil
	}
	return slices.Clone(lines), true, nil
}

func (m *Memory) SaveReservation(_ context.Context, orderID string, lines []core.Line) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reservations[orderID] = slices.Clone(lines)
	return nil
}

func (m *Memory) DeleteReservation(_ context.Context, orderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.reservations, orderID)
	return nil
}
