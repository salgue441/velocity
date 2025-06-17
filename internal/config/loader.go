package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a YAML file and merges it with
// defaults.
//
// This function:
//  1. Start with default configuration values
//  2. Reads the specified YAML file
//  3. Unmarshals YAML data over the defaults
//  4. Returns the merged configuration
//
// The file path can be absolute or relative to the current working directory.
// If the file doesn't exist or has invalid YAML syntax, an error is returned.
//
// Parameters:
//
//	filename: Path to the YAML configuration file
//
// Returns:
//
//	*Config: Loaded configuration with defaults applied
//	error: File reading or YAML parsing error
//
// Example:
//
//	 cfg, err := LoadFromFile("config.yaml")
//	 if err != nil {
//	   log.Fatalf("Failed to load config: %v", err)
//	}
func LoadFromFile(filename string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return cfg, nil
}
