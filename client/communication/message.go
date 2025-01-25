package communication

import (
	"net"
	"time"
)

// TickMsg is used to send a time-based tick message.
type TickMsg struct {
	Time time.Time
}

// LoginMsg is used to pass input field to meta model
type LoginMsg struct {
  Username string
  Password string
}

/*
ResponseMsg is used to validate login
code 0: login succes

code 1: invalid credential
*/
type ResponseMsg struct {
  Code int
}


// BoardMsg is used to transfer the board to game model
type BoardMsg struct {
  Board [20][50]int
}

type DeltaMsg struct {
  Deltas [][3]int
}

// ConnectionMsg to pass the connection to meta model
type ConnectionMsg struct {
    Conn *net.TCPConn
}

// GamePacketMsg is a default message send, but isn't handle yet
type GamePacketMsg struct {
  Packet []byte
}

// ReconnectMsg serve to signal the connection success 
type ReconnectMsg struct{}

