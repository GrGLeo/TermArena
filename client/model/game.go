package model

import (
	"log"
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

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case communication.BoardMsg:
    m.board = msg.Board
    log.Println(m.board)
  case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlC, tea.KeyEsc:
      return m, tea.Quit
    }
  }
  return m, nil
}

func (m GameModel) View() string {
  // Define styles
  bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
  blueStyle := lipgloss.NewStyle().Background(lipgloss.Color("4"))

  var builder strings.Builder

  // Iterate through the board and apply styles
  for _, row := range m.board {
    for _, cell := range row {
      if cell == 0 {
        builder.WriteString(bgStyle.Render(" ")) // Render empty space for 0
      } else if cell == 2 {
        builder.WriteString(blueStyle.Render(" ")) // Render blue for 1
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
