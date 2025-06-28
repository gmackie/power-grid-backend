package game

import (
	"errors"
)

// Player represents a player in the game
type Player struct {
	ID            string
	Name          string
	Color         string
	Money         int
	PowerPlants   []*PowerPlant
	Resources     map[string]int
	Cities        []string
	PoweredCities int
	IsActive      bool
	CurrentBid    int
	HasPassed     bool
}

// NewPlayer creates a new player
func NewPlayer(id, name, color string) *Player {
	return &Player{
		ID:            id,
		Name:          name,
		Color:         color,
		Money:         50, // Starting money
		PowerPlants:   []*PowerPlant{},
		Resources:     make(map[string]int),
		Cities:        []string{},
		PoweredCities: 0,
		IsActive:      true,
		CurrentBid:    0,
		HasPassed:     false,
	}
}

// AddCity adds a city to the player's network
func (p *Player) AddCity(cityID string) {
	p.Cities = append(p.Cities, cityID)
}

// AddPowerPlant adds a power plant to the player's collection
func (p *Player) AddPowerPlant(plant *PowerPlant) {
	p.PowerPlants = append(p.PowerPlants, plant)

	// Sort power plants by cost (lowest to highest)
	for i := 0; i < len(p.PowerPlants)-1; i++ {
		for j := i + 1; j < len(p.PowerPlants); j++ {
			if p.PowerPlants[i].Cost > p.PowerPlants[j].Cost {
				p.PowerPlants[i], p.PowerPlants[j] = p.PowerPlants[j], p.PowerPlants[i]
			}
		}
	}

	// If player has more than 3 power plants, they must discard one
	if len(p.PowerPlants) > 3 {
		// Discard logic would go here - for now we'll just remove the lowest value
		p.RemovePowerPlant(p.PowerPlants[0].ID)
	}
}

// RemovePowerPlant removes a power plant from the player's collection
func (p *Player) RemovePowerPlant(plantID int) *PowerPlant {
	for i, plant := range p.PowerPlants {
		if plant.ID == plantID {
			// Remove from slice
			removed := plant
			p.PowerPlants = append(p.PowerPlants[:i], p.PowerPlants[i+1:]...)
			return removed
		}
	}
	return nil
}

// GetPowerPlant gets a power plant by ID
func (p *Player) GetPowerPlant(plantID int) *PowerPlant {
	for _, plant := range p.PowerPlants {
		if plant.ID == plantID {
			return plant
		}
	}
	return nil
}

// AddResources adds resources to the player's supply
func (p *Player) AddResources(resourceType string, amount int) {
	p.Resources[resourceType] += amount
}

// RemoveResources removes resources from the player's supply
func (p *Player) RemoveResources(resourceType string, amount int) bool {
	if p.Resources[resourceType] < amount {
		return false
	}
	p.Resources[resourceType] -= amount
	return true
}

// HasEnoughMoney checks if the player has enough money
func (p *Player) HasEnoughMoney(amount int) bool {
	return p.Money >= amount
}

// SpendMoney deducts money from the player
func (p *Player) SpendMoney(amount int) bool {
	if !p.HasEnoughMoney(amount) {
		return false
	}
	p.Money -= amount
	return true
}

// EarnMoney adds money to the player
func (p *Player) EarnMoney(amount int) {
	p.Money += amount
}

// CalculatePoweredCities calculates how many cities the player can power
func (p *Player) CalculatePoweredCities() int {
	maxPowered := 0
	for _, plant := range p.PowerPlants {
		resourceNeeded := plant.ResourceCost
		availableResources := p.Resources[plant.ResourceType]

		if availableResources >= resourceNeeded {
			// Can power this plant
			maxPowered += plant.Capacity
		}
	}

	// Cannot power more cities than the player has
	if maxPowered > len(p.Cities) {
		maxPowered = len(p.Cities)
	}

	return maxPowered
}

// PowerCities powers cities and consumes resources
func (p *Player) PowerCities(plantIDs []int) (int, error) {
	// Create a copy of resources to simulate consumption
	resourcesCopy := make(map[string]int)
	for k, v := range p.Resources {
		resourcesCopy[k] = v
	}

	totalPowered := 0

	for _, plantID := range plantIDs {
		plant := p.GetPowerPlant(plantID)
		if plant == nil {
			return 0, errors.New("player does not own specified power plant")
		}

		// Check if we have enough resources to power this plant
		if resourcesCopy[plant.ResourceType] < plant.ResourceCost {
			return 0, errors.New("not enough resources to power this plant")
		}

		// Consume resources
		resourcesCopy[plant.ResourceType] -= plant.ResourceCost

		// Add to powered cities
		totalPowered += plant.Capacity
	}

	// Cannot power more cities than the player has
	if totalPowered > len(p.Cities) {
		totalPowered = len(p.Cities)
	}

	// Actually consume the resources
	for k, v := range resourcesCopy {
		p.Resources[k] = v
	}

	p.PoweredCities = totalPowered
	return totalPowered, nil
}

// CanBuildInCity checks if the player can build in a city
func (p *Player) CanBuildInCity(cityID string, network *Map) bool {
	// If player already has a house in this city, they can't build another
	for _, playerCityID := range p.Cities {
		if playerCityID == cityID {
			return false
		}
	}

	// If this is the first city, player can build anywhere
	if len(p.Cities) == 0 {
		return true
	}

	// Check if this city is connected to player's network
	connected := false
	cityConnections := network.GetConnectedCities(cityID)

	for connectedCityID := range cityConnections {
		for _, playerCityID := range p.Cities {
			if connectedCityID == playerCityID {
				connected = true
				break
			}
		}
		if connected {
			break
		}
	}

	return connected
}

// Reset resets the player's state for a new turn
func (p *Player) Reset() {
	p.CurrentBid = 0
	p.HasPassed = false
}

// ResetForNewPhase resets the player's state for a new phase
func (p *Player) ResetForNewPhase() {
	p.Reset()
	p.IsActive = true
}
