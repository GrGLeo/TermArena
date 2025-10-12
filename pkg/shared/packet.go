package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
)

/*
Packet Code Organization (0-255)

Authentication packets (0-49):
  0: register user request (username + pubkey)
  1: register user response (success, message, challenge)
  2: login challenge request (username)
  3: login challenge response (challenge)
  4: auth request (username + signed challenge)
  5: auth response (success, message, session_token)

Room management packets (50-99):
  50: send a find room
  51: send a create room
  52: send a join room
  53: looking for a room response
  54: move to lobby room with users packet
  55: update spell request
  56: update spell response
  57: game server is ready
	58: client quit room

Game packets (100-149):
  100: game start response
  101: send action
  102: receive RLEboard
  103: receive Delta
  104: game close
  105: game end
  106: spell selection
  107: shop request
  108: shop response
  109: purchase item

Message packets (150-199):
  150: message packet (sender + message content)
  151: message response (routed message content)
  152: message error (error description)

Special packets (250-255):
  255: error message for rate limit exceed
*/

type Packet interface {
	Version() int
	Code() int
	Serialize() []byte
}

/*
AUTHENTIFICATION PACKETS
*/

type RegisterRequestPacket struct {
	version, code int
	Username      string
	PublicKey     []byte
}

func NewRegisterRequestPacket(username string, publicKey []byte) *RegisterRequestPacket {
	return &RegisterRequestPacket{
		version:   1,
		code:      CodeRegisterRequest,
		Username:  username,
		PublicKey: publicKey,
	}
}

func (rrp *RegisterRequestPacket) Version() int { return rrp.version }
func (rrp *RegisterRequestPacket) Code() int    { return rrp.code }
func (rrp *RegisterRequestPacket) Serialize() []byte {
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
	Success       bool
	Message       string
	Challenge     []byte
}

func NewRegisterResponsePacket(success bool, message string, challenge []byte) *RegisterResponsePacket {
	return &RegisterResponsePacket{
		version:   1,
		code:      CodeRegisterResponse,
		Success:   success,
		Message:   message,
		Challenge: challenge,
	}
}

func (rrp *RegisterResponsePacket) Version() int { return rrp.version }
func (rrp *RegisterResponsePacket) Code() int    { return rrp.code }
func (rrp *RegisterResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(rrp.version))
	buf.WriteByte(byte(rrp.code))
	if rrp.Success {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.Message))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(rrp.Message); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(rrp.Challenge))); err != nil {
		return nil
	}
	if _, err := buf.Write(rrp.Challenge); err != nil {
		return nil
	}
	return buf.Bytes()
}

type LoginChallengeRequestPacket struct {
	version, code int
	Username      string
}

func NewLoginChallengeRequestPacket(username string) *LoginChallengeRequestPacket {
	return &LoginChallengeRequestPacket{
		version:  1,
		code:     CodeLoginChallengeRequest,
		Username: username,
	}
}

func (lcrp *LoginChallengeRequestPacket) Version() int { return lcrp.version }
func (lcrp *LoginChallengeRequestPacket) Code() int    { return lcrp.code }
func (lcrp *LoginChallengeRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(lcrp.version))
	buf.WriteByte(byte(lcrp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lcrp.Username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(lcrp.Username); err != nil {
		return nil
	}
	return buf.Bytes()
}

type LoginChallengeResponsePacket struct {
	version, code int
	Challenge     []byte
}

func NewLoginChallengeResponsePacket(challenge []byte) *LoginChallengeResponsePacket {
	return &LoginChallengeResponsePacket{
		version:   1,
		code:      CodeLoginChallengeResponse,
		Challenge: challenge,
	}
}

func (lcrp *LoginChallengeResponsePacket) Version() int { return lcrp.version }
func (lcrp *LoginChallengeResponsePacket) Code() int    { return lcrp.code }
func (lcrp *LoginChallengeResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(lcrp.version))
	buf.WriteByte(byte(lcrp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(lcrp.Challenge))); err != nil {
		return nil
	}
	if _, err := buf.Write(lcrp.Challenge); err != nil {
		return nil
	}
	return buf.Bytes()
}

type AuthRequestPacket struct {
	version, code   int
	Username        string
	SignedChallenge []byte
}

