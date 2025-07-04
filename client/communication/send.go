package communication

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/GrGLeo/ctf/shared"
	tea "github.com/charmbracelet/bubbletea"
)

func MakeConnection(port string) (*net.TCPConn, error) {
  serverIP := os.Getenv("SERVER_IP")
  if serverIP == "" {
    serverIP = "localhost" // Default to localhost if not set
  }

  log.Printf("Connection Attempt: %s:%s\n", serverIP, port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", serverIP, port))
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
    log.Printf("Failed to make connection: %q\n", err)
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

func SendSignInPacket(conn *net.TCPConn, username, password string) error {
  log.Print("sending message")
  createPacket := shared.NewSignInPacket(username, password)
  data := createPacket.Serialize()
  _, err := conn.Write(data)
  return err
}

func SendRoomRequestPacket(conn *net.TCPConn, roomType int) error {
  log.Println("sending room request")
  roomRequestPacket := shared.NewRoomRequestPacket(roomType)
  data := roomRequestPacket.Serialize()
  _, err := conn.Write(data)
  return err
}

func SendRoomJoinPacket(conn *net.TCPConn, roomID string) error {
  log.Println("sending room join")
  roomJoinPakcet := shared.NewRoomJoinPacket(roomID)
  data := roomJoinPakcet.Serialize()
  _, err := conn.Write(data)
  return err
}

func SendRoomCreatePacket(conn *net.TCPConn, roomType int) error {
  log.Println("sending room creation")
  roomCreatePacket := shared.NewRoomCreatePacket(roomType)
  data := roomCreatePacket.Serialize()
  _, err := conn.Write(data)
  return err
}

func SendAction(conn *net.TCPConn, action int) error {
  log.Println("Sent action")
  actionPacket := shared.NewActionPacket(action)
  data := actionPacket.Serialize()
  _, err := conn.Write(data)
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
    message, err := shared.DeSerialize(buf[:n])
    if err != nil {
      return
    }
    switch msg := message.(type) {
    case *shared.RespPacket:
      msgs <- ResponseMsg{Code: msg.Success}
    case *shared.LookRoomPacket:
      msgs <- LookRoomMsg{Code: msg.Success, RoomID: msg.RoomID, RoomIP: msg.RoomIP}
    case *shared.GameStartPacket:
      log.Println("Game started packet found")
      msgs <- GameStartMsg{Code: msg.Success}
    case *shared.GameClosePacket:
      msgs <- GameCloseMsg{Code: msg.Success}
    case *shared.BoardPacket:
      board, err := DecodeRLE(msg.EncodedBoard)
      if err != nil {
        log.Print(err.Error())
      }
      health := [2]int{msg.Health, msg.MaxHealth}
      xp := [2]int{msg.Xp, msg.XpNeeded}
      msgs <- BoardMsg{Points: msg.Points, Health: health, Level: msg.Level, Xp: xp,  Board: board}
    case *shared.DeltaPacket:
      deltas := DecodeDeltas(msg.Deltas)
      msgs <- DeltaMsg{Points: msg.Points, Deltas: deltas, TickID: msg.TickID}
    case *shared.EndGamePacket:
      msgs <- EndGameMsg{Win: msg.Win}
    default:
      log.Printf("Unknown type: %T\n", message)
      msgs <- GamePacketMsg{Packet: buf[:n]}
    }
  }
}
