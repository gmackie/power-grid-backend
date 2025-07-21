package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"powergrid/internal/analytics"
	"powergrid/internal/database"
	"powergrid/pkg/logger"
)

// DatabaseHandler handles database management and monitoring endpoints
type DatabaseHandler struct {
	service *analytics.DatabaseService
	db      *database.DB
	logger  *logger.ColoredLogger
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(service *analytics.DatabaseService, db *database.DB) *DatabaseHandler {
	return &DatabaseHandler{
		service: service,
		db:      db,
		logger:  logger.CreateAILogger("DatabaseAPI", logger.ColorBrightRed),
	}
}

// RegisterRoutes registers database management routes
func (h *DatabaseHandler) RegisterRoutes(mux *http.ServeMux) {
	// Database statistics and monitoring
	mux.HandleFunc("/api/database/stats", h.handleDatabaseStats)
	mux.HandleFunc("/api/database/pool", h.handlePoolStats)
	mux.HandleFunc("/api/database/optimizer", h.handleOptimizerStats)
	mux.HandleFunc("/api/database/size", h.handleDatabaseSize)
	mux.HandleFunc("/api/database/tables", h.handleTableSizes)
	mux.HandleFunc("/api/database/indexes", h.handleIndexUsage)
	
	// Database operations
	mux.HandleFunc("/api/database/optimize", h.handleOptimize)
	mux.HandleFunc("/api/database/vacuum", h.handleVacuum)
	mux.HandleFunc("/api/database/analyze", h.handleAnalyze)
	mux.HandleFunc("/api/database/backup", h.handleBackup)
	
	// Query analysis
	mux.HandleFunc("/api/database/query-plan", h.handleQueryPlan)
	
	// Health check
	mux.HandleFunc("/api/database/health", h.handleDatabaseHealth)
	
	h.logger.Info("Database management API routes registered")
}

// Database statistics endpoints

func (h *DatabaseHandler) handleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	stats := h.db.GetStats()
	poolStats := h.db.GetPoolStats()
	optimizerStats := h.db.GetOptimizerStats()
	
	response := map[string]interface{}{
		"database_stats":  stats,
		"pool_stats":      poolStats,
		"optimizer_stats": optimizerStats,
		"timestamp":       time.Now(),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *DatabaseHandler) handlePoolStats(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	poolStats := h.db.GetPoolStats()
	if poolStats == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Connection pool not available")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"pool_stats": poolStats,
		"timestamp":  time.Now(),
	})
}

func (h *DatabaseHandler) handleOptimizerStats(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	optimizerStats := h.db.GetOptimizerStats()
	if optimizerStats == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Database optimizer not available")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"optimizer_stats": optimizerStats,
		"timestamp":       time.Now(),
	})
}

func (h *DatabaseHandler) handleDatabaseSize(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	sizes, err := h.db.GetDatabaseSize()
	if err != nil {
		h.logger.Error("Failed to get database size: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get database size")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"sizes":     sizes,
		"timestamp": time.Now(),
	})
}

func (h *DatabaseHandler) handleTableSizes(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	sizes, err := h.db.GetTableSizes()
	if err != nil {
		h.logger.Error("Failed to get table sizes: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get table sizes")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"table_sizes": sizes,
		"timestamp":   time.Now(),
	})
}

func (h *DatabaseHandler) handleIndexUsage(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	usage, err := h.db.GetIndexUsage()
	if err != nil {
		h.logger.Error("Failed to get index usage: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get index usage")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"index_usage": usage,
		"timestamp":   time.Now(),
	})
}

// Database operations endpoints

func (h *DatabaseHandler) handleOptimize(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	h.logger.Info("Manual database optimization requested")
	
	start := time.Now()
	err := h.db.OptimizeNow()
	duration := time.Since(start)
	
	if err != nil {
		h.logger.Error("Database optimization failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Optimization failed: "+err.Error())
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Database optimization completed successfully",
		"duration":  duration.String(),
		"timestamp": time.Now(),
	})
}

func (h *DatabaseHandler) handleVacuum(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	h.logger.Info("Manual database vacuum requested")
	
	start := time.Now()
	
	// Use optimizer if available, otherwise fallback to basic VACUUM
	var err error
	if optimizer := h.db.GetOptimizer(); optimizer != nil {
		err = optimizer.VacuumDatabase()
	} else {
		_, err = h.db.Exec("VACUUM")
	}
	
	duration := time.Since(start)
	
	if err != nil {
		h.logger.Error("Database vacuum failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Vacuum failed: "+err.Error())
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Database vacuum completed successfully",
		"duration":  duration.String(),
		"timestamp": time.Now(),
	})
}

func (h *DatabaseHandler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	h.logger.Info("Manual database analyze requested")
	
	start := time.Now()
	
	// Use optimizer if available, otherwise fallback to basic ANALYZE
	var err error
	if optimizer := h.db.GetOptimizer(); optimizer != nil {
		err = optimizer.AnalyzeDatabase()
	} else {
		_, err = h.db.Exec("ANALYZE")
	}
	
	duration := time.Since(start)
	
	if err != nil {
		h.logger.Error("Database analyze failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Analyze failed: "+err.Error())
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Database analyze completed successfully",
		"duration":  duration.String(),
		"timestamp": time.Now(),
	})
}

func (h *DatabaseHandler) handleBackup(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get backup path from query parameter
	backupPath := r.URL.Query().Get("path")
	if backupPath == "" {
		backupPath = "./data/backup/analytics_" + time.Now().Format("20060102_150405") + ".db"
	}
	
	h.logger.Info("Database backup requested to: %s", backupPath)
	
	start := time.Now()
	err := h.db.Backup(backupPath)
	duration := time.Since(start)
	
	if err != nil {
		h.logger.Error("Database backup failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Backup failed: "+err.Error())
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":     "Database backup completed successfully",
		"backup_path": backupPath,
		"duration":    duration.String(),
		"timestamp":   time.Now(),
	})
}

// Query analysis endpoints

func (h *DatabaseHandler) handleQueryPlan(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Parse request body
	var request struct {
		Query string        `json:"query"`
		Args  []interface{} `json:"args"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if request.Query == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Query is required")
		return
	}
	
	plan, err := h.db.GetQueryPlan(request.Query, request.Args...)
	if err != nil {
		h.logger.Error("Failed to get query plan: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to analyze query")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"query_plan": plan,
		"timestamp":  time.Now(),
	})
}

// Health check endpoint

func (h *DatabaseHandler) handleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Perform health check
	err := h.db.Health()
	
	if err != nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "Database health check failed: "+err.Error())
		return
	}
	
	// Get additional health metrics
	stats := h.db.GetStats()
	poolStats := h.db.GetPoolStats()
	
	response := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now(),
		"db_stats":   stats,
		"pool_stats": poolStats,
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (h *DatabaseHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (h *DatabaseHandler) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response: %v", err)
	}
}

func (h *DatabaseHandler) writeErrorResponse(w http.ResponseWriter, status int, message string) {
	h.writeJSONResponse(w, status, map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now(),
	})
}