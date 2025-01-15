package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LoginModel struct {
  userInput textinput.Model
  passInput textinput.Model
  height, width int
}

func NewLoginModel() LoginModel {
  username := textinput.New()
  username.Placeholder = "username"
  username.Focus()

  password := textinput.New()
  password.Placeholder = "password"
  password.EchoMode = textinput.EchoPassword
  password.EchoCharacter = '*'

  return LoginModel{
    userInput: username,
    passInput: password,
  }
}

func (m *LoginModel) SetDimension(height, width int) {
  m.height = height
  m.width = width
}

func (m LoginModel) Init() tea.Cmd {
  return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmds []tea.Cmd 
  var usernameCmd tea.Cmd 
  var passwordCmd tea.Cmd 

  switch msg := msg.(type) {
  case tea.KeyMsg:
    switch msg.Type{
    case tea.KeyTab:
      if m.userInput.Focused() {
        m.userInput.Blur()
        m.passInput.Focus()
      } else {
        m.passInput.Blur()
        m.userInput.Focus()
      }
    case tea.KeyCtrlC:
      return m, tea.Quit
    }
  }
  m.userInput, usernameCmd = m.userInput.Update(msg)
  cmds = append(cmds, usernameCmd)
  m.passInput, passwordCmd = m.passInput.Update(msg)
  cmds = append(cmds, passwordCmd)
  return m, tea.Batch(cmds...)
}

func (m LoginModel) View() string {
  centeredStyle := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)

	// Center the username and password inputs horizontally and vertically
	var view strings.Builder

	// Vertical centering: Repeat "\n" to get the correct vertical alignment
	// Horizontal centering: Use lipgloss to center text input

	// Center the username input
	view.WriteString(centeredStyle.Render(m.userInput.View()))
	view.WriteString("\n")

	// Center the password input
	view.WriteString(centeredStyle.Render(m.passInput.View()))
	view.WriteString("\n")

	// You can add any additional text or elements below the inputs
	view.WriteString(centeredStyle.Render("Press Enter to Login"))

	return view.String()
  return m.userInput.View() + "\n\n" + m.passInput.View()
}
