package model

import (
	"log"
	"net"
	"strings"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LobbyModel struct {
	tabSelected   int
	queueModel    QueueModel
	createModel   CreateModel
	conn          *net.TCPConn
	looking       bool
	width, height int
}

func NewLobbyModel(conn *net.TCPConn) LobbyModel {
	queueModel := NewQueueModel(conn)
	createModel := NewCreateModel(conn)

	return LobbyModel{
		tabSelected: 0,
		queueModel:  queueModel,
		createModel: createModel,
		conn:        conn,
	}
}

func (m *LobbyModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
	m.queueModel.SetConn(conn)
	m.createModel.SetConn(conn)
}

func (m *LobbyModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.queueModel.SetDimension(height, width)
	m.createModel.SetDimension(height, width)
}

func (m *LobbyModel) SetLooking(search bool) {
	m.looking = search
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
	case tea.KeyMsg:
		switch msg.String() {
		case "h":
			m.tabSelected = (m.tabSelected - 1 + 2) % 2
		case "l":
			m.tabSelected = (m.tabSelected + 1) % 2
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	if m.tabSelected == 0 {
		var qm tea.Model
		qm, cmd = m.queueModel.Update(msg)
		m.queueModel = qm.(QueueModel)
	} else {
		var cm tea.Model
		cm, cmd = m.createModel.Update(msg)
		m.createModel = cm.(CreateModel)
	}

	return m, cmd
}

func (m LobbyModel) View() string {
	var tabSelection string
	var content string

	if m.tabSelected == 0 {
		tabSelection = lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Queue Game"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Join/Create Game"),
		)
		content = m.queueModel.View()
	} else {
		tabSelection = lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Queue Game"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Join/Create Game"),
		)
		content = m.createModel.View()
	}

  lobby := lipgloss.JoinVertical(
		lipgloss.Top,
		tabSelection,
		content,
	)


  return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Padding(2, 4).
			Render(lobby),
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
		options:  []string{"Solo (1 player 1 bot vs 2 bots)", "2 players vs 2 bots", "2 players vs 2 players"},
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
  // Build the tab selection
	var leftTab string
	var rightTab string

  leftTab = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Queue up for a game")
  rightTab = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Create or Join Room")
	leftStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false)

	rightStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false)

	tabSelection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftTab),
		rightStyle.Render(rightTab),
	)


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
		Padding(1, 0)

	instructionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false, true).
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

  lobby := lipgloss.JoinVertical(
    lipgloss.Top,
    tabSelection,
    layout,
  )

  return lipgloss.Place(
    m.width,
    m.height,
    lipgloss.Center,
    lipgloss.Center,
    lipgloss.NewStyle().
    Padding(2, 4).
    Render(lobby),
  )
}

type CreateModel struct {
	options       []string
	selected      int
	roomIDInput   textinput.Model
	width, height int
	looking       bool
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
				// Add your logic to handle the room ID submission
			} else {
				selectedOption := m.options[m.selected]
				communication.SendRoomRequestPacket(m.conn, m.selected)
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
	// Build the tab selection
	var leftTab string
	var rightTab string

	leftTab = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Queue up for a game")
	rightTab = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("Create or Join Room")
	leftStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false)

	rightStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false)

	tabSelection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftTab),
		rightStyle.Render(rightTab),
	)

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
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Padding(1, 0)

	instructionsStyle := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false, true).
		Padding(1, 0)

	if m.looking {
		optionsBuilder.WriteString("\n\n")
		optionsBuilder.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Render("Looking for a room... " + m.spinner.View()),
		)
	}

  var createBuilder strings.Builder
  createBuilder.WriteString("Enter the roomID to join an existing room.\n")
  createBuilder.WriteString(m.roomIDInput.View())

	layout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		optionsStyle.Render(optionsBuilder.String()),
		instructionsStyle.Render(createBuilder.String()),
	)

	lobby := lipgloss.JoinVertical(
		lipgloss.Top,
		tabSelection,
		layout,
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Padding(2, 4).
			Render(lobby),
	)
}
