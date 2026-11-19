package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "finsec-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "test_cache.json")
	cache, err := NewCache(cacheFile)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	itemID := "abc12345"

	// Record seen
	cache.RecordSeen(itemID)
	if cache.IsRead(itemID) {
		t.Errorf("new item should not be read initially")
	}

	// Mark read
	cache.MarkRead(itemID)
	if !cache.IsRead(itemID) {
		t.Errorf("item should be marked as read")
	}

	// Save to disk
	if err := cache.Save(); err != nil {
		t.Fatalf("cache.Save failed: %v", err)
	}

	// Reload from disk
	reloadedCache, err := NewCache(cacheFile)
	if err != nil {
		t.Fatalf("NewCache reload failed: %v", err)
	}
	if !reloadedCache.IsRead(itemID) {
		t.Errorf("reloaded cache should preserve read state")
	}
}
