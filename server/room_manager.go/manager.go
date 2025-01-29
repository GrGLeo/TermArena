package manager

import (
	"sync"

	"github.com/GrGLeo/ctf/server/event"
	"github.com/GrGLeo/ctf/server/game"
	"go.uber.org/zap"
)

type RoomManager struct {
	RoomQueues  map[int][]*game.GameRoom
	RoomStarted []*game.GameRoom
	logger      *zap.SugaredLogger
	mu          sync.Mutex
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

	rm.mu.Lock()
	defer rm.mu.Unlock()
	var maxPlayer int
	roomType := roomRequest.RoomType
	switch roomType {
	case 0:
		maxPlayer = 1
	case 1:
		maxPlayer = 2
	case 2:
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
    rm.RoomStarted = append(rm.RoomStarted, newRoom)
    go newRoom.StartGame()
  } else if room, ok := rm.RoomQueues[roomType]; ok {
    // We check if there is already a room in a waiting statew
		if len(room) > 0 {
			rm.logger.Infoln("Adding player to an existing room")
			oldestRoom := room[0]
			oldestRoom.AddPlayer(conn)
			if oldestRoom.PlayerNumber == oldestRoom.PlayersIn() {
				rm.RoomStarted = append(rm.RoomStarted, oldestRoom)
				// we remove the room that is starting
				rm.RoomQueues[roomType] = room[1:]
				go oldestRoom.StartGame()
			}
		} else {
			// In case all room are started we create a new one
			rm.logger.Infoln("Creating new room")
			newRoom := game.NewGameRoom(maxPlayer, rm.logger)
			newRoom.AddPlayer(conn)
		rm.RoomQueues[roomType] = append(rm.RoomQueues[roomType], newRoom)
		}
	} else {
		// If the server just started the map is not yet initialize
		rm.logger.Infoln("Initializing a new room queue and creating a room")
		newRoom := game.NewGameRoom(maxPlayer, rm.logger)
		newRoom.AddPlayer(conn)
		rm.RoomQueues[roomType] = append(rm.RoomQueues[roomType], newRoom)
	}
	response := event.RoomSearchMessage{
		Success: 0,
	}
	return response
}
