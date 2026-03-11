package manager

import "github.com/charmbracelet/lipgloss"

const (
	windowWidth = 100
	tableHeight = 10
	logHeight   = 15
)

var charmPink = lipgloss.Color("#FF5F87")
var tableStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(charmPink).Width(windowWidth)
var logStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(charmPink).Padding(0, 1).Width(windowWidth)
