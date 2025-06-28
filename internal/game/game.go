package game

import (
	"errors"
	"sync"
	"time"

	"powergrid/internal/network"
	"powergrid/pkg/protocol"
)

// Game represents the main game state
type Game struct {
	ID           string
	Name         string
	CurrentPhase protocol.GamePhase
	CurrentTurn  int
	CurrentRound int
	Map          *Map
	Players      map[string]*Player
	Market       *ResourceMarket
	PowerPlants  []*PowerPlant
	AuctionState *AuctionState
	TurnOrder    []string
	Status       protocol.GameStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time

	mutex    sync.RWMutex
	sessions map[string]*network.Session
}

// NewGame creates a new game instance
func NewGame(id, name string, mapName string) (*Game, error) {
	game := &Game{
		ID:           id,
		Name:         name,
		CurrentPhase: protocol.PhasePlayerOrder,
		CurrentTurn:  0,
		CurrentRound: 1,
		Players:      make(map[string]*Player),
		TurnOrder:    []string{},
		Status:       protocol.StatusLobby,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		sessions:     make(map[string]*network.Session),
	}

	// Load map
	var err error
	game.Map, err = LoadMap(mapName)
	if err != nil {
		return nil, err
	}

	// Initialize resource market
	game.Market = NewResourceMarket()

	// Initialize power plant deck
	game.PowerPlants = InitializePowerPlants()

	return game, nil
}

// AddPlayer adds a new player to the game
func (g *Game) AddPlayer(id, name, color string, session *network.Session) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Check if game is full or already started
	if len(g.Players) >= 6 {
		return errors.New("game is full")
	}

	if g.Status != protocol.StatusLobby {
		return errors.New("game has already started")
	}

	// Check if color is already taken
	for _, p := range g.Players {
		if p.Color == color {
			return errors.New("color already taken")
		}
	}

	// Create new player
	player := NewPlayer(id, name, color)
	g.Players[id] = player
	g.sessions[id] = session

	g.UpdatedAt = time.Now()
	return nil
}

// RemovePlayer removes a player from the game
func (g *Game) RemovePlayer(id string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.Status != protocol.StatusLobby {
		return errors.New("cannot remove player after game has started")
	}

	if _, exists := g.Players[id]; !exists {
		return errors.New("player not in game")
	}

	delete(g.Players, id)
	delete(g.sessions, id)

	g.UpdatedAt = time.Now()
	return nil
}

// Start starts the game
func (g *Game) Start() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.Status != protocol.StatusLobby {
		return errors.New("game has already started")
	}

	if len(g.Players) < 2 {
		return errors.New("need at least 2 players to start")
	}

	// Set initial game state
	g.Status = protocol.StatusPlaying
	g.CurrentPhase = protocol.PhasePlayerOrder
	g.CurrentRound = 1
	g.CurrentTurn = 0

	// Determine initial player order (random for first round)
	g.DeterminePlayerOrder()

	// Initialize auction market
	g.InitializeAuctionMarket()

	g.UpdatedAt = time.Now()

	// Notify players that game has started
	g.BroadcastGameState()

	return nil
}

// ProcessAction processes a player action
func (g *Game) ProcessAction(playerID string, action protocol.MessageType, payload interface{}) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.Status != protocol.StatusPlaying {
		return errors.New("game is not in playing state")
	}

	// Verify it's the player's turn for actions that require it
	currentPlayerID := g.GetCurrentPlayerID()
	if action != protocol.MsgBidPlant && playerID != currentPlayerID {
		return errors.New("not your turn")
	}

	var err error

	// Process the action based on type and current phase
	switch action {
	case protocol.MsgBidPlant:
		if g.CurrentPhase != protocol.PhaseAuction {
			return errors.New("not in auction phase")
		}
		bidPayload, ok := payload.(protocol.BidPlantPayload)
		if !ok {
			return errors.New("invalid payload for bid plant")
		}
		err = g.ProcessBid(playerID, bidPayload.PlantID, bidPayload.Bid)

	case protocol.MsgBuyResources:
		if g.CurrentPhase != protocol.PhaseBuyResources {
			return errors.New("not in buy resources phase")
		}
		resourcesPayload, ok := payload.(protocol.BuyResourcesPayload)
		if !ok {
			return errors.New("invalid payload for buy resources")
		}
		err = g.ProcessResourcePurchase(playerID, resourcesPayload.Resources)

	case protocol.MsgBuildCity:
		if g.CurrentPhase != protocol.PhaseBuildCities {
			return errors.New("not in build cities phase")
		}
		buildPayload, ok := payload.(protocol.BuildCityPayload)
		if !ok {
			return errors.New("invalid payload for build city")
		}
		err = g.ProcessCityBuild(playerID, buildPayload.CityID)

	case protocol.MsgPowerCities:
		if g.CurrentPhase != protocol.PhaseBureaucracy {
			return errors.New("not in bureaucracy phase")
		}
		powerPayload, ok := payload.(protocol.PowerCitiesPayload)
		if !ok {
			return errors.New("invalid payload for power cities")
		}
		err = g.ProcessPowerCities(playerID, powerPayload.PowerPlants)

	case protocol.MsgEndTurn:
		err = g.EndPlayerTurn(playerID)

	default:
		err = errors.New("unknown action type")
	}

	if err != nil {
		return err
	}

	g.UpdatedAt = time.Now()
	g.BroadcastGameState()

	return nil
}

// GetCurrentPlayerID returns the ID of the current player
func (g *Game) GetCurrentPlayerID() string {
	if len(g.TurnOrder) == 0 {
		return ""
	}
	return g.TurnOrder[g.CurrentTurn]
}

// AdvancePhase advances to the next game phase
func (g *Game) AdvancePhase() {
	switch g.CurrentPhase {
	case protocol.PhasePlayerOrder:
		g.CurrentPhase = protocol.PhaseAuction
		g.InitializeAuctionMarket()
	case protocol.PhaseAuction:
		g.CurrentPhase = protocol.PhaseBuyResources
		g.CurrentTurn = 0 // Reset turn counter for new phase
	case protocol.PhaseBuyResources:
		g.CurrentPhase = protocol.PhaseBuildCities
		g.CurrentTurn = 0
	case protocol.PhaseBuildCities:
		g.CurrentPhase = protocol.PhaseBureaucracy
		g.CurrentTurn = 0
	case protocol.PhaseBureaucracy:
		// Check for game end condition
		if g.CheckGameEnd() {
			g.CurrentPhase = protocol.PhaseGameEnd
			g.Status = protocol.StatusFinished
		} else {
			// Start a new round
			g.CurrentRound++
			g.CurrentPhase = protocol.PhasePlayerOrder
			g.DeterminePlayerOrder()
		}
	}

	// Broadcast phase change
	g.BroadcastPhaseChange()
}

// EndPlayerTurn ends the current player's turn
func (g *Game) EndPlayerTurn(playerID string) error {
	// Verify it's this player's turn
	if g.GetCurrentPlayerID() != playerID {
		return errors.New("not your turn")
	}

	// Advance to next player
	g.CurrentTurn = (g.CurrentTurn + 1) % len(g.TurnOrder)

	// If we've completed a full round of turns, advance to the next phase
	if g.CurrentTurn == 0 {
		g.AdvancePhase()
	}

	// Broadcast turn change
	g.BroadcastTurnChange()

	return nil
}
