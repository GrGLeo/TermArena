package game

import "testing"


func TestMove(t *testing.T) {
	tests := []struct {
		name         string
		player       *Player
		board        *Board
		expectedX    int
		expectedY    int
		expectedGrid [20][50]Cell
	}{
		{
			name: "Move Up",
			player: &Player{X: 25, Y: 10, Action: moveUp, number: Player1},
			board: &Board{Grid: createBoardWithPlayer(25,10)}, 
			expectedX: 25,
			expectedY: 9,
      expectedGrid: createBoardWithPlayer(25,9),
		},
		{
			name: "Move Down at Bottom Edge",
			player: &Player{X: 25, Y: 19, Action: moveDown, number: Player1},
			board: &Board{Grid: createBoardWithPlayer(25, 19)},
			expectedX: 25,
			expectedY: 19,
			expectedGrid: createBoardWithPlayer(25, 19),
		},
		{
			name: "Move Left",
			player: &Player{X: 25, Y: 10, Action: moveLeft, number: Player1},
			board: &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX: 24,
			expectedY: 10,
			expectedGrid: createBoardWithPlayer(24, 10)},
		{
			name: "Move Right at Right Edge",
			player: &Player{X: 49, Y: 10, Action: moveRight, number: Player1},
			board: &Board{Grid: createBoardWithPlayer(49, 10)},
			expectedX: 49,
			expectedY: 10,
			expectedGrid: createBoardWithPlayer(49, 10),
    },
		{
			name: "No Action",
			player: &Player{X: 25, Y: 10, Action: NoAction, number: Player1},
			board: &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX: 25,
			expectedY: 10,
			expectedGrid: createBoardWithPlayer(25, 10),
		},
		{
			name: "Invalid Action",
			player: &Player{X: 25, Y: 10, Action: actionType(999), number: Player1},
			board: &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX: 25,
			expectedY: 10,
			expectedGrid: createBoardWithPlayer(25, 10),
    },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Move(tt.player, tt.board)
			if tt.player.X != tt.expectedX || tt.player.Y != tt.expectedY {
				t.Errorf("Move() = %v, %v; want %v, %v", tt.player.X, tt.player.Y, tt.expectedX, tt.expectedY)
			}
			for y, row := range tt.board.Grid {
				for x, val := range row {
					if val != tt.expectedGrid[y][x] {
						t.Errorf("Grid[%d][%d] = %v; want %v", y, x, val, tt.expectedGrid[y][x])
					}
				}
			}
		})
	}
}

func createBoardWithPlayer(x, y int) [20][50]Cell {
	var grid [20][50]Cell
	for i := range grid {
		for j := range grid[i] {
			grid[i][j] = Empty
		}
	}
	grid[y][x] = Player1
	return grid
}

