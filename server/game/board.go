package game

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Cell int 

const (
  Empty Cell = iota
  Wall
  Player1
)

type Board struct {
  Grid [20][50]Cell
}

func Init() *Board {
  return &Board{}
}

func (b *Board) PlacePlayer(n int) {
  switch n {
  case 0:
    b.Grid[6][1] = Player1
  }
}

// Check if the position is within the grid bounds
// And if the position is not a wall
func (b *Board) IsValidPosition(x, y int) bool {
	if y < 0 || y >= len(b.Grid) || x < 0 || x >= len(b.Grid[y]) {
		return false
	}
	return b.Grid[y][x] != Wall
}

func (b *Board) PlaceWall(wall WallPosition) {
  ys,xs := wall.GetStartPos()
  ye,xe := wall.GetEndPos()
  for i := ys; i <= ye; i++ {
    for j := xs; j <= xe; j++ {
      b.Grid[i][j] = Wall
    }
  }
}

func (b *Board) PlaceAllWall(walls []WallPosition) {
  for _, wall := range walls {
    b.PlaceWall(wall)
  }
}

func (b *Board) RunLengthEncode() []byte {
  var rle []string

  for _, row := range b.Grid {
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


type WallPosition struct {
  StartPos [2]int // Y and X start position on the borad
  EndPos [2]int // Y and X end position on the board
}

// Return Y and X start position
func (w WallPosition) GetStartPos() (int, int) {
  return w.StartPos[0], w.StartPos[1]
}
  
// Return Y and X end position
func (w WallPosition) GetEndPos() (int, int) {
  return w.EndPos[0], w.EndPos[1]
}

type WallJSON struct {
  Walls []WallPosition `json:"walls"`
}

// Read from a config file to get all walls placement
func LoadWalls(filename string) ([]WallPosition, error) {
  var wallJSON WallJSON
  file, err := os.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  err = json.Unmarshal(file, &wallJSON)
  if err != nil {
    return nil, err
  }
  return wallJSON.Walls, nil
}
