package event

import (
	"errors"
)

type Message interface {
  Type() string
  Validate() error
}

type LoginMessage struct {
  Username string
  Password string
}

func (lm LoginMessage) Type() string {
  return "login"
}

func (lm LoginMessage) Validate() error {
  if lm.Username == "" || lm.Password == "" {
    return errors.New("Username and Password are required")
  }
  return nil
}

type AuthMessage struct {
  Success int 
}

func (am AuthMessage) Type() string {
  return "auth"
}

func (am AuthMessage) Validate() error {
  if am.Success != 0 {
    return errors.New("Wrong credential")
  }
  return nil
}
