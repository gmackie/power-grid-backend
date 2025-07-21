package ai

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// ClientConfig holds configuration for AI client
type ClientConfig struct {
	ServerURL     string
	Strategy      string
	PlayerName    string
	PlayerColor   string
	GameID        string
	AutoPlay      bool
	ThinkTime     time.Duration
	Interactive   bool
	LogDecisions  bool
	Personality   string
	Difficulty    string
	DecisionDelay time.Duration
}

// AIClient represents an AI player client
type AIClient struct {
	config    ClientConfig
	conn      *websocket.Conn
	sessionID string
	gameID    string
	playerID  string
	strategy  Strategy
	logger    *logger.ColoredLogger
	
	// Game state
	gameState     *protocol.GameStatePayload
	stateTracker  *GameStateTracker
	decisionEngine *DecisionEngine
	isConnected   bool
	isInGame      bool
	
	// Decision logging
	decisionLogs chan DecisionLog
	
	// Synchronization
	mu       sync.RWMutex
	shutdown chan struct{}
}

// NewAIClient creates a new AI client
func NewAIClient(config ClientConfig) (*AIClient, error) {
	// Generate defaults if not provided
	if config.PlayerName == "" {
		config.PlayerName = fmt.Sprintf("AI_%s_%d", config.Strategy, rand.Intn(1000))
	}
	
	if config.PlayerColor == "" {
		colors := []string{"red", "blue", "green", "yellow", "purple", "orange"}
		config.PlayerColor = colors[rand.Intn(len(colors))]
	}
	
	// Create strategy
	strategy := CreateStrategy(config.Strategy)
	if strategy == nil {
		return nil, fmt.Errorf("unknown strategy: %s", config.Strategy)
	}
	
	// Create colored logger for this AI
	loggerColor := getColorForStrategy(config.Strategy)
	aiLogger := logger.CreateAILogger(config.Strategy, loggerColor)
	
	client := &AIClient{
		config:       config,
		sessionID:    uuid.New().String(),
		strategy:     strategy,
		logger:       aiLogger,
		shutdown:     make(chan struct{}),
		decisionLogs: make(chan DecisionLog, 100),
	}
	
	// Initialize state tracker
	client.stateTracker = NewGameStateTracker(client.sessionID, aiLogger)
	
	// Initialize decision engine
	client.decisionEngine = NewDecisionEngine(client.stateTracker, strategy, aiLogger)
	
	// Enable decision logging if requested
	if config.LogDecisions {
		client.decisionEngine.SetDecisionLogging(true, client.decisionLogs)
		// Start decision logger
		go client.runDecisionLogger()
	}
	
	return client, nil
}

// Connect establishes connection to the server
func (c *AIClient) Connect() error {
	u, err := url.Parse(c.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	
	c.logger.Info("Connecting to %s...", c.config.ServerURL)
	
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	
	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.mu.Unlock()
	
	// Start message handler
	go c.messageHandler()
	
	// Send connect message
	connectMsg := protocol.NewMessage(protocol.MsgConnect, map[string]interface{}{
		"session_id":   c.sessionID,
		"player_name":  c.config.PlayerName,
		"player_color": c.config.PlayerColor,
	})
	connectMsg.SessionID = c.sessionID
	
	if err := c.sendMessage(connectMsg); err != nil {
		return fmt.Errorf("failed to send connect message: %w", err)
	}
	
	c.logger.Info("Connected as %s (%s)", c.config.PlayerName, c.config.PlayerColor)
	
	// Join or create game
	time.Sleep(100 * time.Millisecond) // Wait for connection to be established
	
	if c.config.GameID != "" {
		c.joinGame(c.config.GameID)
	} else {
		c.createGame()
	}
	
	return nil
}

// Disconnect closes the connection
func (c *AIClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.isConnected {
		return
	}
	
	close(c.shutdown)
	c.isConnected = false
	
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	
	c.logger.Info("Disconnected")
}

// runDecisionLogger sends decision logs to the server for admin monitoring
func (c *AIClient) runDecisionLogger() {
	for {
		select {
		case <-c.shutdown:
			return
		case decision := <-c.decisionLogs:
			// Send decision log to server
			msg := protocol.NewMessage(protocol.MsgAIDecision, map[string]interface{}{
				"gameId":       c.gameID,
				"playerId":     c.playerID,
				"playerName":   decision.PlayerName,
				"timestamp":    decision.Timestamp,
				"phase":        decision.Phase,
				"decisionType": decision.DecisionType,
				"decision":     decision.Decision,
				"reasoning":    decision.Reasoning,
				"factors":      decision.Factors,
			})
			msg.SessionID = c.sessionID
			
			if err := c.sendMessage(msg); err != nil {
				c.logger.Error("Failed to send decision log: %v", err)
			} else {
				c.logger.Debug("Sent decision log: %s", decision.DecisionType)
			}
		}
	}
}

// sendMessage sends a message to the server
func (c *AIClient) sendMessage(msg *protocol.Message) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	
	if conn == nil {
		return fmt.Errorf("not connected")
	}
	
	msg.Timestamp = time.Now().Unix()
	if msg.SessionID == "" {
		msg.SessionID = c.sessionID
	}
	if msg.GameID == "" && c.gameID != "" {
		msg.GameID = c.gameID
	}
	
	data, err := protocol.SerializeMessage(*msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}
	
	return conn.WriteMessage(websocket.TextMessage, data)
}

