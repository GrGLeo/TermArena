package communication

import (
	"net"

	"github.com/GrGLeo/ctf/shared"
	tea "github.com/charmbracelet/bubbletea"
)

type GamePacketMsg struct {
  Packet []byte
}

func SendLoginPacket(conn *net.TCPConn, username, password string) error {
  payload := []byte(username + string([]byte{0x00}) + password)
  packet := shared.NewPacket(1, 0, payload)
  data, err := packet.Serialize()
  if err != nil {
    return err
  }
  _, err = conn.Write(data)
  return err
}

func ListenForPackets(conn *net.TCPConn, msgs chan<- tea.Msg) {
  buf := make([]byte, 1024)
  for {
    n, err := conn.Read(buf)
    if err != nil {
      // Handle disconnection or read error
      return
    }
    // Send the packet as a message to the model
    msgs <- GamePacketMsg{Packet: buf[:n]}
  }
}
