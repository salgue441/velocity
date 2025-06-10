// Package config provides configuration management for the Velocity Gateway.
// It defines the configuration structures and validation logic for all gateway
// components including server settings, backend targets, and feature flags.
//
// The configuration system supports:
//   - YAML file-based configuration with environment variable overrides
//   - Validation of all configuration parameters with clear error messages
//   - Hot-reloading of configuration for non-disruptive updates
//   - Backward compatibility with command-line flag configuration
//
// Configuration Loading Priority (highest to lowest):
//   1. Command-line flags
//   2. Environment variables
//   3. Configuration file
//   4. Default values
//
// Example configuration file:
//   server:
//     port: 8080
//     host: "0.0.0.0"
//   targets:
//     - url: "http://backend1.com:3000"
//       weight: 60
//       enabled: true
//     - url: "http://backend2.com:3000"
//       weight: 40
//       enabled: true
//   load_balancing:
//     algorithm: "round_robin"
//
// Author: Velocity Gateway Project
// Version: 0.2.0
package config

import (
	"fmt"
	"net/url"
	"time"
)

// Config represents the complete configuration for the Velocity Gateway.
// This is the root configuration structure that contains all settings
// for server operation, backend targets, and feature configuration.
//
// The configuration is designed to be serializable to/from YAML and
// includes validation tags for automatic input validation.
type Config struct {
	// Server contains HTTP server configuration including port, timeouts, and limits
	Server ServerConfig `yaml:"server" json:"server" validate:"required"`
	
	// Targets defines the list of backend services to proxy requests to
	Targets []TargetConfig `yaml:"targets" json:"targets" validate:"required,min=1,dive"`
	
	// LoadBalancing configures the load balancing algorithm and behavior
	LoadBalancing LoadBalancingConfig `yaml:"load_balancing" json:"load_balancing"`
	
	// HealthCheck configures health checking for backend targets
	HealthCheck HealthCheckConfig `yaml:"health_check" json:"health_check"`
	
	// Logging configures log output format and verbosity
	Logging LoggingConfig `yaml:"logging" json:"logging"`
	
	// Version tracks the configuration file version for compatibility
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// ServerConfig defines HTTP server configuration parameters including
// network settings, timeouts, and performance tuning options.
//
// These settings control how the gateway accepts and processes incoming
// HTTP requests from clients.
type ServerConfig struct {
	// Host specifies the network interface to bind to (default: "0.0.0.0")
	Host string `yaml:"host" json:"host" validate:"required"`
	
	// Port specifies the TCP port to listen on (default: 8080)
	Port int `yaml:"port" json:"port" validate:"required,min=1,max=65535"`
	
	// ReadTimeout limits the time spent reading the request headers and body
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`
	
	// WriteTimeout limits the time spent writing the response
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	
	// IdleTimeout limits the time connections remain idle before closure
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	
	// MaxHeaderBytes limits the size of request headers
	MaxHeaderBytes int `yaml:"max_header_bytes" json:"max_header_bytes"`
	
	// GracefulTimeout specifies how long to wait during graceful shutdown
	GracefulTimeout time.Duration `yaml:"graceful_timeout" json:"graceful_timeout"`
}

// TargetConfig defines configuration for a single backend target service.
// Each target represents a backend server that can receive proxied requests.
//
// Targets support weighted load balancing, individual health checking,
// and can be enabled/disabled without restarting the gateway.
type TargetConfig struct {
	// URL is the complete backend service URL including scheme, host, and port
	URL string `yaml:"url" json:"url" validate:"required,url"`
	
	// Weight determines the relative amount of traffic this target receives
	// Higher weights receive proportionally more requests (1-100, default: 100)
	Weight int `yaml:"weight" json:"weight" validate:"min=0,max=100"`
	
	// Enabled determines if this target is currently active for load balancing
	// Disabled targets are excluded from request routing
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// Name is an optional human-readable identifier for this target
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	
	// Tags provide metadata for grouping and filtering targets
	Tags map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	
	// MaxConnections limits concurrent connections to this target (0 = unlimited)
	MaxConnections int `yaml:"max_connections,omitempty" json:"max_connections,omitempty" validate:"min=0"`
	
	// Timeout overrides the default request timeout for this specific target
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// LoadBalancingConfig defines the load balancing algorithm and related settings.
// This controls how requests are distributed among available backend targets.
//
// Supported algorithms:
//   - round_robin: Distributes requests evenly across targets
//   - weighted_round_robin: Uses target weights for proportional distribution
//   - least_connections: Routes to target with fewest active connections
//   - ip_hash: Routes based on client IP hash for session affinity
type LoadBalancingConfig struct {
	// Algorithm specifies the load balancing method to use
	Algorithm string `yaml:"algorithm" json:"algorithm" validate:"oneof=round_robin weighted_round_robin least_connections ip_hash"`
	
	// StickySession enables session affinity using cookies or headers
	StickySession StickySessionConfig `yaml:"sticky_session,omitempty" json:"sticky_session,omitempty"`
	
	// RetryPolicy defines retry behavior for failed requests
	RetryPolicy RetryPolicyConfig `yaml:"retry_policy,omitempty" json:"retry_policy,omitempty"`
}

// StickySessionConfig configures session affinity to ensure requests
// from the same client are routed to the same backend target.
//
// This is useful for applications that maintain server-side session state.
type StickySessionConfig struct {
	// Enabled determines if sticky sessions are active
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// CookieName specifies the cookie used for session tracking
	CookieName string `yaml:"cookie_name" json:"cookie_name"`
	
	// TTL defines how long session affinity is maintained
	TTL time.Duration `yaml:"ttl" json:"ttl"`
	
	// Header specifies an alternative header for session tracking
	Header string `yaml:"header,omitempty" json:"header,omitempty"`
}

// RetryPolicyConfig defines retry behavior for failed backend requests.
// This improves reliability by automatically retrying transient failures.
type RetryPolicyConfig struct {
	// Enabled determines if automatic retries are active
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// MaxRetries limits the number of retry attempts per request
	MaxRetries int `yaml:"max_retries" json:"max_retries" validate:"min=0,max=10"`
	
	// RetryDelay specifies the initial delay between retry attempts
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay"`
	
	// BackoffMultiplier increases delay between successive retries
	BackoffMultiplier float64 `yaml:"backoff_multiplier" json:"backoff_multiplier" validate:"min=1.0,max=5.0"`
	
	// RetriableStatusCodes lists HTTP status codes that should trigger retries
	RetriableStatusCodes []int `yaml:"retriable_status_codes,omitempty" json:"retriable_status_codes,omitempty"`
}

// HealthCheckConfig defines health checking parameters for backend targets.
// Health checks continuously monitor target availability and remove
// unhealthy targets from the load balancing pool.
type HealthCheckConfig struct {
	// Enabled determines if health checking is active
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// Interval specifies the time between health check requests
	Interval time.Duration `yaml:"interval" json:"interval"`
	
	// Timeout limits the time spent waiting for health check responses
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	
	// Path specifies the endpoint to request for health checks
	Path string `yaml:"path" json:"path"`
	
	// Method specifies the HTTP method for health check requests (default: GET)
	Method string `yaml:"method" json:"method" validate:"oneof=GET POST HEAD"`
	
	// ExpectedStatus lists HTTP status codes indicating healthy targets
	ExpectedStatus []int `yaml:"expected_status" json:"expected_status"`
	
	// UnhealthyThreshold specifies consecutive failures before marking unhealthy
	UnhealthyThreshold int `yaml:"unhealthy_threshold" json:"unhealthy_threshold" validate:"min=1,max=10"`
	
	// HealthyThreshold specifies consecutive successes before marking healthy
	HealthyThreshold int `yaml:"healthy_threshold" json:"healthy_threshold" validate:"min=1,max=10"`
	
	// Headers contains additional headers to include in health check requests
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// LoggingConfig defines logging output format and verbosity settings.
// This controls how the gateway generates and formats log messages.
type LoggingConfig struct {
	// Level specifies the minimum log level to output
	Level string `yaml:"level" json:"level" validate:"oneof=debug info warn error fatal"`
	
	// Format specifies the log output format
	Format string `yaml:"format" json:"format" validate:"oneof=text json"`
	
	// AccessLog enables/disables HTTP access logging
	AccessLog bool `yaml:"access_log" json:"access_log"`
	
	// File specifies an optional log file path (empty = stdout)
	File string `yaml:"file,omitempty" json:"file,omitempty"`
}

// DefaultConfig returns a configuration with sensible default values.
// This provides a baseline configuration that works out of the box
// for development and testing scenarios.
//
// Default configuration includes:
//   - Server listening on 0.0.0.0:8080
//   - Single target pointing to localhost:3000
//   - Round-robin load balancing
//   - Health checks enabled with 30-second interval
//   - Info-level logging to stdout
//
// Returns:
//   *Config: Configuration with default values applied
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxHeaderBytes:  1048576, // 1MB
			GracefulTimeout: 30 * time.Second,
		},
		Targets: []TargetConfig{
			{
				URL:     "http://localhost:3000",
				Weight:  100,
				Enabled: true,
				Name:    "default",
			},
		},
		LoadBalancing: LoadBalancingConfig{
			Algorithm: "round_robin",
			RetryPolicy: RetryPolicyConfig{
				Enabled:           true,
				MaxRetries:        3,
				RetryDelay:        100 * time.Millisecond,
				BackoffMultiplier: 2.0,
				RetriableStatusCodes: []int{502, 503, 504},
			},
		},
		HealthCheck: HealthCheckConfig{
			Enabled:            true,
			Interval:           30 * time.Second,
			Timeout:            5 * time.Second,
			Path:               "/health",
			Method:             "GET",
			ExpectedStatus:     []int{200, 204},
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		},
		Logging: LoggingConfig{
			Level:     "info",
			Format:    "text",
			AccessLog: true,
		},
		Version: "0.2.0",
	}
}

