package store

import (
	"context"
	"slices"
	"strings"
	"sync"

	"orderflow/pkg/xerrors"
	"orderflow/services/inventory/internal/core"
)

// Memory holds the shelf in a map. The mutex covers both maps: a reservation
// has to see the levels it checked.
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

// Reserve checks every line and only then writes, so two callers cannot both
// pass a check that only one of them can pay for.
func (m *Memory) Reserve(_ context.Context, orderID string, lines []core.Line) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, held := m.reservations[orderID]; held {
		return nil
	}

	for _, line := range lines {
		level, ok := m.levels[line.SKU]
		if !ok {
			return xerrors.Wrapf(core.ErrUnknownSKU, "reserve %s", line.SKU)
		}
		if level.Available() < line.Quantity {
			return xerrors.Wrapf(core.ErrOutOfStock, "reserve %d of %s, %d available",
				line.Quantity, line.SKU, level.Available())
		}
	}

	for _, line := range lines {
		level := m.levels[line.SKU]
		level.Reserved += line.Quantity
		m.levels[line.SKU] = level
	}
	m.reservations[orderID] = slices.Clone(lines)
	return nil
}

func (m *Memory) Release(_ context.Context, orderID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines, held := m.reservations[orderID]
	if !held {
		return false, nil
	}

	for _, line := range lines {
		level, ok := m.levels[line.SKU]
		if !ok {
			continue
		}
		level.Reserved = max(level.Reserved-line.Quantity, 0)
		m.levels[line.SKU] = level
	}
	delete(m.reservations, orderID)
	return true, nil
}

func (m *Memory) Restock(_ context.Context, sku string, quantity int) (core.Level, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	level, ok := m.levels[sku]
	if !ok {
		level = core.Level{SKU: sku}
	}
	level.OnHand += quantity
	m.levels[sku] = level
	return level, nil
}
