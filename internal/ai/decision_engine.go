package ai

import (
	"fmt"
	"math"
	"sort"
	"time"

	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// DecisionEngine provides sophisticated decision-making logic
type DecisionEngine struct {
	tracker      *GameStateTracker
	logger       *logger.ColoredLogger
	strategy     Strategy
	logDecisions bool
	decisionChan chan<- DecisionLog
}

// DecisionLog represents a logged AI decision
type DecisionLog struct {
	Timestamp    time.Time              `json:"timestamp"`
	PlayerID     string                 `json:"playerId"`
	PlayerName   string                 `json:"playerName"`
	Phase        string                 `json:"phase"`
	DecisionType string                 `json:"decisionType"`
	Decision     interface{}            `json:"decision"`
	Reasoning    string                 `json:"reasoning"`
	Factors      map[string]interface{} `json:"factors"`
}

// NewDecisionEngine creates a new decision engine
func NewDecisionEngine(tracker *GameStateTracker, strategy Strategy, logger *logger.ColoredLogger) *DecisionEngine {
	return &DecisionEngine{
		tracker:  tracker,
		strategy: strategy,
		logger:   logger,
	}
}

// SetDecisionLogging enables decision logging
func (e *DecisionEngine) SetDecisionLogging(enabled bool, decisionChan chan<- DecisionLog) {
	e.logDecisions = enabled
	e.decisionChan = decisionChan
}

// logDecision logs a decision if logging is enabled
func (e *DecisionEngine) logDecision(decisionType string, decision interface{}, reasoning string, factors map[string]interface{}) {
	if !e.logDecisions || e.decisionChan == nil {
		return
	}
	
	state := e.tracker.GetState()
	if state == nil {
		return
	}
	
	player := state.Players[e.tracker.playerID]
	
	log := DecisionLog{
		Timestamp:    time.Now(),
		PlayerID:     e.tracker.playerID,
		PlayerName:   player.Name,
		Phase:        string(state.CurrentPhase),
		DecisionType: decisionType,
		Decision:     decision,
		Reasoning:    reasoning,
		Factors:      factors,
	}
	
	// Non-blocking send
	select {
	case e.decisionChan <- log:
	default:
		e.logger.Warn("Decision log channel full, dropping log")
	}
}

// MakeDecision returns the best move for the current game state
func (e *DecisionEngine) MakeDecision() *protocol.Message {
	state := e.tracker.GetState()
	if state == nil || !e.tracker.IsMyTurn() {
		return nil
	}

	e.logger.Debug("Making decision for phase: %s", state.CurrentPhase)

	switch state.CurrentPhase {
	case protocol.PhaseAuction:
		return e.makeAuctionDecision(state)
	case protocol.PhaseBuyResources:
		return e.makeResourceDecision(state)
	case protocol.PhaseBuildCities:
		return e.makeBuildingDecision(state)
	case protocol.PhaseBureaucracy:
		return e.makeBureaucracyDecision(state)
	default:
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
}

// Auction Phase Decision Making

func (e *DecisionEngine) makeAuctionDecision(state *protocol.GameStatePayload) *protocol.Message {
	playerID := e.tracker.playerID
	player := state.Players[playerID]
	
	// Get available plants for auction
	availablePlants := e.getAuctionablePlants(state)
	if len(availablePlants) == 0 {
		e.logger.Debug("No plants available for auction")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	// Evaluate each plant
	plantScores := make(map[int]float64)
	for _, plant := range availablePlants {
		score := e.evaluatePowerPlant(plant, player, state)
		plantScores[plant.ID] = score
		e.logger.Debug("Plant %d score: %.2f", plant.ID, score)
	}

	// Select best plant
	bestPlant := e.selectBestPlant(availablePlants, plantScores, player.Money)
	if bestPlant == nil {
		e.logger.Debug("No suitable plant found")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	// Calculate bid amount based on strategy
	bidAmount := e.calculateBidAmount(bestPlant, plantScores[bestPlant.ID], player.Money)
	
	e.logger.Info("Bidding %d on plant %d (capacity: %d)", bidAmount, bestPlant.ID, bestPlant.Capacity)
	
	// Log the decision
	e.logDecision("auction_bid", 
		map[string]interface{}{
			"plantId": bestPlant.ID,
			"bid": bidAmount,
			"capacity": bestPlant.Capacity,
		},
		fmt.Sprintf("Bidding %d on plant %d because it has capacity %d and efficiency score %.2f",
			bidAmount, bestPlant.ID, bestPlant.Capacity, plantScores[bestPlant.ID]),
		map[string]interface{}{
			"plantScore": plantScores[bestPlant.ID],
			"playerMoney": player.Money,
			"plantCost": bestPlant.Cost,
			"plantCapacity": bestPlant.Capacity,
			"plantResourceType": bestPlant.ResourceType,
		},
	)
	
	return protocol.NewMessage(protocol.MsgBidPlant, protocol.BidPlantPayload{
		PlantID: bestPlant.ID,
		Bid:     bidAmount,
	})
}

func (e *DecisionEngine) evaluatePowerPlant(plant protocol.PowerPlantInfo, player protocol.PlayerInfo, state *protocol.GameStatePayload) float64 {
	score := 0.0

	// Base score from capacity
	score += float64(plant.Capacity) * 10

	// Efficiency score (capacity per cost)
	efficiency := float64(plant.Capacity) / float64(plant.Cost)
	score += efficiency * 20

	// Resource availability score
	if plant.ResourceType != "" && plant.ResourceType != "eco" {
		resourceAvailability := e.evaluateResourceAvailability(plant.ResourceType, state)
		score += resourceAvailability * 15
	} else if plant.ResourceType == "eco" {
		score += 30 // Bonus for eco plants
	}

	// Synergy with existing plants
	synergy := e.evaluatePlantSynergy(plant, player.PowerPlants)
	score += synergy * 10

	// Phase of game consideration
	gameProgress := float64(state.CurrentRound) / 10.0
	if gameProgress < 0.3 {
		// Early game: prefer cheaper plants
		score -= float64(plant.Cost) * 0.5
	} else if gameProgress > 0.7 {
		// Late game: prefer high capacity
		score += float64(plant.Capacity) * 15
	}

	// Competition factor
	position := e.tracker.GetPlayerPosition()
	if position > len(state.Players)/2 {
		// If behind, be more aggressive
		score += 10
	}

	return score
}

func (e *DecisionEngine) evaluateResourceAvailability(resourceType string, state *protocol.GameStatePayload) float64 {
	trends := e.tracker.GetMarketTrends()
	
	// Check current availability
	supply := trends.SupplyLevels[resourceType]
	maxSupply := map[string]int{
		"coal":    24,
		"oil":     24,
		"garbage": 24,
		"uranium": 12,
	}[resourceType]

	availability := float64(supply) / float64(maxSupply)

	// Check price trend
	if trends.PriceDirection[resourceType] == "up" {
		availability *= 0.8 // Penalize if prices are rising
	} else if trends.PriceDirection[resourceType] == "down" {
		availability *= 1.2 // Bonus if prices are falling
	}

	return availability * 100
}

func (e *DecisionEngine) evaluatePlantSynergy(newPlant protocol.PowerPlantInfo, existingPlants []protocol.PowerPlantInfo) float64 {
	synergy := 0.0

	// Diversity bonus
	resourceTypes := make(map[string]int)
	for _, plant := range existingPlants {
		resourceTypes[plant.ResourceType]++
	}

	if resourceTypes[newPlant.ResourceType] == 0 {
		synergy += 20 // New resource type
	} else {
		synergy -= float64(resourceTypes[newPlant.ResourceType]) * 5 // Penalty for same type
	}

	return synergy
}

func (e *DecisionEngine) selectBestPlant(plants []protocol.PowerPlantInfo, scores map[int]float64, money int) *protocol.PowerPlantInfo {
	// Sort by score
	sort.Slice(plants, func(i, j int) bool {
		return scores[plants[i].ID] > scores[plants[j].ID]
	})

	// Find best affordable plant
	for _, plant := range plants {
		maxBid := e.calculateMaxBid(&plant, money)
		if maxBid >= plant.Cost {
			return &plant
		}
	}

	return nil
}

func (e *DecisionEngine) calculateBidAmount(plant *protocol.PowerPlantInfo, score float64, money int) int {
	baseBid := plant.Cost
	maxBid := e.calculateMaxBid(plant, money)

	// Adjust based on score
	scoreMultiplier := score / 100.0
	if scoreMultiplier > 1.5 {
		scoreMultiplier = 1.5
	}

	// Strategy-specific adjustments
	strategyMultiplier := 1.0
	switch e.strategy.GetName() {
	case "Aggressive":
		strategyMultiplier = 1.3
	case "Conservative":
		strategyMultiplier = 1.05
	case "Balanced":
		strategyMultiplier = 1.15
	}

	bid := int(float64(baseBid) * scoreMultiplier * strategyMultiplier)
	
	// Ensure within bounds
	if bid < baseBid {
		bid = baseBid
	}
	if bid > maxBid {
		bid = maxBid
	}

	return bid
}

func (e *DecisionEngine) calculateMaxBid(plant *protocol.PowerPlantInfo, money int) int {
	// Reserve money for other phases
	reserveRatio := 0.4
	switch e.strategy.GetName() {
	case "Aggressive":
		reserveRatio = 0.2
	case "Conservative":
		reserveRatio = 0.6
	}

	maxSpend := int(float64(money) * (1 - reserveRatio))
	return maxSpend
}

// Resource Phase Decision Making

func (e *DecisionEngine) makeResourceDecision(state *protocol.GameStatePayload) *protocol.Message {
	playerID := e.tracker.playerID
	player := state.Players[playerID]
	
	// Calculate resource needs
	resourceNeeds := e.calculateResourceNeeds(player)
	if len(resourceNeeds) == 0 {
		e.logger.Debug("No resources needed")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	// Optimize resource purchases
	purchases := e.optimizeResourcePurchases(resourceNeeds, player.Money, state)
	if len(purchases) == 0 {
		e.logger.Debug("Cannot afford resources")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	e.logger.Info("Buying resources: %v", purchases)
	
	// Log the decision
	totalCost := 0
	for resource, amount := range purchases {
		totalCost += e.calculateResourceCost(resource, amount, state)
	}
	
	e.logDecision("resource_purchase",
		purchases,
		fmt.Sprintf("Purchasing resources to power plants: %v for total cost %d", purchases, totalCost),
		map[string]interface{}{
			"resourceNeeds": resourceNeeds,
			"totalCost": totalCost,
			"playerMoney": player.Money,
			"marketTrends": e.tracker.GetMarketTrends(),
		},
	)
	
	return protocol.NewMessage(protocol.MsgBuyResources, protocol.BuyResourcesPayload{
		Resources: purchases,
	})
}

func (e *DecisionEngine) calculateResourceNeeds(player protocol.PlayerInfo) map[string]int {
	needs := make(map[string]int)

	for _, plant := range player.PowerPlants {
		if plant.ResourceType == "" || plant.ResourceType == "eco" {
			continue
		}

		// Calculate how many resources needed to power plant
		currentResources := player.Resources[plant.ResourceType]
		needed := plant.ResourceCost - currentResources

		// Strategy adjustment
		switch e.strategy.GetName() {
		case "Aggressive":
			needed = plant.ResourceCost * 2 - currentResources // Stock up
		case "Conservative":
			needed = int(math.Max(0, float64(needed))) // Only what's needed
		default:
			needed = int(float64(plant.ResourceCost)*1.5) - currentResources
		}

		if needed > 0 {
			needs[plant.ResourceType] = needed
		}
	}

	return needs
}

func (e *DecisionEngine) optimizeResourcePurchases(needs map[string]int, money int, state *protocol.GameStatePayload) map[string]int {
	purchases := make(map[string]int)
	
	// Reserve money for building phase
	reserveAmount := int(float64(money) * 0.3)
	if e.strategy.GetName() == "Aggressive" {
		reserveAmount = int(float64(money) * 0.2)
	} else if e.strategy.GetName() == "Conservative" {
		reserveAmount = int(float64(money) * 0.4)
	}
	
	availableMoney := money - reserveAmount

	// Prioritize resources by scarcity and price
	priorities := e.prioritizeResources(needs, state)
	
	for _, resource := range priorities {
		amount := needs[resource]
		cost := e.calculateResourceCost(resource, amount, state)
		
		if cost <= availableMoney {
			purchases[resource] = amount
			availableMoney -= cost
		} else {
			// Buy what we can afford
			affordableAmount := e.calculateAffordableAmount(resource, availableMoney, state)
			if affordableAmount > 0 {
				purchases[resource] = affordableAmount
				availableMoney -= e.calculateResourceCost(resource, affordableAmount, state)
			}
		}
	}

	return purchases
}

func (e *DecisionEngine) prioritizeResources(needs map[string]int, state *protocol.GameStatePayload) []string {
	type resourcePriority struct {
		resource string
		score    float64
	}

	priorities := make([]resourcePriority, 0)
	trends := e.tracker.GetMarketTrends()

	for resource, amount := range needs {
		score := float64(amount) // Base score is need

		// Adjust for price trends
		if trends.PriceDirection[resource] == "up" {
			score *= 1.5 // Higher priority if prices rising
		}

		// Adjust for scarcity
		supply := trends.SupplyLevels[resource]
		if supply < 10 {
			score *= 1.3
		}

		priorities = append(priorities, resourcePriority{resource, score})
	}

	// Sort by priority
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].score > priorities[j].score
	})

	result := make([]string, len(priorities))
	for i, p := range priorities {
		result[i] = p.resource
	}

	return result
}

// Building Phase Decision Making

func (e *DecisionEngine) makeBuildingDecision(state *protocol.GameStatePayload) *protocol.Message {
	playerID := e.tracker.playerID
	player := state.Players[playerID]
	
	// Get buildable cities
	buildableCities := e.getBuildableCities(state, player)
	if len(buildableCities) == 0 {
		e.logger.Debug("No cities available to build")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	// Evaluate each city
	cityScores := make(map[string]float64)
	for _, city := range buildableCities {
		score := e.evaluateCity(city, player, state)
		cityScores[city.ID] = score
		e.logger.Debug("City %s score: %.2f", city.Name, score)
	}

	// Select best city
	bestCity := e.selectBestCity(buildableCities, cityScores, player.Money, state)
	if bestCity == nil {
		e.logger.Debug("No suitable city found")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	connectionCost := e.calculateConnectionCost(bestCity.ID, player.Cities, state)
	e.logger.Info("Building in city: %s", bestCity.Name)
	
	// Log the decision
	e.logDecision("city_build",
		map[string]interface{}{
			"cityId": bestCity.ID,
			"cityName": bestCity.Name,
			"region": bestCity.Region,
		},
		fmt.Sprintf("Building in %s (region %s) with connection cost %d based on score %.2f",
			bestCity.Name, bestCity.Region, connectionCost, cityScores[bestCity.ID]),
		map[string]interface{}{
			"cityScore": cityScores[bestCity.ID],
			"connectionCost": connectionCost,
			"playerMoney": player.Money,
			"playerCities": len(player.Cities),
			"strategicValue": e.evaluateStrategicValue(*bestCity, state),
		},
	)
	
	return protocol.NewMessage(protocol.MsgBuildCity, protocol.BuildCityPayload{
		CityID: bestCity.ID,
	})
}

func (e *DecisionEngine) evaluateCity(city protocol.CityInfo, player protocol.PlayerInfo, state *protocol.GameStatePayload) float64 {
	score := 0.0

	// Connection cost
	connectionCost := e.calculateConnectionCost(city.ID, player.Cities, state)
	score -= float64(connectionCost) * 2

	// Strategic value
	strategicValue := e.evaluateStrategicValue(city, state)
	score += strategicValue

	// Region control
	regionControl := e.evaluateRegionControl(city.Region, player, state)
	score += regionControl * 20

	// Competition factor
	competition := float64(len(city.Slots)) * 10
	score -= competition

	return score
}

func (e *DecisionEngine) evaluateStrategicValue(city protocol.CityInfo, state *protocol.GameStatePayload) float64 {
	value := 50.0 // Base value

	// Central locations are more valuable
	connections := 0
	for _, conn := range state.Map.Connections {
		if conn.CityA == city.ID || conn.CityB == city.ID {
			connections++
		}
	}
	value += float64(connections) * 5

	return value
}

func (e *DecisionEngine) evaluateRegionControl(region string, player protocol.PlayerInfo, state *protocol.GameStatePayload) float64 {
	playerCitiesInRegion := 0
	totalCitiesInRegion := 0

	for _, city := range state.Map.Cities {
		if city.Region == region {
			totalCitiesInRegion++
			for _, slot := range city.Slots {
				if slot == player.ID {
					playerCitiesInRegion++
					break
				}
			}
		}
	}

	if totalCitiesInRegion == 0 {
		return 0
	}

	return float64(playerCitiesInRegion) / float64(totalCitiesInRegion) * 100
}

// Bureaucracy Phase Decision Making

func (e *DecisionEngine) makeBureaucracyDecision(state *protocol.GameStatePayload) *protocol.Message {
	playerID := e.tracker.playerID
	player := state.Players[playerID]
	
	// Determine which plants to use
	plantsToUse := e.selectPlantsToUse(player)
	if len(plantsToUse) == 0 {
		e.logger.Debug("No plants can be powered")
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}

	powerCapacity := e.calculatePowerCapacity(plantsToUse, player)
	e.logger.Info("Powering %d cities with plants: %v", powerCapacity, plantsToUse)
	
	// Log the decision
	plantsInfo := make([]map[string]interface{}, 0)
	for _, plantID := range plantsToUse {
		for _, plant := range player.PowerPlants {
			if plant.ID == plantID {
				plantsInfo = append(plantsInfo, map[string]interface{}{
					"id": plant.ID,
					"capacity": plant.Capacity,
					"resourceType": plant.ResourceType,
				})
				break
			}
		}
	}
	
	e.logDecision("power_cities",
		map[string]interface{}{
			"powerPlants": plantsToUse,
			"citiesPowered": powerCapacity,
		},
		fmt.Sprintf("Powering %d cities using %d power plants for maximum efficiency",
			powerCapacity, len(plantsToUse)),
		map[string]interface{}{
			"plantsUsed": plantsInfo,
			"totalCities": len(player.Cities),
			"playerResources": player.Resources,
		},
	)
	
	return protocol.NewMessage(protocol.MsgPowerCities, protocol.PowerCitiesPayload{
		PowerPlants: plantsToUse,
	})
}

func (e *DecisionEngine) selectPlantsToUse(player protocol.PlayerInfo) []int {
	// Sort plants by efficiency
	plants := make([]protocol.PowerPlantInfo, len(player.PowerPlants))
	copy(plants, player.PowerPlants)
	
	sort.Slice(plants, func(i, j int) bool {
		effI := e.calculatePlantEfficiency(plants[i], player)
		effJ := e.calculatePlantEfficiency(plants[j], player)
		return effI > effJ
	})

	selectedPlants := make([]int, 0)
	totalCapacity := 0
	cityCount := len(player.Cities)

	for _, plant := range plants {
		if e.canPowerPlant(plant, player) {
			selectedPlants = append(selectedPlants, plant.ID)
			totalCapacity += plant.Capacity
			
			if totalCapacity >= cityCount {
				break // Enough capacity
			}
		}
	}

	return selectedPlants
}

func (e *DecisionEngine) calculatePlantEfficiency(plant protocol.PowerPlantInfo, player protocol.PlayerInfo) float64 {
	if plant.ResourceType == "" || plant.ResourceType == "eco" {
		return 100.0 // Max efficiency for eco plants
	}

	// Efficiency = capacity per resource
	return float64(plant.Capacity) / float64(plant.ResourceCost)
}

func (e *DecisionEngine) canPowerPlant(plant protocol.PowerPlantInfo, player protocol.PlayerInfo) bool {
	if plant.ResourceType == "" || plant.ResourceType == "eco" {
		return true
	}

	available := player.Resources[plant.ResourceType]
	return available >= plant.ResourceCost
}

func (e *DecisionEngine) calculatePowerCapacity(plantIDs []int, player protocol.PlayerInfo) int {
	capacity := 0
	for _, id := range plantIDs {
		for _, plant := range player.PowerPlants {
			if plant.ID == id {
				capacity += plant.Capacity
				break
			}
		}
	}
	return capacity
}

// Helper methods

func (e *DecisionEngine) getAuctionablePlants(state *protocol.GameStatePayload) []protocol.PowerPlantInfo {
	// In a real game, this would return the current market plants
	// For now, return available plants
	return state.PowerPlants
}

func (e *DecisionEngine) getBuildableCities(state *protocol.GameStatePayload, player protocol.PlayerInfo) []protocol.CityInfo {
	buildable := make([]protocol.CityInfo, 0)

	for _, city := range state.Map.Cities {
		// Check if player already has this city
		hasCity := false
		for _, slot := range city.Slots {
			if slot == player.ID {
				hasCity = true
				break
			}
		}

		// Check if city has available slots
		if !hasCity && len(city.Slots) < 3 {
			buildable = append(buildable, city)
		}
	}

	return buildable
}

func (e *DecisionEngine) calculateConnectionCost(cityID string, playerCities []string, state *protocol.GameStatePayload) int {
	if len(playerCities) == 0 {
		return 10 // Base cost for first city
	}

	// Find cheapest connection
	minCost := 1000
	for _, playerCity := range playerCities {
		for _, conn := range state.Map.Connections {
			if (conn.CityA == playerCity && conn.CityB == cityID) ||
			   (conn.CityA == cityID && conn.CityB == playerCity) {
				if conn.Cost < minCost {
					minCost = conn.Cost
				}
			}
		}
	}

	return minCost + 10 // Add city cost
}

func (e *DecisionEngine) selectBestCity(cities []protocol.CityInfo, scores map[string]float64, money int, state *protocol.GameStatePayload) *protocol.CityInfo {
	// Sort by score
	sort.Slice(cities, func(i, j int) bool {
		return scores[cities[i].ID] > scores[cities[j].ID]
	})

	player := state.Players[e.tracker.playerID]
	
	// Find best affordable city
	for _, city := range cities {
		cost := e.calculateConnectionCost(city.ID, player.Cities, state)
		if cost <= money {
			return &city
		}
	}

	return nil
}

func (e *DecisionEngine) calculateResourceCost(resourceType string, amount int, state *protocol.GameStatePayload) int {
	// Simplified cost calculation
	basePrice := map[string]int{
		"coal":    3,
		"oil":     3,
		"garbage": 7,
		"uranium": 12,
	}[resourceType]

	return basePrice * amount
}

func (e *DecisionEngine) calculateAffordableAmount(resourceType string, money int, state *protocol.GameStatePayload) int {
	basePrice := map[string]int{
		"coal":    3,
		"oil":     3,
		"garbage": 7,
		"uranium": 12,
	}[resourceType]

	return money / basePrice
}