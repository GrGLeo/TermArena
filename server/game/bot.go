package game

import (
	"fmt"
	"time"
)

type State int

const (
	random State = iota
	search
	capture
	defend
)

type Bot struct {
	Player
	Path   [][2]int
	status State
}

func (b *Bot) BotTurn(tick int, board *Board) bool {
	switch b.status {
	case random:
		return b.RandomAction(tick, board)
	}
	return false
}

func (b *Bot) CalculatePath(board *Board) actionType {
	var action string
  var fx, fy int
	minCost := 68
  
  if !b.HasFlag{
    if b.TeamID == 6 {
	flag := board.Flags[1]
	fx, fy = flag.PosX, flag.PosY
    } else {
	flag := board.Flags[0]
	fx, fy = flag.PosX, flag.PosY
    }
  } else {
    if b.TeamID == 6 {
	flag := board.Flags[0]
	fx, fy = flag.baseX, flag.baseY
    } else {
	flag := board.Flags[1]
	fx, fy = flag.baseX, flag.baseY
    }
  }

	neighbors := map[string][2]int{
		"down":  {0, 1},
		"right": {1, 0},
		"up":    {0, -1},
		"left":  {-1, 0},
	}

	for key, dir := range neighbors {
		x, y := b.X+dir[0], b.Y+dir[1]
		if y < 0 || y >= len(board.CurrentGrid) || x < 0 || x >= len(board.CurrentGrid[0]) || board.CurrentGrid[y][x] == 1 {
			continue
		}
		costX := Absolute(x - fx)
		costY := Absolute(y - fy)
		cost := costX + costY
		if cost <= minCost {
			minCost = cost
			action = key
		}
	}
  fmt.Printf("Action: %q | Cost: %d | bot: %d\n", action, minCost, b.TeamID)
	switch action {
	case "down":
		return moveDown
	case "up":
		return moveUp
	case "left":
		return moveLeft
	case "right":
		return moveRight
	default:
		return NoAction
	}
}

func (b *Bot) RandomAction(tick int, board *Board) bool {
  lastUsed := b.Dash.LastUsed
  cooldown := time.Duration(b.Dash.Cooldown) * time.Second
  EndCd := lastUsed.Add(cooldown)
  if time.Now().Before(EndCd) {
    if tick%3 == 0 {
      action := b.CalculatePath(board)
      b.Action = action
      return b.TakeAction(board)
    }
    b.Action = NoAction
    return b.TakeAction(board)
  } else {
    b.Action = spellOne
    return b.TakeAction(board)
  }
}
