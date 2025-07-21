# AI Clients and Simulation System

This directory contains robust AI clients for Power Grid gameplay testing and simulation with colored logging support and advanced decision-making capabilities.

## Features

### AI Client System
- **Multiple Strategies**: Aggressive, Conservative, Balanced, and Random AI players
- **Sophisticated Decision Engine**: Advanced evaluation of power plants, resources, and city expansion
- **Game State Tracking**: Historical analysis and opponent behavior prediction
- **Colored Logging**: Each AI strategy has distinct colored output for easy monitoring
- **Configurable Behavior**: Adjustable think times, auto-play mode, and interactive mode
- **WebSocket Communication**: Full integration with the Power Grid server protocol

### Simulation Framework
- **Automated Testing**: Run multiple games with AI players for balance testing
- **Game Monitoring**: Real-time tracking of game progress and automatic completion detection
- **Performance Analysis**: Track game completion rates, duration, and strategy effectiveness
- **Concurrent Execution**: Support for running multiple games simultaneously
- **Comprehensive Statistics**: Detailed reporting with insights and trend analysis
- **JSON Export**: Save simulation results for further analysis

## Quick Start

### Demo Mode
```bash
# Run interactive demo with 4 AI players
make ai-demo

# Or manually:
./scripts/demo_ai.sh
```

### Launch AI Clients
```bash
# Launch 4 AI players with mixed strategies
./scripts/launch_ai_clients.sh

# Launch specific number with custom strategies
./scripts/launch_ai_clients.sh -n 6 -t "aggressive,balanced,conservative"

# Join existing game
./scripts/launch_ai_clients.sh -g <game_id> -n 2
```

### Run Simulations
```bash
# Run 10 games with default settings
make simulation

# Run custom simulation
./scripts/run_simulation.sh -i 50 -n 6 -c

# Detailed simulation with specific strategies
./scripts/run_simulation.sh -i 100 -t "aggressive,balanced" -d 100ms
```

## AI Strategies

### Aggressive Strategy 🔴
- **Bidding**: Bids high on power plants (+10 above minimum)
- **Resources**: Buys double the required resources
- **Building**: Expands to new regions quickly
- **Risk**: High risk, high reward approach

### Conservative Strategy 🔵  
- **Bidding**: Minimal bidding (+1 above minimum)
- **Resources**: Buys only what's needed
- **Building**: Only builds with plenty of money (>30)
- **Risk**: Low risk, steady progress

### Balanced Strategy 🟢
- **Bidding**: Moderate bidding strategy
- **Resources**: Balanced resource purchasing
- **Building**: Calculated expansion
- **Risk**: Balanced risk/reward

### Random Strategy 🟡
- **Bidding**: Random valid bids for testing
- **Resources**: Random resource purchases
- **Building**: Random valid moves
- **Risk**: Unpredictable for chaos testing

## Colored Logging System

The system uses ANSI color codes to distinguish different components:

- **🟢 SERVER**: Server operations and game state
- **🔵 CLIENT**: Client connections and messages  
- **🟣 GAME**: Game logic and phase transitions
- **🔴 AI:AGGRESSIVE**: Aggressive AI strategy
- **🔵 AI:CONSERVATIVE**: Conservative AI strategy
- **🟢 AI:BALANCED**: Balanced AI strategy
- **🟡 AI:RANDOM**: Random AI strategy
- **🟡 TEST**: Testing and simulation framework

### Log Levels
- `DEBUG`: Detailed debugging information
- `INFO`: General operational information
- `WARN`: Warning messages
- `ERROR`: Error conditions
- `FATAL`: Fatal errors that cause shutdown

## Configuration

### AI Client Options
```bash
./cmd/ai_client/ai_client [options]
  -server string         WebSocket server URL
  -strategy string       AI strategy (aggressive/conservative/balanced/random)
  -name string          Player name (auto-generated if empty)
  -color string         Player color (auto-generated if empty)
  -game string          Game ID to join (creates new if empty)
  -think-time duration  Time between moves (default 1s)
  -log-level string     Log level (debug/info/warn/error)
  -interactive          Enable interactive mode
  -auto-play           Enable automatic move making (default true)
```

### Simulation Options
```bash
./cmd/simulator/simulator [options]
  -players int           Number of AI players 2-6 (default 4)
  -iterations int        Number of games to simulate (default 10)
  -strategies string     Comma-separated strategies
  -concurrent           Run games concurrently
  -think-time duration  AI think time (default 500ms)
  -max-game-time duration Maximum time per game (default 30m)
  -stats               Generate statistics report (default true)
```

## Architecture

### Decision-Making Components

#### Game State Tracker
Maintains game history and analyzes patterns:
- **Historical Data**: Tracks up to 100 game state snapshots
- **Player Analysis**: Monitors opponent behavior and infers strategies
- **Market Trends**: Tracks resource prices and supply/demand patterns
- **Phase Statistics**: Analyzes game flow and timing

