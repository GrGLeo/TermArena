package internal

import (
	"log/slog"
  "fmt"

	pb "github.com/GrGLeo/ctf_game/pkg/shared/proto/message"
	"google.golang.org/grpc"
)

type Server struct {
	config *Config
	logger *slog.Logger
  handler *RoomManagerHandler
	server *grpc.Server
}

func (s *MessageServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Host, s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.grpcServer = grpc.NewServer()
	s.logger.Info("gRPC server starting", "host", s.config.Host, "port", s.config.Port, "component", "MessageService")
	pb.RegisterMessageServiceServer(s.grpcServer, s.handler)

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
