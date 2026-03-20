package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	savePathView := borderStyle.Render("Save Path: " + m.savePathInput.View())

	chatView := borderStyle.Render(m.chatViewport.View())

	attachBtnView := buttonStyle.Render("[ Attach File ]")
	if m.focus == focusAttachBtn {
		attachBtnView = activeBtn.Render("[ Attach File ]")
	}

	sendBtnView := buttonStyle.Render("[ Send ]")
	if m.focus == focusSendBtn {
		sendBtnView = activeBtn.Render("[ Send ]")
	}

	msgInputView := lipgloss.NewStyle().Width(m.width - 35).Render(m.msgInput.View())
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Center, attachBtnView, "  ", msgInputView, "  ", sendBtnView)
	bottomRowBox := borderStyle.Render(bottomRow)

	statusText := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("● Disconnected")
	if m.connected {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("● Connected")
	}
	statusLine := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, statusText)

	mainUI := lipgloss.JoinVertical(lipgloss.Left, savePathView, chatView, statusLine, bottomRowBox)
	if m.state == statePopup {
		popupContent := "Select Attachment Path:\n\n"
		popupContent += m.popupPathInput.View() + "\n\n"
		popupContent += "Select Client:\n"

		for i, client := range m.clients {
			cursor := " "
			if m.clientCursor == i {
				cursor = ">"
			}
			popupContent += fmt.Sprintf("%s %s\n", cursor, client)
		}

		popupContent += "\n(Press Enter to confirm, Esc to cancel)"

		popupView := popupStyle.Render(popupContent)

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popupView)
	}

	return mainUI
}
