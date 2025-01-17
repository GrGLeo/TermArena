package shared

import (
	"bytes"
	"errors"
	"log"
)

/*
code 0: send login
code 1: receive login response
code 2: send action
code 3: receive board
*/

type MessageType interface {
  GetMessage() string
}

type Packet struct {
  Version byte
  MessageType byte
  Payload []byte
}

func NewPacket(version, message byte, payload []byte) *Packet {
  return &Packet{
    Version: version,
    MessageType: message,
    Payload: payload,
  }
}

type LoginPacket struct {
  Username, Password string
}

func (lp LoginPacket) GetMessage() string {
  return "login"
}

type RespPacket struct {
  Code int
}

func (rp RespPacket) GetMessage() string {
  return "respLogin"
}

type BoardPacket struct {
  EncodedBoard []byte
}
 
func (bp BoardPacket) GetMessage() string {
  return "BoardPacket"
}

func (p *Packet) Serialize() ([]byte, error) {
  var buf bytes.Buffer
	if err := buf.WriteByte(p.Version); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(p.MessageType); err != nil {
		return nil, err
	}
	if _, err := buf.Write(p.Payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}


func DeSerialize(data []byte) (MessageType, error) {
  log.Print(string(data))
  if len(data) < 2 {
    return &LoginPacket{}, errors.New("invalid packet length")
  }
  version := data[0]
  if version != 1 {
    return &LoginPacket{}, errors.New("invalid version")
  }
  messageType := data[1]
  switch messageType {
  case 0:
    payloadData := data[2:]
    parts := bytes.SplitN(payloadData, []byte{0x00}, 2)
    username := string(parts[0])
    password := string(parts[1])
    return &LoginPacket{
      Username: username,
      Password: password,
    }, nil
  case 1:
    code := int(data[2])
    return &RespPacket{Code: code}, nil
  case 3:
    return &BoardPacket{EncodedBoard: data[2:]}, nil
  }
  return &LoginPacket{}, nil
}
