package model

import (
	"net"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


type Styles struct {
	BorderColor    lipgloss.Color
	InputField     lipgloss.Style
	Button         lipgloss.Style
	SelectedButton lipgloss.Style
	// Tabs
	ActiveTabBorder lipgloss.Border
	ActiveTab       lipgloss.Style
	InactiveTab     lipgloss.Style
	TabGap          lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.BorderColor = lipgloss.Color("69")

	s.InputField = lipgloss.NewStyle().
		BorderForeground(s.BorderColor).
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(20)

	s.Button = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("57")).
		Padding(0, 3).
		MarginTop(1).
		MarginRight(1)

	s.SelectedButton = s.Button.
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("105")).
		Underline(true)

	// --- Tab Styles ---
	inactiveTabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}
	activeTabBorder := inactiveTabBorder
	activeTabBorder.Bottom = " "

	s.ActiveTab = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 3).
		Foreground(lipgloss.Color("205")).
		Border(activeTabBorder, true).
		BorderForeground(s.BorderColor)

	s.InactiveTab = lipgloss.NewStyle().
		Padding(0, 3).
		Foreground(lipgloss.Color("240")).
		Border(inactiveTabBorder, true).
		BorderForeground(s.BorderColor)

	// Gap style to put between tabs
	s.TabGap = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(s.BorderColor).
		PaddingRight(1)

	return s
}

// --- AuthModel (MetaModel) ---
type AuthModel struct {
	styles       *Styles
	tabSelected  int // 0 for Login, 1 for Create Account
	loginModel   LoginModel
	accountModel AccountModel
	conn         *net.TCPConn
	width, height int
}

func NewAuthModel(conn *net.TCPConn) AuthModel {
	styles := DefaultStyles()
	loginModel := NewLoginModel(conn, styles)
	accountModel := NewAccountModel(conn, styles)

	// Start with login focused
	loginModel.Focus()

	return AuthModel{
		styles:       styles,
		tabSelected:  0,
		loginModel:   loginModel,
		accountModel: accountModel,
		conn:         conn,
	}
}

func (m *AuthModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
	m.loginModel.SetConn(conn)
	m.accountModel.SetConn(conn)
}

func (m *AuthModel) SetDimension(height, width int) {
	m.width = width
	m.height = height
	m.loginModel.SetDimension(height, width)
	m.accountModel.SetDimension(height, width)
}

func (m AuthModel) Init() tea.Cmd {
	if m.tabSelected == 0 {
		return m.loginModel.Init()
	}
	return m.accountModel.Init()
}

func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimension(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Tab Navigation (use h/l like LobbyModel or Tab)
		// Using 'left' and 'right' for consistency with LobbyModel
		case "left":
			if m.tabSelected > 0 {
				m.tabSelected--
				m.accountModel.BlurAll()
				m.loginModel.Focus()
				cmds = append(cmds, m.loginModel.Init())
			}
		case "right":
			if m.tabSelected < 1 {
				m.tabSelected++
				m.loginModel.BlurAll()
				m.accountModel.Focus()
				cmds = append(cmds, m.accountModel.Init())
			}

		// --- Pass other keys down to the active model ---
		default:
			if m.tabSelected == 0 {
				var updatedLoginModel tea.Model
				updatedLoginModel, cmd = m.loginModel.Update(msg)
				m.loginModel = updatedLoginModel.(LoginModel)
				cmds = append(cmds, cmd)
			} else {
				var updatedAccountModel tea.Model
				updatedAccountModel, cmd = m.accountModel.Update(msg)
				m.accountModel = updatedAccountModel.(AccountModel)
				cmds = append(cmds, cmd)
			}
		}
	// --- Pass non-key messages down ---
	default:
		if m.tabSelected == 0 {
			var updatedLoginModel tea.Model
			updatedLoginModel, cmd = m.loginModel.Update(msg)
			m.loginModel = updatedLoginModel.(LoginModel)
			cmds = append(cmds, cmd)
		} else {
			var updatedAccountModel tea.Model
			updatedAccountModel, cmd = m.accountModel.Update(msg)
			m.accountModel = updatedAccountModel.(AccountModel)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AuthModel) View() string {
	var renderedTabs []string
	var content string

	// Render Tabs based on selection
	loginTabStr := "Login"
	createTabStr := "Create Account"
	if m.tabSelected == 0 {
		renderedTabs = append(renderedTabs, m.styles.ActiveTab.Render(loginTabStr))
		renderedTabs = append(renderedTabs, m.styles.InactiveTab.Render(createTabStr))
		content = m.loginModel.View() // Get content from the active model
	} else {
		renderedTabs = append(renderedTabs, m.styles.InactiveTab.Render(loginTabStr))
		renderedTabs = append(renderedTabs, m.styles.ActiveTab.Render(createTabStr))
		content = m.accountModel.View() // Get content from the active model
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
	finalView := lipgloss.NewStyle().Padding(1, 2).Render(ui)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		finalView,
	)
}

