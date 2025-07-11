package protocol

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of message being sent
type MessageType string

// Game status constants
type GameStatus string
type GamePhase string

// Message types
const (
	// Connection messages
	MsgConnect    MessageType = "CONNECT"
	MsgDisconnect MessageType = "DISCONNECT"
	MsgPing       MessageType = "PING"
	MsgPong       MessageType = "PONG"

	// Lobby messages
	MsgCreateGame MessageType = "CREATE_GAME"
	MsgJoinGame   MessageType = "JOIN_GAME"
	MsgLeaveGame  MessageType = "LEAVE_GAME"
	MsgListGames  MessageType = "LIST_GAMES"
	MsgStartGame  MessageType = "START_GAME"

	// Game action messages
	MsgGameState    MessageType = "GAME_STATE"
	MsgBidPlant     MessageType = "BID_PLANT"
	MsgBuyResources MessageType = "BUY_RESOURCES"
	MsgBuildCity    MessageType = "BUILD_CITY"
	MsgPowerCities  MessageType = "POWER_CITIES"
	MsgEndTurn      MessageType = "END_TURN"

	// Notification messages
	MsgError        MessageType = "ERROR"
	MsgPhaseChange  MessageType = "PHASE_CHANGE"
	MsgTurnChange   MessageType = "TURN_CHANGE"
	MsgPlayerJoined MessageType = "PLAYER_JOINED"
)

// Game status values
const (
	StatusLobby    GameStatus = "LOBBY"
	StatusPlaying  GameStatus = "PLAYING"
	StatusFinished GameStatus = "FINISHED"
)

// Game phases
const (
	PhasePlayerOrder  GamePhase = "PLAYER_ORDER"
	PhaseAuction      GamePhase = "AUCTION"
	PhaseBuyResources GamePhase = "BUY_RESOURCES"
	PhaseBuildCities  GamePhase = "BUILD_CITIES"
	PhaseBureaucracy  GamePhase = "BUREAUCRACY"
	PhaseGameEnd      GamePhase = "GAME_END"
)

// Message represents a communication between client and server
type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	SessionID string      `json:"session_id,omitempty"`
	GameID    string      `json:"game_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, payload interface{}) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}
}

// ErrorPayload contains information about an error
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateGamePayload contains data to create a new game
type CreateGamePayload struct {
	Name       string `json:"name"`
	Map        string `json:"map"`
	MaxPlayers int    `json:"max_players"`
}

// JoinGamePayload contains data to join an existing game
type JoinGamePayload struct {
	GameID     string `json:"game_id"`
	PlayerName string `json:"player_name"`
	Color      string `json:"color"`
}

// GameListPayload contains data about available games
type GameListPayload struct {
	Games []GameInfo `json:"games"`
}

// GameInfo contains summary information about a game
type GameInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Status     GameStatus `json:"status"`
	Map        string     `json:"map"`
	Players    int        `json:"players"`
	MaxPlayers int        `json:"max_players"`
	CreatedAt  int64      `json:"created_at"`
}

// GameStatePayload contains the complete game state
type GameStatePayload struct {
	GameID       string                `json:"game_id"`
	Name         string                `json:"name"`
	Status       GameStatus            `json:"status"`
	CurrentPhase GamePhase             `json:"current_phase"`
	CurrentTurn  string                `json:"current_turn"`
	CurrentRound int                   `json:"current_round"`
	Players      map[string]PlayerInfo `json:"players"`
	Map          MapInfo               `json:"map"`
	Market       MarketInfo            `json:"market"`
	PowerPlants  []PowerPlantInfo      `json:"power_plants"`
	TurnOrder    []string              `json:"turn_order"`
}

// PlayerInfo contains information about a player
type PlayerInfo struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Color         string           `json:"color"`
	Money         int              `json:"money"`
	Cities        []string         `json:"cities"`
	PowerPlants   []PowerPlantInfo `json:"power_plants"`
	Resources     map[string]int   `json:"resources"`
	PoweredCities int              `json:"powered_cities"`
}

// MapInfo contains information about the game map
type MapInfo struct {
	Name        string              `json:"name"`
	Cities      map[string]CityInfo `json:"cities"`
	Connections []ConnectionInfo    `json:"connections"`
}

// CityInfo contains information about a city
type CityInfo struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Region   string     `json:"region"`
	Position [2]float64 `json:"position"`
	Slots    []string   `json:"slots"` // Player IDs that have built here
}

// ConnectionInfo contains information about a connection between cities
type ConnectionInfo struct {
	CityA string `json:"city_a"`
	CityB string `json:"city_b"`
	Cost  int    `json:"cost"`
}

// MarketInfo contains information about the resource market
type MarketInfo struct {
	Resources map[string][]int `json:"resources"` // Resource type -> [price] -> count
}

// PowerPlantInfo contains information about a power plant
type PowerPlantInfo struct {
	ID           int    `json:"id"`
	Cost         int    `json:"cost"`
	Capacity     int    `json:"capacity"`
	ResourceType string `json:"resource_type"`
	ResourceCost int    `json:"resource_cost"`
}

// Action payloads

// BidPlantPayload contains data for bidding on a power plant
type BidPlantPayload struct {
	PlantID int `json:"plant_id"`
	Bid     int `json:"bid"`
}

// BuyResourcesPayload contains data for buying resources
type BuyResourcesPayload struct {
	Resources map[string]int `json:"resources"` // Resource type -> count
}

// BuildCityPayload contains data for building in a city
type BuildCityPayload struct {
	CityID string `json:"city_id"`
}

// PowerCitiesPayload contains data for powering cities
type PowerCitiesPayload struct {
	PowerPlants []int `json:"power_plants"` // IDs of power plants to use
}

// PhaseChangePayload contains data for phase change notifications
type PhaseChangePayload struct {
	Phase string `json:"phase"`
	Round int    `json:"round"`
}

// TurnChangePayload contains data for turn change notifications
type TurnChangePayload struct {
	CurrentPlayerID string `json:"current_player_id"`
	Turn           int    `json:"turn"`
}

// SerializeMessage converts a message to JSON bytes
func SerializeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// DeserializeMessage converts JSON bytes to a message
func DeserializeMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// Game related structs

// GameState represents the complete state of a game
type GameState struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	CurrentPhase GamePhase         `json:"current_phase"`
	CurrentTurn  int               `json:"current_turn"`
	CurrentRound int               `json:"current_round"`
	Map          Map               `json:"map"`
	Players      map[string]Player `json:"players"`
	Market       ResourceMarket    `json:"market"`
	PowerPlants  []PowerPlant      `json:"power_plants"`
	AuctionState *AuctionState     `json:"auction_state,omitempty"`
	TurnOrder    []string          `json:"turn_order"`
	Status       GameStatus        `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Map represents the game map
