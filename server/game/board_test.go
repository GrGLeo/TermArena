package game_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GrGLeo/ctf/server/game"
)


func TestRunLengthEncode(t *testing.T) {
  // First test set up
  gridOne := game.Board{}
  for i := 0; i < len(gridOne.Grid); i++ {
    for j := 0; j < len(gridOne.Grid[i]); j++ {
      gridOne.Grid[i][j] = 1
    } 
  }
  stringOne := []string{}
  for i := 0; i < 20; i++ {
    stringOne = append(stringOne, "1:50")
  }
  expectOne := strings.Join(stringOne, "|")
  
  // Second test set up
  gridSec := game.Board{}
  for i := 0; i < len(gridSec.Grid); i++ {
    for j := 0; j < len(gridSec.Grid[i]); j++ {
      if j % 5 == 0 {
        gridSec.Grid[i][j] = 1
      } else {
        gridSec.Grid[i][j] = 2
      }
    }
  }
  stringSec := []string{}
  for i := 0; i < 20; i++ {
    for j := 0; j < 10; j++ {
      stringSec = append(stringSec, "1:1")
      stringSec = append(stringSec, "2:4")
    }
  }
  expectSec := strings.Join(stringSec, "|")

  // Third test set up
  pattern := []game.Cell{1, 1, 2, 2, 2, 3, 3, 1, 1, 1}
	patternLength := len(pattern)
	var gridThree [20][50]game.Cell
	for i := 0; i < 20; i++ {
		for j := 0; j < 50; j++ {
			gridThree[i][j] = pattern[j%patternLength]
		}
	}

	var stringThree []string
	for i := 0; i < 20; i++ {
    part := fmt.Sprintf("1:2|2:3|3:2|1:5|2:3|3:2|1:5|2:3|3:2|1:5|2:3|3:2|1:5|2:3|3:2|1:3")
		stringThree = append(stringThree, part)
	}
	expectedThree := strings.Join(stringThree, "|")

	tests := []struct {
		name     string
		board    game.Board
		expected string
	}{
		{
			name: "Single value row",
			board: gridOne,
			expected: expectOne,
		},
		{
			name: "Alternating values row",
			board: gridSec,
			expected: expectSec,
    },
		{
			name: "Mixed values row",
			board: game.Board{
				Grid: gridThree,
      },
			expected: expectedThree,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.board.RunLengthEncode()
			if string(result) != tt.expected {
				t.Errorf("expected\n %q, got\n %q", tt.expected, result)
			}
		})
	}
}

func TestWallPlacement(t *testing.T) {
	tests := []struct {
		name         string
		board        *game.Board
		wallPosition game.WallPosition
		expectedGrid [20][50]game.Cell
	}{
		{
			name:  "Single cell wall",
			board: &game.Board{},
			wallPosition: game.WallPosition{
				StartPos: [2]int{1, 1},
				EndPos:   [2]int{1, 1},
			},
			expectedGrid: func() [20][50]game.Cell {
				var grid [20][50]game.Cell
				grid[1][1] = game.Wall
				return grid
			}(),
		},
		{
			name:  "Horizontal wall",
			board: &game.Board{},
			wallPosition: game.WallPosition{
				StartPos: [2]int{2, 3},
				EndPos:   [2]int{2, 6},
			},
			expectedGrid: func() [20][50]game.Cell {
				var grid [20][50]game.Cell
				for j := 3; j <= 6; j++ {
					grid[2][j] = game.Wall
				}
				return grid
			}(),
		},
		{
			name:  "Vertical wall",
			board: &game.Board{},
			wallPosition: game.WallPosition{
				StartPos: [2]int{4, 5},
				EndPos:   [2]int{7, 5},
			},
			expectedGrid: func() [20][50]game.Cell {
				var grid [20][50]game.Cell
				for i := 4; i <= 7; i++ {
					grid[i][5] = game.Wall
				}
				return grid
			}(),
		},
		{
			name:  "Rectangular wall",
			board: &game.Board{},
			wallPosition: game.WallPosition{
				StartPos: [2]int{6, 6},
				EndPos:   [2]int{8, 8},
			},
			expectedGrid: func() [20][50]game.Cell {
				var grid [20][50]game.Cell
				for i := 6; i <= 8; i++ {
					for j := 6; j <= 8; j++ {
						grid[i][j] = game.Wall
					}
				}
				return grid
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.board.PlaceWall(tt.wallPosition)
			for i := 0; i < len(tt.board.Grid); i++ {
				for j := 0; j < len(tt.board.Grid[i]); j++ {
					if tt.board.Grid[i][j] != tt.expectedGrid[i][j] {
						t.Errorf("Mismatch at Grid[%d][%d]: got %v, want %v", i, j, tt.board.Grid[i][j], tt.expectedGrid[i][j])
					}
				}
			}
		})
	}
}

func TestPlaceAllWall(t *testing.T) {
  b := game.Init()
  walls := []game.WallPosition{
    {StartPos: [2]int{0, 0}, EndPos: [2]int{0, 0}}, // Single cell
    {StartPos: [2]int{5, 5}, EndPos: [2]int{5, 10}}, // Horizontal wall
  }
  b.PlaceAllWalls(walls)

  // Check single-cell wall
  if b.Grid[0][0] != game.Wall {
    t.Error("Wall not placed at (0,0)")
  }

  // Check horizontal wall
  for x := 5; x <= 10; x++ {
    if b.Grid[5][x] != game.Wall {
      t.Errorf("Wall not placed at (5, %d)", x)
    }
  }
}
