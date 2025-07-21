.PHONY: all build test test-unit test-integration test-e2e clean run build-ai build-simulator ai-demo simulation

# Default target
all: build

# Build the server
build:
	go build -o powergrid_server ./cmd/server/

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v ./handlers/... ./internal/game/... ./models/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	go test -v ./test/...

# Run end-to-end tests with real server
test-e2e: build
	@echo "Running basic end-to-end tests..."
	./scripts/test_e2e.sh

# Run comprehensive gameplay end-to-end tests
test-full-game: build
	@echo "Running full gameplay end-to-end tests..."
	./scripts/test_full_game_e2e.sh

# Run all end-to-end tests
test-e2e-all: test-e2e test-full-game

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Clean build artifacts
clean: clean-ai
	rm -f powergrid_server
	rm -f coverage.out coverage.html
	rm -rf logs/

# Run the server
run: build
	./powergrid_server -addr=:4080

# Run server with debug logging
run-debug: build
	DEBUG=true ./powergrid_server -addr=:4080

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Generate mocks for testing
mocks:
	go generate ./...

# Benchmark tests
bench:
	go test -bench=. -benchmem ./...

# AI Client targets
build-ai:
	@echo "Building AI client..."
	go build -o cmd/ai_client/ai_client ./cmd/ai_client/

build-simulator:
	@echo "Building simulator..."
	go build -o cmd/simulator/simulator ./cmd/simulator/

# Launch AI clients for demo/testing
ai-demo: build-ai
	@echo "Launching AI demo with 4 players..."
	./scripts/launch_ai_clients.sh -n 4 -t "aggressive,conservative,balanced,random"

# Run AI simulation
simulation: build-simulator
	@echo "Running AI simulation..."
	./scripts/run_simulation.sh -i 10 -n 4

# Build analytics demo
build-analytics-demo:
	@echo "Building analytics demo..."
	go build -o cmd/analytics_demo/analytics_demo ./cmd/analytics_demo/

# Generate demo analytics data
demo-analytics: build-analytics-demo
	@echo "Generating demo analytics data..."
	./cmd/analytics_demo/analytics_demo -games 15

# Test analytics API
test-analytics:
	@echo "Testing analytics API..."
	./scripts/test_analytics_api.sh

# Build all AI tools
build-ai-all: build-ai build-simulator build-analytics-demo

# Clean AI builds
clean-ai:
	rm -f cmd/ai_client/ai_client
	rm -f cmd/simulator/simulator
	rm -f cmd/analytics_demo/analytics_demo