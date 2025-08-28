package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	logger := NewLogger(config.LogLevel)
	slog.SetDefault(logger)
	server := NewMessageServer(config, logger)

	// channel  to listen for interruption
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("Shutting down Message service ...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Service forced to shutdown!", "error", err)
		os.Exit(1)
	}

	logger.Info("Service exited")
}
