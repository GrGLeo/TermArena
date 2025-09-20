package model

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/GrGLeo/TermArena/client/communication"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MessagingModel struct {
	messages      []Message
	input         textinput.Model
	viewport      viewport.Model
	conn          *net.TCPConn
	width, height int
	showInput     bool
	username      string
	styles        *Styles
}

type Message struct {
	Content   string
	SenderID  string
	Timestamp time.Time
	IsSystem  bool
}

// Tab navigation messages
type TabLeftMsg struct{}
type TabRightMsg struct{}

func NewMessagingModel(conn *net.TCPConn, userID string) MessagingModel {
	ti := textinput.New()
	ti.Placeholder = "Type your message... (use /all or /userID for routing)"
	ti.Width = 60
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	ti.Focus()

	vp := viewport.New(70, 21)
	vp.SetContent("Welcome to the messaging system!\nUse /all to broadcast or /userID for private messages.\n")

	return MessagingModel{
		messages:  []Message{},
		input:     ti,
		viewport:  vp,
		conn:      conn,
		showInput: true,
		username:  userID,
		styles:    DefaultStyles(),
	}
}

func (m MessagingModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m MessagingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// For the moment we block to a fix size, this is not ideal.
		m.viewport.Width = 70
		m.viewport.Height = 21 - 4

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.showInput && m.input.Value() != "" {
				message := m.input.Value()
				message = strings.Trim(message, " ")
				m.input.SetValue("")

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
			}
		case tea.KeyTab:
			m.showInput = !m.showInput
			if m.showInput {
				cmd = m.input.Focus()
			} else {
				m.input.Blur()
			}
			cmds = append(cmds, cmd)
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

		// Handle navigation keys only when NOT in input mode
		if !m.showInput {
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
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
	}

	// Update input if focused
	if m.showInput {
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		cmds = append(cmds, inputCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m MessagingModel) View() string {
	var sections []string

	// Message history
	sections = append(sections, m.viewport.View())

	// Help text
	helpText := m.styles.Help.Render("Commands: /all (broadcast), /userID (private), TAB (toggle mode)")
	sections = append(sections, helpText)

	// Input field (if enabled)
	if m.showInput {
		inputSection := lipgloss.JoinHorizontal(
			lipgloss.Left,
			m.styles.InputPrompt.Render(fmt.Sprintf("%s> ", m.username)),
			m.input.View(),
		)
		sections = append(sections, inputSection)
	} else {
		scrollHelp := m.styles.Help.Render("↑/↓ (scroll), ←/→ (tabs), TAB (input mode)")
		sections = append(sections, scrollHelp)
	}

	return lipgloss.JoinVertical(lipgloss.Top, sections...)
}

func (m *MessagingModel) addMessage(msg Message) {
	m.messages = append(m.messages, msg)

	// Keep only last 100 messages to prevent memory issues
	if len(m.messages) > 100 {
		m.messages = m.messages[len(m.messages)-100:]
	}
}

func (m *MessagingModel) updateViewport() {
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
				} else if strings.Contains(msg.Content, "(room)") {
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
func (m *MessagingModel) wrapMessage(text string, width int) []string {
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
func (m *MessagingModel) padToWidth(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := width - len(text)
	return text + strings.Repeat(" ", padding)
}

func (m *MessagingModel) SetDimension(width, height int) {
	m.width = width
	m.height = height
}

func (m *MessagingModel) validateMessage(message string) error {
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

func isValidUserID(userID string) bool {
	// Basic validation - can be enhanced based on your user ID format
	return len(userID) > 0 && len(userID) <= 50 &&
		!strings.ContainsAny(userID, " \t\n\r")
}
