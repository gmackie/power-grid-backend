package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"powergrid/pkg/logger"
)

// Optimizer provides database optimization and maintenance features
type Optimizer struct {
	db     *sql.DB
	pool   *ConnectionPool
	config *OptimizerConfig
	logger *logger.ColoredLogger
	
	// Optimization state
	running    bool
	stopChan   chan struct{}
	runningMux sync.RWMutex
	
	// Statistics
	stats *OptimizationStats
}

// OptimizerConfig holds optimizer configuration
type OptimizerConfig struct {
	// Automatic optimization
	AutoOptimize         bool
	OptimizeInterval     time.Duration
	VacuumInterval       time.Duration
	AnalyzeInterval      time.Duration
	
	// WAL management
	WALCheckpointInterval time.Duration
	WALSizeThreshold     int64 // bytes
	AutoWALCheckpoint    bool
	
	// Index optimization
	AutoReindex          bool
	ReindexInterval      time.Duration
	IndexUsageThreshold  float64
	
	// Query optimization
	QueryPlanAnalysis    bool
	SlowQueryOptimization bool
	
	// Maintenance windows
	MaintenanceWindows   []MaintenanceWindow
	SkipOptimizationDuringPeak bool
}

// MaintenanceWindow defines when maintenance can be performed
type MaintenanceWindow struct {
	StartHour int
	EndHour   int
	Days      []time.Weekday
}

// OptimizationStats tracks optimization metrics
type OptimizationStats struct {
	// Vacuum statistics
	VacuumCount         int64
	LastVacuum          time.Time
	VacuumDuration      time.Duration
	
	// Analyze statistics
	AnalyzeCount        int64
	LastAnalyze         time.Time
	AnalyzeDuration     time.Duration
	
	// WAL statistics
	WALCheckpointCount  int64
	LastWALCheckpoint   time.Time
	WALSize             int64
	
	// Index statistics
	ReindexCount        int64
	LastReindex         time.Time
	UnusedIndexes       []string
	
	// Query statistics
	QueryPlansAnalyzed  int64
	SlowQueriesOptimized int64
	
	// Overall statistics
	StartTime           time.Time
	TotalOptimizations  int64
	OptimizationErrors  int64
}

// QueryPlan represents a query execution plan
type QueryPlan struct {
	Query       string
	Plan        string
	Cost        float64
	Duration    time.Duration
	Optimizable bool
	Suggestions []string
}

// IndexUsage tracks index usage statistics
type IndexUsage struct {
	IndexName   string
	TableName   string
	UsageCount  int64
	LastUsed    time.Time
	SizeBytes   int64
	Efficiency  float64
}

// DefaultOptimizerConfig returns default optimizer configuration
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		// Automatic optimization
		AutoOptimize:         true,
		OptimizeInterval:     1 * time.Hour,
		VacuumInterval:       6 * time.Hour,
		AnalyzeInterval:      2 * time.Hour,
		
		// WAL management
		WALCheckpointInterval: 15 * time.Minute,
		WALSizeThreshold:     50 * 1024 * 1024, // 50MB
		AutoWALCheckpoint:    true,
		
		// Index optimization
		AutoReindex:          true,
		ReindexInterval:      24 * time.Hour,
		IndexUsageThreshold:  0.1, // 10% usage threshold
		
		// Query optimization
		QueryPlanAnalysis:    true,
		SlowQueryOptimization: true,
		
		// Maintenance windows (default: 2-6 AM)
		MaintenanceWindows: []MaintenanceWindow{
			{
				StartHour: 2,
				EndHour:   6,
				Days:      []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
			},
		},
		SkipOptimizationDuringPeak: true,
	}
}

// NewOptimizer creates a new database optimizer
func NewOptimizer(db *sql.DB, pool *ConnectionPool, config *OptimizerConfig) *Optimizer {
	opt := &Optimizer{
		db:     db,
		pool:   pool,
		config: config,
		logger: logger.CreateAILogger("Optimizer", logger.ColorBrightYellow),
		stats: &OptimizationStats{
			StartTime: time.Now(),
		},
		stopChan: make(chan struct{}),
	}
	
	return opt
}

// Start begins automatic optimization
func (o *Optimizer) Start() {
	o.runningMux.Lock()
	defer o.runningMux.Unlock()
	
	if o.running {
		return
	}
	
	o.running = true
	o.logger.Info("Starting database optimizer")
	
	// Start optimization routines
	if o.config.AutoOptimize {
		go o.optimizationLoop()
	}
	
	if o.config.AutoWALCheckpoint {
		go o.walCheckpointLoop()
	}
}

