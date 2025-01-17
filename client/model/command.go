package model

import (
	"github.com/GrGLeo/ctf/client/communication"
	tea "github.com/charmbracelet/bubbletea"
)

func PassLogin(username, password string) tea.Cmd{
  return func() tea.Msg {
    return communication.LoginMsg{
      Username: username,
      Password: password,
    }
  }
}
