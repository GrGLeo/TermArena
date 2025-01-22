package game

import (
	"fmt"
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


func(b *Board) RunLengthEncode() []byte {
  var rle []string
  
  for row := 0; row < len(b.Grid); row++ {
    count := 0
    for col := 1; col < len(b.Grid[row]); col++ {
      if b.Grid[row][col] == b.Grid[row][col-1] {
        count++
      } else {
        count += 1
        rle = append(rle, fmt.Sprintf("%d:%d", b.Grid[row][col-1], count))
        count = 0
      }
    }
    if count > 0 {
      count += 1
      rle = append(rle, fmt.Sprintf("%d:%d", b.Grid[row][len(b.Grid[row])-1], count))
    }
  }
  rleString := strings.Join(rle, "|")
  return []byte(rleString)
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