#### Decision Engine
Sophisticated move evaluation system:
- **Power Plant Evaluation**: Scores plants based on capacity, efficiency, resource availability, and synergy
- **Resource Optimization**: Prioritizes purchases based on scarcity and price trends
- **City Selection**: Evaluates strategic value, connection costs, and regional control
- **Adaptive Behavior**: Adjusts strategy based on game phase and player position

#### Game Monitor
Real-time game tracking:
- **Completion Detection**: Monitors win conditions and game end states
- **Statistics Collection**: Tracks turns, phases, actions, and resource usage
- **Performance Metrics**: Measures game duration and player effectiveness

### Statistics and Reporting

The simulation framework generates comprehensive reports including:

1. **Overall Statistics**
   - Total games, completion rate, average duration
   - Success/failure analysis

2. **Strategy Performance**
   - Win rates, average cities, resource usage
   - Comparative analysis between strategies

3. **Phase Analysis**
   - Phase frequency and duration
   - Action patterns per phase

4. **Resource Trends**
   - Total consumption by type
   - Peak demand analysis
   - Market volatility

5. **Key Insights**
   - Dominant strategies
   - Performance anomalies
   - Optimization recommendations

## Development

### Adding New Strategies

1. **Create Strategy**: Implement the `Strategy` interface in `internal/ai/strategy.go`
```go
type NewStrategy struct{}

func (s *NewStrategy) GetName() string {
    return "NewStrategy"
}

func (s *NewStrategy) GetDescription() string {
    return "Description of strategy behavior"
}

func (s *NewStrategy) GetMove(gameState *protocol.GameStatePayload, playerID string) *protocol.Message {
    // Basic strategy logic
    // For advanced logic, use DecisionEngine
}
```

2. **Register Strategy**: Add to `CreateStrategy()` function
3. **Add Color**: Define color in `getColorForStrategy()`
4. **Test**: Use with AI clients and simulations

### Enhancing Decision Logic

To improve AI decision-making:

1. **Extend Evaluation Functions**
```go
// In decision_engine.go
func (e *DecisionEngine) evaluateCustomFactor(state *protocol.GameStatePayload) float64 {
    // Add new evaluation criteria
}
```

2. **Add Historical Analysis**
```go
// In game_state_tracker.go
func (t *GameStateTracker) analyzePattern() PatternResult {
    // Implement pattern recognition
}
```

3. **Implement Learning**
```go
// Create new learning module
type LearningEngine struct {
    history []GameResult
    weights map[string]float64
}
```

### Monitoring and Debugging

#### Real-time Monitoring
```bash
# Monitor server logs
tail -f logs/server.log

# Monitor all logs with colors
./scripts/view_logs.sh

# Monitor specific AI strategy
grep "AI:AGGRESSIVE" logs/*.log
```

#### Game State Analysis
```bash
# Run with debug logging
./cmd/ai_client/ai_client -log-level=debug -strategy=aggressive

# Show caller information
./cmd/ai_client/ai_client -show-caller -log-level=debug
```

### Integration with Testing

The AI system integrates with existing test infrastructure:

```bash
# Run tests with AI clients
make test-e2e-all

# Custom test scenarios
./scripts/test_full_game_e2e.sh -ai 4 -strategies "aggressive,conservative"
```

## Performance Considerations

### Resource Usage
- Each AI client uses minimal CPU when idle
- Memory usage scales with game state complexity
- Network traffic is optimized through WebSocket connections

### Scalability
- Server supports up to 500 concurrent connections
- Recommended: Max 20 concurrent AI clients per server
- Simulation framework can handle 100+ games sequentially

### Optimization Tips
- Use shorter think times (`-d 100ms`) for faster simulations
- Enable concurrent mode (`-c`) for bulk testing
- Reduce log level (`-l warn`) for performance testing

## Troubleshooting

### Common Issues

**Connection Failed**
```bash
# Check server status
curl http://localhost:4080/health

# Start server if needed
make run
```

**Build Errors**
```bash
# Clean and rebuild
make clean
make build-ai-all
```

**Performance Issues**
```bash
# Reduce concurrent connections
./scripts/launch_ai_clients.sh -n 2

# Increase think time
./scripts/launch_ai_clients.sh -d 2s
```

### Debug Mode
```bash
# Enable detailed logging
./cmd/ai_client/ai_client -log-level=debug -show-caller

# Monitor WebSocket traffic
./cmd/ai_client/ai_client -log-level=debug | grep "WebSocket"
```

## Integration Examples

### Continuous Integration
```yaml
# Example GitHub Action
- name: Test AI Gameplay
  run: |
    make build-ai-all
    ./scripts/run_simulation.sh -i 5 -n 4 -q
```

### Load Testing
```bash
# Stress test server
./scripts/run_simulation.sh -i 20 -c -n 6 -d 100ms
```

### Balance Testing
```bash
# Test strategy balance
./scripts/run_simulation.sh -i 100 -t "aggressive,conservative,balanced" -stats
```