func NewAuthRequestPacket(username string, signedChallenge []byte) *AuthRequestPacket {
	return &AuthRequestPacket{
		version:         1,
		code:            CodeAuthRequest,
		Username:        username,
		SignedChallenge: signedChallenge,
	}
}

func (arp *AuthRequestPacket) Version() int { return arp.version }
func (arp *AuthRequestPacket) Code() int    { return arp.code }
func (arp *AuthRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(arp.version))
	buf.WriteByte(byte(arp.code))
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.Username))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(arp.Username); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.SignedChallenge))); err != nil {
		return nil
	}
	if _, err := buf.Write(arp.SignedChallenge); err != nil {
		return nil
	}
	return buf.Bytes()
}

type AuthResponsePacket struct {
	version, code int
	Success       bool
	Message       string
	SessionToken  string
}

func NewAuthResponsePacket(success bool, message, sessionToken string) *AuthResponsePacket {
	return &AuthResponsePacket{
		version:      1,
		code:         CodeAuthResponse,
		Success:      success,
		Message:      message,
		SessionToken: sessionToken,
	}
}

func (arp *AuthResponsePacket) Version() int { return arp.version }
func (arp *AuthResponsePacket) Code() int    { return arp.code }
func (arp *AuthResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(arp.version))
	buf.WriteByte(byte(arp.code))
	if arp.Success {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.Message))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(arp.Message); err != nil {
		return nil
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(len(arp.SessionToken))); err != nil {
		return nil
	}
	if _, err := buf.WriteString(arp.SessionToken); err != nil {
		return nil
	}
	return buf.Bytes()
}

/*
ROOM & GAME PACKETS
*/
type RoomRequestPacket struct {
	version, code, RoomType int
}

func NewRoomRequestPacket(RoomType int) *RoomRequestPacket {
	return &RoomRequestPacket{
		version:  1,
		code:     CodeRoomRequest,
		RoomType: RoomType,
	}
}

func (fp *RoomRequestPacket) Version() int { return fp.version }
func (fp *RoomRequestPacket) Code() int    { return fp.code }
func (fp *RoomRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(fp.version))
	buf.WriteByte(byte(fp.code))
	buf.WriteByte(byte(fp.RoomType))
	return buf.Bytes()
}

type RoomCreatePacket struct {
	version, code, RoomType int
}

func NewRoomCreatePacket(RoomType int) *RoomCreatePacket {
	return &RoomCreatePacket{
		version:  1,
		code:     CodeRoomCreate,
		RoomType: RoomType,
	}
}

func (cp *RoomCreatePacket) Version() int { return cp.version }
func (cp *RoomCreatePacket) Code() int    { return cp.code }
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
		code:    CodeRoomJoin,
		RoomID:  roomID,
	}
}

func (cp *RoomJoinPacket) Version() int { return cp.version }
func (cp *RoomJoinPacket) Code() int    { return cp.code }
func (cp *RoomJoinPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(cp.version))
	buf.WriteByte(byte(cp.code))
	buf.WriteString(cp.RoomID)
	return buf.Bytes()
}

type LookRoomPacket struct {
	version, code int
	Success       bool
	RoomID        uint32
}

func NewLookRoomPacket(success bool, roomID uint32) *LookRoomPacket {
	return &LookRoomPacket{
		version: 1,
		code:    CodeLookRoom,
		Success: success,
		RoomID:  roomID,
	}
}

func (lp *LookRoomPacket) Version() int { return lp.version }
func (lp *LookRoomPacket) Code() int    { return lp.code }
func (lp *LookRoomPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(lp.version))
	buf.WriteByte(byte(lp.code))
	if lp.Success {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	binary.Write(&buf, binary.BigEndian, uint32(lp.RoomID))
	return buf.Bytes()
}

type MoveToLobbyPacket struct {
	version, code int
	RoomID        uint32
	UserInfos     []*pb.UserInfo
}

func NewMoveToLobbyPacket(roomID uint32, userInfos []*pb.UserInfo) *MoveToLobbyPacket {
	return &MoveToLobbyPacket{
		version:   1,
		code:      CodeMoveToLobby,
		RoomID:    roomID,
		UserInfos: userInfos,
	}
}

