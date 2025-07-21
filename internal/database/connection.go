package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"powergrid/pkg/logger"
)

// DB represents a database connection with additional functionality
type DB struct {
	*sql.DB
	logger    *logger.ColoredLogger
	migrator  *Migrator
	pool      *ConnectionPool
	optimizer *Optimizer
}

// Config holds database configuration
type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	MigrateOnStart  bool
	SeedOnMigrate   bool
	
	// Advanced configuration
	EnableConnectionPool bool
	EnableOptimizer     bool
	PoolConfig          *PoolConfig
	OptimizerConfig     *OptimizerConfig
}

// DefaultConfig returns a default database configuration
func DefaultConfig(dataDir string) *Config {
	return &Config{
		Path:            filepath.Join(dataDir, "analytics.db"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		MigrateOnStart:  true,
		SeedOnMigrate:   true,
		
		// Advanced configuration
		EnableConnectionPool: true,
		EnableOptimizer:     true,
		PoolConfig:          DefaultPoolConfig(),
		OptimizerConfig:     DefaultOptimizerConfig(),
	}
}

// NewConnection creates a new database connection
func NewConnection(config *Config) (*DB, error) {
	logger := logger.CreateAILogger("DB", logger.ColorBlue)
	
	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := ensureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connect to SQLite database
	sqlDB, err := sql.Open("sqlite3", config.Path+"?_foreign_keys=on&_journal_mode=WAL&_timeout=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		DB:     sqlDB,
		logger: logger,
	}

	// Initialize migrator
	db.migrator = NewMigrator(sqlDB)

	// Run migrations if enabled
	if config.MigrateOnStart {
		if err := db.migrator.Migrate(); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		// Seed database if enabled
		if config.SeedOnMigrate {
			if err := db.migrator.Seed(); err != nil {
				logger.Warn("Failed to seed database: %v", err)
			}
		}
	}

	// Initialize connection pool if enabled
	if config.EnableConnectionPool && config.PoolConfig != nil {
		db.pool = NewConnectionPool(sqlDB, config.PoolConfig)
		logger.Info("Connection pool initialized")
	}

	// Initialize optimizer if enabled
	if config.EnableOptimizer && config.OptimizerConfig != nil {
		db.optimizer = NewOptimizer(sqlDB, db.pool, config.OptimizerConfig)
		db.optimizer.Start()
		logger.Info("Database optimizer initialized and started")
	}

	logger.Info("Connected to SQLite database: %s", config.Path)
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	// Stop optimizer first
	if db.optimizer != nil {
		db.optimizer.Stop()
	}
	
	// Close connection pool
	if db.pool != nil {
		if err := db.pool.Close(); err != nil {
			db.logger.Error("Failed to close connection pool: %v", err)
		}
	}
	
	if db.DB != nil {
		db.logger.Info("Closing database connection")
		return db.DB.Close()
	}
	return nil
}

// GetMigrator returns the database migrator
func (db *DB) GetMigrator() *Migrator {
	return db.migrator
}

// Health checks database health
func (db *DB) Health() error {
	return db.Ping()
}

// GetStats returns database statistics
func (db *DB) GetStats() sql.DBStats {
	return db.Stats()
}

// BeginTx starts a new transaction with context
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// WithTx executes a function within a transaction
func (db *DB) WithTx(fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.Error("Failed to rollback transaction: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Backup creates a backup of the database
func (db *DB) Backup(backupPath string) error {
	// Ensure backup directory exists
	dir := filepath.Dir(backupPath)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Use SQLite backup API
	return db.WithTx(func(tx *sql.Tx) error {
		// Simple backup using VACUUM INTO (SQLite 3.27+)
		_, err := tx.Exec("VACUUM INTO ?", backupPath)
		if err != nil {
			return fmt.Errorf("failed to backup database: %w", err)
		}
		
		db.logger.Info("Database backed up to: %s", backupPath)
		return nil
	})
}

// Optimize performs database optimization
func (db *DB) Optimize() error {
	db.logger.Info("Optimizing database...")
	
	operations := []string{
		"PRAGMA optimize",
		"VACUUM",
		"PRAGMA wal_checkpoint(TRUNCATE)",
	}

	for _, op := range operations {
		if _, err := db.Exec(op); err != nil {
			db.logger.Warn("Failed to execute optimization %s: %v", op, err)
		} else {
			db.logger.Debug("Executed: %s", op)
		}
	}

	db.logger.Info("Database optimization completed")
	return nil
}

// GetPool returns the connection pool
func (db *DB) GetPool() *ConnectionPool {
	return db.pool
}

// GetOptimizer returns the database optimizer
func (db *DB) GetOptimizer() *Optimizer {
	return db.optimizer
}

// GetPoolStats returns connection pool statistics
func (db *DB) GetPoolStats() *PoolStats {
	if db.pool != nil {
		return db.pool.GetStats()
	}
	return nil
}

// GetOptimizerStats returns optimizer statistics
func (db *DB) GetOptimizerStats() *OptimizationStats {
	if db.optimizer != nil {
		return db.optimizer.GetStats()
	}
	return nil
}

// OptimizeNow performs immediate database optimization
func (db *DB) OptimizeNow() error {
	if db.optimizer != nil {
		return db.optimizer.OptimizeNow()
	}
	return db.Optimize() // Fallback to basic optimization
}

// GetQueryPlan analyzes a query's execution plan
func (db *DB) GetQueryPlan(query string, args ...interface{}) (*QueryPlan, error) {
	if db.optimizer != nil {
		return db.optimizer.GetQueryPlan(query, args...)
	}
	return nil, fmt.Errorf("optimizer not available")
}

// GetIndexUsage returns index usage statistics
func (db *DB) GetIndexUsage() ([]IndexUsage, error) {
	if db.optimizer != nil {
		return db.optimizer.GetIndexUsage()
	}
	return nil, fmt.Errorf("optimizer not available")
}

// GetDatabaseSize returns detailed database size information
func (db *DB) GetDatabaseSize() (map[string]int64, error) {
	if db.optimizer != nil {
		return db.optimizer.GetDatabaseSize()
	}
	
	// Fallback to basic size calculation
	sizes := make(map[string]int64)
	var pageCount, pageSize int64
	
	if err := db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}
	
	if err := db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return nil, fmt.Errorf("failed to get page size: %w", err)
	}
	
	sizes["total_size"] = pageCount * pageSize
	sizes["page_count"] = pageCount
	sizes["page_size"] = pageSize
	
	return sizes, nil
}

// Query delegates to connection pool if available
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if db.pool != nil {
		return db.pool.Query(query, args...)
	}
	return db.DB.Query(query, args...)
}

