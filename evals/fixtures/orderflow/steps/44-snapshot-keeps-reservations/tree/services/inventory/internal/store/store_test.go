package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"orderflow/services/inventory/internal/core"
	"orderflow/services/inventory/internal/store"
)

const snapshot = `[{"sku":"desk-lamp","on_hand":42},{"sku":"monitor-arm","on_hand":8}]`

func TestMemory(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *store.Memory)
	}{
		{
			name: "levels come back sorted by sku",
			run: func(t *testing.T, levels *store.Memory) {
				found, err := levels.Levels(context.Background())
				if err != nil {
					t.Fatalf("Levels() error = %v", err)
				}
				if len(found) != 2 || found[0].SKU != "desk-lamp" || found[1].SKU != "monitor-arm" {
					t.Fatalf("Levels() = %+v, want desk-lamp then monitor-arm", found)
				}
			},
		},
		{
			name: "a reservation holds stock",
			run: func(t *testing.T, levels *store.Memory) {
				err := levels.Reserve(context.Background(), "ord_1", []core.Line{{SKU: "monitor-arm", Quantity: 3}})
				if err != nil {
					t.Fatalf("Reserve() error = %v", err)
				}
				level, err := levels.Level(context.Background(), "monitor-arm")
				if err != nil {
					t.Fatalf("Level() error = %v", err)
				}
				if level.Available() != 5 {
					t.Fatalf("available = %d, want 5", level.Available())
				}
			},
		},
		{
			name: "an unknown sku cannot be reserved",
			run: func(t *testing.T, levels *store.Memory) {
				err := levels.Reserve(context.Background(), "ord_1", []core.Line{{SKU: "hammock", Quantity: 1}})
				if !errors.Is(err, core.ErrUnknownSKU) {
					t.Fatalf("Reserve() error = %v, want %v", err, core.ErrUnknownSKU)
				}
			},
		},
		{
			name: "releasing an unknown order changes nothing",
			run: func(t *testing.T, levels *store.Memory) {
				released, err := levels.Release(context.Background(), "ord_unknown")
				if err != nil {
					t.Fatalf("Release() error = %v", err)
				}
				if released {
					t.Fatal("Release() = true, want false")
				}
			},
		},
		{
			name: "restock adds to the shelf",
			run: func(t *testing.T, levels *store.Memory) {
				level, err := levels.Restock(context.Background(), "desk-lamp", 8)
				if err != nil {
					t.Fatalf("Restock() error = %v", err)
				}
				if level.OnHand != 50 {
					t.Fatalf("on hand = %d, want 50", level.OnHand)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, store.NewMemory(
				core.Level{SKU: "desk-lamp", OnHand: 42},
				core.Level{SKU: "monitor-arm", OnHand: 8},
			))
		})
	}
}

func TestLoadSnapshotKeepsReservations(t *testing.T) {
	path := writeSnapshot(t, snapshot)
	levels := store.NewMemory()

	if err := levels.LoadSnapshot(path); err != nil {
		t.Fatalf("LoadSnapshot() error = %v", err)
	}
	if err := levels.Reserve(context.Background(), "ord_1", []core.Line{{SKU: "desk-lamp", Quantity: 4}}); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := levels.LoadSnapshot(path); err != nil {
		t.Fatalf("second LoadSnapshot() error = %v", err)
	}

	level, err := levels.Level(context.Background(), "desk-lamp")
	if err != nil {
		t.Fatalf("Level() error = %v", err)
	}
	if level.Reserved != 4 {
		t.Fatalf("reserved = %d, want 4: a fresh export does not release orders", level.Reserved)
	}
}

func TestSnapshotCache(t *testing.T) {
	path := writeSnapshot(t, snapshot)
	cache := store.NewSnapshotCache()

	first, err := cache.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("loaded %d levels, want 2", len(first))
	}
	if _, err := cache.Load(path); err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if cache.Len() != 1 {
		t.Fatalf("cache holds %d entries, want 1", cache.Len())
	}

	cache.Invalidate(path)
	if cache.Len() != 0 {
		t.Fatalf("cache holds %d entries after Invalidate, want 0", cache.Len())
	}

	if _, err := cache.Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("Load() of a missing file = nil, want an error")
	}
}

func writeSnapshot(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stock.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	return path
}
