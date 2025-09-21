package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
	conm "github.com/GrGLeo/TermArena/server/conn_manager"
	"github.com/GrGLeo/TermArena/server/event"
	ratelimiter "github.com/GrGLeo/TermArena/server/rate_limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RoomServiceClient struct {
	Client      pb.RoomServiceClient
	connManager *conm.ConnectionManager
	broker      *event.EventBroker
	rateLimiter *ratelimiter.GlobalRateLimiter
	logger      *slog.Logger
}

func NewRoomServiceClient(connManager *conm.ConnectionManager, broker *event.EventBroker, logger *slog.Logger, rateLimiter *ratelimiter.GlobalRateLimiter) (*RoomServiceClient, error) {
	messageServiceAddr := os.Getenv("MESSAGE_SERVICE_ADDR")
	if messageServiceAddr == "" {
		messageServiceAddr = "localhost:8084"
	}
	conn, err := grpc.NewClient(messageServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewRoomServiceClient(conn)
	return &RoomServiceClient{
		Client:      client,
		connManager: connManager,
    broker: broker,
		rateLimiter: rateLimiter,
		logger:      logger,
	}, nil
}

func (rs *RoomServiceClient) HandleLookRoom(msg event.Message) event.Message {
	req := msg.(event.RoomRequestMessage)

	// Rate limiting checks
	allowed, err := rs.rateLimiter.Allow(req.Username, req.Type(), false)
	if err != nil {
		rs.logger.Error("Failed to retrieve bucket", "component", "messages", "error", err, "user", req.Username)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}
	if !allowed {
		rs.logger.Warn("Rate limit exceed", "component", "messages", "username", req.Username)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}
	rs.logger.Info("HandleLookRoom called", "component", "messages", "sender", req.Username, "roomType", req.RoomType)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := rs.Client.LookRoom(ctx, &pb.LookRoomRequest{
		RoomType: uint32(req.RoomType),
		Username: req.Username,
	})

	if err != nil {
		rs.logger.Error("gRPC LookRoom call failed", "component", "messages", "error", err, "client_id", req.Username)
		return event.LookRoomResponseMessage{
			Success: false,
			RoomID:  res.RoomID,
		}
	}

	// We move the client to the assigned room
	regResponseCh := make(chan event.Message, 1)
	clientRegistration := event.ClientRegistrationMessage{
		ClientID:   req.Username,
		RoomID:     res.RoomID,
		Conn:       req.Conn,
		ResponseCh: regResponseCh,
	}
	rs.broker.Publish(clientRegistration)

	// Wait for client registration to complete
	//_ = <-regResponseCh
	return event.LookRoomResponseMessage{
		Success: true,
		RoomID:  res.RoomID,
    ResponseCh: req.ResponseCh,
	}
}
