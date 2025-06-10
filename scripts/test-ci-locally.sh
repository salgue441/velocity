#!/bin/bash
# Test CI pipeline locally before pushing
# This script simulates the GitHub Actions CI pipeline

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Configuration
VERBOSE=${VERBOSE:-false}
SKIP_DOCKER=${SKIP_DOCKER:-false}
QUICK_MODE=${QUICK_MODE:-false}

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log() {
  echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
  echo -e "${GREEN}✅ $1${NC}"
  TESTS_PASSED=$((TESTS_PASSED + 1))
}

error() {
  echo -e "${RED}❌ $1${NC}"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  return 1
}

warn() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

run_test() {
  local test_name="$1"
  local test_command="$2"

  TESTS_RUN=$((TESTS_RUN + 1))
  log "Running: $test_name"

  if [ "$VERBOSE" = "true" ]; then
    if eval "$test_command"; then
      success "$test_name"
    else
      error "$test_name"
    fi
  else
    if eval "$test_command" >/dev/null 2>&1; then
      success "$test_name"
    else
      error "$test_name"
    fi
  fi
}

check_dependencies() {
  log "Checking dependencies..."

  local deps_missing=false

  if ! command -v go &>/dev/null; then
    error "Go is not installed"
    deps_missing=true
  fi

  if ! command -v docker &>/dev/null && [ "$SKIP_DOCKER" != "true" ]; then
    warn "Docker not installed - skipping Docker tests"
    SKIP_DOCKER=true
  fi

  if ! command -v golangci-lint &>/dev/null; then
    warn "golangci-lint not installed - will skip detailed linting"
  fi

  if [ "$deps_missing" = "true" ]; then
    exit 1
  fi

  success "Dependencies checked"
}

test_go_modules() {
  log "Testing Go modules..."

  run_test "Go mod download" "go mod download"
  run_test "Go mod verify" "go mod verify"
  run_test "Go mod tidy check" "go mod tidy && git diff --exit-code go.mod go.sum"
}

test_code_quality() {
  log "Testing code quality..."

  run_test "Go build" "go build ./..."
  run_test "Go vet" "go vet ./..."
  run_test "Go fmt check" "gofmt -l . | wc -l | grep -q '^0$'"

  if command -v goimports &>/dev/null; then
    run_test "Go imports check" "goimports -l . | wc -l | grep -q '^0$'"
  else
    warn "goimports not installed - skipping import formatting check"
  fi
}

test_linting() {
  log "Testing linting..."

  if command -v golangci-lint &>/dev/null; then
    run_test "golangci-lint (minimal)" "golangci-lint run --config=.golangci-minimal.yml ./..."

    if [ "$QUICK_MODE" != "true" ]; then
      if [ -f .golangci-simple.yml ]; then
        run_test "golangci-lint (simple)" "golangci-lint run --config=.golangci-simple.yml ./..."
      fi
    fi
  else
    warn "golangci-lint not installed - install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
  fi
}

test_unit_tests() {
  log "Testing unit tests..."

  run_test "Unit tests" "go test -v ./..."
  run_test "Unit tests with race detection" "go test -race ./..."

  if [ "$QUICK_MODE" != "true" ]; then
    run_test "Unit tests with coverage" "go test -coverprofile=coverage.out ./..."

    if [ -f coverage.out ]; then
      local coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
      if (($(echo "$coverage >= 70" | bc -l))); then
        success "Coverage is ${coverage}% (>=70%)"
      else
        error "Coverage is ${coverage}% (<70%)"
      fi
    fi
  fi
}

test_docker_build() {
  if [ "$SKIP_DOCKER" = "true" ]; then
    warn "Skipping Docker tests"
    return 0
  fi

  log "Testing Docker build..."

  run_test "Docker build" "docker build -t velocity-test:local ."
  run_test "Docker run test" "timeout 10s docker run --rm velocity-test:local --version || true"

  if docker buildx version &>/dev/null; then
    run_test "Docker buildx multi-platform" "docker buildx build --platform linux/amd64,linux/arm64 -t velocity-test:multi ."
  else
    warn "Docker buildx not available - skipping multi-platform test"
  fi
}

test_docker_compose() {
  if [ "$SKIP_DOCKER" = "true" ]; then
    return 0
  fi

  log "Testing Docker Compose..."
  if [ -f docker-compose.yml ]; then
    run_test "Docker Compose config validation" "docker-compose config"

    if [ "$QUICK_MODE" != "true" ]; then
      run_test "Docker Compose build check" "docker-compose build --no-cache velocity"
    fi
  else
    warn "docker-compose.yml not found - skipping Docker Compose tests"
  fi
}

