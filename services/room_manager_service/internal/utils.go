package internal

import (
	"crypto/rand"
	"encoding/binary"
)

func GenerateRoomID() RoomID {
  var id RoomID
  binary.Read(rand.Reader, binary.LittleEndian, &id)
  return id
}

