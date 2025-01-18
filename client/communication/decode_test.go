package communication_test

import (
	"strings"
	"testing"

	"github.com/GrGLeo/ctf/client/communication"
)

func TestDecodeRLE(t *testing.T) {
	// Test 1: Board with only 0s
	gridZero, rle := generateZeroGrid()
	decodedGridZero, err := communication.DecodeRLE(rle)
	if err != nil {
		t.Fatalf("Expected no error for grid with 0s, got %v", err)
	}
	checkGridsMatch(t, gridZero, decodedGridZero)

	// Test 2: Board alternating 0 and 1
	gridAlt, rle := generateAltGrid()
	decodedGridAlt, err := communication.DecodeRLE(rle)
	if err != nil {
		t.Fatalf("Expected no error for alternating grid, got %v", err)
	}
	checkGridsMatch(t, gridAlt, decodedGridAlt)

	// Test 3: Board with more variation
	gridVaried, rle := generateVariedGrid()
	decodedGridVaried, err := communication.DecodeRLE(rle)
	if err != nil {
		t.Fatalf("Expected no error for varied grid, got %v", err)
	}
	checkGridsMatch(t, gridVaried, decodedGridVaried)
}

// Helper function to generate a 20x50 grid filled with 0s
func generateZeroGrid() ([20][50]int, []byte) {
	var grid [20][50]int
  var rleString []string
  for i := 0; i < 20; i++ {
    rleString = append(rleString, "0:50")
  }
  rleByte := []byte(strings.Join(rleString, "|"))
	return grid, rleByte
}

// Helper function to generate a 20x50 grid alternating between 0 and 1
func generateAltGrid() ([20][50]int, []byte) {
	var grid [20][50]int
	var rleString []string
	for i := 0; i < 20; i++ {
		for j := 0; j < 50; j++ {
      if j % 2 == 0 {
        grid[i][j] = 0
        rleString = append(rleString, "0:1")
      } else {
        grid[i][j] = 1
        rleString = append(rleString, "1:1")
      }
		}
	}
	rleByte := []byte(strings.Join(rleString, "|"))
	return grid, rleByte
}

// Helper function to generate a 20x50 grid with more varied patterns
func generateVariedGrid() ([20][50]int, []byte) {
	var grid [20][50]int
	var rleString []string

	for i := 0; i < 20; i++ {
		for j := 0; j < 50; j++ {
			grid[i][j] = (i*j + j) % 3
      if j % 3 == 0 {
			  grid[i][j] = 0
        rleString = append(rleString, "0:1")
      } else if j % 3 == 1 {
        grid[i][j] = 1
        rleString = append(rleString, "1:1")
      } else {
        grid[i][j] = 2
        rleString = append(rleString, "2:1")
      }
		}
	}

	rleByte := []byte(strings.Join(rleString, "|"))
	return grid, rleByte
}

// Helper function to check if two grids match
func checkGridsMatch(t *testing.T, grid1, grid2 [20][50]int) {
	for i := 0; i < 20; i++ {
		for j := 0; j < 50; j++ {
			if grid1[i][j] != grid2[i][j] {
				t.Errorf("Mismatch at [%d][%d]: expected %d, got %d", i, j, grid1[i][j], grid2[i][j])
        return
			}
		}
	}
}

