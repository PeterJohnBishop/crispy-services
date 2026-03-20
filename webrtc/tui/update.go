package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case statusMsg:
		m.connected = bool(msg)
		return m, listenForConnection(m.statusChan)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chatViewport.Width = msg.Width - 4
		m.chatViewport.Height = msg.Height - 10

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		// Handle Popup State
		if m.state == statePopup {
			switch msg.String() {
			case "esc":
				m.state = stateChat
			case "up", "k":
				if m.clientCursor > 0 {
					m.clientCursor--
				}
			case "down", "j":
				if m.clientCursor < len(m.clients)-1 {
					m.clientCursor++
				}
			case "enter":
				// Handle Attachment Logic Here
				targetClient := m.clients[m.clientCursor]
				m.messages = append(m.messages, fmt.Sprintf("System: Attached %s for %s", m.popupPathInput.Value(), targetClient))
				m.chatViewport.SetContent(strings.Join(m.messages, "\n"))
				m.chatViewport.GotoBottom()
				m.popupPathInput.Reset()
				m.state = stateChat
			}
			m.popupPathInput, cmd = m.popupPathInput.Update(msg)
			return m, cmd
		}

		// Handle Main Chat State
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 4
			m.updateFocus()
		case "shift+tab":
			m.focus--
			if m.focus < 0 {
				m.focus = 3
			}
			m.updateFocus()
		case "enter":
			if m.focus == focusAttachBtn {
				m.state = statePopup
				m.popupPathInput.Focus()
			} else if m.focus == focusSendBtn || m.focus == focusMsgInput {
				if m.msgInput.Value() != "" {
					m.messages = append(m.messages, fmt.Sprintf("Me: %s", m.msgInput.Value()))
					m.chatViewport.SetContent(strings.Join(m.messages, "\n"))
					m.chatViewport.GotoBottom()
					m.msgInput.Reset()
				}
			}
		}

		// Update active inputs
		m.savePathInput, cmd = m.savePathInput.Update(msg)
		cmds = append(cmds, cmd)
		m.msgInput, cmd = m.msgInput.Update(msg)
		cmds = append(cmds, cmd)
		m.chatViewport, cmd = m.chatViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateFocus() {
	m.savePathInput.Blur()
	m.msgInput.Blur()
	if m.focus == focusSavePath {
		m.savePathInput.Focus()
	} else if m.focus == focusMsgInput {
		m.msgInput.Focus()
	}
}
