package main

import (
	"log"
	"net"
	"os"

	"github.com/GrGLeo/ctf/client/communication"
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
    Connection: MakeConnection(),
	}
}

func MakeConnection() *net.TCPConn{
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil
	}
  return conn
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
  log.Print(msg)
  var cmd tea.Cmd
  var newmodel tea.Model
  switch m.state {
  case Intro:
    switch msg := msg.(type) {
    case model.TickMsg:
      newmodel, cmd = m.AnimationModel.Update(msg)
      m.AnimationModel = newmodel.(model.AnimationModel)
      return m, cmd
    case tea.KeyMsg:
      if msg.Type == tea.KeyEnter {
        m.state = Login
        return m, m.LoginModel.Init()
      }
      return m, cmd
    case tea.WindowSizeMsg:
      log.Print("this was called")
      m.width = msg.Width
      m.height = msg.Height

      m.AnimationModel.SetDimension(m.width, m.height)
      m.LoginModel.SetDimension(m.width, m.height)
    }

  case Login:
    newmodel, cmd := m.LoginModel.Update(msg)
    m.LoginModel = newmodel.(model.LoginModel)
    if loginMsg, ok := msg.(model.LoginMsg); ok {
      communication.SendLoginPacket(m.Connection, loginMsg.Username, loginMsg.Password)
    }
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


  f, err := tea.LogToFile("debug.log", "debug")
  if err != nil {
    log.Fatal(err)
  }
  defer f.Close()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
