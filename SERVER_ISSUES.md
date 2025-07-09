# Power Grid Game Server Issues Summary

## Overview
This document summarizes the server-side issues discovered during React client end-to-end testing with Playwright. These issues prevent successful lobby creation and multiplayer game functionality.

## Primary Issue: WebSocket Connection Management

### Problem Description
The Go server is unable to properly track player connections, resulting in "Player not found" errors when players attempt to create lobbies.

### Root Cause Analysis

1. **Multiple WebSocket Connections Per Client**
   - Each client establishes 2 WebSocket connections instead of 1
   - Server logs show duplicate connection IDs for the same client:
   ```
   [backend] 2025/07/08 06:04:29 New client connected with session ID: d5155811-d4d5-4803-a595-2a2286275467
   [backend] 2025/07/08 06:04:29 New client connected with session ID: 26e4b469-2cd4-41e5-a23c-c893605731e0
   ```

2. **Player Registration Flow**
   - Client sends `CONNECT` message with player name
   - Server responds with `CONNECTED` confirmation
   - Server maintains a `clients` map: `conn -> playerID`
   - When `CREATE_LOBBY` is sent, server cannot find player in the `clients` map

3. **Connection Lifecycle Issues**
   - WebSocket connections close unexpectedly: `websocket: close 1001 (going away)`
   - This happens during navigation between React components
   - The connection that registered the player may not be the same one sending CREATE_LOBBY

### Affected Server Code

**File**: `/go_server/handlers/lobby_handler.go`

Key areas:
- Line 255-257: Player registration in `handleConnect()`
  ```go
  h.mu.Lock()
  h.clients[conn] = playerID
  h.mu.Unlock()
  ```

- Line 269-275: Player lookup in `handleCreateLobby()`
  ```go
  h.mu.Lock()
  playerID, exists := h.clients[conn]
  h.mu.Unlock()
  
  if !exists {
      h.sendErrorMessage(conn, sessionID, "Player not found")
      return
  }
  ```

### Test Results

**Successful Operations:**
- ✅ WebSocket connection establishment
- ✅ Player registration (CONNECT message)
- ✅ Lobby listing (LIST_LOBBIES message)
- ✅ Client receives server responses

**Failed Operations:**
- ❌ Lobby creation (CREATE_LOBBY returns "Player not found")
- ❌ Any operation requiring player lookup after initial connection

## Recommended Fixes

### 1. Connection Deduplication
- Investigate why React client creates multiple WebSocket connections
- Possible causes:
  - React StrictMode in development (double-mounting components)
  - Multiple WebSocket manager instances
  - Auto-reconnection logic creating duplicate connections

### 2. Player Session Management
- Consider using session IDs instead of WebSocket connections as primary key
- Implement player session persistence across reconnections
- Example approach:
  ```go
  type LobbyHandler struct {
      sessions map[string]*PlayerSession  // sessionID -> PlayerSession
      connections map[*websocket.Conn]string  // conn -> sessionID
  }
  ```

### 3. Connection State Tracking
- Add logging to track connection lifecycle:
  - Connection establishment
  - Player registration
  - Connection closure
  - Player lookup attempts

### 4. Client-Side Connection Management
- Ensure single WebSocket instance per client
- Handle reconnection without losing player state
- Consider implementing connection pooling or singleton pattern

## Testing Evidence

### Client Logs Showing Issue
```
[Client]: WebSocket connected
[Client]: Player registered successfully: {message: Welcome to Power Grid Game Server}
[Client]: Received message: CONNECTED {message: Welcome to Power Grid Game Server}
[Client]: WebSocket connected  // <-- Duplicate connection
[Client]: Player registered successfully: {message: Welcome to Power Grid Game Server}
[Client]: Received message: CONNECTED {message: Welcome to Power Grid Game Server}
```

### Server Response to CREATE_LOBBY
```
[Server]: Received message: {"type":"CREATE_LOBBY","data":{"lobby_name":"Test Lobby","player_name":"TestPlayer","max_players":4,"map_id":"usa","password":""}}
[Client]: Unhandled message: ERROR {message: Player not found}
```

## Impact on Testing

These issues prevent:
- Full end-to-end multiplayer testing
- Lobby creation and management testing
- Game phase progression testing
- Multi-player interaction testing

## Additional Observations

1. **React Client Behavior**
   - Client appears to handle navigation correctly
   - UI indicates successful operations despite server errors
   - WebSocket connections close during component transitions

2. **Server Stability**
   - Server remains stable despite connection issues
   - No crashes or panics observed
   - Proper error handling for missing players

## Debugging Recommendations

1. Add detailed logging in `lobby_handler.go`:
   - Log all connection establishments with timestamps
   - Log all player registrations with connection details
   - Log all connection closures
   - Log player lookup attempts with connection info

2. Implement connection tracking middleware:
   - Track active connections
   - Monitor connection lifecycle
   - Detect duplicate connections from same client

3. Test with simplified client:
   - Create minimal WebSocket client without React
   - Verify single connection behavior
   - Test lobby creation flow in isolation

## Success Criteria

The server should:
1. Accept single WebSocket connection per client
2. Maintain player session across component navigation
3. Successfully create lobbies after player registration
4. Handle reconnections without losing player state

## References

- Server code: `/go_server/handlers/lobby_handler.go`
- Client WebSocket manager: `/react_client/src/services/websocket.ts`
- Client store: `/react_client/src/store/gameStore.ts`
- Test logs: Available in Playwright test output