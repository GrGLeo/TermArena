package main

import (
	"context"
	"fmt"
	"net"
	"os"

	conm "github.com/GrGLeo/ctf/server/conn_manager"
	"github.com/GrGLeo/ctf/server/event"
	handler "github.com/GrGLeo/ctf/server/handlers"
	ratelimiter "github.com/GrGLeo/ctf/server/rate_limiter"
	manager "github.com/GrGLeo/ctf/server/room_manager"

	"github.com/GrGLeo/ctf/pkg/shared"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type CtxKey string

const (
	loggerKey CtxKey = "logger"
)

var (
	env  string
	port string
  rateLimiterConfigPath string
)

func init() {
	godotenv.Load()
	env = os.Getenv("ENV")
	port = os.Getenv("SERVER")
  rateLimiterConfigPath = os.Getenv("RATE_LIMITER_CONFIG_PATH")
}

func NewLogger(env string) *zap.SugaredLogger {
	var (
		log *zap.Logger
		err error
	)
	if env == "DEV" {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Println("Failed to build logger")
	}
	logger := log.Sugar()
	return logger
}

func HandleClient(ctx context.Context, server *net.TCPListener, connChannel chan *net.TCPConn) {
	log, _ := ctx.Value(loggerKey).(*zap.SugaredLogger)
	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			log.Errorln("[SERVER] Failed to accept connection")
			continue
		}
		log.Infow("[SERVER] Accept new connection", "ip", conn.RemoteAddr())
		connChannel <- conn
	}
}

func ProcessClient(conn *net.TCPConn, log *zap.SugaredLogger, broker *event.EventBroker, connManager *conm.ConnectionManager) {
	buffer := make([]byte, 4096) // A persistent buffer for each client
	var data []byte

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				log.Infow("[SERVER] Client disconnected", "ip", conn.RemoteAddr())
				// Unregister the client
				client, exist := connManager.Unregister(conn)
				if exist {
					responseChan := make(chan event.Message)
					msg := event.ClientUnregistrationMessage{
						ClientID:  client,
						ReponseCh: responseChan,
					}
					broker.Publish(msg)
				}
			} else {
				log.Infow("[SERVER] Error reading from client", "ip", conn.RemoteAddr(), "error", err)
			}
			return // Exit if there's an error or if the client disconnects
		}

		if n > 0 {
			data = append(data, buffer[:n]...)
			for len(data) > 0 {
				packet, bytesConsumed, err := shared.DeSerialize(data)
				if err != nil {
					if err.Error() == "incomplete packet" {
						break // Wait for more data
					}
					log.Infow("[SERVER] Error deserializing packet", "ip", conn.RemoteAddr(), "error", err)
					data = nil // Clear buffer on persistent error
					continue
				}

				msg, err := event.CreateMessage(packet, conn, connManager)
				if err != nil {
					log.Infow("[SERVER] Error creating message from packet", "ip", conn.RemoteAddr(), "error", err)
					data = data[bytesConsumed:]
					continue
				}

				// Log message processing
				if msg.Type() == "message_request" {
					if reqMsg, ok := msg.(event.MessageRequestMessage); ok {
						log.Infow("[SERVER] MessageRequest received", "sender", reqMsg.Sender, "message", reqMsg.Message, "ip", conn.RemoteAddr())
					}
				}

				// Unified logic: publish message, wait for response, create packet, send response.
				log.Infow("[SERVER] Publishing message to broker", "message_type", msg.Type(), "ip", conn.RemoteAddr())
				broker.Publish(msg)
				log.Infow("[SERVER] Waiting for response from broker", "message_type", msg.Type())
				response := <-msg.ResponseChan()
				log.Infow("[SERVER] Received response from broker", "response_type", response.Type(), "ip", conn.RemoteAddr())

				switch resp := response.(type) {
				case event.MessageResponseMessage:
					log.Infow("[SERVER] MessageResponseMessage found", "message", resp.Message)
					responsePacket, err := event.CreatePacketFromMessage(resp)
					if err != nil {
						log.Errorw("[SERVER] Error creating packet from message", "error", err.Error())
						data = data[bytesConsumed:]
						continue
					}
					for _, receiverID := range resp.Receivers {
						receiverConn, exist := connManager.GetConn(receiverID)
						if exist {
							if _, err := receiverConn.Write(responsePacket); err != nil {
								log.Errorw("[SERVER] Error writing response to client", "error", err)
							}
						} else {
							log.Warnw("[SERVER] Could not find connection for receiver", "receiver", receiverID)
						}
					}
					data = data[bytesConsumed:]
				default:
					responsePacket, err := event.CreatePacketFromMessage(resp)
					if err != nil {
						log.Errorw("[SERVER] Error creating packet from message", "error", err.Error())
						data = data[bytesConsumed:]
						continue
					}
					if _, err := conn.Write(responsePacket); err != nil {
						log.Errorw("[SERVER] Error writing response to client", "error", err)
					}
					// Special case for room search where connection ownership changes
					if _, ok := response.(event.RoomSearchMessage); ok {
						return
					}
					data = data[bytesConsumed:]
				}
			}
		}
	}
}

