package game

import "math/rand"

type Bot struct {
  Player
  Path [][2]int
}

func (b *Bot) CalculatePath(board *Board) {
}

func (b *Bot) RandomAction(board *Board) bool {
  action := rand.Intn(7)
  b.Action = actionType(action)
  return b.TakeAction(board)
}
