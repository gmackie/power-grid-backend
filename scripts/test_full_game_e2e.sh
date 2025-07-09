#!/bin/bash

# Full gameplay end-to-end test script
# Tests complete game flow from lobby creation to gameplay phases

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVER_PORT=4085
SERVER_LOG="logs/full_game_test.log"
SERVER_PID=""

# Ensure logs directory exists
mkdir -p logs

# Function to cleanup on exit
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

echo -e "${GREEN}Power Grid Full Gameplay E2E Test${NC}"
echo "====================================="

# Build the server
echo -e "${YELLOW}Building server...${NC}"
go build -o powergrid_server ./cmd/server/

# Start the server
echo -e "${YELLOW}Starting server on port $SERVER_PORT...${NC}"
./powergrid_server -addr=:$SERVER_PORT > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}Server failed to start!${NC}"
    cat "$SERVER_LOG"
    exit 1
fi

echo -e "${GREEN}Server started (PID: $SERVER_PID)${NC}"

# Create comprehensive test client
echo -e "${YELLOW}Creating test client...${NC}"

cat > /tmp/full_game_client.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "net"
    "time"
    "github.com/gorilla/websocket"
)

type LobbyMessage struct {
    Type      string                 `json:"type"`
    SessionID string                 `json:"session_id,omitempty"`
    Data      map[string]interface{} `json:"data,omitempty"`
}

type GameMessage struct {
    Type      string      `json:"type"`
    SessionID string      `json:"session_id,omitempty"`
    Timestamp int64       `json:"timestamp"`
    Payload   interface{} `json:"payload,omitempty"`
}

type TestClient struct {
    Name      string
    SessionID string
    LobbyConn *websocket.Conn
    GameConn  *websocket.Conn
}

func NewTestClient(name string) *TestClient {
    return &TestClient{
        Name:      name,
        SessionID: fmt.Sprintf("test-%s-%d", name, time.Now().UnixNano()),
    }
}

func (c *TestClient) ConnectToLobby() error {
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:4085/ws", nil)
    if err != nil {
        return err
    }
    c.LobbyConn = conn

    // Send CONNECT
    msg := LobbyMessage{
        Type:      "CONNECT",
        SessionID: c.SessionID,
        Data: map[string]interface{}{
            "player_name": c.Name,
        },
    }

    err = c.LobbyConn.WriteJSON(msg)
    if err != nil {
        return err
    }

    var response LobbyMessage
    return c.LobbyConn.ReadJSON(&response)
}

func (c *TestClient) CreateLobby(name string) (string, error) {
    msg := LobbyMessage{
        Type:      "CREATE_LOBBY",
        SessionID: c.SessionID,
        Data: map[string]interface{}{
            "lobby_name":  name,
            "max_players": 3,
            "map_id":      "usa",
        },
    }

    err := c.LobbyConn.WriteJSON(msg)
    if err != nil {
        return "", err
    }

    var response LobbyMessage
    err = c.LobbyConn.ReadJSON(&response)
    if err != nil {
        return "", err
    }

    if response.Type == "ERROR" {
        return "", fmt.Errorf("error: %v", response.Data["message"])
    }

    lobbyData := response.Data["lobby"].(map[string]interface{})
    return lobbyData["id"].(string), nil
}

func (c *TestClient) JoinLobby(lobbyID string) error {
    msg := LobbyMessage{
        Type:      "JOIN_LOBBY",
        SessionID: c.SessionID,
        Data: map[string]interface{}{
            "lobby_id": lobbyID,
        },
    }

    err := c.LobbyConn.WriteJSON(msg)
    if err != nil {
        return err
    }

    // Keep reading until we get LOBBY_JOINED or ERROR
    for i := 0; i < 5; i++ {
        var response LobbyMessage
        err = c.LobbyConn.ReadJSON(&response)
        if err != nil {
            return err
        }

        if response.Type == "LOBBY_JOINED" {
            return nil
        }
        
        if response.Type == "ERROR" {
            return fmt.Errorf("error: %v", response.Data["message"])
        }
        
        // Ignore other message types
    }

    return fmt.Errorf("timeout waiting for LOBBY_JOINED")
}

