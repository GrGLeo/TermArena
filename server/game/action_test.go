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
			name:         "Move Up",
			player:       &Player{X: 25, Y: 10, Action: moveUp, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX:    25,
			expectedY:    9,
			expectedGrid: createBoardWithPlayer(25, 9),
		},
		{
			name:         "Move Down at Bottom Edge",
			player:       &Player{X: 25, Y: 19, Action: moveDown, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(25, 19)},
			expectedX:    25,
			expectedY:    19,
			expectedGrid: createBoardWithPlayer(25, 19),
		},
		{
			name:         "Move Left",
			player:       &Player{X: 25, Y: 10, Action: moveLeft, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX:    24,
			expectedY:    10,
			expectedGrid: createBoardWithPlayer(24, 10)},
		{
			name:         "Move Right at Right Edge",
			player:       &Player{X: 49, Y: 10, Action: moveRight, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(49, 10)},
			expectedX:    49,
			expectedY:    10,
			expectedGrid: createBoardWithPlayer(49, 10),
		},
		{
			name:         "No Action",
			player:       &Player{X: 25, Y: 10, Action: NoAction, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX:    25,
			expectedY:    10,
			expectedGrid: createBoardWithPlayer(25, 10),
		},
		{
			name:         "Invalid Action",
			player:       &Player{X: 25, Y: 10, Action: actionType(999), number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(25, 10)},
			expectedX:    25,
			expectedY:    10,
			expectedGrid: createBoardWithPlayer(25, 10),
		},
		{
			name:         "Wall collision left",
			player:       &Player{X: 36, Y: 15, Action: moveLeft, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(36, 15)},
			expectedX:    36,
			expectedY:    15,
			expectedGrid: createBoardWithPlayer(36, 15),
		},
		{
			name:         "Wall collision right",
			player:       &Player{X: 34, Y: 15, Action: moveRight, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(34, 15)},
			expectedX:    34,
			expectedY:    15,
			expectedGrid: createBoardWithPlayer(34, 15),
		},
		{
			name:         "Wall collision up",
			player:       &Player{X: 35, Y: 16, Action: moveUp, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(35, 16)},
			expectedX:    35,
			expectedY:    16,
			expectedGrid: createBoardWithPlayer(35, 16),
		},
		{
			name:         "Wall collision down",
			player:       &Player{X: 35, Y: 14, Action: moveDown, number: Player1},
			board:        &Board{Grid: createBoardWithPlayer(35, 14)},
			expectedX:    35,
			expectedY:    14,
			expectedGrid: createBoardWithPlayer(35, 14),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.player.Move(tt.board)
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

func TestMoveWithFlag(t *testing.T) {
	flag := &Flag{
		PosX: 30,
		PosY: 13,
	}
	flagWall := &Flag{
		PosX: 35,
		PosY: 17,
	}
	tests := []struct {
		name         string
		player       *Player
		board        *Board
		flag         *Flag
		expectedX    int
		expectedY    int
	}{
		{
			name:         "Flag follow up",
			player:       &Player{X: 30, Y: 12, Action: moveUp, number: Player1, HasFlag: true, Flag: flag},
			board:        &Board{Grid: createBoardWithPlayer(30, 12)},
			flag:         flag,
			expectedX:    30,
			expectedY:    12,
		},
		{
			name:         "Flag follow down",
			player:       &Player{X: 30, Y: 14, Action: moveDown, number: Player1, HasFlag: true, Flag: flag},
			board:        &Board{Grid: createBoardWithPlayer(30, 14)},
			flag:         flag,
			expectedX:    30,
			expectedY:    14,
		},
		{
			name:         "Flag follow left",
			player:       &Player{X: 29, Y: 13, Action: moveLeft, number: Player1, HasFlag: true, Flag: flag},
			board:        &Board{Grid: createBoardWithPlayer(29, 13)},
			flag:         flag,
			expectedX:    29,
			expectedY:    13,
		},
		{
			name:         "Flag follow right",
			player:       &Player{X: 31, Y: 13, Action: moveRight, number: Player1, HasFlag: true, Flag: flag},
			board:        &Board{Grid: createBoardWithPlayer(31, 13)},
			flag:         flag,
			expectedX:    31,
			expectedY:    13,
		},
		{
			name:         "Flag follow up with wall",
			player:       &Player{X: 35, Y: 16, Action: moveUp, number: Player1, HasFlag: true, Flag: flagWall},
			board:        &Board{Grid: createBoardWithPlayer(35, 16)},
			flag:         flagWall,
			expectedX:    35,
			expectedY:    17,
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.player.Move(tt.board)
			if tt.flag.PosX != tt.expectedX || tt.flag.PosY != tt.expectedY {
				t.Errorf("Move() = %v, %v; want %v, %v", tt.flag.PosX, tt.flag.PosY, tt.expectedX, tt.expectedY)
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
	grid[15][35] = Wall
	grid[y][x] = Player1
	return grid
}
