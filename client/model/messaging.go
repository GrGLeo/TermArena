package model

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/GrGLeo/ctf/client/communication"
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
	ti.Width = 50
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(80, 20)
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
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.showInput && m.input.Value() != "" {
				message := m.input.Value()
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
				} else {
					// Add sent message to local display
					m.addMessage(Message{
						Content:   message,
						SenderID:  m.username,
						Timestamp: time.Now(),
						IsSystem:  false,
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
			switch msg.String() {
			case "j", "k":
				var vpCmd tea.Cmd
				if msg.String() == "j" {
					m.viewport, vpCmd = m.viewport.Update(tea.KeyMsg{Type: tea.KeyDown})
				} else {
					m.viewport, vpCmd = m.viewport.Update(tea.KeyMsg{Type: tea.KeyUp})
				}
				cmds = append(cmds, vpCmd)
			case "h":
				return m, func() tea.Msg { return TabLeftMsg{} }
			case "l":
				return m, func() tea.Msg { return TabRightMsg{} }
			}

			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		}

	case communication.IncomingMessageMsg:
		// Handle incoming messages from server
		m.addMessage(Message{
			Content:   msg.Content,
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
		scrollHelp := m.styles.Help.Render("↑/↓ (scroll), h/l (tabs), TAB (input mode)")
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

		if msg.IsSystem {
			content.WriteString(m.styles.SystemMessage.Render(
				fmt.Sprintf("[%s] *** %s ***\n", timestamp, msg.Content),
			))
		} else {
			if msg.SenderID == m.username {
				content.WriteString(m.styles.OwnMessage.Render(
					fmt.Sprintf("[%s] %s\n", timestamp, msg.Content),
				))
			} else {
				content.WriteString(m.styles.OtherMessage.Render(
					fmt.Sprintf("[%s] %s\n", timestamp, msg.Content),
				))
			}
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m *MessagingModel) SetDimension(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 6
}

func (m *MessagingModel) validateMessage(message string) error {
	if len(message) == 0 {
		return fmt.Errorf("message cannot be empty")
	}
	if len(message) > 1024 {
		return fmt.Errorf("message too long (max 1024 characters)")
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
