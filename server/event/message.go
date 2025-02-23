package event

import (
	"errors"
	"net"
)

type Message interface {
	Type() string
	Validate() error
}

type LoginMessage struct {
	Username string
	Password string
}

func (lm LoginMessage) Type() string {
	return "login"
}

func (lm LoginMessage) Validate() error {
	if lm.Username == "" || lm.Password == "" {
		return errors.New("Username and Password are required")
	}
	return nil
}

type AuthMessage struct {
	Success int
}

func (am AuthMessage) Type() string {
	return "auth"
}

func (am AuthMessage) Validate() error {
	if am.Success != 0 {
		return errors.New("Wrong credential")
	}
	return nil
}

type RoomRequestMessage struct {
	RoomType int
	Conn     *net.TCPConn
}

func (fm RoomRequestMessage) Type() string {
	return "find-room"
}

func (fm RoomRequestMessage) Validate() error {
	if fm.RoomType < 0 || fm.RoomType >= 2 {
		return errors.New("Invalid room type")
	}

	if fm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}

type RoomJoinMessage struct {
	RoomID string
	Conn   *net.TCPConn
}

func (rm RoomJoinMessage) Type() string {
	return "join-room"
}

func (rm RoomJoinMessage) Validate() error {
	if len(rm.RoomID) != 5 {
		return errors.New("Invalid room id")
	}

	if rm.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}

type RoomCreateMessage struct {
	RoomType int
	Conn     *net.TCPConn
}

func (rc RoomCreateMessage) Type() string {
	return "create-room"
}

func (rc RoomCreateMessage) Validate() error {
	if rc.RoomType < 0 || rc.RoomType >= 3 {
		return errors.New("Invalid room type")
	}

	if rc.Conn == nil {
		return errors.New("Connection cannot be nil")
	}
	return nil
}

type RoomSearchMessage struct {
	Success int
	RoomID  string
}

func (rs RoomSearchMessage) Type() string {
	return "search-room"
}

func (rs RoomSearchMessage) Validate() error {
	if rs.Success == 1 {
		return errors.New("Failed to search for a room")
	}
	return nil
}
