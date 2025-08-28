package event

import (
	"errors"
	"net"
)

type Message interface {
	Type() string
	Validate() error
	ResponseChan() chan Message
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
	ClientID  string
	RoomID    int
	Conn      *net.TCPConn
	ReponseCh chan Message
}

func (m ClientRegistrationMessage) Type() string { return "client_registration" }
func (m ClientRegistrationMessage) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientRegistrationMessage) ResponseChan() chan Message { return m.ReponseCh }

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
	ClientID  string
	ReponseCh chan Message
}

func (m ClientUnregistrationMessage) Type() string { return "client_unregistration" }
func (m ClientUnregistrationMessage) Validate() error {
	if m.ClientID == "" {
		return errors.New("client ID cannot be empty")
	}
	return nil
}
func (m ClientUnregistrationMessage) ResponseChan() chan Message { return m.ReponseCh }

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
	Conn     *net.TCPConn
	// ResponseCh is the channel to send the response to.
	ResponseCh chan Message
}

func (fm RoomRequestMessage) Type() string { return "find-room" }
func (fm RoomRequestMessage) Validate() error {
	if fm.RoomType < 0 || fm.RoomType >= 2 {
		return errors.New("Invalid room type")
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

type RoomCreateMessage struct {
	RoomType int
	Conn     *net.TCPConn
	// ResponseCh is the channel to send the response to.
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

type RoomSearchMessage struct {
	Success int
	RoomID  string
	RoomIP  string
}

func (rs RoomSearchMessage) Type() string { return "search-room" }
func (rs RoomSearchMessage) Validate() error {
	if rs.Success == 1 {
		return errors.New("Failed to search for a room")
	}
	return nil
}
func (rs RoomSearchMessage) ResponseChan() chan Message { return nil }

type MessageRequestMessage struct {
	Sender     string
	Message    string
	Conn       *net.TCPConn
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
