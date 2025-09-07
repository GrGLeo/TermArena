package manager

import (
	"fmt"
	"os"
	"os/exec"
  "math/rand"
	"time"
)

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
