package game

type Player struct {
  number Cell
  X, Y int
  Action actionType
}


func NewPlayer(n int) *Player {
  switch n { 
  case 0:
    return &Player{
      number: Player1,
      X: 1,
      Y: 6,
      Action: NoAction,
    }
  default:
    return &Player{}
  }
}
