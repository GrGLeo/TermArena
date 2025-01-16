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
  rleString := strings.Join(rle, string([]byte{0x00}))
  return []byte(rleString)
}

