package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/GrGLeo/ctf/server/event"
)

/*
code 0: register user
code 1: register user response
code 2: sending username
code 3: return challenge
code 4: send username + signed_challenge
code 5: return success and message
code 6: send a find room
code 7: send a create room
code 8: send a join room
code 9: looking for a room response
code 10: game start  response
code 11: send action
code 12: receive RLEboard
code 13: receive Delta
code 14: game close
code 15: game end
code 16: spell selection
code 17: shop request
code 18: shop response
code 19: purchase item
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
AUTHENTIFICATION PACKET
*/
type RegisterRequestPacket struct {
  version, code int
  Username string
  PublicKey []byte
}

func NewRegisterRequestPacket(username string, publicKey []byte) *RegisterRequestPacket {
  return &RegisterRequestPacket{
    version: 1,
    code: 0,
    Username: username,
    PublicKey: publicKey,
  }
}

func (rrp RegisterRequestPacket) Version() int {
  return rrp.version
}

func (rrp RegisterRequestPacket) Code() int {
  return rrp.code
}

func (rrp RegisterRequestPacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(rrp.version))
  buf.WriteByte(byte(rrp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.Username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(rrp.Username); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.PublicKey))); err != nil {
		return nil
	}
	if _, err := buf.Write(rrp.PublicKey); err != nil {
		return nil
	}
	return buf.Bytes()
}

type RegisterResponsePacket struct {
  version, code int
  success bool
  message string
  challenge []byte
}

func NewRegisterResponsePacket(success bool, message string, challenge []byte) *RegisterResponsePacket {
  return &RegisterResponsePacket{
    version: 1,
    code: 1,
    success: success,
    message: message,
    challenge: challenge,
  }
}

func (rrp RegisterResponsePacket) Version() int {
  return rrp.version
}

func (rrp RegisterResponsePacket) Code() int {
  return rrp.code
}

func (rrp RegisterResponsePacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(rrp.version))
  buf.WriteByte(byte(rrp.code))
  if rrp.success {
    buf.WriteByte(1)
  } else {
    buf.WriteByte(0)
  }
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.message))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(rrp.message); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.challenge))); err != nil {
		return nil
	}
  if _, err := buf.Write(rrp.challenge); err != nil {
    return nil
  }
  return buf.Bytes()
}

type LoginChallengeRequestPacket struct {
  version, code int
  username string
}

func NewLoginChallengeRequestPacket(username string) *LoginChallengeRequestPacket {
  return &LoginChallengeRequestPacket{
    version: 1,
    code: 2,
    username: username,
  }
}

func (lcrp LoginChallengeRequestPacket) Version() int {
  return lcrp.version
}

func (lcrp LoginChallengeRequestPacket) Code() int {
  return lcrp.code
}

func (lcrp LoginChallengeRequestPacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(lcrp.version))
  buf.WriteByte(byte(lcrp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lcrp.username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(lcrp.username); err != nil {
		return nil
	}
  return buf.Bytes()
}

type LoginChallengeResponsePacket struct {
  version, code int
  challenge []byte
}

func NewLoginChallengeResponsePacket(challenge []byte) *LoginChallengeResponsePacket {
  return &LoginChallengeResponsePacket{
    version: 1,
    code: 3,
    challenge: challenge,
  }
}

func (lcrp LoginChallengeResponsePacket) Version() int {
  return lcrp.version
}

func (lcrp LoginChallengeResponsePacket) Code() int {
  return lcrp.code
}

func (lcrp LoginChallengeResponsePacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(lcrp.version))
  buf.WriteByte(byte(lcrp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lcrp.challenge))); err != nil {
		return nil
	}
	if _, err := buf.Write(lcrp.challenge); err != nil {
		return nil
	}
  return buf.Bytes()
}

type AuthRequestPacket struct {
  version, code int
  username string
  signedChallenge []byte
}

func NewAuthRequestPacket(username string, signedChallenge []byte) *AuthRequestPacket {
  return &AuthRequestPacket{
    version: 1,
    code: 4,
    username: username,
    signedChallenge: signedChallenge,
  }
}

func (arp AuthRequestPacket) Version() int {
  return arp.version
}

func (arp AuthRequestPacket) Code() int {
  return arp.code
}

