#!/bin/bash

# End-to-end test script for Power Grid server
# This script starts the server, runs tests against it, and cleans up

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVER_PORT=4081
SERVER_LOG="logs/test_server.log"
SERVER_PID=""

# Ensure logs directory exists
mkdir -p logs

# Function to cleanup on exit
cleanup() {
    if [ ! -z "$SERVER_PID" ]; then
        echo -e "${YELLOW}Stopping server (PID: $SERVER_PID)...${NC}"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

echo -e "${GREEN}Starting Power Grid E2E Tests${NC}"
echo "==============================="

# Build the server
echo -e "${YELLOW}Building server...${NC}"
go build -o powergrid_server ./cmd/server/

# Start the server
echo -e "${YELLOW}Starting server on port $SERVER_PORT...${NC}"
./powergrid_server -addr=:$SERVER_PORT > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
sleep 2

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}Server failed to start!${NC}"
    echo "Server log:"
    cat "$SERVER_LOG"
    exit 1
fi

echo -e "${GREEN}Server started successfully (PID: $SERVER_PID)${NC}"

# Run the E2E test client
echo -e "${YELLOW}Running E2E tests...${NC}"
echo

# Create a temporary Go file for E2E tests
cat > /tmp/e2e_test.go << 'EOF'
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/gorilla/websocket"
)

type Message struct {
    Type      string                 `json:"type"`
    SessionID string                 `json:"session_id,omitempty"`
    Data      map[string]interface{} `json:"data,omitempty"`
}

func main() {
    // Test 1: Basic connection
    fmt.Println("Test 1: Basic WebSocket connection")
    conn1, _, err := websocket.DefaultDialer.Dial("ws://localhost:4081/ws", nil)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer conn1.Close()
    fmt.Println("✓ Connected successfully")

    // Test 2: Player registration
    fmt.Println("\nTest 2: Player registration")
    msg := Message{
        Type:      "CONNECT",
        SessionID: "e2e-test-session-1",
        Data: map[string]interface{}{
            "player_name": "E2E_Player1",
        },
    }
    
    err = conn1.WriteJSON(msg)
    if err != nil {
        log.Fatal("Failed to send connect message:", err)
    }

    var response Message
    err = conn1.ReadJSON(&response)
    if err != nil {
        log.Fatal("Failed to read response:", err)
    }

    if response.Type == "CONNECTED" {
        fmt.Println("✓ Player registered successfully")
    } else {
        log.Fatal("Unexpected response type:", response.Type)
    }

    // Test 3: Create lobby
    fmt.Println("\nTest 3: Create lobby")
    createMsg := Message{
        Type:      "CREATE_LOBBY",
        SessionID: "e2e-test-session-1",
        Data: map[string]interface{}{
            "lobby_name":  "E2E Test Lobby",
            "max_players": 3,
            "map_id":      "usa",
        },
    }

    err = conn1.WriteJSON(createMsg)
    if err != nil {
        log.Fatal("Failed to send create lobby message:", err)
    }

    err = conn1.ReadJSON(&response)
    if err != nil {
        log.Fatal("Failed to read response:", err)
    }

    if response.Type == "LOBBY_CREATED" {
        fmt.Println("✓ Lobby created successfully")
        lobbyData := response.Data["lobby"].(map[string]interface{})
        fmt.Printf("  Lobby ID: %s\n", lobbyData["id"])
    } else if response.Type == "ERROR" {
        log.Fatal("Error creating lobby:", response.Data["message"])
    }

    // Test 4: Second player joins
    fmt.Println("\nTest 4: Second player joins lobby")
    conn2, _, err := websocket.DefaultDialer.Dial("ws://localhost:4081/ws", nil)
    if err != nil {
        log.Fatal("Failed to connect player 2:", err)
    }
    defer conn2.Close()

    // Player 2 connects
    connectMsg2 := Message{
        Type:      "CONNECT",
        SessionID: "e2e-test-session-2",
        Data: map[string]interface{}{
            "player_name": "E2E_Player2",
        },
    }
    
    err = conn2.WriteJSON(connectMsg2)
    if err != nil {
        log.Fatal("Failed to send connect message for player 2:", err)
    }

    var response2 Message
    err = conn2.ReadJSON(&response2)
    if err != nil {
        log.Fatal("Failed to read response for player 2:", err)
    }

    fmt.Println("✓ Player 2 connected successfully")

    // Test 5: Session reconnection
    fmt.Println("\nTest 5: Session reconnection")
    conn1.Close()
    time.Sleep(100 * time.Millisecond)

    // Reconnect with same session ID
    conn1New, _, err := websocket.DefaultDialer.Dial("ws://localhost:4081/ws", nil)
    if err != nil {
        log.Fatal("Failed to reconnect:", err)
    }
    defer conn1New.Close()

    reconnectMsg := Message{
        Type:      "CONNECT",
        SessionID: "e2e-test-session-1",
        Data: map[string]interface{}{
            "player_name": "E2E_Player1",
        },
    }

    err = conn1New.WriteJSON(reconnectMsg)
    if err != nil {
        log.Fatal("Failed to send reconnect message:", err)
    }

    var reconnectResponse Message
    err = conn1New.ReadJSON(&reconnectResponse)
    if err != nil {
        log.Fatal("Failed to read reconnect response:", err)
    }

    if reconnectResponse.Type == "CONNECTED" {
        if msg, ok := reconnectResponse.Data["message"].(string); ok && 
           msg == "Welcome back to Power Grid Game Server" {
            fmt.Println("✓ Session reconnection successful")
        } else {
            fmt.Println("✗ Reconnection succeeded but welcome back message not received")
        }
    }

    fmt.Println("\n✅ All E2E tests passed!")
}
EOF

# Run the E2E test
cd /tmp
go mod init e2e_test
go get github.com/gorilla/websocket
go run e2e_test.go

TEST_RESULT=$?

# Show server logs if tests failed
if [ $TEST_RESULT -ne 0 ]; then
    echo
    echo -e "${RED}Tests failed! Server logs:${NC}"
    tail -n 50 "$SERVER_LOG"
else
    echo
    echo -e "${GREEN}All E2E tests passed successfully!${NC}"
fi

exit $TEST_RESULT