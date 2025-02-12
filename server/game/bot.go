package game

import "math/rand"

type State int 
const (
  random State = iota
  search
  capture
  defend
)



type Bot struct {
  Player
  Path [][2]int
  status State
}

func (b *Bot) BotTurn (tick int, board *Board) bool {
  switch b.status {
  case random:
    return b.RandomAction(tick, board)
  }
  return false
}

func (b *Bot) CalculatePath(board *Board) {
}

func (b *Bot) RandomAction (tick int, board *Board) bool {
  if tick % 3 == 0 {
    action := rand.Intn(7)
    b.Action = actionType(action)
    return b.TakeAction(board)
  }
  b.Action = NoAction
  return b.TakeAction(board)
}

