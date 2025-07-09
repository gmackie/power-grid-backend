.PHONY: all build test test-unit test-integration test-e2e clean run

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
clean:
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