package game


type actionType int 

const (
  moveUp actionType = iota
  moveDown
  moveLeft
  moveRight
  spellOne
  spellTwo
  NoAction
)

func (p *Player) Move(board *Board) {
  posX := p.X
  posY := p.Y
  newX := p.X
  newY := p.Y

  switch p.Action {
  case moveUp:
    newY--
  case moveDown:
    newY++
  case moveLeft:
    newX--
  case moveRight:
    newX++
  }
  valid := board.IsValidPosition(newX, newY)
  if valid {
    p.X = newX
    p.Y = newY
  }
  // old position is cleared
  board.Grid[posY][posX] = 0
  // moving the char on the board
  board.Grid[p.Y][p.X] = p.number
  p.Action = NoAction 
}


type ActionMsg struct {
  ConnAddr string
  Action int
}
