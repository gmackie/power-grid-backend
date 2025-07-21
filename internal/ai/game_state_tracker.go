package ai

import (
	"sync"
	"time"

	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// GameStateTracker tracks and analyzes game state for AI decision making
type GameStateTracker struct {
	mu              sync.RWMutex
	currentState    *protocol.GameStatePayload
	playerID        string
	logger          *logger.ColoredLogger
	
	// Historical data
	history         []GameStateSnapshot
	maxHistory      int
	
	// Game analysis
	playerAnalysis  map[string]*PlayerAnalysis
	marketTrends    *MarketTrends
	phaseStats      map[protocol.GamePhase]*PhaseStatistics
}

// GameStateSnapshot represents a point-in-time game state
type GameStateSnapshot struct {
	State     *protocol.GameStatePayload
	Timestamp time.Time
	Phase     protocol.GamePhase
	Round     int
	Turn      string
}

// PlayerAnalysis tracks opponent behavior patterns
type PlayerAnalysis struct {
	PlayerID        string
	TotalBids       int
	AverageBid      float64
	WinRate         float64
	ResourceUsage   map[string]int
	ExpansionRate   float64
	LastAction      time.Time
	Strategy        string // Inferred strategy type
}

// MarketTrends tracks resource market patterns
type MarketTrends struct {
	ResourcePrices  map[string][]int
	DemandPatterns  map[string]float64
	SupplyLevels    map[string]int
	PriceDirection  map[string]string // "up", "down", "stable"
}

// PhaseStatistics tracks phase-specific metrics
type PhaseStatistics struct {
	AverageDuration time.Duration
	ActionsPerPhase int
	CommonPatterns  []string
}

// NewGameStateTracker creates a new game state tracker
func NewGameStateTracker(playerID string, logger *logger.ColoredLogger) *GameStateTracker {
	return &GameStateTracker{
		playerID:       playerID,
		logger:         logger,
		history:        make([]GameStateSnapshot, 0),
		maxHistory:     100,
		playerAnalysis: make(map[string]*PlayerAnalysis),
		marketTrends:   &MarketTrends{
			ResourcePrices: make(map[string][]int),
			DemandPatterns: make(map[string]float64),
			SupplyLevels:   make(map[string]int),
			PriceDirection: make(map[string]string),
		},
		phaseStats: make(map[protocol.GamePhase]*PhaseStatistics),
	}
}

// UpdateState updates the tracked game state
func (t *GameStateTracker) UpdateState(state *protocol.GameStatePayload) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Store previous state in history
	if t.currentState != nil {
		snapshot := GameStateSnapshot{
			State:     t.currentState,
			Timestamp: time.Now(),
			Phase:     t.currentState.CurrentPhase,
			Round:     t.currentState.CurrentRound,
			Turn:      t.currentState.CurrentTurn,
		}
		
		t.history = append(t.history, snapshot)
		if len(t.history) > t.maxHistory {
			t.history = t.history[1:]
		}
	}
	
	t.currentState = state
	
	// Update analysis
	t.updatePlayerAnalysis(state)
	t.updateMarketTrends(state)
	t.updatePhaseStatistics(state)
}

// GetState returns the current game state
func (t *GameStateTracker) GetState() *protocol.GameStatePayload {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentState
}

// GetPlayerPosition returns the current player's position in turn order
func (t *GameStateTracker) GetPlayerPosition() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil {
		return -1
	}
	
	for i, pid := range t.currentState.TurnOrder {
		if pid == t.playerID {
			return i
		}
	}
	return -1
}

// GetOpponentAnalysis returns analysis of a specific opponent
func (t *GameStateTracker) GetOpponentAnalysis(opponentID string) *PlayerAnalysis {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.playerAnalysis[opponentID]
}

// GetMarketTrends returns current market analysis
func (t *GameStateTracker) GetMarketTrends() *MarketTrends {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.marketTrends
}

// IsMyTurn checks if it's the current player's turn
func (t *GameStateTracker) IsMyTurn() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil {
		return false
	}
	
	return t.currentState.CurrentTurn == t.playerID
}

// GetAvailableActions returns valid actions for current phase
func (t *GameStateTracker) GetAvailableActions() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil || !t.IsMyTurn() {
		return []string{}
	}
	
	switch t.currentState.CurrentPhase {
	case protocol.PhaseAuction:
		return []string{"bid_plant", "pass"}
	case protocol.PhaseBuyResources:
		return []string{"buy_resources", "pass"}
	case protocol.PhaseBuildCities:
		return []string{"build_city", "pass"}
	case protocol.PhaseBureaucracy:
		return []string{"power_cities", "pass"}
	default:
		return []string{"end_turn"}
	}
}

