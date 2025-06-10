# Velocity Gateway - MVP Makefile
.PHONY: build run test clean

# Build the gateway binary
build:
	@echo "🔨 Building Velocity Gateway..."
	go build -o bin/velocity-gateway cmd/main.go

# Run the gateway with default settings
run: build
	@echo "🚀 Starting Velocity Gateway..."
	./bin/velocity-gateway

# Run with custom target
run-target: build
	@echo "🚀 Starting with custom target..."
	./bin/velocity-gateway -target=$(TARGET)

# Test the implementation
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f bin/velocity-gateway

# Install dependencies (none for MVP!)
deps:
	@echo "📦 No dependencies yet!"
	go mod tidy

# Quick demo with echo server
demo:
	@echo "🎯 Starting demo..."
	@echo "1. Starting echo server on :3000..."
	@(go run -c 'package main; import ("fmt"; "net/http"); func main() { http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Echo: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr) }); http.ListenAndServe(":3000", nil) }' &)
	@sleep 2
	@echo "2. Starting gateway on :8080..."
	@make run

# Format code
fmt:
	@echo "💅 Formatting code..."
	go fmt ./...