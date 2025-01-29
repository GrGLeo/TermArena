package model

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GameModel struct {
	currentBoard  [20][50]int
	conn          *net.TCPConn
	height, width int
	progress      progress.Model
	dashed        bool
	dashcooldown  time.Duration
	dashStart     time.Time
	percent       float64
}

func NewGameModel(conn *net.TCPConn) GameModel {
  yellowGradient := progress.WithGradient(
    "#FFFF00", // Bright yellow
    "#FFD700", // Gold
  )
	return GameModel{
		conn:         conn,
		progress:     progress.New(yellowGradient),
		dashcooldown: 5 * time.Second,
	}
}

func (m GameModel) Init() tea.Cmd {
	return nil
}

func (m *GameModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.progress.Width = 50
}

func (m *GameModel) SetConnection(conn *net.TCPConn) {
	m.conn = conn
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case communication.BoardMsg:
		m.currentBoard = msg.Board
	case communication.DeltaMsg:
		ApplyDeltas(msg.Deltas, &m.currentBoard)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
		switch msg.String() {
		case "w":
			communication.SendAction(m.conn, 1)
			return m, nil
		case "s":
			communication.SendAction(m.conn, 2)
			return m, nil
		case "a":
			communication.SendAction(m.conn, 3)
			return m, nil
		case "d":
			communication.SendAction(m.conn, 4)
			return m, nil
		case " ":
			if !m.dashed {
				communication.SendAction(m.conn, 5)
				m.dashed = true
				m.dashStart = time.Now()
				return m, doTick()
			}
		}
	case communication.CooldownTickMsg:
		var percent float64
		if m.dashed {
			elapsed := time.Since(m.dashStart)
			percent = float64(elapsed) / float64(m.dashcooldown)
			log.Println(percent)
			if percent >= 1.0 {
				percent = 0
				m.dashed = false
			}
		}
		m.percent = percent
		return m, doTick()
	}
	return m, nil
}

func (m GameModel) View() string {
	// Define styles
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
	p1Style := lipgloss.NewStyle().Background(lipgloss.Color("21"))
	p2Style := lipgloss.NewStyle().Background(lipgloss.Color("91"))
	p3Style := lipgloss.NewStyle().Background(lipgloss.Color("34"))
	p4Style := lipgloss.NewStyle().Background(lipgloss.Color("220"))
	grayStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
	Flag1Style := lipgloss.NewStyle().Background(lipgloss.Color("201"))
	Flag2Style := lipgloss.NewStyle().Background(lipgloss.Color("94"))

	var builder strings.Builder

	// Iterate through the board and apply styles
	for _, row := range m.currentBoard {
		for _, cell := range row {
			switch cell {
			case 0:
				builder.WriteString(bgStyle.Render(" ")) // Render empty space for 0
			case 1:
				builder.WriteString(grayStyle.Render(" ")) // Render gray for walls
			case 2:
				builder.WriteString(p1Style.Render(" ")) // Render blue for player1
			case 3:
				builder.WriteString(p2Style.Render(" ")) // Render blue for player2
			case 4:
				builder.WriteString(p3Style.Render(" ")) // Render blue for player3
			case 5:
				builder.WriteString(p4Style.Render(" ")) // Render blue for player4
			case 6:
				builder.WriteString(Flag1Style.Render(" ")) // Render for flag1
			case 7:
				builder.WriteString(Flag2Style.Render(" ")) // Render for flag2
			case 8:
				builder.WriteString("⣿") // Render for flag2
			case 9:
				builder.WriteString("⣶") // Render for flag2
			case 10:
				builder.WriteString("⣤") // Render for flag2
			case 11:
				builder.WriteString("⣀") // Render for flag2
			}
		}
		builder.WriteString("\n") // New line at the end of each row
	}

  var progressBar string
  log.Println("progress percent", m.progress.Percent())
  if m.percent != 0.0 {
	  progressBar = m.progress.ViewAs(m.percent)
  }
	builder.WriteString(progressBar)
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		builder.String(),
	)
}
func doTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return communication.CooldownTickMsg{}
	})
}

func ApplyDeltas(deltas [][3]int, currentBoard *[20][50]int) {
	for _, delta := range deltas {
		x := delta[0]
		y := delta[1]
		value := delta[2]
		currentBoard[y][x] = value
	}
}
