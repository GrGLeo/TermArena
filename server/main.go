package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

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
    if err != nil {
      fmt.Println("Failed to build logger")
      os.Exit(1)
    }
  } else {
    log, err = zap.NewProduction()
    if err != nil {
      fmt.Println("Failed to build logger")
      os.Exit(1)
    }
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
  ticker := time.NewTicker(1 * time.Second)
  for {
    select {
    case <- ticker.C:
      log.Infow("Writing to client", "ip", conn.RemoteAddr())
      if _, err := conn.Write([]byte{65, 66, 67}); err != nil {
        log.Infow("Client disconnect", "ip", conn.RemoteAddr())
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
