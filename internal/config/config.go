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
	StackKeywords   []string `yaml:"stack_keywords"`
	TechKeywords    []string `yaml:"tech_keywords"`
	IDMacroKeywords []string `yaml:"id_macro_keywords"`
	IDPolKeywords   []string `yaml:"id_pol_keywords"`
	GlobalKeywords  []string `yaml:"global_keywords"`
}

type FixerConfig struct {
	APIKey        string        `yaml:"api_key"`
	CacheDuration time.Duration `yaml:"cache_duration"`
}

type Config struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	MaxItemsPerFeed int           `yaml:"max_items_per_feed"`
	CacheFile       string        `yaml:"cache_file"`
	Fixer           FixerConfig   `yaml:"fixer"`
	Feeds           FeedsConfig   `yaml:"feeds"`
	Rules           RulesConfig   `yaml:"rules"`
}

func DefaultConfig() *Config {
	return &Config{
		RefreshInterval: 15 * time.Minute,
		RequestTimeout:  14 * time.Second,
		MaxItemsPerFeed: 60,
		CacheFile:       "~/.finsec-term/cache.json",
		Fixer: FixerConfig{
			APIKey:        "4b779965896935e4b8bd63e01918be6c",
			CacheDuration: 72 * time.Hour,
		},
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
				{
					Name: "The Verge Tech",
					URL:  "https://www.theverge.com/rss/index.xml",
					Type: FeedTypeRSS,
				},
				{
					Name: "Ars Technica",
					URL:  "https://feeds.arstechnica.com/arstechnica/index",
					Type: FeedTypeRSS,
				},
				{
					Name: "Hacker News",
					URL:  "https://news.ycombinator.com/rss",
					Type: FeedTypeRSS,
				},
				{
					Name: "TechCrunch",
					URL:  "https://techcrunch.com/feed/",
					Type: FeedTypeRSS,
				},
				{
					Name: "GitHub Blog",
					URL:  "https://github.blog/feed/",
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
			TechKeywords: []string{
				"AI", "Artificial Intelligence", "LLM", "OpenAI", "Anthropic", "Claude",
				"ChatGPT", "Gemini", "DeepSeek", "Nvidia", "GPU", "Machine Learning",
				"Semiconductor", "Apple", "Microsoft", "Google", "Meta", "Quantum",
				"Framework", "Rust", "Python", "Kubernetes", "Docker", "Open Source",
				"Architecture", "Robotics", "Hardware", "Linux", "TypeScript", "React",
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
			if filePath == "config.yaml" {
				// Cek jika ada typo file config..yaml atau config.example.yaml
				if typoData, typoErr := os.ReadFile("config..yaml"); typoErr == nil {
					_ = yaml.Unmarshal(typoData, cfg)
					return cfg, nil
				}
				if exData, exErr := os.ReadFile("config.example.yaml"); exErr == nil {
					_ = yaml.Unmarshal(exData, cfg)
					return cfg, nil
				}
			}
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Fallback jika API key kosong di config.yaml tapi ada di config..yaml
	if cfg.Fixer.APIKey == "" {
		if typoData, typoErr := os.ReadFile("config..yaml"); typoErr == nil {
			var typoCfg Config
			if err := yaml.Unmarshal(typoData, &typoCfg); err == nil && typoCfg.Fixer.APIKey != "" {
				cfg.Fixer.APIKey = typoCfg.Fixer.APIKey
			}
		}
	}

	return cfg, nil
}
