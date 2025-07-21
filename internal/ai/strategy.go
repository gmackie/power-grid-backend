package ai

import (
	"math/rand"
	"sort"

	"powergrid/pkg/protocol"
)

// Strategy defines the interface for AI decision making
type Strategy interface {
	GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message
	GetName() string
	GetDescription() string
}

// CreateStrategy creates a strategy instance by name
func CreateStrategy(name string) Strategy {
	switch name {
	case "aggressive":
		return &AggressiveStrategy{}
	case "conservative":
		return &ConservativeStrategy{}
	case "balanced":
		return &BalancedStrategy{}
	case "random":
		return &RandomStrategy{}
	default:
		return &BalancedStrategy{} // Default fallback
	}
}

// AggressiveStrategy implements an aggressive playing style
type AggressiveStrategy struct{}

func (s *AggressiveStrategy) GetName() string {
	return "Aggressive"
}

func (s *AggressiveStrategy) GetDescription() string {
	return "Bids high, expands quickly, takes risks"
}

func (s *AggressiveStrategy) GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	switch gameState.CurrentPhase {
	case protocol.PhaseAuction:
		return s.makeAuctionMove(gameState, playerID)
	case protocol.PhaseBuyResources:
		return s.makeResourceMove(gameState, playerID)
	case protocol.PhaseBuildCities:
		return s.makeBuildingMove(gameState, playerID)
	case protocol.PhaseBureaucracy:
		return s.makeBureaucracyMove(gameState, playerID)
	default:
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
}

