package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"finsec-term/internal/config"
	"finsec-term/internal/fetcher"
	"finsec-term/internal/storage"
	"finsec-term/internal/ui"
)

const version = "1.0.0"

func main() {
	var (
		configPath  string
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
	flag.StringVar(&configPath, "c", "config.yaml", "Path to YAML configuration file (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.BoolVar(&showVersion, "v", false, "Print version information (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Printf("finsec-term v%s - Bloomberg-style Real-Time Threat & Macro Intelligence Terminal\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config from %s (%v), using default settings.\n", configPath, err)
		cfg = config.DefaultConfig()
	}

	// Initialize local cache
	cache, err := storage.NewCache(cfg.CacheFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize cache (%v), proceeding in-memory.\n", err)
	}

	// Initialize manager
	mgr := fetcher.NewManager(cfg, cache)

	// Create Bubble Tea app model
	appModel := ui.NewAppModel(cfg, mgr, cache)

	// Run full-screen Bubble Tea program with AltScreen
	p := tea.NewProgram(
		appModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running finsec-term: %v\n", err)
		os.Exit(1)
	}
}
