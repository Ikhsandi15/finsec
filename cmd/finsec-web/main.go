package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finsec-term/internal/config"
	"finsec-term/internal/fetcher"
	"finsec-term/internal/storage"
	"finsec-term/internal/ui"
	"finsec-term/internal/web"
)

const version = "1.0.0"

func findAvailableListener(startPort int) (net.Listener, int, error) {
	for p := startPort; p < startPort+50; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			return ln, p, nil
		}
	}
	return nil, 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+50)
}

func main() {
	var (
		reqPort     int
		configPath  string
		fixerKey    string
		forceFixer  bool
		autoOpen    bool
		showVersion bool
	)

	flag.IntVar(&reqPort, "port", 8080, "Port to serve the web dashboard on")
	flag.IntVar(&reqPort, "p", 8080, "Port to serve the web dashboard on (shorthand)")
	flag.StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
	flag.StringVar(&configPath, "c", "config.yaml", "Path to YAML configuration file (shorthand)")
	flag.StringVar(&fixerKey, "fixer-key", "", "Fixer.io API Key (Free tier 72h cached)")
	flag.BoolVar(&forceFixer, "force-fixer", false, "Force immediate live hit to Fixer.io, clearing local cache")
	flag.BoolVar(&autoOpen, "open", true, "Automatically open browser on startup")
	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.BoolVar(&showVersion, "v", false, "Print version information (shorthand)")
	flag.Parse()

	// Cloud-native support: Read PORT from environment variable (Render, Railway, Koyeb, Fly.io, etc.)
	if envPort := os.Getenv("PORT"); envPort != "" {
		var p int
		if _, err := fmt.Sscanf(envPort, "%d", &p); err == nil && p > 0 {
			reqPort = p
			autoOpen = false // Disable browser auto-open in headless cloud environments
		}
	}

	if showVersion {
		fmt.Printf("finsec-web v%s - Bloomberg-style Real-Time Threat & Macro Intelligence Web Dashboard\n", version)
		os.Exit(0)
	}

	// Find available listener (auto fallback if 8080 is in use)
	listener, actualPort, err := findAvailableListener(reqPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding to port: %v\n", err)
		os.Exit(1)
	}

	if actualPort != reqPort {
		fmt.Printf("Notice: Port %d is already in use. Switched to port %d automatically.\n", reqPort, actualPort)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config from %s (%v), using default settings.\n", configPath, err)
		cfg = config.DefaultConfig()
	}

	// Priority: Flag > ENV > config file
	if fixerKey != "" {
		cfg.Fixer.APIKey = fixerKey
	} else if envKey := os.Getenv("FIXER_API_KEY"); envKey != "" {
		cfg.Fixer.APIKey = envKey
	}

	// Initialize local cache
	cache, err := storage.NewCache(cfg.CacheFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize cache (%v), proceeding in-memory.\n", err)
	}

	// Initialize manager
	mgr := fetcher.NewManager(cfg, cache)

	// Initialize Web Server
	srv := web.NewServer(cfg, mgr, cache)

	if forceFixer && srv.FixerManager() != nil {
		srv.FixerManager().ResetCache()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background feed synchronizer
	srv.StartBackgroundSync(ctx)

	dashboardURL := fmt.Sprintf("http://localhost:%d", actualPort)

	httpServer := &http.Server{
		Handler:      srv.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	fixerStatus := "Not configured (fallback to Coinbase FX)"
	if cfg.Fixer.APIKey != "" {
		masked := cfg.Fixer.APIKey
		if len(masked) > 8 {
			masked = masked[:4] + "..." + masked[len(masked)-4:]
		}
		fixerStatus = fmt.Sprintf("Active (Key: %s, 72h cache)", masked)
	}

	// Print startup banner
	fmt.Println("=================================================================")
	fmt.Println("  FINSEC-TERM // REAL-TIME MACRO & THREAT INTELLIGENCE (WEB)    ")
	fmt.Println("=================================================================")
	fmt.Printf("  • Local Dashboard:  %s\n", dashboardURL)
	fmt.Printf("  • Fixer.io Status:  %s\n", fixerStatus)
	fmt.Printf("  • Refresh Interval: %v\n", cfg.RefreshInterval)
	fmt.Printf("  • Cache Location:   %s\n", cfg.CacheFile)
	fmt.Println("  • Press Ctrl+C to terminate server")
	fmt.Println("=================================================================")

	// Open browser if requested
	if autoOpen {
		go func() {
			time.Sleep(350 * time.Millisecond)
			_ = ui.OpenBrowser(dashboardURL)
		}()
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Println("\nShutting down finsec-web gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
	}

	if cache != nil {
		_ = cache.Save()
	}

	fmt.Println("Shutdown completed.")
}
