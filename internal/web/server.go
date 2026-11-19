package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"finsec-term/internal/config"
	"finsec-term/internal/fetcher"
	"finsec-term/internal/model"
	"finsec-term/internal/storage"
)

//go:embed static/*
var staticFS embed.FS

type SSEClient chan string

type Server struct {
	cfg        *config.Config
	manager    *fetcher.Manager
	cache      *storage.Cache
	mu         sync.RWMutex
	lastResult *fetcher.FetchResult
	isSyncing  bool
	clients    map[SSEClient]bool
	clientsMu  sync.Mutex
}

type StatsResponse struct {
	TotalItems  int       `json:"total_items"`
	UnreadCount int       `json:"unread_count"`
	CritCount   int       `json:"crit_count"`
	LastSync    time.Time `json:"last_sync"`
	IsSyncing   bool      `json:"is_syncing"`
}

type FeedsResponse struct {
	Pane1CVE         []model.FeedItem `json:"pane1_cve"`
	Pane2GlobalMacro []model.FeedItem `json:"pane2_global_macro"`
	Pane3IDMacro     []model.FeedItem `json:"pane3_id_macro"`
	Stats            StatsResponse    `json:"stats"`
	Errors           []string         `json:"errors"`
}

func NewServer(cfg *config.Config, manager *fetcher.Manager, cache *storage.Cache) *Server {
	return &Server{
		cfg:     cfg,
		manager: manager,
		cache:   cache,
		clients: make(map[SSEClient]bool),
	}
}

// StartBackgroundSync periodically fetches updates
func (s *Server) StartBackgroundSync(ctx context.Context) {
	// Initial fetch
	go s.SyncFeeds()

	interval := s.cfg.RefreshInterval
	if interval < time.Minute {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.SyncFeeds()
			}
		}
	}()
}

// SyncFeeds initiates concurrent feed retrieval and broadcasts results to SSE clients
func (s *Server) SyncFeeds() {
	s.mu.Lock()
	if s.isSyncing {
		s.mu.Unlock()
		return
	}
	s.isSyncing = true
	s.mu.Unlock()

	s.broadcastSSE("sync_started", map[string]interface{}{"is_syncing": true})

	res, err := s.manager.FetchAll(context.Background())

	s.mu.Lock()
	s.isSyncing = false
	if err == nil && res != nil {
		s.lastResult = res
	}
	s.mu.Unlock()

	s.broadcastSSE("sync_finished", s.buildFeedsResponse())
}

func (s *Server) buildFeedsResponse() FeedsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var (
		p1, p2, p3 []model.FeedItem
		errs       []string
		lastSync   time.Time
	)

	if s.lastResult != nil {
		p1 = s.lastResult.Pane1Items
		p2 = s.lastResult.Pane2Items
		p3 = s.lastResult.Pane3Items
		errs = s.lastResult.Errors
		lastSync = s.lastResult.FetchedAt
	}

	unread := 0
	crit := 0
	countItem := func(items []model.FeedItem) {
		for _, it := range items {
			if !it.IsRead {
				unread++
			}
			if it.Severity == model.SeverityCritical {
				crit++
			}
		}
	}
	countItem(p1)
	countItem(p2)
	countItem(p3)

	return FeedsResponse{
		Pane1CVE:         p1,
		Pane2GlobalMacro: p2,
		Pane3IDMacro:     p3,
		Stats: StatsResponse{
			TotalItems:  len(p1) + len(p2) + len(p3),
			UnreadCount: unread,
			CritCount:   crit,
			LastSync:    lastSync,
			IsSyncing:   s.isSyncing,
		},
		Errors: errs,
	}
}