func (ml *MoveToLobbyPacket) Version() int { return ml.version }
func (ml *MoveToLobbyPacket) Code() int    { return ml.code }
func (ml *MoveToLobbyPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(ml.version))
	buf.WriteByte(byte(ml.code))
	binary.Write(&buf, binary.BigEndian, uint32(ml.RoomID))
	numUsers := uint32(len(ml.UserInfos))
	binary.Write(&buf, binary.BigEndian, numUsers)
	for _, ui := range ml.UserInfos {
		usernameLen := uint16(len(ui.Username))
		binary.Write(&buf, binary.BigEndian, usernameLen)
		buf.WriteString(ui.Username)
		binary.Write(&buf, binary.BigEndian, uint32(ui.Team))
		binary.Write(&buf, binary.BigEndian, uint32(ui.Spell1))
		binary.Write(&buf, binary.BigEndian, uint32(ui.Spell2))
	}
	return buf.Bytes()
}

type UpdateSpellReqPacket struct {
	version, code      int
	RoomType           int
	RoomID             int
	Username           string
	SpellOne, SpellTwo int
}

func NewUpdateSpellReqPacket(roomType int, roomID int, username string, spellOne, spellTwo int) *UpdateSpellReqPacket {
	return &UpdateSpellReqPacket{
		version:  1,
		code:     CodeUpdateSpellReq,
		RoomType: roomType,
		RoomID:   roomID,
		Username: username,
		SpellOne: spellOne,
		SpellTwo: spellTwo,
	}
}
func (us *UpdateSpellReqPacket) Version() int { return us.version }
func (us *UpdateSpellReqPacket) Code() int    { return us.code }
func (us *UpdateSpellReqPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(us.version))
	buf.WriteByte(byte(us.code))
	buf.WriteByte(byte(us.RoomType))
	binary.Write(&buf, binary.BigEndian, uint32(us.RoomID))
	binary.Write(&buf, binary.BigEndian, uint16(len(us.Username)))
	buf.WriteString(us.Username)
	buf.WriteByte(byte(us.SpellOne))
	buf.WriteByte(byte(us.SpellTwo))
	return buf.Bytes()
}

type UpdateSpellResPacket struct {
	version, code      int
	Username           string
	SpellOne, SpellTwo int
}

func NewUpdateSpellResPacket(username string, spellOne, spellTwo int) *UpdateSpellResPacket {
	return &UpdateSpellResPacket{
		version:  1,
		code:     CodeUpdateSpellRes,
		Username: username,
		SpellOne: spellOne,
		SpellTwo: spellTwo,
	}
}

func (us *UpdateSpellResPacket) Version() int { return us.version }
func (us *UpdateSpellResPacket) Code() int    { return us.code }
func (us *UpdateSpellResPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(us.version))
	buf.WriteByte(byte(us.code))
	binary.Write(&buf, binary.BigEndian, uint16(len(us.Username)))
	buf.WriteString(us.Username)
	buf.WriteByte(byte(us.SpellOne))
	buf.WriteByte(byte(us.SpellTwo))
	return buf.Bytes()
}

type GameServerReadyPacket struct {
	version, code int
	RoomIP        uint16
}

func NewGameServerReadyPacket(roomIP uint16) *GameServerReadyPacket {
	return &GameServerReadyPacket{
		version: 1,
		code:    CodeGameServerReady,
		RoomIP:  roomIP,
	}
}
func (gsp *GameServerReadyPacket) Version() int { return gsp.version }
func (gsp *GameServerReadyPacket) Code() int    { return gsp.code }
func (gsp *GameServerReadyPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(gsp.version))
	buf.WriteByte(byte(gsp.code))
	binary.Write(&buf, binary.BigEndian, gsp.RoomIP)
	return buf.Bytes()
}

type QuitRoomPacket struct {
	version, code int
}

func NewQuitRoomPacket() *QuitRoomPacket {
	return &QuitRoomPacket{
		version: 1,
		code:    CodeQuitRoom,
	}
}
func (qrp *QuitRoomPacket) Version() int { return qrp.version }
func (qrp *QuitRoomPacket) Code() int    { return qrp.code }
func (qrp *QuitRoomPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(qrp.version))
	buf.WriteByte(byte(qrp.code))
	return buf.Bytes()
}

