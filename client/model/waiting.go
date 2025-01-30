package model

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WaitingModel struct {
  height, width int
  spinner spinner.Model
  retry bool
}

func (m *WaitingModel) SetDimension(height, width int) {
  m.height = height
  m.width = width
}

func (m WaitingModel) Init() tea.Cmd {
  return m.spinner.Tick
}

func NewWaitingModel() WaitingModel {
  s := spinner.New()
  s.Spinner = spinner.Dot
  s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("256"))
  return WaitingModel{spinner: s}
}

func (m WaitingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmd tea.Cmd
  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlC, tea.KeyEsc:
      return m, tea.Quit
    default:
      return m, nil
    }

  default:
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
  }
}

func (m WaitingModel) View() string {
	str := fmt.Sprintf("\n\n   %s Trying to reach server. ...press Esc/Ctrl+c to quit\n\n", m.spinner.View())

	return lipgloss.Place(
    m.width,
    m.height,
    lipgloss.Center,
    lipgloss.Center,
    str,
  )
}
