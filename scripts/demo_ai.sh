#!/bin/bash

# Demo script to showcase AI clients with colored logging
# Usage: ./scripts/demo_ai.sh

set -e

echo "🎮 Power Grid AI Demo"
echo "====================="
echo ""
echo "This demo will:"
echo "1. Start the Power Grid server with colored logging"
echo "2. Launch 4 AI clients with different strategies"
echo "3. Show the game in action with colored logs"
echo ""

# Change to go_server directory
cd "$(dirname "$0")/../"

# Check if server binary exists, build if not
if [[ ! -f "powergrid_server" ]]; then
    echo "🔨 Building server..."
    make build
fi

# Build AI clients
if [[ ! -f "cmd/ai_client/ai_client" ]]; then
    echo "🔨 Building AI client..."
    make build-ai
fi

echo "🚀 Starting server with colored logging..."

# Start server in background with colored logging
./powergrid_server --log-level=info --show-caller=false &
SERVER_PID=$!

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping demo..."
    if kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID"
    fi
    # Kill any remaining AI clients
    pkill -f "ai_client" || true
    echo "✅ Demo stopped"
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Wait for server to start
echo "⏳ Waiting for server to start..."
sleep 3

echo ""
echo "🤖 Launching AI clients with different strategies..."
echo "📊 Watch the colored logs to see each AI's decision-making:"
echo "   🔴 Aggressive AI - High bidding, risky moves"
echo "   🔵 Conservative AI - Careful spending, safe moves" 
echo "   🟢 Balanced AI - Mix of strategies"
echo "   🟡 Random AI - Random valid moves"
echo ""

# Launch AI clients with staggered timing
./scripts/launch_ai_clients.sh -n 4 -t "aggressive,conservative,balanced,random" -d 2s -l info &

echo "🎮 Game is now running!"
echo "📊 Monitor the colored logs to see AI decision-making"
echo "⌨️  Press Ctrl+C to stop the demo"
echo ""

# Wait for user to stop
wait