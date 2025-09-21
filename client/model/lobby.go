package model

import (
	"log"
	"net"
	"strings"

	"github.com/GrGLeo/TermArena/client/communication"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LobbyModel struct {
	styles              *Styles
	tabSelected         int
	queueModel          QueueModel
	createModel         CreateModel
	spellSelectionModel SpellSelectionModel
	messagingModel      MessagingModel
	conn                *net.TCPConn
	looking             bool
	width, height       int
	SelectedSpells      [2]int
	Username            string
}

func NewLobbyModel(conn *net.TCPConn, username string) LobbyModel {
	queueModel := NewQueueModel(conn)
	createModel := NewCreateModel(conn)
	s := DefaultStyles()
	spellSelectionModel := NewSpellSelection(s)
	messagingModel := NewMessagingModel(conn, username)

	return LobbyModel{
		styles:              s,
		tabSelected:         0,
		queueModel:          queueModel,
		createModel:         createModel,
		spellSelectionModel: spellSelectionModel,
		messagingModel:      messagingModel,
		conn:                conn,
		Username:            username,
	}
}

func (m *LobbyModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
	m.queueModel.SetConn(conn)
	m.createModel.SetConn(conn)
	m.messagingModel.conn = conn
}

func (m *LobbyModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.queueModel.SetDimension(height, width)
	m.createModel.SetDimension(height, width)
	m.spellSelectionModel.SetDimension(height, width)
	m.messagingModel.SetDimension(width, height)
}

func (m *LobbyModel) SetLooking(search bool) {
	m.looking = search
}

func (m LobbyModel) Init() tea.Cmd {
	return m.spellSelectionModel.Init()
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.tabSelected = (m.tabSelected - 1 + 4) % 4
		case "right":
			m.tabSelected = (m.tabSelected + 1) % 4
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
	case SpellsSelectedMsg:
		m.SelectedSpells = msg.SpellIDs
		log.Printf("LobbyModel received selected spells: %v", m.SelectedSpells)
	case TabLeftMsg:
		m.tabSelected = (m.tabSelected - 1 + 4) % 4
	case TabRightMsg:
		m.tabSelected = (m.tabSelected + 1) % 4
	case communication.IncomingMessageMsg, communication.MessageErrorMsg:
		// Always handle incoming messages regardless of current tab
		var mm tea.Model
		mm, cmd = m.messagingModel.Update(msg)
		m.messagingModel = mm.(MessagingModel)
		return m, cmd
	}

	if m.tabSelected == 0 {
		var ssm tea.Model
		ssm, cmd = m.spellSelectionModel.Update(msg)
		m.spellSelectionModel = ssm.(SpellSelectionModel)
	} else if m.tabSelected == 1 {
		var qm tea.Model
		qm, cmd = m.queueModel.Update(msg)
		m.queueModel = qm.(QueueModel)
	} else if m.tabSelected == 2 {
		var cm tea.Model
		cm, cmd = m.createModel.Update(msg)
		m.createModel = cm.(CreateModel)
	} else if m.tabSelected == 3 {
		var mm tea.Model
		mm, cmd = m.messagingModel.Update(msg)
		m.messagingModel = mm.(MessagingModel)
	}

	return m, cmd
}

func (m LobbyModel) View() string {
	var renderedTabs []string
	var content string

	// Render Tabs based on selection
	spellSelectionTabStr := "Spell Selection"
	joinGameTabStr := "Join a game"
	createGameTabStr := "Create a game"
	messagingTabStr := "Messaging"

	tabs := []string{spellSelectionTabStr, joinGameTabStr, createGameTabStr, messagingTabStr}

	for i, tab := range tabs {
		if i == m.tabSelected {
			renderedTabs = append(renderedTabs, m.styles.ActiveTab.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, m.styles.InactiveTab.Render(tab))
		}
	}

	if m.tabSelected == 0 {
		content = m.spellSelectionModel.View()
	} else if m.tabSelected == 1 {
		content = m.queueModel.View()
	} else if m.tabSelected == 2 {
		content = m.createModel.View()
	} else if m.tabSelected == 3 {
		content = m.messagingModel.View()
	}

	// Join the individual tab strings horizontally
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Create the tab bar container with a bottom border to act as the underline
	// The border will automatically be the width of the tabRow
	tabBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(m.styles.BorderColor).
		Render(tabRow)

	// Combine the tab bar and the content vertically
	ui := lipgloss.JoinVertical(lipgloss.Center,
		tabBar,
		content,
	)

	// Place the entire UI block in the center of the available space
	finalView := lipgloss.NewStyle().Render(ui)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		finalView,
	)
}

