package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"powergrid/models"
)

// RPCHandler handles RPC calls for the game
type RPCHandler struct {
	lobbyManager *models.LobbyManager
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(lobbyManager *models.LobbyManager) *RPCHandler {
	return &RPCHandler{
		lobbyManager: lobbyManager,
	}
}

// HandleRPC handles RPC requests
func (h *RPCHandler) HandleRPC(w http.ResponseWriter, r *http.Request) {
	// Only POST method is allowed for RPC
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var request struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Handle the request based on the method
	var response interface{}

	switch request.Method {
	case "ListLobbies":
		response = h.lobbyManager.ListLobbiesJSON()
	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
		return
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
