package game

import (
	"errors"
)

// Map represents the game map
type Map struct {
	Name        string
	Cities      map[string]*City
	Connections []*Connection
	Regions     map[string]*Region
}

// City represents a city on the map
type City struct {
	ID       string
	Name     string
	Region   string
	Position [2]float64
	Slots    []string // Player IDs that have built here
}

// Connection represents a connection between two cities
type Connection struct {
	CityA string
	CityB string
	Cost  int
}

// Region represents a region on the map
type Region struct {
	ID    string
	Name  string
	Color string
}

// GetCity returns a city by ID
func (m *Map) GetCity(cityID string) (*City, bool) {
	city, exists := m.Cities[cityID]
	return city, exists
}

// AddPlayerToCity adds a player to a city
func (m *Map) AddPlayerToCity(cityID, playerID string) error {
	city, exists := m.Cities[cityID]
	if !exists {
		return errors.New("city not found")
	}

	// Check if player already has a spot in this city
	for _, player := range city.Slots {
		if player == playerID {
			return errors.New("player already has a spot in this city")
		}
	}

	// Check if city is full
	if len(city.Slots) >= 3 {
		return errors.New("city is full")
	}

	city.Slots = append(city.Slots, playerID)
	return nil
}

// GetConnectedCities returns all cities connected to the given city
func (m *Map) GetConnectedCities(cityID string) map[string]int {
	connected := make(map[string]int)

	for _, conn := range m.Connections {
		if conn.CityA == cityID {
			connected[conn.CityB] = conn.Cost
		} else if conn.CityB == cityID {
			connected[conn.CityA] = conn.Cost
		}
	}

	return connected
}

// GetConnectionCost returns the cost to connect two cities
func (m *Map) GetConnectionCost(cityA, cityB string) (int, error) {
	for _, conn := range m.Connections {
		if (conn.CityA == cityA && conn.CityB == cityB) ||
			(conn.CityA == cityB && conn.CityB == cityA) {
			return conn.Cost, nil
		}
	}
	return 0, errors.New("cities are not connected")
}

// MapData contains the JSON structure of map data
type MapData struct {
	Name        string           `json:"name"`
	Regions     []RegionData     `json:"regions"`
	Cities      []CityData       `json:"cities"`
	Connections []ConnectionData `json:"connections"`
}

// RegionData contains data about a region
type RegionData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CityData contains data about a city
type CityData struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Region   string     `json:"region"`
	Position [2]float64 `json:"position"`
}

// ConnectionData contains data about a connection
type ConnectionData struct {
	CityA string `json:"city_a"`
	CityB string `json:"city_b"`
	Cost  int    `json:"cost"`
}

// LoadMap loads a map from JSON file
func LoadMap(mapName string) (*Map, error) {
	// In a real implementation, this would read from a file
	// For now, we'll return a simple hard-coded map for testing
	return createTestMap(), nil
}

// createTestMap creates a simple test map
func createTestMap() *Map {
	// Create a simple map with a few cities and connections
	m := &Map{
		Name:        "Test Map",
		Cities:      make(map[string]*City),
		Connections: []*Connection{},
		Regions:     make(map[string]*Region),
	}

	// Add regions
	m.Regions["region1"] = &Region{
		ID:    "region1",
		Name:  "Region 1",
		Color: "#FF0000",
	}
	m.Regions["region2"] = &Region{
		ID:    "region2",
		Name:  "Region 2",
		Color: "#00FF00",
	}

	// Add cities
	m.Cities["city1"] = &City{
		ID:       "city1",
		Name:     "City 1",
		Region:   "region1",
		Position: [2]float64{100, 100},
		Slots:    []string{},
	}
	m.Cities["city2"] = &City{
		ID:       "city2",
		Name:     "City 2",
		Region:   "region1",
		Position: [2]float64{200, 100},
		Slots:    []string{},
	}
	m.Cities["city3"] = &City{
		ID:       "city3",
		Name:     "City 3",
		Region:   "region2",
		Position: [2]float64{300, 200},
		Slots:    []string{},
	}
	m.Cities["city4"] = &City{
		ID:       "city4",
		Name:     "City 4",
		Region:   "region2",
		Position: [2]float64{400, 200},
		Slots:    []string{},
	}

	// Add connections
	m.Connections = append(m.Connections, &Connection{
		CityA: "city1",
		CityB: "city2",
		Cost:  10,
	})
	m.Connections = append(m.Connections, &Connection{
		CityA: "city2",
		CityB: "city3",
		Cost:  15,
	})
	m.Connections = append(m.Connections, &Connection{
		CityA: "city3",
		CityB: "city4",
		Cost:  10,
	})

	return m
}
