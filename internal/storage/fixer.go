package storage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FixerCacheData struct {
	Rate         float64   `json:"rate"`
	LastHitAt    time.Time `json:"last_hit_at"`
	LastHitDay   string    `json:"last_hit_day"`
	NextHitAfter time.Time `json:"next_hit_after"`
	Provider     string    `json:"provider"`
	ApiKeyUsed   string    `json:"api_key_used,omitempty"`
}

type FixerManager struct {
	mu            sync.RWMutex
	filePath      string
	apiKey        string
	cacheDuration time.Duration
	data          FixerCacheData
}

func NewFixerManager(apiKey string, cacheDuration time.Duration) *FixerManager {
	if cacheDuration <= 0 {
		cacheDuration = 72 * time.Hour // Default: 3 hari
	}

	home, err := os.UserHomeDir()
	var cachePath string
	if err == nil && home != "" {
		cachePath = filepath.Join(home, ".finsec-term", "fixer_cache.json")
	} else {
		cachePath = ".fixer_cache.json"
	}

	fm := &FixerManager{
		filePath:      cachePath,
		apiKey:        apiKey,
		cacheDuration: cacheDuration,
	}

	_ = fm.load()
	return fm
}

func (fm *FixerManager) load() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Cek primary path
	data, err := os.ReadFile(fm.filePath)
	if err != nil {
		// Cek fallback path di current directory
		if fm.filePath != ".fixer_cache.json" {
			data, err = os.ReadFile(".fixer_cache.json")
			if err != nil {
				return err
			}
			fm.filePath = ".fixer_cache.json"
		} else {
			return err
		}
	}

	return json.Unmarshal(data, &fm.data)
}

