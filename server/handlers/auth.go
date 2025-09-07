package handlers

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/GrGLeo/ctf/pkg/shared"
	"github.com/GrGLeo/ctf/server/event"
	"github.com/GrGLeo/ctf/server/proto/auth"
	pb "github.com/GrGLeo/ctf/server/proto/auth"
	ratelimiter "github.com/GrGLeo/ctf/server/rate_limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	Client      pb.AuthServiceClient
	broker      *event.EventBroker
	rateLimiter *ratelimiter.GlobalRateLimiter
	logger      *slog.Logger
}

func NewAuthClient(broker *event.EventBroker, logger *slog.Logger, rateLimiter *ratelimiter.GlobalRateLimiter) (*AuthClient, error) {
	authServiceAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authServiceAddr == "" {
		authServiceAddr = "localhost:50051"
	}
	conn, err := grpc.NewClient(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := auth.NewAuthServiceClient(conn)
	return &AuthClient{
		Client:      client,
		broker:      broker,
		rateLimiter: rateLimiter,
		logger:      logger,
	}, nil
}

func (ac *AuthClient) HandleRegistration(msg event.Message) event.Message {
	req := msg.(event.RegisterRequestMessage)
	ip, err := shared.ExtractIP(req.Conn)
	if err != nil {
		ac.logger.Error("failed to extract TCP IP from connection", "component", "auth", "error", err)
		return event.RegisterResponseMessage{
			Success:   false,
			Message:   "Failed to extract TCP IP",
			Challenge: nil,
			Conn:      req.Conn,
		}
	}
	allowed, err := ac.rateLimiter.Allow(ip, req.Type(), true)

	if err != nil {
		ac.logger.Error("Failed to retrieve bucket", "component", "auth", "error", err, "ip", ip)
		return event.RegisterResponseMessage{
			Success:   false,
			Message:   "Internal server error",
			Challenge: nil,
			Conn:      req.Conn,
		}
	}

	if !allowed {
		ac.logger.Warn("Rate limit exceed", "component", "auth", "ip", ip, "username", req.Username)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regResp, err := ac.Client.Register(ctx, &auth.RegisterRequest{
		Username:  req.Username,
		PublicKey: req.PublicKey,
	})

	if err != nil {
		ac.logger.Error("gRPC Register call failed", "component", "auth", "error", err, "username", req.Username)
		return event.RegisterResponseMessage{
			Success:   false,
			Message:   "Internal server error",
			Challenge: nil,
			Conn:      req.Conn,
		}
	}

	if !regResp.Success {
		return event.RegisterResponseMessage{
			Success:   regResp.Success,
			Message:   regResp.Message,
			Challenge: nil,
			Conn:      req.Conn,
		}
	} else {
		return event.RegisterResponseMessage{
			Success:   regResp.Success,
			Message:   regResp.Message,
			Challenge: regResp.Challenge,
			Conn:      req.Conn,
		}
	}
}

func (ac *AuthClient) HandleLoginChallenge(msg event.Message) event.Message {
	req := msg.(event.LoginChallengeRequestMessage)
	ip, err := shared.ExtractIP(req.Conn)
	if err != nil {
		ac.logger.Error("Failed to extract TCP IP from connection", "component", "auth", "error", err)
		return event.LoginChallengeResponseMessage{
			Challenge: nil,
			Conn:      req.Conn,
		}
	}
	allowed, err := ac.rateLimiter.Allow(ip, req.Type(), true)

	if err != nil {
		ac.logger.Error("Failed to retrieve bucket", "component", "auth", "error", err, "ip", ip)
		return event.LoginChallengeResponseMessage{
			Challenge: nil,
			Conn:      req.Conn,
		}
	}

	if !allowed {
		ac.logger.Warn("Rate limit exceed", "component", "auth", "ip", ip, "username", req.Username)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	challengeResp, err := ac.Client.GetLoginChallenge(ctx, &auth.GetLoginChallengeRequest{
		Username: req.Username,
	})
	if err != nil {
		return event.LoginChallengeResponseMessage{
			Challenge: nil,
			Conn:      req.Conn,
		}
	}

	return event.LoginChallengeResponseMessage{
		Challenge: challengeResp.Challenge,
		Conn:      req.Conn,
	}
}

func (ac *AuthClient) HandleAuth(msg event.Message) event.Message {
	req := msg.(event.AuthRequestMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authResp, err := ac.Client.Authentificate(ctx, &auth.AuthentificateRequest{
		Username:        req.Username,
		SignedChallenge: req.SignedChallenge,
	})
	if err != nil {
		return event.AuthResponseMessage{
			Success:      false,
			Message:      "Error verifying the signed challenge",
			SessionToken: "",
			Conn:         req.Conn,
		}
	}

	if !authResp.Success {
		return event.AuthResponseMessage{
			Success:      false,
			Message:      authResp.Message,
			SessionToken: "",
			Conn:         req.Conn,
		}
	}

	// When the client connect the room is the main lobby with ID 0
	regResponseCh := make(chan event.Message, 1)
	clientRegistration := event.ClientRegistrationMessage{
		ClientID:   req.Username,
		RoomID:     0,
		Conn:       req.Conn,
		ResponseCh: regResponseCh,
	}
	ac.broker.Publish(clientRegistration)

	// Wait for client registration to complete
	regResponse := <-regResponseCh
	if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && !regResp.Success {
		return event.AuthResponseMessage{
			Success:      false,
			Message:      "Failed to register client: " + regResp.Message,
			SessionToken: "",
			Conn:         req.Conn,
		}
	}

	// TODO: Generate a real session token
	return event.AuthResponseMessage{
		Success:      true,
		Message:      authResp.Message,
		SessionToken: "a-real-session-token",
		Conn:         req.Conn,
	}
}
