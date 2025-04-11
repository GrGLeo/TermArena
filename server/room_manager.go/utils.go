package manager

import (
	"fmt"
	"os/exec"
)

func StartGame(ip, map_id string) error {
    command := "./game/target/debug/ctf_game"
    args := []string{"--port", ip, "--map", map_id}
    cmd := exec.Command(command, args...)
    output, err := cmd.CombinedOutput()
    fmt.Println("Output:\n", string(output))
    cmd.Wait()
    if err != nil {
      return err
    }
    return nil
  }