// Validate performs comprehensive validation of the configuration.
// It checks all required fields, validates URLs, port ranges,
// and logical consistency between related settings.
//
// Validation includes:
//   - Required field presence
//   - URL format validation
//   - Port range validation (1-65535)
//   - Timeout value validation (positive durations)
//   - Load balancing algorithm validation
//   - Target weight validation (0-100)
//
// Returns:
//   error: Validation error with detailed message, nil if valid
func (c *Config) Validate() error {
	// Validate server configuration
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server configuration invalid: %w", err)
	}
	
	// Validate targets
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target must be configured")
	}
	
	enabledTargets := 0
	for i, target := range c.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("target %d invalid: %w", i, err)
		}
		if target.Enabled {
			enabledTargets++
		}
	}
	
	if enabledTargets == 0 {
		return fmt.Errorf("at least one target must be enabled")
	}
	
	// Validate load balancing configuration
	if err := c.LoadBalancing.Validate(); err != nil {
		return fmt.Errorf("load balancing configuration invalid: %w", err)
	}
	
	// Validate health check configuration
	if err := c.HealthCheck.Validate(); err != nil {
		return fmt.Errorf("health check configuration invalid: %w", err)
	}
	
	return nil
}

// Validate validates the server configuration parameters.
// It ensures all required fields are present and values are within valid 
// ranges.
//
// Returns:
//   error: Validation error if configuration is invalid, nil otherwise
func (s *ServerConfig) Validate() error {
	if s.Host == "" {
		return fmt.Errorf("host is required")
	}
	
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", s.Port)
	}
	
	if s.ReadTimeout < 0 {
		return fmt.Errorf("read_timeout must be positive, got %v", s.ReadTimeout)
	}
	
	if s.WriteTimeout < 0 {
		return fmt.Errorf("write_timeout must be positive, got %v", s.WriteTimeout)
	}
	
	return nil
}

