package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"powergrid/pkg/logger"
)

// ConnectionPool manages database connections with advanced pooling features
type ConnectionPool struct {
	db     *sql.DB
	config *PoolConfig
	logger *logger.ColoredLogger
	
	// Pool statistics
	stats     *PoolStats
	statsLock sync.RWMutex
	
	// Health monitoring
	healthChecker *HealthChecker
	stopChan      chan struct{}
	stopped       int32
}

// PoolConfig holds advanced connection pool configuration
type PoolConfig struct {
	// Basic pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	
	// Advanced pool settings
	QueryTimeout         time.Duration
	TransactionTimeout   time.Duration
	HealthCheckInterval  time.Duration
	SlowQueryThreshold   time.Duration
	
	// Connection validation
	ValidateConnections bool
	MaxRetries         int
	RetryDelay         time.Duration
	
	// Monitoring
	EnableMetrics      bool
	MetricsInterval    time.Duration
	LogSlowQueries     bool
	LogConnectionStats bool
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	// Connection statistics
	TotalConnections     int64
	ActiveConnections    int64
	IdleConnections      int64
	WaitCount           int64
	WaitDuration        time.Duration
	
	// Query statistics
	QueryCount          int64
	SlowQueryCount      int64
	ErrorCount          int64
	AvgQueryDuration    time.Duration
	
	// Transaction statistics
	TransactionCount    int64
	TransactionErrors   int64
	AvgTransactionDuration time.Duration
	
	// Health statistics
	HealthCheckCount    int64
	HealthCheckErrors   int64
	LastHealthCheck     time.Time
	
	// Timestamps
	StartTime           time.Time
	LastStatsUpdate     time.Time
}

// HealthChecker monitors database health
type HealthChecker struct {
	pool     *ConnectionPool
	logger   *logger.ColoredLogger
	interval time.Duration
	stopChan chan struct{}
}

// QueryStats tracks individual query performance
type QueryStats struct {
	Query       string
	Duration    time.Duration
	Error       error
	StartTime   time.Time
	EndTime     time.Time
	RowsAffected int64
}

// DefaultPoolConfig returns optimized default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		// Basic settings
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 3 * time.Minute,
		
		// Advanced settings
		QueryTimeout:         30 * time.Second,
		TransactionTimeout:   60 * time.Second,
		HealthCheckInterval:  30 * time.Second,
		SlowQueryThreshold:   1 * time.Second,
		
		// Connection validation
		ValidateConnections: true,
		MaxRetries:         3,
		RetryDelay:         100 * time.Millisecond,
		
		// Monitoring
		EnableMetrics:      true,
		MetricsInterval:    1 * time.Minute,
		LogSlowQueries:     true,
		LogConnectionStats: true,
	}
}

// NewConnectionPool creates a new enhanced connection pool
func NewConnectionPool(db *sql.DB, config *PoolConfig) *ConnectionPool {
	pool := &ConnectionPool{
		db:     db,
		config: config,
		logger: logger.CreateAILogger("Pool", logger.ColorBrightBlue),
		stats: &PoolStats{
			StartTime: time.Now(),
		},
		stopChan: make(chan struct{}),
	}
	
	// Configure database connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	// Start health checker
	if config.HealthCheckInterval > 0 {
		pool.healthChecker = &HealthChecker{
			pool:     pool,
			logger:   logger.CreateAILogger("Health", logger.ColorGreen),
			interval: config.HealthCheckInterval,
			stopChan: make(chan struct{}),
		}
		go pool.healthChecker.start()
	}
	
	// Start metrics collector
	if config.EnableMetrics && config.MetricsInterval > 0 {
		go pool.startMetricsCollector()
	}
	
	pool.logger.Info("Database connection pool initialized with %d max connections", config.MaxOpenConns)
	return pool
}

// Query executes a query with monitoring and optimization
func (p *ConnectionPool) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

