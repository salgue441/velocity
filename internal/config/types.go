// Package config provides configuration management for Velocity Gateway.
//
// This package handles YAML configuration loading, validation, and default
// values for the Velocity Gateway. It supports both file-based configuration
// and programmatic values.
//
// Example usage:
//
//  cfg := config.DefaultConfig()
//  cfg, err := config.LoadFromFile("config.yaml")
//
// Author: Carlos Salguero
// Version: 0.2.0

package config

import "time"

// Config represents the main configuration structure.
// It contains all settings needed to run the gateway including server
// configuration and backend target definitions.
type Config struct {
	// Server contains HTTP server settings like port and timeouts
	Server ServerConfig `yaml:"server"`

	// Targets defines the list of backend services to proxy requests to
	Targets []TargetConfig `yaml:"targets"`

	// Logging configures log output format and verbosity
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig defines HTTP server configuration parameters.
// These settings control how the gateway accepts and handles incoming requests.
type ServerConfig struct {
	// Host specifies the network interface to bind to.
	// Use "0.0.0.0" for all interfaces, "127.0.0.1" for localhost only.
	Host string `yaml:"host"`

	// Port specifies the TCP port number to listen on.
	// Must be between 1 and 65535
	Port int `yaml:"port"`

	// ReadTimeout limits the time spent reading request headers and body.
	// Prevents slow clients from holding connections open indefinitely.
	ReadTimeout time.Duration `yaml:"read_timeout"`

	// WriteTimeout limits the time spent writing the response.
	// Prevents slow clients from causing resource exhaustion.
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// TargetConfig defines configuration for a single backend target service.
// Each target represents a backend server that can receive proxied requets.
type TargetConfig struct {
	// URL is the complete backend service URL including scheme, host, and port.
	// Examples: "http://backend1.com:3000", "https://api.service.com"
	URL string `yaml:"url"`

	// Enabled determines if this target is currently active for load balancing.
	// Disabled targets are excluded from request routing but kept in config.
	Enabled bool `yaml:"enabled"`
}

// LoggingConfig defines logging output format and verbosity settings
type LoggingConfig struct {
	// Level specifies the minimum log level (debug, info, warn, error)
	Level string `yaml:"level"`

	// Format specifies the log output format (text, json)
	Format string `yaml:"format"`
}

// DefaultConfig returns a configuration with sensible default values.
// This configuration works out of the box for development and testing.
//
// Default values:
//   - Server listens on 0.0.0.0:8000
//   - 30 second read/write timeouts
//   - Single target pointing to localhost:3000
//
// Returns a pointer to a new Config instance.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Targets: []TargetConfig{
			{
				URL:     "http://localhost:3000",
				Enabled: true,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