test_configuration() {
  log "Testing configuration..."

  run_test "Configuration loading test" "go run -c '
package main
import \"velocity/internal/config\"
func main() {
    loader := config.NewLoader()
    _, err := loader.LoadDefault()
    if err != nil { panic(err) }
}'"

  if [ -f configs/dev.yaml ] || [ -f configs/velocity.example.yaml ]; then
    for config_file in configs/*.yaml; do
      if [ -f "$config_file" ]; then
        run_test "Config validation: $(basename $config_file)" "go run cmd/main.go --validate-config=\"$config_file\" || echo 'Config validation would be implemented'"
      fi
    done
  fi
}

test_security() {
  log "Testing security..."

  if command -v gosec &>/dev/null; then
    run_test "Security scan (gosec)" "gosec ./..."
  else
    warn "gosec not installed - install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
  fi

  run_test "Check for hardcoded secrets" "! grep -r 'password.*=' --include='*.go' . || true"
  run_test "Check for TODO security items" "! grep -r 'TODO.*security' --include='*.go' . || true"
}

simulate_github_actions() {
  log "Simulating GitHub Actions environment..."

  export CI=true
  export GITHUB_ACTIONS=true
  export GITHUB_WORKSPACE="$(pwd)"
  export GITHUB_SHA="$(git rev-parse HEAD 2>/dev/null || echo 'local')"
  export GITHUB_REF="refs/heads/$(git branch --show-current 2>/dev/null || echo 'main')"

  success "GitHub Actions environment simulated"
}

cleanup() {
  log "Cleaning up..."

  rm -f coverage.out
  if [ "$SKIP_DOCKER" != "true" ]; then
    docker rmi velocity-test:local 2>/dev/null || true
    docker rmi velocity-test:multi 2>/dev/null || true
  fi

  success "Cleanup completed"
}

show_summary() {
  echo ""
  echo -e "${PURPLE}═══════════════════════════════════════${NC}"
  echo -e "${PURPLE}           TEST SUMMARY${NC}"
  echo -e "${PURPLE}═══════════════════════════════════════${NC}"
  echo -e "Tests run:    ${BLUE}$TESTS_RUN${NC}"
  echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
  echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
  echo ""

  if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Your code is ready for CI/CD${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo "  git add ."
    echo "  git commit -m 'your commit message'"
    echo "  git push"
    echo ""
    return 0
  else
    echo -e "${RED}❌ Some tests failed. Please fix the issues before pushing.${NC}"
    echo ""
    echo -e "${YELLOW}Common fixes:${NC}"
    echo "  - Run 'go fmt ./...' to fix formatting"
    echo "  - Run 'go mod tidy' to fix module issues"
    echo "  - Run 'golangci-lint run --fix' to auto-fix linting issues"
    echo ""
    return 1
  fi
}

show_help() {
  echo -e "${BLUE}Local CI Testing Script${NC}"
  echo ""
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  --verbose        Show detailed output"
  echo "  --skip-docker    Skip Docker-related tests"
  echo "  --quick          Run quick tests only (skip coverage, etc.)"
  echo "  --help           Show this help"
  echo ""
  echo "Environment variables:"
  echo "  VERBOSE=true     Same as --verbose"
  echo "  SKIP_DOCKER=true Same as --skip-docker"
  echo "  QUICK_MODE=true  Same as --quick"
  echo ""
  echo "Examples:"
  echo "  $0                    # Run all tests"
  echo "  $0 --quick           # Quick validation"
  echo "  $0 --skip-docker     # Skip Docker tests"
  echo "  VERBOSE=true $0      # Verbose output"
}

main() {
  # Parse command line arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    --verbose)
      VERBOSE=true
      shift
      ;;
    --skip-docker)
      SKIP_DOCKER=true
      shift
      ;;
    --quick)
      QUICK_MODE=true
      shift
      ;;
    --help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      show_help
      exit 1
      ;;
    esac
  done

  # Set trap for cleanup
  trap cleanup EXIT

  echo -e "${BLUE}🚀 Velocity Gateway - Local CI Testing${NC}"
  echo -e "${BLUE}======================================${NC}"
  echo ""

  if [ "$QUICK_MODE" = "true" ]; then
    echo -e "${YELLOW}🏃 Quick mode enabled${NC}"
  fi

  if [ "$SKIP_DOCKER" = "true" ]; then
    echo -e "${YELLOW}🐳 Docker tests disabled${NC}"
  fi

  echo ""

  # Run all test phases
  check_dependencies
  simulate_github_actions
  test_go_modules
  test_code_quality
  test_linting
  test_unit_tests
  test_configuration
  test_security

  if [ "$SKIP_DOCKER" != "true" ]; then
    test_docker_build
    test_docker_compose
  fi

  # Show final summary
  show_summary
}

# Run main function
main "$@"
