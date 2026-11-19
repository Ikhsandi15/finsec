package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"finsec-term/internal/config"
)

type CacheItem struct {
	ID        string    `json:"id"`
	FirstSeen time.Time `json:"first_seen"`
	IsRead    bool      `json:"is_read"`
	ReadAt    time.Time `json:"read_at,omitempty"`
}

type CacheData struct {
	Version   int                  `json:"version"`
	UpdatedAt time.Time            `json:"updated_at"`
	Items     map[string]CacheItem `json:"items"`
}

type Cache struct {
	mu       sync.RWMutex
	filePath string
	data     CacheData
	dirty    bool
}

// NewCache initializes or loads the cache from disk
func NewCache(filePath string) (*Cache, error) {
	resolved := config.ExpandHome(filePath)
	c := &Cache{
		filePath: resolved,
		data: CacheData{
			Version:   1,
			UpdatedAt: time.Now(),
			Items:     make(map[string]CacheItem),
		},
	}

	if err := c.load(); err != nil {
		// If file doesn't exist, ignore and use empty cache
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return c, nil
}

func (c *Cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return err
	}

	if cacheData.Items == nil {
		cacheData.Items = make(map[string]CacheItem)
	}

	c.data = cacheData
	return nil
}

// Save writes cache data to disk atomically
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil
	}

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	c.data.UpdatedAt = time.Now()

	// Prune old entries if more than 8000 items
	if len(c.data.Items) > 8000 {
		cutoff := time.Now().Add(-60 * 24 * time.Hour)
		for id, item := range c.data.Items {
			if item.FirstSeen.Before(cutoff) {
				delete(c.data.Items, id)
			}
		}
	}

	bytes, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := c.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, c.filePath); err != nil {
		// Fallback for Windows if rename fails due to existing destination
		_ = os.Remove(c.filePath)
		if err := os.Rename(tmpFile, c.filePath); err != nil {
			return err
		}
	}

	c.dirty = false
	return nil
}

// RecordSeen records an item if not present
func (c *Cache) RecordSeen(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data.Items[id]; !exists {
		c.data.Items[id] = CacheItem{
			ID:        id,
			FirstSeen: time.Now(),
			IsRead:    false,
		}
		c.dirty = true
	}
}

// IsRead returns true if item is marked as read
func (c *Cache) IsRead(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.data.Items[id]
	return exists && item.IsRead
}

// MarkRead marks an item as read
func (c *Cache) MarkRead(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.data.Items[id]
	if !exists {
		item = CacheItem{
			ID:        id,
			FirstSeen: time.Now(),
		}
	}
	item.IsRead = true
	item.ReadAt = time.Now()
	c.data.Items[id] = item
	c.dirty = true
}

// MarkMultipleRead marks multiple IDs as read
func (c *Cache) MarkMultipleRead(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for _, id := range ids {
		item, exists := c.data.Items[id]
		if !exists {
			item = CacheItem{
				ID:        id,
				FirstSeen: now,
			}
		}
		item.IsRead = true
		item.ReadAt = now
		c.data.Items[id] = item
	}
	c.dirty = true
}
