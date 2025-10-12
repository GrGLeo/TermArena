package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	conm "github.com/GrGLeo/TermArena/server/conn_manager"
	"github.com/GrGLeo/TermArena/server/event"
	handler "github.com/GrGLeo/TermArena/server/handlers"
	ratelimiter "github.com/GrGLeo/TermArena/server/rate_limiter"

	//manager "github.com/GrGLeo/TermArena/server/room_manager"

	"github.com/GrGLeo/TermArena/pkg/shared"
	"github.com/joho/godotenv"
)

type CtxKey string

const (
	loggerKey CtxKey = "logger"
)

var (
	env                   string
	port                  string
	rateLimiterConfigPath string
)

func init() {
	godotenv.Load("server/.env")
	env = os.Getenv("ENV")
	port = os.Getenv("SERVER")
	rateLimiterConfigPath = os.Getenv("RATE_LIMITER_CONFIG_PATH")
}

func NewLogger(env string) *slog.Logger {
	var level slog.Level
	if env == "DEV" {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func HandleClient(ctx context.Context, server *net.TCPListener, connChannel chan *net.TCPConn) {
	log, _ := ctx.Value(loggerKey).(*slog.Logger)
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			log.Error("Failed to accept connection", "component", "server", "error", err)
			continue
		}
		log.Info("Accept new connection", "component", "server", "ip", conn.RemoteAddr())
		connChannel <- conn
	}
}

func ProcessClient(conn *net.TCPConn, log *slog.Logger, broker *event.EventBroker, connManager *conm.ConnectionManager) {
	buffer := make([]byte, 4096) // A persistent buffer for each client
	var data []byte

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				log.Info("Client disconnected", "component", "server", "ip", conn.RemoteAddr())
				// Unregister the client
				client, exist := connManager.Unregister(conn)
				if exist {
					responseChan := make(chan event.Message)
					msg := event.ClientUnregistrationMessage{
						ClientID:   client,
						ResponseCh: responseChan,
					}
					broker.Publish(msg)
				}
			} else {
				log.Error("Error reading from client", "component", "server", "ip", conn.RemoteAddr(), "error", err)
			}
			return // Exit if there's an error or if the client disconnects
		}

		if n > 0 {
			data = append(data, buffer[:n]...)
			for len(data) > 0 {
				packet, bytesConsumed, err := shared.DeSerialize(data)
				if err != nil {
					if err.Error() == "incomplete packet" {
						log.Debug("Waiting for more data", "component", "server", "ip", conn.RemoteAddr(), "buffer_size", len(data))
						break // Wait for more data
					}
					log.Error("Error deserializing packet", "component", "server", "ip", conn.RemoteAddr(), "error", err)
					data = nil // Clear buffer on persistent error
					continue
				}

				msg, err := event.CreateMessage(packet, conn, connManager)
				if err != nil {
					log.Error("Error creating message from packet", "component", "server", "ip", conn.RemoteAddr(), "error", err)
					data = data[bytesConsumed:]
					continue
				}

				// Unified logic: publish message, wait for response, create packet, send response.
				log.Debug("Publishing message to broker", "component", "server", "message_type", msg.Type(), "ip", conn.RemoteAddr())
				broker.Publish(msg)
				log.Debug("Waiting for response from broker", "component", "server", "message_type", msg.Type())
				response := <-msg.ResponseChan()
				log.Debug("Received response from broker", "component", "server", "response_type", response.Type(), "ip", conn.RemoteAddr())

				switch resp := response.(type) {
				case event.MessageResponseMessage:
					log.Debug("MessageResponseMessage found", "component", "server", "message", resp.Message)
					responsePacket, err := event.CreatePacketFromMessage(resp)
					if err != nil {
						log.Error("Error creating packet from message", "component", "server", "error", err)
						data = data[bytesConsumed:]
						continue
					}
					for _, receiverID := range resp.Receivers {
						receiverConn, exist := connManager.GetConn(receiverID)
						if exist {
							if _, err := receiverConn.Write(responsePacket); err != nil {
								log.Error("Error writing response to client", "component", "server", "receiver", receiverID, "error", err)
							}
						} else {
							log.Warn("Could not find connection for receiver", "component", "server", "receiver", receiverID)
						}
					}
					data = data[bytesConsumed:]
				case event.UpdateSpellResMessage:
					responsePacket, err := event.CreatePacketFromMessage(resp)
					if err != nil {
						log.Error("Error creating packet from message", "component", "server", "error", err)
						data = data[bytesConsumed:]
						continue
					}
					for _, receiverID := range resp.Usernames {
						receiverConn, exist := connManager.GetConn(receiverID)
						if exist {
							if _, err := receiverConn.Write(responsePacket); err != nil {
								log.Error("Error writing response to client", "component", "server", "receiver", receiverID, "error", err)
							}
						} else {
							log.Warn("Could not find connection for receiver", "component", "server", "receiver", receiverID)
						}
					}
					data = data[bytesConsumed:]
				default:
					responsePacket, err := event.CreatePacketFromMessage(resp)
					if err != nil {
						log.Error("Error creating packet from message", "component", "server", "error", err)
						data = data[bytesConsumed:]
						continue
					}
					if _, err := conn.Write(responsePacket); err != nil {
						log.Error("Error writing response to client", "component", "server", "ip", conn.RemoteAddr(), "error", err)
					}
					data = data[bytesConsumed:]
				}
			}
		}
	}
}

