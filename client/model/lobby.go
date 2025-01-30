package model

import (
	"log"
	"net"
	"strings"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LobbyModel struct {
	options       []string
	selected      int
	width, height int
	looking       bool
	conn          *net.TCPConn
	spinner       spinner.Model
}

func NewLobbyModel(conn *net.TCPConn) LobbyModel {
  sp := spinner.New()
  sp.Spinner = spinner.Dot
  sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return LobbyModel{
		options:  []string{"Solo (1 player 1 bot vs 2 bots)", "2 players vs 2 bots", "2 players vs 2 players"},
		selected: 0,
		conn:     conn,
    spinner: sp,
	}
}

func (m *LobbyModel) SetConn(conn *net.TCPConn) {
  m.conn = conn
}

func (m *LobbyModel) SetLooking(search bool) {
  m.looking = search
}

func (m *LobbyModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func (m LobbyModel) Init() tea.Cmd {
	return nil
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case communication.LookRoomMsg:
		m.looking = true
    return m, m.spinner.Tick
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			selectedOption := m.options[m.selected]
			communication.SendRoomRequestPacket(m.conn, m.selected)
			log.Println("Selected option:", selectedOption)
			return m, nil
		}

		switch msg.String() {
		case "k":
			if m.selected == 0 {
				m.selected = len(m.options) - 1
			} else {
				m.selected--
			}
			return m, nil
		case "j":
			if m.selected == len(m.options)-1 {
				m.selected = 0
			} else {
				m.selected++
			}
			return m, nil
		}
	}

  if m.looking {
    m.spinner, cmd = m.spinner.Update(msg)
  }
	return m, cmd
}

func (m LobbyModel) View() string {
	// Build the options list
	var optionsBuilder strings.Builder
	selectedChar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Render("> ")

	for i, opt := range m.options {
		if m.selected == i {
			optionsBuilder.WriteString(selectedChar)
			optionsBuilder.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("205")).
					Bold(true).
					Render(opt),
			)
		} else {
			optionsBuilder.WriteString("  ")
			optionsBuilder.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("240")).
					Render(opt),
			)
		}
		optionsBuilder.WriteString("\n")
	}

	// Game instructions
	gameInstruction := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Render(
			"Capture the flag is a 2v2 multiplayer game.\n" +
				"Each team needs to capture the enemy flag and bring it back to their own base.\n" +
				"Player can move around the map using w,a,s,d.\n" +
				"Player can dash using space. Dash moves the player instantly by a short distance.\n" +
				"Dash passes through walls.",
		)

	optionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(1, 2)

	instructionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false, true).
		Padding(1, 2)
    
    if m.looking {
		optionsBuilder.WriteString("\n\n")
		optionsBuilder.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Render("Looking for a room... " + m.spinner.View()),
		)
	}

	layout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		optionsStyle.Render(optionsBuilder.String()),
		instructionsStyle.Render(gameInstruction),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Padding(2, 4).
			Render(layout),
	)
}
