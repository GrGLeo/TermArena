package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
	config "github.com/GrGLeo/TermArena/services/room_manager_service/config"
	"google.golang.org/grpc"
)

type RoomServer struct {
	config     *config.Config
	logger     *slog.Logger
	handler    *RoomHandler
	grpcServer *grpc.Server
}

func NewRoomServer(config *config.Config, logger *slog.Logger) *RoomServer {
  changes := make(chan *pb.RoomChangeNotification, 100)
	manager := NewRoomManager(changes, config.MaxRoom, logger)
	handler := NewRoomHandler(changes, manager, logger)
	return &RoomServer{
		config:  config,
		logger:  logger,
		handler: handler,
	}
}

func (s *RoomServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Host, s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.grpcServer = grpc.NewServer()
	s.logger.Info("gRPC server starting", "host", s.config.Host, "port", s.config.Port)
	pb.RegisterRoomServiceServer(s.grpcServer, s.handler)

	return s.grpcServer.Serve(lis)
}

func (s *RoomServer) Shutdown(ctx context.Context) error {
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
