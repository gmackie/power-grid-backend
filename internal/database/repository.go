package database

import (
	"powergrid/internal/database/repositories"
	"powergrid/pkg/logger"
)

// Repository provides access to all database repositories
type Repository struct {
	Player      *repositories.PlayerRepository
	Game        *repositories.GameRepository
	Achievement *repositories.AchievementRepository
	Analytics   *repositories.AnalyticsRepository
	logger      *logger.ColoredLogger
}

// NewRepository creates a new repository collection
func NewRepository(db *DB) *Repository {
	return &Repository{
		Player:      repositories.NewPlayerRepository(db.DB),
		Game:        repositories.NewGameRepository(db.DB),
		Achievement: repositories.NewAchievementRepository(db.DB),
		Analytics:   repositories.NewAnalyticsRepository(db.DB),
		logger:      logger.CreateAILogger("Repository", logger.ColorWhite),
	}
}

// Close closes all repository connections
func (r *Repository) Close() error {
	r.logger.Debug("Closing repository connections")
	// Individual repositories don't need explicit closing since they share the same DB connection
	return nil
}

// Health checks the health of all repositories
func (r *Repository) Health() error {
	// Could add health checks for each repository here if needed
	return nil
}