package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/GrGLeo/ctf/server/event"
	pb "github.com/GrGLeo/ctf/server/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Authentificate(msg event.Message) event.Message {
  conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
  if err != nil {
    fmt.Printf("Error: %q", err.Error())
    return event.AuthMessage{Success: 0}
  }
  defer conn.Close()

  c := pb.NewLoginServiceClient(conn)
  ctx, cancel := context.WithTimeout(context.Background(), time.Second)
  defer cancel()

  r := &pb.AuthentificationRequest{
    Username: "heelo",
    Password: "passd123",
  }

  resp, err := c.Authentificate(ctx, r)
  if err != nil {
    fmt.Printf("Error on sending message: %q", err.Error())
    return event.AuthMessage{Success: 0}
  }

  if resp.Success {
    return event.AuthMessage{Success: 1}
  } else {
    return event.AuthMessage{Success: 0}
  }
}
