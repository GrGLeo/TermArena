package model

import (
	"net"
	"strings"

	"github.com/GrGLeo/ctf/client/communication"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GameModel struct {
  board [20][50]int
  conn *net.TCPConn
  height, width int
}

func NewGameModel(conn *net.TCPConn) GameModel {
  return GameModel{
    conn: conn,
  }
}


func (m GameModel) Init() tea.Cmd {
  return nil
}

func (m *GameModel) SetDimension(height, width int) {
  m.height = height
  m.width = width
}

func (m *GameModel) SetConnection (conn *net.TCPConn) {
  m.conn = conn
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case communication.BoardMsg:
    m.board = msg.Board
  case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlC, tea.KeyEsc:
      return m, tea.Quit
    }
    switch msg.String() {
    case "w":
      communication.SendAction(m.conn,0)
      return m, nil
    case "s":
      communication.SendAction(m.conn,1)
      return m, nil
    case "a":
      communication.SendAction(m.conn,2)
      return m, nil
    case "d":
      communication.SendAction(m.conn,3)
      return m, nil
    }
  }
  return m, nil
}

func (m GameModel) View() string {
  // Define styles
  bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
  blueStyle := lipgloss.NewStyle().Background(lipgloss.Color("21"))
  grayStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))

  var builder strings.Builder

  // Iterate through the board and apply styles
  for _, row := range m.board {
    for _, cell := range row {
      switch cell {
      case 0:
        builder.WriteString(bgStyle.Render(" ")) // Render empty space for 0
      case 1:
        builder.WriteString(grayStyle.Render(" ")) // Render gray for 1
      case 2:
        builder.WriteString(blueStyle.Render(" ")) // Render blue for 2
      }
    }
    builder.WriteString("\n") // New line at the end of each row
  }
  return lipgloss.Place(
    m.width,
    m.height,
    lipgloss.Center,
    lipgloss.Center,
    builder.String(),
  )
}
