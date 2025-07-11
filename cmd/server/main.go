package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"powergrid/handlers"
	"powergrid/internal/maps"
	"powergrid/pkg/config"
)

var (
	addr       = flag.String("addr", "", "http service address (overrides config)")
	configFile = flag.String("config", "config.yml", "path to config file")
)

// Custom logger that adds [backend] prefix
type backendLogger struct {
	logger *log.Logger
}

func newBackendLogger() *backendLogger {
	return &backendLogger{
		logger: log.New(os.Stdout, "[backend] ", log.LstdFlags),
	}
}

func (l *backendLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

func (l *backendLogger) Println(v ...interface{}) {
	l.logger.Println(v...)
}

func (l *backendLogger) Fatal(v ...interface{}) {
	l.logger.Fatal(v...)
}

func (l *backendLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
}

// Simple handler for home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"name": "Power Grid Game Server", "version": "0.1.0", "status": "running"}`)
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy"}`)
}

func main() {
	flag.Parse()

	// Create custom logger with [backend] prefix
	logger := newBackendLogger()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		// If config file doesn't exist, use defaults
		logger.Printf("Warning: Could not load config file %s: %v", *configFile, err)
		logger.Println("Using default configuration")
		cfg = &config.Config{
			Server: config.ServerConfig{
				Host:        "0.0.0.0",
				Port:        4080,
				Environment: "development",
			},
			WebSocket: config.WebSocketConfig{
				MaxConnections: 500,
				ReadTimeout:    30 * time.Second,
				WriteTimeout:   10 * time.Second,
				PingInterval:   25 * time.Second,
				MaxMessageSize: 8192,
			},
			Game: config.GameConfig{
				MaxPlayersPerGame:  6,
				MinPlayersPerGame:  2,
				GameTimeout:        45 * time.Minute,
				TurnTimeout:        5 * time.Minute,
				LobbyTimeout:       10 * time.Minute,
				MaxConcurrentGames: 100,
			},
		}
	} else {
		logger.Printf("Loaded configuration from %s", *configFile)
	}

	// Override address if provided via command line
	serverAddr := cfg.GetAddr()
	if *addr != "" {
		serverAddr = *addr
	}

	logger.Printf("Starting Power Grid Game Server on %s", serverAddr)
	logger.Printf("Environment: %s", cfg.Server.Environment)

	// Initialize map manager
	mapDir := filepath.Join(".", "maps")
	mapManager := maps.NewMapManager(mapDir)
	
	// Load all maps
	if err := mapManager.LoadMaps(); err != nil {
		logger.Fatalf("Failed to load maps: %v", err)
	}
	
	mapList := mapManager.GetMapList()
	logger.Printf("Loaded %d maps: ", len(mapList))
	for _, mapInfo := range mapList {
		logger.Printf("  - %s (%s): %d cities, %d--%d players", 
			mapInfo.ID, mapInfo.Name, mapInfo.CityCount, 
			mapInfo.PlayerCount.Min, mapInfo.PlayerCount.Max)
	}

	// Create a new lobby handler with map manager
	lobbyHandler := handlers.NewLobbyHandler()
	lobbyHandler.SetMapManager(mapManager)

	// Pass the logger to the handlers
	lobbyHandler.SetLogger(logger)

	// Start session cleanup routine
	// Clean up sessions inactive for 30 minutes, check every 5 minutes
	lobbyHandler.StartSessionCleanup(5*time.Minute, 30*time.Minute)
	logger.Println("Started session cleanup routine")

	// Register handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/maps", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleMapsAPI(w, r, mapManager)
	})
	http.HandleFunc("/ws", lobbyHandler.HandleWebSocket)
	http.HandleFunc("/game", handlers.HandleGameWebSocket)

	// Create HTTP server
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      nil, // Use DefaultServeMux
		ReadTimeout:  cfg.WebSocket.ReadTimeout,
		WriteTimeout: cfg.WebSocket.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Server listening on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for shutdown signal
	sig := <-quit
	logger.Printf("Received shutdown signal: %v", sig)

	// Create shutdown context with timeout
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Stop accepting new connections and wait for existing ones to finish
	logger.Println("Shutting down server...")
	
	// Stop session cleanup routine
	lobbyHandler.StopSessionCleanup()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server gracefully stopped")
}
