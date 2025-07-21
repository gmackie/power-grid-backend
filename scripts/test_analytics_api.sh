#!/bin/bash

# Test Analytics API endpoints
# Usage: ./scripts/test_analytics_api.sh [base_url]

set -e

BASE_URL="${1:-http://localhost:4080}"
API_URL="$BASE_URL/api"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to make API request and display results
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local description="$3"
    
    echo -e "${BLUE}Testing:${NC} $description"
    echo -e "${YELLOW}$method $API_URL$endpoint${NC}"
    
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$API_URL$endpoint")
    status=$(echo "$response" | tail -n1 | cut -d: -f2)
    body=$(echo "$response" | sed '$d')
    
    if [[ "$status" == "200" ]]; then
        echo -e "${GREEN}✓ Success (200)${NC}"
        echo "$body" | jq . 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Failed ($status)${NC}"
        echo "$body"
    fi
    echo ""
}

echo "🔍 Power Grid Analytics API Test Suite"
echo "======================================"
echo "Testing server at: $BASE_URL"
echo ""

# Test API health
test_endpoint "GET" "/health" "API Health Check"

# Test achievements list
test_endpoint "GET" "/achievements" "List All Achievements"

# Test leaderboard
test_endpoint "GET" "/leaderboard?limit=5" "Top 5 Leaderboard"

# Test game analytics
test_endpoint "GET" "/analytics/games" "Game Analytics Overview"

# Test player endpoints (these might return 404 if no data exists)
echo -e "${BLUE}Testing Player Endpoints:${NC}"
echo "Note: These may return 404 if no player data exists yet"
echo ""

test_endpoint "GET" "/players/TestPlayer" "Player Stats (might not exist)"
test_endpoint "GET" "/players/TestPlayer/achievements" "Player Achievements (might not exist)"
test_endpoint "GET" "/players/TestPlayer/history" "Player Game History (might not exist)"
test_endpoint "GET" "/players/TestPlayer/progress" "Player Progress (might not exist)"

# Test error cases
echo -e "${BLUE}Testing Error Cases:${NC}"
echo ""

test_endpoint "POST" "/leaderboard" "Invalid Method (should fail)"
test_endpoint "GET" "/invalid/endpoint" "Invalid Endpoint (should fail)"

echo -e "${GREEN}Analytics API test completed!${NC}"
echo ""
echo "📊 Available endpoints:"
echo "  GET  $API_URL/health"
echo "  GET  $API_URL/achievements"
echo "  GET  $API_URL/leaderboard?limit=N"
echo "  GET  $API_URL/analytics/games"
echo "  GET  $API_URL/players/{name}"
echo "  GET  $API_URL/players/{name}/achievements"
echo "  GET  $API_URL/players/{name}/history"
echo "  GET  $API_URL/players/{name}/progress"
echo ""
echo "🎮 To generate test data, run some games with AI clients:"
echo "  make ai-demo"
echo "  ./scripts/run_simulation.sh -i 5 -n 4"