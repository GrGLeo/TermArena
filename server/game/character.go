package game

type Player struct {
  number Cell
  TeamID int
  X, Y int
  Action actionType
  HasFlag bool
}


func NewPlayer(n int) *Player {
  switch n { 
  case 0:
    return &Player{
      number: Player1,
      TeamID: 1,
      X: 1,
      Y: 6,
      Action: NoAction,
    }
  default:
    return &Player{}
  }
}
