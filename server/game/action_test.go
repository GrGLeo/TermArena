package game

import (
	"testing"
)

func TestMove(t *testing.T) {
	tests := []struct {
		name           string
		player         *Player
		board          *Board
		expectedX      int
		expectedY      int
		expectedPlayer *Player
		expectedDeltas []Delta // Expected deltas after the move
	}{
		{
			name:   "Move Up",
			player: &Player{X: 25, Y: 10, Action: moveUp, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX:      25,
			expectedY:      9,
			expectedPlayer: &Player{X: 25, Y: 9, Number: Player1},
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0}, // Clear old position
				{X: 25, Y: 9, Value: 2},  // Set new position (assuming Player1 is represented by 1)
			},
		},
		{
			name:   "Move Down at Bottom Edge",
			player: &Player{X: 25, Y: 19, Action: moveDown, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(25, 19), Tracker: ChangeTracker{}},
			expectedX:      25,
			expectedY:      19,
			expectedPlayer: &Player{X: 25, Y: 19, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
		{
			name:   "Move Left",
			player: &Player{X: 25, Y: 10, Action: moveLeft, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX:      24,
			expectedY:      10,
			expectedPlayer: &Player{X: 24, Y: 10, Number: Player1},
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0}, // Clear old position
				{X: 24, Y: 10, Value: 2}, // Set new position
			},
		},
		{
			name:   "Move Right at Right Edge",
			player: &Player{X: 49, Y: 10, Action: moveRight, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(49, 10), Tracker: ChangeTracker{}},
			expectedX:      49,
			expectedY:      10,
			expectedPlayer: &Player{X: 49, Y: 10, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
		{
			name:   "No Action",
			player: &Player{X: 25, Y: 10, Action: NoAction, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX:      25,
			expectedY:      10,
			expectedPlayer: &Player{X: 25, Y: 10, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because no action was taken
		},
		{
			name:   "Invalid Action",
			player: &Player{X: 25, Y: 10, Action: actionType(999), Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX:      25,
			expectedY:      10,
			expectedPlayer: &Player{X: 25, Y: 10, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the action was invalid
		},
		{
			name:   "Wall collision left",
			player: &Player{X: 36, Y: 15, Action: moveLeft, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(36, 15), Tracker: ChangeTracker{}},
			expectedX:      36,
			expectedY:      15,
			expectedPlayer: &Player{X: 36, Y: 15, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
		{
			name:   "Wall collision right",
			player: &Player{X: 34, Y: 15, Action: moveRight, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(34, 15), Tracker: ChangeTracker{}},
			expectedX:      34,
			expectedY:      15,
			expectedPlayer: &Player{X: 34, Y: 15, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
		{
			name:   "Wall collision up",
			player: &Player{X: 35, Y: 16, Action: moveUp, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(35, 16), Tracker: ChangeTracker{}},
			expectedX:      35,
			expectedY:      16,
			expectedPlayer: &Player{X: 35, Y: 16, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
		{
			name:   "Wall collision down",
			player: &Player{X: 35, Y: 14, Action: moveDown, Number: Player1},
			board:  &Board{PastGrid: createBoardWithPlayer(35, 14), Tracker: ChangeTracker{}},
			expectedX:      35,
			expectedY:      14,
			expectedPlayer: &Player{X: 35, Y: 14, Number: Player1},
			expectedDeltas: []Delta{}, // No deltas because the player didn't move
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the tracker before each test
			tt.board.Tracker.Reset()

			// Perform the move
			tt.player.Move(tt.board)

			// Check the player's position
			if tt.player.X != tt.expectedX || tt.player.Y != tt.expectedY {
				t.Errorf("Move() = %v, %v; want %v, %v", tt.player.X, tt.player.Y, tt.expectedX, tt.expectedY)
			}

			// Check the deltas in the tracker
			actualDeltas := tt.board.Tracker.GetDeltas()
			if !compareDeltas(actualDeltas, tt.expectedDeltas) {
				t.Errorf("Deltas = %v; want %v", actualDeltas, tt.expectedDeltas)
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
		name      string
		player    *Player
		board     *Board
		flag      *Flag
		expectedX int
		expectedY int
	}{
		{
			name:      "Flag follow up",
			player:    &Player{X: 30, Y: 12, Action: moveUp, Number: Player1, HasFlag: true, Flag: flag},
			board:     &Board{PastGrid: createBoardWithPlayer(30, 12)},
			flag:      flag,
			expectedX: 30,
			expectedY: 12,
		},
		{
			name:      "Flag follow down",
			player:    &Player{X: 30, Y: 14, Action: moveDown, Number: Player1, HasFlag: true, Flag: flag},
			board:     &Board{PastGrid: createBoardWithPlayer(30, 14)},
			flag:      flag,
			expectedX: 30,
			expectedY: 14,
		},
		{
			name:      "Flag follow left",
			player:    &Player{X: 29, Y: 13, Action: moveLeft, Number: Player1, HasFlag: true, Flag: flag},
			board:     &Board{PastGrid: createBoardWithPlayer(29, 13)},
			flag:      flag,
			expectedX: 29,
			expectedY: 13,
		},
		{
			name:      "Flag follow right",
			player:    &Player{X: 31, Y: 13, Action: moveRight, Number: Player1, HasFlag: true, Flag: flag},
			board:     &Board{PastGrid: createBoardWithPlayer(31, 13)},
			flag:      flag,
			expectedX: 31,
			expectedY: 13,
		},
		{
			name:      "Flag follow up with wall",
			player:    &Player{X: 35, Y: 16, Action: moveUp, Number: Player1, HasFlag: true, Flag: flagWall},
			board:     &Board{PastGrid: createBoardWithPlayer(35, 16)},
			flag:      flagWall,
			expectedX: 35,
			expectedY: 17,
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

func compareDeltas(a, b []Delta) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
