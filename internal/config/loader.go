// Package config provides configuration loading and management functionality.
// This file implements the configuration loading logic with support for
// YAML files, environment variables, and command-line flag overrides.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader provides configuration loading functionality with support for
// multiple configuration sources and validation.
//
// The loader implements a priority-based configuration system:
//  1. Explicit values (command-line flags, API calls)
//  2. Environment variables with VELOCITY_ prefix
//  3. Configuration file (YAML)
//  4. Default values
//
// Example usage:
//
//	loader := NewLoader()
//	config, err := loader.LoadFromFile("config.yaml")
//	if err != nil {
//	    log.Fatalf("Failed to load configuration: %v", err)
//	}
type Loader struct {
	// envPrefix is the prefix for environment variable lookups
	envPrefix string
}

// NewLoader creates a new configuration loader with default settings.
// The loader is configured to use the VELOCITY_ prefix for environment variables.
//
// Returns:
//
//	*Loader: New configuration loader instance
func NewLoader() *Loader {
	return &Loader{
		envPrefix: "VELOCITY_",
	}
}

// LoadFromFile loads configuration from a YAML file with environment variable
// overrides and validation.
//
// The loading process follows these steps:
//  1. Start with default configuration values
//  2. Load and merge YAML file configuration
//  3. Apply environment variable overrides
//  4. Validate the final configuration
//  5. Return validated configuration or error
//
// Environment Variable Mapping:
//   - VELOCITY_SERVER_PORT overrides server.port
//   - VELOCITY_SERVER_HOST overrides server.host
//   - VELOCITY_LOGGING_LEVEL overrides logging.level
//   - VELOCITY_HEALTH_CHECK_ENABLED overrides health_check.enabled
//
// Parameters:
//
//	filename: Path to the YAML configuration file
//
// Returns:
//
//	*Config: Loaded and validated configuration
//	error: Loading or validation error
//
// Example:
//
//	config, err := loader.LoadFromFile("config.yaml")
//	if err != nil {
//	    return fmt.Errorf("configuration loading failed: %w", err)
//	}
func (l *Loader) LoadFromFile(filename string) (*Config, error) {
	config := DefaultConfig()

	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file '%s' not found", filename)
		}

		return nil, fmt.Errorf("error accessing configuration file '%s': %w", filename, err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file '%s': %w", filename, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	if err := l.applyEnvironmentOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadDefault loads the default configuration with environment variable
// overrides applied. This is useful for development and when no configuration
// file is available.
//
// Returns:
//
//	*Config: Default configuration with environment overrides
//	error: Validation error if environment overrides create invalid
//	       configuration
func (l *Loader) LoadDefault() (*Config, error) {
	config := DefaultConfig()

	if err := l.applyEnvironmentOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// SaveToFile saves the configuration to a YAML file with proper formatting
// and comments. This is useful for generating example configurations or
// persisting runtime configuration changes.
//
// Parameters:
//
//	config: Configuration to save
//	filename: Target file path
//
// Returns:
//
//	error: File writing error
func (l *Loader) SaveToFile(config *Config, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", dir, err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file '%s': %w", filename, err)
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to the
// configuration. This allows runtime configuration without modifying files.
//
// Supported environment variables:
//   - VELOCITY_SERVER_HOST: Override server host
//   - VELOCITY_SERVER_PORT: Override server port
//   - VELOCITY_LOGGING_LEVEL: Override logging level
//   - VELOCITY_HEALTH_CHECK_ENABLED: Override health check enabled status
//
// Parameters:
//
//	config: Configuration to modify with environment overrides
//
// Returns:
//
//	error: Parsing error for environment variable values
func (l *Loader) applyEnvironmentOverrides(config *Config) error {
	if host := os.Getenv(l.envPrefix + "SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	if port := os.Getenv(l.envPrefix + "SERVER_PORT"); port != "" {
		var portInt int
		if _, err := fmt.Sscanf(port, "%d", &portInt); err != nil {
			return fmt.Errorf("invalid SERVER_PORT value '%s': must be integer", port)
		}

		config.Server.Port = portInt
	}

	if level := os.Getenv(l.envPrefix + "LOGGING_LEVEL"); level != "" {
		validLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
		}

		if !validLevels[strings.ToLower(level)] {
			return fmt.Errorf("invalid LOGGING_LEVEL value '%s': must be debug, info, warn, error, or fatal", level)
		}

		config.Logging.Level = strings.ToLower(level)
	}

	if format := os.Getenv(l.envPrefix + "LOGGING_FORMAT"); format != "" {
		validFormats := map[string]bool{"text": true, "json": true}
		if !validFormats[strings.ToLower(format)] {
			return fmt.Errorf("invalid LOGGING_FORMAT value '%s': must be text or json", format)
		}

		config.Logging.Format = strings.ToLower(format)
	}

	// Health check configuration overrides
	if enabled := os.Getenv(l.envPrefix + "HEALTH_CHECK_ENABLED"); enabled != "" {
		switch strings.ToLower(enabled) {
		case "true", "1", "yes", "on":
			config.HealthCheck.Enabled = true

		case "false", "0", "no", "off":
			config.HealthCheck.Enabled = false

		default:
			return fmt.Errorf("invalid HEALTH_CHECK_ENABLED value '%s': must be true/false", enabled)
		}
	}

	if algorithm := os.Getenv(l.envPrefix + "LOAD_BALANCING_ALGORITHM"); algorithm != "" {
		validAlgorithms := map[string]bool{
			"round_robin": true, "weighted_round_robin": true,
			"least_connections": true, "ip_hash": true,
		}

		if !validAlgorithms[algorithm] {
			return fmt.Errorf("invalid LOAD_BALANCING_ALGORITHM value '%s'", algorithm)
		}

		config.LoadBalancing.Algorithm = algorithm
	}

	return nil
}

// GenerateExample generates an example configuration file with comments
// and all available options. This is useful for documentation and
// initial setup.
//
// Returns:
//
//	string: YAML configuration with comments and examples
func (l *Loader) GenerateExample() string {
	return `# Velocity Gateway Configuration Example
# This file demonstrates all available configuration options

# Server configuration
server:
  # Network interface to bind to (0.0.0.0 = all interfaces)
  host: "0.0.0.0"
  
  # TCP port to listen on
  port: 8080
  
  # Request timeout settings
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  
  # Maximum size of request headers (in bytes)
  max_header_bytes: 1048576
  
  # Time to wait during graceful shutdown
  graceful_timeout: "30s"

# Backend target configuration
targets:
  # Primary backend server
  - url: "http://backend1.example.com:3000"
    weight: 60                    # Relative weight for load balancing (1-100)
    enabled: true                 # Whether this target is active
    name: "primary"               # Optional human-readable name
    max_connections: 100          # Optional connection limit
    timeout: "30s"                # Optional request timeout override
    tags:                         # Optional metadata tags
      environment: "production"
      datacenter: "us-east-1"
  
  # Secondary backend server
  - url: "http://backend2.example.com:3000"
    weight: 40
    enabled: true
    name: "secondary"
    tags:
      environment: "production"
      datacenter: "us-west-1"

# Load balancing configuration
load_balancing:
  # Algorithm: round_robin, weighted_round_robin, least_connections, ip_hash
  algorithm: "round_robin"
  
  # Optional sticky session configuration
  sticky_session:
    enabled: false
    cookie_name: "velocity-session"
    ttl: "1h"
    header: "X-Session-ID"        # Alternative to cookie
  
  # Retry policy for failed requests
  retry_policy:
    enabled: true
    max_retries: 3
    retry_delay: "100ms"
    backoff_multiplier: 2.0
    retriable_status_codes: [502, 503, 504]

# Health check configuration
health_check:
  enabled: true
  interval: "30s"                 # Time between health checks
  timeout: "5s"                   # Health check request timeout
  path: "/health"                 # Endpoint to check
  method: "GET"                   # HTTP method for health checks
  expected_status: [200, 204]     # Status codes indicating healthy service
  unhealthy_threshold: 3          # Failures before marking unhealthy
  healthy_threshold: 2            # Successes before marking healthy
  headers:                        # Optional headers for health checks
    User-Agent: "Velocity-Gateway-HealthCheck/0.2.0"

# Logging configuration
logging:
  level: "info"                   # debug, info, warn, error, fatal
  format: "text"                  # text or json
  access_log: true                # Enable HTTP access logging
  file: ""                        # Optional log file path (empty = stdout)

# Configuration version for compatibility tracking
version: "0.2.0"
`
}

// ValidateFile validates a configuration file without loading it into memory.
// This is useful for configuration validation in CI/CD pipelines or
// administrative tools.
//
// Parameters:
//
//	filename: Path to the configuration file to validate
//
// Returns:
//
//	error: Validation error if file is invalid, nil if valid
func (l *Loader) ValidateFile(filename string) error {
	_, err := l.LoadFromFile(filename)
	return err
}
