package shared

import (
	"bytes"
	"errors"
)

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

type LoginPayload struct {
  Username, Password string
}

func (lp LoginPayload) GetMessage() string {
  return "login"
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


func DeSerialize(data []byte) (*LoginPayload, error) {
  if len(data) < 2 {
    return &LoginPayload{}, errors.New("invalid packet length")
  }
  version := data[0]
  if version != 1 {
    return &LoginPayload{}, errors.New("invalid version")
  }
  messageType := data[1]
  if messageType == 0 {
    payloadData := data[2:]
    parts := bytes.SplitN(payloadData, []byte{0x00}, 2)
    username := string(parts[0])
    password := string(parts[1])
    return &LoginPayload{
      Username: username,
      Password: password,
    }, nil
  }
  return &LoginPayload{}, nil
}