func (s *Server) broadcastSSE(event string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(data))

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for client := range s.clients {
		select {
		case client <- msg:
		default:
		}
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/feeds", s.handleGetFeeds)
	mux.HandleFunc("/api/refresh", s.handleRefresh)
	mux.HandleFunc("/api/read", s.handleMarkRead)
	mux.HandleFunc("/api/events", s.handleSSE)
	mux.HandleFunc("/api/rates", s.handleMarketRates)

	// Static Assets with no-cache headers to prevent stale UI
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fileServer.ServeHTTP(w, r)
	})

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleGetFeeds(w http.ResponseWriter, r *http.Request) {
	resp := s.buildFeedsResponse()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go s.SyncFeeds()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "sync_triggered"})
}

type MarkReadPayload struct {
	ID   string `json:"id"`
	Pane int    `json:"pane"`
	All  bool   `json:"all"`
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MarkReadPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.lastResult != nil {
		if req.All {
			markSlice := func(items []model.FeedItem) {
				for i := range items {
					items[i].IsRead = true
					if s.cache != nil {
						s.cache.MarkRead(items[i].ID)
					}
				}
			}
			if req.Pane == 1 || req.Pane == 0 {
				markSlice(s.lastResult.Pane1Items)
			}
			if req.Pane == 2 || req.Pane == 0 {
				markSlice(s.lastResult.Pane2Items)
			}
			if req.Pane == 3 || req.Pane == 0 {
				markSlice(s.lastResult.Pane3Items)
			}
		} else if req.ID != "" {
			markItem := func(items []model.FeedItem) {
				for i := range items {
					if items[i].ID == req.ID {
						items[i].IsRead = true
					}
				}
			}
			markItem(s.lastResult.Pane1Items)
			markItem(s.lastResult.Pane2Items)
			markItem(s.lastResult.Pane3Items)
			if s.cache != nil {
				s.cache.MarkRead(req.ID)
			}
		}
	}
	if s.cache != nil {
		_ = s.cache.Save()
	}
	s.mu.Unlock()

	resp := s.buildFeedsResponse()
	s.broadcastSSE("feeds_updated", resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp.Stats)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(SSEClient, 10)
	s.clientsMu.Lock()
	s.clients[clientChan] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientChan)
		close(clientChan)
		s.clientsMu.Unlock()
	}()

	// Send current data immediately upon connect
	initialData, _ := json.Marshal(s.buildFeedsResponse())
	_, _ = fmt.Fprintf(w, "event: initial\ndata: %s\n\n", string(initialData))
	flusher.Flush()

	notify := r.Context().Done()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-notify:
			return
		case msg := <-clientChan:
			_, _ = fmt.Fprint(w, msg)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

