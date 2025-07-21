#!/bin/bash

# Launch AI clients for testing and simulation
# Usage: ./scripts/launch_ai_clients.sh [options]

set -e

# Default values
SERVER_URL="ws://localhost:4080/ws"
NUM_PLAYERS=4
STRATEGIES="balanced,aggressive,conservative,random"
THINK_TIME="1s"
LOG_LEVEL="info"
AUTO_PLAY=true
GAME_ID=""

# Colors for different strategies
declare -A STRATEGY_COLORS=(
    ["aggressive"]="red"
    ["conservative"]="blue"
    ["balanced"]="green"
    ["random"]="yellow"
    ["smart"]="purple"
    ["experimental"]="cyan"
)

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -s, --server URL        WebSocket server URL (default: $SERVER_URL)"
    echo "  -n, --players NUM       Number of AI players 2-6 (default: $NUM_PLAYERS)"
    echo "  -t, --strategies LIST   Comma-separated strategies (default: $STRATEGIES)"
    echo "  -d, --think-time TIME   Think time between moves (default: $THINK_TIME)"
    echo "  -l, --log-level LEVEL   Log level: debug,info,warn,error (default: $LOG_LEVEL)"
    echo "  -g, --game-id ID        Join specific game ID (creates new if empty)"
    echo "  -i, --interactive       Enable interactive mode"
    echo "  -q, --quiet             Reduce logging output"
    echo "  -h, --help              Show this help"
    echo ""
    echo "Strategies:"
    echo "  aggressive    - High bidding, quick expansion, risky moves"
    echo "  conservative  - Low bidding, careful spending, safe moves"
    echo "  balanced      - Mix of aggressive and conservative"
    echo "  random        - Random valid moves for testing"
    echo ""
    echo "Examples:"
    echo "  $0                                    # 4 players with mixed strategies"
    echo "  $0 -n 2 -t aggressive,conservative   # 2 players, specific strategies"
    echo "  $0 -g abc123 -n 3                    # Join game abc123 with 3 players"
    echo "  $0 -i -n 1                           # Single interactive AI"
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
        -d|--think-time)
            THINK_TIME="$2"
            shift 2
            ;;
        -l|--log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        -g|--game-id)
            GAME_ID="$2"
            shift 2
            ;;
        -i|--interactive)
            AUTO_PLAY=false
            shift
            ;;
        -q|--quiet)
            LOG_LEVEL="warn"
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

# Convert strategies to array
IFS=',' read -ra STRATEGY_ARRAY <<< "$STRATEGIES"

echo "🤖 Launching $NUM_PLAYERS AI clients..."
echo "📡 Server: $SERVER_URL"
echo "🧠 Strategies: $STRATEGIES"
echo "⏱️  Think time: $THINK_TIME"
echo "📊 Log level: $LOG_LEVEL"
if [[ -n "$GAME_ID" ]]; then
    echo "🎮 Joining game: $GAME_ID"
else
    echo "🎮 Creating new game"
fi
echo ""

# Change to go_server directory
cd "$(dirname "$0")/../"

# Build AI client if it doesn't exist
if [[ ! -f "cmd/ai_client/ai_client" ]]; then
    echo "🔨 Building AI client..."
    go build -o cmd/ai_client/ai_client ./cmd/ai_client/
fi

# Array to store process IDs
declare -a PIDS=()

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping AI clients..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
        fi
    done
    wait
    echo "✅ All AI clients stopped"
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Launch AI clients
for ((i=0; i<NUM_PLAYERS; i++)); do
    # Select strategy
    strategy_index=$((i % ${#STRATEGY_ARRAY[@]}))
    strategy="${STRATEGY_ARRAY[$strategy_index]}"
    
    # Generate player name and color
    player_name="AI_${strategy}_$(date +%s)_${i}"
    player_color="${STRATEGY_COLORS[$strategy]:-white}"
    
    # Build command
    cmd="./cmd/ai_client/ai_client"
    cmd="$cmd --server=\"$SERVER_URL\""
    cmd="$cmd --strategy=\"$strategy\""
    cmd="$cmd --name=\"$player_name\""
    cmd="$cmd --color=\"$player_color\""
    cmd="$cmd --think-time=\"$THINK_TIME\""
    cmd="$cmd --log-level=\"$LOG_LEVEL\""
    
    if [[ -n "$GAME_ID" ]]; then
        cmd="$cmd --game=\"$GAME_ID\""
    fi
    
    if [[ "$AUTO_PLAY" == "false" ]]; then
        cmd="$cmd --interactive"
    fi
    
    echo "🚀 Starting $strategy AI ($player_color)..."
    
    # Launch in background
    eval "$cmd" &
    PIDS+=($!)
    
    # Stagger launches to avoid connection issues
    sleep 0.5
done

echo ""
echo "✅ All $NUM_PLAYERS AI clients launched"
echo "🎮 Game should start automatically once all players are ready"
echo "📊 Monitor logs for game progress"
echo "⌨️  Press Ctrl+C to stop all clients"
echo ""

# Wait for all processes
wait