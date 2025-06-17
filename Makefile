# Velocity Gateway - Basic Build System
BINARY_NAME := velocity
MAIN_PATH := ./cmd/velocity

.PHONY: help
help: ## Show available commands
	@echo "Available commands:"
	@echo "  run         - Run the development server"
	@echo "  run-config  - Run with config.yaml"
	@echo "  build       - Build the binary"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"

.PHONY: run
run: ## Run development server
	go run $(MAIN_PATH)

.PHONY: run-config
run-config: ## Run with config file
	go run $(MAIN_PATH) -config=config.yaml

.PHONY: build
build: ## Build the application
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	go clean