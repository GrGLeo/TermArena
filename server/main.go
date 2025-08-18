package main

import (
	"context"
	"fmt"
	"net"
	"os"

	auth "github.com/GrGLeo/ctf/server/authentification"
	"github.com/GrGLeo/ctf/server/event"
	manager "github.com/GrGLeo/ctf/server/room_manager"

	"github.com/GrGLeo/ctf/shared"
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
)

func init() {
	godotenv.Load()
	env = os.Getenv("ENV")
	port = os.Getenv("SERVER")
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
			log.Errorln("Failed to accept connection")
			continue
		}
		log.Infow("Accept new connection", "ip", conn.RemoteAddr())
		connChannel <- conn
	}
}

func ProcessClient(conn *net.TCPConn, log *zap.SugaredLogger, broker *event.EventBroker) {
	buffer := make([]byte, 4096) // A persistent buffer for each client
	var data []byte

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				log.Infow("Client disconnected", "ip", conn.RemoteAddr())
			} else {
				log.Infow("Error reading from client", "ip", conn.RemoteAddr(), "error", err)
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
					log.Infow("Error deserializing packet", "ip", conn.RemoteAddr(), "error", err)
					data = nil // Clear buffer on persistent error
					continue
				}

				msg, err := shared.CreateMessage(packet, conn)
				if err != nil {
					log.Infow("Error creating message from packet", "ip", conn.RemoteAddr(), "error", err)
					data = data[bytesConsumed:]
					continue
				}

				// Unified logic: publish message, wait for response, create packet, send response.
				broker.Publish(msg)
				response := <-broker.ResponseChannel(msg.Type())

				responsePacket, err := shared.CreatePacketFromMessage(response)
				if err != nil {
					log.Errorw("Error creating packet from message", "error", err.Error())
					data = data[bytesConsumed:]
					continue
				}

				if _, err := conn.Write(responsePacket); err != nil {
					log.Errorw("Error writing response to client", "error", err)
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

func main() {
	log := NewLogger(env)
	log.Info("Starting server...")
	serverAddr, err := net.ResolveTCPAddr("tcp", ":8082")
	if err != nil {
		log.Fatalln("Failed to resolve TCP Addr", err.Error())
	}
	server, err := net.ListenTCP("tcp", serverAddr)
	if err != nil {
		log.Fatalln("Failed to launch TCP server", err.Error())
	}
	log.Info("Server started and listening")
	connChannel := make(chan *net.TCPConn)
	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, log)

	broker := event.NewEventBroker(log)
	log.Info("New Event Broker initialize")

	roomManager := manager.NewRoomManager(log)
	log.Info("New room manager initialize")

	authClient, err := auth.NewAuthClient()
	if err != nil {
		log.Fatalln("Failed to create auth client:", err)
	}
	log.Info("Auth client initialized")

	go broker.ProcessMessage()
	log.Info("Broker ready to process message")

	// Subscribe new authentication handlers
	broker.Subscribe("register_request", authClient.HandleRegistration)
	broker.Subscribe("login_challenge_request", authClient.HandleLoginChallenge)
	broker.Subscribe("auth_request", authClient.HandleAuth)

	// Subscribe existing room handlers
	broker.Subscribe("find-room", roomManager.FindRoom)
	broker.Subscribe("create-room", roomManager.CreateRoom)
	broker.Subscribe("join-room", roomManager.JoinRoom)

	go HandleClient(ctx, server, connChannel)
	for conn := range connChannel {
		go ProcessClient(conn, log, broker)
	}
	select {}
}