type GameStartPacket struct {
	version, code, Success int
}

func NewGameStartPacket(success int) *GameStartPacket {
	return &GameStartPacket{
		version: 1,
		code:    CodeGameStart,
		Success: success,
	}
}

func (gp *GameStartPacket) Version() int { return gp.version }
func (gp *GameStartPacket) Code() int    { return gp.code }
func (gp *GameStartPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(gp.version))
	buf.WriteByte(byte(gp.code))
	buf.WriteByte(byte(gp.Success))
	return buf.Bytes()
}

type ActionPacket struct {
	version, code int
	action        int
}

func NewActionPacket(action int) *ActionPacket {
	return &ActionPacket{
		version: 1,
		code:    CodeAction,
		action:  action,
	}
}

func (ap *ActionPacket) Version() int { return ap.version }
func (ap *ActionPacket) Code() int    { return ap.code }
func (ap *ActionPacket) Action() int  { return ap.action }
func (ap *ActionPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(ap.version))
	buf.WriteByte(byte(ap.code))
	buf.WriteByte(byte(ap.action))
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
		code:         CodeBoard,
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

func (bp *BoardPacket) Version() int { return bp.version }
func (bp *BoardPacket) Code() int    { return bp.code }
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
		code:    CodeDelta,
		Points:  points,
		TickID:  tickID,
		Deltas:  deltas,
	}
}

func (dp *DeltaPacket) Version() int { return dp.version }
func (dp *DeltaPacket) Code() int    { return dp.code }
func (dp *DeltaPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(dp.version))
	buf.WriteByte(byte(dp.code))
	TickIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(TickIDBytes, dp.TickID)
	buf.Write(TickIDBytes)
	buf.WriteByte(byte(dp.Points[0]))
	buf.WriteByte(byte(dp.Points[1]))
	deltaCount := len(dp.Deltas)
	deltaCountBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(deltaCountBytes, uint16(deltaCount))
	buf.Write(deltaCountBytes)
	for _, delta := range dp.Deltas {
		buf.Write(delta[:])
	}
	return buf.Bytes()
}

type GameClosePacket struct {
	version, code, Success int
}

func NewGameClosePacket(success int) *GameClosePacket {
	return &GameClosePacket{
		version: 1,
		code:    CodeGameClose,
		Success: success,
	}
}

func (gc *GameClosePacket) Version() int { return gc.version }
func (gc *GameClosePacket) Code() int    { return gc.code }
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
		code:    CodeEndGame,
		Win:     win,
	}
}

func (egp *EndGamePacket) Version() int { return egp.version }
func (egp *EndGamePacket) Code() int    { return egp.code }
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

type UsernamePacket struct {
	version, code int
	Username      string
}

func NewUsernamePacket(username string) *UsernamePacket {
	return &UsernamePacket{
		version:  1,
		code:     CodeSpellSelection,
		Username: username,
	}
}

func (ssp *UsernamePacket) Version() int { return ssp.version }
func (ssp *UsernamePacket) Code() int    { return ssp.code }
func (ssp *UsernamePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(ssp.version))
	buf.WriteByte(byte(ssp.code))
	usernameBytes := []byte(ssp.Username)
	buf.WriteByte(byte(len(usernameBytes)))
	buf.Write(usernameBytes)
	return buf.Bytes()
}

type ShopRequestPacket struct {
	version, code int
}

func NewShopRequestPacket() *ShopRequestPacket {
	return &ShopRequestPacket{
		version: 1,
		code:    CodeShopRequest,
	}
}

func (srp *ShopRequestPacket) Version() int { return srp.version }
func (srp *ShopRequestPacket) Code() int    { return srp.code }
func (srp *ShopRequestPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(srp.version))
	buf.WriteByte(byte(srp.code))
	return buf.Bytes()
}

type ShopResponsePacket struct {
	version, code int
	Health        int
	Mana          int
	AttackDamage  int
	MagicPower    int
	Armor         int
	Gold          int
	Inventory     []int
}

