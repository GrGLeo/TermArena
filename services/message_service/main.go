package main

import (
	"log/slog"
	"os"
)

func NewLogger(level string) *slog.Logger {
	// Parse log level with fallback to Info
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo // Default to Info level
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler).With("service", "message_service")
}

func main() {
	config := NewConfig()

	// Initialize logger
	logger := NewLogger(config.LogLevel)
	slog.SetDefault(logger)

	// Log service startup
	slog.Info("Message service starting",
		"log_level", config.LogLevel,
		"version", "1.0.0",
	)
}
