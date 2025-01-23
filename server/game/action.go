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
  // Check if flag is attached and need to move
  if p.HasFlag {
    if board.CheckFlagWon(p.TeamID, p.Y, p.X) {
      // We need to reset the flag pos.
      board.Grid[p.Flag.PosY][p.Flag.PosX] = Empty
      p.Flag.ResetPos()
      p.HasFlag = false
      p.Flag = nil
    } else if p.X != posX || p.Y != posY {
      p.Flag.Move(posX, posY, board)
      p.Action = NoAction 
    }
    return
  }
  // Check if player catch flag
  if flag := board.CheckFlagCaptured(p.TeamID, p.Y, p.X); flag != nil  {
    p.HasFlag = true
    p.Flag = flag
    p.Action = NoAction 
    return
  }
  p.Action = NoAction 
  return
}


type ActionMsg struct {
  ConnAddr string
  Action int
}
