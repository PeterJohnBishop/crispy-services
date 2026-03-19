package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// model

type Model struct {
	status    int
	message   textinput.Model
	savePath  textinput.Model
	filePath  textinput.Model
	senderID  string
	targetIDs []string
	err       error
}

func initialModel() Model {
	messageti := textinput.New()
	messageti.Placeholder = "hey!"
	messageti.Focus()
	messageti.CharLimit = 256
	messageti.Width = 64

	saveti := textinput.New()
	saveti.Placeholder = "./save_file_directory/here"
	saveti.CharLimit = 256
	saveti.Width = 64

	fileti := textinput.New()
	fileti.Placeholder = "./file_location/here"
	fileti.CharLimit = 265
	fileti.Width = 64

	return Model{
		status:    0,
		message:   messageti,
		savePath:  saveti,
		filePath:  fileti,
		senderID:  "",
		targetIDs: []string{},
		err:       nil,
	}

}

// status indicator (is the websocket client connected to the server?)

// define a savePath for files, if none is defined show warning

// select target for message or file (popup)

// message input

// file viewer

// file path input (popup)

// send button, if target not defined disable button
