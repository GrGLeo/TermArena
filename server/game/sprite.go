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

type FreezeSprite struct {
	X, Y      int
	lifeCycle int
	Facing    Direction
}

func (s *FreezeSprite) Update() (int, int, Cell) {
  var changes bool 
  milestone := []int{14, 11, 9, 7, 5, 4, 3, 2, 1, 0}
  for _, i := range milestone {
    if i == s.lifeCycle {
      changes = true
      break
    }
  }
  s.lifeCycle--
  if !changes {
    return s.X, s.Y, Frozen
  }

  switch s.Facing {
  case Up:
    s.Y = max(s.Y-1, 0)
    if s.Y == 0 {
      s.lifeCycle = 0
    }
  case Down:
    s.Y = min(s.Y+1, 19)
    if s.Y == 19 {
      s.lifeCycle = 0
    }
  case Left:
    s.X = max(s.X-1, 0)
    if s.X == 0 {
      s.lifeCycle = 0
    }
  case Right:
    s.X = min(s.X+1, 49)
    if s.X == 49 {
      s.lifeCycle = 0
    }
  }
  return s.X, s.Y, Frozen
}

func (s *FreezeSprite) Clear() bool {
  return s.lifeCycle <= 0
}
