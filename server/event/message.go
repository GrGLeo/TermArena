package event

import (
	"errors"
	"fmt"
	"net"

	"github.com/GrGLeo/TermArena/pkg/shared"
	conm "github.com/GrGLeo/TermArena/server/conn_manager"
)

type Message interface {
	Type() string
	Validate() error
	ResponseChan() chan Message
}

func CreateMessage(packet shared.Packet, conn *net.TCPConn, connManager *conm.ConnectionManager) (Message, error) {
	responseChan := make(chan Message)
	switch pkt := packet.(type) {
	case *shared.RegisterRequestPacket:
		return RegisterRequestMessage{
			Username:   pkt.Username,
			PublicKey:  pkt.PublicKey,
			Conn:       conn,
			ResponseCh: responseChan,
		}, nil
	case *shared.LoginChallengeRequestPacket:
		return LoginChallengeRequestMessage{
			Username:   pkt.Username,
			Conn:       conn,
			ResponseCh: responseChan,
		}, nil
	case *shared.AuthRequestPacket:
		return AuthRequestMessage{
			Username:        pkt.Username,
			SignedChallenge: pkt.SignedChallenge,
			Conn:            conn,
			ResponseCh:      responseChan,
		}, nil
	case *shared.RoomRequestPacket:
		user, exist := connManager.GetUser(conn)
		if !exist {
			return nil, fmt.Errorf("failed to find associated user")
		}
		return RoomRequestMessage{
			RoomType:   pkt.RoomType,
			Conn:       conn,
			Username:   user,
			ResponseCh: responseChan,
		}, nil
	case *shared.RoomCreatePacket:
		return RoomCreateMessage{
			RoomType:   pkt.RoomType,
			Conn:       conn,
			ResponseCh: responseChan,
		}, nil
	case *shared.RoomJoinPacket:
		return RoomJoinMessage{
			RoomID:     pkt.RoomID,
			Conn:       conn,
			ResponseCh: responseChan,
		}, nil
	case *shared.QuitRoomPacket:
		user, exist := connManager.GetUser(conn)
		if !exist {
			return nil, fmt.Errorf("failed to find associated user")
		}
		return QuitRoomMessage{
			RoomID:     pkt.RoomID,
			Username:   user,
			Conn:       conn,
			ResponseCh: responseChan,
		}, nil
	case *shared.UpdateSpellReqPacket:
		_, exist := connManager.GetUser(conn)
		if !exist {
			return nil, fmt.Errorf("failed to find associated user")
		}
		return UpdateSpellReqMessage{
			RoomType:   pkt.RoomType,
			RoomID:     pkt.RoomID,
			Username:   pkt.Username,
			SpellOne:   pkt.SpellOne,
			SpellTwo:   pkt.SpellTwo,
			ResponseCh: responseChan,
		}, nil

	case *shared.MessagePacket:
		user, exist := connManager.GetUser(conn)
		if !exist {
			return nil, fmt.Errorf("failed to find associated user")
		}
		return MessageRequestMessage{
			Sender:     pkt.Sender,
			Message:    pkt.Message,
			Conn:       conn,
			User:       user,
			ResponseCh: responseChan,
		}, nil
	default:
		return nil, errors.New("No message to create from packet")
	}
}

func CreatePacketFromMessage(msg Message) ([]byte, error) {
	switch m := msg.(type) {
	case RegisterResponseMessage:
		packet := shared.NewRegisterResponsePacket(m.Success, m.Message, m.Challenge)
		return packet.Serialize(), nil
	case LoginChallengeResponseMessage:
		packet := shared.NewLoginChallengeResponsePacket(m.Challenge)
		return packet.Serialize(), nil
	case AuthResponseMessage:
		packet := shared.NewAuthResponsePacket(m.Success, m.Message, m.SessionToken)
		return packet.Serialize(), nil
	case LookRoomResponseMessage:
		packet := shared.NewLookRoomPacket(m.Success, m.RoomID)
		return packet.Serialize(), nil
	case UpdateSpellResMessage:
		packet := shared.NewUpdateSpellResPacket(m.Username, m.SpellOne, m.SpellTwo)
		return packet.Serialize(), nil
	case MessageResponseMessage:
		packet := shared.NewMessageResponsePacket(m.Message)
		return packet.Serialize(), nil
	case MessageErrorResponse:
		packet := shared.NewMessageErrorPacket(m.Error)
		return packet.Serialize(), nil
	case RateLimitResponse:
		packet := shared.NewRateLimitPacket()
		return packet.Serialize(), nil
	default:
		return nil, errors.New("Failed to create packet from message")
	}
}

// --- AUTH REQUESTS ---

