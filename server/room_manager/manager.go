package manager

import (
	"strconv"
	"sync"

	"github.com/GrGLeo/ctf/server/event"
	ratelimiter "github.com/GrGLeo/ctf/server/rate_limiter"
	"go.uber.org/zap"
)

var (
	portCounter = 50053
	portMutex   = &sync.Mutex{}
)

const (
	SOLO = iota
	PRACTICE
	CLASSIC
	RANKED
)

type ClassicRoom struct {
	Port       string
	PlayersIn  int
	MaxPlayers int
}

type PracticeRoom struct {
	Port       string
	PlayersIn  int
	MaxPlayers int
}

// RoomManager handles the queueing and starting of game rooms.
type RoomManager struct {
	ClassicRooms  map[string]*ClassicRoom
	PracticeRooms map[string]*PracticeRoom
	logger        *zap.SugaredLogger
	rateLimiter   *ratelimiter.GlobalRateLimiter
	mu            sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
func NewRoomManager(logger *zap.SugaredLogger, rateLimiter *ratelimiter.GlobalRateLimiter) *RoomManager {
	return &RoomManager{
		ClassicRooms:  make(map[string]*ClassicRoom),
		PracticeRooms: make(map[string]*PracticeRoom),
    rateLimiter: rateLimiter,
		logger:        logger,
	}
}

// getMaxPlayers returns the maximum players for a given room type.
func getMaxPlayers(roomType int) int {
	switch roomType {
	case SOLO:
		return 1
	case PRACTICE:
		return 4
	case CLASSIC:
		return 8
	case RANKED:
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

	allowed, err := rm.rateLimiter.Allow(roomRequest.User, roomRequest.Type(), false)

	if err != nil {
		rm.logger.Errorw("[SERVER HANDLER] Failed to retrieve bucket", "error", err, "user", roomRequest.User)
		return event.RoomSearchMessage{
			Success: 1,
			RoomIP:  "",
		}
	}

	if !allowed {
		rm.logger.Warn("[SERVER HANDLER] Rate limit exceed", "username", roomRequest.User)
		return event.RateLimitResponse{ResponseCh: roomRequest.ResponseCh}
	}

	if err := roomRequest.Validate(); err != nil {
		rm.logger.Errorw("Invalid RoomRequestMessage", "error", err)
		return nil
	}

	roomType := roomRequest.RoomType
	maxPlayers := getMaxPlayers(roomType)

	rm.logger.Infow("Finding room", "roomType", roomType)

	// Solo rooms can be started immediately.
	if roomType == SOLO {
		portMutex.Lock()
		port := portCounter
		portCounter++
		if portCounter > 50153 {
			portCounter = 50053
		}
		portMutex.Unlock()

		portStr := strconv.Itoa(port)
		StartGame(portStr, "1", "1")
		return event.RoomSearchMessage{
			Success: 0,
			RoomIP:  portStr,
		}
	}

	switch roomType {
	case PRACTICE:
		rm.mu.Lock()
		defer rm.mu.Unlock()

		// Find an existing room with space
		for port, room := range rm.PracticeRooms {
			if room.PlayersIn < room.MaxPlayers {
				room.PlayersIn++
				rm.logger.Infow("[ROOM MANAGER] Player joined existing practice room", "port", port, "players", room.PlayersIn)

				if room.PlayersIn == room.MaxPlayers {
					rm.logger.Infow("[ROOM MANAGER] Practice room is now full, removing from queue", "port", port)
					delete(rm.ClassicRooms, port)
				}

				return event.RoomSearchMessage{
					Success: 0,
					RoomIP:  port,
				}
			}
		}
	case CLASSIC:
		rm.mu.Lock()
		defer rm.mu.Unlock()

		// Find an existing room with space
		for port, room := range rm.ClassicRooms {
			if room.PlayersIn < room.MaxPlayers {
				room.PlayersIn++
				rm.logger.Infow("[ROOM MANAGER] Player joined existing classic room", "port", port, "players", room.PlayersIn)

				if room.PlayersIn == room.MaxPlayers {
					rm.logger.Infow("[ROOM MANAGER] Classic room is now full, removing from queue", "port", port)
					delete(rm.ClassicRooms, port)
				}

				return event.RoomSearchMessage{
					Success: 0,
					RoomIP:  port,
				}
			}
		}
	}
	// No available rooms, create a new one
	portMutex.Lock()
	port := portCounter
	portCounter++
	if portCounter > 50153 {
		portCounter = 50053
	}
	portMutex.Unlock()

	portStr := strconv.Itoa(port)
	maxPlayersStr := strconv.Itoa(maxPlayers)
	StartGame(portStr, "1", maxPlayersStr)

	switch roomType {
	case PRACTICE:
		newRoom := &PracticeRoom{
			Port:       portStr,
			PlayersIn:  1,
			MaxPlayers: maxPlayers,
		}
		rm.PracticeRooms[portStr] = newRoom
		rm.logger.Infow("[ROOM MANAGER] Created new pratice room", "port", portStr)
	case CLASSIC:
		newRoom := &ClassicRoom{
			Port:       portStr,
			PlayersIn:  1,
			MaxPlayers: maxPlayers,
		}
		rm.ClassicRooms[portStr] = newRoom
		rm.logger.Infow("[ROOM MANAGER] Created new classic room", "port", portStr)
	}

	return event.RoomSearchMessage{
		Success: 0,
		RoomIP:  portStr,
	}
	// Default for other game modes like RANKED for now
	return event.RoomSearchMessage{
		Success: 0,
		RoomIP:  "50052", // Placeholder
	}
}

func (rm *RoomManager) JoinRoom(msg event.Message) event.Message {
	return nil
}

func (rm *RoomManager) CreateRoom(msg event.Message) event.Message {
	return nil
}
