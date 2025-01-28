package auth

import (
	"github.com/GrGLeo/ctf/server/event"
)

func Authentificate(msg event.Message) event.Message {
  return event.AuthMessage{Success: 1}
}
