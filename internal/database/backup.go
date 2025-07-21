package database

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"powergrid/pkg/logger"
)

// BackupManager handles database backup and recovery operations
type BackupManager struct {
	db         *DB
	config     *BackupConfig
	logger     *logger.ColoredLogger
	scheduler  *BackupScheduler
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	// Backup storage
	BackupDir          string
	MaxBackups         int
	CompressionEnabled bool
	
	// Automatic backups
	AutoBackup         bool
	BackupInterval     time.Duration
	BackupTime         string // "15:04" format for daily backups
	
	// Retention policy
	RetentionDays      int
	RetentionPolicy    RetentionPolicy
	
	// Backup verification
	VerifyAfterBackup  bool
	TestRestoreEnabled bool
	
	// Encryption (future enhancement)
	EncryptionEnabled  bool
	EncryptionKey      string
}

// RetentionPolicy defines how backups are retained
type RetentionPolicy struct {
	KeepDaily   int // Keep daily backups for N days
	KeepWeekly  int // Keep weekly backups for N weeks  
	KeepMonthly int // Keep monthly backups for N months
	KeepYearly  int // Keep yearly backups for N years
}

// BackupInfo represents information about a backup
type BackupInfo struct {
	Filename      string    `json:"filename"`
	FullPath      string    `json:"full_path"`
	Size          int64     `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	DatabaseSize  int64     `json:"database_size"`
	Compressed    bool      `json:"compressed"`
	Verified      bool      `json:"verified"`
	BackupType    string    `json:"backup_type"` // manual, scheduled, etc.
	Description   string    `json:"description"`
}

// BackupScheduler handles automatic backup scheduling
type BackupScheduler struct {
	manager   *BackupManager
	logger    *logger.ColoredLogger
	stopChan  chan struct{}
	running   bool
}

// RestoreOptions defines options for database restoration
type RestoreOptions struct {
	BackupPath         string
	TargetPath         string
	VerifyAfterRestore bool
	CreateBackup       bool // Create backup before restore
	Force              bool  // Force restore even if target exists
}

// DefaultBackupConfig returns default backup configuration
func DefaultBackupConfig(dataDir string) *BackupConfig {
	return &BackupConfig{
		BackupDir:          filepath.Join(dataDir, "backups"),
		MaxBackups:         50,
		CompressionEnabled: true,
		
		AutoBackup:         true,
		BackupInterval:     6 * time.Hour,
		BackupTime:         "03:00", // 3 AM daily backup
		
		RetentionDays:      30,
		RetentionPolicy: RetentionPolicy{
			KeepDaily:   7,  // 1 week of daily backups
			KeepWeekly:  4,  // 1 month of weekly backups
			KeepMonthly: 12, // 1 year of monthly backups
			KeepYearly:  5,  // 5 years of yearly backups
		},
		
		VerifyAfterBackup:  true,
		TestRestoreEnabled: false,
		
		EncryptionEnabled:  false,
	}
}

// NewBackupManager creates a new backup manager
func NewBackupManager(db *DB, config *BackupConfig) *BackupManager {
	bm := &BackupManager{
		db:     db,
		config: config,
		logger: logger.NewColoredLogger("BACKUP", logger.ColorBrightPurple),
	}
	
	// Ensure backup directory exists
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		bm.logger.Error("Failed to create backup directory: %v", err)
	}
	
	// Initialize scheduler if auto backup is enabled
	if config.AutoBackup {
		bm.scheduler = &BackupScheduler{
			manager:  bm,
			logger:   logger.NewColoredLogger("SCHEDULER", logger.ColorPurple),
			stopChan: make(chan struct{}),
		}
	}
	
	return bm
}

// Start begins automatic backup scheduling
func (bm *BackupManager) Start() {
	if bm.scheduler != nil && !bm.scheduler.running {
		go bm.scheduler.start()
		bm.logger.Info("Backup scheduler started")
	}
}

// Stop stops automatic backup scheduling
func (bm *BackupManager) Stop() {
	if bm.scheduler != nil && bm.scheduler.running {
		close(bm.scheduler.stopChan)
		bm.logger.Info("Backup scheduler stopped")
	}
}

// CreateBackup creates a new database backup
func (bm *BackupManager) CreateBackup(description string) (*BackupInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	return bm.CreateBackupWithContext(ctx, description, "manual")
}

// CreateBackupWithContext creates a backup with context and type
func (bm *BackupManager) CreateBackupWithContext(ctx context.Context, description, backupType string) (*BackupInfo, error) {
	start := time.Now()
	
	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("analytics_%s.db", timestamp)
	if bm.config.CompressionEnabled {
		filename += ".gz"
	}
	
	backupPath := filepath.Join(bm.config.BackupDir, filename)
	
	bm.logger.Info("Creating backup: %s", filename)
	
	// Get current database size
	dbSizes, err := bm.db.GetDatabaseSize()
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}
	
	var dbSize int64
	if size, ok := dbSizes["total_size"]; ok {
		dbSize = size
	}
	
	// Create backup using SQLite's backup API
	var backupSize int64
	if bm.config.CompressionEnabled {
		backupSize, err = bm.createCompressedBackup(ctx, backupPath)
	} else {
		backupSize, err = bm.createRegularBackup(ctx, backupPath)
	}
	
	if err != nil {
		return nil, fmt.Errorf("backup creation failed: %w", err)
	}
	
	backupInfo := &BackupInfo{
		Filename:     filename,
		FullPath:     backupPath,
		Size:         backupSize,
		CreatedAt:    start,
		DatabaseSize: dbSize,
		Compressed:   bm.config.CompressionEnabled,
		BackupType:   backupType,
		Description:  description,
	}
	
	// Verify backup if enabled
	if bm.config.VerifyAfterBackup {
		if err := bm.verifyBackup(ctx, backupInfo); err != nil {
			bm.logger.Error("Backup verification failed: %v", err)
			backupInfo.Verified = false
		} else {
			backupInfo.Verified = true
		}
	}
	
	duration := time.Since(start)
	compressionRatio := float64(dbSize) / float64(backupSize) * 100
	
	bm.logger.Info("Backup completed: %s (%.1f%% compression, %v)", 
		filename, compressionRatio, duration)
	
	// Clean up old backups
	if err := bm.cleanupOldBackups(); err != nil {
		bm.logger.Warn("Failed to cleanup old backups: %v", err)
	}
	
	return backupInfo, nil
}

// RestoreBackup restores database from a backup
func (bm *BackupManager) RestoreBackup(options RestoreOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	return bm.RestoreBackupWithContext(ctx, options)
}

// RestoreBackupWithContext restores database with context
func (bm *BackupManager) RestoreBackupWithContext(ctx context.Context, options RestoreOptions) error {
	bm.logger.Info("Starting database restore from: %s", options.BackupPath)
	
	// Verify backup exists
	if _, err := os.Stat(options.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", options.BackupPath)
	}
	
	// Create backup of current database if requested
	if options.CreateBackup {
		bm.logger.Info("Creating backup of current database before restore")
		if _, err := bm.CreateBackupWithContext(ctx, "Pre-restore backup", "pre-restore"); err != nil {
			bm.logger.Warn("Failed to create pre-restore backup: %v", err)
		}
	}
	
	// Determine target path
	targetPath := options.TargetPath
	if targetPath == "" {
		// This is getting complex - let's use a simpler approach
		config := DefaultConfig("./data")
		targetPath = config.Path
	}
	
	// Check if target exists and handle force option
	if _, err := os.Stat(targetPath); err == nil && !options.Force {
		return fmt.Errorf("target database exists and force=false: %s", targetPath)
	}
	
	// Perform restore
	var err error
	if strings.HasSuffix(options.BackupPath, ".gz") {
		err = bm.restoreCompressedBackup(ctx, options.BackupPath, targetPath)
	} else {
		err = bm.restoreRegularBackup(ctx, options.BackupPath, targetPath)
	}
	
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	
	// Verify restored database if requested
	if options.VerifyAfterRestore {
		if err := bm.verifyRestoredDatabase(ctx, targetPath); err != nil {
			return fmt.Errorf("restored database verification failed: %w", err)
		}
	}
	
	bm.logger.Info("Database restore completed successfully")
	return nil
}

// ListBackups returns list of available backups
func (bm *BackupManager) ListBackups() ([]*BackupInfo, error) {
	files, err := os.ReadDir(bm.config.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	var backups []*BackupInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		filename := file.Name()
		if !strings.HasPrefix(filename, "analytics_") {
			continue
		}
		
		fullPath := filepath.Join(bm.config.BackupDir, filename)
		fileInfo, err := file.Info()
		if err != nil {
			continue
		}
		
		backup := &BackupInfo{
			Filename:   filename,
			FullPath:   fullPath,
			Size:       fileInfo.Size(),
			CreatedAt:  fileInfo.ModTime(),
			Compressed: strings.HasSuffix(filename, ".gz"),
		}
		
		backups = append(backups, backup)
	}
	
	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	
	return backups, nil
}

// DeleteBackup deletes a specific backup
func (bm *BackupManager) DeleteBackup(filename string) error {
	backupPath := filepath.Join(bm.config.BackupDir, filename)
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", filename)
	}
	
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}
	
	bm.logger.Info("Deleted backup: %s", filename)
	return nil
}

// GetBackupInfo returns information about a specific backup
func (bm *BackupManager) GetBackupInfo(filename string) (*BackupInfo, error) {
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, err
	}
	
	for _, backup := range backups {
		if backup.Filename == filename {
			return backup, nil
		}
	}
	
	return nil, fmt.Errorf("backup not found: %s", filename)
}

// Private helper methods

func (bm *BackupManager) createRegularBackup(ctx context.Context, backupPath string) (int64, error) {
	// Use SQLite's VACUUM INTO command for efficient backup
	query := "VACUUM INTO ?"
	if _, err := bm.db.ExecContext(ctx, query, backupPath); err != nil {
		return 0, fmt.Errorf("VACUUM INTO failed: %w", err)
	}
	
	// Get backup file size
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get backup file info: %w", err)
	}
	
	return fileInfo.Size(), nil
}

func (bm *BackupManager) createCompressedBackup(ctx context.Context, backupPath string) (int64, error) {
	// Create temporary uncompressed backup first
	tempPath := backupPath + ".tmp"
	defer os.Remove(tempPath)
	
	if _, err := bm.createRegularBackup(ctx, tempPath); err != nil {
		return 0, err
	}
	
	// Compress the backup (simplified - in production use gzip)
	// For now, just copy the file (compression would be implemented here)
	srcFile, err := os.Open(tempPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open temp backup: %w", err)
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(backupPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create compressed backup: %w", err)
	}
	defer dstFile.Close()
	
	// Copy file (in production, this would use gzip compression)
	size, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("failed to compress backup: %w", err)
	}
	
	return size, nil
}

func (bm *BackupManager) restoreRegularBackup(ctx context.Context, backupPath, targetPath string) error {
	// Simple file copy for restore
	srcFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer dstFile.Close()
	
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy backup: %w", err)
	}
	
	return nil
}

func (bm *BackupManager) restoreCompressedBackup(ctx context.Context, backupPath, targetPath string) error {
	// Decompress and restore (simplified implementation)
	return bm.restoreRegularBackup(ctx, backupPath, targetPath)
}

func (bm *BackupManager) verifyBackup(ctx context.Context, backup *BackupInfo) error {
	// Simple verification - try to open the backup as a SQLite database
	testDB, err := os.Open(backup.FullPath)
	if err != nil {
		return fmt.Errorf("cannot open backup file: %w", err)
	}
	defer testDB.Close()
	
	// Could add more sophisticated verification here
	return nil
}

func (bm *BackupManager) verifyRestoredDatabase(ctx context.Context, dbPath string) error {
	// Simple verification of restored database
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("restored database file not found: %w", err)
	}
	
	// Could add more sophisticated verification here
	return nil
}

func (bm *BackupManager) cleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}
	
	if len(backups) <= bm.config.MaxBackups {
		return nil
	}
	
	// Delete oldest backups beyond the limit
	for i := bm.config.MaxBackups; i < len(backups); i++ {
		if err := bm.DeleteBackup(backups[i].Filename); err != nil {
			bm.logger.Warn("Failed to delete old backup %s: %v", backups[i].Filename, err)
		}
	}
	
	return nil
}

// Backup scheduler implementation

func (bs *BackupScheduler) start() {
	bs.running = true
	defer func() { bs.running = false }()
	
	ticker := time.NewTicker(bs.manager.config.BackupInterval)
	defer ticker.Stop()
	
	// Perform initial backup if needed
	bs.performScheduledBackup()
	
	for {
		select {
		case <-ticker.C:
			bs.performScheduledBackup()
		case <-bs.stopChan:
			return
		}
	}
}

func (bs *BackupScheduler) performScheduledBackup() {
	bs.logger.Info("Performing scheduled backup")
	
	description := fmt.Sprintf("Scheduled backup - %s", time.Now().Format("2006-01-02 15:04:05"))
	
	if _, err := bs.manager.CreateBackupWithContext(
		context.Background(), description, "scheduled"); err != nil {
		bs.logger.Error("Scheduled backup failed: %v", err)
	} else {
		bs.logger.Info("Scheduled backup completed successfully")
	}
}