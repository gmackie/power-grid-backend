package game

import (
	"errors"
	"math/rand"
	"time"
)

// PowerPlant represents a power plant in the game
type PowerPlant struct {
	ID           int
	Cost         int
	Capacity     int
	ResourceType string
	ResourceCost int
	InMarket     bool
}

// NewPowerPlant creates a new power plant
func NewPowerPlant(id, cost, capacity int, resourceType string, resourceCost int) *PowerPlant {
	return &PowerPlant{
		ID:           id,
		Cost:         cost,
		Capacity:     capacity,
		ResourceType: resourceType,
		ResourceCost: resourceCost,
		InMarket:     false,
	}
}

// PowerPlantMarket represents the power plant market
type PowerPlantMarket struct {
	Current []*PowerPlant
	Future  []*PowerPlant
	Deck    []*PowerPlant
}

// NewPowerPlantMarket creates a new power plant market
func NewPowerPlantMarket(plants []*PowerPlant) *PowerPlantMarket {
	// Sort plants by cost
	sortPowerPlants(plants)

	market := &PowerPlantMarket{
		Current: make([]*PowerPlant, 0, 4),
		Future:  make([]*PowerPlant, 0, 4),
		Deck:    make([]*PowerPlant, 0),
	}

	// Set up initial market
	for i, plant := range plants {
		if i < 4 {
			plant.InMarket = true
			market.Current = append(market.Current, plant)
		} else if i < 8 {
			plant.InMarket = true
			market.Future = append(market.Future, plant)
		} else {
			market.Deck = append(market.Deck, plant)
		}
	}

	return market
}

// RemovePlant removes a plant from the market and replaces it
func (m *PowerPlantMarket) RemovePlant(plantID int) (*PowerPlant, error) {
	// Find the plant in the current market
	var plant *PowerPlant
	var index int
	for i, p := range m.Current {
		if p.ID == plantID {
			plant = p
			index = i
			break
		}
	}

	if plant == nil {
		return nil, errors.New("plant not found in current market")
	}

	// Remove from current market
	m.Current = append(m.Current[:index], m.Current[index+1:]...)

	// Move a plant from future to current
	if len(m.Future) > 0 {
		m.Current = append(m.Current, m.Future[0])
		m.Future = m.Future[1:]
	}

	// Add a new plant to future if deck is not empty
	if len(m.Deck) > 0 {
		nextPlant := m.Deck[0]
		m.Deck = m.Deck[1:]
		nextPlant.InMarket = true
		m.Future = append(m.Future, nextPlant)
	}

	// Make sure current and future are sorted
	sortPowerPlants(m.Current)
	sortPowerPlants(m.Future)

	// Mark the removed plant as not in the market
	plant.InMarket = false
	return plant, nil
}

// InitializePowerPlants creates the initial set of power plants
func InitializePowerPlants() []*PowerPlant {
	plants := []*PowerPlant{
		NewPowerPlant(3, 3, 1, "Oil", 2),
		NewPowerPlant(4, 4, 1, "Coal", 2),
		NewPowerPlant(5, 5, 1, "Hybrid", 2), // Can use either coal or oil
		NewPowerPlant(6, 6, 1, "Garbage", 1),
		NewPowerPlant(7, 7, 2, "Oil", 3),
		NewPowerPlant(8, 8, 2, "Coal", 3),
		NewPowerPlant(9, 9, 1, "Oil", 1),
		NewPowerPlant(10, 10, 2, "Coal", 2),
		NewPowerPlant(11, 11, 2, "Uranium", 1),
		NewPowerPlant(12, 12, 2, "Hybrid", 2), // Can use either coal or oil
		NewPowerPlant(13, 13, 1, "Wind", 0),   // Renewable - no resource cost
		NewPowerPlant(14, 14, 2, "Garbage", 2),
		NewPowerPlant(15, 15, 3, "Coal", 2),
		NewPowerPlant(16, 16, 3, "Oil", 2),
		NewPowerPlant(17, 17, 2, "Uranium", 1),
		NewPowerPlant(18, 18, 2, "Wind", 0), // Renewable - no resource cost
		NewPowerPlant(19, 19, 3, "Garbage", 2),
		NewPowerPlant(20, 20, 5, "Coal", 3),
		NewPowerPlant(21, 21, 4, "Hybrid", 2), // Can use either coal or oil
		NewPowerPlant(22, 22, 2, "Wind", 0),   // Renewable - no resource cost
		NewPowerPlant(23, 23, 3, "Uranium", 1),
		NewPowerPlant(24, 24, 4, "Garbage", 2),
		NewPowerPlant(25, 25, 5, "Coal", 2),
		NewPowerPlant(26, 26, 5, "Oil", 2),
		NewPowerPlant(27, 27, 3, "Wind", 0), // Renewable - no resource cost
		NewPowerPlant(28, 28, 4, "Uranium", 1),
		NewPowerPlant(29, 29, 4, "Hybrid", 1), // Can use either coal or oil
		NewPowerPlant(30, 30, 6, "Garbage", 3),
		NewPowerPlant(31, 31, 6, "Coal", 3),
		NewPowerPlant(32, 32, 6, "Oil", 3),
		NewPowerPlant(33, 33, 4, "Wind", 0), // Renewable - no resource cost
		NewPowerPlant(34, 34, 5, "Uranium", 1),
		NewPowerPlant(35, 35, 5, "Oil", 1),
		NewPowerPlant(36, 36, 7, "Coal", 3),
		NewPowerPlant(37, 37, 4, "Wind", 0), // Renewable - no resource cost
		NewPowerPlant(38, 38, 7, "Garbage", 3),
		NewPowerPlant(39, 39, 6, "Uranium", 1),
		NewPowerPlant(40, 40, 6, "Oil", 2),
		NewPowerPlant(42, 42, 6, "Coal", 2),
		NewPowerPlant(44, 44, 5, "Wind", 0),   // Renewable - no resource cost
		NewPowerPlant(46, 46, 7, "Hybrid", 3), // Can use either coal or oil
		NewPowerPlant(50, 50, 6, "Wind", 0),   // Renewable - no resource cost
	}

	// Shuffle plants (except the first 8, which are placed in order)
	rand.Seed(time.Now().UnixNano())
	deck := plants[8:]
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	// Recombine
	plants = append(plants[:8], deck...)

	return plants
}

// Helper function to sort power plants by cost
func sortPowerPlants(plants []*PowerPlant) {
	for i := 0; i < len(plants)-1; i++ {
		for j := i + 1; j < len(plants); j++ {
			if plants[i].Cost > plants[j].Cost {
				plants[i], plants[j] = plants[j], plants[i]
			}
		}
	}
}
