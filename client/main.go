package main

import (
	"log"
	"net"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/GrGLeo/ctf/client/model"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	Disconnect = "dc"
	Intro      = "animation"
	Login      = "login"
	Menu       = "menu"
	Game       = "game"
)

type MetaModel struct {
	WaitingModel   model.WaitingModel
	AnimationModel model.AnimationModel
	LoginModel     model.LoginModel
	GameModel      model.GameModel
	state          string
	Username       string
	Connection     *net.TCPConn
	msgs           chan tea.Msg
	width          int
	height         int
}

func NewMetaModel() MetaModel {
	state := Intro
	conn, err := communication.MakeConnection()
	msgs := make(chan tea.Msg)

	if err != nil {
		state = Disconnect
	} else {
		go communication.ListenForPackets(conn, msgs)
	}
	return MetaModel{
		state:          state,
		AnimationModel: model.NewAnimationModel(),
		LoginModel:     model.NewLoginModel(conn),
		Connection:     conn,
		GameModel:      model.NewGameModel(conn),
	}
}

func (m MetaModel) Init() tea.Cmd {
	switch m.state {
	case Disconnect:
		return m.WaitingModel.Init()
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
	case Disconnect:
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
      m.width = msg.Width
      m.height = msg.Height
      m.WaitingModel.SetDimension(m.height, m.width)
			m.AnimationModel.SetDimension(m.height, m.width)
			m.LoginModel.SetDimension(m.height, m.width)
			m.GameModel.SetDimension(m.height, m.width)
    case communication.ConnectionMsg:
      m.Connection = msg.Conn
      m.state = Intro
      go communication.ListenForPackets(m.Connection, m.msgs)
      return m, m.AnimationModel.Init()
    default:
      newmodel, cmd = m.WaitingModel.Update(msg)
      m.WaitingModel = newmodel.(model.WaitingModel)
      return m, tea.Batch(cmd, communication.AttemptReconnect())
    }
	case Intro:
		switch msg := msg.(type) {
		case communication.TickMsg:
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
			m.width = msg.Width
			m.height = msg.Height
      m.WaitingModel.SetDimension(m.height, m.width)
			m.AnimationModel.SetDimension(m.height, m.width)
			m.LoginModel.SetDimension(m.height, m.width)
			m.GameModel.SetDimension(m.height, m.width)
		}

	case Login:
		newmodel, cmd = m.LoginModel.Update(msg)
		m.LoginModel = newmodel.(model.LoginModel)
		if loginMsg, ok := msg.(communication.LoginMsg); ok {
			communication.SendLoginPacket(m.Connection, loginMsg.Username, loginMsg.Password)
		}
		switch msg.(type) {
		case communication.ResponseMsg:
			log.Print("enter communication response msg")
			m.state = Game
			return m, m.GameModel.Init()
		}

	case Game:
		log.Print("enter Game")
		newmodel, cmd = m.GameModel.Update(msg)
		m.GameModel = newmodel.(model.GameModel)

		return m, cmd
	}
	return m, nil
}

func (m MetaModel) View() string {
	switch m.state {
  case Disconnect:
    return m.WaitingModel.View()
	case Intro:
		return m.AnimationModel.View()
	case Login:
		return m.LoginModel.View()
	case Game:
		return m.GameModel.View()
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
  // Serve as a bridge to pass message from outise to model
  go func() {
        for msg := range model.msgs {
            p.Send(msg)
        }
    }()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
