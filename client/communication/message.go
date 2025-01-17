package communication

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

/* ResponseMsg is used to validate login
code 0: login succes
code 1: invalid credential */
type ResponseMsg struct {
  Code int
}


// BoardMsg is used to transfer the board to game model
type BoardMsg struct {
  Board [20][50]int
}
