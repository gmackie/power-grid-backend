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
		g.StartAuctionPhase()
	case protocol.PhaseAuction:
		g.CurrentPhase = protocol.PhaseBuyResources
		g.CurrentTurn = 0 // Reset turn counter for new phase
		// Set up reverse turn order for resource buying
		g.setupReversePlayerOrder()
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

// DeterminePlayerOrder sets the turn order for the current round
func (g *Game) DeterminePlayerOrder() {
	var playerIDs []string
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	if g.CurrentRound == 1 {
		// First round: random order
		for i := len(playerIDs) - 1; i > 0; i-- {
			j := int(time.Now().UnixNano()) % (i + 1)
			playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
		}
	} else {
		// Subsequent rounds: order by cities (ascending), then by highest power plant (descending)
		// This implements the Power Grid player order rules
		type playerRank struct {
			id          string
			cities      int
			highestPlant int
		}

		var rankings []playerRank
		for _, id := range playerIDs {
			player := g.Players[id]
			highestPlant := 0
			for _, plant := range player.PowerPlants {
				if plant.Number > highestPlant {
					highestPlant = plant.Number
				}
			}
			rankings = append(rankings, playerRank{
				id:          id,
				cities:      len(player.Cities),
				highestPlant: highestPlant,
			})
		}

		// Sort by cities (ascending), then by highest power plant (descending)
		for i := 0; i < len(rankings)-1; i++ {
			for j := i + 1; j < len(rankings); j++ {
				// Primary sort: fewer cities come first
				if rankings[i].cities > rankings[j].cities {
					rankings[i], rankings[j] = rankings[j], rankings[i]
				} else if rankings[i].cities == rankings[j].cities {
					// Secondary sort: higher power plant comes first (for same city count)
					if rankings[i].highestPlant < rankings[j].highestPlant {
						rankings[i], rankings[j] = rankings[j], rankings[i]
					}
				}
			}
		}

		// Extract sorted player IDs
		playerIDs = make([]string, len(rankings))
		for i, rank := range rankings {
			playerIDs[i] = rank.id
		}
	}

	g.TurnOrder = playerIDs
}

// GetCurrentPlayerID returns the ID of the current player
func (g *Game) GetCurrentPlayerID() string {
	if len(g.TurnOrder) == 0 {
		return ""
	}
	return g.TurnOrder[g.CurrentTurn]
}

// GetCurrentPlayer returns the current player
func (g *Game) GetCurrentPlayer() *Player {
	playerID := g.GetCurrentPlayerID()
	if playerID == "" {
		return nil
	}
	return g.Players[playerID]
}

// BroadcastGameState sends the current game state to all players
func (g *Game) BroadcastGameState() {
	if len(g.sessions) == 0 {
		return
	}

	// Create game state payload
	gameState := g.GetGameStatePayload()
	
	message := protocol.NewMessage(protocol.MsgGameState, gameState)
	
	// Send to all players
	for _, session := range g.sessions {
		if session != nil {
			session.Send(message)
		}
	}
}

// BroadcastPhaseChange notifies all players of a phase change
func (g *Game) BroadcastPhaseChange() {
	phasePayload := protocol.PhaseChangePayload{
		Phase: string(g.CurrentPhase),
		Round: g.CurrentRound,
	}
	
	message := protocol.NewMessage(protocol.MsgPhaseChange, phasePayload)
	
	for _, session := range g.sessions {
		if session != nil {
			session.Send(message)
		}
	}
}

// BroadcastTurnChange notifies all players of a turn change
func (g *Game) BroadcastTurnChange() {
	turnPayload := protocol.TurnChangePayload{
		CurrentPlayerID: g.GetCurrentPlayerID(),
		Turn:           g.CurrentTurn,
	}
	
	message := protocol.NewMessage(protocol.MsgTurnChange, turnPayload)
	
	for _, session := range g.sessions {
		if session != nil {
			session.Send(message)
		}
	}
}

