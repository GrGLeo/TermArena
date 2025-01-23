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
  Player2
  Player3
  Player4
  Flag1
  Flag2
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

/*
MANAGE ALL CELLS PLACEMENT
*/

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

func (b *Board) PlaceFlags(flags []Flag) {
  for _, flag := range flags {
    posX := flag.PosX
    posY := flag.PosY
    if flag.TeamId == 1 {
      b.Grid[posY][posX] = Flag1
    } else {
      b.Grid[posY][posX] = Flag2
    }
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


/*
MANAGE MAP FROM CONFIG
*/

type ConfigJSON struct {
  Walls []WallPosition `json:"walls"`
  Flag []Flag `json:"flags"`
}

// Read from a config file to get all walls placement
func LoadConfig(filename string) ([]WallPosition, []Flag, error) {
  var configJSON ConfigJSON
  file, err := os.ReadFile(filename)
  if err != nil {
    return nil, nil, err
  }
  err = json.Unmarshal(file, &configJSON)
  if err != nil {
    fmt.Println(err.Error())
    return nil, nil, err
  }
  return configJSON.Walls, configJSON.Flag, nil
}
