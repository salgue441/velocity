// Package config provides configuration management for Velocity Gateway

package config

import "time"

// Config represents the main configuration structure
type Config struct {
	Server  ServerConfig   `yaml:"server"`
	Targets []TargetConfig `yaml:"targets"`
}

// ServerConfig defines basic server settings
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// TargetConfig defines a backend target
type TargetConfig struct {
	URL     string `yaml:"url"`
	Enabled bool   `yaml:"enabled"`
}

// DefaultConfig returns a configuration with sensible defaults
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
				URL: "http://localhost:3000",
				Enabled: true,
			},
		},
	}
}
