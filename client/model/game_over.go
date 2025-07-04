package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GameOverModel struct {
	win     bool
	height, width int
}

func NewGameOverModel(win bool) GameOverModel {
	return GameOverModel{win: win}
}

func (m GameOverModel) Init() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return BackToLobbyMsg{}
	})
}

func (m *GameOverModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func (m GameOverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case BackToLobbyMsg:
		return m, func() tea.Msg { return GoToLobbyMsg{} }
	}
	return m, nil
}

func (m GameOverModel) View() string {
	var s string
	if m.win {
		s = "You Win!"
	} else {
		s = "You Lose!"
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		s,
	)
}

type BackToLobbyMsg struct{}
type GoToLobbyMsg struct{}
