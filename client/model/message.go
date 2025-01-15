package model

import "time"

// TickMsg is used to send a time-based tick message.
type TickMsg struct {
	Time time.Time
}

