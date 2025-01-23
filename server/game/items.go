package game

type Flag struct {
	TeamId     Cell  `json:"teamID"`
	PosX       int  `json:"posx"`
	PosY       int  `json:"posy"`
	baseX      int  
	baseY      int  
	IsCaptured bool `json:"is_captured"`
}

func (f *Flag) Move(x, y int, board *Board) {
	board.Grid[f.PosY][f.PosX] = Empty
	f.PosX = x
	f.PosY = y
	board.Grid[y][x] = f.TeamId
}

func (f *Flag) SetBase() {
  f.baseX = f.PosX
  f.baseY = f.PosY
}

// SetBase need to be called first
func (f *Flag) ResetPos() {
  f.PosX = f.baseX
  f.PosY = f.baseY
}

// Return base flag position Y and X coordinate
func (f *Flag) GetBase() (int, int) {
  return f.baseY, f.baseX
}

type WallPosition struct {
	StartPos [2]int // Y and X start position on the board
	EndPos   [2]int // Y and X end position on the board
}

// Return Y and X start position
func (w WallPosition) GetStartPos() (int, int) {
	return w.StartPos[0], w.StartPos[1]
}

// Return Y and X end position
func (w WallPosition) GetEndPos() (int, int) {
	return w.EndPos[0], w.EndPos[1]
}