// QueryRow delegates to connection pool if available
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	if db.pool != nil {
		return db.pool.QueryRow(query, args...)
	}
	return db.DB.QueryRow(query, args...)
}

// Exec delegates to connection pool if available
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	if db.pool != nil {
		return db.pool.Exec(query, args...)
	}
	return db.DB.Exec(query, args...)
}

// GetTableSizes returns the size of each table in the database
func (db *DB) GetTableSizes() (map[string]int, error) {
	query := `
		SELECT name, 
		       (SELECT COUNT(*) FROM sqlite_master s2 WHERE s2.name = s1.name AND s2.type = 'table') as row_count
		FROM sqlite_master s1
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table sizes: %w", err)
	}
	defer rows.Close()

	sizes := make(map[string]int)
	for rows.Next() {
		var tableName string
		var rowCount int
		if err := rows.Scan(&tableName, &rowCount); err != nil {
			return nil, fmt.Errorf("failed to scan table size: %w", err)
		}

		// Get actual row count
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := db.QueryRow(countQuery).Scan(&rowCount); err != nil {
			db.logger.Warn("Failed to get row count for %s: %v", tableName, err)
			rowCount = 0
		}

		sizes[tableName] = rowCount
	}

	return sizes, nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if dir == "" {
		return nil
	}
	
	// Use os.MkdirAll to create directory structure
	return os.MkdirAll(dir, 0755)
}