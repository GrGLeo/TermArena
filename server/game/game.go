package game

import (
	"errors"
	"math/rand"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GrGLeo/ctf/shared"
	"go.uber.org/zap"
)

type GameRoom struct {
	GameID           string
	RoomSize         int
	board            *Board
	tickID           atomic.Int32
	actions          []actionType
	points           [2]int
	playerConnection []*net.TCPConn
	playerChar       map[string]*Player
	logger           *zap.SugaredLogger
	actionChan       chan *ActionMsg
	gameMutex        sync.Mutex
}

func NewGameRoom(number int, logger *zap.SugaredLogger) *GameRoom {
	gr := GameRoom{
		GameID:     GenerateGameID(),
		RoomSize:   number,
		logger:     logger,
		actionChan: make(chan *ActionMsg),
		playerChar: make(map[string]*Player),
	}
	// Place walls on map
	// if any error occur we skip the walls placement
	walls, flags, players, err := LoadConfig("server/game/config.json")
	logger.Infow("Opening new game room", "roomID", gr.GameID, "type", number)
	if err != nil {
		gr.logger.Warnw("Error while reading the config", "roomID", gr.GameID, "error", err.Error())
	} else {
		board := InitBoard(walls, flags, players)
		gr.board = board
	}
	return &gr
}

func (gr *GameRoom) PlayersIn() int {
	return len(gr.playerConnection)
}

func (gr *GameRoom) AddPlayer(conn *net.TCPConn) {
	gr.gameMutex.Lock()
	defer gr.gameMutex.Unlock()
	playerNumber := len(gr.playerConnection)
	player := gr.board.Players[playerNumber]
	gr.playerChar[conn.RemoteAddr().String()] = player
	gr.playerConnection = append(gr.playerConnection, conn)
	// Send the initial grid to the player
	go gr.ListenToConnection(conn)
	gr.logger.Infow("Player joined", "id", conn.RemoteAddr())
}

func (gr *GameRoom) StartGame() {
	if len(gr.playerConnection) == gr.RoomSize {
		// Game init
		gr.logger.Infow("Game starting", "roomID", gr.GameID)
		gr.SendGameStart()
		gr.sendInitGrid()
		time.Sleep(1 * time.Second)
		// Game start
		go gr.HandleAction()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Game is won
				if gr.points[0] == 3 || gr.points[1] == 3 {
					gr.CloseGame(1)
					return
				}
				// Else we keep the game loop
				gr.tickID.Add(1)
				gr.board.ReplaceHiddenFlag()
				gr.board.UpdateSprite()
				for _, player := range gr.playerChar {
					// process each player action
					flagWon := player.TakeAction(gr.board)
					if flagWon {
						switch player.TeamID {
						case 6:
							gr.points[0]++
						case 7:
							gr.points[1]++
						}
					}
				}
				gr.board.Update()
				if err := gr.broadcastState(); err != nil {
					gr.logger.Infoln(err.Error())
					return
				}
			}
		}
	}
}

func (gr *GameRoom) CloseGame(success int) {
	gr.logger.Infow("Closing game", "roomID", gr.GameID)
	closeGamePacket := shared.NewGameClosePacket(success)
	data := closeGamePacket.Serialize()
	for _, conn := range gr.playerConnection {
		if _, err := conn.Write(data); err != nil {
			gr.logger.Infow("Failed to send close game message")
		}
		// we close the connection for now.
		// Client who receive message will recontact the server
		// this is not clean
		conn.Close()
	}
}

func (gr *GameRoom) sendInitGrid() {
	grid := gr.board.GetCurrentGrid()
	encodedBoard := RunLengthEncode(grid)
  length := len(encodedBoard)
	packet := shared.NewBoardPacket(0, 0, gr.points, length, encodedBoard)
	data := packet.Serialize()
	for _, conn := range gr.playerConnection {
		_, err := conn.Write(data)
		if err != nil {
			gr.logger.Warnw("Failed to send initial board to player", "roomID", gr.GameID, "id", conn.RemoteAddr(), "error", err)
			return
		}
	}
}

func (gr *GameRoom) broadcastState() error {
	var data []byte
	grid := gr.board.GetCurrentGrid()
	deltas := gr.board.Tracker.GetDeltasByte()
	defer gr.board.Tracker.Reset()
	// Check if full board needs to be resend
	totalCells := len(grid) * len(grid[0])
	// If more than 50% of the board has change we resend the board
	if len(deltas) > totalCells/2 {
		gr.logger.Infow("Sending back full board", "roomID", gr.GameID)
		encodedBoard := RunLengthEncode(grid)
    length := len(encodedBoard)
		packet := shared.NewBoardPacket(0, 0, gr.points, length, encodedBoard)
		data = packet.Serialize()
	} else {
		tickID := uint32(gr.tickID.Load())
		packet := shared.NewDeltaPacket(tickID, gr.points, deltas)
		data = packet.Serialize()
	}
	for _, conn := range gr.playerConnection {
		_, err := conn.Write(data)
		if err != nil {
			err := errors.New("Player disconnect. Closing game")
			// We send closing on error to clients
			gr.CloseGame(2)
			return err
		}
	}
	return nil
}

func (gr *GameRoom) SendGameStart() {
	packet := shared.NewGameStartPacket(0)
	data := packet.Serialize()
	for _, conn := range gr.playerConnection {
		_, err := conn.Write(data)
		if err != nil {
			gr.logger.Warn("Player disconnect. Closing game")
			// For now we stop the game
			os.Exit(1)
		}
		gr.logger.Infow("Send start game", "id", conn.RemoteAddr().String())
	}
}

func (gr *GameRoom) HandleAction() {
	for {
		select {
		case actionMsg, ok := <-gr.actionChan:
			if !ok {
				gr.logger.Info("Action channel closed, stopping action handling")
				return // Exit the loop if the channel is closed
			}
			gr.gameMutex.Lock()
			player, exists := gr.playerChar[actionMsg.ConnAddr]
			if exists {
				player.Action = actionType(actionMsg.Action)
				//gr.logger.Infow("Processed action", "roomID", gr.GameID, "conn", actionMsg.ConnAddr, "action", player.Action)
			} else {
				gr.logger.Warnw("No player found for connection", "roomID", gr.GameID, "conn", actionMsg.ConnAddr)
			}
			gr.gameMutex.Unlock()
		}
	}
}

func (gr *GameRoom) ListenToConnection(conn *net.TCPConn) {
	gr.logger.Infow("Started listening to connection", "roomID", gr.GameID, "id", conn.RemoteAddr())

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				gr.logger.Infow("Client disconnected", "ip", conn.RemoteAddr())
				// Client disconnected cleanly
			} else {
				gr.logger.Warnf("Error reading from connection %s: %v", conn.RemoteAddr(), err)
			}
			return
		}
		if n > 0 {
			message, err := shared.DeSerialize(buffer[:n])
			if err != nil {
				gr.logger.Infow("Error deserializing packet", "ip", conn.RemoteAddr(), "error", err)
			}
			switch msg := message.(type) {
			case *shared.ActionPacket:
				playerAction := &ActionMsg{
					ConnAddr: conn.RemoteAddr().String(),
					Action:   msg.Action(),
				}
				gr.actionChan <- playerAction
			}
		}
	}
}

func GenerateGameID() string {
	var gameId string
	char := "QWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfgjkzxcvbnm1234567890"
	for i := 0; i < 5; i++ {
		id := rand.Intn(len(char) - 1)
		gameId += string(char[id])
	}
	return gameId
}
