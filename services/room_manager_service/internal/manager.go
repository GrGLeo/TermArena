package internal

import (
	"log/slog"
	"sync"
	"time"
)

type Room struct {
	mu         sync.RWMutex
	maxPlayers int
	Players    map[string]Spells
}

func NewRoom(roomType RoomType) *Room {
	var maxPlayers int
	switch roomType {
	case SANDBOX:
		maxPlayers = 1
	case PRACTICE:
		maxPlayers = 4
	case CLASSIC:
		maxPlayers = 8
	}
	players := make(map[string]Spells)
	return &Room{
		maxPlayers: maxPlayers,
		Players:    players,
	}
}

func (r *Room) SetPlayerSpell(username string, spellOne, spellTwo int) RoomStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players[username] = Spells{spellOne, spellTwo}
	if len(r.Players) == r.maxPlayers {
		return LOBBY
	} else {
		return WAITING
	}
}

func (r *Room) GetUsernames() []string {
  r.mu.RLock()
  r.mu.Unlock()
  keys := make([]string, 0, len(r.Players))
  for k := range r.Players {
    keys = append(keys, k)
  }
  return keys
}
  
type RoomManager struct {
	mu     sync.RWMutex
	logger *slog.Logger
	rooms  map[RoomType]map[RoomStatus]map[RoomID]*Room
}

func NewRoomManager(logger *slog.Logger) *RoomManager {
	rooms := make(map[RoomType]map[RoomStatus]map[RoomID]*Room)
	for _, rt := range []RoomType{SANDBOX, PRACTICE, CLASSIC} {
		rooms[rt] = make(map[RoomStatus]map[RoomID]*Room)
		for _, rs := range []RoomStatus{WAITING, LOBBY, READY, PROGRESS} {
			rooms[rt][rs] = make(map[RoomID]*Room)
		}
	}
	return &RoomManager{
		logger: logger,
		rooms:  rooms,
	}
}

func (rm *RoomManager) moveRoom(roomStatusIn, roomStatusOut RoomStatus, roomType RoomType, roomID RoomID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exist := rm.rooms[roomType][roomStatusIn][roomID]; exist {
		delete(rm.rooms[roomType][roomStatusIn], roomID)
		rm.rooms[roomType][roomStatusOut][roomID] = room
		rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", roomStatusIn, "roomStatusOut", roomStatusOut)
	}
}

func (rm *RoomManager) LookRoom(username string, roomType RoomType) RoomID {
	// Check WAITING room
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if waitingRooms, exist := rm.rooms[roomType][WAITING]; exist {
		for roomID, room := range waitingRooms {
			status := room.SetPlayerSpell(username, 0, 1)
			if status == LOBBY {
				delete(rm.rooms[roomType][WAITING], roomID)
				rm.rooms[roomType][LOBBY][roomID] = room
				rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "WAITING", "roomStatusOut", "LOBBY")
				go func() {
					<-time.After(1 * time.Minute)
					rm.mu.Lock()
					delete(rm.rooms[roomType][LOBBY], roomID)
					rm.rooms[roomType][READY][roomID] = room
					rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "LOBBY", "roomStatusOut", "READY")
					rm.mu.Unlock()
				}()
			}
			return roomID
		}
	}
	// We need to create the room
	newRoom := NewRoom(roomType)
	status := newRoom.SetPlayerSpell(username, 0, 1)
	roomID := GenerateRoomID()
	if status == LOBBY {
		rm.rooms[roomType][LOBBY][roomID] = newRoom
		rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "WAITING", "roomStatusOut", "LOBBY")
		go func() {
			<-time.After(1 * time.Minute)
			rm.mu.Lock()
			delete(rm.rooms[roomType][LOBBY], roomID)
			rm.rooms[roomType][READY][roomID] = newRoom
			rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "LOBBY", "roomStatusOut", "READY")
			rm.mu.Unlock()
		}()
	} else {
	  rm.rooms[roomType][WAITING][roomID] = newRoom
  }
	return roomID
}

func (rm *RoomManager) ScanReady() {
}
