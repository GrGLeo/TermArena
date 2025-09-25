package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/GrGLeo/TermArena/pkg/shared"
	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
	conm "github.com/GrGLeo/TermArena/server/conn_manager"
	"github.com/GrGLeo/TermArena/server/event"
	ratelimiter "github.com/GrGLeo/TermArena/server/rate_limiter"
	manager "github.com/GrGLeo/TermArena/server/room_manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RoomServiceClient struct {
	Client        pb.RoomServiceClient
	connManager   *conm.ConnectionManager
	broker        *event.EventBroker
	rateLimiter   *ratelimiter.GlobalRateLimiter
	logger        *slog.Logger
	stream        pb.RoomService_NotifyRoomChangesClient
	streamContext context.Context
	steamCancel   context.CancelFunc
	portCounter   uint32
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

	streamCtx, streamCancel := context.WithCancel(context.Background())
	stream, err := client.NotifyRoomChanges(streamCtx)
	if err != nil {
		streamCancel() // Clean up context
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	roomClient := &RoomServiceClient{
		Client:        client,
		connManager:   connManager,
		broker:        broker,
		rateLimiter:   rateLimiter,
		logger:        logger,
		stream:        stream,
		streamContext: streamCtx,
		steamCancel:   streamCancel,
		portCounter:   50053,
	}

	go roomClient.handleNotifications()

	return roomClient, nil

}

func (rs *RoomServiceClient) HandleLookRoom(msg event.Message) event.Message {
	req := msg.(event.RoomRequestMessage)

	// Rate limiting checks
	allowed, err := rs.rateLimiter.Allow(req.Username, req.Type(), false)
	if err != nil {
		rs.logger.Error("Failed to retrieve bucket", "component", "room_manager", "error", err, "user", req.Username)
		return event.MessageErrorResponse{
			Error:      fmt.Sprintf("Failed to route message: %v", err),
			ResponseCh: req.ResponseCh,
		}
	}
	if !allowed {
		rs.logger.Warn("Rate limit exceed", "component", "room_manager", "username", req.Username)
		return event.RateLimitResponse{ResponseCh: req.ResponseCh}
	}
	rs.logger.Info("HandleLookRoom called", "component", "room_manager", "sender", req.Username, "roomType", req.RoomType)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := rs.Client.LookRoom(ctx, &pb.LookRoomRequest{
		RoomType: uint32(req.RoomType),
		Username: req.Username,
	})

	if err != nil {
		rs.logger.Error("gRPC LookRoom call failed", "component", "room_manager", "error", err, "client_id", req.Username)
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
		Success:    true,
		RoomID:     res.RoomID,
		ResponseCh: req.ResponseCh,
	}
}

func (rs *RoomServiceClient) HandleUpdateSpell(msg event.Message) event.Message {
	req := msg.(event.UpdateSpellReqMessage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := rs.Client.UpdateSpell(ctx, &pb.UpdateSpellRequest{
		Username: req.Username,
		RomType:  uint32(req.RoomType),
		RoomID:   uint32(req.RoomID),
		Spell1:   uint32(req.SpellOne),
		Spell2:   uint32(req.SpellTwo),
	})
	if err != nil {
		rs.logger.Error("gRPC UpdateSpell call failed", "component", "room_manager", "error", err, "client_id", req.Username)
	}

	return event.UpdateSpellResMessage{
		Usernames:  res.Usernames,
		Username:   req.Username,
		SpellOne:   req.SpellOne,
		SpellTwo:   req.SpellTwo,
		ResponseCh: req.ResponseCh,
	}

}

func (rs *RoomServiceClient) handleNotifications() {
	rs.logger.Info("Stream connected. Waiting for notifications...")
	for {
		select {
		// Check if the stream context is done (e.g., canceled)
		case <-rs.streamContext.Done():
			rs.logger.Info("Stream closed or canceled")
			return

		default:
			// Receive a notification from the server
			notification, err := rs.stream.Recv()
			if err == io.EOF {
				rs.logger.Info("Server closed the stream")
				return
			}
			if err != nil {
				rs.logger.Error("Failed to receive notification", "error", err)
				return
			}

			rs.logger.Info(
				"Received notification",
				"room_id", notification.RoomID,
				"ready", notification.Ready,
				"user_infos", notification.UserInfos,
			)

			// Send an Ack back to the server
			ack := &pb.Ack{
				Success: true,
			}
			if err := rs.stream.Send(ack); err != nil {
				rs.logger.Error("Failed to send Ack", "error", err)
				return
			}
			rs.logger.Info("Sent Ack", "room_id", notification.RoomID)

			var data []byte
			if !notification.Ready {
				packet := shared.NewMoveToLobbyPacket(notification.RoomID, notification.UserInfos)
				data = packet.Serialize()
			} else {
				// Start the game
				port := atomic.LoadUint32(&rs.portCounter)
				var usernames []string
				var teams []string
				var spell1s []string
				var spell2s []string
				for _, ui := range notification.UserInfos {
					usernames = append(usernames, ui.Username)
					teams = append(teams, strconv.Itoa(int(ui.Team)))
					spell1s = append(spell1s, strconv.Itoa(int(ui.Spell1)))
					spell2s = append(spell2s, strconv.Itoa(int(ui.Spell2)))
				}
				portStr := strconv.Itoa(int(port))
				maxPlayers := strconv.Itoa(len(usernames))
				err := manager.StartGame(portStr, maxPlayers, notification.RoomID, usernames, teams, spell1s, spell2s)
				if err != nil {
					rs.logger.Error("Failed to start game", "error", err)
				}
				// Increment port
				newPort := port + 1
				if newPort > 50153 {
					newPort = 50053
				}
				atomic.StoreUint32(&rs.portCounter, newPort)
        // Create a new packet to notify each client the port to connect
        packet := shared.NewGameServerReadyPacket(uint16(port))
        data = packet.Serialize()
			}

			for _, userInfo := range notification.UserInfos {
				receiverID := userInfo.Username
				if receiverConn, exist := rs.connManager.GetConn(receiverID); exist {
					if _, err := receiverConn.Write(data); err != nil {
						rs.logger.Error("Error writing response to client", "component", "server", "receiver", receiverID, "error", err)
					}
				} else {
					rs.logger.Warn("Could not find connection for receiver", "component", "server", "receiver", receiverID)
				}
			}
		}
	}
}
