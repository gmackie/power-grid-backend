#!/bin/bash

# End-to-end test script for Power Grid Game Server

set -e

echo "🚀 Starting Power Grid Game Server E2E Test..."

# Kill any existing processes on port 4080
echo "🧹 Cleaning up existing processes..."
lsof -ti:4080 | xargs kill -9 2>/dev/null || true
sleep 1

# Build the server
echo "🔨 Building server..."
go build -o powergrid_server cmd/server/main.go

# Start the server in the background
echo "🚀 Starting server..."
./powergrid_server &
SERVER_PID=$!

# Wait for server to start
echo "⏳ Waiting for server to start..."
sleep 2

# Check if server is running
if ! curl -s http://localhost:4080/health > /dev/null; then
    echo "❌ Server failed to start"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo "✅ Server is running on :4080"

# Function to cleanup on exit
cleanup() {
    echo "🧹 Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    lsof -ti:4080 | xargs kill -9 2>/dev/null || true
    exit 0
}

# Set trap to cleanup on script exit
trap cleanup EXIT INT TERM

# Run the test client
echo "🧪 Starting test client..."
echo "📝 Test logs:"
echo "=============="

go run test_client.go localhost:4080 &
CLIENT_PID=$!

# Wait for test to complete or timeout
sleep 10

echo "=============="
echo "✅ Test completed"

# Kill the client if it's still running
kill $CLIENT_PID 2>/dev/null || true

echo "🎉 E2E test finished. Check the logs above for results."
echo "💡 Server was running on http://localhost:4080"
echo "💡 WebSocket endpoints:"
echo "   - /ws (lobby)"
echo "   - /game (protocol-based game)"