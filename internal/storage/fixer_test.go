package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFixerManager_CacheExpiry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fixer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "fixer_cache.json")

	fm := &FixerManager{
		filePath:      cacheFile,
		apiKey:        "dummy_key",
		cacheDuration: 72 * time.Hour,
		data: FixerCacheData{
			Rate:         18265.0,
			LastHitAt:    time.Now().Add(-24 * time.Hour), // 1 day ago
			LastHitDay:   "Rabu, 18 Nov 2026",
			NextHitAfter: time.Now().Add(48 * time.Hour), // still valid for 2 days
			Provider:     "Fixer.io",
			ApiKeyUsed:   "dummy_key",
		},
	}

	// Should return cached rate without error because still valid (< 72h)
	rate, status, err := fm.GetRate()
	if err != nil {
		t.Fatalf("expected cached rate, got error: %v", err)
	}

	if rate != 18265.0 {
		t.Errorf("expected 18265.0, got %f", rate)
	}

	if status == "" {
		t.Errorf("expected non-empty status description")
	}
}
