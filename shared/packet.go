package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
)

/*
code 0: send login
code 1: receive login response
code 2: send action
code 3: receive board
*/

type Packet interface {
  Version() int
  Code() int
  Serialize() []byte
}


type LoginPacket struct {
  version, code int
  Username, Password string
}

func NewLoginPacket (username, password string) *LoginPacket {
  return &LoginPacket{
    version: 1,
    code: 0,
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
    code: 1,
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


type ActionPacket struct {
  version, code int
  action int
}

func NewActionPacket(action int) *ActionPacket {
  return &ActionPacket{
    version: 1,
    code: 2,
    action: action,
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
  EncodedBoard []byte
}

func NewBoardPacket(encodedBoard []byte) *BoardPacket {
  return &BoardPacket{
    version: 1,
    code: 3,
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
        username := string(data[4:4+usernameLen])
        
        // Password
        pwStart := 4 + usernameLen
        if len(data) < int(pwStart+2) {
            return nil, errors.New("invalid packet length for password length")
        }
        passwordLen := binary.BigEndian.Uint16(data[pwStart:pwStart+2])
        if len(data) < int(pwStart+2+passwordLen) {
            return nil, errors.New("invalid password length")
        }
        password := string(data[pwStart+2:pwStart+2+passwordLen])
        
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
      action := int(data[2])
      return &ActionPacket{
        version: version,
        code: code,
        action: action,
      }, nil

    case 3: // BoardPacket
        // All remaining bytes are the encoded board
        encodedBoard := data[2:]
        return &BoardPacket{
            version:      version,
            code:        code,
            EncodedBoard: encodedBoard,
        }, nil
        
    default:
        return nil, errors.New("unknown message type")
    }
}
