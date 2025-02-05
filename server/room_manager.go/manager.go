package manager

import (
	"sync"

	"github.com/GrGLeo/ctf/server/event"
	"github.com/GrGLeo/ctf/server/game"
	"go.uber.org/zap"
)

const (
  SOLO = iota
  DUO
  QUAD
)

type RoomManager struct {
	RoomQueues  map[int][]*game.GameRoom
	RoomStarted []*game.GameRoom
	logger      *zap.SugaredLogger
	mu          sync.RWMutex
}

func NewRoomManager(logger *zap.SugaredLogger) *RoomManager {
	return &RoomManager{
		RoomQueues: make(map[int][]*game.GameRoom),
    logger: logger,
	}
}

func (rm *RoomManager) FindRoom(msg event.Message) event.Message {
	if msg.Type() != "find-room" {
		rm.logger.Error("Invalid message type for FindRoom")
		return nil
	}

	roomRequest, ok := msg.(event.RoomRequestMessage)
	if !ok {
		rm.logger.Error("Failed to cast message to RoomRequestMessage")
		return nil
	}

	if err := roomRequest.Validate(); err != nil {
		rm.logger.Errorw("Invalid RoomRequestMessage", "error", err)
		return nil
	}

	var maxPlayer int
	roomType := roomRequest.RoomType
	switch roomType {
	case SOLO:
		maxPlayer = 1
	case DUO:
		maxPlayer = 2
	case QUAD:
		maxPlayer = 4
	}
  conn := roomRequest.Conn

	rm.logger.Infow("Finding room", "roomType", roomType)
  rm.logger.Infof("RoomStates: %+v\n", rm.RoomQueues)

  // We check for game type
  if  maxPlayer ==  1 {
    // We start the game instantly as the player is solo
		rm.logger.Infoln("Initializing a new room queue and creating a room")
		newRoom := game.NewGameRoom(maxPlayer, rm.logger)
		newRoom.AddPlayer(conn)

    rm.mu.Lock()
    rm.RoomStarted = append(rm.RoomStarted, newRoom)
    rm.mu.Unlock()

    go newRoom.StartGame()
  } else {
		// Use RLock to check if a room is available
		rm.mu.RLock()
		room, exists := rm.RoomQueues[roomType]
		rm.mu.RUnlock()

		if exists && len(room) > 0 {
			rm.logger.Infoln("Adding player to an existing room")

			rm.mu.Lock()
			oldestRoom := rm.RoomQueues[roomType][0]
			oldestRoom.AddPlayer(conn)

			if oldestRoom.RoomSize == oldestRoom.PlayersIn() {
				rm.RoomStarted = append(rm.RoomStarted, oldestRoom)
				rm.RoomQueues[roomType] = rm.RoomQueues[roomType][1:] // Remove the room that is starting
				go oldestRoom.StartGame()
			}
			rm.mu.Unlock()
		} else {
			rm.logger.Infoln("Creating new room")

			newRoom := game.NewGameRoom(maxPlayer, rm.logger)
			newRoom.AddPlayer(conn)

			rm.mu.Lock()
			rm.RoomQueues[roomType] = append(rm.RoomQueues[roomType], newRoom)
			rm.mu.Unlock()
		}
	}

	response := event.RoomSearchMessage{
		Success: 0,
	}
	return response
} 