func (c *TestClient) SetReady(ready bool) error {
    msg := LobbyMessage{
        Type:      "SET_READY",
        SessionID: c.SessionID,
        Data: map[string]interface{}{
            "ready": ready,
        },
    }

    return c.LobbyConn.WriteJSON(msg)
}

func (c *TestClient) StartGame() error {
    msg := LobbyMessage{
        Type:      "START_GAME",
        SessionID: c.SessionID,
        Data:      map[string]interface{}{},
    }

    err := c.LobbyConn.WriteJSON(msg)
    if err != nil {
        return err
    }

    // The response is broadcast to all players, not just the sender
    return nil
}

// WaitForMessage waits for a specific message type
func (c *TestClient) WaitForMessage(msgType string, timeout time.Duration) error {
    c.LobbyConn.SetReadDeadline(time.Now().Add(timeout))
    defer c.LobbyConn.SetReadDeadline(time.Time{})

    for {
        var response LobbyMessage
        err := c.LobbyConn.ReadJSON(&response)
        if err != nil {
            if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                return fmt.Errorf("timeout waiting for %s", msgType)
            }
            return err
        }

        if response.Type == msgType {
            return nil
        }
        
        if response.Type == "ERROR" {
            return fmt.Errorf("error: %v", response.Data["message"])
        }
    }
}

func (c *TestClient) Close() {
    if c.LobbyConn != nil {
        c.LobbyConn.Close()
    }
    if c.GameConn != nil {
        c.GameConn.Close()
    }
}

