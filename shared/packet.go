package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"

	"github.com/GrGLeo/ctf/server/event"
)

/*
code 0: send login
code 1: send create user
code 2: receive login response
code 3: send a find room
code 4: send a create room
code 5: send a join room
code 6: looking for a room response
code 7: game start  response
code 8: send action
code 9: receive RLEboard
code 10: receive Delta
code 11: game close
code 12: game end
*/

type Packet interface {
	Version() int
	Code() int
	Serialize() []byte
}

func CreateMessage(packet Packet, conn *net.TCPConn) (event.Message, error) {
	switch pkt := packet.(type) {
	case *LoginPacket:
		return event.LoginMessage{
			Username: pkt.Username,
			Password: pkt.Password,
		}, nil
	case *SignInPacket:
		return event.SignInMessage{
			Username: pkt.Username,
			Password: pkt.Password,
		}, nil
	case *RoomRequestPacket:
		return event.RoomRequestMessage{
			RoomType: pkt.RoomType,
			Conn:     conn,
		}, nil
	case *RoomCreatePacket:
		return event.RoomCreateMessage{
			RoomType: pkt.RoomType,
			Conn:     conn,
		}, nil
	case *RoomJoinPacket:
		return event.RoomJoinMessage{
			RoomID: pkt.RoomID,
			Conn:   conn,
		}, nil
	default:
		return nil, errors.New("No message to create from packet")
	}
}

func CreatePacketFromMessage(msg event.Message) ([]byte, error) {
	switch m := msg.(type) {
	case event.AuthMessage:
		if err := msg.Validate(); err != nil {
			packet := NewRespPacket(true)
			return packet.Serialize(), nil
		}
		packet := NewRespPacket(false)
		return packet.Serialize(), nil
	case event.RoomSearchMessage:
		packet := NewLookRoomPacket(m.Success, m.RoomID, m.RoomIP)
		return packet.Serialize(), nil
	default:
		return nil, errors.New("Failed to create packet from message")
	}
}

/*
LOGIN PACKET
*/

type LoginPacket struct {
	version, code      int
	Username, Password string
}

func NewLoginPacket(username, password string) *LoginPacket {
	return &LoginPacket{
		version:  1,
		code:     0,
		Username: username,
		Password: password,
	}
}

func (lp *LoginPacket) Version() int {
	return lp.version
}

func (lp *LoginPacket) Code() int {
	return lp.code
}

func (lp *LoginPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(lp.version))
	buf.WriteByte(byte(lp.code))

	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lp.Username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(lp.Username); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lp.Password))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(lp.Password); err != nil {
		return nil
	}
	return buf.Bytes()
}

type SignInPacket struct {
	version, code      int
	Username, Password string
}

func NewSignInPacket(username, password string) *SignInPacket {
	return &SignInPacket{
		version:  1,
		code:     1,
		Username: username,
		Password: password,
	}
}

func (cp *SignInPacket) Version() int {
	return cp.version
}

func (cp *SignInPacket) Code() int {
	return cp.code
}

func (cp *SignInPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(cp.version))
	buf.WriteByte(byte(cp.code))

	if err := binary.Write(&buf, binary.BigEndian, uint16(len(cp.Username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(cp.Username); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(cp.Password))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(cp.Password); err != nil {
		return nil
	}
	return buf.Bytes()
}

type RespPacket struct {
	version, code int
	Success       bool
}

func NewRespPacket(success bool) *RespPacket {
	return &RespPacket{
		version: 1,
		code:    2,
		Success: success,
	}
}

func (rp RespPacket) Version() int {
	return rp.version
}

func (rp RespPacket) Code() int {
	return rp.code
}

func (rp *RespPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(rp.version))
	buf.WriteByte(byte(rp.code))
	if rp.Success {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

/*
FIND GAME PACKET
*/
type RoomRequestPacket struct {
	version, code, RoomType int
}

func NewRoomRequestPacket(RoomType int) *RoomRequestPacket {
	return &RoomRequestPacket{
		version:  1,
		code:     3,
		RoomType: RoomType,
	}
}

func (fp RoomRequestPacket) Version() int {
	return fp.version
}

func (fp RoomRequestPacket) Code() int {
	return fp.code
}

func (fp *RoomRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(fp.version))
	buf.WriteByte(byte(fp.code))
	buf.WriteByte(byte(fp.RoomType))
	return buf.Bytes()
}

// TODO: implement private or not
type RoomCreatePacket struct {
	version, code, RoomType int
}

func NewRoomCreatePacket(RoomType int) *RoomCreatePacket {
	return &RoomCreatePacket{
		version:  1,
		code:     4,
		RoomType: RoomType,
	}
}

func (cp RoomCreatePacket) Version() int {
	return cp.version
}

func (cp RoomCreatePacket) Code() int {
	return cp.code
}

func (cp *RoomCreatePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(cp.version))
	buf.WriteByte(byte(cp.code))
	buf.WriteByte(byte(cp.RoomType))
	return buf.Bytes()
}

