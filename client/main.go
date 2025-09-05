package main

import (
	"go.dalton.dog/bubbleup"
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
	alert          bubbleup.AlertModel
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
		alert:          *bubbleup.NewAlertModel(80, true),
	}
}

func (m MetaModel) Init() tea.Cmd {
	switch m.state {
	case Disconnect:
		return tea.Batch(m.WaitingModel.Init(), communication.AttemptReconnect(), m.alert.Init())
	case Intro:
		return tea.Batch(m.AnimationModel.Init(), m.alert.Init())
	case Login:
		return tea.Batch(m.AuthModel.Init(), m.alert.Init())
	}
	return tea.Batch(communication.AttemptReconnect(), m.alert.Init())
}

func (m MetaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var alertCmd tea.Cmd
	var cmd tea.Cmd
	var newmodel tea.Model

	// Handle alert-specific messages
	switch msg := msg.(type) {
	case communication.RateLimitMsg:
		alertCmd = m.alert.NewAlertCmd(bubbleup.WarnKey, "Rate limit exceeded, wait one minute...")
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Always update alert model with current message
	outAlert, outCmd := m.alert.Update(msg)
	m.alert = outAlert.(bubbleup.AlertModel)

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
			return m, tea.Batch(m.AnimationModel.Init(), outCmd, alertCmd)
		case communication.ReconnectMsg:
			newmodel, cmd = m.WaitingModel.Update(msg)
			m.WaitingModel = newmodel.(model.WaitingModel)
			return m, tea.Batch(cmd, communication.AttemptReconnect(), outCmd, alertCmd)
		default:
			newmodel, cmd = m.WaitingModel.Update(msg)
			m.WaitingModel = newmodel.(model.WaitingModel)
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}
	case Intro:
		switch msg := msg.(type) {
		case communication.TickMsg:
			newmodel, cmd = m.AnimationModel.Update(msg)
			m.AnimationModel = newmodel.(model.AnimationModel)
			return m, tea.Batch(cmd, outCmd, alertCmd)
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				m.state = Login
				m.AuthModel = model.NewAuthModel(m.Connection)
				m.AuthModel.SetDimension(m.height, m.width)
				return m, tea.Batch(m.AuthModel.Init(), outCmd, alertCmd)
			}
			return m, tea.Batch(cmd, outCmd, alertCmd)
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
		case communication.AuthResultMsg:
			if !msg.Success {
				log.Println("Failed to log in")
			} else {
				m.Username = m.AuthModel.Username
				m.state = Lobby
				m.LobbyModel = model.NewLobbyModel(m.Connection, m.Username)
				m.LobbyModel.SetDimension(m.height, m.width)
				return m, tea.Batch(m.LobbyModel.Init(), outCmd, alertCmd)
			}
		default:
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}

	case Lobby:
		newmodel, cmd = m.LobbyModel.Update(msg)
		m.LobbyModel = newmodel.(model.LobbyModel)
		switch msg := msg.(type) {
		case communication.LookRoomMsg:
			return m, tea.Batch(communication.AttemptGameConnection(msg.RoomIP), outCmd, alertCmd)
		case communication.GameConnectionMsg:
			m.GameConnection = msg.Conn
			communication.SendSpellSelectionPacket(m.GameConnection, m.LobbyModel.SelectedSpells[0], m.LobbyModel.SelectedSpells[1])
			go communication.ListenForPackets(m.GameConnection, m.msgs)
			return m, tea.Batch(outCmd, alertCmd)
		case communication.GameConnectionFailedMsg:
			log.Println("Failed to connect to game server after multiple attempts.")
			return m, tea.Batch(outCmd, alertCmd)
		case communication.GameStartMsg:
			m.state = Game
			m.GameModel = model.NewGameModel(m.GameConnection)
			m.GameModel.SetDimension(m.height, m.width)
			return m, tea.Batch(m.GameModel.Init(), outCmd, alertCmd)
		default:
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}

	case Game:
		newmodel, cmd = m.GameModel.Update(msg)
		m.GameModel = newmodel.(model.GameModel)
		switch msg := msg.(type) {
		case communication.GoToShopMsg:
			m.state = Shop
			m.ShopModel = model.NewShopModel(
				model.DefaultStyles(),
				msg.Health,
				msg.Mana,
				msg.Attack_damage,
				msg.Armor,
				msg.Gold,
				msg.Inventory,
				m.GameConnection,
			)
			m.ShopModel.SetDimension(m.height, m.width)
			return m, tea.Batch(m.ShopModel.Init(), outCmd, alertCmd)
		case communication.GameCloseMsg:
			m.state = GameOver
			m.GameOverModel = model.NewGameOverModel(msg.Code)
			m.GameOverModel.SetDimension(m.height, m.width)
			return m, tea.Batch(m.GameOverModel.Init(), outCmd, alertCmd)
		default:
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}
	case Shop:
		switch msg := msg.(type) {
		case communication.BackToGameMsg:
			m.state = Game
			return m, tea.Batch(outCmd, alertCmd)
		default:
			newmodel, cmd = m.ShopModel.Update(msg)
			m.ShopModel = newmodel.(model.ShopModel)
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}
	case GameOver:
		switch msg := msg.(type) {
		case model.GoToLobbyMsg:
			if m.GameConnection != nil {
				m.GameConnection.Close()
				m.GameConnection = nil
			}

			// Attempt to reconnect to the lobby server silently
			conn, err := communication.MakeConnection("8082")
			if err != nil {
				// If reconnect fails, fall back to the Disconnect screen for robust retries
				log.Println("Failed to reconnect to lobby server, falling back to disconnect screen:", err)
				m.state = Disconnect
				return m, tea.Batch(m.Init(), outCmd, alertCmd)
			}

			// If successful, set the new connection and start a new packet listener
			m.Connection = conn
			go communication.ListenForPackets(m.Connection, m.msgs)

			m.state = Lobby
			m.LobbyModel = model.NewLobbyModel(m.Connection, m.Username)
			m.LobbyModel.SetDimension(m.height, m.width)
			return m, tea.Batch(m.LobbyModel.Init(), outCmd, alertCmd)
		default:
			newmodel, cmd = m.GameOverModel.Update(msg)
			m.GameOverModel = newmodel.(model.GameOverModel)
			return m, tea.Batch(cmd, outCmd, alertCmd)
		}
	}
	return m, tea.Batch(outCmd, alertCmd)
}

func (m MetaModel) View() string {
	var content string
	switch m.state {
	case Disconnect:
		content = m.WaitingModel.View()
	case Intro:
		content = m.AnimationModel.View()
	case Login:
		content = m.AuthModel.View()
	case Lobby:
		content = m.LobbyModel.View()
	case Game:
		content = m.GameModel.View()
	case Shop:
		content = m.ShopModel.View()
	case GameOver:
		content = m.GameOverModel.View()
	default:
		content = ""
	}
	return m.alert.Render(content)
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