func (fm *FixerManager) save() error {
	dir := filepath.Dir(fm.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	bytes, err := json.MarshalIndent(fm.data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(fm.filePath, bytes, 0644)
	if err != nil && fm.filePath != ".fixer_cache.json" {
		// Fallback save to current directory if user home is restricted
		fm.filePath = ".fixer_cache.json"
		return os.WriteFile(fm.filePath, bytes, 0644)
	}
	return err
}

// ResetCache clears local cache to force a fresh hit on next call
func (fm *FixerManager) ResetCache() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.data = FixerCacheData{}
	_ = os.Remove(fm.filePath)
	_ = os.Remove(".fixer_cache.json")
	fmt.Println("[Fixer.io] Local cache reset. Next hit will fetch live data.")
}

// GetRate returns the USD/IDR rate. If cache is valid (< 3 days and same apiKey), returns cached.
// If cache is expired (> 3 days) or API key changed, hits Fixer.io and saves.
func (fm *FixerManager) GetRate() (float64, string, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	now := time.Now()

	// Valid cache check: rate > 0, before next hit, and API key matches (or was set)
	sameKey := (fm.apiKey == fm.data.ApiKeyUsed)
	hasValidCache := fm.data.Rate > 0 && now.Before(fm.data.NextHitAfter) && (sameKey || fm.apiKey == "")

	if hasValidCache {
		status := fmt.Sprintf("Fixer.io Cache (Hit: %s, Next: %s)",
			fm.data.LastHitDay, fm.data.NextHitAfter.Format("02 Jan 15:04"))
		return fm.data.Rate, status, nil
	}

	// If no API key configured
	if fm.apiKey == "" {
		if fm.data.Rate > 0 {
			return fm.data.Rate, "Fixer.io (API Key belum diisi, memakai cache terakhir)", nil
		}
		return 0, "", fmt.Errorf("Fixer API key belum dikonfigurasi di config.yaml")
	}

	// Cache expired or first run -> Hit Fixer.io
	fmt.Printf("[Fixer.io] Cache expired (>3 days) or new key. Calling Fixer.io API (Key: %s...)\n",
		fm.maskKey(fm.apiKey))

	// Fixer Free Tier standard endpoint is HTTP and base is EUR
	url := fmt.Sprintf("http://data.fixer.io/api/latest?access_key=%s&symbols=USD,IDR", fm.apiKey)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("[Fixer.io] Network request failed: %v\n", err)
		if fm.data.Rate > 0 {
			return fm.data.Rate, "Fixer.io (Hit gagal, memakai cache lokal)", nil
		}
		return 0, "", fmt.Errorf("fixer request error: %w", err)
	}
	defer resp.Body.Close()

	var fixerResp struct {
		Success bool               `json:"success"`
		Base    string             `json:"base"`
		Rates   map[string]float64 `json:"rates"`
		Error   struct {
			Code int    `json:"code"`
			Type string `json:"type"`
			Info string `json:"info"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fixerResp); err != nil {
		fmt.Printf("[Fixer.io] JSON parse error: %v\n", err)
		if fm.data.Rate > 0 {
			return fm.data.Rate, "Fixer.io (Decode error, memakai cache lokal)", nil
		}
		return 0, "", fmt.Errorf("fixer json error: %w", err)
	}

	if !fixerResp.Success {
		fmt.Printf("[Fixer.io] API returned error (code %d: %s - %s)\n",
			fixerResp.Error.Code, fixerResp.Error.Type, fixerResp.Error.Info)
		if fm.data.Rate > 0 {
			return fm.data.Rate, fmt.Sprintf("Fixer.io Error: %s (memakai cache)", fixerResp.Error.Type), nil
		}
		return 0, "", fmt.Errorf("fixer api error %d: %s (%s)",
			fixerResp.Error.Code, fixerResp.Error.Type, fixerResp.Error.Info)
	}

	var calculatedRate float64
	idr := fixerResp.Rates["IDR"]
	usd := fixerResp.Rates["USD"]

	if fixerResp.Base == "USD" {
		calculatedRate = idr
	} else if usd > 0 && idr > 0 {
		// Free tier Fixer base is EUR: USD/IDR = rate(IDR) / rate(USD)
		calculatedRate = idr / usd
	} else if idr > 0 {
		calculatedRate = idr
	}

	if calculatedRate <= 0 {
		if fm.data.Rate > 0 {
			return fm.data.Rate, "Fixer.io (Kalkulasi gagal, memakai cache)", nil
		}
		return 0, "", fmt.Errorf("gagal menghitung kurs dari Fixer response (IDR: %f, USD: %f)", idr, usd)
	}

	// Format Hari & Tanggal Indonesia
	locWIB := time.FixedZone("WIB", 7*3600)
	timeWIB := now.In(locWIB)
	dayNames := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	dayName := dayNames[timeWIB.Weekday()]
	formattedDate := fmt.Sprintf("%s, %s", dayName, timeWIB.Format("02 Jan 2006 15:04 WIB"))
	nextHit := now.Add(fm.cacheDuration)

	fm.data = FixerCacheData{
		Rate:         calculatedRate,
		LastHitAt:    now,
		LastHitDay:   formattedDate,
		NextHitAfter: nextHit,
		Provider:     "Fixer.io (Official Free Tier 72h Cache)",
		ApiKeyUsed:   fm.apiKey,
	}

	_ = fm.save()

	fmt.Printf("[Fixer.io] Live hit SUCCESS! USD/IDR = Rp %.2f (Base: %s, EUR/USD: %.4f, EUR/IDR: %.2f)\n",
		calculatedRate, fixerResp.Base, usd, idr)
	fmt.Printf("[Fixer.io] Cached locally for 72h. Next API call scheduled after: %s\n",
		nextHit.In(locWIB).Format("02 Jan 2006 15:04 WIB"))

	status := fmt.Sprintf("Fixer.io Live Hit (%s, Next: %s)",
		formattedDate, nextHit.In(locWIB).Format("02 Jan 15:04 WIB"))
	return calculatedRate, status, nil
}

func (fm *FixerManager) maskKey(k string) string {
	if len(k) <= 6 {
		return "***"
	}
	return k[:4] + "..." + k[len(k)-2:]
}
