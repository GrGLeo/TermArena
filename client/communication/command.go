package communication

import (
	"log"
  "time"

	tea "github.com/charmbracelet/bubbletea"
)


func AttemptReconnect() tea.Cmd {
    return tea.Tick(time.Second, func(time.Time) tea.Msg {
      log.Println("Enter AttemptReconnect")
        conn, err := MakeConnection("8082")
        if err != nil {
            return ReconnectMsg{}
        }
        return ConnectionMsg{Conn: conn}
    })
}


func LoginCommand(code bool) tea.Cmd {
  log.Println("Enter LoginCommand")
    return func() tea.Msg {
      return ResponseMsg{Code: code}
    }
}
