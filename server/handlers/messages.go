package handlers

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	conm "github.com/GrGLeo/ctf/server/conn_manager"
	"github.com/GrGLeo/ctf/server/event"
	pb "github.com/GrGLeo/ctf/shared/proto/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessagesServiceClient struct {
	Client      pb.MessageServiceClient
	connManager *conm.ConnectionManager
}

func NewMessageServiceClient(connManager *conm.ConnectionManager) (*MessagesServiceClient, error) {
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
		log.Printf("gRPC Register call failed: %v", err)
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
		log.Printf("gRPC Unregister call failed: %v", err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ms.Client.RouteMessage(ctx, &pb.RouteMessageRequest{
		Sender:  req.Sender,
		Content: req.Message,
	})
	if err != nil {
		log.Printf("gRPC Unregister call failed: %v", err)
    //TODO: how to handle error
		return event.MessageResponseMessage{
		}
	}
	return event.MessageResponseMessage{
		Receivers: resp.Receivers,
		Message:   resp.Content,
	}
}