type RoomJoinPacket struct {
	version, code int
	RoomID        string
}

func NewRoomJoinPacket(roomID string) *RoomJoinPacket {
	return &RoomJoinPacket{
		version: 1,
		code:    5,
		RoomID:  roomID,
	}
}

func (cp RoomJoinPacket) Version() int {
	return cp.version
}

func (cp RoomJoinPacket) Code() int {
	return cp.code
}

func (cp *RoomJoinPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(cp.version))
	buf.WriteByte(byte(cp.code))
	buf.WriteString(cp.RoomID)
	return buf.Bytes()
}

type LookRoomPacket struct {
	version, code, Success int
	RoomID, RoomIP         string
}

func NewLookRoomPacket(success int, roomID, roomIP string) *LookRoomPacket {
	return &LookRoomPacket{
		version: 1,
		code:    6,
		Success: success,
		RoomID:  roomID,
		RoomIP:  roomIP,
	}
}

func (lp LookRoomPacket) Version() int {
	return lp.version
}

func (lp LookRoomPacket) Code() int {
	return lp.code
}

func (lp *LookRoomPacket) Serialize() []byte {
	var buf bytes.Buffer
	capacity := 3 + len(lp.RoomID) + len(lp.RoomIP)
	buf.Grow(capacity)
	buf.WriteByte(byte(lp.version))
	buf.WriteByte(byte(lp.code))
	buf.WriteByte(byte(lp.Success))
	if lp.RoomID != "" {
		buf.WriteString(lp.RoomID)
	} else {
		buf.WriteString("     ")
	}
	buf.WriteString(lp.RoomIP)
	return buf.Bytes()
}

type GameStartPacket struct {
	version, code, Success int
}

func NewGameStartPacket(success int) *GameStartPacket {
	return &GameStartPacket{
		version: 1,
		code:    7,
		Success: success,
	}
}

func (gp GameStartPacket) Version() int {
	return gp.version
}

func (gp GameStartPacket) Code() int {
	return gp.code
}

func (gp *GameStartPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(gp.version))
	buf.WriteByte(byte(gp.code))
	buf.WriteByte(byte(gp.Success))
	return buf.Bytes()
}

type GameClosePacket struct {
	version, code, Success int
}

func NewGameClosePacket(success int) *GameStartPacket {
	return &GameStartPacket{
		version: 1,
		code:    11,
		Success: success,
	}
}

func (gc GameClosePacket) Version() int {
	return gc.version
}

func (gc GameClosePacket) Code() int {
	return gc.code
}

func (gc *GameClosePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(gc.version))
	buf.WriteByte(byte(gc.code))
	buf.WriteByte(byte(gc.Success))
	return buf.Bytes()
}

type EndGamePacket struct {
	version, code int
	Win           bool
}

func NewEndGamePacket(win bool) *EndGamePacket {
	return &EndGamePacket{
		version: 1,
		code:    12,
		Win:     win,
	}
}

func (egp EndGamePacket) Version() int {
	return egp.version
}

func (egp EndGamePacket) Code() int {
	return egp.code
}

func (egp *EndGamePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(egp.version))
	buf.WriteByte(byte(egp.code))
	if egp.Win {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

/*
GAME PACKETS
*/

type ActionPacket struct {
	version, code int
	action        int
}

func NewActionPacket(action int) *ActionPacket {
	return &ActionPacket{
		version: 1,
		code:    8,
		action:  action,
	}
}

func (ap ActionPacket) Version() int {
	return ap.version
}

func (ap ActionPacket) Code() int {
	return ap.code
}

func (ap ActionPacket) Action() int {
	return ap.action
}

func (ap ActionPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(ap.version))
	buf.WriteByte(byte(ap.code))
	buf.WriteByte(byte(ap.action))
	return buf.Bytes()

}

