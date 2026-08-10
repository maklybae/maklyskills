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
			name: "a written level is read back",
			run: func(t *testing.T, levels *store.Memory) {
				err := levels.SetLevel(context.Background(), core.Level{SKU: "monitor-arm", OnHand: 8, Reserved: 3})
				if err != nil {
					t.Fatalf("SetLevel() error = %v", err)
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
			name: "an unknown sku is not on the shelf",
			run: func(t *testing.T, levels *store.Memory) {
				_, err := levels.Level(context.Background(), "hammock")
				if !errors.Is(err, core.ErrUnknownSKU) {
					t.Fatalf("Level() error = %v, want %v", err, core.ErrUnknownSKU)
				}
			},
		},
		{
			name: "a reservation is stored and read back",
			run: func(t *testing.T, levels *store.Memory) {
				lines := []core.Line{{SKU: "desk-lamp", Quantity: 2}}
				if err := levels.SaveReservation(context.Background(), "ord_1", lines); err != nil {
					t.Fatalf("SaveReservation() error = %v", err)
				}
				stored, held, err := levels.Reservation(context.Background(), "ord_1")
				if err != nil {
					t.Fatalf("Reservation() error = %v", err)
				}
				if !held || len(stored) != 1 || stored[0].Quantity != 2 {
					t.Fatalf("Reservation() = %+v, %v, want the stored line", stored, held)
				}
			},
		},
		{
			name: "a dropped reservation is gone",
			run: func(t *testing.T, levels *store.Memory) {
				lines := []core.Line{{SKU: "desk-lamp", Quantity: 2}}
				if err := levels.SaveReservation(context.Background(), "ord_1", lines); err != nil {
					t.Fatalf("SaveReservation() error = %v", err)
				}
				if err := levels.DeleteReservation(context.Background(), "ord_1"); err != nil {
					t.Fatalf("DeleteReservation() error = %v", err)
				}
				_, held, err := levels.Reservation(context.Background(), "ord_1")
				if err != nil {
					t.Fatalf("Reservation() error = %v", err)
				}
				if held {
					t.Fatal("Reservation() = held, want gone")
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

func TestLoadSnapshot(t *testing.T) {
	path := writeSnapshot(t, snapshot)
	levels := store.NewMemory()

	if err := levels.LoadSnapshot(path); err != nil {
		t.Fatalf("LoadSnapshot() error = %v", err)
	}

	level, err := levels.Level(context.Background(), "desk-lamp")
	if err != nil {
		t.Fatalf("Level() error = %v", err)
	}
	if level.OnHand != 42 {
		t.Fatalf("on hand = %d, want 42", level.OnHand)
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
