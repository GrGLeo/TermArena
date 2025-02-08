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

// RoomManager handles the queueing and starting of game rooms.
type RoomManager struct {
	// RoomQueues holds rooms waiting for players to join, keyed by room type.
	RoomQueues map[int]map[string]*game.GameRoom
	// RoomStarted holds rooms that have started.
	RoomStarted []*game.GameRoom
	logger      *zap.SugaredLogger
	mu          sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
func NewRoomManager(logger *zap.SugaredLogger) *RoomManager {
	return &RoomManager{
		RoomQueues: make(map[int]map[string]*game.GameRoom),
		logger:     logger,
	}
}

// getMaxPlayers returns the maximum players for a given room type.
func getMaxPlayers(roomType int) int {
	switch roomType {
	case SOLO:
		return 1
	case DUO:
		return 2
	case QUAD:
		return 4
	default:
		return 0
	}
}

// FindRoom processes a room search request and assigns the connecting player to a room.
func (rm *RoomManager) FindRoom(msg event.Message) event.Message {
	// Validate message type.
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

	roomType := roomRequest.RoomType
	maxPlayers := getMaxPlayers(roomType)
	conn := roomRequest.Conn

	rm.logger.Infow("Finding room", "roomType", roomType)
	rm.logger.Infof("RoomQueues: %+v", rm.RoomQueues)

	// Solo rooms can be started immediately.
	if maxPlayers == 1 {
		newRoom := game.NewGameRoom(maxPlayers, rm.logger)
		newRoom.AddPlayer(conn)
		rm.mu.Lock()
		rm.RoomStarted = append(rm.RoomStarted, newRoom)
		rm.mu.Unlock()
		go newRoom.StartGame()
	} else {
		// For DUO and QUAD, lock for the entire process to avoid race conditions.
		rm.mu.Lock()
		queue := rm.RoomQueues[roomType]
		if len(queue) > 0 {
			// Add player to the existing room.
      for roomID, room :=  range queue {
        room.AddPlayer(conn)
        // Check if the room is full.
        if room.RoomSize == room.PlayersIn() {
          rm.RoomStarted = append(rm.RoomStarted, room)
          // Remove the room from the queue.
          delete (queue, roomID)
          go room.StartGame()
          break
        }
      }
		} else {
			// No waiting room exists; create a new one.
			newRoom := game.NewGameRoom(maxPlayers, rm.logger)
			newRoom.AddPlayer(conn)
      if rm.RoomQueues[roomType] == nil {
        rm.RoomQueues[roomType] = make(map[string]*game.GameRoom)
      }
			rm.RoomQueues[roomType][newRoom.GameID] = newRoom 
		}
		rm.mu.Unlock()
	}

	return event.RoomSearchMessage{
		Success: 0,
	}
}

// FindRoom processes a room search request and assigns the connecting player to a room.
func (rm *RoomManager) JoinRoom(msg event.Message) event.Message {
  // Validate message type.
  if msg.Type() != "join-room" {
    rm.logger.Error("Invalid message type for FindRoom")
    return nil
  }

  roomJoin, ok := msg.(event.RoomJoinMessage)
  if !ok {
    rm.logger.Error("Failed to cast message to RoomJoinMessage")
    return nil
  }

  if err := roomJoin.Validate(); err != nil {
    rm.logger.Errorw("Invalid RoomJoinMessage", "error", err)
    return nil
  }

  roomID := roomJoin.RoomID
  conn := roomJoin.Conn

  for _, roomMap := range rm.RoomQueues {
    if room, ok := roomMap[roomID]; ok {
      room.AddPlayer(conn)
      // Check if the room is full.
      if room.RoomSize == room.PlayersIn() {
        rm.RoomStarted = append(rm.RoomStarted, room)
        // Remove the room from the queue.
        delete(roomMap, roomID)
        go room.StartGame()
      }
    }
  }
  return event.RoomSearchMessage{
    Success: 0,
  }
}


func (rm *RoomManager) CreateRoom(msg event.Message) event.Message {
  // Validate message type.
  if msg.Type() != "create-room" {
    rm.logger.Error("Invalid message type for CreateRoom")
    return nil
  }

  roomCreate, ok := msg.(event.RoomCreateMessage)
  if !ok {
    rm.logger.Error("Failed to cast message to RoomCreateMessage")
    return nil
  }

  if err := roomCreate.Validate(); err != nil {
    rm.logger.Errorw("Invalid RoomCreateMessage", "error", err)
    return nil
  }

  roomType := roomCreate.RoomType
  maxPlayers := getMaxPlayers(roomType)
  newRoom := game.NewGameRoom(maxPlayers, rm.logger)
  newRoom.AddPlayer(roomCreate.Conn)

  rm.mu.Lock()
  defer rm.mu.Unlock()
  if rm.RoomQueues[roomType] == nil {
    rm.RoomQueues[roomType] = make(map[string]*game.GameRoom)
  }
  rm.RoomQueues[roomType][newRoom.GameID] = newRoom 
  return event.RoomSearchMessage{
    Success: 0,
  }
}