// QueryContext executes a query with context, monitoring, and optimization
func (p *ConnectionPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// Create timeout context if needed
	if p.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.QueryTimeout)
		defer cancel()
	}
	
	// Track query statistics
	start := time.Now()
	
	// Execute query with retries
	rows, err := p.executeWithRetry(ctx, func() (*sql.Rows, error) {
		return p.db.QueryContext(ctx, query, args...)
	})
	
	// Update statistics
	p.updateQueryStats(query, time.Since(start), err)
	
	return rows, err
}

// QueryRow executes a query that returns a single row
func (p *ConnectionPool) QueryRow(query string, args ...interface{}) *sql.Row {
	return p.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext executes a query that returns a single row with context
func (p *ConnectionPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// Create timeout context if needed
	if p.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.QueryTimeout)
		defer cancel()
	}
	
	// Track query start time
	start := time.Now()
	
	// Execute query
	row := p.db.QueryRowContext(ctx, query, args...)
	
	// Update statistics (we can't easily get the error here, so we'll estimate)
	p.updateQueryStats(query, time.Since(start), nil)
	
	return row
}

// Exec executes a query without returning rows
func (p *ConnectionPool) Exec(query string, args ...interface{}) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a query without returning rows with context
func (p *ConnectionPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Create timeout context if needed
	if p.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.QueryTimeout)
		defer cancel()
	}
	
	// Track query statistics
	start := time.Now()
	
	// Execute query with retries
	result, err := p.executeExecWithRetry(ctx, func() (sql.Result, error) {
		return p.db.ExecContext(ctx, query, args...)
	})
	
	// Update statistics
	p.updateQueryStats(query, time.Since(start), err)
	
	return result, err
}

// BeginTx starts a transaction with enhanced monitoring
func (p *ConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	// Create timeout context if needed
	if p.config.TransactionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.TransactionTimeout)
		defer cancel()
	}
	
	start := time.Now()
	tx, err := p.db.BeginTx(ctx, opts)
	
	// Update transaction statistics
	atomic.AddInt64(&p.stats.TransactionCount, 1)
	if err != nil {
		atomic.AddInt64(&p.stats.TransactionErrors, 1)
	}
	
	// Track transaction duration (simplified)
	p.updateTransactionStats(time.Since(start))
	
	return tx, err
}

// WithTransaction executes a function within a transaction with monitoring
func (p *ConnectionPool) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := p.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("Failed to rollback transaction: %v", rbErr)
		}
		return err
	}
	
	return tx.Commit()
}

// GetStats returns current pool statistics
func (p *ConnectionPool) GetStats() *PoolStats {
	p.statsLock.RLock()
	defer p.statsLock.RUnlock()
	
	// Get current database stats
	dbStats := p.db.Stats()
	
	// Update connection statistics
	p.stats.TotalConnections = int64(dbStats.OpenConnections)
	p.stats.ActiveConnections = int64(dbStats.InUse)
	p.stats.IdleConnections = int64(dbStats.Idle)
	p.stats.WaitCount = dbStats.WaitCount
	p.stats.WaitDuration = dbStats.WaitDuration
	p.stats.LastStatsUpdate = time.Now()
	
	// Return a copy of the stats
	statsCopy := *p.stats
	return &statsCopy
}

// Health checks database health
func (p *ConnectionPool) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Simple ping test
	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	
	// Try a simple query
	row := p.QueryRowContext(ctx, "SELECT 1")
	var result int
	if err := row.Scan(&result); err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}
	
	return nil
}

// Close closes the connection pool
func (p *ConnectionPool) Close() error {
	// Mark as stopped
	if !atomic.CompareAndSwapInt32(&p.stopped, 0, 1) {
		return nil // Already stopped
	}
	
	// Stop health checker
	if p.healthChecker != nil {
		close(p.healthChecker.stopChan)
	}
	
	// Stop metrics collector
	close(p.stopChan)
	
	// Close database
	p.logger.Info("Closing database connection pool")
	return p.db.Close()
}

