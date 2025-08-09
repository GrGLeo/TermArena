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
		//serverIP = "endurace.cloud" // Default to localhost if not set
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

func SendShopRequest(conn *net.TCPConn) error {
	log.Println("Sent shop request")
	shopReqPacket := shared.NewShopRequestPacket()
	data := shopReqPacket.Serialize()
	_, err := conn.Write(data)
	return err
}

func SendPurchaseItemPacket(conn *net.TCPConn, itemID int) error {
	log.Printf("Sending purchase item request for item ID: %d", itemID)
	purchasePacket := shared.NewPurchaseItemPacket(itemID)
	data := purchasePacket.Serialize()
	log.Println(data)
	_, err := conn.Write(data)
	return err
}

func SendSpellSelectionPacket(conn *net.TCPConn, spell1, spell2 int) error {
	log.Printf("Sending spell selection: %d, %d", spell1, spell2)
	spellPacket := shared.NewSpellSelectionPacket(spell1, spell2)
	data := spellPacket.Serialize()
	_, err := conn.Write(data)
	return err
}

func ListenForPackets(conn *net.TCPConn, msgs chan<- tea.Msg) {
	var data []byte
	buf := make([]byte, 4096)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			msgs <- GameCloseMsg{Code: 2} // Server error/unexpected close
			return
		}

		data = append(data, buf[:n]...)

		for len(data) > 0 {
			packet, bytesConsumed, err := shared.DeSerialize(data)
			if err != nil {
				if err.Error() == "incomplete packet" {
					// Not enough data, wait for more
					break
				} else {
					log.Printf("Error deserializing packet: %v", err)
					// Discard the buffer to prevent getting stuck on a bad packet
					data = nil
					continue
				}
			}

			data = data[bytesConsumed:]

			log.Printf("Deserialized packet type: %T", packet)
			switch msg := packet.(type) {
			case *shared.RespPacket:
				log.Printf("Sending RespMsg: %+v", msg)
				msgs <- ResponseMsg{Code: msg.Success}
			case *shared.LookRoomPacket:
				log.Printf("Sending LookRoomMsg: %+v", msg)
				msgs <- LookRoomMsg{Code: msg.Success, RoomID: msg.RoomID, RoomIP: msg.RoomIP}
			case *shared.GameStartPacket:
				log.Println("Game started packet found")
				log.Printf("Sending GameStartMsg: %+v", msg)
				msgs <- GameStartMsg{Code: msg.Success}
			case *shared.GameClosePacket:
				log.Printf("Sending GameCloseMsg: %+v", msg)
				msgs <- GameCloseMsg{Code: msg.Success}
			case *shared.ShopResponsePacket:
				log.Println("Sending GoToShopMsg")
				msgs <- GoToShopMsg{Health: msg.Health, Mana: msg.Mana, Attack_damage: msg.Attack_damage, Armor: msg.Armor, Gold: msg.Gold, Inventory: msg.Inventory}
			case *shared.BoardPacket:
				board, err := DecodeRLE(msg.EncodedBoard)
				if err != nil {
					log.Print(err.Error())
				}
				health := [2]int{msg.Health, msg.MaxHealth}
				mana := [2]int{msg.Mana, msg.MaxMana}
				xp := [2]int{msg.Xp, msg.XpNeeded}
				log.Printf("Sending BoardMsg: Health=%v, Level=%d, Xp=%v", health, msg.Level, xp)
				msgs <- BoardMsg{Points: msg.Points, Health: health, Mana: mana, Level: msg.Level, Xp: xp, Board: board}
			case *shared.DeltaPacket:
				deltas := DecodeDeltas(msg.Deltas)
				log.Printf("Sending DeltaMsg: TickID=%d, Deltas=%v", msg.TickID, deltas)
				msgs <- DeltaMsg{Points: msg.Points, Deltas: deltas, TickID: msg.TickID}
			case *shared.EndGamePacket:
				log.Printf("Sending EndGameMsg: Win=%t", msg.Win)
				msgs <- EndGameMsg{Win: msg.Win}
			default:
				log.Printf("Unknown type: %T, raw: %x", packet, data)
				msgs <- GamePacketMsg{Packet: data}
			}
		}
	}
}