type QueueModel struct {
	options       []string
	selected      int
	width, height int
	looking       bool
	conn          *net.TCPConn
	spinner       spinner.Model
}

func NewQueueModel(conn *net.TCPConn) QueueModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return QueueModel{
		options:  []string{"Sandbox (solo)", "Practice (2v2)", "Classic (4v4)", "Ranked (4v4)"},
		selected: 0,
		conn:     conn,
		spinner:  sp,
	}
}

func (m *QueueModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
}

func (m *QueueModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func (m QueueModel) Init() tea.Cmd {
	return nil
}

func (m QueueModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.selected = (m.selected - 1 + len(m.options)) % len(m.options)
			return m, nil
		case "j":
			m.selected = (m.selected + 1) % len(m.options)
			return m, nil
		}
	}

	if m.looking {
		m.spinner, cmd = m.spinner.Update(msg)
	}
	return m, cmd
}

func (m QueueModel) View() string {
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

	gameInstruction := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Render(
			"TermArena is a MOBA game. Each team needs to destroy the enemy base.\n" +
				"To progress on the map you will need to beat minions, destroy tower, fight enemy champion.\n" +
        "Each 30 seconds minions are spawn, they will follow there respective lane.\n" +
        "Jungle monster respawn after 90 seconds.\n" +
				"Player can move around the map using w,a,s,d.\n" +
				"Spell selected are bind to q and e.",
		)

	optionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Padding(1, 0)

	instructionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		Padding(1, 0)

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

	return lipgloss.JoinVertical(
		lipgloss.Top,
		layout,
	)

}

type CreateModel struct {
	options       []string
	selected      int
	roomIDInput   textinput.Model
	width, height int
	looking       bool
	roomID        uint32
	conn          *net.TCPConn
	spinner       spinner.Model
	inputFocused  bool
}

func NewCreateModel(conn *net.TCPConn) CreateModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter Room ID"
	ti.Width = 20
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return CreateModel{
		options:     []string{"2 players vs 2 bots", "2 players vs 2 players"},
		selected:    0,
		conn:        conn,
		spinner:     sp,
		roomIDInput: ti,
	}
}

func (m *CreateModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
}

func (m *CreateModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func (m CreateModel) Init() tea.Cmd {
	return nil
}

func (m CreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case communication.LookRoomMsg:
		m.looking = true
		m.roomID = msg.RoomID
		log.Println(msg.RoomIP)
		return m, m.spinner.Tick
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.inputFocused {
				// Handle room ID input submission
				roomID := m.roomIDInput.Value()
				log.Println("Room ID entered:", roomID)
				communication.SendRoomJoinPacket(m.conn, roomID)
			} else {
				selectedOption := m.options[m.selected]
				// Since we dont show 1player we need to increment the selected by one to have the right room
				communication.SendRoomCreatePacket(m.conn, m.selected+1)
				log.Println("Selected option:", selectedOption)
			}
			return m, nil
		case tea.KeyTab:
			if m.inputFocused {
				m.roomIDInput.Blur()
				m.inputFocused = !m.inputFocused
			} else {
				m.roomIDInput.Focus()
				m.inputFocused = !m.inputFocused
			}
			return m, nil
		}

		switch msg.String() {
		case "k":
			if !m.inputFocused {
				m.selected = (m.selected - 1 + len(m.options)) % len(m.options)
			}
			return m, nil
		case "j":
			if !m.inputFocused {
				m.selected = (m.selected + 1) % len(m.options)
			}
			return m, nil
		}
	}

	if m.inputFocused {
		m.roomIDInput, cmd = m.roomIDInput.Update(msg)
	}

	if m.looking {
		m.spinner, cmd = m.spinner.Update(msg)
	}
	return m, cmd
}

func (m CreateModel) View() string {
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

	optionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Padding(1, 0)

	instructionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		Padding(1, 0)

	if m.looking {
		optionsBuilder.WriteString("\n\n")
		optionsBuilder.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Render("Joining room " + m.spinner.View()),
		)
	}

	var createBuilder strings.Builder
	createBuilder.WriteString("Enter the roomID to join an existing room.\n")
	createBuilder.WriteString(m.roomIDInput.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		optionsStyle.Render(optionsBuilder.String()),
		instructionsStyle.Render(createBuilder.String()),
	)
}