// Stop stops automatic optimization
func (o *Optimizer) Stop() {
	o.runningMux.Lock()
	defer o.runningMux.Unlock()
	
	if !o.running {
		return
	}
	
	o.running = false
	close(o.stopChan)
	o.logger.Info("Database optimizer stopped")
}

// OptimizeNow performs immediate optimization
func (o *Optimizer) OptimizeNow() error {
	o.logger.Info("Starting manual database optimization")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	var errors []error
	
	// Run PRAGMA optimize
	if err := o.pragmaOptimize(ctx); err != nil {
		errors = append(errors, fmt.Errorf("pragma optimize failed: %w", err))
	}
	
	// Run VACUUM
	if err := o.vacuum(ctx); err != nil {
		errors = append(errors, fmt.Errorf("vacuum failed: %w", err))
	}
	
	// Run ANALYZE
	if err := o.analyze(ctx); err != nil {
		errors = append(errors, fmt.Errorf("analyze failed: %w", err))
	}
	
	// WAL checkpoint
	if err := o.walCheckpoint(ctx); err != nil {
		errors = append(errors, fmt.Errorf("wal checkpoint failed: %w", err))
	}
	
	o.stats.TotalOptimizations++
	
	if len(errors) > 0 {
		o.stats.OptimizationErrors++
		return fmt.Errorf("optimization completed with errors: %v", errors)
	}
	
	o.logger.Info("Database optimization completed successfully")
	return nil
}

// VacuumDatabase performs database vacuum operation
func (o *Optimizer) VacuumDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	return o.vacuum(ctx)
}

// AnalyzeDatabase performs database analysis
func (o *Optimizer) AnalyzeDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	return o.analyze(ctx)
}

// GetQueryPlan analyzes a query's execution plan
func (o *Optimizer) GetQueryPlan(query string, args ...interface{}) (*QueryPlan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Use EXPLAIN QUERY PLAN
	explainQuery := "EXPLAIN QUERY PLAN " + query
	rows, err := o.pool.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get query plan: %w", err)
	}
	defer rows.Close()
	
	var planLines []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			return nil, fmt.Errorf("failed to scan query plan: %w", err)
		}
		planLines = append(planLines, detail)
	}
	
	plan := &QueryPlan{
		Query: query,
		Plan:  strings.Join(planLines, "\n"),
	}
	
	// Analyze plan for optimization opportunities
	o.analyzeQueryPlan(plan)
	
	o.stats.QueryPlansAnalyzed++
	
	return plan, nil
}

// GetIndexUsage returns index usage statistics
func (o *Optimizer) GetIndexUsage() ([]IndexUsage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	query := `
		SELECT 
			name,
			tbl_name,
			0 as usage_count,
			datetime('now') as last_used,
			0 as size_bytes,
			0.0 as efficiency
		FROM sqlite_master 
		WHERE type = 'index' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`
	
	rows, err := o.pool.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get index usage: %w", err)
	}
	defer rows.Close()
	
	var usage []IndexUsage
	for rows.Next() {
		var idx IndexUsage
		var lastUsedStr string
		
		if err := rows.Scan(&idx.IndexName, &idx.TableName, &idx.UsageCount, 
			&lastUsedStr, &idx.SizeBytes, &idx.Efficiency); err != nil {
			return nil, fmt.Errorf("failed to scan index usage: %w", err)
		}
		
		// Parse last used time
		if lastUsed, err := time.Parse("2006-01-02 15:04:05", lastUsedStr); err == nil {
			idx.LastUsed = lastUsed
		}
		
		usage = append(usage, idx)
	}
	
	return usage, nil
}

// GetDatabaseSize returns database size information
func (o *Optimizer) GetDatabaseSize() (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	sizes := make(map[string]int64)
	
	// Get total database size
	var totalSize int64
	if err := o.pool.QueryRowContext(ctx, "PRAGMA page_count").Scan(&totalSize); err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}
	
	var pageSize int64
	if err := o.pool.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return nil, fmt.Errorf("failed to get page size: %w", err)
	}
	
	sizes["total_size"] = totalSize * pageSize
	sizes["page_size"] = pageSize
	sizes["page_count"] = totalSize
	
	// Get WAL size
	var walSize int64
	if err := o.pool.QueryRowContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)").Scan(&walSize); err == nil {
		sizes["wal_size"] = walSize
	}
	
	return sizes, nil
}

// GetStats returns optimization statistics
func (o *Optimizer) GetStats() *OptimizationStats {
	// Return a copy of the current stats
	statsCopy := *o.stats
	return &statsCopy
}

