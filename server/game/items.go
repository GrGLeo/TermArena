package game

type Flag struct {
	TeamId     int  `json:"teamID"`
	PosX       int  `json:"posx"`
	PosY       int  `json:"posy"`
	IsCaptured bool `json:"is_captured"`
}



type WallPosition struct {
  StartPos [2]int // Y and X start position on the borad
  EndPos [2]int // Y and X end position on the board
}

// Return Y and X start position
func (w WallPosition) GetStartPos() (int, int) {
  return w.StartPos[0], w.StartPos[1]
}
  
// Return Y and X end position
func (w WallPosition) GetEndPos() (int, int) {
  return w.EndPos[0], w.EndPos[1]
}


