package game

import "time"

type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

type Player struct {
	Number  Cell `json:"number"`
	TeamID  int  `json:"teamID"`
	X       int  `json:"X"`
	Y       int  `json:"Y"`
	Facing  Direction  //Facing direction
	Action  actionType
	HasFlag bool
	Flag    *Flag
  Dash    Dash `json:"dash"`
}

type Dash struct {
  Range    int `json:"range"`
  Cooldown int `json:"cooldown"`
  LastUsed time.Time `json:"-"`
}
