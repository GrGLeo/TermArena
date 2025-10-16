package model

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/GrGLeo/TermArena/client/communication"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Player struct {
	username           string
	spellOne, spellTwo string
}

func NewPlayer(username string) *Player {
	return &Player{
		username: username,
	}
}

func (p *Player) UpdateSpell(spellOne, spellTwo string) {
	p.spellOne = spellOne
	p.spellTwo = spellTwo
}

func (p *Player) Display() string {
	return fmt.Sprintf("%s\n- %s\n- %s", p.username, p.spellOne, p.spellTwo)
}

type TeamPanel struct {
	team      string
	players   []*Player
	maxPlayer int
}

func NewTeamPanel(team string, maxPlayer int) *TeamPanel {
	return &TeamPanel{
		team:      team,
		maxPlayer: maxPlayer,
	}
}

func (tp *TeamPanel) AddPlayer(player *Player) {
	tp.players = append(tp.players, player)
}

func (tp *TeamPanel) Display() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Team: %s (%d/%d players)\n", tp.team, len(tp.players), tp.maxPlayer))

	// Display all 4 slots
	for i := range tp.maxPlayer {
		if i < len(tp.players) {
			// Show actual player (takes 3 lines)
			builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, tp.players[i].Display()))
		} else {
			// Show placeholder for empty slot (3 lines to match player display)
			builder.WriteString(fmt.Sprintf("  %d. [Empty Slot]\n", i+1))
			builder.WriteString("     -\n")
			builder.WriteString("     -\n")
		}
	}

	return builder.String()
}

type LobbyRoomModel struct {
	cursor         int
	blueTeam       *TeamPanel
	roomType       int
	roomID         int
	redTeam        *TeamPanel
	SpellSelection SpellSelectionModel
	activePanel    int // 0 = message, 1 = spells
	width, height  int // Terminal dimensions
	viewport       viewport.Model
	textInput      textinput.Model
	timer          timer.Model
	messages       []Message
	conn           *net.TCPConn
	username       string
	styles         *Styles
}

func NewLobbyRoomModel(conn *net.TCPConn, username string, roomType int, roomID int) LobbyRoomModel {
	// Initialize viewport for messaging
	vp := viewport.New(76, 8) // Fixed size: 80 width - 4 for padding
	vp.SetContent("Welcome to TermArena!\n\nThis is the messaging area.\nMessages will appear here...\n")

	// Initialize text input for chat
	ti := textinput.New()
	ti.Placeholder = "Type your message here..."
	ti.Width = 76 // Fixed size: 80 width - 4 for padding
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	ti.Focus()

	return LobbyRoomModel{
		blueTeam:       NewTeamPanel("Blue", 4),
		redTeam:        NewTeamPanel("Red", 4),
		roomType:       roomType,
		roomID:         roomID,
		SpellSelection: NewSpellSelection(DefaultStyles()),
		activePanel:    0,   // Start with message panel
		width:          120, // Default width
		height:         30,  // Default height
		viewport:       vp,
		textInput:      ti,
		timer:          timer.NewWithInterval(time.Minute, time.Second),
		messages:       []Message{},
		conn:           conn,
		username:       username,
		styles:         DefaultStyles(),
	}
}

func (m *LobbyRoomModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
	m.SpellSelection.SetDimension(width, height)
}

func (m LobbyRoomModel) Init() tea.Cmd {
	return m.timer.Init()
}

