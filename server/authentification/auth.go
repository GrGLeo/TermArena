package auth

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/GrGLeo/ctf/server/event"
	"github.com/GrGLeo/ctf/server/proto/auth"
	pb "github.com/GrGLeo/ctf/server/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	Client pb.AuthServiceClient
}

func NewAuthClient() (*AuthClient, error) {
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
		Client: client,
	}, nil
}

func (ac *AuthClient) HandleRegistration(msg event.Message) event.Message {
	req := msg.(event.RegisterRequestMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	regResp, err := ac.Client.Register(ctx, &auth.RegisterRequest{
		Username:  req.Username,
		PublicKey: req.PublicKey,
	})

	if err != nil {
		log.Printf("gRPC Register call failed: %v", err)
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

	// TODO: Generate a real session token
	return event.AuthResponseMessage{
		Success:      true,
		Message:      authResp.Message,
		SessionToken: "a-real-session-token",
		Conn:         req.Conn,
	}
}
