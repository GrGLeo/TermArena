package model

import (
	"strings"
	"time"

	"github.com/GrGLeo/ctf/client/communication"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Define some styles
	cellStyle1 = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // Red color
	cellStyle2 = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")) // Green color
	cellStyle0 = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")) // White color
	borderStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder(), false, false, true, false).Align(lipgloss.Center, lipgloss.Center).Padding(1, 1, 1, 1)
)

type AnimationModel struct {
	Frames     [][][]int
	FrameIndex int
  height, width int
}

func NewAnimationModel() AnimationModel {
	return AnimationModel{
		Frames:     animationFrames,
		FrameIndex: 0,
	}
}

func (m *AnimationModel) SetDimension(height, width int) {
  m.height = height
  m.width = width
}

func (m AnimationModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		return communication.TickMsg{Time: t}
	})
}

func (m AnimationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case communication.TickMsg:
		m.FrameIndex = (m.FrameIndex + 1) % len(m.Frames)
		return m, tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
			return communication.TickMsg{Time: t}
		})
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AnimationModel) View() string {
	frame := m.Frames[m.FrameIndex]
	var view strings.Builder

  view.WriteString("CaptureTheFlag\n")
	for _, row := range frame {
		for _, cell := range row {
			switch cell {
			case 1:
				view.WriteString(cellStyle1.Render("█"))
			case 2:
				view.WriteString(cellStyle2.Render("█"))
			default:
				view.WriteString(cellStyle0.Render(" "))
			}
		}
		view.WriteString("\n")
	}
  view.WriteString("Press Enter to pass")
  return lipgloss.Place(
    m.width * 2,
    m.height / 2,
    lipgloss.Center,
    lipgloss.Center,
    borderStyle.Render(view.String()),
  )
}