// --- AccountModel (SubModel) ---
type AccountModel struct {
	styles        *Styles
	username      textinput.Model
	password      textinput.Model
	passwordConf  textinput.Model
	focusIndex    int
	buttons       []string
	width, height int
	conn          *net.TCPConn
}

// Modified NewAccountModel to accept styles
func NewAccountModel(conn *net.TCPConn, styles *Styles) AccountModel {
	tiUser := textinput.New()
	tiUser.Placeholder = "Username"
	tiUser.CharLimit = 32
	tiUser.Width = 20

	tiPass := textinput.New()
	tiPass.Placeholder = "Password"
	tiPass.EchoMode = textinput.EchoPassword
	tiPass.EchoCharacter = '*'
	tiPass.CharLimit = 32
	tiPass.Width = 20

	tiPassConf := textinput.New()
	tiPassConf.Placeholder = "Confirm Password"
	tiPassConf.EchoMode = textinput.EchoPassword
	tiPassConf.EchoCharacter = '*'
	tiPassConf.CharLimit = 32
	tiPassConf.Width = 20

	return AccountModel{
		styles:       styles,
		username:     tiUser,
		password:     tiPass,
		passwordConf: tiPassConf,
		focusIndex:   0,
		buttons:      []string{"Create Account", "Quit"},
		conn:         conn,
	}
}

func (m *AccountModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
}

func (m *AccountModel) SetDimension(width, height int) {
	m.width = width
	m.height = height
}

// Focus sets focus on the first input field.
func (m *AccountModel) Focus() {
	m.focusIndex = 0
	m.username.Focus()
	m.password.Blur()
	m.passwordConf.Blur()
}

// BlurAll removes focus from all input fields.
func (m *AccountModel) BlurAll() {
	m.username.Blur()
	m.password.Blur()
	m.passwordConf.Blur()
}

func (m AccountModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the AccountModel *only*
func (m AccountModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		// --- Navigation within the form ---
		case tea.KeyTab, tea.KeyDown: // Cycle focus forward
			m.focusIndex = (m.focusIndex + 1) % (3 + len(m.buttons))
			m.username.Blur()
			m.password.Blur()
			m.passwordConf.Blur()
			if m.focusIndex == 0 {
				m.username.Focus()
			} else if m.focusIndex == 1 {
				m.password.Focus()
			} else if m.focusIndex == 2 {
				m.passwordConf.Focus()
			}
			if m.focusIndex < 3 {
				cmds = append(cmds, textinput.Blink)
			}

		case tea.KeyUp:
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = (3 + len(m.buttons)) - 1
			}
			// Update focus on text inputs
			m.username.Blur()
			m.password.Blur()
			m.passwordConf.Blur()
			if m.focusIndex == 0 {
				m.username.Focus()
			} else if m.focusIndex == 1 {
				m.password.Focus()
			} else if m.focusIndex == 2 {
				m.passwordConf.Focus()
			}
			// If focus is on an input, activate blink
			if m.focusIndex < 3 {
				cmds = append(cmds, textinput.Blink)
			}

		// --- Action ---
		case tea.KeyEnter:
			// Check if focus is on a button
			if m.focusIndex >= 3 {
				buttonIndex := m.focusIndex - 3
				switch m.buttons[buttonIndex] {
				case "Create Account":
					if m.password.Value() == m.passwordConf.Value() {
						// TODO: Add real packet for account creation
						communication.SendLoginPacket(m.conn, m.username.Value(), m.password.Value()) // Placeholder, use correct packet
					} else {
						// TODO: Display error message (e.g., set errMsg in AuthModel)
					}
					return m, cmd
				case "Quit":
					return m, tea.Quit
				}
			}

		// --- Input Handling ---
		// Let the focused input handle the key press
		default:
			if m.username.Focused() {
				m.username, cmd = m.username.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.password.Focused() {
				m.password, cmd = m.password.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.passwordConf.Focused() {
				m.passwordConf, cmd = m.passwordConf.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AccountModel) View() string {
	var buttonsView []string
	buttonOffset := 3

	for i, btn := range m.buttons {
		style := m.styles.Button
		if m.focusIndex == i+buttonOffset {
			style = m.styles.SelectedButton
		}
		buttonsView = append(buttonsView, style.Render(btn))
	}

	inputUserStyle := m.styles.InputField
	inputPassStyle := m.styles.InputField
	inputConfStyle := m.styles.InputField

	if m.username.Focused() {
		inputUserStyle = inputUserStyle.BorderForeground(lipgloss.Color("205"))
	}
	if m.password.Focused() {
		inputPassStyle = inputPassStyle.BorderForeground(lipgloss.Color("205"))
	}
	if m.passwordConf.Focused() {
		inputConfStyle = inputConfStyle.BorderForeground(lipgloss.Color("205"))
	}

	inputs := lipgloss.JoinVertical(lipgloss.Left,
		inputUserStyle.Render(m.username.View()),
		inputPassStyle.Render(m.password.View()),
		inputConfStyle.Render(m.passwordConf.View()),
	)

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, buttonsView...)

	// Combine inputs and buttons vertically, centered
	content := lipgloss.NewStyle().MarginTop(1).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			inputs,
			buttons,
		),
	)

	return content
}

