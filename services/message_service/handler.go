package main

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	pb "github.com/GrGLeo/ctf_game/pkg/shared/proto/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ManagerInterface defines the contract for message management
type ManagerInterface interface {
	RegisterClient(client string, roomID int) error
	UnregisterClient(client string) error
	RouteMessage(sender string, content string) ([]string, string, error)
}

type MessageHandler struct {
	pb.UnimplementedMessageServiceServer
	manager ManagerInterface
	logger  *slog.Logger
}

func NewMessageHandler(manager ManagerInterface, logger *slog.Logger) *MessageHandler {
	return &MessageHandler{
		manager: manager,
		logger:  logger,
	}
}

func (mh *MessageHandler) RouteMessage(ctx context.Context, req *pb.RouteMessageRequest) (*pb.RouteMessageResponse, error) {
	mh.logger.Debug("[MESSAGE SERVICE] RouteMessage called", "sender", req.Sender, "content", req.Content)

	receivers, message, err := mh.manager.RouteMessage(req.Sender, req.Content)
	if err != nil {
		mh.logger.Error("[MESSAGE SERVICE] Failed to route message", "sender", req.Sender, "error", err)
		return nil, MapToGRPCError(err)
	}

	mh.logger.Debug("[MESSAGE SERVICE] RouteMessage completed", "sender", req.Sender, "receivers", receivers, "processed_message", message)
	return &pb.RouteMessageResponse{
		Receivers: receivers,
		Content:   message,
	}, nil
}

func (mh *MessageHandler) RegisterClient(ctx context.Context, req *pb.RegisterClientRequest) (*pb.RegisterClientResponse, error) {
	if req.Client == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client ID cannot be empty")
	}
	roomID, err := strconv.Atoi(req.RoomId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse room ID")
	}

	err = mh.manager.RegisterClient(req.Client, roomID)
	if err != nil {
		mh.logger.Error("Failed to register client", "client", req.Client, "error", err)
		return nil, MapToGRPCError(err)
	}
	return &pb.RegisterClientResponse{}, nil
}
func (mh *MessageHandler) UnregisterClient(ctx context.Context, req *pb.UnregisterClientRequest) (*pb.UnregisterClientResponse, error) {
	if err := mh.manager.UnregisterClient(req.Client); err != nil {
		mh.logger.Error("Failed to unregister client", "client", req.Client, "error", err)
		return nil, MapToGRPCError(err)
	}
	return &pb.UnregisterClientResponse{}, nil
}

func MapToGRPCError(err error) error {
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "already in the room"):
		return status.Errorf(codes.AlreadyExists, errMsg)
	case strings.Contains(errMsg, "not registered"):
		return status.Errorf(codes.NotFound, errMsg)
	case strings.Contains(errMsg, "cannot be empty"):
		return status.Errorf(codes.InvalidArgument, errMsg)
	case strings.Contains(errMsg, "not found"):
		return status.Errorf(codes.NotFound, errMsg)
	case strings.Contains(errMsg, "Failed to find"):
		return status.Errorf(codes.NotFound, errMsg)
	case strings.Contains(errMsg, "not in room"):
		return status.Errorf(codes.NotFound, errMsg)
	case strings.Contains(errMsg, "too long"):
		return status.Errorf(codes.InvalidArgument, errMsg)
	default:
		return status.Errorf(codes.Internal, errMsg)
	}
}
