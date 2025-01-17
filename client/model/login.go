package model

import (
	"log"
	"net"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
  BorderColor lipgloss.Color
  InputField lipgloss.Style
  Button      lipgloss.Style
	SelectedButton lipgloss.Style
}

func DefaultStyle() *Styles {
    s := new(Styles)
    s.BorderColor = lipgloss.Color("420")
    
    s.InputField = lipgloss.NewStyle().
        BorderForeground(s.BorderColor).
        BorderStyle(lipgloss.RoundedBorder()).
        Width(20)
    
    s.Button = lipgloss.NewStyle().
        BorderForeground(s.BorderColor).
        BorderStyle(lipgloss.NormalBorder()).
        Foreground(lipgloss.Color("5"))
    
    s.SelectedButton = lipgloss.NewStyle().
        BorderForeground(s.BorderColor).
        BorderStyle(lipgloss.NormalBorder()).
        Foreground(lipgloss.Color("208")).
        Background(lipgloss.Color("58")).
        Bold(true)
    
    return s
}

type LoginModel struct {
  username textinput.Model
  password textinput.Model
  selected int
  buttons []string
  width, height int
  conn *net.TCPConn
  style *Styles
}

func NewLoginModel(conn *net.TCPConn) LoginModel {
  tiUser := textinput.New()
  tiUser.Placeholder = "username"
  tiUser.Focus()

  tiPass := textinput.New()
  tiPass.EchoMode = textinput.EchoPassword
  tiPass.EchoCharacter = '*'
  tiPass.Placeholder = "password"

  return LoginModel{
    username: tiUser,
    password: tiPass,
    selected: 0,
    buttons: []string{"login", "quit"},
    conn: conn,
    style: DefaultStyle(),
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
  log.Print("login: ", m.height, m.width)
  switch msg := msg.(type) {
  case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
  case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyEsc, tea.KeyCtrlC:
      return m, tea.Quit
    case tea.KeyTab:
      m.selected = (m.selected + 1) % (2 + len(m.buttons))
      if m.selected < 2 {
        m.username.Blur()
        m.password.Blur()
        if m.selected == 0 {
          m.username.Focus()
        } else {
          m.password.Focus()
        }
      } else {
        m.username.Blur()
        m.password.Blur()
      }
    case tea.KeyEnter:
      if m.selected >= 2 {
        switch m.buttons[m.selected - 2] {
        case "login":
          
          communication.SendLoginPacket(m.conn, m.username.Value(), m.password.Value())
          return m, PassLogin(m.username.Value(), m.password.Value())
        case "quit":
          return m, tea.Quit
        }
      }
    }
  }
  var cmd tea.Cmd
  if m.username.Focused() {
    m.username, cmd = m.username.Update(msg)
  } else {
    m.password, cmd = m.password.Update(msg)
  }
  return m, cmd
}


func (m LoginModel) View() string {
  var buttonsView []string
	for i, btn := range m.buttons {
		var btnStyle lipgloss.Style
		if i == m.selected - 2 {
			btnStyle = m.style.SelectedButton
		} else {
			btnStyle = m.style.Button
		}
		buttonsView = append(buttonsView, btnStyle.Width(10).Render(btn))
	}

  return lipgloss.Place(
    m.width,
    m.height,
    lipgloss.Center,
    lipgloss.Center,
    lipgloss.JoinVertical(
      lipgloss.Center,
      lipgloss.JoinHorizontal(
        lipgloss.Center,
        m.style.InputField.Render(m.username.View()),
        m.style.InputField.Render(m.password.View()),
      ),
      lipgloss.JoinHorizontal(
        lipgloss.Center,
        buttonsView...,
      ),
    ),
  )
}

// func main() {
//   m := NewModel()
//   p := tea.NewProgram(m, tea.WithAltScreen())
//   if _, err := p.Run(); err != nil {
//     os.Exit(1)
//   }
// }