// GetGameStatePayload creates a complete game state payload for broadcasting
func (g *Game) GetGameStatePayload() protocol.GameStatePayload {
	// Convert players map to match protocol format
	players := make(map[string]protocol.PlayerInfo)
	for id, player := range g.Players {
		// Convert power plants
		var powerPlants []protocol.PowerPlantInfo
		for _, plant := range player.PowerPlants {
			powerPlants = append(powerPlants, protocol.PowerPlantInfo{
				ID:           plant.Number, // Using Number as ID for protocol
				Cost:         plant.Cost,
				Capacity:     plant.Capacity,
				ResourceType: plant.ResourceType,
				ResourceCost: plant.ResourceCost,
			})
		}

		players[id] = protocol.PlayerInfo{
			ID:            player.ID,
			Name:          player.Name,
			Color:         player.Color,
			Money:         player.Money,
			PowerPlants:   powerPlants,
			Cities:        player.Cities,
			Resources:     player.Resources,
			PoweredCities: player.PoweredCities,
		}
	}

	// Convert available power plants
	var availablePlants []protocol.PowerPlantInfo
	for _, plant := range g.PowerPlants {
		availablePlants = append(availablePlants, protocol.PowerPlantInfo{
			ID:           plant.Number,
			Cost:         plant.Cost,
			Capacity:     plant.Capacity,
			ResourceType: plant.ResourceType,
			ResourceCost: plant.ResourceCost,
		})
	}

	// Create map info
	mapInfo := protocol.MapInfo{
		Name:        g.Map.Name,
		Cities:      g.Map.GetCitiesInfo(),
		Connections: g.Map.GetConnectionsInfo(),
	}

	// Create market info
	marketInfo := protocol.MarketInfo{
		Resources: g.Market.GetResourcesInfo(),
	}

	return protocol.GameStatePayload{
		GameID:       g.ID,
		Name:         g.Name,
		Status:       g.Status,
		CurrentPhase: g.CurrentPhase,
		CurrentTurn:  g.GetCurrentPlayerID(),
		CurrentRound: g.CurrentRound,
		Players:      players,
		Map:          mapInfo,
		Market:       marketInfo,
		PowerPlants:  availablePlants,
		TurnOrder:    g.TurnOrder,
	}
}

// CheckGameEnd checks if the game should end
func (g *Game) CheckGameEnd() bool {
	// Game ends when a player reaches the target number of cities
	// Target varies by player count: 21 (2p), 17 (3p), 17 (4p), 15 (5p), 14 (6p)
	targetCities := map[int]int{
		2: 21,
		3: 17,
		4: 17,
		5: 15,
		6: 14,
	}

	target, exists := targetCities[len(g.Players)]
	if !exists {
		target = 17 // Default
	}

	for _, player := range g.Players {
		if len(player.Cities) >= target {
			return true
		}
	}

	return false
}

// InitializeAuctionMarket sets up the power plant market for auction
func (g *Game) InitializeAuctionMarket() {
	// This is now implemented in powerplant.go through InitializePowerPlants
	// The actual market setup is handled when plants are initialized
}

// ProcessBuyResources processes a resource buying action
func (g *Game) ProcessBuyResources(playerID string, resources map[string]int) error {
	player := g.Players[playerID]
	if player == nil {
		return errors.New("player not found")
	}

	// Calculate total cost
	totalCost := 0
	for resourceType, amount := range resources {
		cost, err := g.Market.CalculateCost(resourceType, amount)
		if err != nil {
			return err
		}
		totalCost += cost
	}

	// Check if player can afford it
	if !player.HasEnoughMoney(totalCost) {
		return errors.New("not enough money")
	}

	// Buy the resources
	for resourceType, amount := range resources {
		err := g.Market.BuyResources(resourceType, amount)
		if err != nil {
			return err
		}
		player.AddResources(resourceType, amount)
	}

	// Deduct money
	player.SpendMoney(totalCost)

	return nil
}

// ProcessBuildCity processes a city building action
func (g *Game) ProcessBuildCity(playerID string, cityID string) error {
	player := g.Players[playerID]
	if player == nil {
		return errors.New("player not found")
	}

	city, exists := g.Map.GetCity(cityID)
	if !exists {
		return errors.New("city not found")
	}

	// Check if city has space
	if len(city.Slots) >= 3 {
		return errors.New("city is full")
	}

	// Check if player already has a house in this city
	for _, slot := range city.Slots {
		if slot == playerID {
			return errors.New("player already has a house in this city")
		}
	}

	// Calculate building cost
	baseCost := 10 + (len(city.Slots) * 5) // 10, 15, 20 for 1st, 2nd, 3rd house

	// Calculate connection cost
	connectionCost := 0
	if len(player.Cities) > 0 {
		// Find cheapest connection to player's network
		cheapestConnection := 999
		for _, playerCityID := range player.Cities {
			cost, err := g.Map.GetConnectionCost(playerCityID, cityID)
			if err == nil && cost < cheapestConnection {
				cheapestConnection = cost
			}
		}
		if cheapestConnection == 999 {
			return errors.New("city not connected to player's network")
		}
		connectionCost = cheapestConnection
	}

	totalCost := baseCost + connectionCost

	// Check if player can afford it
	if !player.HasEnoughMoney(totalCost) {
		return errors.New("not enough money")
	}

	// Build the house
	err := g.Map.AddPlayerToCity(cityID, playerID)
	if err != nil {
		return err
	}

	player.AddCity(cityID)
	player.SpendMoney(totalCost)

	return nil
}

