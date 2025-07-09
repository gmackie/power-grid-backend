package maps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
)

// MapData represents a complete map configuration
type MapData struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	PlayerCount PlayerCountConfig   `json:"playerCount"`
	Regions     []Region            `json:"regions"`
	Cities      []City              `json:"cities"`
	Connections []Connection        `json:"connections"`
	GameRules   GameRules           `json:"gameRules"`
}

// PlayerCountConfig defines player count constraints
type PlayerCountConfig struct {
	Min         int   `json:"min"`
	Max         int   `json:"max"`
	Recommended []int `json:"recommended"`
}

// Region represents a geographical region on the map
type Region struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// City represents a city on the map
type City struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Region string  `json:"region"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// Connection represents a connection between cities
type Connection struct {
	From string `json:"from"`
	To   string `json:"to"`
	Cost int    `json:"cost"`
}

// GameRules contains map-specific game rules
type GameRules struct {
	Step2Trigger    int                    `json:"step2Trigger"`
	Step3Trigger    int                    `json:"step3Trigger"`
	StartingMoney   int                    `json:"startingMoney"`
	ResourceSupply  map[string]int         `json:"resourceSupply"`
	ResourcePrices  map[string][]int       `json:"resourcePrices"`
	EarningsTable   []int                  `json:"earningsTable"`
	WinConditions   map[string]int         `json:"winConditions"`
}

// MapManager handles loading and managing game maps
type MapManager struct {
	maps     map[string]*MapData
	mapDir   string
	mutex    sync.RWMutex
}

// NewMapManager creates a new map manager
func NewMapManager(mapDir string) *MapManager {
	return &MapManager{
		maps:   make(map[string]*MapData),
		mapDir: mapDir,
	}
}

// LoadMaps loads all map files from the maps directory
func (mm *MapManager) LoadMaps() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Get all JSON files in maps directory
	files, err := filepath.Glob(filepath.Join(mm.mapDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to find map files: %w", err)
	}

	for _, file := range files {
		if err := mm.loadMapFile(file); err != nil {
			return fmt.Errorf("failed to load map file %s: %w", file, err)
		}
	}

	return nil
}

// loadMapFile loads a single map file
func (mm *MapManager) loadMapFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var mapData MapData
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	// Validate map data
	if err := mm.validateMap(&mapData); err != nil {
		return fmt.Errorf("invalid map data: %w", err)
	}

	mm.maps[mapData.ID] = &mapData
	return nil
}

// validateMap validates map data for consistency
func (mm *MapManager) validateMap(mapData *MapData) error {
	// Check required fields
	if mapData.ID == "" {
		return fmt.Errorf("map ID is required")
	}
	if mapData.Name == "" {
		return fmt.Errorf("map name is required")
	}
	if len(mapData.Cities) == 0 {
		return fmt.Errorf("map must have at least one city")
	}

	// Validate player count
	if mapData.PlayerCount.Min < 2 {
		return fmt.Errorf("minimum player count must be at least 2")
	}
	if mapData.PlayerCount.Max < mapData.PlayerCount.Min {
		return fmt.Errorf("maximum player count must be >= minimum")
	}

	// Create city lookup for connection validation
	cityMap := make(map[string]bool)
	for _, city := range mapData.Cities {
		if city.ID == "" {
			return fmt.Errorf("city ID is required")
		}
		if city.Name == "" {
			return fmt.Errorf("city name is required")
		}
		cityMap[city.ID] = true
	}

	// Validate connections
	for _, conn := range mapData.Connections {
		if !cityMap[conn.From] {
			return fmt.Errorf("connection references unknown city: %s", conn.From)
		}
		if !cityMap[conn.To] {
			return fmt.Errorf("connection references unknown city: %s", conn.To)
		}
		if conn.Cost < 0 {
			return fmt.Errorf("connection cost cannot be negative")
		}
	}

	// Validate game rules
	if mapData.GameRules.StartingMoney <= 0 {
		return fmt.Errorf("starting money must be positive")
	}

	return nil
}

// GetMap returns a map by ID
func (mm *MapManager) GetMap(id string) (*MapData, bool) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	mapData, exists := mm.maps[id]
	return mapData, exists
}

// GetAllMaps returns all available maps
func (mm *MapManager) GetAllMaps() map[string]*MapData {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]*MapData)
	for id, mapData := range mm.maps {
		result[id] = mapData
	}
	return result
}

// GetMapList returns a list of map metadata (without full data)
func (mm *MapManager) GetMapList() []MapInfo {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	var maps []MapInfo
	for _, mapData := range mm.maps {
		maps = append(maps, MapInfo{
			ID:          mapData.ID,
			Name:        mapData.Name,
			Description: mapData.Description,
			PlayerCount: mapData.PlayerCount,
			RegionCount: len(mapData.Regions),
			CityCount:   len(mapData.Cities),
		})
	}
	return maps
}

// MapInfo contains basic information about a map
type MapInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	PlayerCount PlayerCountConfig `json:"playerCount"`
	RegionCount int               `json:"regionCount"`
	CityCount   int               `json:"cityCount"`
}

// GetConnectionsForCity returns all connections for a given city
func (mm *MapManager) GetConnectionsForCity(mapID, cityID string) []Connection {
	mapData, exists := mm.GetMap(mapID)
	if !exists {
		return nil
	}

	var connections []Connection
	for _, conn := range mapData.Connections {
		if conn.From == cityID || conn.To == cityID {
			connections = append(connections, conn)
		}
	}
	return connections
}

// GetCityByID returns a city by ID from a specific map
func (mm *MapManager) GetCityByID(mapID, cityID string) (*City, bool) {
	mapData, exists := mm.GetMap(mapID)
	if !exists {
		return nil, false
	}

	for _, city := range mapData.Cities {
		if city.ID == cityID {
			return &city, true
		}
	}
	return nil, false
}

// GetRegionCities returns all cities in a specific region
func (mm *MapManager) GetRegionCities(mapID, regionID string) []City {
	mapData, exists := mm.GetMap(mapID)
	if !exists {
		return nil
	}

	var cities []City
	for _, city := range mapData.Cities {
		if city.Region == regionID {
			cities = append(cities, city)
		}
	}
	return cities
}

// FindShortestPath finds the shortest path between two cities (Dijkstra's algorithm)
func (mm *MapManager) FindShortestPath(mapID, fromCity, toCity string) ([]string, int, bool) {
	mapData, exists := mm.GetMap(mapID)
	if !exists {
		return nil, 0, false
	}

	// Build adjacency list
	graph := make(map[string]map[string]int)
	for _, city := range mapData.Cities {
		graph[city.ID] = make(map[string]int)
	}
	
	for _, conn := range mapData.Connections {
		graph[conn.From][conn.To] = conn.Cost
		graph[conn.To][conn.From] = conn.Cost // Bidirectional
	}

	// Dijkstra's algorithm
	distances := make(map[string]int)
	previous := make(map[string]string)
	unvisited := make(map[string]bool)

	// Initialize
	for _, city := range mapData.Cities {
		distances[city.ID] = int(^uint(0) >> 1) // Max int
		unvisited[city.ID] = true
	}
	distances[fromCity] = 0

	for len(unvisited) > 0 {
		// Find unvisited city with minimum distance
		current := ""
		minDist := int(^uint(0) >> 1)
		for city := range unvisited {
			if distances[city] < minDist {
				minDist = distances[city]
				current = city
			}
		}

		if current == "" || distances[current] == int(^uint(0) >> 1) {
			break // No more reachable cities
		}

		delete(unvisited, current)

		if current == toCity {
			break // Found target
		}

		// Update distances to neighbors
		for neighbor, cost := range graph[current] {
			if !unvisited[neighbor] {
				continue
			}
			
			alt := distances[current] + cost
			if alt < distances[neighbor] {
				distances[neighbor] = alt
				previous[neighbor] = current
			}
		}
	}

	// Check if path exists
	if distances[toCity] == int(^uint(0) >> 1) {
		return nil, 0, false
	}

	// Reconstruct path
	var path []string
	current := toCity
	for current != "" {
		path = append([]string{current}, path...)
		current = previous[current]
	}

	return path, distances[toCity], true
}