// Private methods

func (o *Optimizer) optimizationLoop() {
	ticker := time.NewTicker(o.config.OptimizeInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if o.shouldOptimize() {
				if err := o.OptimizeNow(); err != nil {
					o.logger.Error("Automatic optimization failed: %v", err)
				}
			}
		case <-o.stopChan:
			return
		}
	}
}

func (o *Optimizer) walCheckpointLoop() {
	ticker := time.NewTicker(o.config.WALCheckpointInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if o.shouldPerformWALCheckpoint() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if err := o.walCheckpoint(ctx); err != nil {
					o.logger.Error("WAL checkpoint failed: %v", err)
				}
				cancel()
			}
		case <-o.stopChan:
			return
		}
	}
}

func (o *Optimizer) shouldOptimize() bool {
	if o.config.SkipOptimizationDuringPeak {
		return o.isInMaintenanceWindow()
	}
	return true
}

func (o *Optimizer) shouldPerformWALCheckpoint() bool {
	// Check WAL size threshold
	sizes, err := o.GetDatabaseSize()
	if err != nil {
		return true // If we can't check, do it anyway
	}
	
	if walSize, ok := sizes["wal_size"]; ok {
		return walSize > o.config.WALSizeThreshold
	}
	
	return true
}

func (o *Optimizer) isInMaintenanceWindow() bool {
	now := time.Now()
	currentHour := now.Hour()
	currentWeekday := now.Weekday()
	
	for _, window := range o.config.MaintenanceWindows {
		// Check if current day is in the maintenance window
		dayMatch := false
		for _, day := range window.Days {
			if day == currentWeekday {
				dayMatch = true
				break
			}
		}
		
		if !dayMatch {
			continue
		}
		
		// Check if current hour is in the maintenance window
		if window.StartHour <= window.EndHour {
			// Same day window
			if currentHour >= window.StartHour && currentHour < window.EndHour {
				return true
			}
		} else {
			// Overnight window
			if currentHour >= window.StartHour || currentHour < window.EndHour {
				return true
			}
		}
	}
	
	return false
}

func (o *Optimizer) pragmaOptimize(ctx context.Context) error {
	start := time.Now()
	_, err := o.pool.ExecContext(ctx, "PRAGMA optimize")
	if err != nil {
		return err
	}
	
	o.logger.Debug("PRAGMA optimize completed in %v", time.Since(start))
	return nil
}

func (o *Optimizer) vacuum(ctx context.Context) error {
	start := time.Now()
	_, err := o.pool.ExecContext(ctx, "VACUUM")
	if err != nil {
		return err
	}
	
	duration := time.Since(start)
	o.stats.VacuumCount++
	o.stats.LastVacuum = time.Now()
	o.stats.VacuumDuration = duration
	
	o.logger.Info("VACUUM completed in %v", duration)
	return nil
}

func (o *Optimizer) analyze(ctx context.Context) error {
	start := time.Now()
	_, err := o.pool.ExecContext(ctx, "ANALYZE")
	if err != nil {
		return err
	}
	
	duration := time.Since(start)
	o.stats.AnalyzeCount++
	o.stats.LastAnalyze = time.Now()
	o.stats.AnalyzeDuration = duration
	
	o.logger.Info("ANALYZE completed in %v", duration)
	return nil
}

func (o *Optimizer) walCheckpoint(ctx context.Context) error {
	start := time.Now()
	_, err := o.pool.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return err
	}
	
	o.stats.WALCheckpointCount++
	o.stats.LastWALCheckpoint = time.Now()
	
	o.logger.Debug("WAL checkpoint completed in %v", time.Since(start))
	return nil
}

func (o *Optimizer) analyzeQueryPlan(plan *QueryPlan) {
	// Simple plan analysis - in production, this would be more sophisticated
	planLower := strings.ToLower(plan.Plan)
	
	// Check for common optimization opportunities
	if strings.Contains(planLower, "scan") && !strings.Contains(planLower, "index") {
		plan.Optimizable = true
		plan.Suggestions = append(plan.Suggestions, "Consider adding an index for table scan")
	}
	
	if strings.Contains(planLower, "temp b-tree") {
		plan.Optimizable = true
		plan.Suggestions = append(plan.Suggestions, "Query creates temporary B-tree, consider optimizing ORDER BY or GROUP BY")
	}
	
	if strings.Contains(planLower, "nested loop") {
		plan.Optimizable = true
		plan.Suggestions = append(plan.Suggestions, "Nested loop join detected, consider index optimization")
	}
}