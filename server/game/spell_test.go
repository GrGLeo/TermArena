package game

import "testing"

func TestDash(t *testing.T) {
	tests := []struct {
		name           string
		player         *Player
		board          *Board
		expectedX      int
		expectedY      int
		expectedDeltas []Delta
	}{
		{
			name:      "Dash Up",
			player:    createPlayer(25, 10, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 25,
			expectedY: 6,
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0},
				{X: 25, Y: 6, Value: 2},
			},
		},
		{
			name:      "Dash Down",
			player:    createPlayer(25, 10, Down),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 25,
			expectedY: 14,
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0},
				{X: 25, Y: 14, Value: 2},
			},
		},
		{
			name:      "Dash Left",
			player:    createPlayer(25, 10, Left),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 21,
			expectedY: 10,
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0},
				{X: 21, Y: 10, Value: 2},
			},
		},
		{
			name:      "Dash Right",
			player:    createPlayer(25, 10, Right),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 29,
			expectedY: 10,
			expectedDeltas: []Delta{
				{X: 25, Y: 10, Value: 0},
				{X: 29, Y: 10, Value: 2},
			},
		},
		{
			name:      "Dash block by Wall",
			player:    createPlayer(35, 19, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 35,
			expectedY: 16,
			expectedDeltas: []Delta{
				{X: 35, Y: 19, Value: 0},
				{X: 35, Y: 16, Value: 2},
			},
		},
		{
			name:      "Dash pass Wall",
			player:    createPlayer(35, 18, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedX: 35,
			expectedY: 14,
			expectedDeltas: []Delta{
				{X: 35, Y: 18, Value: 0},
				{X: 35, Y: 14, Value: 2},
			},
		},
		{
			name:      "Dash out of bound",
			player:    createPlayer(25, 2, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 2), Tracker: ChangeTracker{}},
			expectedX: 25,
			expectedY: 0,
			expectedDeltas: []Delta{
				{X: 25, Y: 2, Value: 0},
				{X: 25, Y: 0, Value: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the tracker before each test
			tt.board.Tracker.Reset()

			// Perform the move
			tt.player.TakeAction(tt.board)

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

func createPlayer(x, y int, direction Direction) *Player {
	return &Player{
		X:      x,
		Y:      y,
		Action: spellOne,
		Facing: direction,
		Number: Player1,
		Dash: Dash{
			Range:    4,
			Cooldown: 5,
		},
	}
}
