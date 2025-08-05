package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebSocket connections
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powergrid_websocket_connections_active",
		Help: "Number of active WebSocket connections",
	})

	TotalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powergrid_websocket_connections_total",
		Help: "Total number of WebSocket connections",
	})

	// Game metrics
	ActiveGames = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powergrid_games_active",
		Help: "Number of active games",
	})

	GamesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powergrid_games_created_total",
		Help: "Total number of games created",
	})

	// Lobby metrics
	ActiveLobbies = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powergrid_lobbies_active",
		Help: "Number of active lobbies",
	})

	// Message metrics
	MessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "powergrid_messages_received_total",
		Help: "Total number of messages received by type",
	}, []string{"type"})

	MessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "powergrid_messages_sent_total",
		Help: "Total number of messages sent by type",
	}, []string{"type"})

	// Error metrics
	Errors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "powergrid_errors_total",
		Help: "Total number of errors by type",
	}, []string{"type"})

	// Performance metrics
	MessageProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "powergrid_message_processing_duration_seconds",
		Help:    "Time spent processing messages",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	GamePhaseDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "powergrid_game_phase_duration_seconds",
		Help:    "Duration of game phases",
		Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
	}, []string{"phase"})
)