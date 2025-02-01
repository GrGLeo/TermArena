package game

import "time"


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

// TakeAction processes the player's current action on the given board.
//
// It handles different types of actions:
//  - NoAction: Does nothing and returns false.
//  - moveUp, moveDown, moveLeft, moveRight: Calls Move and returns true if the player captures the flag, false otherwise.
//  - spellOne: Calls MakeDash but does not return a capture result.
//  - spellTwo: Calls MakeFreeze but does not return a capture result.
// Returns:
//  - true if the player's movement results in capturing the flag.
//  - false otherwise.
func (p *Player) TakeAction(board *Board) bool {
  switch p.Action {
  case NoAction:
    return false
  case moveUp, moveDown, moveLeft, moveRight:
    return p.Move(board)
  case spellOne:
    p.MakeDash(board)
    return false
  case spellTwo:
    p.MakeFreeze(board)
    return false
  default:
    return false
  }
}

func (p *Player) Move(board *Board) bool {
  posX := p.X
  posY := p.Y
  newX := p.X
  newY := p.Y

  switch p.Action {
  case moveUp:
    p.Facing = Up
    newY--
  case moveDown:
    p.Facing = Down
    newY++
  case moveLeft:
    p.Facing = Left
    newX--
  case moveRight:
    p.Facing = Right
    newX++
  }
  valid := board.IsValidPosition(newX, newY)
  if valid {
    p.X = newX
    p.Y = newY
    board.Tracker.SaveDelta(posX, posY, Empty)
    board.Tracker.SaveDelta(p.X, p.Y, p.Number)
  }
  // We save change as delta
  // Check if flag is attached and need to move
  if p.HasFlag {
    if board.CheckFlagWon(p.TeamID, p.Y, p.X) {
      // We need to reset the flag pos.
      board.Tracker.SaveDelta(p.Flag.PosX, p.Flag.PosY, Empty)
      p.Flag.ResetPos()
      p.HasFlag = false
      p.Flag = nil
      p.Action = NoAction 
      return true
    } else if p.X != posX || p.Y != posY {
      p.Flag.Move(posX, posY, board)
      p.Action = NoAction 
    }
    return false
  }
  // Check if player catch flag
  if flag := board.CheckFlagCaptured(p.TeamID, p.Y, p.X); flag != nil  {
    p.HasFlag = true
    p.Flag = flag
    p.Action = NoAction 
    return false
  }
  p.Action = NoAction 
  return false
}

func (p *Player) MakeDash(board *Board){
  // Verify player is allowed to dash
  lastUsed := p.Dash.LastUsed
  cooldown := time.Duration(p.Dash.Cooldown) * time.Second
  EndCd := lastUsed.Add(cooldown)
  if p.HasFlag || time.Now().Before(EndCd) {
    p.Action = NoAction
    return
  }
  posX := p.X
  posY := p.Y
  newX := p.X
  newY := p.Y

  dashRange := p.Dash.Range
  switch p.Facing {
  case Up:
    newY -= dashRange
    for !board.IsValidPosition(newX, newY) {
      newY++
    }
  case Down:
    newY += dashRange
    for !board.IsValidPosition(newX, newY) {
      newY--
    }
  case Left:
    newX -= dashRange
    for !board.IsValidPosition(newX, newY) {
      newX++
    }
  case Right:
    newX += dashRange
    for !board.IsValidPosition(newX, newY) {
      newX--
    }
  }
  p.X = newX
  p.Y = newY
  
  // Generate sprite
  dx := newX - posX
  dy := newY - posY
  // Calculate off by one
  var corX int
  var corY int
  if dx != 0 {
    if dx < 0 {
      corX = 1
    } else {
      corX = -1
    } 
  } else {
    if dy < 0 {
      corY = 1
    } else {
      corY = -1
    }
  }
  maxSprite := max(Absolute(dx), Absolute(dy))
  if maxSprite != 0 {
    stepX := dx / maxSprite
    stepY := dy / maxSprite
    for i := 1; i <= maxSprite; i++ {
      x := posX + stepX * i + corX
      y := posY + stepY * i + corY
      if !board.IsValidPosition(x, y) {
        continue
      }
      lifecycle := 1 + i * 10
      sprite := &DashSprite{
        X: x,
        Y: y,
        lifeCycle: lifecycle,
      }
      board.Sprite = append(board.Sprite, sprite)
    }
  }


  board.Tracker.SaveDelta(posX, posY, Empty)
  board.Tracker.SaveDelta(newX, newY, p.Number)
  p.Dash.LastUsed = time.Now()
  p.Action = NoAction
}


func (p *Player) MakeFreeze(board *Board) {
  // Verify player is allowed to cast freeze
  lastUsed := p.Freeze.LastUsed
  cooldown := time.Duration(p.Dash.Cooldown) * time.Second
  EndCd := lastUsed.Add(cooldown)
  if time.Now().Before(EndCd) {
    p.Action = NoAction
    return
  }
  switch p.Facing {
  case Up:
    if p.Y == 0 {
      return
    }
    minX := p.X - 1
    if minX < 0 {
      minX = 0
    }
    maxX := p.X + 1
    if maxX > 49 {
      maxX = 49
    }
    for i := minX; i <= maxX; i++ {
      if board.CurrentGrid[p.Y-1][i] == Wall {
        continue
      }
      sprite := &FreezeSprite{
        X: i,
        Y: p.Y - 1,
        lifeCycle: 17,
        Facing: Up,
      }
      board.Sprite = append(board.Sprite, sprite)
    }
  case Down:
    if p.Y == 19 {
      return
    }
    minX := p.X - 1
    if minX < 0 {
      minX = 0
    }
    maxX := p.X + 1
    if maxX > 49 {
      maxX = 49
    }
    for i := minX; i <= maxX; i++ {
      if board.CurrentGrid[p.Y+1][i] == Wall {
        continue
      }
      sprite := &FreezeSprite{
        X: i,
        Y: p.Y + 1,
        lifeCycle: 17,
        Facing: Down,
      }
      board.Sprite = append(board.Sprite, sprite)
    }
  case Left:
    if p.X == 0 {
      return
    }
    minY := p.Y - 1
    if minY < 0 {
      minY = 0
    }
    maxY := p.Y + 1
    if maxY > 19 {
      maxY = 19
    }
    for i := minY; i <= maxY; i++ {
      if board.CurrentGrid[i][p.X-1] == Wall {
        continue
      }
      sprite := &FreezeSprite{
        X: p.X - 1,
        Y: i,
        lifeCycle: 17,
        Facing: Left,
      }
      board.Sprite = append(board.Sprite, sprite)
    }
  case Right:
    if p.X == 49 {
      return
    }
    minY := p.Y - 1
    if minY < 0 {
      minY = 0
    }
    maxY := p.Y + 1
    if maxY > 19 {
      maxY = 19
    }
    for i := minY; i <= maxY; i++ {
      if board.CurrentGrid[i][p.X+1] == Wall {
        continue
      }
      sprite := &FreezeSprite{
        X: p.X + 1,
        Y: i,
        lifeCycle: 17,
        Facing: Right,
      }
      board.Sprite = append(board.Sprite, sprite)
    }
  }
  p.Freeze.LastUsed = time.Now()
  p.Action = NoAction
}


type ActionMsg struct {
  ConnAddr string
  Action int
}

func Absolute(v int) int {
  if v > 0 {
    return v
  } else {
    return -v
  }
}