type RegisterRequestMessage struct {
	Username  string
	PublicKey []byte
	Conn      *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (rrm RegisterRequestMessage) Type() string               { return "register_request" }
func (rrm RegisterRequestMessage) Validate() error            { return nil }
func (rrm RegisterRequestMessage) ResponseChan() chan Message { return rrm.ResponseCh }

type LoginChallengeRequestMessage struct {
	Username string
	Conn     *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (lcrm LoginChallengeRequestMessage) Type() string    { return "login_challenge_request" }
func (lcrm LoginChallengeRequestMessage) Validate() error { return nil }
func (lcrm LoginChallengeRequestMessage) ResponseChan() chan Message {
	return lcrm.ResponseCh
}

type AuthRequestMessage struct {
	Username        string
	SignedChallenge []byte
	Conn            *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (arm AuthRequestMessage) Type() string               { return "auth_request" }
func (arm AuthRequestMessage) Validate() error            { return nil }
func (arm AuthRequestMessage) ResponseChan() chan Message { return arm.ResponseCh }

// --- AUTH RESPONSES ---

type RegisterResponseMessage struct {
	Success   bool
	Message   string
	Challenge []byte
	Conn      *net.TCPConn
}

func (m RegisterResponseMessage) Type() string               { return "register_response" }
func (m RegisterResponseMessage) Validate() error            { return nil }
func (m RegisterResponseMessage) ResponseChan() chan Message { return nil }

type LoginChallengeResponseMessage struct {
	Challenge []byte
	Conn      *net.TCPConn
}

func (m LoginChallengeResponseMessage) Type() string               { return "login_challenge_response" }
func (m LoginChallengeResponseMessage) Validate() error            { return nil }
func (m LoginChallengeResponseMessage) ResponseChan() chan Message { return nil }

type AuthResponseMessage struct {
	Success      bool
	Message      string
	SessionToken string
	Conn         *net.TCPConn
}

func (m AuthResponseMessage) Type() string               { return "auth_response" }
func (m AuthResponseMessage) Validate() error            { return nil }
func (m AuthResponseMessage) ResponseChan() chan Message { return nil }

// --- CLIENT REGISTRATION ---
type ClientRegistrationMessage struct {
	ClientID   string
	RoomID     uint32
	TeamID     uint32
	Conn       *net.TCPConn
	ResponseCh chan Message
}

func (m ClientRegistrationMessage) Type() string { return "client_registration" }
func (m ClientRegistrationMessage) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientRegistrationMessage) ResponseChan() chan Message { return m.ResponseCh }

type ClientRegistrationResponse struct {
	Success  bool
	Message  string
	ClientID string
}

func (m ClientRegistrationResponse) Type() string { return "client_registration_response" }
func (m ClientRegistrationResponse) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientRegistrationResponse) ResponseChan() chan Message { return nil }

type ClientUnregistrationMessage struct {
	ClientID   string
	ResponseCh chan Message
}

func (m ClientUnregistrationMessage) Type() string { return "client_unregistration" }
func (m ClientUnregistrationMessage) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientUnregistrationMessage) ResponseChan() chan Message { return m.ResponseCh }

type ClientUnregistrationResponse struct {
	Success  bool
	Message  string
	ClientID string
}

func (m ClientUnregistrationResponse) Type() string { return "client_unregistration_response" }
func (m ClientUnregistrationResponse) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientUnregistrationResponse) ResponseChan() chan Message { return nil }

// --- ROOM MESSAGES ---

