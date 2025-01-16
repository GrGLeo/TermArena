package shared

import (
	"bytes"
	"errors"
	"strconv"
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

type LoginPayload struct {
  Username, Password string
}

func (lp LoginPayload) GetMessage() string {
  return "login"
}

type LoginResp struct {
  code int
}

func (lr LoginResp) GetMessage() string {
  return "respLogin"
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
  if len(data) < 2 {
    return &LoginPayload{}, errors.New("invalid packet length")
  }
  version := data[0]
  if version != 1 {
    return &LoginPayload{}, errors.New("invalid version")
  }
  messageType := data[1]
  switch messageType {
  case 0:
    payloadData := data[2:]
    parts := bytes.SplitN(payloadData, []byte{0x00}, 2)
    username := string(parts[0])
    password := string(parts[1])
    return &LoginPayload{
      Username: username,
      Password: password,
    }, nil
  case 1:
    payLoadData := data[:2]
    code, err := strconv.Atoi(string(payLoadData))
    if err != nil {
      return LoginResp{}, err
    }
    return LoginResp{code: code}, nil
  }
  return LoginPayload{}, nil
}
