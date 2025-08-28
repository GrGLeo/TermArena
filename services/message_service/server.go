package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type MessageServer struct {
	config     *Config
	logger     *slog.Logger
	grpcServer *grpc.Server
}

func NewMessageServer(config *Config, logger *slog.Logger) *MessageServer {
	return &MessageServer{
		config: config,
		logger: logger,
	}
}

func (s *MessageServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Host, s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.grpcServer = grpc.NewServer()
	s.logger.Info("gRPC server starting", "host", s.config.Host, "port", s.config.Port)
	return s.grpcServer.Serve(lis)
}

func (s *MessageServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Initiating graceful shutdown", "timeout_seconds", s.config.ShutdownTimeoutSeconds)
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("gRPC server stop gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout exceed. Forcing shutdown")
		s.grpcServer.Stop()
		return ctx.Err()
	}
}
