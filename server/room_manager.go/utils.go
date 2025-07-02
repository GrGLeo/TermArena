package manager

import (
	"fmt"
	"os"
	"os/exec"
  "math/rand"
	"time"
)

func StartGame(ip, map_id string) error {
	command := "./game/target/debug/game"
	args := []string{"--port", ip, "--map", map_id}
	cmd := exec.Command(command, args...)

  FileId := rand.Intn(9999) + 1
  logFileName := fmt.Sprintf("rust_game_%d.log", FileId)
  logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 066)
  if err != nil {
    fmt.Printf("Failed to open log file for rust_game_%s.log", ip)
    return error(err)
  }
  defer logFile.Close()

  cmd.Stdout = logFile
  cmd.Stderr = logFile

  err = cmd.Start()
  if err != nil {
    fmt.Printf("Failed to start game server %q", err)
  }
  fmt.Printf("Rust game server process started with PID: %d on port %s\n", cmd.Process.Pid, ip)
  fmt.Fprintf(logFile, "Rust game server process started with PID: %d on port %s.\n", cmd.Process.Pid, ip)
  time.Sleep(1 * time.Second)
	return nil
}