type RoomRequestMessage struct {
	RoomType int
	Username string
	Conn     *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (fm RoomRequestMessage) Type() string { return "find-room" }
func (fm RoomRequestMessage) Validate() error {
	if fm.RoomType < 0 || fm.RoomType >= 4 {
		return fmt.Errorf("Invalid room type: %d", fm.RoomType)
	}

	if fm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}
func (fm RoomRequestMessage) ResponseChan() chan Message { return fm.ResponseCh }

type RoomJoinMessage struct {
	RoomID string
	Conn   *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (rm RoomJoinMessage) Type() string { return "join-room" }
func (rm RoomJoinMessage) Validate() error {
	if len(rm.RoomID) != 5 {
		return errors.New("Invalid room id")
	}

	if rm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}
func (rm RoomJoinMessage) ResponseChan() chan Message { return rm.ResponseCh }

type QuitRoomMessage struct {
	RoomID     uint32
	Username   string
	Conn       *net.TCPConn
	ResponseCh chan Message
}

func (rm QuitRoomMessage) Type() string { return "quit-room" }
func (rm QuitRoomMessage) Validate() error {
	if rm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}
func (rm QuitRoomMessage) ResponseChan() chan Message { return rm.ResponseCh }

type QuitRoomResponseMessage struct {
	RoomID     uint32
	Username   string
	Conn       *net.TCPConn
	ResponseCh chan Message
}

func (rm QuitRoomResponseMessage) Type() string { return "quit-room-response" }
func (rm QuitRoomResponseMessage) Validate() error {
	if rm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}
func (rm QuitRoomResponseMessage) ResponseChan() chan Message { return rm.ResponseCh }

type RoomCreateMessage struct {
	RoomType   int
	Conn       *net.TCPConn
	ResponseCh chan Message
}

func (rc RoomCreateMessage) Type() string { return "create-room" }
func (rc RoomCreateMessage) Validate() error {
	if rc.RoomType < 0 || rc.RoomType >= 3 {
		return errors.New("Invalid room type")
	}

	if rc.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}
func (rc RoomCreateMessage) ResponseChan() chan Message { return rc.ResponseCh }

type LookRoomResponseMessage struct {
	Success    bool
	RoomID     uint32
	RoomIP     string
	ResponseCh chan Message
}

func (rs LookRoomResponseMessage) Type() string { return "search-room" }
func (rs LookRoomResponseMessage) Validate() error {
	if rs.Success == false {
		return errors.New("Failed to search for a room")
	}
	return nil
}
func (rs LookRoomResponseMessage) ResponseChan() chan Message { return rs.ResponseCh }

type MoveToLobbyMessage struct {
	Usernames []string
	RoomID    uint32
}

func (ml MoveToLobbyMessage) Type() string { return "move-lobby" }
func (ml MoveToLobbyMessage) Validate() error {
	if len(ml.Usernames) == 0 {
		return errors.New("Room cannot be empty when ready")
	}
	return nil
}
func (ml MoveToLobbyMessage) ResponseChan() chan Message { return nil }

type UpdateSpellReqMessage struct {
	RoomType   int
	RoomID     int
	Username   string
	SpellOne   int
	SpellTwo   int
	ResponseCh chan Message
}

func (usm UpdateSpellReqMessage) Type() string { return "update-spell-request" }
func (usm UpdateSpellReqMessage) Validate() error {
	if usm.RoomType == 0 || usm.RoomID == 0 {
		return errors.New("Room information missing to perform spell update")
	}
	if usm.Username == "" {
		return errors.New("Username missing to perform spell update")
	}
	if usm.SpellOne == usm.SpellTwo {
		return errors.New("Spells cannot be identical")
	}
	return nil
}
func (usm UpdateSpellReqMessage) ResponseChan() chan Message { return usm.ResponseCh }

type UpdateSpellResMessage struct {
	Usernames  []string
	Username   string
	SpellOne   int
	SpellTwo   int
	ResponseCh chan Message
}

func (usm UpdateSpellResMessage) Type() string { return "update-spell-response" }
func (usm UpdateSpellResMessage) Validate() error {
	return nil
}
func (usm UpdateSpellResMessage) ResponseChan() chan Message { return usm.ResponseCh }

type MessageRequestMessage struct {
	Sender     string // TODO: to be removed in issue #159
	Message    string
	Conn       *net.TCPConn
	User       string
	ResponseCh chan Message
}

func (mrm MessageRequestMessage) Type() string { return "message_request" }
func (mrm MessageRequestMessage) Validate() error {
	if mrm.Sender == "" {
		return errors.New("Sender cannot be empty")
	}
	if mrm.Message == "" {
		return errors.New("Message cannot be empty")
	}
	return nil
}
func (mrm MessageRequestMessage) ResponseChan() chan Message { return mrm.ResponseCh }

type MessageResponseMessage struct {
	Receivers  []string
	Message    string
	ResponseCh chan Message
}

func (mrm MessageResponseMessage) Type() string { return "message_response" }
func (mrm MessageResponseMessage) Validate() error {
	if len(mrm.Receivers) == 0 {
		return errors.New("Receivers cannot be empty")
	}
	if mrm.Message == "" {
		return errors.New("Message cannot be empty")
	}
	return nil
}
func (mrm MessageResponseMessage) ResponseChan() chan Message { return mrm.ResponseCh }

type MessageErrorResponse struct {
	Error      string
	ResponseCh chan Message
}

func (mer MessageErrorResponse) Type() string               { return "message_error" }
func (mer MessageErrorResponse) Validate() error            { return nil }
func (mer MessageErrorResponse) ResponseChan() chan Message { return mer.ResponseCh }

type RateLimitResponse struct {
	ResponseCh chan Message
}

func (rlr RateLimitResponse) Type() string               { return "rate_limit_error" }
func (rlr RateLimitResponse) Validate() error            { return nil }
func (rlr RateLimitResponse) ResponseChan() chan Message { return rlr.ResponseCh }
