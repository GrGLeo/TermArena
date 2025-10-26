package model

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/GrGLeo/TermArena/client/communication"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GameModel struct {
	currentBoard    [21][51]int
	conn            *net.TCPConn
	gameClock       time.Duration
	height, width   int
	healthProgress  progress.Model
	manaProgress    progress.Model
	xpProgress      progress.Model
	castingProgress progress.Model
	health          [2]int
	mana            [2]int
	level           int
	xp              [2]int
	casting         [2]int
	attackMode      bool
	recall          bool
	recallDuration  time.Duration
	recallStart     time.Time
	percent         float64
	targetRow       int
	targetCol       int
	targetHealth    [2]int
	targetMana      [2]int
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
		conn:            conn,
		healthProgress:  progress.New(redSolid),
		manaProgress:    progress.New(blueSolid),
		xpProgress:      progress.New(purpleSolid),
		castingProgress: progress.New(yellowGradient),
		recallDuration:  6 * time.Second,
	}
}

func (m GameModel) Init() tea.Cmd {
	return nil
}

func (m *GameModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.castingProgress.Width = 51
}

func (m *GameModel) SetConnection(conn *net.TCPConn) {
	m.conn = conn
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case communication.BoardMsg:
		m.casting = msg.Casting
		m.health = msg.Health
		m.mana = msg.Mana
		m.level = msg.Level
		m.xp = msg.Xp
		m.targetRow = msg.TargetRow
		m.targetCol = msg.TargetCol
		m.targetHealth = msg.TargetHealth
		m.targetMana = msg.TargetMana
		log.Printf("Received target position: row=%d, col=%d", msg.TargetRow, msg.TargetCol)
		m.currentBoard = msg.Board
	case communication.DeltaMsg:
		m.gameClock = time.Duration(50*int(msg.TickID)) * time.Millisecond
		points := msg.Points
		m.casting = points
		ApplyDeltas(msg.Deltas, &m.currentBoard)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "w":
			// Move up
			communication.SendAction(m.conn, 1)
			return m, nil
		case "s":
			// Move down
			communication.SendAction(m.conn, 2)
			return m, nil
		case "a":
			// Move left
			communication.SendAction(m.conn, 3)
			return m, nil
		case "d":
			// Move right
			communication.SendAction(m.conn, 4)
			return m, nil
		case "q":
			// Cast spell 1
			communication.SendAction(m.conn, 5)
			return m, nil
		case "e":
			// Cast spell 2
			communication.SendAction(m.conn, 6)
			return m, nil
		case "v":
			// Champion only target
			if m.attackMode {
				m.attackMode = false
			} else {
				m.attackMode = true
			}
			communication.SendAction(m.conn, 7)
			return m, nil
		case "b":
			// Recall action
			m.recall = true
			m.recallStart = time.Now()
			communication.SendAction(m.conn, 8)
		case "tab":
			// Cycle target
			communication.SendAction(m.conn, 9)
			return m, nil
		case "esc":
			// Clear target
			communication.SendAction(m.conn, 10)
			return m, nil
		case "p":
			// toggle shop
			communication.SendShopRequest(m.conn)
			return m, nil
		case "ctrl+c":
			// quit game
			return m, tea.Quit
		}
	case communication.CooldownTickMsg:
		var percent float64
		if m.recall {
			elapsed := time.Since(m.recallStart)
			percent = float64(elapsed) / float64(m.recallDuration)
			if percent >= 1.0 {
				percent = 0
				m.recall = false
			}
		}
		m.percent = percent
		return m, doTick()
	}
	return m, nil
}

