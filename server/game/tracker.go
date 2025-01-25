package game

import "fmt"


type Delta struct {
	X     int
	Y     int
	Value Cell
}

type ChangeTracker struct {
	Deltas []Delta
}

func (ct *ChangeTracker) SaveDelta(x, y int, value Cell) {
  fmt.Println(x, y, value)
  ct.Deltas = append(ct.Deltas, Delta{X: x, Y: y, Value: value})
}

func (ct *ChangeTracker) GetDeltas() []Delta {
  return ct.Deltas
}

func (ct *ChangeTracker) Reset() {
  ct.Deltas = nil
}
