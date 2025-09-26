package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/message"
	conm "github.com/GrGLeo/TermArena/server/conn_manager"
	"github.com/GrGLeo/TermArena/server/event"
	ratelimiter "github.com/GrGLeo/TermArena/server/rate_limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessagesServiceClient struct {
	Client      pb.MessageServiceClient
	connManager *conm.ConnectionManager
	rateLimiter *ratelimiter.GlobalRateLimiter
	logger      *slog.Logger
}

func NewMessageServiceClient(connManager *conm.ConnectionManager, logger *slog.Logger, rateLimiter *ratelimiter.GlobalRateLimiter) (*MessagesServiceClient, error) {
	messageServiceAddr := os.Getenv("MESSAGE_SERVICE_ADDR")
	if messageServiceAddr == "" {
		messageServiceAddr = "localhost:8083"
	}
	conn, err := grpc.NewClient(messageServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewMessageServiceClient(conn)
	return &MessagesServiceClient{
		Client:      client,
		connManager: connManager,
		rateLimiter: rateLimiter,
		logger:      logger,
	}, nil
}

func (ms *MessagesServiceClient) HandleClientRegistration(msg event.Message) event.Message {
	req := msg.(event.ClientRegistrationMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ms.Client.RegisterClient(ctx, &pb.RegisterClientRequest{
		Client: req.ClientID,
		RoomId: req.RoomID,
    TeamId: req.TeamID,
	})

	if err != nil {
		ms.logger.Error("gRPC Register call failed", "component", "messages", "error", err, "client_id", req.ClientID)
		return event.ClientRegistrationResponse{
			Success:  false,
			Message:  "Failed to register with message service",
			ClientID: req.ClientID,
		}
	}
	ms.connManager.Register(req.Conn, req.ClientID)
	return event.ClientRegistrationResponse{
		Success:  true,
		Message:  "Registration successful",
		ClientID: req.ClientID,
	}
}

func (ms *MessagesServiceClient) HandleClientUnregistration(msg event.Message) event.Message {
	req := msg.(event.ClientUnregistrationMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ms.Client.UnregisterClient(ctx, &pb.UnregisterClientRequest{
		Client: req.ClientID,
	})

	if err != nil {
		ms.logger.Error("gRPC Unregister call failed", "component", "messages", "error", err, "client_id", req.ClientID)
		return event.ClientUnregistrationResponse{
			Success:  false,
			Message:  "Failed to unregister with message service",
			ClientID: req.ClientID,
		}
	}
	return event.ClientUnregistrationResponse{
		Success:  true,
		Message:  "Unregistration successful",
		ClientID: req.ClientID,
	}
}

func (ms *MessagesServiceClient) HandleRouteMessage(msg event.Message) event.Message {
	req := msg.(event.MessageRequestMessage)
	allowed, err := ms.rateLimiter.Allow(req.User, req.Type(), false)

	if err != nil {
		ms.logger.Error("Failed to retrieve bucket", "component", "messages", "error", err, "user", req.User)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}

	if !allowed {
		ms.logger.Warn("Rate limit exceed", "component", "messages", "username", req.User)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}
	ms.logger.Info("HandleRouteMessage called", "component", "messages", "sender", req.Sender, "message", req.Message)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ms.logger.Debug("Calling message service", "component", "messages", "sender", req.Sender)
	resp, err := ms.Client.RouteMessage(ctx, &pb.RouteMessageRequest{
		Sender:  req.Sender,
		Content: req.Message,
	})
	if err != nil {
		ms.logger.Error("gRPC Route call failed", "component", "messages", "error", err, "sender", req.Sender)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}

	ms.logger.Debug("Message service responded", "component", "messages", "receivers", resp.Receivers, "response_content", resp.Content)
	return event.MessageResponseMessage{
		Receivers:  resp.Receivers,
		Message:    resp.Content,
		ResponseCh: req.ResponseCh,
	}
}
