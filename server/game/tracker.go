package game


type Delta struct {
	X     int
	Y     int
	Value Cell
}

type ChangeTracker struct {
	Deltas []Delta
}

func (ct *ChangeTracker) SaveDelta(x, y int, value Cell) {
  ct.Deltas = append(ct.Deltas, Delta{X: x, Y: y, Value: value})
}

func (ct *ChangeTracker) GetDeltas() []Delta {
  return ct.Deltas
}

func (ct *ChangeTracker) GetDeltasByte() [][3]byte {
  var deltasByte [][3]byte
  for _, delta := range ct.Deltas {
    deltaByte := [3]byte{byte(delta.X), byte(delta.Y), byte(delta.Value)}
    deltasByte = append(deltasByte, deltaByte)
  }
  return deltasByte
}


func (ct *ChangeTracker) Reset() {
  ct.Deltas = nil
}
