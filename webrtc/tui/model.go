package tui

import (
	wsclient "webrtc/websocket"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI States
const (
	stateChat = iota
	statePopup
)

// Focus Elements
const (
	focusSavePath = iota
	focusAttachBtn
	focusMsgInput
	focusSendBtn
)

// Styling
var (
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	buttonStyle  = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	activeBtn    = buttonStyle.Copy().Background(lipgloss.Color("205"))
	popupStyle   = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2).Background(lipgloss.Color("0"))
)

type model struct {
	connected  bool
	statusChan chan bool
	wsc        *wsclient.WSClient
	state      int
	focus      int
	width      int
	height     int

	// Main Chat UI
	savePathInput textinput.Model
	chatViewport  viewport.Model
	msgInput      textinput.Model
	messages      []string

	// Popup UI
	popupPathInput textinput.Model
	clients        []string
	clientCursor   int
}

type statusMsg bool

func listenForConnection(sub chan bool) tea.Cmd {
	return func() tea.Msg {
		return statusMsg(<-sub)
	}
}

func initialModel(statusChan chan bool, wsc *wsclient.WSClient) model {
	sp := textinput.New()
	sp.Placeholder = "/path/to/save/files"
	sp.Focus()

	mi := textinput.New()
	mi.Placeholder = "Type a message..."

	pp := textinput.New()
	pp.Placeholder = "/path/to/attachment"

	vp := viewport.New(0, 0)
	vp.SetContent("Welcome to the chat!\n")

	return model{
		connected:      false,
		statusChan:     statusChan,
		wsc:            wsc,
		state:          stateChat,
		focus:          focusSavePath,
		savePathInput:  sp,
		msgInput:       mi,
		chatViewport:   vp,
		popupPathInput: pp,
		messages:       []string{"System: Connected."},
		clients:        []string{"Client A", "Client B", "Client C"},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, listenForConnection(m.statusChan))
}