func (m LobbyRoomModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case communication.MoveLobbyRoomMsg:
		for _, ui := range msg.UserInfos {
			player := NewPlayer(ui.Username)
			spell1Name := availableSpells[ui.SpellOne].Name
			spell2Name := availableSpells[ui.SpellTwo].Name
			player.UpdateSpell(spell1Name, spell2Name)
			if ui.Team == 0 {
				m.blueTeam.AddPlayer(player)
			} else {
				m.redTeam.AddPlayer(player)
			}
		}
		return m, nil
	case communication.UpdateSpellMsg:
		spell1Name := availableSpells[msg.SpellOne].Name
		spell2Name := availableSpells[msg.SpellTwo].Name
		// Find and update the player
		for _, player := range m.blueTeam.players {
			if player.username == msg.Username {
				player.UpdateSpell(spell1Name, spell2Name)
				break
			}
		}
		for _, player := range m.redTeam.players {
			if player.username == msg.Username {
				player.UpdateSpell(spell1Name, spell2Name)
				break
			}
		}
		return m, nil
	case tea.WindowSizeMsg:
		// Update terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		m.SpellSelection.SetDimension(m.height, m.width)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.activePanel == 0 && m.textInput.Value() != "" {
				message := m.textInput.Value()
				message = strings.Trim(message, " ")
				m.textInput.SetValue("")

				// Handling /quit message
				if message == "/quit" {
					if err := communication.SendQuitRoom(m.conn, m.roomID); err != nil {
						m.addMessage(Message{
							Content:   fmt.Sprintf("Failed to quit room: %v", err),
							SenderID:  "System",
							Timestamp: time.Now(),
							IsSystem:  true,
						})
						m.updateViewport()
						return m, nil
					}
					log.Printf("[CLIENT] Quit packet sent successfully for user %s in room %d", m.username, m.roomID)
					m.updateViewport()
					return m, nil
				}
				// Validate message before sending
				if err := m.validateMessage(message); err != nil {
					m.addMessage(Message{
						Content:   fmt.Sprintf("Validation error: %v", err),
						SenderID:  "System",
						Timestamp: time.Now(),
						IsSystem:  true,
					})
					m.updateViewport()
					return m, nil
				}

				// Send message
				err := communication.SendMessage(m.conn, m.username, message)
				if err != nil {
					m.addMessage(Message{
						Content:   fmt.Sprintf("Failed to send message: %v", err),
						SenderID:  "System",
						Timestamp: time.Now(),
						IsSystem:  true,
					})
				}
				m.updateViewport()
				return m, nil
			} else if m.activePanel == 1 {
				// Handle spell selection
				newModel, _ := m.SpellSelection.Update(msg)
				m.SpellSelection = newModel.(SpellSelectionModel)
				spellOne := m.SpellSelection.Spells[m.SpellSelection.SelectedIndices[0]].ID
				spellTwo := m.SpellSelection.Spells[m.SpellSelection.SelectedIndices[1]].ID
				log.Printf("spellOne: %d", spellOne)
				log.Printf("spellTwo: %d", spellTwo)
				spells := []int{spellOne, spellTwo}
				communication.SendUpdateSpell(m.conn, m.roomType, m.roomID, m.username, spells)
			}
		default:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "tab":
				// Switch between message and spells
				m.activePanel = 1 - m.activePanel
				if m.activePanel == 0 {
					m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}
			default:
				if m.activePanel == 0 {
					// Handle text input
					newInput, cmd := m.textInput.Update(msg)
					m.textInput = newInput
					return m, cmd
				} else if m.activePanel == 1 {
					// Handle spell selection
					newModel, _ := m.SpellSelection.Update(msg)
					m.SpellSelection = newModel.(SpellSelectionModel)
				}
			}
		}

	case communication.IncomingMessageMsg:
		// Handle incoming messages from server
		m.addMessage(Message{
			Content:   strings.Trim(msg.Content, " "),
			Timestamp: time.Now(),
			IsSystem:  false,
		})
		m.updateViewport()

	case communication.MessageErrorMsg:
		m.addMessage(Message{
			Content:   fmt.Sprintf("Error: %s", msg.Error),
			SenderID:  "System",
			Timestamp: time.Now(),
			IsSystem:  true,
		})
		m.updateViewport()
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m LobbyRoomModel) View() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")). // Pink color
		Padding(1, 2).
		Margin(0, 1)

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")). // Pink color
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // Gray color

	blueTitle := "Blue Team"
	blueContent := blueTitle + "\n" + m.blueTeam.Display()
	bluePanel := borderStyle.Render(blueContent)

	// Spell selection panel
	spellTitle := "Spell Selection"
	if m.activePanel == 1 {
		spellTitle = activeStyle.Render("> Spell Selection")
	} else {
		spellTitle = normalStyle.Render("  Spell Selection")
	}

	spellContent := spellTitle + "\n" + m.SpellSelection.View()
	spellPanel := borderStyle.Render(spellContent)

	// Red team panel
	redTitle := "Red Team"

	redContent := redTitle + "\n" + m.redTeam.Display()
	redPanel := borderStyle.Render(redContent)

	// Combine all three panels side by side
	allPanelsView := lipgloss.JoinHorizontal(lipgloss.Top, bluePanel, spellPanel, redPanel)

	// Messaging section
	messagingStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Margin(1, 0).
		Width(138) // Fixed width for messaging

	messagingTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Render("💬 Chat")

	messagingContent := messagingTitle + "\n" + m.viewport.View() + "\nGame start in: " + m.timer.View()

	// Text input section
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 0, 1, 0).
		Width(138) // Fixed width for text input

	var inputContent string
	if m.activePanel == 0 {
		inputSection := lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.styles.InputPrompt.Render(fmt.Sprintf("%s> ", m.username)),
			m.textInput.View(),
		)
		inputContent = inputStyle.Render(inputSection)
	} else {
		inputContent = inputStyle.Render("TAB to enter message mode")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		allPanelsView,
		messagingStyle.Render(messagingContent),
		inputContent,
	)

	centeredContent := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	return centeredContent
}

