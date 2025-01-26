package auth

import (
	"fmt"

	"github.com/GrGLeo/ctf/server/event"
)

func Authentificate(msg event.Message) event.Message {
  fmt.Println("auth user")
  return event.AuthMessage{Success: 1}
}
