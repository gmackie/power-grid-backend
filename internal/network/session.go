package network

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/pkg/protocol"
)

// Session represents a connected client
type Session struct {
	ID          string
	PlayerID    string
	PlayerName  string
	ConnectedAt time.Time
	LastActive  time.Time
	conn        *websocket.Conn
	sendQueue   chan []byte
	mutex       sync.Mutex
	rooms       map[string]bool // rooms this session is part of
}

// SessionManager handles all active sessions
type SessionManager struct {
	sessions     map[string]*Session
	mutex        sync.RWMutex
	messageQueue chan messageTask
}

type messageTask struct {
	session *Session
	message protocol.Message
}

// Global session manager instance
var Manager *SessionManager

func init() {
	Manager = &SessionManager{
		sessions:     make(map[string]*Session),
		messageQueue: make(chan messageTask, 1000),
	}
	go Manager.processMessages()
}

// AddSession adds a session to the manager
func (sm *SessionManager) AddSession(session *Session) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.sessions[session.ID] = session
}

// RemoveSession removes a session from the manager
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, sessionID)
}

// GetSession gets a session by ID
func (sm *SessionManager) GetSession(sessionID string) *Session {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.sessions[sessionID]
}

// BroadcastToRoom sends a message to all sessions in a room
func (sm *SessionManager) BroadcastToRoom(roomID string, msgType protocol.MessageType, payload interface{}) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	for _, session := range sm.sessions {
		if session.InRoom(roomID) {
			session.SendMessage(msgType, payload)
		}
	}
}

// HandleMessage queues a message for processing
func (sm *SessionManager) HandleMessage(session *Session, msg protocol.Message) {
	select {
	case sm.messageQueue <- messageTask{session: session, message: msg}:
		// Successfully queued
	default:
		// Queue is full, respond with an error
		session.SendMessage(protocol.MsgError, protocol.ErrorPayload{
			Code:    "SERVER_BUSY",
			Message: "Server is too busy to process your request",
		})
	}
}

// processMessages processes messages in the queue
func (sm *SessionManager) processMessages() {
	for task := range sm.messageQueue {
		err := ProcessGameMessage(task.session, task.message)
		if err != nil {
			task.session.SendMessage(protocol.MsgError, protocol.ErrorPayload{
				Code:    "MESSAGE_ERROR",
				Message: err.Error(),
			})
		}
	}
}

// NewSession creates a new session for a connected client
func NewSession(conn *websocket.Conn) *Session {
	sessionID := uuid.New().String()
	session := &Session{
		ID:          sessionID,
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
		conn:        conn,
		sendQueue:   make(chan []byte, 100), // Buffer up to 100 messages
		rooms:       make(map[string]bool),
	}

	// Start goroutines for reading and writing
	go session.readPump()
	go session.writePump()

	// Add to session manager
	Manager.AddSession(session)

	return session
}

// Close closes the session
func (s *Session) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
		close(s.sendQueue)
		Manager.RemoveSession(s.ID)
	}
}

// AddToRoom adds this session to a room
func (s *Session) AddToRoom(roomID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.rooms[roomID] = true
}

// RemoveFromRoom removes this session from a room
func (s *Session) RemoveFromRoom(roomID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.rooms, roomID)
}

// InRoom checks if the session is in a room
func (s *Session) InRoom(roomID string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.rooms[roomID]
	return ok
}

// SendMessage sends a message to the client
func (s *Session) SendMessage(msgType protocol.MessageType, payload interface{}) error {
	msg := protocol.NewMessage(msgType, payload)
	msg.SessionID = s.ID
	return s.Send(msg)
}

// Send sends a protocol message to the client
func (s *Session) Send(msg *protocol.Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	if s.conn == nil {
		s.mutex.Unlock()
		return nil // Session closed
	}
	s.mutex.Unlock()

	select {
	case s.sendQueue <- jsonData:
		// Message queued successfully
	default:
		// Queue is full, this is an error condition
		return errors.New("send queue full")
	}

	return nil
}

// readPump reads messages from the websocket connection
func (s *Session) readPump() {
	defer s.Close()

	s.conn.SetReadLimit(4096) // Limit message size
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		s.LastActive = time.Now()

		// Parse message
		var msg protocol.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			// Send error back to client
			s.SendMessage(protocol.MsgError, protocol.ErrorPayload{
				Code:    "INVALID_MESSAGE",
				Message: "Could not parse message",
			})
			continue
		}

		// Process message
		s.processMessage(msg)
	}
}

// writePump writes messages to the websocket connection
func (s *Session) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		s.Close()
	}()

	for {
		select {
		case message, ok := <-s.sendQueue:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processMessage handles an incoming message
func (s *Session) processMessage(msg protocol.Message) {
	// Update the session ID if it was included
	if msg.SessionID != "" {
		s.ID = msg.SessionID
	}

	// Handle basic session messages
	switch msg.Type {
	case protocol.MsgPing:
		s.SendMessage(protocol.MsgPong, nil)
		return
	case protocol.MsgDisconnect:
		s.Close()
		return
	}

	// Other messages are handled by the main server handler
	Manager.HandleMessage(s, msg)
}
