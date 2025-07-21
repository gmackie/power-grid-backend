package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"powergrid/internal/analytics"
	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

var (
	dataDir = flag.String("data-dir", "./data/analytics", "Directory to store analytics data")
	games   = flag.Int("games", 10, "Number of demo games to generate")
)

func main() {
	flag.Parse()

	logger.InitLoggers(logger.INFO, false)
	
	// Create analytics service
	service := analytics.NewService(*dataDir)
	
	logger.TestLogger.Info("Generating %d demo games for analytics testing", *games)
	
	playerNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"}
	maps := []string{"usa", "germany"}
	
	for i := 0; i < *games; i++ {
		generateDemoGame(service, i+1, playerNames, maps)
		time.Sleep(100 * time.Millisecond) // Small delay for realistic timestamps
	}
	
	logger.TestLogger.Info("Demo data generation completed!")
	logger.TestLogger.Info("Test the API with: ./scripts/test_analytics_api.sh")
}

func generateDemoGame(service *analytics.Service, gameNum int, playerNames, maps []string) {
	// Generate random game
	gameID := fmt.Sprintf("demo_game_%d", gameNum)
	gameName := fmt.Sprintf("Demo Game #%d", gameNum)
	mapName := maps[rand.Intn(len(maps))]
	
	// Select random players (2-6 players)
	numPlayers := rand.Intn(5) + 2
	players := make([]string, numPlayers)
	selectedPlayers := make(map[string]bool)
	
	for i := 0; i < numPlayers; i++ {
		for {
			player := playerNames[rand.Intn(len(playerNames))]
			if !selectedPlayers[player] {
				players[i] = player
				selectedPlayers[player] = true
				break
			}
		}
	}
	
	// Start game tracking
	service.TrackGameStart(gameID, gameName, mapName, players)
	
	// Generate some game state updates
	for round := 1; round <= rand.Intn(8)+5; round++ {
		gameState := generateGameState(gameID, gameName, players, round)
		service.TrackGameState(gameID, gameState)
	}
	
	// End game with random winner
	winner := players[rand.Intn(len(players))]
	finalState := generateFinalGameState(gameID, gameName, players, winner)
	service.TrackGameEnd(gameID, winner, finalState)
	
	logger.TestLogger.Debug("Generated demo game %s with winner %s", gameID, winner)
}

func generateGameState(gameID, gameName string, playerNames []string, round int) *protocol.GameStatePayload {
	players := make(map[string]protocol.PlayerInfo)
	turnOrder := make([]string, len(playerNames))
	
	for i, name := range playerNames {
		playerID := fmt.Sprintf("player_%d", i)
		turnOrder[i] = playerID
		
		// Generate realistic game progression
		baseCities := round/2 + rand.Intn(3)
		basePlants := min(round/3 + 1, 4)
		baseMoney := 50 - round*3 + rand.Intn(40)
		
		resources := make(map[string]int)
		if round > 2 {
			resources["coal"] = rand.Intn(8)
			resources["oil"] = rand.Intn(8)
			resources["garbage"] = rand.Intn(6)
			resources["uranium"] = rand.Intn(4)
		}
		
		cities := make([]string, baseCities)
		for j := 0; j < baseCities; j++ {
			cities[j] = fmt.Sprintf("city_%d_%d", i, j)
		}
		
		plants := make([]protocol.PowerPlantInfo, basePlants)
		for j := 0; j < basePlants; j++ {
			plants[j] = protocol.PowerPlantInfo{
				ID:           (j + 1) * 10,
				Cost:         (j+1)*15 + rand.Intn(20),
				Capacity:     j + 2,
				ResourceType: []string{"coal", "oil", "garbage", "uranium", "eco"}[rand.Intn(5)],
				ResourceCost: rand.Intn(3) + 1,
			}
		}
		
		players[playerID] = protocol.PlayerInfo{
			ID:            playerID,
			Name:          name,
			Color:         fmt.Sprintf("color_%d", i),
			Money:         baseMoney,
			Cities:        cities,
			PowerPlants:   plants,
			Resources:     resources,
			PoweredCities: min(baseCities, basePlants*2),
		}
	}
	
	phases := []protocol.GamePhase{
		protocol.PhasePlayerOrder,
		protocol.PhaseAuction,
		protocol.PhaseBuyResources,
		protocol.PhaseBuildCities,
		protocol.PhaseBureaucracy,
	}
	
	return &protocol.GameStatePayload{
		GameID:       gameID,
		Name:         gameName,
		Status:       protocol.StatusPlaying,
		CurrentPhase: phases[rand.Intn(len(phases))],
		CurrentTurn:  turnOrder[rand.Intn(len(turnOrder))],
		CurrentRound: round,
		Players:      players,
		TurnOrder:    turnOrder,
		Map: protocol.MapInfo{
			Name:   "Demo Map",
			Cities: make(map[string]protocol.CityInfo),
		},
		Market: protocol.MarketInfo{
			Resources: make(map[string][]int),
		},
		PowerPlants: []protocol.PowerPlantInfo{},
	}
}

func generateFinalGameState(gameID, gameName string, playerNames []string, winner string) *protocol.GameStatePayload {
	state := generateGameState(gameID, gameName, playerNames, 12) // Final round
	
	// Make sure winner has winning stats
	for playerID, player := range state.Players {
		if player.Name == winner {
			// Winner gets 17+ cities
			cities := make([]string, 17+rand.Intn(3))
			for i := range cities {
				cities[i] = fmt.Sprintf("city_win_%d", i)
			}
			player.Cities = cities
			player.PoweredCities = len(cities)
			player.Money = rand.Intn(100) + 50
			state.Players[playerID] = player
			break
		}
	}
	
	state.Status = protocol.StatusFinished
	state.CurrentPhase = protocol.PhaseGameEnd
	
	return state
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}