package model

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GameModel struct {
	currentBoard   [21][51]int
	conn           *net.TCPConn
	gameClock      time.Duration
	height, width  int
	healthProgress progress.Model
	manaProgress   progress.Model
	xpProgress     progress.Model
	progress       progress.Model
	health         [2]int
	mana           [2]int
	level          int
	xp             [2]int
	points         [2]int
	attackMode     bool
	dashed         bool
	dashcooldown   time.Duration
	dashStart      time.Time
	percent        float64
}

func NewGameModel(conn *net.TCPConn) GameModel {
	yellowGradient := progress.WithGradient(
		"#FFFF00", // Bright yellow
		"#FFD700", // Gold
	)
	redSolid := progress.WithSolidFill("#AB2C0F")
	blueSolid := progress.WithSolidFill("#3E84D4")
	purpleSolid := progress.WithSolidFill("#A51CC4")
	return GameModel{
		conn:           conn,
		healthProgress: progress.New(redSolid),
		manaProgress:   progress.New(blueSolid),
		xpProgress:     progress.New(purpleSolid),
		progress:       progress.New(yellowGradient),
		dashcooldown:   5 * time.Second,
	}
}

func (m GameModel) Init() tea.Cmd {
	return nil
}

func (m *GameModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.progress.Width = 51
}

