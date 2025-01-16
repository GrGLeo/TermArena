package model

import tea "github.com/charmbracelet/bubbletea"

func PerformLogin(username, password string) tea.Cmd{
  return func() tea.Msg {
    return LoginMsg{
      username: username,
      password: password,
    }
  }
}
