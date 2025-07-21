package network

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"powergrid/internal/game"
	"powergrid/pkg/protocol"
)

// GameManager manages all active games
type GameManager struct {
	games         map[string]*game.Game
	mutex         sync.RWMutex
	analyticsHook *AnalyticsHook
}

// Global game manager instance
var Games *GameManager

func init() {
	Games = &GameManager{
		games: make(map[string]*game.Game),
	}
}

// SetAnalyticsHook sets the analytics hook for the game manager
func (gm *GameManager) SetAnalyticsHook(hook *AnalyticsHook) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	gm.analyticsHook = hook
}

// CreateGame creates a new game
func (gm *GameManager) CreateGame(name, mapName string) (*game.Game, error) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	gameID := uuid.New().String()
	newGame, err := game.NewGame(gameID, name, mapName)
	if err != nil {
		return nil, err
	}

	// Setup analytics if available
	if gm.analyticsHook != nil {
		gm.analyticsHook.SetupGameAnalytics(newGame)
	}

	gm.games[gameID] = newGame
	return newGame, nil
}

// GetGame gets a game by ID
func (gm *GameManager) GetGame(gameID string) *game.Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return gm.games[gameID]
}

// RemoveGame removes a game
func (gm *GameManager) RemoveGame(gameID string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	delete(gm.games, gameID)
}

// ProcessGameMessage processes a game-related message
func ProcessGameMessage(session *Session, msg protocol.Message) error {
	switch msg.Type {
	case protocol.MsgConnect:
		return handleConnect(session, msg)
	case protocol.MsgCreateGame:
		return handleCreateGame(session, msg)
	case protocol.MsgJoinGame:
		return handleJoinGame(session, msg)
	case protocol.MsgStartGame:
		return handleStartGame(session, msg)
	case protocol.MsgBidPlant:
		return handleBidPlant(session, msg)
	case protocol.MsgBuyResources:
		return handleBuyResources(session, msg)
	case protocol.MsgBuildCity:
		return handleBuildCity(session, msg)
	case protocol.MsgPowerCities:
		return handlePowerCities(session, msg)
	case protocol.MsgEndTurn:
		return handleEndTurn(session, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func handleConnect(session *Session, msg protocol.Message) error {
	// Extract connection payload
	var payload struct {
		PlayerName string `json:"player_name"`
		PlayerID   string `json:"player_id"`
		GameID     string `json:"game_id"`
	}
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	// Update session with player info
	session.PlayerID = payload.PlayerID
	session.PlayerName = payload.PlayerName

	// Find the game
	gameObj := Games.GetGame(payload.GameID)
	if gameObj == nil {
		return fmt.Errorf("game not found: %s", payload.GameID)
	}

	// Add session to the game room
	session.AddToRoom(gameObj.ID)

	// Send the current game state to the player
	session.SendMessage(protocol.MsgGameState, gameObj.GetGameStatePayload())

	// Notify other players
	Manager.BroadcastToRoom(gameObj.ID, protocol.MsgPlayerJoined, map[string]interface{}{
		"player_id":   payload.PlayerID,
		"player_name": payload.PlayerName,
	})

	return nil
}

func handleCreateGame(session *Session, msg protocol.Message) error {
	var payload protocol.CreateGamePayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	newGame, err := Games.CreateGame(payload.Name, payload.Map)
	if err != nil {
		return err
	}

	// Send back game created confirmation
	session.SendMessage(protocol.MsgGameState, newGame.GetGameStatePayload())
	
	// Store the game ID in the message for future reference
	session.AddToRoom(newGame.ID)
	
	return nil
}

func handleJoinGame(session *Session, msg protocol.Message) error {
	var payload protocol.JoinGamePayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	gameObj := Games.GetGame(payload.GameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	// Generate a player ID if not provided
	playerID := session.PlayerID
	if playerID == "" {
		playerID = uuid.New().String()
		session.PlayerID = playerID
	}
	session.PlayerName = payload.PlayerName

	err := gameObj.AddPlayer(playerID, payload.PlayerName, payload.Color, session)
	if err != nil {
		return err
	}

	session.AddToRoom(gameObj.ID)
	gameObj.BroadcastGameState()
	
	return nil
}

func handleStartGame(session *Session, msg protocol.Message) error {
	// Find the game this session is in
	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.Start()
	if err != nil {
		return err
	}

	return nil
}

func handleBidPlant(session *Session, msg protocol.Message) error {
	var payload protocol.BidPlantPayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.ProcessAction(session.PlayerID, protocol.MsgBidPlant, payload)
	return err
}

func handleBuyResources(session *Session, msg protocol.Message) error {
	var payload protocol.BuyResourcesPayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.ProcessAction(session.PlayerID, protocol.MsgBuyResources, payload)
	return err
}

func handleBuildCity(session *Session, msg protocol.Message) error {
	var payload protocol.BuildCityPayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.ProcessAction(session.PlayerID, protocol.MsgBuildCity, payload)
	return err
}

func handlePowerCities(session *Session, msg protocol.Message) error {
	var payload protocol.PowerCitiesPayload
	if err := parsePayload(msg.Payload, &payload); err != nil {
		return err
	}

	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.ProcessAction(session.PlayerID, protocol.MsgPowerCities, payload)
	return err
}

func handleEndTurn(session *Session, msg protocol.Message) error {
	gameID := findGameForSession(session)
	if gameID == "" {
		return fmt.Errorf("session not in any game")
	}

	gameObj := Games.GetGame(gameID)
	if gameObj == nil {
		return fmt.Errorf("game not found")
	}

	err := gameObj.ProcessAction(session.PlayerID, protocol.MsgEndTurn, nil)
	return err
}

// Helper functions

func parsePayload(payload interface{}, target interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

func findGameForSession(session *Session) string {
	// Look through all rooms to find a game ID
	session.mutex.Lock()
	defer session.mutex.Unlock()
	
	for roomID := range session.rooms {
		if Games.GetGame(roomID) != nil {
			return roomID
		}
	}
	return ""
}