// messageHandler handles incoming messages
func (c *AIClient) messageHandler() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Message handler panic: %v", r)
		}
	}()
	
	for {
		select {
		case <-c.shutdown:
			return
		default:
		}
		
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		
		if conn == nil {
			return
		}
		
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error: %v", err)
			}
			return
		}
		
		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Error("Failed to parse message: %v", err)
			continue
		}
		
		c.handleMessage(&msg)
	}
}

// handleMessage processes an incoming message
func (c *AIClient) handleMessage(msg *protocol.Message) {
	c.logger.Debug("Received %s message", msg.Type)
	
	switch msg.Type {
	case protocol.MsgError:
		c.handleError(msg)
	case protocol.MsgGameState:
		c.handleGameState(msg)
	case protocol.MsgPhaseChange:
		c.handlePhaseChange(msg)
	case protocol.MsgTurnChange:
		c.handleTurnChange(msg)
	case protocol.MsgPlayerJoined:
		c.handlePlayerJoined(msg)
	default:
		c.logger.Debug("Unhandled message type: %s", msg.Type)
	}
}

// handleError processes error messages
func (c *AIClient) handleError(msg *protocol.Message) {
	if payload, ok := msg.Payload.(map[string]interface{}); ok {
		if errMsg, exists := payload["message"]; exists {
			c.logger.Error("Server error: %v", errMsg)
		}
	}
}

// handleGameState processes game state updates
func (c *AIClient) handleGameState(msg *protocol.Message) {
	data, err := json.Marshal(msg.Payload)
	if err != nil {
		c.logger.Error("Failed to marshal game state: %v", err)
		return
	}
	
	var gameState protocol.GameStatePayload
	if err := json.Unmarshal(data, &gameState); err != nil {
		c.logger.Error("Failed to unmarshal game state: %v", err)
		return
	}
	
	c.mu.Lock()
	c.gameState = &gameState
	c.gameID = gameState.GameID
	c.isInGame = true
	
	// Find our player ID
	for id, player := range gameState.Players {
		if player.Name == c.config.PlayerName {
			c.playerID = id
			c.stateTracker.playerID = id // Update tracker's player ID
			break
		}
	}
	c.mu.Unlock()
	
	// Update state tracker
	c.stateTracker.UpdateState(&gameState)
	
	c.logger.Info("Game state updated - Phase: %s, Turn: %s, Round: %d", 
		gameState.CurrentPhase, gameState.CurrentTurn, gameState.CurrentRound)
	
	// Make move if it's our turn and auto-play is enabled
	if c.config.AutoPlay && gameState.CurrentTurn == c.playerID {
		// Use decision delay if specified, otherwise use think time
		delay := c.config.ThinkTime
		if c.config.DecisionDelay > 0 {
			delay = c.config.DecisionDelay
		}
		go c.makeMoveWithDelay(delay)
	}
}

