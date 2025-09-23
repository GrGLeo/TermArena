package internal

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
)

type Player struct {
	playerSpells Spells
	playerTeam   Team
}

func NewPlayer(team Team) *Player {
	return &Player{
		playerSpells: Spells{0, 1},
		playerTeam:   team,
	}
}

func (p *Player) UpdateSpells(spells Spells) {
	p.playerSpells = spells
}

type Room struct {
	mu         sync.RWMutex
	blueCount  int
	redCount   int
	maxPlayers int
	Players    map[string]*Player
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
	players := make(map[string]*Player)
	return &Room{
		maxPlayers: maxPlayers,
		Players:    players,
		blueCount:  0,
		redCount:   0,
	}
}

func (r *Room) AddPlayer(username string) (Team, RoomStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var team Team
	if r.blueCount <= r.redCount {
		team = BLUETEAM
		r.blueCount++
	} else {
		team = REDTEAM
		r.redCount++
	}

	newPlayer := NewPlayer(team)
	r.Players[username] = newPlayer
	if len(r.Players) == r.maxPlayers {
		return team, LOBBY
	} else {
		return team, WAITING
	}
}

func (r *Room) RemovePlayer(username string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if player, exist := r.Players[username]; exist {
		if player.playerTeam == BLUETEAM {
			r.blueCount--
		} else {
			r.redCount--
		}
	}
	delete(r.Players, username)
}

func (r *Room) UpdatePlayerSpell(username string, spellOne, spellTwo int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players[username].UpdateSpells(Spells{spellOne, spellTwo})
}

func (r *Room) GetUsernames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.Players))
	for k := range r.Players {
		keys = append(keys, k)
	}
	return keys
}

type RoomManager struct {
	mu          sync.RWMutex
	maxRoom     int
	roomCounter int
	logger      *slog.Logger
	roomLookup  map[RoomID]struct {
		roomType   RoomType
		roomStatus RoomStatus
	}
	rooms map[RoomType]map[RoomStatus]map[RoomID]*Room
}

func NewRoomManager(maxRoom int, logger *slog.Logger) *RoomManager {
	roomLookup := make(map[RoomID]struct {
		roomType   RoomType
		roomStatus RoomStatus
	})
	rooms := make(map[RoomType]map[RoomStatus]map[RoomID]*Room)
	for _, rt := range []RoomType{SANDBOX, PRACTICE, CLASSIC} {
		rooms[rt] = make(map[RoomStatus]map[RoomID]*Room)
		for _, rs := range []RoomStatus{WAITING, LOBBY, READY, PROGRESS} {
			rooms[rt][rs] = make(map[RoomID]*Room)
		}
	}
	rm := &RoomManager{
		maxRoom:     maxRoom,
		roomCounter: 0,
		roomLookup:  roomLookup,
		logger:      logger,
		rooms:       rooms,
	}
	return rm
}

func (rm *RoomManager) MoveRoom(roomStatusIn, roomStatusOut RoomStatus, roomType RoomType, roomID RoomID) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exist := rm.rooms[roomType][roomStatusIn][roomID]; exist {
		delete(rm.rooms[roomType][roomStatusIn], roomID)
		rm.rooms[roomType][roomStatusOut][roomID] = room
		rm.roomLookup[roomID] = struct {
			roomType   RoomType
			roomStatus RoomStatus
		}{roomType, roomStatusOut}
		rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", roomStatusIn, "roomStatusOut", roomStatusOut)
	}
}

func (rm *RoomManager) LookRoom(username string, roomType RoomType) (Team, RoomID, RoomStatus) {
	// Check WAITING room
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if waitingRooms, exist := rm.rooms[roomType][WAITING]; exist {
		for roomID, room := range waitingRooms {
			team, status := room.AddPlayer(username)
			if status == LOBBY {
				delete(rm.rooms[roomType][WAITING], roomID)
				rm.rooms[roomType][LOBBY][roomID] = room
				rm.roomLookup[roomID] = struct {
					roomType   RoomType
					roomStatus RoomStatus
				}{roomType, LOBBY}
				rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "WAITING", "roomStatusOut", "LOBBY")
				go func() {
					<-time.After(1 * time.Minute)
					rm.mu.Lock()
					delete(rm.rooms[roomType][LOBBY], roomID)
					rm.rooms[roomType][READY][roomID] = room
					rm.roomLookup[roomID] = struct {
						roomType   RoomType
						roomStatus RoomStatus
					}{roomType, READY}
					rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "LOBBY", "roomStatusOut", "READY")
					rm.mu.Unlock()
				}()
			}
			return team, roomID, status
		}
	}
	// We need to create the room
	if rm.roomCounter >= rm.maxRoom {
		// TODO: we should throw an error server full or something
	}
	rm.roomCounter += 1
	newRoom := NewRoom(roomType)
	team, status := newRoom.AddPlayer(username)
	roomID := GenerateRoomID()
	if status == LOBBY {
		rm.rooms[roomType][LOBBY][roomID] = newRoom
		rm.roomLookup[roomID] = struct {
			roomType   RoomType
			roomStatus RoomStatus
		}{roomType, LOBBY}
		rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "WAITING", "roomStatusOut", "LOBBY")
		go func() {
			<-time.After(1 * time.Minute)
			rm.mu.Lock()
			delete(rm.rooms[roomType][LOBBY], roomID)
			rm.rooms[roomType][READY][roomID] = newRoom
			rm.roomLookup[roomID] = struct {
				roomType   RoomType
				roomStatus RoomStatus
			}{roomType, READY}
			rm.logger.Info("room moved", "roomType", roomType, "roomStatusIn", "LOBBY", "roomStatusOut", "READY")
			rm.mu.Unlock()
		}()
	} else {
		rm.rooms[roomType][WAITING][roomID] = newRoom
		rm.roomLookup[roomID] = struct {
			roomType   RoomType
			roomStatus RoomStatus
		}{roomType, WAITING}
	}
	return team, roomID, status
}

func (rm *RoomManager) RemovePlayer(roomID RoomID, username string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if meta, exist := rm.roomLookup[roomID]; exist {
		if room, exist := rm.rooms[meta.roomType][meta.roomStatus][roomID]; exist {
			room.RemovePlayer(username)
			return nil
		}
	}
	return errors.New("room or player not found")
}

func (rm *RoomManager) GetRoom(roomID RoomID) (*Room, RoomType, RoomStatus, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if meta, exist := rm.roomLookup[roomID]; exist {
		if room, exists := rm.rooms[meta.roomType][meta.roomStatus][roomID]; exists {
			return room, meta.roomType, meta.roomStatus, true
		}
	}
	return nil, 0, 0, false
}

func (rm *RoomManager) GetRoomUsers(roomType RoomType, roomStatus RoomStatus, roomID RoomID) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if room, exist := rm.rooms[roomType][roomStatus][roomID]; exist {
		users := room.GetUsernames()
		return users, nil
	}
	return nil, errors.New("room not found")
}

func (rm *RoomManager) GetRoomInfo(roomType RoomType, roomStatus RoomStatus, roomID RoomID) ([]*pb.UserInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if room, exist := rm.rooms[roomType][roomStatus][roomID]; exist {
		var usersInfo []*pb.UserInfo
		for username, player := range room.Players {
			userInfo := &pb.UserInfo{
				Username: username,
				Team:     uint32(player.playerTeam),
				Spell1:   uint32(player.playerSpells[0]),
				Spell2:   uint32(player.playerSpells[1]),
			}
			usersInfo = append(usersInfo, userInfo)
		}
		return usersInfo, nil
	}
	return nil, errors.New("room not found")
}