// Private helper methods

func (p *ConnectionPool) executeWithRetry(ctx context.Context, fn func() (*sql.Rows, error)) (*sql.Rows, error) {
	var lastErr error
	for i := 0; i < p.config.MaxRetries; i++ {
		rows, err := fn()
		if err == nil {
			return rows, nil
		}
		
		lastErr = err
		
		// Check if we should retry
		if !p.shouldRetry(err) {
			break
		}
		
		// Wait before retry
		if i < p.config.MaxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(p.config.RetryDelay):
			}
		}
	}
	
	return nil, lastErr
}

func (p *ConnectionPool) executeExecWithRetry(ctx context.Context, fn func() (sql.Result, error)) (sql.Result, error) {
	var lastErr error
	for i := 0; i < p.config.MaxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check if we should retry
		if !p.shouldRetry(err) {
			break
		}
		
		// Wait before retry
		if i < p.config.MaxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(p.config.RetryDelay):
			}
		}
	}
	
	return nil, lastErr
}

func (p *ConnectionPool) shouldRetry(err error) bool {
	// Simple retry logic - in production, this would be more sophisticated
	if err == nil {
		return false
	}
	
	// Check for common retryable errors
	errStr := err.Error()
	retryableErrors := []string{
		"database is locked",
		"connection reset",
		"connection refused",
		"timeout",
	}
	
	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}
	
	return false
}

func (p *ConnectionPool) updateQueryStats(query string, duration time.Duration, err error) {
	atomic.AddInt64(&p.stats.QueryCount, 1)
	
	if err != nil {
		atomic.AddInt64(&p.stats.ErrorCount, 1)
	}
	
	if duration > p.config.SlowQueryThreshold {
		atomic.AddInt64(&p.stats.SlowQueryCount, 1)
		
		if p.config.LogSlowQueries {
			p.logger.Warn("Slow query detected: %s (duration: %v)", 
				truncateQuery(query), duration)
		}
	}
	
	// Update average query duration (simplified)
	p.statsLock.Lock()
	p.stats.AvgQueryDuration = (p.stats.AvgQueryDuration + duration) / 2
	p.statsLock.Unlock()
}

func (p *ConnectionPool) updateTransactionStats(duration time.Duration) {
	p.statsLock.Lock()
	p.stats.AvgTransactionDuration = (p.stats.AvgTransactionDuration + duration) / 2
	p.statsLock.Unlock()
}

func (p *ConnectionPool) startMetricsCollector() {
	ticker := time.NewTicker(p.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if p.config.LogConnectionStats {
				p.logConnectionStats()
			}
		case <-p.stopChan:
			return
		}
	}
}

func (p *ConnectionPool) logConnectionStats() {
	stats := p.GetStats()
	p.logger.Info("Pool Stats - Active: %d, Idle: %d, Wait: %d, Queries: %d, Errors: %d, Slow: %d",
		stats.ActiveConnections, stats.IdleConnections, stats.WaitCount,
		stats.QueryCount, stats.ErrorCount, stats.SlowQueryCount)
}

// Health checker implementation

func (h *HealthChecker) start() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.performHealthCheck()
		case <-h.stopChan:
			return
		}
	}
}

func (h *HealthChecker) performHealthCheck() {
	atomic.AddInt64(&h.pool.stats.HealthCheckCount, 1)
	
	if err := h.pool.Health(); err != nil {
		atomic.AddInt64(&h.pool.stats.HealthCheckErrors, 1)
		h.logger.Error("Health check failed: %v", err)
	} else {
		h.pool.stats.LastHealthCheck = time.Now()
		h.logger.Debug("Health check passed")
	}
}

// Utility functions

func truncateQuery(query string) string {
	if len(query) > 100 {
		return query[:100] + "..."
	}
	return query
}

func contains(s, substr string) bool {
	// Use a simple contains check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}