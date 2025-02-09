package game

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Cell int

const (
	Empty Cell = iota
	Wall
	Player1
	Player2
	Player3
	Player4
	Flag1
	Flag2
	Dash1
	Dash2
	Dash3
	Dash4
	Frozen
)

type Board struct {
	CurrentGrid [20][50]Cell
	PastGrid    [20][50]Cell
	Tracker     ChangeTracker
	Flags       []*Flag
	Players     []*Player
	Bots        []*Bot
	Sprite      []Sprite
	mu          sync.RWMutex
}

func InitBoard(walls []WallPosition, flags []*Flag, players []*Player, bots []*Bot) *Board {
	board := Board{}
	board.PlaceAllWalls(walls)
	board.PlaceAllFlags(flags)
	board.PlaceAllPlayers(players)
  board.PlaceAllBots(bots)
	board.CurrentGrid = board.GetPastGrid()
	return &board
}

/*
GAME LOGIC
*/

// Check if the position is within the grid bounds
// And if the position is not a wall
func (b *Board) IsValidPosition(x, y int) bool {
	if y < 0 || y >= len(b.PastGrid) || x < 0 || x >= len(b.PastGrid[y]) {
		return false
	}
	return b.PastGrid[y][x] != Wall
}

func (b *Board) CheckFlagCaptured(team, y, x int) *Flag {
	for i := range b.Flags {
		flag := b.Flags[i]
		// Check if flag is captured
		if int(flag.TeamId) != team && flag.PosX == x && flag.PosY == y {
			flag.IsCaptured = true
			return flag
		}
	}
	return nil
}

func (b *Board) CheckFlagWon(team, y, x int) bool {
	for i := range b.Flags {
		flag := b.Flags[i]
		posY, posX := flag.GetBase()
		// Check if flag is won
		if int(flag.TeamId) == team && posX == x && posY == y {
			// I need to replace the other enemy flag
			enemyFalgIdx := (i + 1) % 2
			enemyFalg := b.Flags[enemyFalgIdx]
			b.Tracker.SaveDelta(enemyFalg.baseX, enemyFalg.baseY, enemyFalg.TeamId)
			return true
		}
	}
	return false
}

/*
UPDATE AND RETURN BOARDS
*/

func (b *Board) Update() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, delta := range b.Tracker.GetDeltas() {
		b.CurrentGrid[delta.Y][delta.X] = delta.Value
	}
	b.PastGrid = b.CurrentGrid
}

func (b *Board) UpdateSprite() {
	for i := 0; i < len(b.Sprite); i++ {
		switch sprite := b.Sprite[i].(type) {
		case *DashSprite:
			x, y, cell := sprite.Update()
			b.Tracker.SaveDelta(x, y, cell)
		case *FreezeSprite:
			b.Tracker.SaveDelta(sprite.X, sprite.Y, Empty)
			x, y, cell := sprite.Update()
			switch b.PastGrid[y][x] {
			case Wall:
				sprite.lifeCycle = -1
			case Player1, Player2, Player3, Player4:
				for _, p := range b.Players {
					if p.Number == b.PastGrid[y][x] && p.TeamID != sprite.TeamID {
						p.IsFrozen = 20
						sprite.lifeCycle = -1
					}
				}
			default:
				b.Tracker.SaveDelta(x, y, cell)
			}
		}
		if b.Sprite[i].Clear() {
			b.Sprite = append(b.Sprite[:i], b.Sprite[i+1:]...)
			i-- // Adjust index
		}
	}
}

func (b *Board) GetCurrentGrid() [20][50]Cell {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.CurrentGrid
}

func (b *Board) GetPastGrid() [20][50]Cell {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.PastGrid
}

/*
MANAGE ALL CELLS PLACEMENT
*/

// Inital Player placement
func (b *Board) PlacePlayer(player *Player) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch player.Number {
	case 2:
		b.PastGrid[player.Y][player.X] = Player1
	case 3:
		b.PastGrid[player.Y][player.X] = Player2
	case 4:
		b.PastGrid[player.Y][player.X] = Player3
	case 5:
		b.PastGrid[player.Y][player.X] = Player4
	}
}