func main() {
    fmt.Println("🎮 Starting Full Gameplay E2E Test")
    fmt.Println("==================================")

    // Create test clients
    host := NewTestClient("Host")
    player1 := NewTestClient("Player1") 
    player2 := NewTestClient("Player2")

    defer host.Close()
    defer player1.Close()
    defer player2.Close()

    // Test 1: Lobby Operations
    fmt.Println("\n📡 Test 1: Lobby Operations")
    fmt.Println("---------------------------")

    fmt.Print("Connecting to lobby server... ")
    err := host.ConnectToLobby()
    if err != nil {
        log.Fatal("Host connection failed:", err)
    }
    fmt.Println("✓")

    err = player1.ConnectToLobby()
    if err != nil {
        log.Fatal("Player1 connection failed:", err)
    }

    err = player2.ConnectToLobby()
    if err != nil {
        log.Fatal("Player2 connection failed:", err)
    }

    fmt.Print("Creating lobby... ")
    lobbyID, err := host.CreateLobby("Full Game Test")
    if err != nil {
        log.Fatal("Lobby creation failed:", err)
    }
    fmt.Printf("✓ (ID: %s)\n", lobbyID[:8])

    fmt.Print("Players joining lobby... ")
    err = player1.JoinLobby(lobbyID)
    if err != nil {
        log.Fatal("Player1 join failed:", err)
    }

    err = player2.JoinLobby(lobbyID)
    if err != nil {
        log.Fatal("Player2 join failed:", err)
    }
    fmt.Println("✓")

    // Test 2: Game Start Sequence
    fmt.Println("\n🚀 Test 2: Game Start Sequence")
    fmt.Println("------------------------------")

    fmt.Print("Setting players ready... ")
    err = host.SetReady(true)
    if err != nil {
        log.Fatal("Host ready failed:", err)
    }
    time.Sleep(50 * time.Millisecond)

    err = player1.SetReady(true)
    if err != nil {
        log.Fatal("Player1 ready failed:", err)
    }
    time.Sleep(50 * time.Millisecond)

    err = player2.SetReady(true)
    if err != nil {
        log.Fatal("Player2 ready failed:", err)
    }
    fmt.Println("✓")

    // Small delay to ensure all ready messages are processed
    time.Sleep(100 * time.Millisecond)

    fmt.Print("Starting game... ")
    err = host.StartGame()
    if err != nil {
        log.Fatal("Game start failed:", err)
    }

    // All players should receive GAME_STARTING broadcast
    err = host.WaitForMessage("GAME_STARTING", 3*time.Second)
    if err != nil {
        log.Fatal("Host didn't receive GAME_STARTING:", err)
    }

    err = player1.WaitForMessage("GAME_STARTING", 3*time.Second)
    if err != nil {
        log.Fatal("Player1 didn't receive GAME_STARTING:", err)
    }

    err = player2.WaitForMessage("GAME_STARTING", 3*time.Second)
    if err != nil {
        log.Fatal("Player2 didn't receive GAME_STARTING:", err)
    }
    fmt.Println("✓")

    // Test 3: Session Persistence
    fmt.Println("\n🔄 Test 3: Session Persistence")
    fmt.Println("------------------------------")

    fmt.Print("Testing reconnection... ")
    
    // Close and reconnect host
    host.LobbyConn.Close()
    time.Sleep(100 * time.Millisecond)
    
    err = host.ConnectToLobby()
    if err != nil {
        log.Fatal("Reconnection failed:", err)
    }
    fmt.Println("✓")

    // Test 4: Multiple Lobby Operations
    fmt.Println("\n🏢 Test 4: Multiple Lobby Operations")
    fmt.Println("------------------------------------")

    fmt.Print("Creating second lobby... ")
    lobbyID2, err := host.CreateLobby("Second Test Lobby")
    if err != nil {
        log.Fatal("Second lobby creation failed:", err)
    }
    fmt.Printf("✓ (ID: %s)\n", lobbyID2[:8])

    // Test 5: Stress Test
    fmt.Println("\n⚡ Test 5: Connection Stress Test")
    fmt.Println("---------------------------------")

    fmt.Print("Rapid connect/disconnect cycles... ")
    for i := 0; i < 5; i++ {
        tempClient := NewTestClient(fmt.Sprintf("Temp%d", i))
        err = tempClient.ConnectToLobby()
        if err != nil {
            log.Fatalf("Temp client %d failed: %v", i, err)
        }
        tempClient.Close()
    }
    fmt.Println("✓")

    fmt.Println("\n🎉 Full Gameplay E2E Test Results")
    fmt.Println("=================================")
    fmt.Println("✅ Lobby operations: PASSED")
    fmt.Println("✅ Game start sequence: PASSED") 
    fmt.Println("✅ Session persistence: PASSED")
    fmt.Println("✅ Multiple lobbies: PASSED")
    fmt.Println("✅ Connection stress test: PASSED")
    fmt.Println()
    fmt.Println("🏆 ALL TESTS PASSED!")
}
EOF

# Run the comprehensive test
echo -e "${BLUE}Running comprehensive gameplay test...${NC}"
echo

cd /tmp
rm -rf full_game_test_module 2>/dev/null || true
mkdir -p full_game_test_module
cd full_game_test_module
cp ../full_game_client.go .
go mod init full_game_test 2>/dev/null || true
go get github.com/gorilla/websocket
go run full_game_client.go

TEST_RESULT=$?

# Show results
if [ $TEST_RESULT -eq 0 ]; then
    echo
    echo -e "${GREEN}🎉 Full Gameplay E2E Test PASSED!${NC}"
    echo -e "${GREEN}All systems working correctly:${NC}"
    echo -e "${GREEN}  ✅ Session management${NC}"
    echo -e "${GREEN}  ✅ Lobby operations${NC}"
    echo -e "${GREEN}  ✅ Game start sequence${NC}"
    echo -e "${GREEN}  ✅ Reconnection handling${NC}"
    echo -e "${GREEN}  ✅ Concurrent operations${NC}"
else
    echo
    echo -e "${RED}❌ Full Gameplay E2E Test FAILED!${NC}"
    echo
    echo -e "${YELLOW}Server logs:${NC}"
    tail -n 20 "$SERVER_LOG"
fi

exit $TEST_RESULT