// ProcessPowerCities processes the bureaucracy phase power cities action
func (g *Game) ProcessPowerCities(playerID string, plantIDs []int) error {
	player := g.Players[playerID]
	if player == nil {
		return errors.New("player not found")
	}

	// Power the cities using the specified plants
	citiesPowered, err := player.PowerCities(plantIDs)
	if err != nil {
		return err
	}

	player.PoweredCities = citiesPowered

	// Calculate and give income based on cities powered
	income := g.calculateIncome(citiesPowered)
	player.EarnMoney(income)

	return nil
}

// calculateIncome returns income based on cities powered
func (g *Game) calculateIncome(citiesPowered int) int {
	// Standard Power Grid income table
	incomeTable := []int{
		10,  // 0 cities
		22,  // 1 city
		33,  // 2 cities
		44,  // 3 cities
		54,  // 4 cities
		64,  // 5 cities
		73,  // 6 cities
		82,  // 7 cities
		90,  // 8 cities
		98,  // 9 cities
		105, // 10 cities
		112, // 11 cities
		118, // 12 cities
		124, // 13 cities
		129, // 14 cities
		134, // 15 cities
		138, // 16 cities
		142, // 17 cities
		145, // 18 cities
		148, // 19 cities
		150, // 20+ cities
	}

	if citiesPowered >= len(incomeTable) {
		return incomeTable[len(incomeTable)-1]
	}
	return incomeTable[citiesPowered]
}

// setupReversePlayerOrder sets up reverse turn order for resource buying and building phases
func (g *Game) setupReversePlayerOrder() {
	// Create a reversed copy of the turn order
	reversedOrder := make([]string, len(g.TurnOrder))
	for i, playerID := range g.TurnOrder {
		reversedOrder[len(g.TurnOrder)-1-i] = playerID
	}
	g.TurnOrder = reversedOrder
}

// StartAuctionPhase initializes the auction phase
func (g *Game) StartAuctionPhase() error {
	// Reset all players for auction
	for _, player := range g.Players {
		player.ResetForNewPhase()
	}

	// Auction state will be created when first plant is nominated
	g.AuctionState = nil

	return nil
}

// NominatePlant allows the current player to nominate a plant for auction
func (g *Game) NominatePlant(playerID string, plantID int) error {
	// Check if it's this player's turn
	if g.GetCurrentPlayerID() != playerID {
		return errors.New("not your turn to nominate")
	}

	// Check if player has already bought a plant or passed
	player := g.Players[playerID]
	if player.HasPassed {
		return errors.New("player has already passed this round")
	}

	// Check if player already has 3 plants
	if len(player.PowerPlants) >= 3 {
		// Player must discard a plant first
		// For now, we'll prevent buying a 4th plant
		return errors.New("player already has 3 power plants")
	}

	// Start the auction for this plant
	err := g.StartAuction(plantID)
	if err != nil {
		return err
	}

	return nil
}

// PassOnNomination allows a player to pass on nominating a plant
func (g *Game) PassOnNomination(playerID string) error {
	// Check if it's this player's turn
	if g.GetCurrentPlayerID() != playerID {
		return errors.New("not your turn")
	}

	player := g.Players[playerID]
	player.HasPassed = true
	player.IsActive = false

	// Move to next player
	g.CurrentTurn++
	
	// Check if all players have passed or bought
	allDone := true
	for _, p := range g.Players {
		if !p.HasPassed && len(p.PowerPlants) < 3 {
			// This player could still buy
			allDone = false
			break
		}
	}

	if allDone || g.CurrentTurn >= len(g.TurnOrder) {
		// Auction phase is complete
		g.AdvancePhase()
	}

	return nil
}