// Validate validates the target configuration parameters.
// It ensures the URL is valid and weight is within acceptable range.
//
// Returns:
//   error: Validation error if configuration is invalid, nil otherwise
func (t *TargetConfig) Validate() error {
	if t.URL == "" {
		return fmt.Errorf("URL is required")
	}
	
	// Parse and validate URL
	parsedURL, err := url.Parse(t.URL)
	if err != nil {
		return fmt.Errorf("invalid URL '%s': %w", t.URL, err)
	}
	
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got '%s'", parsedURL.Scheme)
	}
	
	if t.Weight < 0 || t.Weight > 100 {
		return fmt.Errorf("weight must be between 0 and 100, got %d", t.Weight)
	}
	
	if t.MaxConnections < 0 {
		return fmt.Errorf("max_connections must be non-negative, got %d", t.MaxConnections)
	}
	
	return nil
}

// Validate validates the load balancing configuration.
// It ensures the algorithm is supported and related settings are consistent.
//
// Returns:
//   error: Validation error if configuration is invalid, nil otherwise
func (lb *LoadBalancingConfig) Validate() error {
	validAlgorithms := map[string]bool{
		"round_robin":          true,
		"weighted_round_robin": true,
		"least_connections":    true,
		"ip_hash":              true,
	}
	
	if !validAlgorithms[lb.Algorithm] {
		return fmt.Errorf("unsupported load balancing algorithm '%s'", lb.Algorithm)
	}
	
	return nil
}

// Validate validates the health check configuration.
// It ensures intervals, timeouts, and thresholds are reasonable.
//
// Returns:
//   error: Validation error if configuration is invalid, nil otherwise
func (hc *HealthCheckConfig) Validate() error {
	if hc.Enabled {
		if hc.Interval <= 0 {
			return fmt.Errorf("health check interval must be positive")
		}
		
		if hc.Timeout <= 0 {
			return fmt.Errorf("health check timeout must be positive")
		}
		
		if hc.Timeout >= hc.Interval {
			return fmt.Errorf("health check timeout (%v) must be less than interval (%v)", 
				hc.Timeout, hc.Interval)
		}
		
		if hc.UnhealthyThreshold < 1 {
			return fmt.Errorf("unhealthy threshold must be at least 1")
		}
		
		if hc.HealthyThreshold < 1 {
			return fmt.Errorf("healthy threshold must be at least 1")
		}
	}
	
	return nil
}