func (arp AuthRequestPacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(arp.version))
  buf.WriteByte(byte(arp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(arp.username); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.signedChallenge))); err != nil {
		return nil
	}
  if _, err := buf.Write(arp.signedChallenge); err != nil {
    return nil
  }
  return buf.Bytes()
}

type AuthResponsePacket struct {
  version, code int
  success bool
}

func NewAuthResponsePacket(success bool) *AuthResponsePacket {
  return &AuthResponsePacket{
    version: 1,
    code: 5,
    success: success,
  }
}

func (arp AuthResponsePacket) Version() int {
  return arp.version
}

func (arp AuthResponsePacket) Code() int {
  return arp.code
}

func (arp AuthResponsePacket) Serialize() []byte {
  var buf bytes.Buffer
  buf.WriteByte(byte(arp.version))
  buf.WriteByte(byte(arp.code))
  if arp.success {
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
		code:     6,
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
		code:     7,
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
		code:    8,
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
		code:    9,
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
		code:    10,
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
		code:    14,
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
		code:    15,
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

type SpellSelectionPacket struct {
	version, code  int
	Spell1, Spell2 int
}

func NewSpellSelectionPacket(spell1, spell2 int) *SpellSelectionPacket {
	return &SpellSelectionPacket{
		version: 1,
		code:    16,
		Spell1:  spell1,
		Spell2:  spell2,
	}
}

func (ssp SpellSelectionPacket) Version() int {
	return ssp.version
}

func (ssp SpellSelectionPacket) Code() int {
	return ssp.code
}

func (ssp *SpellSelectionPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(ssp.version))
	buf.WriteByte(byte(ssp.code))
	buf.WriteByte(byte(ssp.Spell1))
	buf.WriteByte(byte(ssp.Spell2))
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
		code:    11,
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

type ShopRequestPacket struct {
	version, code int
}

func NewShopRequestPacket() *ShopRequestPacket {
	return &ShopRequestPacket{
		version: 1,
		code:    17,
	}
}

func (srp ShopRequestPacket) Version() int {
	return srp.version
}

func (srp ShopRequestPacket) Code() int {
	return srp.code
}

func (srp ShopRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(srp.version))
	buf.WriteByte(byte(srp.code))
	return buf.Bytes()
}

type ShopResponsePacket struct {
	version, code int
	Health        int
	Mana          int
	Attack_damage int
	Armor         int
	Gold          int
	Inventory     []int
}

func NewShopResponsePacket(health, mana, attack_damage, armor, gold int, inventory []int) *ShopResponsePacket {
	return &ShopResponsePacket{
		version:       1,
		code:          18,
		Health:        health,
		Mana:          mana,
		Attack_damage: attack_damage,
		Armor:         armor,
		Gold:          gold,
		Inventory:     inventory,
	}
}

func (srp ShopResponsePacket) Version() int {
	return srp.version
}

func (srp ShopResponsePacket) Code() int {
	return srp.code
}

func (srp *ShopResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(srp.version))
	buf.WriteByte(byte(srp.code))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Health))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Mana))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Attack_damage))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Armor))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Gold))
	// Always write 6 inventory slots
	for i := range 6 {
		if i < len(srp.Inventory) {
			binary.Write(&buf, binary.BigEndian, uint16(srp.Inventory[i]))
		} else {
			binary.Write(&buf, binary.BigEndian, uint16(0)) // Empty slot
		}
	}
	return buf.Bytes()
}

type PurchaseItemPacket struct {
	version, code, ItemID int
}

func NewPurchaseItemPacket(itemID int) *PurchaseItemPacket {
	return &PurchaseItemPacket{
		version: 1,
		code:    19,
		ItemID:  itemID,
	}
}

func (pip PurchaseItemPacket) Version() int {
	return pip.version
}

func (pip PurchaseItemPacket) Code() int {
	return pip.code
}

func (pip PurchaseItemPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(pip.version))
	buf.WriteByte(byte(pip.code))
	binary.Write(&buf, binary.BigEndian, uint16(pip.ItemID))
	return buf.Bytes()
}

type BoardPacket struct {
	version, code int
	CastTime      int
	CastDuration  int
	Health        int
	MaxHealth     int
	Mana          int
	MaxMana       int
	Level         int
	Xp            int
	XpNeeded      int
	Length        int
	EncodedBoard  []byte
}

