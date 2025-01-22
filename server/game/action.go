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

func Move(p *Player, board *Board) {
  maxY := len(board.Grid)
  maxX := len(board.Grid[0])

  posX := p.X
  posY := p.Y
  switch p.Action {
  case moveUp:
    if posY >= 1 {
      if board.Grid[posY - 1][posX] != Wall {
        p.Y -= 1
      }
    }
  case moveDown:
    if posY < maxY-1 {
      if board.Grid[posY + 1][posX] != Wall {
        p.Y += 1
      }
    }
  case moveLeft:
    if posX >= 1 {
      if board.Grid[posY][posX - 1] != Wall {
        p.X -= 1
      }
    }
  case moveRight:
    if posX < maxX-1 {
      if board.Grid[posY][posX + 1] != Wall {
        p.X += 1
      }
    }
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