func (m GameModel) View() string {
	log.Printf("Player Health: %d | %d\n", m.health[0], m.health[1])
	// Define styles
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
	fgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("#5665B8"))
	fogStyle := lipgloss.NewStyle().Background(lipgloss.Color("235"))
	blueTeamStyle := lipgloss.NewStyle().Background(lipgloss.Color("4"))
	redTeamStyle := lipgloss.NewStyle().Background(lipgloss.Color("1"))
	baseBlueStyle := lipgloss.NewStyle().Background(lipgloss.Color("21"))
	baseRedStyle := lipgloss.NewStyle().Background(lipgloss.Color("196"))
	monsterStyle := lipgloss.NewStyle().Background(lipgloss.Color("208"))
	towerDest := lipgloss.NewStyle().Background(lipgloss.Color("91"))
	bushStyle := lipgloss.NewStyle().Background(lipgloss.Color("34"))
	grayStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
	freezeStyle := lipgloss.NewStyle().Background(lipgloss.Color("39"))
	healStyle := lipgloss.NewStyle().Background(lipgloss.Color("30"))

	BluePointStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("21"))
	RedPointStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("34"))
	HudStyle := lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("0"))

	var minionHealthChars = []string{"⡀", "⣀", "⣄", "⣤", "⣦", "⣶", "⣷", "⣿"} // 1/8 to 8/8 health

	var builder strings.Builder

	// Construct score board
	bluePoints := strconv.Itoa(m.casting[0])
	redPoints := strconv.Itoa(m.casting[1])
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
	for rowIdx, row := range m.currentBoard {
		for colIdx, cell := range row {
			var style lipgloss.Style
			var char string
			switch cell {
			case 0:
				style = grayStyle
				char = " "
			case 1:
				style = bgStyle
				char = " "
			case 2:
				style = fogStyle
				char = " "
			case 3:
				style = bushStyle
				char = " "
			case 4:
				style = blueTeamStyle
				char = " "
			case 5:
				style = redTeamStyle
				char = " "
			case 6:
				style = bgStyle
				char = "⍓"
			case 7:
				style = towerDest
				char = " "
			case 8:
				style = baseBlueStyle
				char = " "
			case 9:
				style = baseRedStyle
				char = " "
			case 10:
				style = monsterStyle
				char = " "
			case 11:
				style = fgStyle
				char = "x"
			case 12:
				style = fgStyle
				char = "+"
			case 13:
				style = bgStyle
				char = "𐙢"
			case 14:
				style = freezeStyle
				char = "𐙂"
			case 15:
				style = bgStyle
				char = "𐁙"
			case 16:
				style = healStyle
				char = "𐫱"
			case 100, 101, 102, 103, 104, 105, 106, 107: // Friendly minion health (1/8 to 8/8)
				healthIndex := cell - 100
				style = blueTeamStyle
				char = minionHealthChars[healthIndex]
			case 108, 109, 110, 111, 112, 113, 114, 115: // Enemy minion health (1/8 to 8/8)
				healthIndex := cell - 108
				style = redTeamStyle
				char = minionHealthChars[healthIndex]
			}
			if rowIdx == m.targetRow && colIdx == m.targetCol && m.targetRow >= 0 && m.targetCol >= 0 && m.targetRow < 65535 && m.targetCol < 65535 {
				char = "X"
			}
			cellStr := style.Render(char)
			builder.WriteString(cellStr)
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
	if m.mana[1] > 0 {
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

	builder.WriteString("Target Information\n")

	var targetHealthBar string
	if m.targetHealth[1] > 0 {
		targetHealthPercent := (float32(m.targetHealth[0]) / float32(m.targetHealth[1]))
		targetHealthBar = m.healthProgress.ViewAs(float64(targetHealthPercent))
	}
	targetHealthInfo := fmt.Sprintf("%d / %d", m.targetHealth[0], m.targetHealth[1])
	targetHealthHUD := lipgloss.JoinHorizontal(
		lipgloss.Right,
		targetHealthInfo,
		targetHealthBar,
	)
	builder.WriteString(targetHealthHUD)
	builder.WriteString("\n")

	var targetManaBar string
	if m.targetMana[1] > 0 {
		targetManaPercent := (float32(m.targetMana[0]) / float32(m.targetMana[1]))
		targetManaBar = m.manaProgress.ViewAs(float64(targetManaPercent))
	}
	targetManaInfo := fmt.Sprintf("%d / %d", m.targetMana[0], m.targetMana[1])
	targetManaHUD := lipgloss.JoinHorizontal(
		lipgloss.Right,
		targetManaInfo,
		targetManaBar,
	)
	builder.WriteString(targetManaHUD)
	builder.WriteString("\n")

	var castBar string
	if m.casting[1] > 0 {
		castPercent := min(float64(m.casting[0])/float64(m.casting[1]), 1.0)
		castBar = m.castingProgress.ViewAs(castPercent)
		builder.WriteString(castBar)
		builder.WriteString("\n")
	} else {
		builder.WriteString("\n")
	}
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