func (m *LobbyRoomModel) addMessage(msg Message) {
	m.messages = append(m.messages, msg)

	// Keep only last 100 messages to prevent memory issues
	if len(m.messages) > 100 {
		m.messages = m.messages[len(m.messages)-100:]
	}
}

func (m *LobbyRoomModel) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		var messageText string

		if msg.IsSystem {
			messageText = fmt.Sprintf("[%s] *** %s ***", timestamp, msg.Content)
		} else {
			if msg.SenderID == m.username {
				messageText = fmt.Sprintf("[%s] %s", timestamp, msg.Content)
			} else {
				messageText = fmt.Sprintf("[%s] %s", timestamp, msg.Content)
			}
		}

		// Handle message wrapping for long messages
		messageLines := m.wrapMessage(messageText, m.viewport.Width)

		for _, line := range messageLines {
			// Pad the line to viewport width
			paddedLine := m.padToWidth(line, m.viewport.Width)

			if msg.IsSystem {
				content.WriteString(m.styles.SystemMessage.Render(paddedLine))
			} else {
				var style lipgloss.Style
				if msg.SenderID == m.username {
					style = m.styles.OwnMessage
				} else if strings.Contains(msg.Content, "(all)") {
					style = m.styles.AllMessage
				} else if strings.Contains(msg.Content, "(whisper)") {
					style = m.styles.WhisperMessage
				} else if strings.Contains(msg.Content, "(room)") || strings.Contains(msg.Content, "(team)") {
					style = m.styles.RoomMessage
				} else {
					style = m.styles.OtherMessage
				}
				content.WriteString(style.Render(paddedLine))
			}
			content.WriteString("\n")
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// Helper function to wrap long messages
func (m *LobbyRoomModel) wrapMessage(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}

// Helper function to pad text to specified width
func (m *LobbyRoomModel) padToWidth(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := width - len(text)
	return text + strings.Repeat(" ", padding)
}

func (m *LobbyRoomModel) validateMessage(message string) error {
	if len(message) == 0 {
		return fmt.Errorf("message cannot be empty")
	}
	if len(message) > 256 {
		return fmt.Errorf("message too long (max 256 characters)")
	}

	// Validate prefix format
	if strings.HasPrefix(message, "/") {
		parts := strings.Fields(message)
		if len(parts) == 0 {
			return fmt.Errorf("invalid command format")
		}

		prefix := parts[0][1:] // Remove the "/"
		if prefix != "all" && !isValidUserID(prefix) {
			return fmt.Errorf("invalid user ID in private message")
		}
	}

	return nil
}
