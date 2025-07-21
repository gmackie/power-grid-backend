package ai

import (
	"sync"
	"time"

	"powergrid/pkg/protocol"
)

// GameMonitor monitors game state for completion and collects statistics
type GameMonitor struct {
	mu              sync.RWMutex
	gameID          string
	startTime       time.Time
	endTime         time.Time
	isCompleted     bool
	winner          string
	finalState      *protocol.GameStatePayload
	
	// Statistics
	turnCount       int
	phaseCount      map[protocol.GamePhase]int
	playerActions   map[string]int
	resourcesUsed   map[string]int
	plantsAuctioned int
	citiesBuilt     map[string]int
	
	// Callbacks
	onComplete      func(monitor *GameMonitor)
}

// NewGameMonitor creates a new game monitor
func NewGameMonitor(gameID string) *GameMonitor {
	return &GameMonitor{
		gameID:        gameID,
		startTime:     time.Now(),
		phaseCount:    make(map[protocol.GamePhase]int),
		playerActions: make(map[string]int),
		resourcesUsed: make(map[string]int),
		citiesBuilt:   make(map[string]int),
	}
}

// SetCompletionCallback sets the callback for game completion
func (m *GameMonitor) SetCompletionCallback(callback func(monitor *GameMonitor)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onComplete = callback
}

// UpdateGameState updates the monitor with new game state
func (m *GameMonitor) UpdateGameState(state *protocol.GameStatePayload) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check for game completion
	if state.Status == protocol.StatusFinished || state.CurrentPhase == protocol.PhaseGameEnd {
		if !m.isCompleted {
			m.handleGameCompletion(state)
		}
		return
	}
	
	// Update statistics
	m.phaseCount[state.CurrentPhase]++
	m.turnCount++
	
	// Track city growth
	for playerID, player := range state.Players {
		m.citiesBuilt[playerID] = len(player.Cities)
		
		// Track resource usage
		for resource, count := range player.Resources {
			m.resourcesUsed[resource] += count
		}
	}
	
	// Check win conditions
	if m.checkWinConditions(state) {
		m.handleGameCompletion(state)
	}
}

// RecordAction records a player action
func (m *GameMonitor) RecordAction(playerID string, action protocol.MessageType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.playerActions[playerID]++
	
	switch action {
	case protocol.MsgBidPlant:
		m.plantsAuctioned++
	}
}

// IsCompleted returns whether the game is completed
func (m *GameMonitor) IsCompleted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isCompleted
}

// GetWinner returns the winner of the game
func (m *GameMonitor) GetWinner() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.winner
}

// GetDuration returns the game duration
func (m *GameMonitor) GetDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.isCompleted {
		return m.endTime.Sub(m.startTime)
	}
	return time.Since(m.startTime)
}

// GetStatistics returns game statistics
func (m *GameMonitor) GetStatistics() GameStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := GameStatistics{
		GameID:          m.gameID,
		Duration:        m.GetDuration(),
		TurnCount:       m.turnCount,
		PhaseCount:      make(map[string]int),
		PlayerActions:   m.playerActions,
		ResourcesUsed:   m.resourcesUsed,
		PlantsAuctioned: m.plantsAuctioned,
		CitiesBuilt:     m.citiesBuilt,
		Winner:          m.winner,
		Completed:       m.isCompleted,
	}
	
	// Convert phase count
	for phase, count := range m.phaseCount {
		stats.PhaseCount[string(phase)] = count
	}
	
	return stats
}

// Private methods

func (m *GameMonitor) checkWinConditions(state *protocol.GameStatePayload) bool {
	// Check if any player has reached winning city count
	winningCityCount := 17 // Standard win condition
	
	for _, player := range state.Players {
		if len(player.Cities) >= winningCityCount {
			return true
		}
	}
	
	// Check if game has reached maximum rounds
	maxRounds := 20
	if state.CurrentRound >= maxRounds {
		return true
	}
	
	return false
}

func (m *GameMonitor) handleGameCompletion(state *protocol.GameStatePayload) {
	m.isCompleted = true
	m.endTime = time.Now()
	m.finalState = state
	
	// Determine winner
	m.winner = m.determineWinner(state)
	
	// Call completion callback if set
	if m.onComplete != nil {
		go m.onComplete(m)
	}
}

func (m *GameMonitor) determineWinner(state *protocol.GameStatePayload) string {
	if state == nil || len(state.Players) == 0 {
		return ""
	}
	
	var winner string
	maxScore := 0
	
	for playerID, player := range state.Players {
		// Score based on cities powered, then money
		score := player.PoweredCities*1000 + player.Money
		
		if score > maxScore {
			maxScore = score
			winner = playerID
		}
	}
	
	// Get winner name
	if winnerPlayer, ok := state.Players[winner]; ok {
		return winnerPlayer.Name
	}
	
	return winner
}

// GameStatistics represents collected game statistics
type GameStatistics struct {
	GameID          string
	Duration        time.Duration
	TurnCount       int
	PhaseCount      map[string]int
	PlayerActions   map[string]int
	ResourcesUsed   map[string]int
	PlantsAuctioned int
	CitiesBuilt     map[string]int
	Winner          string
	Completed       bool
}

// MonitoredAIClient wraps an AI client with monitoring
type MonitoredAIClient struct {
	*AIClient
	monitor *GameMonitor
}

// NewMonitoredAIClient creates a new monitored AI client
func NewMonitoredAIClient(config ClientConfig, monitor *GameMonitor) (*MonitoredAIClient, error) {
	client, err := NewAIClient(config)
	if err != nil {
		return nil, err
	}
	
	return &MonitoredAIClient{
		AIClient: client,
		monitor:  monitor,
	}, nil
}

// Override handleGameState to update monitor
func (c *MonitoredAIClient) handleGameState(msg *protocol.Message) {
	// Call parent implementation
	c.AIClient.handleGameState(msg)
	
	// Update monitor
	if c.gameState != nil && c.monitor != nil {
		c.monitor.UpdateGameState(c.gameState)
	}
}

// Override sendMessage to record actions
func (c *MonitoredAIClient) sendMessage(msg *protocol.Message) error {
	// Record action
	if c.monitor != nil && c.playerID != "" {
		c.monitor.RecordAction(c.playerID, msg.Type)
	}
	
	// Call parent implementation
	return c.AIClient.sendMessage(msg)
}