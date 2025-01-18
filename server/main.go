package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/GrGLeo/ctf/server/game"
	"github.com/GrGLeo/ctf/shared"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type CtxKey string 
const (
  loggerKey CtxKey = "logger"
)


var (
  env string
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

func ProcessClient(conn *net.TCPConn, log *zap.SugaredLogger) {
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
      log.Infow("Received data", "ip", conn.RemoteAddr(), "data", buffer[:n])
      message, err := shared.DeSerialize(buffer[:n])
      if err != nil {
        log.Infow("Error deserializing packet", "ip", conn.RemoteAddr(), "error", err)
      }
      switch msg := message.(type) {
      case *shared.LoginPacket:
        log.Infow("Received login", "username", msg.Username)
        // send ok message 
        packet := shared.NewPacket(1, 1, []byte{0})
        data, _ := packet.Serialize()
        n, err := conn.Write(data)
        if err != nil {
          log.Errorw("Error writting login resp", n, "ip", conn.RemoteAddr())
        }
        log.Infow("Login response", "byte", n, "ip", conn.RemoteAddr())
        // create a new game 
        game := game.NewGameRoom(1, log)
        game.AddPlayer(conn)
        go game.StartGame()

      }
    }
  }
}


func main() {
  log := NewLogger(env)
  log.Info("Starting server...")
  serverAddr, err := net.ResolveTCPAddr("tcp", port)
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
  go HandleClient(ctx, server, connChannel)
  for conn := range connChannel {
    go ProcessClient(conn, log)
  }
  select{}
}
