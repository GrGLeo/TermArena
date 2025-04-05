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
	if msg.Type() != "login" {
		fmt.Println("Invalid message type for FindRoom")
		return nil
	}

	loginRequest, ok := msg.(event.LoginMessage)
	if !ok {
		fmt.Println("Failed to cast message to RoomRequestMessage")
		return nil
	}

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
		Username: loginRequest.Username,
		Password: loginRequest.Password,
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

func SignIn(msg event.Message) event.Message {
	if msg.Type() != "signin" {
		fmt.Println("Invalid message type for FindRoom")
		return nil
	}

	signinMessage, ok := msg.(event.SignInMessage)
	if !ok {
		fmt.Println("Failed to cast message to SignInMessage")
		return nil
	}
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Error: %q", err.Error())
		return event.AuthMessage{Success: 0}
	}
	defer conn.Close()

	c := pb.NewCreateServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r := &pb.SigninRequest{
		Username: signinMessage.Username,
		Password: signinMessage.Password,
	}

	resp, err := c.Signin(ctx, r)
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
