package model

import "time"

// TickMsg is used to send a time-based tick message.
type TickMsg struct {
	Time time.Time
}

// LoginMsg is used to pass input field to meta model
type LoginMsg struct {
  Username string
  Password string
}