type Map struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Cities      map[string]City         `json:"cities"`
	Connections map[string][]Connection `json:"connections"`
}

// City represents a city on the map
type City struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Position Position `json:"position"`
	Region   string   `json:"region"`
	Players  []string `json:"players,omitempty"`
}

// Position represents a 2D position
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Connection represents a connection between cities
type Connection struct {
	FromCity string `json:"from_city"`
	ToCity   string `json:"to_city"`
	Cost     int    `json:"cost"`
}

// Player represents a player in the game
type Player struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Money       int            `json:"money"`
	Resources   map[string]int `json:"resources"`
	PowerPlants []int          `json:"power_plants"`
	Cities      []string       `json:"cities"`
	Powered     int            `json:"powered"`
	Color       string         `json:"color"`
	IsConnected bool           `json:"is_connected"`
	IsReady     bool           `json:"is_ready"`
}

// ResourceMarket represents the resource market
type ResourceMarket struct {
	Resources map[string]ResourceSupply `json:"resources"`
}

// ResourceSupply represents the supply of a resource
type ResourceSupply struct {
	Available int `json:"available"`
	Price     int `json:"price"`
}

// ResourceType represents a type of resource
type ResourceType string

const (
	ResourceCoal    ResourceType = "COAL"
	ResourceOil     ResourceType = "OIL"
	ResourceGarbage ResourceType = "GARBAGE"
	ResourceUranium ResourceType = "URANIUM"
)

// PowerPlant represents a power plant
type PowerPlant struct {
	ID             int          `json:"id"`
	Cost           int          `json:"cost"`
	PowerCapacity  int          `json:"power_capacity"`
	ResourceType   ResourceType `json:"resource_type"`
	ResourceAmount int          `json:"resource_amount"`
}

// AuctionState represents the state of an auction
type AuctionState struct {
	CurrentPlant     PowerPlant `json:"current_plant"`
	CurrentBid       int        `json:"current_bid"`
	CurrentBidder    string     `json:"current_bidder"`
	RemainingBidders []string   `json:"remaining_bidders"`
	IsActive         bool       `json:"is_active"`
}
