package model

import (
	"net"

	tea "github.com/charmbracelet/bubbletea"
)

type GameModel struct {
  board [20][50]int
  conn *net.TCPConn
}

func NewGameModel(conn *net.TCPConn) GameModel {
  return GameModel{
    conn: conn,
  }
}


func (m GameModel) Init() tea.Cmd {
  return nil
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  return m, nil
}

func (m GameModel) View() string {
  return ""
}
