package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"powergrid/internal/maps"
)

// HandleMapsAPI handles HTTP requests for map information
func HandleMapsAPI(w http.ResponseWriter, r *http.Request, mapManager *maps.MapManager) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to determine which endpoint
	path := strings.TrimPrefix(r.URL.Path, "/maps")
	
	switch {
	case path == "" || path == "/":
		// GET /maps - return list of all maps
		handleMapsList(w, r, mapManager)
	case strings.HasPrefix(path, "/"):
		// GET /maps/{id} - return specific map data
		mapID := strings.TrimPrefix(path, "/")
		handleMapDetail(w, r, mapManager, mapID)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleMapsList returns a list of all available maps
func handleMapsList(w http.ResponseWriter, r *http.Request, mapManager *maps.MapManager) {
	mapList := mapManager.GetMapList()
	
	response := struct {
		Maps []maps.MapInfo `json:"maps"`
	}{
		Maps: mapList,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleMapDetail returns detailed information about a specific map
func handleMapDetail(w http.ResponseWriter, r *http.Request, mapManager *maps.MapManager, mapID string) {
	mapData, exists := mapManager.GetMap(mapID)
	if !exists {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(mapData); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}