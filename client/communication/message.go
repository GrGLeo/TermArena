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
	Code bool
}

/*
LookRoomMsg is used to inform player is in queue
code 0: player in queue
code 1: error putting player in queue
*/
type LookRoomMsg struct {
	Code   int
	RoomID string
	RoomIP string
}

// GameStart is sent by the server once the number of player are matched
type GameStartMsg struct {
	Code int
}

// GameClose is sent after the server close
// Code: 0 win. 1 losse. 2 server error
type GameCloseMsg struct {
	Code int
}

// GoToShop is sent after receiving a response from the game
type GoToShopMsg struct {
	Health        int
	Mana          int
	Attack_damage int
	Armor         int
	Gold          int
	Inventory     []int
}

// UpdatePlayerStatsMsg is sent when the player buys an item
type UpdatePlayerStatsMsg struct {
	Health        int
	Mana          int
	Attack_damage int
	Armor         int
	Gold          int
	Inventory     []int
}

// BackToGame is sent when the player press 'p' while in Shop
type BackToGameMsg struct{}

// EndGameMsg is sent when the game ends
type EndGameMsg struct {
	Win bool
}

// BoardMsg is used to transfer the board to game model
type BoardMsg struct {
	Points [2]int
	Health [2]int
	Mana   [2]int
	Level  int
	Xp     [2]int
	Board  [21][51]int
}

type DeltaMsg struct {
	Points [2]int
	Deltas [][3]int
	TickID uint32
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

// Cooldown msg for abilities
type CooldownTickMsg struct{}
