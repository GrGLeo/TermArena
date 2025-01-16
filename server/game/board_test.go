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
  expectOne := strings.Join(stringOne, string([]byte{0x00}))
  
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
  expectSec := strings.Join(stringSec, string([]byte{0x00}))

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
    part := fmt.Sprintf("1:2\x002:3\x003:2\x001:5\x002:3\x003:2\x001:5\x002:3\x003:2\x001:5\x002:3\x003:2\x001:5\x002:3\x003:2\x001:3")
		stringThree = append(stringThree, part)
	}
	expectedThree := strings.Join(stringThree, string([]byte{0x00}))

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

