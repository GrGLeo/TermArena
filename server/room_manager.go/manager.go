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
	RoomQueues map[int][]*game.GameRoom
	// RoomStarted holds rooms that have started.
	RoomStarted []*game.GameRoom
	logger      *zap.SugaredLogger
	mu          sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
func NewRoomManager(logger *zap.SugaredLogger) *RoomManager {
	return &RoomManager{
		RoomQueues: make(map[int][]*game.GameRoom),
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
			room := queue[0]
			room.AddPlayer(conn)
			// Check if the room is full.
			if room.RoomSize == room.PlayersIn() {
				rm.RoomStarted = append(rm.RoomStarted, room)
				// Remove the room from the queue.
				rm.RoomQueues[roomType] = queue[1:]
				go room.StartGame()
			}
		} else {
			// No waiting room exists; create a new one.
			newRoom := game.NewGameRoom(maxPlayers, rm.logger)
			newRoom.AddPlayer(conn)
			rm.RoomQueues[roomType] = append(queue, newRoom)
		}
		rm.mu.Unlock()
	}

	return event.RoomSearchMessage{
		Success: 0,
	}
}
