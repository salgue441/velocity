// Package logger provides structured logging for Velocity Gateway
//
// This package wraps Go's standard log/slog package with gateway-specific
// convenience methods and consistent formatting.
package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional convenience methods
type Logger struct {
	*slog.Logger
}

// Config defines logger configuration options
type LoggerConfig struct {
	// Level specifies the minimum log level (debug, info, warn, error)
	Level string `yaml:"level"`

	// Format specifies output format (text, json)
	Format string `yaml:"format"`
}

// New creates a new logger with the specified configuration
func New(cfg LoggerConfig) *Logger {
	if cfg.Level == "" {
		cfg.Level = "info"
	}

	if cfg.Format == "" {
		cfg.Format = "text"
	}

	// Parse log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug

	case "info":
		level = slog.LevelInfo

	case "warn":
		level = slog.LevelWarn

	case "error":
		level = slog.LevelError

	default:
		level = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default creates a logger with default settings
func Default() *Logger {
	return New(LoggerConfig{Level: "info", Format: "text"})
}

// Gateway-specific convenience methods

// LogProxy logs a proxy request attempt
func (l *Logger) LogProxy(method, path, target string, attempt, total int) {
	l.Info("Proxy attempt",
		"method", method,
		"path", path,
		"target", target,
		"attempt", attempt,
		"total_targets", total,
	)
}

// LogProxySuccess logs a successful proxy request
func (l *Logger) LogProxySuccess(target string) {
	l.Info("Proxy success", "target", target)
}

// LogProxyFailure logs a failed proxy request
func (l *Logger) LogProxyFailure(target string, err error) {
	l.Warn("Proxy failure", "target", target, "error", err)
}

// LogAllTargetsFailed logs when all targets fail
func (l *Logger) LogAllTargetsFailed(method, path string) {
	l.Error("All targets failed", "method", method, "path", path)
}