// GetPlayerResources returns the current player's resources
func (t *GameStateTracker) GetPlayerResources() map[string]int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil {
		return make(map[string]int)
	}
	
	if player, ok := t.currentState.Players[t.playerID]; ok {
		return player.Resources
	}
	
	return make(map[string]int)
}

// GetPlayerMoney returns the current player's money
func (t *GameStateTracker) GetPlayerMoney() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil {
		return 0
	}
	
	if player, ok := t.currentState.Players[t.playerID]; ok {
		return player.Money
	}
	
	return 0
}

// GetPlayerCities returns the current player's cities
func (t *GameStateTracker) GetPlayerCities() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if t.currentState == nil {
		return []string{}
	}
	
	if player, ok := t.currentState.Players[t.playerID]; ok {
		return player.Cities
	}
	
	return []string{}
}

// Private helper methods

func (t *GameStateTracker) updatePlayerAnalysis(state *protocol.GameStatePayload) {
	for pid, player := range state.Players {
		if pid == t.playerID {
			continue // Skip self
		}
		
		analysis, exists := t.playerAnalysis[pid]
		if !exists {
			analysis = &PlayerAnalysis{
				PlayerID:      pid,
				ResourceUsage: make(map[string]int),
			}
			t.playerAnalysis[pid] = analysis
		}
		
		// Update resource usage
		for rType, count := range player.Resources {
			analysis.ResourceUsage[rType] += count
		}
		
		// Calculate expansion rate
		if len(t.history) > 0 {
			prevCities := 0
			for _, snapshot := range t.history {
				if prevPlayer, ok := snapshot.State.Players[pid]; ok {
					prevCities = len(prevPlayer.Cities)
					break
				}
			}
			analysis.ExpansionRate = float64(len(player.Cities)-prevCities) / float64(state.CurrentRound)
		}
		
		// Infer strategy based on behavior
		analysis.Strategy = t.inferStrategy(analysis)
		analysis.LastAction = time.Now()
	}
}

func (t *GameStateTracker) updateMarketTrends(state *protocol.GameStatePayload) {
	// Track resource prices over time
	for rType, supply := range state.Market.Resources {
		prices := t.marketTrends.ResourcePrices[rType]
		if prices == nil {
			prices = make([]int, 0)
		}
		
		// Simplified price calculation based on supply
		currentPrice := t.calculateResourcePrice(rType, supply)
		prices = append(prices, currentPrice)
		
		// Keep last 20 price points
		if len(prices) > 20 {
			prices = prices[len(prices)-20:]
		}
		
		t.marketTrends.ResourcePrices[rType] = prices
		t.marketTrends.SupplyLevels[rType] = len(supply)
		
		// Determine price direction
		if len(prices) >= 3 {
			recent := prices[len(prices)-3:]
			if recent[2] > recent[0] {
				t.marketTrends.PriceDirection[rType] = "up"
			} else if recent[2] < recent[0] {
				t.marketTrends.PriceDirection[rType] = "down"
			} else {
				t.marketTrends.PriceDirection[rType] = "stable"
			}
		}
	}
}

func (t *GameStateTracker) updatePhaseStatistics(state *protocol.GameStatePayload) {
	phase := state.CurrentPhase
	stats, exists := t.phaseStats[phase]
	if !exists {
		stats = &PhaseStatistics{
			CommonPatterns: make([]string, 0),
		}
		t.phaseStats[phase] = stats
	}
	
	// Update statistics
	stats.ActionsPerPhase++
}

func (t *GameStateTracker) inferStrategy(analysis *PlayerAnalysis) string {
	// Simple strategy inference based on behavior
	if analysis.AverageBid > 15 && analysis.ExpansionRate > 1.5 {
		return "aggressive"
	} else if analysis.AverageBid < 5 && analysis.ExpansionRate < 0.5 {
		return "conservative"
	}
	return "balanced"
}

func (t *GameStateTracker) calculateResourcePrice(resourceType string, supply []int) int {
	// Simplified price calculation
	basePrice := map[string]int{
		"coal":    3,
		"oil":     3,
		"garbage": 7,
		"uranium": 12,
	}
	
	price := basePrice[resourceType]
	if price == 0 {
		price = 5 // Default
	}
	
	// Adjust based on scarcity
	supplyCount := len(supply)
	if supplyCount < 5 {
		price += 2
	} else if supplyCount < 10 {
		price += 1
	}
	
	return price
}