type BoardPacket struct {
	version, code int
	Points        [2]int
	Health        int
	MaxHealth     int
	Level         int
	Xp            int
	XpNeeded      int
	Length        int
	EncodedBoard  []byte
}

func NewBoardPacket(health, maxHealth, level, xp, xpNeeded, length int, points [2]int, encodedBoard []byte) *BoardPacket {
	return &BoardPacket{
		version:      1,
		code:         9,
		Points:       points,
		Health:       health,
		MaxHealth:    maxHealth,
		Level:        level,
		Xp:           xp,
		XpNeeded:     xpNeeded,
		Length:       length,
		EncodedBoard: encodedBoard,
	}
}

func (bp BoardPacket) Version() int {
	return bp.version
}

func (bp BoardPacket) Code() int {
	return bp.code
}

func (bp *BoardPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(bp.version))
	buf.WriteByte(byte(bp.code))
	buf.WriteByte(byte(bp.Points[0]))
	buf.WriteByte(byte(bp.Points[1]))
	buf.WriteByte(byte(bp.Health))
	buf.WriteByte(byte(bp.MaxHealth))
	buf.WriteByte(byte(bp.Length))
	buf.Write(bp.EncodedBoard)
	return buf.Bytes()
}

type DeltaPacket struct {
	version, code int
	TickID        uint32
	Points        [2]int
	Deltas        [][3]byte
}

func NewDeltaPacket(tickID uint32, points [2]int, deltas [][3]byte) *DeltaPacket {
	return &DeltaPacket{
		version: 1,
		code:    10,
		Points:  points,
		TickID:  tickID,
		Deltas:  deltas,
	}
}

func (dp DeltaPacket) Version() int {
	return dp.version
}

func (dp DeltaPacket) Code() int {
	return dp.code
}

func (dp *DeltaPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(dp.version))
	buf.WriteByte(byte(dp.code))
	// Write the tickID on 4 byte
	TickIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(TickIDBytes, dp.TickID)
	buf.Write(TickIDBytes)
	// Write the teams points
	buf.WriteByte(byte(dp.Points[0]))
	buf.WriteByte(byte(dp.Points[1]))
	// Write the number of deltas on 2 byte
	deltaCount := len(dp.Deltas)
	deltaCountBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(deltaCountBytes, uint16(deltaCount))
	buf.Write(deltaCountBytes)
	// Write the deltas
	for _, delta := range dp.Deltas {
		buf.Write(delta[:])
	}
	return buf.Bytes()
}