// handlePhaseChange processes phase change notifications
func (c *AIClient) handlePhaseChange(msg *protocol.Message) {
	if payload, ok := msg.Payload.(map[string]interface{}); ok {
		if phase, exists := payload["phase"]; exists {
			c.logger.Info("Phase changed to: %v", phase)
		}
	}
}

// handleTurnChange processes turn change notifications
func (c *AIClient) handleTurnChange(msg *protocol.Message) {
	if payload, ok := msg.Payload.(map[string]interface{}); ok {
		if playerID, exists := payload["current_player_id"]; exists {
			c.logger.Info("Turn changed to player: %v", playerID)
			
			// Make move if it's our turn and auto-play is enabled
			if c.config.AutoPlay && playerID == c.playerID {
				// Use decision delay if specified, otherwise use think time
				delay := c.config.ThinkTime
				if c.config.DecisionDelay > 0 {
					delay = c.config.DecisionDelay
				}
				go c.makeMoveWithDelay(delay)
			}
		}
	}
}

// handlePlayerJoined processes player joined notifications
func (c *AIClient) handlePlayerJoined(msg *protocol.Message) {
	if payload, ok := msg.Payload.(map[string]interface{}); ok {
		if name, exists := payload["name"]; exists {
			c.logger.Info("Player joined: %v", name)
		}
	}
}

// makeMoveWithDelay executes a move after a delay
func (c *AIClient) makeMoveWithDelay(delay time.Duration) {
	if delay > 0 {
		time.Sleep(delay)
	}
	c.makeMove()
}

// makeMove executes a move based on the current strategy
func (c *AIClient) makeMove() {
	
	c.mu.RLock()
	gameState := c.gameState
	playerID := c.playerID
	c.mu.RUnlock()
	
	if gameState == nil || playerID == "" {
		return
	}
	
	c.logger.Info("Making move in phase: %s", gameState.CurrentPhase)
	
	// Use decision engine for sophisticated decision making
	move := c.decisionEngine.MakeDecision()
	if move == nil {
		// Fall back to basic strategy
		move = c.strategy.GetMove(gameState, playerID)
	}
	
	if move == nil {
		c.logger.Debug("No move available")
		return
	}
	
	c.logger.Info("Executing move: %s", move.Type)
	
	// Send the move
	if err := c.sendMessage(move); err != nil {
		c.logger.Error("Failed to send move: %v", err)
	}
}

// createGame creates a new game
func (c *AIClient) createGame() {
	gameName := fmt.Sprintf("AI_Game_%s_%d", c.config.Strategy, time.Now().Unix())
	
	createMsg := protocol.NewMessage(protocol.MsgCreateGame, protocol.CreateGamePayload{
		Name:       gameName,
		Map:        "usa",
		MaxPlayers: 6,
	})
	createMsg.SessionID = c.sessionID
	
	c.logger.Info("Creating game: %s", gameName)
	
	if err := c.sendMessage(createMsg); err != nil {
		c.logger.Error("Failed to create game: %v", err)
	}
}

// joinGame joins an existing game
func (c *AIClient) joinGame(gameID string) {
	joinMsg := protocol.NewMessage(protocol.MsgJoinGame, protocol.JoinGamePayload{
		GameID:     gameID,
		PlayerName: c.config.PlayerName,
		Color:      c.config.PlayerColor,
	})
	joinMsg.SessionID = c.sessionID
	
	c.logger.Info("Joining game: %s", gameID)
	
	if err := c.sendMessage(joinMsg); err != nil {
		c.logger.Error("Failed to join game: %v", err)
	}
}

// getColorForStrategy returns an appropriate color for the strategy
func getColorForStrategy(strategy string) string {
	switch strategy {
	case "aggressive":
		return logger.ColorBrightRed
	case "conservative":
		return logger.ColorBrightBlue
	case "balanced":
		return logger.ColorBrightGreen
	case "random":
		return logger.ColorBrightYellow
	default:
		return logger.ColorBrightCyan
	}
}

// Config returns the client configuration
func (c *AIClient) Config() ClientConfig {
	return c.config
}