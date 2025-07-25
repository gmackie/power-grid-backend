# Multi-stage Dockerfile for Power Grid Go Server
# Produces optimized production images

# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o powergrid-server ./cmd/server/main.go

# Production stage
FROM alpine:latest
WORKDIR /

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -g '' powergrid

# Copy the binary
COPY --from=builder /app/powergrid-server .

# Change ownership
RUN chown powergrid:powergrid /powergrid-server

# Switch to non-root user
USER powergrid

# Expose WebSocket port
EXPOSE 4080

CMD ["./powergrid-server"]