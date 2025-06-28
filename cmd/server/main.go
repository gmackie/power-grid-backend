package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"powergrid/handlers"
)

var addr = flag.String("addr", ":4080", "http service address")

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

	logger.Printf("Starting Power Grid Game Server on %s", *addr)

	// Create a new lobby handler
	lobbyHandler := handlers.NewLobbyHandler()

	// Pass the logger to the handlers
	lobbyHandler.SetLogger(logger)

	// Register handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ws", lobbyHandler.HandleWebSocket)

	// Start the server
	logger.Fatal(http.ListenAndServe(*addr, nil))
}
