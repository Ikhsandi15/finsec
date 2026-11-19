package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.RefreshInterval != 15*time.Minute {
		t.Errorf("expected 15m refresh interval, got %v", cfg.RefreshInterval)
	}

	if len(cfg.Feeds.Pane1CVE) < 4 {
		t.Errorf("expected at least 4 Pane 1 feeds, got %d", len(cfg.Feeds.Pane1CVE))
	}
	if len(cfg.Feeds.Pane2GlobalMacro) < 4 {
		t.Errorf("expected at least 4 Pane 2 feeds, got %d", len(cfg.Feeds.Pane2GlobalMacro))
	}
	if len(cfg.Feeds.Pane3IDMacro) < 3 {
		t.Errorf("expected at least 3 Pane 3 feeds, got %d", len(cfg.Feeds.Pane3IDMacro))
	}

	if len(cfg.Rules.StackKeywords) == 0 {
		t.Errorf("expected stack keywords to be defined")
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	cfg, err := LoadConfig("non_existent_file.yaml")
	if err != nil {
		t.Fatalf("expected nil error for missing file fallback, got %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected default config returned")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "finsec-cfg-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	yamlContent := `
refresh_interval: 5m
request_timeout: 8s
max_items_per_feed: 20
cache_file: /tmp/test_cache.json
feeds:
  pane1_cve:
    - name: "Test Feed"
      url: "https://example.com/rss"
      type: "rss"
`
	filePath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(filePath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(filePath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.RefreshInterval != 5*time.Minute {
		t.Errorf("expected 5m, got %v", cfg.RefreshInterval)
	}
	if cfg.MaxItemsPerFeed != 20 {
		t.Errorf("expected 20 items, got %d", cfg.MaxItemsPerFeed)
	}
	if len(cfg.Feeds.Pane1CVE) != 1 || cfg.Feeds.Pane1CVE[0].Name != "Test Feed" {
		t.Errorf("unexpected feeds loaded: %+v", cfg.Feeds.Pane1CVE)
	}
}
