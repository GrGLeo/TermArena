package game


type actionType int 

const (
  NoAction actionType = iota
  moveUp 
  moveDown
  moveLeft
  moveRight
  spellOne
  spellTwo
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
  // We save change as delta
  board.Tracker.SaveDelta(posX, posY, Empty)
  board.Tracker.SaveDelta(p.X, p.Y, p.Number)
  // Check if flag is attached and need to move
  if p.HasFlag {
    if board.CheckFlagWon(p.TeamID, p.Y, p.X) {
      // We need to reset the flag pos.
      board.Tracker.SaveDelta(p.Flag.PosX, p.Flag.PosY, Empty)
      p.Flag.ResetPos()
      p.HasFlag = false
      p.Flag = nil
      p.Action = NoAction 
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
