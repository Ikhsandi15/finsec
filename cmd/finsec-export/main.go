package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"finsec-term/internal/config"
	"finsec-term/internal/fetcher"
	"finsec-term/internal/model"
	"finsec-term/internal/storage"
	"finsec-term/internal/web"
)

func main() {
	var (
		configPath string
		outDir     string
	)

	flag.StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
	flag.StringVar(&configPath, "c", "config.yaml", "Path to YAML configuration file (shorthand)")
	flag.StringVar(&outDir, "out", "dist", "Output directory for static snapshot")
	flag.StringVar(&outDir, "o", "dist", "Output directory for static snapshot (shorthand)")
	flag.Parse()

	log.Printf("[EXPORT] Starting Finsec static snapshot generation...")

	// 1. Load Config (fallback to config.example.yaml if config.yaml not found)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("[EXPORT] %s not found, falling back to config.example.yaml: %v", configPath, err)
		cfg, err = config.LoadConfig("config.example.yaml")
		if err != nil {
			log.Fatalf("[EXPORT] Fatal: cannot load config: %v", err)
		}
	}

	// 2. Initialize Cache & Fetcher Manager
	cache, err := storage.NewCache(cfg.CacheFile)
	if err != nil {
		log.Printf("[EXPORT] Cache init warning: %v", err)
	}
	manager := fetcher.NewManager(cfg, cache)

	// 3. Fetch All Feeds Concurrently
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	log.Println("[EXPORT] Fetching all feeds (CISA, RSS, Direct X SSR, Telegram)...")
	fetchRes, err := manager.FetchAll(ctx)
	if err != nil {
		log.Fatalf("[EXPORT] Fatal error fetching feeds: %v", err)
	}

	// Count statistics
	unreadCount := 0
	critCount := 0
	countItem := func(items []model.FeedItem) {
		for _, it := range items {
			if !it.IsRead {
				unreadCount++
			}
			if it.Severity == model.SeverityCritical {
				critCount++
			}
		}
	}
	countItem(fetchRes.Pane1Items)
	countItem(fetchRes.Pane2Items)
	countItem(fetchRes.Pane3Items)

	feedsResponse := web.FeedsResponse{
		Pane1CVE:         fetchRes.Pane1Items,
		Pane2GlobalMacro: fetchRes.Pane2Items,
		Pane3IDMacro:     fetchRes.Pane3Items,
		Stats: web.StatsResponse{
			TotalItems:  len(fetchRes.Pane1Items) + len(fetchRes.Pane2Items) + len(fetchRes.Pane3Items),
			UnreadCount: unreadCount,
			CritCount:   critCount,
			LastSync:    fetchRes.FetchedAt,
			IsSyncing:   false,
		},
		Errors: fetchRes.Errors,
	}

	// 4. Fetch Market Rates (USD/IDR & BTC/USD)
	rates := fetchMarketRates(cfg)

	// 5. Create Output Directories
	dataDir := filepath.Join(outDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("[EXPORT] Failed to create data dir: %v", err)
	}

	// 6. Write feeds.json & rates.json
	feedsJSON, err := json.MarshalIndent(feedsResponse, "", "  ")
	if err != nil {
		log.Fatalf("[EXPORT] Failed to marshal feeds: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "feeds.json"), feedsJSON, 0644); err != nil {
		log.Fatalf("[EXPORT] Failed to write feeds.json: %v", err)
	}

	ratesJSON, err := json.MarshalIndent(rates, "", "  ")
	if err != nil {
		log.Fatalf("[EXPORT] Failed to marshal rates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "rates.json"), ratesJSON, 0644); err != nil {
		log.Fatalf("[EXPORT] Failed to write rates.json: %v", err)
	}

	// 7. Copy static assets (index.html, style.css, app.js)
	staticFiles := []string{"index.html", "style.css", "app.js"}
	for _, fn := range staticFiles {
		srcPath := filepath.Join("internal", "web", "static", fn)
		dstPath := filepath.Join(outDir, fn)
		if err := copyFile(srcPath, dstPath); err != nil {
			log.Fatalf("[EXPORT] Failed to copy %s: %v", fn, err)
		}
	}

	// 8. Create .nojekyll for GitHub Pages
	_ = os.WriteFile(filepath.Join(outDir, ".nojekyll"), []byte(""), 0644)

	log.Printf("[EXPORT] SUCCESS! Generated static bundle in '%s':", outDir)
	log.Printf("  - Feeds: %d items across 3 panes", feedsResponse.Stats.TotalItems)
	log.Printf("  - Rates: USD/IDR %.2f (%s), BTC/USD %.2f", rates.USDIDR, rates.Source, rates.BTCUSD)
	log.Println("  - Ready for GitHub Pages deployment!")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func fetchMarketRates(cfg *config.Config) web.MarketRatesResponse {
	var usdidr float64
	var source string

	// Fixer API if configured
	if cfg.Fixer.APIKey != "" {
		fixerMgr := storage.NewFixerManager(cfg.Fixer.APIKey, cfg.Fixer.CacheDuration)
		rate, status, err := fixerMgr.GetRate()
		if err == nil && rate > 0 {
			usdidr = rate
			source = status
		}
	}

	// Fallback to Coinbase FX
	if usdidr <= 0 {
		client := &http.Client{Timeout: 6 * time.Second}
		req, err := http.NewRequest(http.MethodGet, "https://api.coinbase.com/v2/exchange-rates?currency=USD", nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var cbResp struct {
					Data struct {
						Rates map[string]string `json:"rates"`
					} `json:"data"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&cbResp); err == nil {
					if idrStr, ok := cbResp.Data.Rates["IDR"]; ok {
						if parsed, err := strconv.ParseFloat(idrStr, 64); err == nil && parsed > 0 {
							usdidr = parsed
							source = "Coinbase Institutional FX"
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	// Fallback hardcoded if offline
	if usdidr <= 0 {
		usdidr = 16250.0
		source = "Interbank Baseline (Estimated)"
	}

	// Fetch BTC/USD from Binance
	var btcusd float64
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT")
	if err == nil && resp.StatusCode == http.StatusOK {
		var binanceResp struct {
			Price string `json:"price"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&binanceResp); err == nil {
			if parsed, err := strconv.ParseFloat(binanceResp.Price, 64); err == nil && parsed > 0 {
				btcusd = parsed
			}
		}
		resp.Body.Close()
	}
	if btcusd <= 0 {
		btcusd = 63200.0
	}

	return web.MarketRatesResponse{
		USDIDR:    usdidr,
		BTCUSD:    btcusd,
		Source:    source,
		UpdatedAt: time.Now(),
	}
}