func NewBoardPacket(castTime, castDuration, health, maxHealth, mana, maxMana, level, xp, xpNeeded int, encodedBoard []byte) *BoardPacket {
	return &BoardPacket{
		version:      1,
		code:         12,
		CastTime:     castTime,
		CastDuration: castDuration,
		Health:       health,
		MaxHealth:    maxHealth,
		Mana:         mana,
		MaxMana:      maxMana,
		Level:        level,
		Xp:           xp,
		XpNeeded:     xpNeeded,
		Length:       len(encodedBoard),
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
	binary.Write(&buf, binary.BigEndian, uint16(bp.CastTime))
	binary.Write(&buf, binary.BigEndian, uint16(bp.CastDuration))
	binary.Write(&buf, binary.BigEndian, uint16(bp.Health))
	binary.Write(&buf, binary.BigEndian, uint16(bp.MaxHealth))
	binary.Write(&buf, binary.BigEndian, uint16(bp.Mana))
	binary.Write(&buf, binary.BigEndian, uint16(bp.MaxMana))
	buf.WriteByte(byte(bp.Level))
	binary.Write(&buf, binary.BigEndian, uint16(bp.Xp))
	binary.Write(&buf, binary.BigEndian, uint16(bp.XpNeeded))
	binary.Write(&buf, binary.BigEndian, uint16(len(bp.EncodedBoard)))
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
		code:    13,
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

// DeSerialize deserializes a byte slice into a specific Packet type.
// It determines the packet type based on the message code and returns the
// parsed packet, the number of bytes consumed, and an error if the packet
// is malformed or the data is incomplete.
//
// Parameters:
// - data: A byte slice containing the serialized packet data.
//
// Returns:
//   - Packet: The deserialized Packet interface.
//   - int: The number of bytes consumed from the data slice to form the packet.
//   - error: An error if the data is malformed, the version is invalid, or if the
//     data slice does not contain a complete packet.
func DeSerialize(data []byte) (Packet, int, error) {
	// Check minimum packet length (version + code)
	if len(data) < 2 {
		return nil, 0, errors.New("incomplete packet header")
	}

	version := int(data[0])
	if version != 1 {
		return nil, 0, errors.New("invalid version")
	}

	code := int(data[1])

	// Parse based on message type
	switch code {
	case 0: // LoginPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(binary.BigEndian.Uint16(data[2:4]))
		pwStart := 4 + usernameLen
		if len(data) < pwStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		passwordLen := int(binary.BigEndian.Uint16(data[pwStart : pwStart+2]))
		totalLen := pwStart + 2 + passwordLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[4:pwStart])
		password := string(data[pwStart+2 : totalLen])
		packet := &LoginPacket{
			version:  version,
			code:     code,
			Username: username,
			Password: password,
		}
		return packet, totalLen, nil

	case 1: // SignInPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(binary.BigEndian.Uint16(data[2:4]))
		pwStart := 4 + usernameLen
		if len(data) < pwStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		passwordLen := int(binary.BigEndian.Uint16(data[pwStart : pwStart+2]))
		totalLen := pwStart + 2 + passwordLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[4:pwStart])
		password := string(data[pwStart+2 : totalLen])
		packet := &SignInPacket{
			version:  version,
			code:     code,
			Username: username,
			Password: password,
		}
		return packet, totalLen, nil

	case 2: // RespPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RespPacket{
			version: version,
			code:    code,
			Success: data[2] == 1,
		}
		return packet, 3, nil

	case 3: // RoomRequestPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RoomRequestPacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}
		return packet, 3, nil

	case 4: // RoomCreatePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RoomCreatePacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}
		return packet, 3, nil

	case 5: // RoomJoinPacket
		// This packet has a variable length RoomID, assuming it's the rest of the packet
		roomID := string(data[2:])
		packet := &RoomJoinPacket{
			version: version,
			code:    code,
			RoomID:  roomID,
		}
		return packet, len(data), nil

	case 6: // LookRoomPacket
		// This packet has a variable length RoomIP
		if len(data) < 8 { // 3 bytes header + 5 bytes RoomID
			return nil, 0, errors.New("incomplete packet")
		}
		roomID := string(data[3:8])
		roomIP := string(data[8:])
		packet := &LookRoomPacket{
			version: version,
			code:    code,
			Success: int(data[2]),
			RoomID:  roomID,
			RoomIP:  roomIP,
		}
		return packet, len(data), nil

	case 7: // GameStartPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &GameStartPacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}
		return packet, 3, nil

	case 8: // ActionPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &ActionPacket{
			version: version,
			code:    code,
			action:  int(data[2]),
		}
		return packet, 3, nil

	case 9: // BoardPacket
		if len(data) < 21 { // header size
			return nil, 0, errors.New("incomplete packet")
		}
		length := int(binary.BigEndian.Uint16(data[19:21]))
		totalLen := 21 + length
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		castTime := int(binary.BigEndian.Uint16(data[2:4]))
		castDuration := int(binary.BigEndian.Uint16(data[4:6]))
		health := int(binary.BigEndian.Uint16(data[6:8]))
		maxHealth := int(binary.BigEndian.Uint16(data[8:10]))
		mana := int(binary.BigEndian.Uint16(data[10:12]))
		maxMana := int(binary.BigEndian.Uint16(data[12:14]))
		level := int(data[14])
		xp := int(binary.BigEndian.Uint16(data[15:17]))
		xpNeeded := int(binary.BigEndian.Uint16(data[17:19]))
		encodedBoard := data[21:totalLen]
		packet := &BoardPacket{
			version:      version,
			code:         code,
			CastTime:     castTime,
			CastDuration: castDuration,
			Health:       health,
			MaxHealth:    maxHealth,
			Mana:         mana,
			MaxMana:      maxMana,
			Level:        level,
			Xp:           xp,
			XpNeeded:     xpNeeded,
			Length:       length,
			EncodedBoard: encodedBoard,
		}
		return packet, totalLen, nil

	case 10: // DeltaPacket
		if len(data) < 10 {
			return nil, 0, errors.New("incomplete packet")
		}
		deltaCount := int(binary.BigEndian.Uint16(data[8:10]))
		totalLen := 10 + deltaCount*3
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		tickID := binary.BigEndian.Uint32(data[2:6])
		points := [2]int{int(data[6]), int(data[7])}
		deltas := make([][3]byte, deltaCount)
		for i := range deltaCount {
			start := 10 + i*3
			copy(deltas[i][:], data[start:start+3])
		}
		packet := &DeltaPacket{
			version: version,
			code:    code,
			TickID:  tickID,
			Points:  points,
			Deltas:  deltas,
		}
		return packet, totalLen, nil

	case 11: // GameClosePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &GameClosePacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}
		return packet, 3, nil

	case 12: // EndGamePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &EndGamePacket{
			version: version,
			code:    code,
			Win:     data[2] == 1,
		}
		return packet, 3, nil

	case 13: // SpellSelectionPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &SpellSelectionPacket{
			version: version,
			code:    code,
			Spell1:  int(data[2]),
			Spell2:  int(data[3]),
		}
		return packet, 4, nil

	case 15: // ShopResponsePacket
		if len(data) < 24 {
			return nil, 0, errors.New("incomplete packet")
		}
		health := int(binary.BigEndian.Uint16(data[2:4]))
		mana := int(binary.BigEndian.Uint16(data[4:6]))
		attack_damage := int(binary.BigEndian.Uint16(data[6:8]))
		armor := int(binary.BigEndian.Uint16(data[8:10]))
		gold := int(binary.BigEndian.Uint16(data[10:12]))
		var inventory []int
		for i := range 6 {
			start := 12 + i*2
			inventory = append(inventory, int(binary.BigEndian.Uint16(data[start:start+2])))
		}
		packet := &ShopResponsePacket{
			version:       version,
			code:          code,
			Health:        health,
			Mana:          mana,
			Attack_damage: attack_damage,
			Armor:         armor,
			Gold:          gold,
			Inventory:     inventory,
		}
		return packet, 24, nil

	case 16: // PurchaseItemPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &PurchaseItemPacket{
			version: version,
			code:    code,
			ItemID:  int(binary.BigEndian.Uint16(data[2:4])),
		}
		return packet, 4, nil

	default:
		return nil, 0, errors.New("unknown message type")
	}
}