func main() {
	log := NewLogger(env)
	log.Info("Starting server", "component", "server")
	serverAddr, err := net.ResolveTCPAddr("tcp", ":8082")
	if err != nil {
		log.Error("Failed to resolve TCP Addr", "component", "server", "error", err)
		os.Exit(1)
	}
	server, err := net.ListenTCP("tcp", serverAddr)
	if err != nil {
		log.Error("Failed to launch TCP server", "component", "server", "error", err)
		os.Exit(1)
	}
	log.Info("Server started and listening", "component", "server", "address", serverAddr.String())
	connectionManager := conm.NewConnectionManager()
	connChannel := make(chan *net.TCPConn)
	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, log)

	broker := event.NewEventBroker(log, 10)
	log.Info("New Event Broker initialized", "component", "server", "worker_pool_size", 10)

	rateLimiter, err := ratelimiter.NewGlobalRateLimiter(rateLimiterConfigPath)
	if err != nil {
    path, _ := os.Getwd()
		log.Error("Failed to create rate limiter", "component", "server", "error", err, "path", rateLimiterConfigPath, "current_path", path)
		os.Exit(1)
	}

	roomManager, err := handler.NewRoomServiceClient(connectionManager, broker, log, rateLimiter)

	//roomManager := manager.NewRoomManager(log, broker, rateLimiter)
	log.Info("New room manager initialized", "component", "server")

	authClient, err := handler.NewAuthClient(broker, log, rateLimiter)
	if err != nil {
		log.Error("Failed to create auth client", "component", "server", "error", err)
		os.Exit(1)
	}
	log.Info("Auth client initialized", "component", "server")

	messagesClient, err := handler.NewMessageServiceClient(connectionManager, log, rateLimiter)
	if err != nil {
		log.Error("Failed to create messages client", "component", "server", "error", err)
		os.Exit(1)
	}
	log.Info("Messages client initialized", "component", "server")

	broker.StartWithMonitoring(ctx)
	log.Info("Broker ready to process messages with monitoring", "component", "server")

	// Subscribe new authentication handlers
	broker.Subscribe("register_request", authClient.HandleRegistration)
	broker.Subscribe("login_challenge_request", authClient.HandleLoginChallenge)
	broker.Subscribe("auth_request", authClient.HandleAuth)

	// Subscribe new messages handlers
	broker.Subscribe("client_registration", messagesClient.HandleClientRegistration)
	broker.Subscribe("client_unregistration", messagesClient.HandleClientUnregistration)
	broker.Subscribe("message_request", messagesClient.HandleRouteMessage)

	// Subscribe existing room handlers
	broker.Subscribe("find-room", roomManager.HandleLookRoom)
	broker.Subscribe("update-spell-request", roomManager.HandleUpdateSpell)
  broker.Subscribe("quit-room", roomManager.HandleQuitRoom)
	//broker.Subscribe("create-room", roomManager.CreateRoom)
	//broker.Subscribe("join-room", roomManager.JoinRoom)

	go HandleClient(ctx, server, connChannel)
	for conn := range connChannel {
		go ProcessClient(conn, log, broker, connectionManager)
	}
	select {}
}
