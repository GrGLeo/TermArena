package game

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
	var flagIdx int
	var action string
	minCost := 68
	if b.TeamID == 6 {
		flagIdx = 1
	} else {
		flagIdx = 0
	}
	flag := board.Flags[flagIdx]
	fx, fy := flag.PosX, flag.PosY

	neighbors := map[string][2]int{
		"down":  {0, 1},
		"right": {1, 0},
		"up":    {0, -1},
		"left":  {-1, 0},
	}

	for key, dir := range neighbors {
		x, y := b.X+dir[0], b.Y+dir[1]
		if x < 0 || x >= len(board.CurrentGrid) || y < 0 || y >= len(board.CurrentGrid[0]) || board.CurrentGrid[x][y] == 1 {
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
	if tick%3 == 0 {
		action := b.CalculatePath(board)
		b.Action = action
		return b.TakeAction(board)
	}
	b.Action = NoAction
	return b.TakeAction(board)
}