func (b *Board) PlaceAllPlayers(players []*Player) {
	for _, player := range players {
		b.PlacePlayer(player)
	}
	b.Players = players
}

// Inital Bot placement
func (b *Board) PlaceBot(bot *Bot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch bot.Number {
	case 3:
		b.PastGrid[bot.Y][bot.X] = Player2
	case 4:
		b.PastGrid[bot.Y][bot.X] = Player3
	case 5:
		b.PastGrid[bot.Y][bot.X] = Player4
	}
}

func (b *Board) PlaceAllBots(bots []*Bot) {
	for _, bot := range bots {
		b.PlaceBot(bot)
	}
	b.Bots = bots
}

// Initial wall placement
func (b *Board) PlaceWall(wall WallPosition) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ys, xs := wall.GetStartPos()
	ye, xe := wall.GetEndPos()
	for i := ys; i <= ye; i++ {
		for j := xs; j <= xe; j++ {
			b.PastGrid[i][j] = Wall
		}
	}
}

// Place all wall at the start of the game
func (b *Board) PlaceAllWalls(walls []WallPosition) {
	for _, wall := range walls {
		b.PlaceWall(wall)
	}
}

// Initial placement of flag
func (b *Board) PlaceFlag(flag *Flag) {
	b.mu.Lock()
	defer b.mu.Unlock()

	posY, posX := flag.GetBase()
	if flag.TeamId == 6 {
		b.PastGrid[posY][posX] = Flag1
	} else {
		b.PastGrid[posY][posX] = Flag2
	}
}

// Place both flag at the start of the game
func (b *Board) PlaceAllFlags(flags []*Flag) {
	for _, flag := range flags {
		flag.SetBase()
		b.PlaceFlag(flag)
	}
	b.Flags = flags
}

// Check if flag needs to be replace
func (b *Board) ReplaceHiddenFlag() {
	for _, flag := range b.Flags {
		if flag.IsSafe() && b.PastGrid[flag.PosY][flag.PosX] == Empty {
			b.Tracker.SaveDelta(flag.PosX, flag.PosY, flag.TeamId)
		}
	}
}

func RunLengthEncode(grid [20][50]Cell) []byte {
	var rle []string

	for _, row := range grid {
		var current Cell = Empty
		count := 0

		for x, cell := range row {
			if x == 0 {
				current = cell
				count = 1
				continue
			}

			if cell == current {
				count++
			} else {
				rle = append(rle, fmt.Sprintf("%d:%d", current, count))
				current = cell
				count = 1
			}
		}
		if count > 0 {
			rle = append(rle, fmt.Sprintf("%d:%d", current, count))
		}
	}

	return []byte(strings.Join(rle, "|"))
}

/*
MANAGE MAP FROM CONFIG
*/

type ConfigJSON struct {
	Walls  []WallPosition `json:"walls"`
	Flag   []Flag         `json:"flags"`
	Player []Player       `json:"players"`
	Bot    []Bot          `json:"bot,omitempty"`
}

// Read from a config file to get all walls placement
func LoadConfig(filename string) ([]WallPosition, []*Flag, []*Player, []*Bot, error) {
	var configJSON ConfigJSON
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = json.Unmarshal(file, &configJSON)
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil, nil, nil, err
	}

	flagPtrs := make([]*Flag, len(configJSON.Flag))
	for i := range configJSON.Flag {
		flagPtrs[i] = &configJSON.Flag[i]
	}

	playerPtrs := make([]*Player, len(configJSON.Player))
	for i := range configJSON.Player {
		playerPtrs[i] = &configJSON.Player[i]
	}

	botPtrs := make([]*Bot, len(configJSON.Bot))
	for i := range configJSON.Bot {
		botPtrs[i] = &configJSON.Bot[i]
	}
  fmt.Println(botPtrs)
	return configJSON.Walls, flagPtrs, playerPtrs, botPtrs, nil
}
