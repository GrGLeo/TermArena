package internal

type Spells [2]int

type RoomStatus int

type RoomID uint32

const (
    WAITING  RoomStatus = iota  // Rooms with space (not full) - players can join
    LOBBY                       // Full rooms waiting for 1-minute timer
    READY                       // Rooms ready to start (after timer)
    PROGRESS                    // Rooms in progress (after scan)
)

type RoomType int

const (
  SANDBOX RoomType = iota
  PRACTICE
  CLASSIC
)