// --- LoginModel (SubModel) ---
type LoginModel struct {
	styles        *Styles
	username      textinput.Model
	password      textinput.Model
	focusIndex    int
	buttons       []string
	width, height int
	conn          *net.TCPConn
}

func NewLoginModel(conn *net.TCPConn, styles *Styles) LoginModel {
	tiUser := textinput.New()
	tiUser.Placeholder = "Username"
	tiUser.Width = 20
	tiUser.CharLimit = 32

	tiPass := textinput.New()
	tiPass.Placeholder = "Password"
	tiPass.EchoMode = textinput.EchoPassword
	tiPass.EchoCharacter = '*'
	tiPass.Width = 20
	tiPass.CharLimit = 32

	return LoginModel{
		styles:     styles,
		username:   tiUser,
		password:   tiPass,
		focusIndex: 0,
		buttons:    []string{"Login", "Quit"},
		conn:       conn,
	}
}

func (m *LoginModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
}

func (m *LoginModel) SetDimension(width, height int) {
	m.width = width
	m.height = height
}

// Focus sets focus on the first input field.
func (m *LoginModel) Focus() {
	m.focusIndex = 0
	m.username.Focus()
	m.password.Blur()
}

// BlurAll removes focus from all input fields.
func (m *LoginModel) BlurAll() {
	m.username.Blur()
	m.password.Blur()
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the LoginModel *only*
func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		// --- Navigation within the form ---
		case tea.KeyTab, tea.KeyDown:
			m.focusIndex = (m.focusIndex + 1) % (2 + len(m.buttons))
			m.username.Blur()
			m.password.Blur()
			if m.focusIndex == 0 {
				m.username.Focus()
			} else if m.focusIndex == 1 {
				m.password.Focus()
			}
			if m.focusIndex < 2 {
				cmds = append(cmds, textinput.Blink)
			}

		case tea.KeyUp:
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = (2 + len(m.buttons)) - 1
			}
			m.username.Blur()
			m.password.Blur()
			if m.focusIndex == 0 {
				m.username.Focus()
			} else if m.focusIndex == 1 {
				m.password.Focus()
			}
			if m.focusIndex < 2 {
				cmds = append(cmds, textinput.Blink)
			}

		// --- Action ---
		case tea.KeyEnter:
			if m.focusIndex >= 2 {
				buttonIndex := m.focusIndex - 2
				switch m.buttons[buttonIndex] {
				case "Login":
					communication.SendLoginPacket(m.conn, m.username.Value(), m.password.Value())
				case "Quit":
					return m, tea.Quit
				}
			}

		// --- Input Handling ---
		default:
			if m.username.Focused() {
				m.username, cmd = m.username.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.password.Focused() {
				m.password, cmd = m.password.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the LoginModel UI
func (m LoginModel) View() string {
	var buttonsView []string
	buttonOffset := 2

	for i, btn := range m.buttons {
		style := m.styles.Button
		if m.focusIndex == i+buttonOffset {
			style = m.styles.SelectedButton
		}
		buttonsView = append(buttonsView, style.Render(btn))
	}

	// Use shared styles
	inputUserStyle := m.styles.InputField
	inputPassStyle := m.styles.InputField

	if m.username.Focused() {
		inputUserStyle = inputUserStyle.BorderForeground(lipgloss.Color("205"))
	}
	if m.password.Focused() {
		inputPassStyle = inputPassStyle.BorderForeground(lipgloss.Color("205"))
	}

	inputs := lipgloss.JoinVertical(lipgloss.Left,
		inputUserStyle.Render(m.username.View()),
		inputPassStyle.Render(m.password.View()),
	)

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, buttonsView...)

	content := lipgloss.NewStyle().MarginTop(1).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			inputs,
			buttons,
		),
	)
	return content
}
