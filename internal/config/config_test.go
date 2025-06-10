package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig verifies that the default configuration is valid
// and contains expected values.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test server defaults
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 30*time.Second, config.Server.ReadTimeout)

	// Test targets defaults
	require.Len(t, config.Targets, 1)
	assert.Equal(t, "http://localhost:3000", config.Targets[0].URL)
	assert.Equal(t, 100, config.Targets[0].Weight)
	assert.True(t, config.Targets[0].Enabled)

	// Test load balancing defaults
	assert.Equal(t, "round_robin", config.LoadBalancing.Algorithm)

	// Test health check defaults
	assert.True(t, config.HealthCheck.Enabled)
	assert.Equal(t, 30*time.Second, config.HealthCheck.Interval)

	// Validate the default configuration
	err := config.Validate()
	assert.NoError(t, err, "Default configuration should be valid")
}

// TestConfigValidation tests the configuration validation logic.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		configFunc  func() *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			configFunc: func() *Config {
				return DefaultConfig()
			},
			expectError: false,
		},
		{
			name: "invalid server port - too low",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Server.Port = 0
				return config
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "invalid server port - too high",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Server.Port = 70000
				return config
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "empty host",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Server.Host = ""
				return config
			},
			expectError: true,
			errorMsg:    "host is required",
		},
		{
			name: "no targets",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Targets = []TargetConfig{}
				return config
			},
			expectError: true,
			errorMsg:    "at least one target must be configured",
		},
		{
			name: "all targets disabled",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Targets[0].Enabled = false
				return config
			},
			expectError: true,
			errorMsg:    "at least one target must be enabled",
		},
		{
			name: "invalid target URL",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Targets[0].URL = "not-a-url"
				return config
			},
			expectError: true,
			errorMsg:    "URL scheme must be http or https",
		},
		{
			name: "invalid target weight",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.Targets[0].Weight = 150
				return config
			},
			expectError: true,
			errorMsg:    "weight must be between 0 and 100",
		},
		{
			name: "invalid load balancing algorithm",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.LoadBalancing.Algorithm = "invalid_algorithm"
				return config
			},
			expectError: true,
			errorMsg:    "unsupported load balancing algorithm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc()
			err := config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTargetConfigValidation tests target-specific validation.
func TestTargetConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		target      TargetConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid HTTP target",
			target: TargetConfig{
				URL:     "http://example.com:8080",
				Weight:  50,
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "valid HTTPS target",
			target: TargetConfig{
				URL:     "https://example.com:443",
				Weight:  100,
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "invalid scheme",
			target: TargetConfig{
				URL:     "ftp://example.com",
				Weight:  50,
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "URL scheme must be http or https",
		},
		{
			name: "negative weight",
			target: TargetConfig{
				URL:     "http://example.com",
				Weight:  -10,
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "weight must be between 0 and 100",
		},
		{
			name: "negative max connections",
			target: TargetConfig{
				URL:            "http://example.com",
				Weight:         50,
				Enabled:        true,
				MaxConnections: -5,
			},
			expectError: true,
			errorMsg:    "max_connections must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHealthCheckValidation tests health check configuration validation.
func TestHealthCheckValidation(t *testing.T) {
	tests := []struct {
		name        string
		healthCheck HealthCheckConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid health check config",
			healthCheck: HealthCheckConfig{
				Enabled:            true,
				Interval:           30 * time.Second,
				Timeout:            5 * time.Second,
				UnhealthyThreshold: 3,
				HealthyThreshold:   2,
			},
			expectError: false,
		},
		{
			name: "disabled health check (should pass)",
			healthCheck: HealthCheckConfig{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "timeout greater than interval",
			healthCheck: HealthCheckConfig{
				Enabled:  true,
				Interval: 5 * time.Second,
				Timeout:  10 * time.Second,
			},
			expectError: true,
			errorMsg:    "timeout",
		},
		{
			name: "zero unhealthy threshold",
			healthCheck: HealthCheckConfig{
				Enabled:            true,
				Interval:           30 * time.Second,
				Timeout:            5 * time.Second,
				UnhealthyThreshold: 0,
				HealthyThreshold:   2,
			},
			expectError: true,
			errorMsg:    "unhealthy threshold must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.healthCheck.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigLoader tests the configuration loading functionality.
func TestConfigLoader(t *testing.T) {
	loader := NewLoader()

	t.Run("load default config", func(t *testing.T) {
		config, err := loader.LoadDefault()
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "0.0.0.0", config.Server.Host)
		assert.Equal(t, 8080, config.Server.Port)
	})

	t.Run("load from valid YAML file", func(t *testing.T) {
		// Create temporary config file
		configYAML := `
server:
  host: "127.0.0.1"
  port: 9090
targets:
  - url: "http://backend1.com:8080"
    weight: 70
    enabled: true
  - url: "http://backend2.com:8080"
    weight: 30
    enabled: true
load_balancing:
  algorithm: "weighted_round_robin"
`
		tmpFile := createTempFile(t, "config.yaml", configYAML)
		defer os.Remove(tmpFile)

		config, err := loader.LoadFromFile(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", config.Server.Host)
		assert.Equal(t, 9090, config.Server.Port)
		assert.Len(t, config.Targets, 2)
		assert.Equal(t, "weighted_round_robin", config.LoadBalancing.Algorithm)
	})

	t.Run("load from non-existent file", func(t *testing.T) {
		_, err := loader.LoadFromFile("non-existent.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("load from invalid YAML", func(t *testing.T) {
		invalidYAML := `
server:
  host: "127.0.0.1"
  port: [this, is, invalid, yaml, syntax
`
		tmpFile := createTempFile(t, "invalid.yaml", invalidYAML)
		defer os.Remove(tmpFile)

		_, err := loader.LoadFromFile(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse YAML")
	})
}

// TestEnvironmentOverrides tests environment variable override functionality.
func TestEnvironmentOverrides(t *testing.T) {
	loader := NewLoader()

	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"VELOCITY_SERVER_HOST",
		"VELOCITY_SERVER_PORT",
		"VELOCITY_LOGGING_LEVEL",
		"VELOCITY_HEALTH_CHECK_ENABLED",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	// Clean up after test
	defer func() {
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	t.Run("server host override", func(t *testing.T) {
		os.Setenv("VELOCITY_SERVER_HOST", "192.168.1.100")
		config, err := loader.LoadDefault()
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.100", config.Server.Host)
	})

	t.Run("server port override", func(t *testing.T) {
		os.Setenv("VELOCITY_SERVER_PORT", "9999")
		config, err := loader.LoadDefault()
		require.NoError(t, err)
		assert.Equal(t, 9999, config.Server.Port)
	})

	t.Run("logging level override", func(t *testing.T) {
		os.Setenv("VELOCITY_LOGGING_LEVEL", "debug")
		config, err := loader.LoadDefault()
		require.NoError(t, err)
		assert.Equal(t, "debug", config.Logging.Level)
	})

	t.Run("health check enabled override", func(t *testing.T) {
		os.Setenv("VELOCITY_HEALTH_CHECK_ENABLED", "false")
		config, err := loader.LoadDefault()
		require.NoError(t, err)
		assert.False(t, config.HealthCheck.Enabled)
	})

	t.Run("invalid port override", func(t *testing.T) {
		os.Setenv("VELOCITY_SERVER_PORT", "invalid")
		_, err := loader.LoadDefault()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SERVER_PORT")
	})
}

// TestSaveToFile tests configuration saving functionality.
func TestSaveToFile(t *testing.T) {
	loader := NewLoader()
	config := DefaultConfig()

	// Create temporary directory
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	// Save configuration
	err := loader.SaveToFile(config, configFile)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Load the saved configuration
	loadedConfig, err := loader.LoadFromFile(configFile)
	require.NoError(t, err)

	// Compare configurations
	assert.Equal(t, config.Server.Host, loadedConfig.Server.Host)
	assert.Equal(t, config.Server.Port, loadedConfig.Server.Port)
	assert.Equal(t, config.LoadBalancing.Algorithm, loadedConfig.LoadBalancing.Algorithm)
}

// TestGenerateExample tests the example configuration generation.
func TestGenerateExample(t *testing.T) {
	loader := NewLoader()
	example := loader.GenerateExample()

	assert.NotEmpty(t, example)
	assert.Contains(t, example, "# Velocity Gateway Configuration Example")
	assert.Contains(t, example, "server:")
	assert.Contains(t, example, "targets:")
	assert.Contains(t, example, "load_balancing:")
	assert.Contains(t, example, "health_check:")
}

// TestValidateFile tests file validation without loading.
func TestValidateFile(t *testing.T) {
	loader := NewLoader()

	t.Run("valid configuration file", func(t *testing.T) {
		validYAML := `
server:
  host: "0.0.0.0"
  port: 8080
targets:
  - url: "http://example.com"
    weight: 100
    enabled: true
`
		tmpFile := createTempFile(t, "valid.yaml", validYAML)
		defer os.Remove(tmpFile)

		err := loader.ValidateFile(tmpFile)
		assert.NoError(t, err)
	})

	t.Run("invalid configuration file", func(t *testing.T) {
		invalidYAML := `
server:
  host: ""
  port: 8080
targets: []
`
		tmpFile := createTempFile(t, "invalid.yaml", invalidYAML)
		defer os.Remove(tmpFile)

		err := loader.ValidateFile(tmpFile)
		assert.Error(t, err)
	})
}

// Helper function to create temporary files for testing.
func createTempFile(t *testing.T, name, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, name)

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}
