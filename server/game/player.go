package game

type Player struct {
	number  Cell
	TeamID  int
	X, Y    int
	Action  actionType
	HasFlag bool
	Flag    *Flag
}

func NewPlayer(n int) *Player {
	switch n {
	case 0:
		return &Player{
			number: Player1,
			TeamID: 6,
			X:      1,
			Y:      6,
			Action: NoAction,
		}
	default:
		return &Player{}
	}
}
