package manager

import (
	"log/slog"
	"sync"

	"github.com/GrGLeo/TermArena/server/event"
	ratelimiter "github.com/GrGLeo/TermArena/server/rate_limiter"
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

// RoomManager handles the queueing and starting of game rooms.
type RoomManager struct {
	ClassicRooms  map[int]Room
	PracticeRooms map[int]Room
	broker        *event.EventBroker
	logger        *slog.Logger
	rateLimiter   *ratelimiter.GlobalRateLimiter
	mu            sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
func NewRoomManager(logger *slog.Logger, broker *event.EventBroker, rateLimiter *ratelimiter.GlobalRateLimiter) *RoomManager {
	return &RoomManager{
		ClassicRooms:  make(map[int]Room),
		PracticeRooms: make(map[int]Room),
		broker:        broker,
		rateLimiter:   rateLimiter,
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
  return event.RoomCreateMessage{}
}

func (rm *RoomManager) JoinRoom(msg event.Message) event.Message {
	return nil
}

func (rm *RoomManager) CreateRoom(msg event.Message) event.Message {
	return nil
}
