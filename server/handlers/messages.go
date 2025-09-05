package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	pb "github.com/GrGLeo/ctf/pkg/shared/proto/message"
	conm "github.com/GrGLeo/ctf/server/conn_manager"
	"github.com/GrGLeo/ctf/server/event"
	ratelimiter "github.com/GrGLeo/ctf/server/rate_limiter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessagesServiceClient struct {
	Client      pb.MessageServiceClient
	connManager *conm.ConnectionManager
	rateLimiter *ratelimiter.GlobalRateLimiter
	logger      *zap.SugaredLogger
}

func NewMessageServiceClient(connManager *conm.ConnectionManager, logger *zap.SugaredLogger, rateLimiter *ratelimiter.GlobalRateLimiter) (*MessagesServiceClient, error) {
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
		RoomId: strconv.Itoa(req.RoomID),
	})

	if err != nil {
		ms.logger.Errorw("gRPC Register call failed", "error", err, "client_id", req.ClientID)
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
		ms.logger.Errorw("gRPC Unregister call failed", "error", err, "client_id", req.ClientID)
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
		ms.logger.Errorw("[SERVER HANDLER] Failed to retrieve bucket", "error", err, "user", req.User)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}

	if !allowed {
		ms.logger.Warn("[SERVER HANDLER] Rate limit exceed", "username", req.User)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}
	ms.logger.Infow("[SERVER HANDLER] HandleRouteMessage called", "sender", req.Sender, "message", req.Message)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ms.logger.Infow("[SERVER HANDLER] Calling message service", "sender", req.Sender)
	resp, err := ms.Client.RouteMessage(ctx, &pb.RouteMessageRequest{
		Sender:  req.Sender,
		Content: req.Message,
	})
	if err != nil {
		ms.logger.Errorw("[SERVER HANDLER] gRPC Route call failed", "error", err, "sender", req.Sender)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}

	ms.logger.Infow("[SERVER HANDLER] Message service responded", "receivers", resp.Receivers, "response_content", resp.Content)
	return event.MessageResponseMessage{
		Receivers:  resp.Receivers,
		Message:    resp.Content,
		ResponseCh: req.ResponseCh,
	}
}
