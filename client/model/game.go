package model

import (
	"fmt"
	"log"
	"net"

	"github.com/GrGLeo/ctf/client/communication"
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
  switch msg := msg.(type) {
  case communication.BoardMsg:
    log.Print("Received board")
    m.board = msg.Board
  case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlC, tea.KeyEsc:
      return m, tea.Quit
    }
  }
  return m, nil
}

func (m GameModel) View() string {
  return fmt.Sprintf("%v", m.board)
}
