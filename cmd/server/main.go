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
	"powergrid/internal/analytics"
	"powergrid/internal/database"
	"powergrid/internal/maps"
	"powergrid/internal/network"
	"powergrid/pkg/config"
	"powergrid/pkg/logger"
)

var (
	addr       = flag.String("addr", "", "http service address (overrides config)")
	configFile = flag.String("config", "config.yml", "path to config file")
	logLevel   = flag.String("log-level", "info", "log level: debug, info, warn, error")
	showCaller = flag.Bool("show-caller", false, "show caller information in logs")
	useDatabase = flag.Bool("use-db", true, "use SQLite database for analytics (default: true)")
	dataDir     = flag.String("data-dir", "./data", "directory for data files")
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

	// Parse log level
	var level logger.LogLevel
	switch *logLevel {
	case "debug":
		level = logger.DEBUG
	case "info":
		level = logger.INFO
	case "warn":
		level = logger.WARN
	case "error":
		level = logger.ERROR
	default:
		level = logger.INFO
	}

	// Initialize log broadcaster for streaming
	logBroadcaster := network.NewLogBroadcaster(1000) // Keep last 1000 log entries

	// Initialize colored loggers
	logger.InitLoggers(level, *showCaller)
	
	// Initialize streaming loggers
	logger.InitStreamingLoggers(logBroadcaster, level, *showCaller)

	// Use streaming server logger (but we need ColoredLogger for adapter)
	serverLogger := logger.ServerLogger

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		// If config file doesn't exist, use defaults
		serverLogger.Warn("Could not load config file %s: %v", *configFile, err)
		serverLogger.Info("Using default configuration")
		cfg = &config.Config{
			Server: config.ServerConfig{
				Host:        "0.0.0.0",
				Port:        5080,
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
		serverLogger.Info("Loaded configuration from %s", *configFile)
	}

	// Override address if provided via command line
	serverAddr := cfg.GetAddr()
	if *addr != "" {
		serverAddr = *addr
	}

	serverLogger.Info("Starting Power Grid Game Server on %s", serverAddr)
	serverLogger.Info("Environment: %s", cfg.Server.Environment)

	// Initialize map manager
	mapDir := filepath.Join(".", "maps")
	mapManager := maps.NewMapManager(mapDir)
	
	// Load all maps
	if err := mapManager.LoadMaps(); err != nil {
		serverLogger.Fatal("Failed to load maps: %v", err)
	}
	
	mapList := mapManager.GetMapList()
	serverLogger.Info("Loaded %d maps:", len(mapList))
	for _, mapInfo := range mapList {
		serverLogger.Info("  - %s (%s): %d cities, %d--%d players", 
			mapInfo.ID, mapInfo.Name, mapInfo.CityCount, 
			mapInfo.PlayerCount.Min, mapInfo.PlayerCount.Max)
	}

	// Create a new lobby handler with map manager
	lobbyHandler := handlers.NewLobbyHandler()
	lobbyHandler.SetMapManager(mapManager)

	// Pass the logger to the handlers (using adapter for interface compatibility)
	lobbyHandler.SetLogger(logger.AsHandlersLogger(serverLogger))

	// Start session cleanup routine
	// Clean up sessions inactive for 30 minutes, check every 5 minutes
	lobbyHandler.StartSessionCleanup(5*time.Minute, 30*time.Minute)
	serverLogger.Info("Started session cleanup routine")

	// Initialize analytics service
	if *useDatabase {
		// Initialize SQLite database
		dbConfig := database.DefaultConfig(*dataDir)
		db, err := database.NewConnection(dbConfig)
		if err != nil {
			serverLogger.Fatal("Failed to initialize database: %v", err)
		}
		defer db.Close()
		
		serverLogger.Info("Connected to SQLite database: %s", dbConfig.Path)
		
		// Initialize database analytics service
		dbAnalyticsService := analytics.NewDatabaseService(db)
		
		// Initialize predefined achievements
		if err := dbAnalyticsService.InitializeAchievements(); err != nil {
			serverLogger.Warn("Failed to initialize achievements: %v", err)
		}
		
		// Create database analytics handler
		dbAnalyticsHandler := handlers.NewDatabaseAnalyticsHandler(dbAnalyticsService)
		serverLogger.Info("Initialized database analytics service")
		
		// Register database analytics routes
		dbAnalyticsHandler.RegisterRoutes(http.DefaultServeMux)
	} else {
		// Use file-based analytics
		analyticsDir := filepath.Join(".", "data", "analytics")
		analyticsService := analytics.NewService(analyticsDir)
		serverLogger.Info("Initialized file-based analytics service")
		
		// Create analytics handler
		analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
		// Register file-based analytics routes
		analyticsHandler.RegisterRoutes(http.DefaultServeMux)
	}
	
	// Create admin handler
	adminHandler := handlers.NewAdminHandler(logBroadcaster)
	
	// Create simulated games manager
	simulatedGamesManager := handlers.NewSimulatedGamesManager(lobbyHandler.GetLobbyManager())
	serverLogger.Info("Initialized simulated games manager")
	
	// Setup analytics hook for game manager (only for file-based analytics)
	if !*useDatabase {
		analyticsDir := filepath.Join(".", "data", "analytics")
		analyticsService := analytics.NewService(analyticsDir)
		analyticsHook := network.NewAnalyticsHook(analyticsService)
		network.Games.SetAnalyticsHook(analyticsHook)
		serverLogger.Info("Integrated analytics with game manager")
	}

	// Register handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/maps", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleMapsAPI(w, r, mapManager)
	})
	http.HandleFunc("/ws", lobbyHandler.HandleWebSocket)
	http.HandleFunc("/game", handlers.HandleGameWebSocket)
	
	// Register admin API routes
	adminHandler.RegisterRoutes(http.DefaultServeMux)
	
	// Register simulated games routes
	simulatedGamesManager.RegisterRoutes(http.DefaultServeMux)

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
		serverLogger.Info("Server listening on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverLogger.Fatal("Server failed to start: %v", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for shutdown signal or admin shutdown request
	select {
	case sig := <-quit:
		serverLogger.Info("Received shutdown signal: %v", sig)
	case <-adminHandler.GetShutdownChannel():
		serverLogger.Info("Received admin shutdown request")
	}

	// Create shutdown context with timeout
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Stop accepting new connections and wait for existing ones to finish
	serverLogger.Info("Shutting down server...")
	
	// Stop session cleanup routine
	lobbyHandler.StopSessionCleanup()
	
	// Stop any running AI clients
	simulatedGamesManager.Shutdown()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		serverLogger.Warn("Server forced to shutdown: %v", err)
	}

	serverLogger.Info("Server gracefully stopped")
}
