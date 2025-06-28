# STRUCTURE

```text

/cmd
    /server
        main.go            // Entry point for server application
/internal
    /app
        server.go          // Server application
        handler.go         // Request handlers
    /game
        game.go            // Game state and logic
        map.go             // Map representation
        city.go            // City representation
        connection.go      // Connection between cities
        player.go          // Player representation
        powerplant.go      // Power plant representation
        resource.go        // Resource representation
        market.go          // Resource market
        auction.go         // Power plant auction
    /network
        session.go         // Player session management
        message.go         // Message structure
        room.go            // Game room management
    /config
        config.go          // Server configuration
    /utils
        utils.go           // Utility functions
/pkg
    /protocol              // Protocol definitions
        messages.go        // Message definitions
        serialization.go   // Message serialization
/test
    /integration           // Integration tests
    /unit                  // Unit tests
/api
    /rpc                   // RPC definitions
        game.go            // Game related RPCs
        lobby.go           // Lobby related RPCs
        user.go            // User related RPCs
/configs                   // Configuration files
/scripts                   // Build and deployment scripts
go.mod                     // Go module definition
go.sum                     // Go module checksums
```
