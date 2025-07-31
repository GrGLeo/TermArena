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
	Lobby      = "lobby"
	Menu       = "menu"
	Game       = "game"
	Shop       = "shop"
	GameOver   = "gameover"
)

type MetaModel struct {
	WaitingModel   model.WaitingModel
	AnimationModel model.AnimationModel
	AuthModel      model.AuthModel
	LobbyModel     model.LobbyModel
	GameModel      model.GameModel
	ShopModel      model.ShopModel
	GameOverModel  model.GameOverModel
	state          string
	Username       string
	Connection     *net.TCPConn
	GameConnection *net.TCPConn
	msgs           chan tea.Msg
	width          int
	height         int
}

func NewMetaModel() MetaModel {
	msgs := make(chan tea.Msg)

	state := Disconnect
	return MetaModel{
		state:          state,
		AnimationModel: model.NewAnimationModel(),
		msgs:           msgs,
	}
}

func (m MetaModel) Init() tea.Cmd {
	switch m.state {
	case Disconnect:
		return tea.Batch(m.WaitingModel.Init(), communication.AttemptReconnect())
	case Intro:
		return m.AnimationModel.Init()
	case Login:
		return m.AuthModel.Init()
	}
	return communication.AttemptReconnect()
}

func (m MetaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case communication.ConnectionMsg:
			m.Connection = msg.Conn
			m.state = Intro
			go communication.ListenForPackets(m.Connection, m.msgs)
			return m, m.AnimationModel.Init()
		case communication.ReconnectMsg:
			newmodel, cmd = m.WaitingModel.Update(msg)
			m.WaitingModel = newmodel.(model.WaitingModel)
			return m, tea.Batch(cmd, communication.AttemptReconnect())
		default:
			newmodel, cmd = m.WaitingModel.Update(msg)
			m.WaitingModel = newmodel.(model.WaitingModel)
			return m, cmd
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
				m.AuthModel = model.NewAuthModel(m.Connection)
				m.AuthModel.SetDimension(m.height, m.width)
				return m, m.AuthModel.Init()
			}
			return m, cmd
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.WaitingModel.SetDimension(m.height, m.width)
			m.AnimationModel.SetDimension(m.height, m.width)
			m.AuthModel.SetDimension(m.height, m.width)
			m.GameModel.SetDimension(m.height, m.width)
			m.ShopModel.SetDimension(m.height, m.width)
		}

	case Login:
		newmodel, cmd = m.AuthModel.Update(msg)
		m.AuthModel = newmodel.(model.AuthModel)
		switch msg := msg.(type) {
		case communication.ResponseMsg:
			if !msg.Code {
				log.Println("Failed to log in")
			} else {
				log.Println("Manage to log in")
				m.state = Lobby
				m.LobbyModel = model.NewLobbyModel(m.Connection)
				m.LobbyModel.SetDimension(m.height, m.width)
				return m, m.LobbyModel.Init()
			}
		default:
			return m, cmd
		}

	case Lobby:
		newmodel, cmd = m.LobbyModel.Update(msg)
		m.LobbyModel = newmodel.(model.LobbyModel)
		switch msg := msg.(type) {
		case communication.LookRoomMsg:
			for {
				conn, err := communication.MakeConnection(msg.RoomIP)
				if err == nil {
					m.GameConnection = conn
					// Send spell selection after successful game connection
					communication.SendSpellSelectionPacket(m.GameConnection, m.LobbyModel.SelectedSpells[0], m.LobbyModel.SelectedSpells[1])
					break
				}
			}
			go communication.ListenForPackets(m.GameConnection, m.msgs)
		case communication.GameStartMsg:
			m.state = Game
			m.GameModel = model.NewGameModel(m.GameConnection)
			m.GameModel.SetDimension(m.height, m.width)
			return m, m.GameModel.Init()
		default:
			return m, cmd
		}

	case Game:
		newmodel, cmd = m.GameModel.Update(msg)
		m.GameModel = newmodel.(model.GameModel)
		switch msg := msg.(type) {
		case model.GoToShopMsg:
			m.state = Shop
      m.ShopModel = model.NewShopModel(model.DefaultStyles())
      m.ShopModel.SetDimension(m.height, m.width)
			return m, m.ShopModel.Init()
		case communication.GameCloseMsg:
			m.state = GameOver
			m.GameOverModel = model.NewGameOverModel(msg.Code)
			m.GameOverModel.SetDimension(m.height, m.width)
			return m, m.GameOverModel.Init()
		default:
			return m, cmd
		}
	case Shop:
		switch msg := msg.(type) {
		case model.BackToGameMsg:
			m.state = Game
			return m, nil
		default:
			newmodel, cmd = m.ShopModel.Update(msg)
			m.ShopModel = newmodel.(model.ShopModel)
			return m, cmd
		}
	case GameOver:
		switch msg := msg.(type) {
		case model.GoToLobbyMsg:
			if m.GameConnection != nil {
				m.GameConnection.Close()
				m.GameConnection = nil
			}
			conn, err := communication.MakeConnection("8082")
			if err != nil {
				log.Println("Failed to make connection after game over: ", err.Error())
				// TODO: add a retry mechanism as when we start the client
				return m, tea.Quit
			}
			m.Connection = conn
			m.state = Lobby
			m.LobbyModel = model.NewLobbyModel(m.Connection)
			m.LobbyModel.SetDimension(m.height, m.width)
			go communication.ListenForPackets(m.Connection, m.msgs)
			return m, m.LobbyModel.Init()
		default:
			newmodel, cmd = m.GameOverModel.Update(msg)
			m.GameOverModel = newmodel.(model.GameOverModel)
			return m, cmd
		}
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
		return m.AuthModel.View()
	case Lobby:
		return m.LobbyModel.View()
	case Game:
		return m.GameModel.View()
	case Shop:
		return m.ShopModel.View()
	case GameOver:
		return m.GameOverModel.View()
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
	// Serve as a bridge to pass message from ListenForPackets to models
	go func() {
		for msg := range model.msgs {
			p.Send(msg)
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
