package tui

import (
	"fmt"
	wsclient "webrtc/websocket"

	tea "github.com/charmbracelet/bubbletea"
)

func BrewTui(statusChan chan bool, wsc *wsclient.WSClient) {
	p := tea.NewProgram(initialModel(statusChan, wsc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Tea spilled: %v", err)
	}
}
