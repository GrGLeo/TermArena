package game


/*
Dash sprite
⣿ ⣶ ⣤ ⣀
*/

type Sprite interface {
	Update() (int, int, Cell)
	Clear() bool
}

type DashSprite struct {
	X, Y      int
	lifeCycle int
}

func (s *DashSprite) Update() (int, int, Cell) {
	cellState := s.lifeCycle / 10
	s.lifeCycle -= 1
	switch cellState {
	case 4:
		return s.X, s.Y, Dash1
	case 3:
		return s.X, s.Y, Dash2
	case 2:
		return s.X, s.Y, Dash3
	case 1:
		return s.X, s.Y, Dash4
	case 0:
		return s.X, s.Y, Empty
	default:
		return s.X, s.Y, Empty
	}
}

func (s *DashSprite) Clear() bool {
	return s.lifeCycle <= 0
}
