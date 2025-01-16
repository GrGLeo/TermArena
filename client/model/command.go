package model

import tea "github.com/charmbracelet/bubbletea"

func PassLogin(username, password string) tea.Cmd{
  return func() tea.Msg {
    return LoginMsg{
      Username: username,
      Password: password,
    }
  }
}
