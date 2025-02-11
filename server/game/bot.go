package game

import "math/rand"

type Bot struct {
  Player
  Path [][2]int
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
