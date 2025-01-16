package communication

import (
	"net"

	"github.com/GrGLeo/ctf/shared"
)

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
