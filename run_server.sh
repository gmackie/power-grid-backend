#!/bin/bash

# Simple script to run the Power Grid Game Server

set -e

echo "🚀 Starting Power Grid Game Server..."

# Kill any existing processes on port 4080
echo "🧹 Cleaning up existing processes..."
lsof -ti:4080 | xargs kill -9 2>/dev/null || true
sleep 1

# Build and run the server
echo "🔨 Building and starting server..."
go run cmd/server/main.go