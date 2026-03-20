package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func BrewTui(statusChan chan bool) {
	p := tea.NewProgram(initialModel(statusChan), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Tea spilled: %v", err)
	}
}
