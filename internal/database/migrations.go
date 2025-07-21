package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"powergrid/pkg/logger"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	logger *logger.ColoredLogger
}

// NewMigrator creates a new migrator
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger.CreateAILogger("DB", logger.ColorBrightBlue),
	}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate() error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	m.logger.Info("Current database version: %d", currentVersion)

	// Load migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Filter migrations that need to be applied
	pendingMigrations := make([]Migration, 0)
	for _, migration := range migrations {
		if migration.Version > currentVersion {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		m.logger.Info("Database is up to date")
		return nil
	}

	m.logger.Info("Found %d pending migrations", len(pendingMigrations))

	// Apply pending migrations
	for _, migration := range pendingMigrations {
		if err := m.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", 
				migration.Version, migration.Name, err)
		}
		m.logger.Info("Applied migration %d: %s", migration.Version, migration.Name)
	}

	m.logger.Info("All migrations completed successfully")
	return nil
}

// GetCurrentVersion returns the current database version
func (m *Migrator) GetCurrentVersion() (int, error) {
	if err := m.createMigrationsTable(); err != nil {
		return 0, err
	}
	return m.getCurrentVersion()
}

// createMigrationsTable creates the migrations tracking table
func (m *Migrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// getCurrentVersion gets the latest applied migration version
func (m *Migrator) getCurrentVersion() (int, error) {
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"
	var version int
	err := m.db.QueryRow(query).Scan(&version)
	return version, err
}

// loadMigrations loads all migration files
func (m *Migrator) loadMigrations() ([]Migration, error) {
	migrations := make([]Migration, 0)

	// Load embedded migration files
	err := fs.WalkDir(migrationFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		// Parse version and name from filename
		filename := filepath.Base(path)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid migration filename format: %s", filename)
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid version in filename %s: %w", filename, err)
		}

		name := strings.TrimSuffix(parts[1], ".sql")

		// Read migration content
		content, err := fs.ReadFile(migrationFiles, path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		migration := Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// applyMigration applies a single migration
func (m *Migrator) applyMigration(migration Migration) error {
	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration in schema_migrations table
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// Rollback rolls back to a specific version (dangerous operation)
func (m *Migrator) Rollback(targetVersion int) error {
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d is not less than current version %d", 
			targetVersion, currentVersion)
	}

	m.logger.Warn("Rolling back from version %d to %d", currentVersion, targetVersion)
	m.logger.Warn("This is a destructive operation that may result in data loss")

	// For SQLite, we'll need to rebuild the schema from scratch
	// This is a simplified rollback - in production, you'd want proper down migrations
	if targetVersion == 0 {
		return m.resetDatabase()
	}

	// For partial rollbacks, you'd need down migration scripts
	return fmt.Errorf("partial rollbacks not implemented - use version 0 to reset database")
}

// resetDatabase drops all tables and starts fresh
func (m *Migrator) resetDatabase() error {
	m.logger.Warn("Resetting database - all data will be lost")

	// Get all table names
	rows, err := m.db.Query(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop all tables
	for _, table := range tables {
		_, err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		m.logger.Debug("Dropped table: %s", table)
	}

	m.logger.Info("Database reset completed")
	return nil
}

// GetMigrationStatus returns the status of all migrations
func (m *Migrator) GetMigrationStatus() ([]MigrationStatus, error) {
	// Ensure migrations table exists
	if err := m.createMigrationsTable(); err != nil {
		return nil, err
	}

	// Get applied migrations
	appliedMigrations := make(map[int]time.Time)
	rows, err := m.db.Query("SELECT version, applied_at FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		appliedMigrations[version] = appliedAt
	}

	// Load all available migrations
	availableMigrations, err := m.loadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Build status list
	status := make([]MigrationStatus, len(availableMigrations))
	for i, migration := range availableMigrations {
		appliedAt, isApplied := appliedMigrations[migration.Version]
		status[i] = MigrationStatus{
			Version:   migration.Version,
			Name:      migration.Name,
			Applied:   isApplied,
			AppliedAt: appliedAt,
		}
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

// Seed populates the database with initial data
func (m *Migrator) Seed() error {
	m.logger.Info("Seeding database with initial data")

	// Insert predefined achievements
	achievements := []struct {
		ID          string
		Name        string
		Description string
		Category    string
		Icon        string
		Points      int
		Criteria    string
	}{
		{"first_win", "First Victory", "Win your first game", "victory", "🏆", 10, "games_won >= 1"},
		{"hat_trick", "Hat Trick", "Win 3 games in a row", "victory", "🎩", 25, "winning_streak >= 3"},
		{"unstoppable", "Unstoppable", "Win 5 games in a row", "victory", "🔥", 50, "winning_streak >= 5"},
		{"master_strategist", "Master Strategist", "Win 50 games", "victory", "🧠", 100, "games_won >= 50"},
		{"money_bags", "Money Bags", "End a game with 200+ elektro", "economic", "💰", 20, "final_money >= 200"},
		{"resource_hoarder", "Resource Hoarder", "Own 20+ resources at once", "economic", "📦", 15, "max_resources >= 20"},
		{"eco_warrior", "Eco Warrior", "Win using only renewable power plants", "economic", "🌱", 30, "renewable_only_win"},
		{"city_builder", "City Builder", "Build in 15+ cities in a single game", "expansion", "🏙️", 25, "max_cities >= 15"},
		{"rapid_expansion", "Rapid Expansion", "Build in 10 cities within 5 rounds", "expansion", "⚡", 30, "rapid_expansion"},
		{"monopolist", "Monopolist", "Control all cities in a region", "expansion", "👑", 35, "region_monopoly"},
		{"plant_collector", "Plant Collector", "Own 5 power plants at once", "plants", "🏭", 20, "max_plants >= 5"},
		{"high_capacity", "High Capacity", "Own a plant that powers 7+ cities", "plants", "⚡", 15, "max_plant_capacity >= 7"},
		{"diversified", "Diversified Portfolio", "Own plants of 4 different resource types", "plants", "🎨", 25, "resource_diversity >= 4"},
		{"underdog", "Underdog Victory", "Win from last place in turn order", "special", "🐕", 40, "underdog_win"},
		{"perfectionist", "Perfectionist", "Win by powering all your cities", "special", "✨", 30, "perfect_power"},
		{"speed_demon", "Speed Demon", "Win a game in under 30 minutes", "special", "🏎️", 35, "speed_win"},
		{"regular_player", "Regular Player", "Play 25 games", "participation", "🎮", 15, "games_played >= 25"},
		{"dedicated_player", "Dedicated Player", "Play 100 games", "participation", "🌟", 50, "games_played >= 100"},
		{"map_explorer", "Map Explorer", "Play on all available maps", "participation", "🗺️", 25, "all_maps_played"},
	}

	for _, ach := range achievements {
		_, err := m.db.Exec(`
			INSERT OR IGNORE INTO achievements 
			(achievement_id, name, description, category, icon, points, criteria) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ach.ID, ach.Name, ach.Description, ach.Category, ach.Icon, ach.Points, ach.Criteria)
		if err != nil {
			return fmt.Errorf("failed to insert achievement %s: %w", ach.ID, err)
		}
	}

	m.logger.Info("Seeded %d achievements", len(achievements))
	return nil
}