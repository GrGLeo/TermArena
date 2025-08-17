package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GrGLeo/ctf/server/event"
	"github.com/GrGLeo/ctf/server/proto/auth"
	pb "github.com/GrGLeo/ctf/server/proto/auth"
	"github.com/GrGLeo/ctf/shared"
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

func (ac AuthClient) HandleRegistration(msg event.RegisterRequestMessage) {
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  regResp, err := ac.Client.Register(ctx, &auth.RegisterRequest{
    Username: msg.Username,
    PublicKey: msg.PublicKey,
  })

  if err != nil {
    errPacket := shared.NewRegisterResponsePacket(false, "Internal server error", nil)
    msg.Conn.Write(errPacket.Serialize())
    return
  }

  if !regResp.Success {
    failPacket := shared.NewRegisterResponsePacket(false, regResp.Message, nil)
    msg.Conn.Write(failPacket.Serialize())
    return
  }

  challengeResp, err := ac.Client.GetLoginChallenge(ctx, &auth.GetLoginChallengeRequest{
    Username: msg.Username,
  })
  
  if err != nil {
    errPacket := shared.NewRegisterResponsePacket(false, "Error getting Login challenge", nil)
    msg.Conn.Write(errPacket.Serialize())
    return
  }
  
  successPacket := shared.NewRegisterResponsePacket(true, regResp.Message, challengeResp.Challenge)
  msg.Conn.Write(successPacket.Serialize())
  return
}

func (ac AuthClient) HandleLoginChallenge(msg event.LoginChallengeRequestMessage) {
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  challengeResp, err := ac.Client.GetLoginChallenge(ctx, &auth.GetLoginChallengeRequest{
    Username: msg.Username,
  })
  if err != nil {
    errPacket := shared.NewLoginChallengeResponsePacket(nil)
    msg.Conn.Write(errPacket.Serialize())
    return
  }

  successPacket := shared.NewLoginChallengeResponsePacket(challengeResp.Challenge)
  msg.Conn.Write(successPacket.Serialize())
  return
}

func (ac AuthClient) HandleAuth(msg event.AuthRequestMessage) {
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  authResp, err := ac.Client.Authentificate(ctx, &auth.AuthentificateRequest{
    Username: msg.Username,
    SignedChallenge: msg.SignedChallenge,
  })
  if err != nil {
    errPacket := shared.NewAuthResponsePacket(false, "Error verifying the signed challenge", "")
    msg.Conn.Write(errPacket.Serialize())
    return
  }

  if !authResp.Success {
    failPacket := shared.NewAuthResponsePacket(false, authResp.Message, "")
    msg.Conn.Write(failPacket.Serialize())
    return
  }

  successPacket := shared.NewAuthResponsePacket(true, authResp.Message, "a")
  msg.Conn.Write(successPacket.Serialize())
  return
}

