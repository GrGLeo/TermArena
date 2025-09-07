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
	RoomID     int
	Port       string
	PlayersIn  int
	MaxPlayers int
}

type PracticeRoom struct {
	RoomID     int
	Port       string
	PlayersIn  int
	MaxPlayers int
}

// RoomManager handles the queueing and starting of game rooms.
type RoomManager struct {
	ClassicRooms  map[int]*ClassicRoom
	PracticeRooms map[int]*PracticeRoom
	broker        *event.EventBroker
	logger        *zap.SugaredLogger
	rateLimiter   *ratelimiter.GlobalRateLimiter
	mu            sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
func NewRoomManager(logger *zap.SugaredLogger, broker *event.EventBroker, rateLimiter *ratelimiter.GlobalRateLimiter) *RoomManager {
	return &RoomManager{
		ClassicRooms:  make(map[int]*ClassicRoom),
		PracticeRooms: make(map[int]*PracticeRoom),
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

	allowed, err := rm.rateLimiter.Allow(roomRequest.Username, roomRequest.Type(), false)

	if err != nil {
		rm.logger.Errorw("[SERVER HANDLER] Failed to retrieve bucket", "error", err, "user", roomRequest.Username)
		return event.RoomSearchMessage{
			Success: 1,
			RoomIP:  "",
		}
	}

	if !allowed {
		rm.logger.Warn("[SERVER HANDLER] Rate limit exceed", "username", roomRequest.Username)
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
		for roomID, room := range rm.PracticeRooms {
			if room.PlayersIn < room.MaxPlayers {
				room.PlayersIn++
				rm.logger.Infow("[ROOM MANAGER] Player joined existing practice room", "port", room.Port, "players", room.PlayersIn)

				if room.PlayersIn == room.MaxPlayers {
					rm.logger.Infow("[ROOM MANAGER] Practice room is now full, removing from queue", "port", room.Port)
					delete(rm.PracticeRooms, roomID)
				}

				// Switch user to the existing room
				regResponseCh := make(chan event.Message, 1)
				clientRegistration := event.ClientRegistrationMessage{
					ClientID:  roomRequest.Username,
					RoomID:    roomID,
					Conn:      roomRequest.Conn,
					ReponseCh: regResponseCh,
				}
				rm.broker.Publish(clientRegistration)

				// Wait for client registration to complete
				regResponse := <-regResponseCh
				if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && !regResp.Success {
					rm.logger.Infow("[ROOM MANAGER] Existing practice room register", "port", room.Port)
				}

				return event.RoomSearchMessage{
					Success: 0,
					RoomID:  roomID,
					RoomIP:  room.Port,
				}
			}
		}
	case CLASSIC:
		rm.mu.Lock()
		defer rm.mu.Unlock()

		// Find an existing room with space
		for roomID, room := range rm.ClassicRooms {
			if room.PlayersIn < room.MaxPlayers {
				room.PlayersIn++
				rm.logger.Infow("[ROOM MANAGER] Player joined existing classic room", "port", room.Port, "players", room.PlayersIn)

				if room.PlayersIn == room.MaxPlayers {
					rm.logger.Infow("[ROOM MANAGER] Classic room is now full, removing from queue", "port", room.Port)
					delete(rm.ClassicRooms, roomID)
				}

				// Switch user to the existing room
				regResponseCh := make(chan event.Message, 1)
				clientRegistration := event.ClientRegistrationMessage{
					ClientID:  roomRequest.Username,
					RoomID:    roomID,
					Conn:      roomRequest.Conn,
					ReponseCh: regResponseCh,
				}
				rm.broker.Publish(clientRegistration)

				// Wait for client registration to complete
				regResponse := <-regResponseCh
				if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && !regResp.Success {
					rm.logger.Infow("[ROOM MANAGER] Existing classic room register", "port", room.Port)
				}

				return event.RoomSearchMessage{
					Success: 0,
					RoomID:  roomID,
					RoomIP:  room.Port,
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
	roomID, err := StartGame(portStr, "1", maxPlayersStr)
	if err != nil {
		rm.logger.Errorw("[ROOM MANAGER] Failed to create room", "error", err)
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
		rm.logger.Infow("[ROOM MANAGER] Created new pratice room", "port", portStr)
	case CLASSIC:
		newRoom := &ClassicRoom{
			RoomID:     roomID,
			Port:       portStr,
			PlayersIn:  1,
			MaxPlayers: maxPlayers,
		}
		rm.ClassicRooms[roomID] = newRoom
		rm.logger.Infow("[ROOM MANAGER] Created new classic room", "port", portStr)
	}

	// We send the roomID to the message service for the user to be switch
	regResponseCh := make(chan event.Message, 1)
	rm.logger.Infow("[ROOM MANAGER] chan created", "port", portStr)
	clientRegistration := event.ClientRegistrationMessage{
		ClientID:  roomRequest.Username,
		RoomID:    roomID,
		Conn:      roomRequest.Conn,
		ReponseCh: regResponseCh,
	}
	rm.broker.Publish(clientRegistration)
	rm.logger.Infow("[ROOM MANAGER] message publish", "port", portStr)
	// Wait for client registration to complete
	regResponse := <-regResponseCh
	if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && !regResp.Success {
		rm.logger.Infow("[ROOM MANAGER] New room register", "port", portStr)

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