func (m *GameModel) SetConnection(conn *net.TCPConn) {
	m.conn = conn
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case communication.BoardMsg:
		points := msg.Points
		m.points = points
		m.health = msg.Health
		m.mana = msg.Mana
		m.level = msg.Level
		m.xp = msg.Xp
		m.currentBoard = msg.Board
	case communication.DeltaMsg:
		m.gameClock = time.Duration(50*int(msg.TickID)) * time.Millisecond
		points := msg.Points
		m.points = points
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
		case "q":
			communication.SendAction(m.conn, 5)
			return m, nil
		case "e":
			communication.SendAction(m.conn, 6)
			return m, nil
		case "v":
      if m.attackMode {
        m.attackMode = false
      } else {
        m.attackMode = true
      }
			communication.SendAction(m.conn, 7)
			return m, nil
		}
	case communication.CooldownTickMsg:
		var percent float64
		if m.dashed {
			elapsed := time.Since(m.dashStart)
			percent = float64(elapsed) / float64(m.dashcooldown)
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
	log.Println(m.points)
	log.Printf("Player Health: %d | %d\n", m.health[0], m.health[1])
	// Define styles
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
	p1Style := lipgloss.NewStyle().Background(lipgloss.Color("21"))
	TowerDest := lipgloss.NewStyle().Background(lipgloss.Color("91"))
	bushStyle := lipgloss.NewStyle().Background(lipgloss.Color("34"))
	p4Style := lipgloss.NewStyle().Background(lipgloss.Color("1"))
	grayStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
	Flag1Style := lipgloss.NewStyle().Background(lipgloss.Color("201"))
	TowerStyle := lipgloss.NewStyle().Background(lipgloss.Color("94"))
	FreezeStyle := lipgloss.NewStyle().Background(lipgloss.Color("39"))
	BaseBlueStyle := lipgloss.NewStyle().Background(lipgloss.Color("21"))
	BaseRedStyle := lipgloss.NewStyle().Background(lipgloss.Color("196"))

	BluePointStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("21"))
	RedPointStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("34"))
	HudStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("0"))

	var minionHealthChars = []string{"⡀", "⣀", "⣄", "⣤", "⣦", "⣶", "⣷", "⣿"} // 1/8 to 8/8 health

	var builder strings.Builder

	// Construct score board
	bluePoints := strconv.Itoa(m.points[0])
	redPoints := strconv.Itoa(m.points[1])
	blueStr := BluePointStyle.Render(bluePoints)
	redStr := RedPointStyle.Render(redPoints)
	splitStr := HudStyle.Render(" | ")
	scoreText := HudStyle.Render(blueStr + splitStr + redStr)

	minutes := int(m.gameClock.Minutes())
	seconds := int(m.gameClock.Seconds()) % 60
	clockStr := HudStyle.Render(fmt.Sprintf("%02d:%02d", minutes, seconds))

	hud := lipgloss.Place(
		46,
		1,
		lipgloss.Center,
		lipgloss.Center,
		scoreText,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(HudStyle.GetBackground()),
	)

	hudContent := lipgloss.JoinHorizontal(lipgloss.Right, hud, clockStr)
	hudContent += "\n"
	builder.WriteString(hudContent)

	// Iterate through the board and apply styles
	for _, row := range m.currentBoard {
		for _, cell := range row {
			switch cell {
			case 0:
				builder.WriteString(grayStyle.Render(" ")) // Render gray for walls
			case 1:
				builder.WriteString(bgStyle.Render(" ")) // Render empty space for 1
			case 2:
				builder.WriteString(bushStyle.Render(" ")) // Render green for bush
			case 3:
				builder.WriteString(TowerDest.Render(" ")) // Render blue for player2
			case 4:
				builder.WriteString(p1Style.Render(" ")) // Render blue for player1
			case 5:
				builder.WriteString(p4Style.Render(" ")) // Render blue for player4
			case 6:
				builder.WriteString(Flag1Style.Render(" ")) // Render for flag1
			case 7:
				builder.WriteString(TowerStyle.Render(" ")) // Render for tower
			case 8:
				builder.WriteString(bgStyle.Render("⍓")) // Render for dash
			case 9:
				builder.WriteString(bgStyle.Render("x")) // Render for dash
			case 10:
				builder.WriteString(bgStyle.Render("𐙢")) // Render for dash
			case 11:
				builder.WriteString(BaseBlueStyle.Render(" ")) // Render for base blue
			case 12:
				builder.WriteString(BaseRedStyle.Render(" ")) // Render for base red
			case 13:
				builder.WriteString(bgStyle.Render("⣀")) // Render for dash
			case 14:
				builder.WriteString(FreezeStyle.Render("𐙂")) // Render for freezing spell
			case 15:
				builder.WriteString(bgStyle.Render("𐁙")) // Render for freezing spell
			case 100, 101, 102, 103, 104, 105, 106, 107: // Friendly minion health (1/8 to 8/8)
				healthIndex := cell - 100
				builder.WriteString(p1Style.Render(minionHealthChars[healthIndex]))
			case 108, 109, 110, 111, 112, 113, 114, 115: // Enemy minion health (1/8 to 8/8)
				healthIndex := cell - 108
				builder.WriteString(p4Style.Render(minionHealthChars[healthIndex]))
			}
		}
		builder.WriteString("\n") // New line at the end of each row
	}


	var healthBar string
	if m.health[1] > 0 {
		healthPercent := (float32(m.health[0]) / float32(m.health[1]))
		healthBar = m.healthProgress.ViewAs(float64(healthPercent))
	}
	healthInfo := fmt.Sprintf("%d / %d", m.health[0], m.health[1])
	healthHUD := lipgloss.JoinHorizontal(
		lipgloss.Right,
		healthInfo,
		healthBar,
	)
	builder.WriteString(healthHUD)
	builder.WriteString("\n")

	var manaBar string
	if m.health[1] > 0 {
		manaPercent := (float32(m.mana[0]) / float32(m.mana[1]))
		manaBar = m.manaProgress.ViewAs(float64(manaPercent))
	}
	manaInfo := fmt.Sprintf("%d / %d", m.mana[0], m.mana[1])
	manaHUD := lipgloss.JoinHorizontal(
		lipgloss.Right,
		manaInfo,
		manaBar,
	)
	builder.WriteString(manaHUD)
	builder.WriteString("\n")

	var xpBar string
	if m.xp[1] > 0 {
		xpPercent := (float32(m.xp[0]) / float32(m.xp[1]))
		xpBar = m.xpProgress.ViewAs(float64(xpPercent))
	}
	xpInfo := fmt.Sprintf("Lvl %d: %d / %d", m.level, m.xp[0], m.xp[1])
	xpHUD := lipgloss.JoinHorizontal(
		lipgloss.Right,
		xpInfo,
		xpBar,
	)
	builder.WriteString(xpHUD)
	builder.WriteString("\n")

	var progressBar string
	if m.percent != 0.0 {
		progressBar = m.progress.ViewAs(m.percent)
	}
	builder.WriteString(progressBar)
  gameStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), m.attackMode).BorderForeground(lipgloss.Color("#ff0000"))




	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		gameStyle.Render(builder.String()),
	)
}

func doTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return communication.CooldownTickMsg{}
	})
}

func ApplyDeltas(deltas [][3]int, currentBoard *[21][51]int) {
	for _, delta := range deltas {
		x := delta[0]
		y := delta[1]
		value := delta[2]
		currentBoard[y][x] = value
	}
}