func NewShopResponsePacket(health, mana, attackDamage, magicPower, armor, gold int, inventory []int) *ShopResponsePacket {
	return &ShopResponsePacket{
		version:      1,
		code:         CodeShopResponse,
		Health:       health,
		Mana:         mana,
		AttackDamage: attackDamage,
		MagicPower:   magicPower,
		Armor:        armor,
		Gold:         gold,
		Inventory:    inventory,
	}
}

func (srp *ShopResponsePacket) Version() int { return srp.version }
func (srp *ShopResponsePacket) Code() int    { return srp.code }
func (srp *ShopResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(srp.version))
	buf.WriteByte(byte(srp.code))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Health))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Mana))
	binary.Write(&buf, binary.BigEndian, uint16(srp.AttackDamage))
	binary.Write(&buf, binary.BigEndian, uint16(srp.MagicPower))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Armor))
	binary.Write(&buf, binary.BigEndian, uint16(srp.Gold))
	for i := range 6 {
		if i < len(srp.Inventory) {
			binary.Write(&buf, binary.BigEndian, uint16(srp.Inventory[i]))
		} else {
			binary.Write(&buf, binary.BigEndian, uint16(0))
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
		code:    CodePurchaseItem,
		ItemID:  itemID,
	}
}

func (pip *PurchaseItemPacket) Version() int { return pip.version }
func (pip *PurchaseItemPacket) Code() int    { return pip.code }
func (pip *PurchaseItemPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(pip.version))
	buf.WriteByte(byte(pip.code))
	binary.Write(&buf, binary.BigEndian, uint16(pip.ItemID))
	return buf.Bytes()
}

type MessagePacket struct {
	version, code   int
	Sender, Message string
}

func NewMessagePacket(sender, message string) *MessagePacket {
	return &MessagePacket{
		version: 1,
		code:    CodeMessage,
		Sender:  sender,
		Message: message,
	}
}

func (mp *MessagePacket) Version() int { return mp.version }
func (mp *MessagePacket) Code() int    { return mp.code }
func (mp *MessagePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(mp.version))
	buf.WriteByte(byte(mp.code))
	binary.Write(&buf, binary.BigEndian, uint16(len(mp.Sender)))
	buf.WriteString(mp.Sender)
	binary.Write(&buf, binary.BigEndian, uint16(len(mp.Message)))
	buf.WriteString(mp.Message)
	return buf.Bytes()
}

type MessageResponsePacket struct {
	version, code int
	Message       string
}

func NewMessageResponsePacket(message string) *MessageResponsePacket {
	return &MessageResponsePacket{
		version: 1,
		code:    CodeMessageResponse,
		Message: message,
	}
}

func (mrp *MessageResponsePacket) Version() int { return mrp.version }
func (mrp *MessageResponsePacket) Code() int    { return mrp.code }
func (mrp *MessageResponsePacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(mrp.version))
	buf.WriteByte(byte(mrp.code))
	binary.Write(&buf, binary.BigEndian, uint16(len(mrp.Message)))
	buf.WriteString(mrp.Message)
	return buf.Bytes()
}

type MessageErrorPacket struct {
	version, code int
	Error         string
}

func NewMessageErrorPacket(errorMsg string) *MessageErrorPacket {
	return &MessageErrorPacket{
		version: 1,
		code:    CodeMessageError,
		Error:   errorMsg,
	}
}

func (mep *MessageErrorPacket) Version() int { return mep.version }
func (mep *MessageErrorPacket) Code() int    { return mep.code }
func (mep *MessageErrorPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(mep.version))
	buf.WriteByte(byte(mep.code))
	binary.Write(&buf, binary.BigEndian, uint16(len(mep.Error)))
	buf.WriteString(mep.Error)
	return buf.Bytes()
}

type RateLimitPacket struct {
	version, code int
}

func NewRateLimitPacket() *RateLimitPacket {
	return &RateLimitPacket{
		version: 1,
		code:    CodeRateLimit,
	}
}

func (rlp *RateLimitPacket) Version() int { return rlp.version }
func (rlp *RateLimitPacket) Code() int    { return rlp.code }
func (rlp *RateLimitPacket) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(rlp.version))
	buf.WriteByte(byte(rlp.code))
	return buf.Bytes()
}

