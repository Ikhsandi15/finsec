package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type FeedType string

const (
	FeedTypeRSS  FeedType = "rss"
	FeedTypeCISA FeedType = "cisa"
)

type FeedSource struct {
	Name string   `yaml:"name"`
	URL  string   `yaml:"url"`
	Type FeedType `yaml:"type"` // "rss" or "cisa"
}

type FeedsConfig struct {
	Pane1CVE         []FeedSource `yaml:"pane1_cve"`
	Pane2GlobalMacro []FeedSource `yaml:"pane2_global_macro"`
	Pane3IDMacro     []FeedSource `yaml:"pane3_id_macro"`
}

type RulesConfig struct {
	StackKeywords    []string `yaml:"stack_keywords"`
	IDMacroKeywords  []string `yaml:"id_macro_keywords"`
	IDPolKeywords    []string `yaml:"id_pol_keywords"`
	GlobalKeywords   []string `yaml:"global_keywords"`
}

type Config struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	MaxItemsPerFeed int           `yaml:"max_items_per_feed"`
	CacheFile       string        `yaml:"cache_file"`
	Feeds           FeedsConfig   `yaml:"feeds"`
	Rules           RulesConfig   `yaml:"rules"`
}

func DefaultConfig() *Config {
	return &Config{
		RefreshInterval: 15 * time.Minute,
		RequestTimeout:  14 * time.Second,
		MaxItemsPerFeed: 60,
		CacheFile:       "~/.finsec-term/cache.json",
		Feeds: FeedsConfig{
			Pane1CVE: []FeedSource{
				{
					Name: "CISA KEV",
					URL:  "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json",
					Type: FeedTypeCISA,
				},
				{
					Name: "BleepingComputer",
					URL:  "https://www.bleepingcomputer.com/feed/",
					Type: FeedTypeRSS,
				},
				{
					Name: "The Hacker News",
					URL:  "https://feeds.feedburner.com/TheHackersNews",
					Type: FeedTypeRSS,
				},
				{
					Name: "SecurityWeek",
					URL:  "https://feeds.feedburner.com/securityweek",
					Type: FeedTypeRSS,
				},
				{
					Name: "Google Cloud Sec",
					URL:  "https://cloudblog.withgoogle.com/products/identity-security/rss/",
					Type: FeedTypeRSS,
				},
			},
			Pane2GlobalMacro: []FeedSource{
				{
					Name: "CNBC Markets",
					URL:  "https://search.cnbc.com/rs/search/view.html?partnerId=2000&keywords=markets&sort=date&type=all&output=rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "CNBC Economy",
					URL:  "https://search.cnbc.com/rs/search/view.html?partnerId=2000&keywords=economy&sort=date&type=all&output=rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "Investing.com",
					URL:  "https://www.investing.com/rss/news.rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "FXStreet",
					URL:  "https://www.fxstreet.com/rss/news",
					Type: FeedTypeRSS,
				},
				{
					Name: "CoinDesk",
					URL:  "https://www.coindesk.com/arc/outboundfeeds/rss/",
					Type: FeedTypeRSS,
				},
			},
			Pane3IDMacro: []FeedSource{
				{
					Name: "CNBC ID Market",
					URL:  "https://www.cnbcindonesia.com/market/rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "CNBC ID News",
					URL:  "https://www.cnbcindonesia.com/news/rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "Antara Ekonomi",
					URL:  "https://www.antaranews.com/rss/ekonomi.xml",
					Type: FeedTypeRSS,
				},
				{
					Name: "Bisnis.com Market",
					URL:  "https://rss.bisnis.com/market",
					Type: FeedTypeRSS,
				},
				{
					Name: "Kontan Investasi",
					URL:  "https://rss.kontan.co.id/news/investasi",
					Type: FeedTypeRSS,
				},
			},
		},
		Rules: RulesConfig{
			StackKeywords: []string{
				"PHP", "Node.js", "NodeJS", "Nginx", "Golang", "Go ", "PostgreSQL",
				"Postgres", "Redis", "RabbitMQ", "Google Cloud", "GCP", "Brotli",
				"gRPC", "RCE", "Remote Code Execution", "Zero-Day", "0-day", "CVSS",
				"Privilege Escalation", "SQL Injection", "Authentication Bypass",
			},
			IDMacroKeywords: []string{
				"Rupiah", "IDR", "Bank Indonesia", "BI Rate", "IHSG", "SBN", "OJK",
				"APBN", "MSCI", "Cadangan Devisa", "Inflasi", "SUN", "Surat Berharga Negara",
				"Bursa Efek", "Fintech", "Perbankan",
			},
			IDPolKeywords: []string{
				"Prabowo", "Purbaya", "Kemenkeu", "Koalisi", "Regulasi", "Danantara",
				"Kebijakan Fiskal", "Menteri Keuangan", "DPR", "Kabinet",
			},
			GlobalKeywords: []string{
				"The Fed", "FOMC", "Jerome Powell", "Powell", "PCE", "CPI", "Treasury",
				"Geopolitik", "Geopolitics", "Oil", "Crude", "Bitcoin", "Crypto",
				"Interest Rate", "Inflation", "Yield", "Tariff", "War",
			},
		},
	}
}

// ExpandHome resolves ~ to the user's home directory
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

// LoadConfig loads the YAML configuration from filePath or returns default if not found
func LoadConfig(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	if filePath == "" {
		return cfg, nil
	}

	expandedPath := ExpandHome(filePath)
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
