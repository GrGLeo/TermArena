package game

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/GrGLeo/ctf/shared"
	"go.uber.org/zap"
)


type GameRoom struct {
  PlayerNumber int
  board *Board
  actions []action
  playerConnection []*net.TCPConn
  logger *zap.SugaredLogger
}

func NewGameRoom(number int, logger *zap.SugaredLogger) *GameRoom {
  gr := GameRoom{
    PlayerNumber: number,
    board: Init(),
    logger: logger,
  }
  for n := range number {
    gr.board.PlacePlayer(n)
  }
  return &gr
}

func (gr *GameRoom) AddPlayer(conn *net.TCPConn) {
  gr.logger.Infow("Payer added", "id", conn.RemoteAddr())
  fmt.Println("Hello2")
  gr.playerConnection = append(gr.playerConnection, conn)
  fmt.Println("Hello3")
}

func (gr *GameRoom) StartGame() {
  if len(gr.playerConnection) == gr.PlayerNumber {
    gr.logger.Info("Game starting")
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
      select {
      case <- ticker.C:
        for _, action := range gr.actions {
          // process each player action
          gr.logger.Infof("Player action: %s", action)
        }
        encodedBoard := gr.board.RunLengthEncode()
        for _, conn := range gr.playerConnection {
          packet := shared.NewPacket(1, 3, encodedBoard)
          data, err := packet.Serialize()
          if err != nil {
            gr.logger.Warn("Error serializing board")
          }
          _, err = conn.Write(data)
          if err != nil {
            gr.logger.Warn("Player disconnect. Closing game")
            // For now we stop the game
            os.Exit(1)
          }
          gr.logger.Infow("Sent board", "board", encodedBoard)
        }
      }
    }
  }
}
      
