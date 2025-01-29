package main

import (
	"context"
	"fmt"
	"net"
	"os"

	auth "github.com/GrGLeo/ctf/server/authentification"
	"github.com/GrGLeo/ctf/server/event"
	manager "github.com/GrGLeo/ctf/server/room_manager.go"

	//"github.com/GrGLeo/ctf/server/game"
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
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				// Client disconnected cleanly
				log.Infow("Client disconnected", "ip", conn.RemoteAddr())
			} else {
				// Other errors
				log.Infow("Error reading from client", "ip", conn.RemoteAddr(), "error", err)
			}
			return // Exit if there's an error or if the client disconnects
		}
		if n > 0 {
			message, err := shared.DeSerialize(buffer[:n])
			if err != nil {
				log.Infow("Error deserializing packet", "ip", conn.RemoteAddr(), "error", err)
			}
			msg, err := shared.CreateMessage(message, conn)
			if err != nil {
				log.Infow("Error creating message from packet", "ip", conn.RemoteAddr(), "error", err)
			}
			broker.Publish(msg)
			response := <-broker.ResponseChannel(msg.Type())
			data, err := shared.CreatePacketFromMessage(response)
      // We need to check message and act accordingly
      switch response.(type) {
      case event.AuthMessage:
        n, err := conn.Write(data)
        if err != nil {
          log.Errorw("Error writting login resp", n, "ip", conn.RemoteAddr())
        }
        log.Infow("Login response", "byte", n, "ip", conn.RemoteAddr())
      case event.RoomSearchMessage:
        n, err := conn.Write(data)
        if err != nil {
          log.Errorw("Error writting login resp", n, "ip", conn.RemoteAddr())
        }
        log.Infow("RoomSearch response", "byte", n, "ip", conn.RemoteAddr())
        return // GameRoom take ownership of the conn
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
	// Initialize new EventBroker
	broker := event.NewEventBroker(log)
	log.Info("New Event Broker initialize")
	roomManager := manager.NewRoomManager(log)
	log.Info("New room manager initialize")
	go broker.ProcessMessage()
	log.Info("Broker ready to process message")
	broker.Subscribe("login", auth.Authentificate)
	broker.Subscribe("find-room", roomManager.FindRoom)
	go HandleClient(ctx, server, connChannel)
	for conn := range connChannel {
		go ProcessClient(conn, log, broker)
	}
	select {}
}
