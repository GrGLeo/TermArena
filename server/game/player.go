package game

type Player struct {
	Number  Cell `json:"number"`
	TeamID  int  `json:"teamID"`
	X       int  `json:"X"`
	Y       int  `json:"Y"`
	Action  actionType
	HasFlag bool
	Flag    *Flag
}

func NewPlayer(n int) *Player {
	switch n {
	case 0:
		return &Player{
			Number: Player1,
			TeamID: 6,
			X:      1,
			Y:      6,
			Action: NoAction,
		}
	default:
		return &Player{}
	}
}
