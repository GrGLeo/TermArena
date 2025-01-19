package communication

import (
	"log"
	"net"

	"github.com/GrGLeo/ctf/shared"
	tea "github.com/charmbracelet/bubbletea"
)

func MakeConnection() (*net.TCPConn, error) {
  log.Println("Connection Attempt")
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, NewConnectionError(500, "Failed to dial server")
	}
	return conn, nil
}

func SendLoginPacket(conn *net.TCPConn, username, password string) error {
  log.Print("sending message")
  loginPacket := shared.NewLoginPacket(username, password)
  data := loginPacket.Serialize()
  _, err := conn.Write(data)
  return err
}

func SendAction(conn *net.TCPConn, action int) error {
  log.Println("sending action")
  actionPacket := shared.NewActionPacket(action)
  data := actionPacket.Serialize()
  _, err := conn.Write(data)
  return err
}

func ListenForPackets(conn *net.TCPConn, msgs chan<- tea.Msg) {
  log.Println("ListenForPackets enter")
  buf := make([]byte, 1024)
  for {
    n, err := conn.Read(buf)
    if err != nil {
      // Handle disconnection or read error
      return
    }
    // Send the packet as a message to the model
    message, err := shared.DeSerialize(buf[:n])
    log.Println("message", message)
    log.Printf("Unknown type: %T\n", message)
    if err != nil {
      return
    }
    switch msg := message.(type) {
    case *shared.RespPacket:
      msgs <- ResponseMsg{Code: msg.Code()}
    case *shared.BoardPacket:
      board, err := DecodeRLE(msg.EncodedBoard)
      if err != nil {
        log.Print(err.Error())
      }
      msgs <- BoardMsg{Board: board}
    default:
      msgs <- GamePacketMsg{Packet: buf[:n]}
    }
  }
}
