package game

import "fmt"

/*
Dash sprite
⣿ ⣶ ⣤ ⣀
*/

type Sprite interface {
	Update(int) (int, int, Cell)
	Clear() bool
}

type DashSprite struct {
	X, Y      int
	lifeCycle int
}

func (s *DashSprite) Update(tick int) (int, int, Cell) {
  fmt.Println("from update sprite")
	cellState := s.lifeCycle / 10
	s.lifeCycle -= tick
	switch cellState {
	case 4:
		return s.X, s.Y, Dash1
	case 3:
		return s.X, s.Y, Dash1
	case 2:
		return s.X, s.Y, Dash1
	case 1:
		return s.X, s.Y, Dash1
	case 0:
		return s.X, s.Y, Empty
	default:
		return s.X, s.Y, Empty
	}
}

func (s *DashSprite) Clear() bool {
	return s.lifeCycle <= 0
}
