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


func TestMakeFreeze(t *testing.T) {
	tests := []struct {
		name           string
		player         *Player
		board          *Board
		expectedSprites []*FreezeSprite
	}{
		{
			name:      "Freeze Up",
			player:    createPlayer(25, 10, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				{X: 24, Y: 9, Facing: Up},
				{X: 25, Y: 9, Facing: Up},
				{X: 26, Y: 9, Facing: Up},
			},
		},
    {
			name:      "Freeze Up at border",
			player:    createPlayer(0, 10, Up),
			board:     &Board{PastGrid: createBoardWithPlayer(0, 10), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				{X: 0, Y: 9, Facing: Up},
				{X: 1, Y: 9, Facing: Up},
			},
		},
		{
			name:      "Freeze Down",
			player:    createPlayer(25, 10, Down),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				{X: 24, Y: 11, Facing: Down},
				{X: 25, Y: 11, Facing: Down},
				{X: 26, Y: 11, Facing: Down},
			},
		},
		{
			name:      "Freeze Left",
			player:    createPlayer(25, 10, Left),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				{X: 24, Y: 9, Facing: Left},
				{X: 24, Y: 10, Facing: Left},
				{X: 24, Y: 11, Facing: Left},
			},
		},
		{
			name:      "Freeze Right",
			player:    createPlayer(25, 10, Right),
			board:     &Board{PastGrid: createBoardWithPlayer(25, 10), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				{X: 26, Y: 9, Facing: Right},
				{X: 26, Y: 10, Facing: Right},
				{X: 26, Y: 11, Facing: Right},
			},
		},
		{
			name:      "Freeze Out of Bounds",
			player:    createPlayer(0, 0, Left),
			board:     &Board{PastGrid: createBoardWithPlayer(0, 0), Tracker: ChangeTracker{}},
			expectedSprites: []*FreezeSprite{
				// No sprites should be created because the player is at the edge
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.board.Sprite = nil

			tt.player.MakeFreeze(tt.board)

			if len(tt.board.Sprite) != len(tt.expectedSprites) {
				t.Errorf("Expected %d sprites, but got %d", len(tt.expectedSprites), len(tt.board.Sprite))
			}

			for i, expectedSprite := range tt.expectedSprites {
				actualSprite := tt.board.Sprite[i]

				sprite, ok := actualSprite.(*FreezeSprite)
				if !ok {
					t.Errorf("Sprite at index %d is not of type *FreezeSprite, got %T", i, actualSprite)
					continue
				}
				if sprite.X != expectedSprite.X || sprite.Y != expectedSprite.Y || sprite.Facing != expectedSprite.Facing {
					t.Errorf("Sprite mismatch at index %d: expected %+v, got %+v", i, expectedSprite, sprite)
				}
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
