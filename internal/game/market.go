package game

import (
	"errors"
)

// ResourceType represents a type of resource
type ResourceType string

// ResourceMarket handles buying and selling of resources
type ResourceMarket struct {
	Resources map[string][]int // Resource type -> price -> count
}

// NewResourceMarket creates a new resource market
func NewResourceMarket() *ResourceMarket {
	market := &ResourceMarket{
		Resources: make(map[string][]int),
	}

	// Initialize coal pricing
	market.Resources["Coal"] = make([]int, 9)
	market.Resources["Coal"][3] = 3
	market.Resources["Coal"][4] = 3
	market.Resources["Coal"][5] = 3
	market.Resources["Coal"][6] = 3
	market.Resources["Coal"][7] = 3
	market.Resources["Coal"][8] = 3

	// Initialize oil pricing
	market.Resources["Oil"] = make([]int, 9)
	market.Resources["Oil"][3] = 3
	market.Resources["Oil"][4] = 3
	market.Resources["Oil"][5] = 3
	market.Resources["Oil"][6] = 3
	market.Resources["Oil"][7] = 3
	market.Resources["Oil"][8] = 3

	// Initialize garbage pricing
	market.Resources["Garbage"] = make([]int, 9)
	market.Resources["Garbage"][4] = 3
	market.Resources["Garbage"][5] = 3
	market.Resources["Garbage"][6] = 3
	market.Resources["Garbage"][7] = 3
	market.Resources["Garbage"][8] = 3

	// Initialize uranium pricing
	market.Resources["Uranium"] = make([]int, 17)
	market.Resources["Uranium"][10] = 1
	market.Resources["Uranium"][12] = 1
	market.Resources["Uranium"][14] = 1
	market.Resources["Uranium"][16] = 1

	return market
}

// GetCost returns the cost to buy a certain amount of resources
func (m *ResourceMarket) GetCost(resourceType string, amount int) (int, error) {
	resourcePrices, exists := m.Resources[resourceType]
	if !exists {
		return 0, errors.New("invalid resource type")
	}

	if amount <= 0 {
		return 0, errors.New("amount must be positive")
	}

	// Check if there are enough resources available
	available := 0
	for _, count := range resourcePrices {
		available += count
	}

	if available < amount {
		return 0, errors.New("not enough resources available")
	}

	// Calculate cost
	totalCost := 0
	remaining := amount

	// Start from the highest price (end of the array) and work backwards
	for price := len(resourcePrices) - 1; price >= 0 && remaining > 0; price-- {
		if resourcePrices[price] > 0 {
			toBuy := min(remaining, resourcePrices[price])
			totalCost += toBuy * price
			remaining -= toBuy
		}
	}

	return totalCost, nil
}

// BuyResources buys resources from the market
func (m *ResourceMarket) BuyResources(resourceType string, amount int) (int, error) {
	totalCost, err := m.GetCost(resourceType, amount)
	if err != nil {
		return 0, err
	}

	// Remove resources from the market
	remaining := amount
	resourcePrices := m.Resources[resourceType]

	// Start from the highest price (end of the array) and work backwards
	for price := len(resourcePrices) - 1; price >= 0 && remaining > 0; price-- {
		if resourcePrices[price] > 0 {
			toBuy := min(remaining, resourcePrices[price])
			m.Resources[resourceType][price] -= toBuy
			remaining -= toBuy
		}
	}

	return totalCost, nil
}

// ReplenishResource adds resources to the market based on the game phase
func (m *ResourceMarket) ReplenishResource(resourceType string, amount int) {
	resourcePrices := m.Resources[resourceType]

	// Add resources from lowest price to highest
	remaining := amount
	for price := 1; price < len(resourcePrices) && remaining > 0; price++ {
		// Calculate how many spots are empty at this price level
		emptySpots := 0
		if resourceType == "Coal" || resourceType == "Oil" {
			if price >= 3 {
				emptySpots = 3 - resourcePrices[price]
			}
		} else if resourceType == "Garbage" {
			if price >= 4 {
				emptySpots = 3 - resourcePrices[price]
			}
		} else if resourceType == "Uranium" {
			if price == 10 || price == 12 || price == 14 || price == 16 {
				emptySpots = 1 - resourcePrices[price]
			}
		}

		if emptySpots > 0 {
			toAdd := min(remaining, emptySpots)
			m.Resources[resourceType][price] += toAdd
			remaining -= toAdd
		}
	}
}

// ReplenishByPhase adds resources based on the number of players and game phase
func (m *ResourceMarket) ReplenishByPhase(numPlayers int, step int) {
	// Coal and Oil
	var coalAmount, oilAmount int
	if numPlayers == 2 || numPlayers == 3 {
		coalAmount = 3
		oilAmount = 2
	} else if numPlayers == 4 || numPlayers == 5 {
		coalAmount = 4
		oilAmount = 2
	} else if numPlayers == 6 {
		coalAmount = 5
		oilAmount = 3
	}

	if step >= 2 {
		coalAmount += 1
		oilAmount += 1
	}

	m.ReplenishResource("Coal", coalAmount)
	m.ReplenishResource("Oil", oilAmount)

	// Garbage
	var garbageAmount int
	if numPlayers == 2 || numPlayers == 3 {
		garbageAmount = 1
	} else if numPlayers == 4 || numPlayers == 5 {
		garbageAmount = 2
	} else if numPlayers == 6 {
		garbageAmount = 3
	}

	if step >= 2 {
		garbageAmount += 1
	}

	m.ReplenishResource("Garbage", garbageAmount)

	// Uranium
	var uraniumAmount int
	if numPlayers == 2 || numPlayers == 3 {
		uraniumAmount = 1
	} else if numPlayers >= 4 {
		uraniumAmount = 1
	}

	if step >= 2 {
		uraniumAmount += 1
	}

	m.ReplenishResource("Uranium", uraniumAmount)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