func main() {
	log := NewLogger(env)
	log.Info("[SERVER] Starting server...")
	serverAddr, err := net.ResolveTCPAddr("tcp", ":8082")
	if err != nil {
		log.Fatalln("[SERVER] Failed to resolve TCP Addr", err.Error())
	}
	server, err := net.ListenTCP("tcp", serverAddr)
	if err != nil {
		log.Fatalln("[SERVER] Failed to launch TCP server", err.Error())
	}
	log.Info("[SERVER] Server started and listening")
	connectionManager := conm.NewConnectionManager()
	connChannel := make(chan *net.TCPConn)
	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, log)

	broker := event.NewEventBroker(log, 10)
	log.Info("[SERVER] New Event Broker initialize")

  rateLimiter, err := ratelimiter.NewGlobalRateLimiter(rateLimiterConfigPath)
  if err != nil {
    log.Fatalln("[SERVER] Failed to create rate limiter", err)
  }

	roomManager := manager.NewRoomManager(log, rateLimiter)
	log.Info("[SERVER] New room manager initialize")

	authClient, err := handler.NewAuthClient(broker, log, rateLimiter)
	if err != nil {
		log.Fatalln("[SERVER] Failed to create auth client:", err)
	}
	log.Info("[SERVER] Auth client initialized")
	messagesClient, err := handler.NewMessageServiceClient(connectionManager, log, rateLimiter)
	if err != nil {
		log.Fatalln("[SERVER] Failed to create messages client:", err)
	}
	log.Info("[SERVER] Messages client initialized")

	broker.StartWithMonitoring(ctx)
	log.Info("[SERVER] Broker ready to process message with monitoring")

	// Subscribe new authentication handlers
	broker.Subscribe("register_request", authClient.HandleRegistration)
	broker.Subscribe("login_challenge_request", authClient.HandleLoginChallenge)
	broker.Subscribe("auth_request", authClient.HandleAuth)

	// Subscribe new messages handlers
	broker.Subscribe("client_registration", messagesClient.HandleClientRegistration)
	broker.Subscribe("client_unregistration", messagesClient.HandleClientUnregistration)
	broker.Subscribe("message_request", messagesClient.HandleRouteMessage)

	// Subscribe existing room handlers
	broker.Subscribe("find-room", roomManager.FindRoom)
	broker.Subscribe("create-room", roomManager.CreateRoom)
	broker.Subscribe("join-room", roomManager.JoinRoom)

	go HandleClient(ctx, server, connChannel)
	for conn := range connChannel {
		go ProcessClient(conn, log, broker, connectionManager)
	}
	select {}
}
