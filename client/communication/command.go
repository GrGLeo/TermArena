package communication

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)


func AttemptReconnect() tea.Cmd {
    return tea.Tick(time.Second, func(time.Time) tea.Msg {
        conn, err := MakeConnection()
        if err != nil {
            return ReconnectMsg{}
        }
        return ConnectionMsg{Conn: conn}
    })
}
