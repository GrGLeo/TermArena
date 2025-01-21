package communication

import (
	"log"
  "time"

	tea "github.com/charmbracelet/bubbletea"
)


func AttemptReconnect() tea.Cmd {
    return tea.Tick(time.Second, func(time.Time) tea.Msg {
      log.Println("Enter AttemptReconnect")
        conn, err := MakeConnection()
        if err != nil {
            return ReconnectMsg{}
        }
        return ConnectionMsg{Conn: conn}
    })
}


func LoginCommand(code int) tea.Cmd {
  log.Println("Enter LoginCommand")
    return func() tea.Msg {
      return ResponseMsg{Code: 1}
    }
}