// DeSerialize deserializes a byte slice into a specific Packet type based on the message code.
// It supports multiple packet types: LoginPacket, RespPacket, ActionPacket, and BoardPacket.
//
// Parameters:
// - data: A byte slice containing the serialized packet data.
//
// Returns:
// - Packet: The deserialized Packet, which could be one of the supported types.
// - error: An error if the data does not conform to the expected format or contains invalid values.
//
// The function performs the following steps:
//
// 1. Checks that the input data has at least 2 bytes for the version and code.
//
// 2. Validates the version, expecting it to be 1.
//
// 3. Reads the message code and parses the data accordingly:
//   - For LoginPacket, it extracts the username and password.
//   - For RespPacket, it returns a basic response packet.
//   - For ActionPacket, it reads the action value.
//   - For BoardPacket, it treats all remaining data as the encoded board.
//
// 4. Returns an error for unsupported or malformed packet types.
func DeSerialize(data []byte) (Packet, error) {
	// Check minimum packet length (version + code)
	if len(data) < 2 {
		return nil, errors.New("invalid packet length")
	}

	version := int(data[0])
	if version != 1 {
		return nil, errors.New("invalid version")
	}

	code := int(data[1])

	// Parse based on message type
	switch code {
	case 0: // LoginPacket
		if len(data) < 4 {
			return nil, errors.New("invalid login packet length")
		}

		// Username
		usernameLen := binary.BigEndian.Uint16(data[2:4])
		if len(data) < int(4+usernameLen) {
			return nil, errors.New("invalid username length")
		}
		username := string(data[4 : 4+usernameLen])

		// Password
		pwStart := 4 + usernameLen
		if len(data) < int(pwStart+2) {
			return nil, errors.New("invalid packet length for password length")
		}
		passwordLen := binary.BigEndian.Uint16(data[pwStart : pwStart+2])
		if len(data) < int(pwStart+2+passwordLen) {
			return nil, errors.New("invalid password length")
		}
		password := string(data[pwStart+2 : pwStart+2+passwordLen])

		return &LoginPacket{
			version:  version,
			code:     code,
			Username: username,
			Password: password,
		}, nil

	case 1: // SigninPacket
		if len(data) < 4 {
			return nil, errors.New("invalid login packet length")
		}

		// Username
		usernameLen := binary.BigEndian.Uint16(data[2:4])
		if len(data) < int(4+usernameLen) {
			return nil, errors.New("invalid username length")
		}
		username := string(data[4 : 4+usernameLen])

		// Password
		pwStart := 4 + usernameLen
		if len(data) < int(pwStart+2) {
			return nil, errors.New("invalid packet length for password length")
		}
		passwordLen := binary.BigEndian.Uint16(data[pwStart : pwStart+2])
		if len(data) < int(pwStart+2+passwordLen) {
			return nil, errors.New("invalid password length")
		}
		password := string(data[pwStart+2 : pwStart+2+passwordLen])

		return &SignInPacket{
			version:  version,
			code:     code,
			Username: username,
			Password: password,
		}, nil

	case 2: // RespPacket
		if data[2] == 0 {
			return &RespPacket{
				version: version,
				code:    code,
				Success: false,
			}, nil
		} else {
			return &RespPacket{
				version: version,
				code:    code,
				Success: true,
			}, nil
		}

	case 3:
		return &RoomRequestPacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}, nil

	case 4:
		return &RoomCreatePacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}, nil

	case 5:
		roomID := string(data[2:])
		return &RoomJoinPacket{
			version: version,
			code:    code,
			RoomID:  roomID,
		}, nil

	case 6:
		return &LookRoomPacket{
			version: version,
			code:    code,
			Success: int(data[2]),
			RoomID:  string(data[3:8]),
			RoomIP:  string(data[8:]),
		}, nil

	case 7:
		return &GameStartPacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}, nil

	case 8: // ActionPacket
		action := int(data[2])
		return &ActionPacket{
			version: version,
			code:    code,
			action:  action,
		}, nil

	case 9: // BoardPacket
		// First two bytes are points
		points := [2]int{}
		points[0] = int(data[2])
		points[1] = int(data[3])
    health := int(binary.BigEndian.Uint16(data[4:6]))
    maxHealth := int(binary.BigEndian.Uint16(data[6:8]))
    level := int(data[8])
    xp := int(binary.BigEndian.Uint32(data[9:13]))
    xpNeeded := int(binary.BigEndian.Uint32(data[13:17]))
		length := int(binary.BigEndian.Uint16(data[17:19]))
    log.Printf("Deserialize health: %d | %d", health, maxHealth)

		// Rest of data is the encodedBoard
		encodedBoard := data[19:length+19]
		return &BoardPacket{
			version:      version,
			code:         code,
			Points:       points,
			Health:       health,
			MaxHealth:    maxHealth,
			Level:        level,
			Xp:           xp,
			XpNeeded:     xpNeeded,
			Length:       length,
			EncodedBoard: encodedBoard,
		}, nil

	case 10: // DeltasPacket
		if len(data) < 6 {
			return nil, errors.New("invalid deltas packet length")
		}
		tickID := binary.BigEndian.Uint32(data[2:6])
		points := [2]int{}
		points[0] = int(data[6])
		points[1] = int(data[7])
		deltaCount := int(binary.BigEndian.Uint16(data[8:10]))
		expectedLength := 10 + deltaCount*3
		if len(data) < expectedLength {
			return nil, errors.New("invalid deltas packet length")
		}
		deltas := make([][3]byte, deltaCount)
		for i := range deltaCount {
			start := 10 + i*3
			end := start + 3
			copy(deltas[i][:], data[start:end])
		}

		return &DeltaPacket{
			version: version,
			code:    code,
			Points:  points,
			TickID:  tickID,
			Deltas:  deltas,
		}, nil

	case 11: // GameClosePacket
		// Success 0: won 1: loose 2: error
		return &GameClosePacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}, nil

	case 12: // EndGamePacket
		win := data[2] == 1
		return &EndGamePacket{
			version: version,
			code:    code,
			Win:     win,
		}, nil

	default:
		return nil, errors.New("unknown message type")
	}
}
