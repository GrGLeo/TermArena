package manager

import (
	"log/slog"
	"strconv"
	"sync"

	"github.com/GrGLeo/ctf/server/event"
	ratelimiter "github.com/GrGLeo/ctf/server/rate_limiter"
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

	// Rate limit checks
	allowed, err := rm.rateLimiter.Allow(roomRequest.Username, roomRequest.Type(), false)
	if err != nil {
		rm.logger.Error("Failed to retrieve bucket", "component", "room_manager", "error", err, "user", roomRequest.Username)
		return event.RoomSearchMessage{
			Success: 1,
			RoomIP:  "",
		}
	}
	if !allowed {
		rm.logger.Warn("Rate limit exceed", "component", "room_manager", "username", roomRequest.Username)
		return event.RateLimitResponse{ResponseCh: roomRequest.ResponseCh}
	}

	if err := roomRequest.Validate(); err != nil {
		rm.logger.Error("Invalid RoomRequestMessage", "component", "room_manager", "error", err)
		return nil
	}

	roomType := roomRequest.RoomType
	maxPlayers := getMaxPlayers(roomType)

	rm.logger.Info("Finding room", "component", "room_manager", "user", roomRequest.Username, "roomType", roomType)

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

	// We first search for rooms with open slot
	switch roomType {
	case PRACTICE:
		rm.mu.Lock()
		defer rm.mu.Unlock()
		result := findRoom(rm.PracticeRooms, roomRequest, "pratice", rm.broker, rm.logger)
		if result.Found {
			return result.Message
		}
	case CLASSIC:
		rm.mu.Lock()
		defer rm.mu.Unlock()
		result := findRoom(rm.ClassicRooms, roomRequest, "classic", rm.broker, rm.logger)
		if result.Found {
			return result.Message
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
	roomID, err := StartGame(portStr, "1", maxPlayersStr)
	if err != nil {
		rm.logger.Error("Failed to create room", "component", "room_manager", "error", err)
	}

	switch roomType {
	case PRACTICE:
		newRoom := &PracticeRoom{
			RoomID:     roomID,
			Port:       portStr,
			PlayersIn:  1,
			MaxPlayers: maxPlayers,
		}
		rm.PracticeRooms[roomID] = newRoom
		rm.logger.Info("Created new practice room", "component", "room_manager", "port", portStr)
	case CLASSIC:
		newRoom := &ClassicRoom{
			RoomID:     roomID,
			Port:       portStr,
			PlayersIn:  1,
			MaxPlayers: maxPlayers,
		}
		rm.ClassicRooms[roomID] = newRoom
		rm.logger.Info("Created new classic room", "component", "room_manager", "port", portStr)
	}

	// We send the roomID to the message service for the user to be switch
	regResponseCh := make(chan event.Message, 1)
	clientRegistration := event.ClientRegistrationMessage{
		ClientID:   roomRequest.Username,
		RoomID:     roomID,
		Conn:       roomRequest.Conn,
		ResponseCh: regResponseCh,
	}
	rm.broker.Publish(clientRegistration)
	// Wait for client registration to complete
	regResponse := <-regResponseCh
	if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && regResp.Success {
		rm.logger.Info("New room register", "component", "room_manager", "port", portStr)
		return event.RoomSearchMessage{
			Success: 0,
			RoomID:  roomID,
			RoomIP:  portStr,
		}
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