func (s *AggressiveStrategy) makeAuctionMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Find available power plants
	availablePlants := getAvailablePowerPlants(gameState)
	if len(availablePlants) == 0 {
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	// Aggressive: bid on the highest capacity plant we can afford
	sort.Slice(availablePlants, func(i, j int) bool {
		return availablePlants[i].Capacity > availablePlants[j].Capacity
	})
	
	for _, plant := range availablePlants {
		bidAmount := plant.Cost + 10 // Aggressive bidding
		if bidAmount <= player.Money {
			return protocol.NewMessage(protocol.MsgBidPlant, protocol.BidPlantPayload{
				PlantID: plant.ID,
				Bid:     bidAmount,
			})
		}
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *AggressiveStrategy) makeResourceMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Buy as many resources as possible for our power plants
	resources := make(map[string]int)
	budget := player.Money
	
	for _, plantInfo := range player.PowerPlants {
		if plantInfo.ResourceType != "" && plantInfo.ResourceType != "eco" {
			resourceType := plantInfo.ResourceType
			needed := plantInfo.ResourceCost
			current := player.Resources[resourceType]
			
			// Aggressive: buy double what we need
			toBuy := (needed * 2) - current
			if toBuy > 0 {
				cost := calculateResourceCost(gameState.Market, resourceType, toBuy)
				if cost <= budget {
					resources[resourceType] += toBuy
					budget -= cost
				}
			}
		}
	}
	
	if len(resources) > 0 {
		return protocol.NewMessage(protocol.MsgBuyResources, protocol.BuyResourcesPayload{
			Resources: resources,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *AggressiveStrategy) makeBuildingMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Aggressive: expand to new regions quickly
	availableCities := getAvailableCities(gameState, playerID)
	if len(availableCities) == 0 {
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	// Sort by connection cost (cheapest first for expansion)
	sort.Slice(availableCities, func(i, j int) bool {
		return getCityCost(gameState, playerID, availableCities[i]) < 
			   getCityCost(gameState, playerID, availableCities[j])
	})
	
	for _, city := range availableCities {
		cost := getCityCost(gameState, playerID, city)
		if cost <= player.Money {
			return protocol.NewMessage(protocol.MsgBuildCity, protocol.BuildCityPayload{
				CityID: city,
			})
		}
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *AggressiveStrategy) makeBureaucracyMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Power as many cities as possible
	powerPlants := make([]int, 0)
	
	for _, plant := range player.PowerPlants {
		if canPowerPlant(player, plant) {
			powerPlants = append(powerPlants, plant.ID)
		}
	}
	
	if len(powerPlants) > 0 {
		return protocol.NewMessage(protocol.MsgPowerCities, protocol.PowerCitiesPayload{
			PowerPlants: powerPlants,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

// ConservativeStrategy implements a conservative playing style
type ConservativeStrategy struct{}

func (s *ConservativeStrategy) GetName() string {
	return "Conservative"
}

func (s *ConservativeStrategy) GetDescription() string {
	return "Saves money, minimal bidding, careful expansion"
}

func (s *ConservativeStrategy) GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	switch gameState.CurrentPhase {
	case protocol.PhaseAuction:
		return s.makeAuctionMove(gameState, playerID)
	case protocol.PhaseBuyResources:
		return s.makeResourceMove(gameState, playerID)
	case protocol.PhaseBuildCities:
		return s.makeBuildingMove(gameState, playerID)
	case protocol.PhaseBureaucracy:
		return s.makeBureaucracyMove(gameState, playerID)
	default:
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
}

func (s *ConservativeStrategy) makeAuctionMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Only bid on cheap, efficient plants
	availablePlants := getAvailablePowerPlants(gameState)
	if len(availablePlants) == 0 {
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	// Conservative: sort by efficiency (capacity/cost)
	sort.Slice(availablePlants, func(i, j int) bool {
		effI := float64(availablePlants[i].Capacity) / float64(availablePlants[i].Cost)
		effJ := float64(availablePlants[j].Capacity) / float64(availablePlants[j].Cost)
		return effI > effJ
	})
	
	for _, plant := range availablePlants {
		bidAmount := plant.Cost + 1 // Minimal bidding
		if bidAmount <= player.Money/2 { // Only spend half our money
			return protocol.NewMessage(protocol.MsgBidPlant, protocol.BidPlantPayload{
				PlantID: plant.ID,
				Bid:     bidAmount,
			})
		}
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *ConservativeStrategy) makeResourceMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Buy only what we need for current power plants
	resources := make(map[string]int)
	budget := player.Money / 2 // Conservative spending
	
	for _, plantInfo := range player.PowerPlants {
		if plantInfo.ResourceType != "" && plantInfo.ResourceType != "eco" {
			resourceType := plantInfo.ResourceType
			needed := plantInfo.ResourceCost
			current := player.Resources[resourceType]
			
			toBuy := needed - current
			if toBuy > 0 {
				cost := calculateResourceCost(gameState.Market, resourceType, toBuy)
				if cost <= budget {
					resources[resourceType] += toBuy
					budget -= cost
				}
			}
		}
	}
	
	if len(resources) > 0 {
		return protocol.NewMessage(protocol.MsgBuyResources, protocol.BuyResourcesPayload{
			Resources: resources,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *ConservativeStrategy) makeBuildingMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Conservative: only build if we have plenty of money
	if player.Money < 30 {
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	availableCities := getAvailableCities(gameState, playerID)
	if len(availableCities) == 0 {
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	// Sort by cost (cheapest first)
	sort.Slice(availableCities, func(i, j int) bool {
		return getCityCost(gameState, playerID, availableCities[i]) < 
			   getCityCost(gameState, playerID, availableCities[j])
	})
	
	// Only build the cheapest city
	city := availableCities[0]
	cost := getCityCost(gameState, playerID, city)
	if cost <= player.Money/3 { // Very conservative spending
		return protocol.NewMessage(protocol.MsgBuildCity, protocol.BuildCityPayload{
			CityID: city,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *ConservativeStrategy) makeBureaucracyMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	// Power cities efficiently
	powerPlants := make([]int, 0)
	
	// Sort plants by efficiency
	plants := make([]protocol.PowerPlantInfo, len(player.PowerPlants))
	copy(plants, player.PowerPlants)
	sort.Slice(plants, func(i, j int) bool {
		return plants[i].Capacity > plants[j].Capacity
	})
	
	for _, plant := range plants {
		if canPowerPlant(player, plant) {
			powerPlants = append(powerPlants, plant.ID)
		}
	}
	
	if len(powerPlants) > 0 {
		return protocol.NewMessage(protocol.MsgPowerCities, protocol.PowerCitiesPayload{
			PowerPlants: powerPlants,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

// BalancedStrategy implements a balanced playing style
type BalancedStrategy struct{}

func (s *BalancedStrategy) GetName() string {
	return "Balanced"
}

func (s *BalancedStrategy) GetDescription() string {
	return "Balanced approach between aggressive and conservative"
}

func (s *BalancedStrategy) GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	// Implement balanced logic - mix of aggressive and conservative
	// For now, use conservative logic with slightly more aggressive bidding
	conservative := &ConservativeStrategy{}
	return conservative.GetMove(gameState, playerID)
}

// RandomStrategy implements random decision making for testing
type RandomStrategy struct{}

func (s *RandomStrategy) GetName() string {
	return "Random"
}

func (s *RandomStrategy) GetDescription() string {
	return "Makes random valid moves for testing purposes"
}

func (s *RandomStrategy) GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	switch gameState.CurrentPhase {
	case protocol.PhaseAuction:
		return s.makeRandomAuctionMove(gameState, playerID)
	case protocol.PhaseBuyResources:
		return s.makeRandomResourceMove(gameState, playerID)
	case protocol.PhaseBuildCities:
		return s.makeRandomBuildingMove(gameState, playerID)
	case protocol.PhaseBureaucracy:
		return s.makeRandomBureaucracyMove(gameState, playerID)
	default:
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
}

func (s *RandomStrategy) makeRandomAuctionMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	availablePlants := getAvailablePowerPlants(gameState)
	
	if len(availablePlants) == 0 || rand.Float32() < 0.3 { // 30% chance to skip
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	plant := availablePlants[rand.Intn(len(availablePlants))]
	bidAmount := plant.Cost + rand.Intn(10)
	
	if bidAmount <= player.Money {
		return protocol.NewMessage(protocol.MsgBidPlant, protocol.BidPlantPayload{
			PlantID: plant.ID,
			Bid:     bidAmount,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *RandomStrategy) makeRandomResourceMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	if rand.Float32() < 0.4 { // 40% chance to skip
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	// Buy random resources
	resources := make(map[string]int)
	resourceTypes := []string{"coal", "oil", "garbage", "uranium"}
	
	for _, resType := range resourceTypes {
		if rand.Float32() < 0.3 { // 30% chance for each resource
			resources[resType] = rand.Intn(3) + 1
		}
	}
	
	if len(resources) > 0 {
		return protocol.NewMessage(protocol.MsgBuyResources, protocol.BuyResourcesPayload{
			Resources: resources,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

func (s *RandomStrategy) makeRandomBuildingMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	availableCities := getAvailableCities(gameState, playerID)
	
	if len(availableCities) == 0 || rand.Float32() < 0.4 { // 40% chance to skip
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	city := availableCities[rand.Intn(len(availableCities))]
	return protocol.NewMessage(protocol.MsgBuildCity, protocol.BuildCityPayload{
		CityID: city,
	})
}

func (s *RandomStrategy) makeRandomBureaucracyMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
	player := gameState.Players[playerID]
	
	if rand.Float32() < 0.3 { // 30% chance to skip
		return protocol.NewMessage(protocol.MsgEndTurn, nil)
	}
	
	powerPlants := make([]int, 0)
	for _, plant := range player.PowerPlants {
		if rand.Float32() < 0.7 { // 70% chance to use each plant
			powerPlants = append(powerPlants, plant.ID)
		}
	}
	
	if len(powerPlants) > 0 {
		return protocol.NewMessage(protocol.MsgPowerCities, protocol.PowerCitiesPayload{
			PowerPlants: powerPlants,
		})
	}
	
	return protocol.NewMessage(protocol.MsgEndTurn, nil)
}

// Helper functions

func getAvailablePowerPlants(gameState *protocol.GameStatePayload) []protocol.PowerPlantInfo {
	// Return available power plants from game state
	return gameState.PowerPlants
}

func getAvailableCities(gameState *protocol.GameStatePayload, playerID string) []string {
	// Return cities that the player can build in
	availableCities := make([]string, 0)
	
	for cityID, city := range gameState.Map.Cities {
		if len(city.Slots) < 3 { // Assuming max 3 players per city
			canBuild := true
			for _, slot := range city.Slots {
				if slot == playerID {
					canBuild = false
					break
				}
			}
			if canBuild {
				availableCities = append(availableCities, cityID)
			}
		}
	}
	
	return availableCities
}

func getCityCost(gameState *protocol.GameStatePayload, playerID string, cityID string) int {
	// Calculate cost to build in this city
	// This would include connection costs, city costs, etc.
	// For now, return a base cost
	baseCost := 10
	
	player := gameState.Players[playerID]
	if len(player.Cities) == 0 {
		return baseCost // First city
	}
	
	// Add connection cost (simplified)
	return baseCost + 5
}

func calculateResourceCost(market protocol.MarketInfo, resourceType string, amount int) int {
	// Calculate cost to buy resources from market
	// This is a simplified calculation
	return amount * 3 // Base price
}

func canPowerPlant(player protocol.PlayerInfo, plant protocol.PowerPlantInfo) bool {
	// Check if player has enough resources to power this plant
	if plant.ResourceType == "" || plant.ResourceType == "eco" {
		return true // Eco plants don't need resources
	}
	
	available := player.Resources[plant.ResourceType]
	return available >= plant.ResourceCost
}