type MarketRatesResponse struct {
	USDIDR    float64   `json:"usdidr"`
	BTCUSD    float64   `json:"btcusd"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	cachedRates     MarketRatesResponse
	cachedRatesTime time.Time
	ratesMu         sync.Mutex
)

func (s *Server) handleMarketRates(w http.ResponseWriter, r *http.Request) {
	ratesMu.Lock()
	defer ratesMu.Unlock()

	// Return cached if fetched in the last 30 seconds
	if time.Since(cachedRatesTime) < 30*time.Second && cachedRates.USDIDR > 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cachedRates)
		return
	}

	client := &http.Client{Timeout: 6 * time.Second}
	var (
		usdidr float64
		btcusd float64
		source = "Interbank Spot FX"
	)

	// 1. Fetch USD/IDR from Coinbase Official FX Rates (Same reliable provider as BTC!)
	reqCoinbaseFX, err := http.NewRequest(http.MethodGet, "https://api.coinbase.com/v2/exchange-rates?currency=USD", nil)
	if err == nil {
		reqCoinbaseFX.Header.Set("User-Agent", "Mozilla/5.0 finsec-term/1.0")
		resp, err := client.Do(reqCoinbaseFX)
		if err == nil && resp.StatusCode == http.StatusOK {
			var cbFXResp struct {
				Data struct {
					Rates map[string]string `json:"rates"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&cbFXResp); err == nil {
				if idrStr, ok := cbFXResp.Data.Rates["IDR"]; ok {
					var r float64
					_, _ = fmt.Sscanf(idrStr, "%f", &r)
					if r > 0 {
						usdidr = r
						source = "Coinbase FX Spot"
					}
				}
			}
			resp.Body.Close()
		}
	}

	// Fallback 1: Yahoo Finance Chart API (Interbank Spot Rate)
	if usdidr == 0 {
		req, err := http.NewRequest(http.MethodGet, "https://query1.finance.yahoo.com/v8/finance/chart/IDR=X", nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var yfResp struct {
					Chart struct {
						Result []struct {
							Meta struct {
								RegularMarketPrice float64 `json:"regularMarketPrice"`
							} `json:"meta"`
						} `json:"result"`
					} `json:"chart"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&yfResp); err == nil {
					if len(yfResp.Chart.Result) > 0 && yfResp.Chart.Result[0].Meta.RegularMarketPrice > 0 {
						usdidr = yfResp.Chart.Result[0].Meta.RegularMarketPrice
						source = "Yahoo Finance (Refinitiv Spot FX)"
					}
				}
				resp.Body.Close()
			}
		}
	}

	// Fallback 1 to Open Exchange Rates
	if usdidr == 0 {
		resp, err := client.Get("https://open.er-api.com/v6/latest/USD")
		if err == nil && resp.StatusCode == http.StatusOK {
			var erResp struct {
				Rates map[string]float64 `json:"rates"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&erResp); err == nil && erResp.Rates["IDR"] > 0 {
				usdidr = erResp.Rates["IDR"]
				source = "Open Exchange Rates API"
			}
			resp.Body.Close()
		}
	}

	// Fallback 2 to ExchangeRate-API
	if usdidr == 0 {
		resp, err := client.Get("https://api.exchangerate-api.com/v4/latest/USD")
		if err == nil && resp.StatusCode == http.StatusOK {
			var erResp struct {
				Rates map[string]float64 `json:"rates"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&erResp); err == nil && erResp.Rates["IDR"] > 0 {
				usdidr = erResp.Rates["IDR"]
				source = "ExchangeRate-API"
			}
			resp.Body.Close()
		}
	}

	// 2. Fetch BTC/USD from Coinbase (Official public API, NEVER blocked in Indonesia!)
	reqCoinbase, err := http.NewRequest(http.MethodGet, "https://api.coinbase.com/v2/prices/BTC-USD/spot", nil)
	if err == nil {
		reqCoinbase.Header.Set("User-Agent", "Mozilla/5.0 finsec-term/1.0")
		resp, err := client.Do(reqCoinbase)
		if err == nil && resp.StatusCode == http.StatusOK {
			var cbResp struct {
				Data struct {
					Amount string `json:"amount"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&cbResp); err == nil {
				var p float64
				_, _ = fmt.Sscanf(cbResp.Data.Amount, "%f", &p)
				if p > 0 {
					btcusd = p
				}
			}
			resp.Body.Close()
		}
	}

	// Fallback BTC to CryptoCompare (Official free API, no block in ID)
	if btcusd == 0 {
		resp, err := client.Get("https://min-api.cryptocompare.com/data/price?fsym=BTC&tsyms=USD")
		if err == nil && resp.StatusCode == http.StatusOK {
			var ccResp struct {
				USD float64 `json:"USD"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&ccResp); err == nil && ccResp.USD > 0 {
				btcusd = ccResp.USD
			}
			resp.Body.Close()
		}
	}

	// Fallback BTC to CoinGecko
	if btcusd == 0 {
		resp, err := client.Get("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd")
		if err == nil && resp.StatusCode == http.StatusOK {
			var cgResp struct {
				Bitcoin struct {
					USD float64 `json:"usd"`
				} `json:"bitcoin"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&cgResp); err == nil && cgResp.Bitcoin.USD > 0 {
				btcusd = cgResp.Bitcoin.USD
			}
			resp.Body.Close()
		}
	}

	if usdidr > 0 {
		cachedRates = MarketRatesResponse{
			USDIDR:    usdidr,
			BTCUSD:    btcusd,
			Source:    source,
			UpdatedAt: time.Now(),
		}
		cachedRatesTime = time.Now()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cachedRates)
}
