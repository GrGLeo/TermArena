package manager

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/GrGLeo/ctf/server/event"
	"go.uber.org/zap"
)

type Room interface {
	GetRoomID() int
	GetPort() string
	GetPlayersIn() int
	GetMaxPlayers() int
	SetPlayersIn(int)
}

type ClassicRoom struct {
	RoomID     int
	Port       string
	PlayersIn  int
	MaxPlayers int
}

func (r *ClassicRoom) GetRoomID() int     { return r.RoomID }
func (r *ClassicRoom) GetPort() string    { return r.Port }
func (r *ClassicRoom) GetPlayersIn() int  { return r.PlayersIn }
func (r *ClassicRoom) GetMaxPlayers() int { return r.MaxPlayers }
func (r *ClassicRoom) SetPlayersIn(p int) { r.PlayersIn = p }

type PracticeRoom struct {
	RoomID     int
	Port       string
	PlayersIn  int
	MaxPlayers int
}

func (r *PracticeRoom) GetRoomID() int     { return r.RoomID }
func (r *PracticeRoom) GetPort() string    { return r.Port }
func (r *PracticeRoom) GetPlayersIn() int  { return r.PlayersIn }
func (r *PracticeRoom) GetMaxPlayers() int { return r.MaxPlayers }
func (r *PracticeRoom) SetPlayersIn(p int) { r.PlayersIn = p }

type FindRoomResult struct {
	Found   bool
	Message event.RoomSearchMessage
}

func StartGame(ip, map_id, max_players string) (int, error) {
	command := "./bin/game"
	args := []string{"--port", ip, "--map", map_id, "--max-players", max_players}
	cmd := exec.Command(command, args...)

	RoomID := rand.Intn(9999) + 1
	logFileName := fmt.Sprintf("rust_game_%d.log", RoomID)
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 066)
	if err != nil {
		return 0, error(err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		return 0, error(err)
	}
	fmt.Printf("Rust game server process started with PID: %d on port %s\n", cmd.Process.Pid, ip)
	fmt.Fprintf(logFile, "Rust game server process started with PID: %d on port %s.\n", cmd.Process.Pid, ip)
	time.Sleep(1 * time.Second)
	return RoomID, nil
}

func findRoom(rooms map[int]Room, roomRequest event.RoomRequestMessage, roomType string, broker *event.EventBroker, logger *zap.SugaredLogger) FindRoomResult {
	logger.Info("lookForRoom enter")
	// Find an existing room with space
	for roomID, room := range rooms {
		if room.GetPlayersIn() < room.GetMaxPlayers() {
			room.SetPlayersIn(room.GetMaxPlayers() + 1)
			logger.Infow(fmt.Sprintf("[ROOM MANAGER] Player joined existing %s room", roomType), "port", room.GetPort(), "players", room.GetPlayersIn())

			if room.GetPlayersIn() == room.GetMaxPlayers() {
				logger.Infow(fmt.Sprintf("[ROOM MANAGER] %s room is now full, removing from queue", roomType), "port", room.GetPort())
				delete(rooms, roomID)
			}

			// Switch user to the existing room
			regResponseCh := make(chan event.Message, 1)
			clientRegistration := event.ClientRegistrationMessage{
				ClientID:   roomRequest.Username,
				RoomID:     roomID,
				Conn:       roomRequest.Conn,
				ResponseCh: regResponseCh,
			}
			broker.Publish(clientRegistration)

			// Wait for client registration to complete
			regResponse := <-regResponseCh
			if regResp, ok := regResponse.(event.ClientRegistrationResponse); ok && !regResp.Success {
				logger.Infow("[ROOM MANAGER] Existing practice room register", "port", room.GetPort())
			}

			return FindRoomResult{
				Found: true,
				Message: event.RoomSearchMessage{
					Success: 0,
					RoomID:  roomID,
					RoomIP:  room.GetPort(),
				},
			}
		}
	}
	return FindRoomResult{
		Found:   false,
		Message: event.RoomSearchMessage{},
	}
}
