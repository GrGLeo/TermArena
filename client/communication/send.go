package communication

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/GrGLeo/TermArena/pkg/shared"
	tea "github.com/charmbracelet/bubbletea"
)

func AttemptGameConnection(roomIP string) tea.Cmd {
	return func() tea.Msg {
		for i := range 5 { // Retry 5 times
			conn, err := MakeConnection(roomIP)
			if err == nil {
				return GameConnectionMsg{Conn: conn}
			}
			log.Printf("Failed to connect to game server at %s (attempt %d/5): %s", roomIP, i+1, err)
			time.Sleep(1 * time.Second)
		}
		return GameConnectionFailedMsg{}
	}
}

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

func SendRegisterRequestPacket(conn *net.TCPConn, username string, publicKey []byte) error {
	registerRequestPacket := shared.NewRegisterRequestPacket(username, publicKey)
	data := registerRequestPacket.Serialize()
	_, err := conn.Write(data)
	return err
}

func SendLoginChallengeRequestPacket(conn *net.TCPConn, username string) error {
	loginChallengeRequestPacket := shared.NewLoginChallengeRequestPacket(username)
	data := loginChallengeRequestPacket.Serialize()
	_, err := conn.Write(data)
	return err
}

func SendAuthRequestPacket(conn *net.TCPConn, username string, signedChallenge []byte) error {
	authRequestPacket := shared.NewAuthRequestPacket(username, signedChallenge)
	data := authRequestPacket.Serialize()
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

func SendUsernamePacket(conn *net.TCPConn, username string) error {
	log.Printf("Sending username: %s", username)
	spellPacket := shared.NewUsernamePacket(username)
	data := spellPacket.Serialize()
	_, err := conn.Write(data)
	return err
}

func SendMessage(conn *net.TCPConn, sender, message string) error {
	log.Printf("[CLIENT] SendMessage: sender=%s, message='%s'", sender, message)
	messageLen := len(message)
	if messageLen == 0 {
		return errors.New("Message cannot be empty")
	} else if messageLen > 1024 {
		return errors.New("Message is too long")
	}
	messagePacket := shared.NewMessagePacket(sender, message)
	data := messagePacket.Serialize()
	log.Printf("[CLIENT] SendMessage: packet created, data length=%d", len(data))
	_, err := conn.Write(data)
	if err != nil {
		log.Printf("[CLIENT] SendMessage: ERROR writing to connection: %v", err)
	} else {
		log.Printf("[CLIENT] SendMessage: SUCCESS - message sent to server")
	}
	return err
}

func SendQuitRoom(conn *net.TCPConn) error {
	log.Printf("[CLIENT] QuitMessage")
	packet := shared.NewQuitRoomPacket()
	data := packet.Serialize()
	_, err := conn.Write(data)
	if err != nil {
		log.Printf("[CLIENT] QuitRoom: ERROR writing to connection: %v", err)
	} else {
		log.Printf("[CLIENT] QuitRoom: SUCCESS - spells sent to server")
	}
	return err

}

func SendUpdateSpell(conn *net.TCPConn, roomType, roomID int, username string, spells []int) error {
	log.Printf("[CLIENT] SendUpateSpell: sender=%s, spells='%d|%d'", username, spells[0], spells[1])
	spellPacket := shared.NewUpdateSpellReqPacket(roomType, roomID, username, spells[0], spells[1])
	data := spellPacket.Serialize()
	_, err := conn.Write(data)
	if err != nil {
		log.Printf("[CLIENT] UpdateSpell: ERROR writing to connection: %v", err)
	} else {
		log.Printf("[CLIENT] UpdateSpell: SUCCESS - spells sent to server")
	}
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
			case *shared.RegisterResponsePacket:
				msgs <- RegistrationResultMsg{Success: msg.Success, Message: msg.Message, Challenge: msg.Challenge}
			case *shared.LoginChallengeResponsePacket:
				msgs <- ChallengeReceivedMsg{Challenge: msg.Challenge}
			case *shared.AuthResponsePacket:
				msgs <- AuthResultMsg{Success: msg.Success, Message: msg.Message, SessionToken: msg.SessionToken}
			case *shared.LookRoomPacket:
				log.Printf("Sending LookRoomMsg: %+v", msg)
				msgs <- LookRoomMsg{Code: msg.Success, RoomID: msg.RoomID}
			case *shared.MoveToLobbyPacket:
				var userInfos []UserInfo
				for _, ui := range msg.UserInfos {
					userInfos = append(userInfos, UserInfo{
						Username: ui.Username,
						Team:     int(ui.Team),
						SpellOne: int(ui.Spell1),
						SpellTwo: int(ui.Spell2),
					})
				}
				log.Printf("Sending MoveLobbyRoomMsg: %+v", msg)
				msgs <- MoveLobbyRoomMsg{RoomID: int(msg.RoomID), UserInfos: userInfos}
			case *shared.UpdateSpellResPacket:
				msgs <- UpdateSpellMsg{Username: msg.Username, SpellOne: msg.SpellOne, SpellTwo: msg.SpellTwo}
			case *shared.GameServerReadyPacket:
				strRoomIP := strconv.Itoa(int(msg.RoomIP))
				msgs <- GameServerReadyMsg{strRoomIP}
			case *shared.GameStartPacket:
				log.Println("Game started packet found")
				log.Printf("Sending GameStartMsg: %+v", msg)
				msgs <- GameStartMsg{Code: msg.Success}
			case *shared.GameClosePacket:
				log.Printf("Sending GameCloseMsg: %+v", msg)
				msgs <- GameCloseMsg{Code: msg.Success}
			case *shared.ShopResponsePacket:
				log.Println("Sending GoToShopMsg")
				msgs <- GoToShopMsg{Health: msg.Health, Mana: msg.Mana, AttackDamage: msg.AttackDamage, MagicPower: msg.MagicPower, Armor: msg.Armor, Gold: msg.Gold, Inventory: msg.Inventory}
			case *shared.BoardPacket:
				board, err := DecodeRLE(msg.EncodedBoard)
				if err != nil {
					log.Print(err.Error())
				}
				casting := [2]int{msg.CastTime, msg.CastDuration}
				health := [2]int{msg.Health, msg.MaxHealth}
				mana := [2]int{msg.Mana, msg.MaxMana}
				xp := [2]int{msg.Xp, msg.XpNeeded}
				log.Printf("Sending BoardMsg: Casting=%v, Health=%v, Level=%d, Xp=%v", casting, health, msg.Level, xp)
				msgs <- BoardMsg{Casting: casting, Health: health, Mana: mana, Level: msg.Level, Xp: xp, Board: board}
			case *shared.DeltaPacket:
				deltas := DecodeDeltas(msg.Deltas)
				log.Printf("Sending DeltaMsg: TickID=%d, Deltas=%v", msg.TickID, deltas)
				msgs <- DeltaMsg{Points: msg.Points, Deltas: deltas, TickID: msg.TickID}
			case *shared.EndGamePacket:
				log.Printf("Sending EndGameMsg: Win=%t", msg.Win)
				msgs <- EndGameMsg{Win: msg.Win}
			case *shared.MessageResponsePacket:
				log.Printf("[CLIENT] Received message response: %s", msg.Message)
				msgs <- IncomingMessageMsg{
					Content: msg.Message,
				}
			case *shared.MessageErrorPacket:
				log.Printf("[CLIENT] Received message error: %s", msg.Error)
				msgs <- MessageErrorMsg{Error: msg.Error}
			case *shared.RateLimitPacket:
				msgs <- RateLimitMsg{}
			default:
				log.Printf("Unknown type: %T, raw: %x", packet, data)
				msgs <- GamePacketMsg{Packet: data}
			}
		}
	}
}
