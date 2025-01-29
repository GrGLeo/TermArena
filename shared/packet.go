package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/GrGLeo/ctf/server/event"
)

/*
code 0: send login
code 1: receive login response
code 2: send a find room
code 3: looking for a room response
code 4: game start  response
code 5: send action
code 6: receive RLEboard
code 7: receive Delta
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
  case *RoomRequestPacket:
    return event.RoomRequestMessage{
      RoomType: pkt.RoomType,
      Conn: conn,
    }, nil
	default:
		return nil, errors.New("No message to create from packet")
	}
}

func CreatePacketFromMessage(msg event.Message) ([]byte, error) {
  switch m := msg.(type) {
	case event.AuthMessage:
		if err := msg.Validate(); err != nil {
			packet := NewRespPacket()
			return packet.Serialize(), nil
		}
		packet := NewRespPacket()
		return packet.Serialize(), nil
  case event.RoomSearchMessage:
    packet := NewLookRoomPacket(m.Success)
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

type RespPacket struct {
	version, code int
}

func NewRespPacket() *RespPacket {
	return &RespPacket{
		version: 1,
		code:    1,
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
		code:     2,
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

type LookRoomPacket struct {
	version, code, Success int
}

func NewLookRoomPacket(success int) *LookRoomPacket {
	return &LookRoomPacket{
		version:  1,
		code:     3,
    Success: success,
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
	buf.WriteByte(byte(lp.version))
	buf.WriteByte(byte(lp.code))
	buf.WriteByte(byte(lp.Success))
	return buf.Bytes()
}

type GameStartPacket struct {
	version, code, Success int
}

func NewGameStartPacket(success int) *GameStartPacket {
	return &GameStartPacket{
		version:  1,
		code:     4,
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
		code:    5,
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
	EncodedBoard  []byte
}

func NewBoardPacket(encodedBoard []byte) *BoardPacket {
	return &BoardPacket{
		version:      1,
		code:         6,
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
	buf.Write(bp.EncodedBoard)
	return buf.Bytes()
}

type DeltaPacket struct {
	version, code int
	tickID        uint32
	Deltas        [][3]byte
}

func NewDeltaPacket(tickID uint32, deltas [][3]byte) *DeltaPacket {
	return &DeltaPacket{
		version: 1,
		code:    7,
		tickID:  tickID,
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
	binary.BigEndian.PutUint32(TickIDBytes, dp.tickID)
	buf.Write(TickIDBytes)
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

	case 1: // RespPacket
		return &RespPacket{
			version: version,
			code:    code,
		}, nil

  case 2:
    return &RoomRequestPacket{
      version: version,
      code: code,
      RoomType: int(data[2]),
    }, nil

  case 3:
    return &LookRoomPacket{
      version: version,
      code: code,
      Success: int(data[2]),
    }, nil

  case 4:
    return &GameStartPacket{
      version: version,
      code: code,
      Success: int(data[2]),
    }, nil

	case 5: // ActionPacket
		action := int(data[2])
		return &ActionPacket{
			version: version,
			code:    code,
			action:  action,
		}, nil

	case 6: // BoardPacket
		// All remaining bytes are the encoded board
		encodedBoard := data[2:]
		return &BoardPacket{
			version:      version,
			code:         code,
			EncodedBoard: encodedBoard,
		}, nil

	case 7: // DeltasPacket
		if len(data) < 6 {
			return nil, errors.New("invalid deltas packet length")
		}
		tickID := binary.BigEndian.Uint32(data[2:6])
		deltaCount := int(binary.BigEndian.Uint16(data[6:8]))
		expectedLength := 8 + deltaCount*3
		if len(data) < expectedLength {
			return nil, errors.New("invalid deltas packet length")
		}
		deltas := make([][3]byte, deltaCount)
		for i := 0; i < deltaCount; i++ {
			start := 8 + i*3
			end := start + 3
			copy(deltas[i][:], data[start:end])
		}

		return &DeltaPacket{
			version: version,
			code:    code,
			tickID:  tickID,
			Deltas:  deltas,
		}, nil

	default:
		return nil, errors.New("unknown message type")
	}
}
