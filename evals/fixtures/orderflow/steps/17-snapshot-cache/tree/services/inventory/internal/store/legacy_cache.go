package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"orderflow/services/inventory/internal/core"
)

// SnapshotCache keeps decoded warehouse exports in memory.
//
// The export is a JSON array of stock levels that the warehouse rewrites a few
// times a day. Decoding it is cheap but not free, and several stores in the
// same process point at the same file, so the decoded copy is kept until the
// modification time of the file changes.
type SnapshotCache struct {
	mu      sync.Mutex
	entries map[string]snapshotEntry
}

// snapshotEntry is a decoded snapshot together with the modification time it
// was read at.
type snapshotEntry struct {
	modified time.Time
	levels   []core.Level
}

// NewSnapshotCache creates an empty cache.
func NewSnapshotCache() *SnapshotCache {
	return &SnapshotCache{entries: make(map[string]snapshotEntry)}
}

// Load returns the snapshot stored at path. The file is read from disk when it
// has not been seen before or when it changed since the last read.
func (c *SnapshotCache) Load(path string) ([]core.Level, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat snapshot %s: %w", path, err)
	}

	entry, cached := c.entries[path]
	if cached && entry.modified.Equal(info.ModTime()) {
		return entry.levels, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}

	var levels []core.Level
	if err := json.Unmarshal(raw, &levels); err != nil {
		return nil, fmt.Errorf("decode snapshot %s: %w", path, err)
	}

	c.entries[path] = snapshotEntry{modified: info.ModTime(), levels: levels}
	return levels, nil
}

// Invalidate forgets the snapshot cached for path.
func (c *SnapshotCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, path)
}

// Len returns how many snapshots the cache holds.
func (c *SnapshotCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.entries)
}
