package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete server configuration
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	WebSocket   WebSocketConfig   `yaml:"websocket"`
	Game        GameConfig        `yaml:"game"`
	Database    DatabaseConfig    `yaml:"database"`
	Logging     LoggingConfig     `yaml:"logging"`
	Security    SecurityConfig    `yaml:"security"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	Performance PerformanceConfig `yaml:"performance"`
	Mobile      MobileConfig      `yaml:"mobile"`
	Features    FeaturesConfig    `yaml:"features"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
}

// WebSocketConfig contains WebSocket settings
type WebSocketConfig struct {
	MaxConnections  int           `yaml:"max_connections"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	PingInterval    time.Duration `yaml:"ping_interval"`
	MaxMessageSize  int64         `yaml:"max_message_size"`
}

// GameConfig contains game-specific settings
type GameConfig struct {
	MaxPlayersPerGame   int           `yaml:"max_players_per_game"`
	MinPlayersPerGame   int           `yaml:"min_players_per_game"`
	GameTimeout         time.Duration `yaml:"game_timeout"`
	TurnTimeout         time.Duration `yaml:"turn_timeout"`
	LobbyTimeout        time.Duration `yaml:"lobby_timeout"`
	MaxConcurrentGames  int           `yaml:"max_concurrent_games"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Type               string `yaml:"type"`
	ConnectionString   string `yaml:"connection_string"`
	MaxConnections     int    `yaml:"max_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	File       string `yaml:"file"`
	MaxSize    string `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	CorsOrigins []string         `yaml:"cors_origins"`
	RateLimit   RateLimitConfig  `yaml:"rate_limit"`
	MaxRooms    int              `yaml:"max_rooms"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	Burst             int `yaml:"burst"`
}

// MonitoringConfig contains monitoring settings
type MonitoringConfig struct {
	Enabled         bool   `yaml:"enabled"`
	MetricsEndpoint string `yaml:"metrics_endpoint"`
	HealthEndpoint  string `yaml:"health_endpoint"`
}

// PerformanceConfig contains performance settings
type PerformanceConfig struct {
	EnableCompression bool `yaml:"enable_compression"`
	CacheStaticFiles  bool `yaml:"cache_static_files"`
	MaxCPUCores       int  `yaml:"max_cpu_cores"`
}

// MobileConfig contains mobile-specific settings
type MobileConfig struct {
	ConnectionRetryAttempts int           `yaml:"connection_retry_attempts"`
	ConnectionRetryDelay    time.Duration `yaml:"connection_retry_delay"`
	BackgroundPingInterval  time.Duration `yaml:"background_ping_interval"`
	LowDataMode             bool          `yaml:"low_data_mode"`
}

// FeaturesConfig contains feature flags
type FeaturesConfig struct {
	Chat           bool `yaml:"chat"`
	SpectatorMode  bool `yaml:"spectator_mode"`
	GameRecording  bool `yaml:"game_recording"`
	Statistics     bool `yaml:"statistics"`
	Leaderboards   bool `yaml:"leaderboards"`
	Tournaments    bool `yaml:"tournaments"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment-specific overrides
	cfg.applyEnvironmentOverrides()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvironmentOverrides applies environment-specific settings
func (c *Config) applyEnvironmentOverrides() {
	// Override with environment variables if set
	if port := os.Getenv("PORT"); port != "" {
		fmt.Sscanf(port, "%d", &c.Server.Port)
	}

	if host := os.Getenv("HOST"); host != "" {
		c.Server.Host = host
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		c.Server.Environment = env
	}

	// Apply development overrides if in development mode
	if c.Server.Environment == "development" {
		c.Logging.Level = "debug"
		c.Logging.Format = "text"
		c.Game.GameTimeout = 60 * time.Minute
		c.Game.TurnTimeout = 10 * time.Minute
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Server.Port)
	}

	if c.Game.MaxPlayersPerGame < c.Game.MinPlayersPerGame {
		return fmt.Errorf("max players (%d) must be >= min players (%d)", 
			c.Game.MaxPlayersPerGame, c.Game.MinPlayersPerGame)
	}

	if c.WebSocket.MaxConnections < 1 {
		return fmt.Errorf("max connections must be at least 1")
	}

	return nil
}

// GetAddr returns the server address in host:port format
func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}