// DeSerialize deserializes a byte slice into a specific Packet type.
func DeSerialize(data []byte) (Packet, int, error) {
	if len(data) < 2 {
		return nil, 0, errors.New("incomplete packet header")
	}
	version := int(data[0])
	if version != 1 {
		return nil, 0, errors.New("invalid version")
	}
	code := int(data[1])

	switch code {
	case CodeRegisterRequest: // RegisterRequestPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(binary.BigEndian.Uint16(data[2:4]))
		publicKeyLenStart := 4 + usernameLen
		if len(data) < publicKeyLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		publicKeyLen := int(binary.BigEndian.Uint16(data[publicKeyLenStart : publicKeyLenStart+2]))
		totalLen := publicKeyLenStart + 2 + publicKeyLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[4:publicKeyLenStart])
		publicKey := data[publicKeyLenStart+2 : totalLen]
		packet := &RegisterRequestPacket{
			version:   version,
			code:      code,
			Username:  username,
			PublicKey: publicKey,
		}
		return packet, totalLen, nil

	case CodeRegisterResponse: // RegisterResponsePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		success := data[2] == 1
		messageLenStart := 3
		if len(data) < messageLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		messageLen := int(binary.BigEndian.Uint16(data[messageLenStart : messageLenStart+2]))
		challengeLenStart := messageLenStart + 2 + messageLen
		if len(data) < challengeLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		challengeLen := int(binary.BigEndian.Uint16(data[challengeLenStart : challengeLenStart+2]))
		totalLen := challengeLenStart + 2 + challengeLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		message := string(data[messageLenStart+2 : challengeLenStart])
		challenge := data[challengeLenStart+2 : totalLen]
		packet := &RegisterResponsePacket{
			version:   version,
			code:      code,
			Success:   success,
			Message:   message,
			Challenge: challenge,
		}
		return packet, totalLen, nil

	case CodeLoginChallengeRequest: // LoginChallengeRequestPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(binary.BigEndian.Uint16(data[2:4]))
		totalLen := 4 + usernameLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[4:totalLen])
		packet := &LoginChallengeRequestPacket{
			version:  version,
			code:     code,
			Username: username,
		}
		return packet, totalLen, nil

	case CodeLoginChallengeResponse: // LoginChallengeResponsePacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		challengeLen := int(binary.BigEndian.Uint16(data[2:4]))
		totalLen := 4 + challengeLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		challenge := data[4:totalLen]
		packet := &LoginChallengeResponsePacket{
			version:   version,
			code:      code,
			Challenge: challenge,
		}
		return packet, totalLen, nil

	case CodeAuthRequest: // AuthRequestPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(binary.BigEndian.Uint16(data[2:4]))
		challengeLenStart := 4 + usernameLen
		if len(data) < challengeLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		challengeLen := int(binary.BigEndian.Uint16(data[challengeLenStart : challengeLenStart+2]))
		totalLen := challengeLenStart + 2 + challengeLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[4:challengeLenStart])
		signedChallenge := data[challengeLenStart+2 : totalLen]
		packet := &AuthRequestPacket{
			version:         version,
			code:            code,
			Username:        username,
			SignedChallenge: signedChallenge,
		}
		return packet, totalLen, nil

	case CodeAuthResponse: // AuthResponsePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		success := data[2] == 1
		messageLenStart := 3
		if len(data) < messageLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		messageLen := int(binary.BigEndian.Uint16(data[messageLenStart : messageLenStart+2]))
		tokenLenStart := messageLenStart + 2 + messageLen
		if len(data) < tokenLenStart+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		tokenLen := int(binary.BigEndian.Uint16(data[tokenLenStart : tokenLenStart+2]))
		totalLen := tokenLenStart + 2 + tokenLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		message := string(data[messageLenStart+2 : tokenLenStart])
		sessionToken := string(data[tokenLenStart+2 : totalLen])
		packet := &AuthResponsePacket{
			version:      version,
			code:         code,
			Success:      success,
			Message:      message,
			SessionToken: sessionToken,
		}
		return packet, totalLen, nil

	case CodeRoomRequest: // RoomRequestPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RoomRequestPacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}
		return packet, 3, nil

	case CodeRoomCreate: // RoomCreatePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RoomCreatePacket{
			version:  version,
			code:     code,
			RoomType: int(data[2]),
		}
		return packet, 3, nil

	case CodeRoomJoin: // RoomJoinPacket
		roomID := string(data[2:])
		packet := &RoomJoinPacket{
			version: version,
			code:    code,
			RoomID:  roomID,
		}
		return packet, len(data), nil

	case CodeLookRoom: // LookRoomPacket
		if len(data) < 7 {
			return nil, 0, errors.New("incomplete packet")
		}
		success := data[2] == 1
		roomID := binary.BigEndian.Uint32(data[3:7])
		packet := &LookRoomPacket{
			version: version,
			code:    code,
			Success: success,
			RoomID:  roomID,
		}
		return packet, len(data), nil

	case CodeMoveToLobby: // MoveToLobbyPacket
		if len(data) < 10 { // version, code, RoomID uint32, numUsers uint32
			return nil, 0, errors.New("incomplete packet")
		}
		roomID := binary.BigEndian.Uint32(data[2:6])
		numUsers := binary.BigEndian.Uint32(data[6:10])
		offset := 10
		var userInfos []*pb.UserInfo
		for i := uint32(0); i < numUsers; i++ {
			if len(data) < offset+2 {
				return nil, 0, errors.New("incomplete packet")
			}
			usernameLen := binary.BigEndian.Uint16(data[offset : offset+2])
			offset += 2
			if len(data) < offset+int(usernameLen) {
				return nil, 0, errors.New("incomplete packet")
			}
			username := string(data[offset : offset+int(usernameLen)])
			offset += int(usernameLen)
			if len(data) < offset+12 { // team, spell1, spell2 uint32
				return nil, 0, errors.New("incomplete packet")
			}
			team := binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
			spell1 := binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
			spell2 := binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
			userInfo := &pb.UserInfo{
				Username: username,
				Team:     team,
				Spell1:   spell1,
				Spell2:   spell2,
			}
			userInfos = append(userInfos, userInfo)
		}
		packet := &MoveToLobbyPacket{
			version:   version,
			code:      code,
			RoomID:    roomID,
			UserInfos: userInfos,
		}
		return packet, offset, nil

	case CodeUpdateSpellReq: // UpdateSpellReqPacket
		if len(data) < 11 {
			return nil, 0, errors.New("incomplete packet")
		}
		roomType := int(data[2])
		roomID := binary.BigEndian.Uint32(data[3:7])
		usernameLen := binary.BigEndian.Uint16(data[7:9])
		offset := 9 + int(usernameLen)
		if len(data) < offset+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[9:offset])
		spellOne := int(data[offset])
		spellTwo := int(data[offset+1])
		totalLen := offset + 2
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &UpdateSpellReqPacket{
			version:  version,
			code:     code,
			RoomType: roomType,
			RoomID:   int(roomID),
			Username: username,
			SpellOne: spellOne,
			SpellTwo: spellTwo,
		}
		return packet, totalLen, nil

	case CodeUpdateSpellRes: // UpdateSpellResPacket
		if len(data) < 6 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := binary.BigEndian.Uint16(data[2:4])
		offset := 4 + int(usernameLen)
		username := string(data[4:offset])
		spellOne := int(data[offset])
		spellTwo := int(data[offset+1])
		totalLen := offset + 2
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &UpdateSpellResPacket{
			version:  version,
			code:     code,
			Username: username,
			SpellOne: spellOne,
			SpellTwo: spellTwo,
		}
		return packet, totalLen, nil

	case CodeGameServerReady:
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		roomID := binary.BigEndian.Uint16(data[2:4])
		packet := &GameServerReadyPacket{
			version: version,
			code:    code,
			RoomIP:  roomID,
		}
		return packet, 4, nil

	case CodeQuitRoom:
		if len(data) < 2 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &QuitRoomPacket{
			version: version,
			code:    code,
		}
		return packet, 2, nil

	case CodeGameStart: // GameStartPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &GameStartPacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}
		return packet, 3, nil

	case CodeAction: // ActionPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &ActionPacket{
			version: version,
			code:    code,
			action:  int(data[2]),
		}
		return packet, 3, nil

	case CodeBoard: // BoardPacket
		if len(data) < 21 {
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

	case CodeDelta: // DeltaPacket
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

	case CodeGameClose: // GameClosePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &GameClosePacket{
			version: version,
			code:    code,
			Success: int(data[2]),
		}
		return packet, 3, nil

	case CodeEndGame: // EndGamePacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &EndGamePacket{
			version: version,
			code:    code,
			Win:     data[2] == 1,
		}
		return packet, 3, nil

	case CodeSpellSelection: // SpellSelectionPacket
		if len(data) < 3 {
			return nil, 0, errors.New("incomplete packet")
		}
		usernameLen := int(data[2])
		if len(data) < 3+usernameLen {
			return nil, 0, errors.New("incomplete packet")
		}
		username := string(data[3 : 3+usernameLen])
		packet := &UsernamePacket{
			version:  version,
			code:     code,
			Username: username,
		}
		return packet, 3 + usernameLen, nil

	case CodeShopResponse: // ShopResponsePacket
		if len(data) < 26 {
			return nil, 0, errors.New("incomplete packet")
		}
		health := int(binary.BigEndian.Uint16(data[2:4]))
		mana := int(binary.BigEndian.Uint16(data[4:6]))
		attackDamage := int(binary.BigEndian.Uint16(data[6:8]))
		magicPower := int(binary.BigEndian.Uint16(data[8:10]))
		armor := int(binary.BigEndian.Uint16(data[10:12]))
		gold := int(binary.BigEndian.Uint16(data[12:14]))
		var inventory []int
		for i := range 6 {
			start := 14 + i*2
			inventory = append(inventory, int(binary.BigEndian.Uint16(data[start:start+2])))
		}
		packet := &ShopResponsePacket{
			version:      version,
			code:         code,
			Health:       health,
			Mana:         mana,
			AttackDamage: attackDamage,
			MagicPower:   magicPower,
			Armor:        armor,
			Gold:         gold,
			Inventory:    inventory,
		}
		return packet, 26, nil

	case CodePurchaseItem: // PurchaseItemPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &PurchaseItemPacket{
			version: version,
			code:    code,
			ItemID:  int(binary.BigEndian.Uint16(data[2:4])),
		}
		return packet, 4, nil

	case CodeMessage: // MessagePacket
		if len(data) < 6 {
			return nil, 0, errors.New("incomplete packet")
		}
		senderLen := int(binary.BigEndian.Uint16(data[2:4]))
		if len(data) < 4+senderLen+2 {
			return nil, 0, errors.New("incomplete packet")
		}
		messageLen := int(binary.BigEndian.Uint16(data[4+senderLen : 4+senderLen+2]))
		totalLen := 4 + senderLen + 2 + messageLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		sender := string(data[4 : 4+senderLen])
		message := string(data[4+senderLen+2 : totalLen])
		packet := &MessagePacket{
			version: version,
			code:    code,
			Sender:  sender,
			Message: message,
		}
		return packet, totalLen, nil

	case CodeMessageResponse: // MessageResponsePacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet header")
		}
		messageLen := int(binary.BigEndian.Uint16(data[2:4]))
		if len(data) < 4+messageLen {
			return nil, 0, errors.New(fmt.Sprintf("incomplete packet message length: %d | %d", messageLen, len(data)))
		}
		totalLen := 4 + messageLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet total length")
		}
		message := string(data[4 : 4+messageLen])
		packet := &MessageResponsePacket{
			version: version,
			code:    code,
			Message: message,
		}
		return packet, totalLen, nil

	case CodeMessageError: // MessageErrorPacket
		if len(data) < 4 {
			return nil, 0, errors.New("incomplete packet")
		}
		errorLen := int(binary.BigEndian.Uint16(data[2:4]))
		totalLen := 4 + errorLen
		if len(data) < totalLen {
			return nil, 0, errors.New("incomplete packet")
		}
		errorMsg := string(data[4:totalLen])
		packet := &MessageErrorPacket{
			version: version,
			code:    code,
			Error:   errorMsg,
		}
		return packet, totalLen, nil
	case CodeRateLimit:
		if len(data) < 2 {
			return nil, 0, errors.New("incomplete packet")
		}
		packet := &RateLimitPacket{
			version: version,
			code:    code,
		}
		return packet, 2, nil
	default:
		return nil, 0, errors.New("unknown message type")
	}
}
