#!/bin/bash

# Run Power Grid AI simulations
# Usage: ./scripts/run_simulation.sh [options]

set -e

# Default values
SERVER_URL="ws://localhost:4080/ws"
NUM_PLAYERS=4
STRATEGIES="balanced,aggressive,conservative,random"
ITERATIONS=10
THINK_TIME="500ms"
LOG_LEVEL="info"
CONCURRENT=false
MAX_GAME_TIME="30m"
REPORT_STATS=true

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -s, --server URL        WebSocket server URL (default: $SERVER_URL)"
    echo "  -n, --players NUM       Number of AI players 2-6 (default: $NUM_PLAYERS)"
    echo "  -t, --strategies LIST   Comma-separated strategies (default: $STRATEGIES)"
    echo "  -i, --iterations NUM    Number of games to simulate (default: $ITERATIONS)"
    echo "  -d, --think-time TIME   AI think time between moves (default: $THINK_TIME)"
    echo "  -l, --log-level LEVEL   Log level: debug,info,warn,error (default: $LOG_LEVEL)"
    echo "  -c, --concurrent        Run games concurrently"
    echo "  -m, --max-time TIME     Maximum time per game (default: $MAX_GAME_TIME)"
    echo "  -q, --quiet             Reduce logging output"
    echo "  --no-stats              Disable statistics report"
    echo "  -h, --help              Show this help"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run 10 games with 4 players"
    echo "  $0 -i 50 -c                          # Run 50 games concurrently"
    echo "  $0 -n 6 -t aggressive,balanced       # 6 players, specific strategies"
    echo "  $0 -i 100 -q                         # 100 games, minimal output"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--server)
            SERVER_URL="$2"
            shift 2
            ;;
        -n|--players)
            NUM_PLAYERS="$2"
            shift 2
            ;;
        -t|--strategies)
            STRATEGIES="$2"
            shift 2
            ;;
        -i|--iterations)
            ITERATIONS="$2"
            shift 2
            ;;
        -d|--think-time)
            THINK_TIME="$2"
            shift 2
            ;;
        -l|--log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        -c|--concurrent)
            CONCURRENT=true
            shift
            ;;
        -m|--max-time)
            MAX_GAME_TIME="$2"
            shift 2
            ;;
        -q|--quiet)
            LOG_LEVEL="warn"
            shift
            ;;
        --no-stats)
            REPORT_STATS=false
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate parameters
if [[ $NUM_PLAYERS -lt 2 || $NUM_PLAYERS -gt 6 ]]; then
    echo "Error: Number of players must be between 2 and 6"
    exit 1
fi

if [[ $ITERATIONS -lt 1 ]]; then
    echo "Error: Iterations must be at least 1"
    exit 1
fi

echo "🎯 Starting Power Grid AI Simulation"
echo "📊 Games: $ITERATIONS"
echo "👥 Players: $NUM_PLAYERS"
echo "🧠 Strategies: $STRATEGIES"
echo "⏱️  Think time: $THINK_TIME"
echo "📡 Server: $SERVER_URL"
echo "🔄 Concurrent: $CONCURRENT"
echo "⏰ Max game time: $MAX_GAME_TIME"
echo ""

# Change to go_server directory
cd "$(dirname "$0")/../"

# Check if server is running
echo "🔍 Checking server availability..."
if ! curl -s "$SERVER_URL" >/dev/null 2>&1; then
    echo "⚠️  Server not reachable at $SERVER_URL"
    echo "💡 Make sure the Power Grid server is running:"
    echo "   ./scripts/launch_server.sh"
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Build simulator if it doesn't exist
if [[ ! -f "cmd/simulator/simulator" ]]; then
    echo "🔨 Building simulator..."
    go build -o cmd/simulator/simulator ./cmd/simulator/
fi

# Create logs directory
mkdir -p logs

# Generate log file name
timestamp=$(date +"%Y%m%d_%H%M%S")
log_file="logs/simulation_${timestamp}.log"

echo "📝 Logging to: $log_file"
echo ""

# Build simulator command
cmd="./cmd/simulator/simulator"
cmd="$cmd --server=\"$SERVER_URL\""
cmd="$cmd --players=$NUM_PLAYERS"
cmd="$cmd --strategies=\"$STRATEGIES\""
cmd="$cmd --iterations=$ITERATIONS"
cmd="$cmd --think-time=\"$THINK_TIME\""
cmd="$cmd --log-level=\"$LOG_LEVEL\""
cmd="$cmd --max-game-time=\"$MAX_GAME_TIME\""

if [[ "$CONCURRENT" == "true" ]]; then
    cmd="$cmd --concurrent"
fi

if [[ "$REPORT_STATS" == "false" ]]; then
    cmd="$cmd --stats=false"
fi

echo "🚀 Starting simulation..."
echo "Command: $cmd"
echo ""

# Run simulation with logging
eval "$cmd" 2>&1 | tee "$log_file"

echo ""
echo "✅ Simulation completed"
echo "📄 Full log available at: $log_file"

# Show summary if available
if [[ -f "$log_file" ]]; then
    echo ""
    echo "📊 Quick Summary:"
    grep -E "(Total Games|Completed|Failed|Success Rate|Average Game Duration)" "$log_file" | tail -5
fi