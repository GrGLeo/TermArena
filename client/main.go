package main

import (
	"net"
	"os"

	"github.com/GrGLeo/ctf/client/model"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	Intro = "animation"
	Login = "login"
	Menu  = "menu"
	Game  = "game"
)

type MetaModel struct {
	AnimationModel model.AnimationModel
  LoginModel model.LoginModel
	state          string
	Username       string
	Connection     *net.TCPConn
	width          int
	height         int
}

func NewMetaModel() MetaModel {
	return MetaModel{
		state:          Intro,
		AnimationModel: model.NewAnimationModel(),
    LoginModel: model.NewLoginModel(),
	}
}

func (m MetaModel) MakeConnection() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return
	}
	m.Connection = conn
}

func (m MetaModel) Init() tea.Cmd {
  switch m.state {
  case Intro:
    return m.AnimationModel.Init()
  case Login:
    return m.LoginModel.Init()
  }
  return nil
}

func (m MetaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmd tea.Cmd
  var newmodel tea.Model
  switch m.state {
  case Intro:
    switch msg := msg.(type) {
    case model.TickMsg:
      newmodel, cmd = m.AnimationModel.Update(msg)
      m.AnimationModel = newmodel.(model.AnimationModel)
    case tea.KeyMsg:
      if msg.String() == "enter" {
        m.state = Login
        return m, m.LoginModel.Init()
      }
      return m, cmd
  case tea.WindowSizeMsg:
    // Capture terminal size change (resize)
    m.width = msg.Width
    m.height = msg.Height

    // Pass the new dimensions to both models
    m.AnimationModel.SetDimension(m.width, m.height)
    m.LoginModel.SetDimension(m.width, m.height)
  }

  case Login:
    newmodel, cmd := m.LoginModel.Update(msg)
    m.LoginModel = newmodel.(model.LoginModel)
    return m, cmd
	}
	return m, nil
}

func (m MetaModel) View() string {
	switch m.state {
	case Intro:
		return m.AnimationModel.View()
  case Login:
    return m.LoginModel.View()
	}
	return ""
}

func main() {
	model := NewMetaModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
