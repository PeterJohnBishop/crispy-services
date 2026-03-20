package main

import (
	"webrtc/tui"
	wsclient "webrtc/websocket"
)

func main() {
	statusChan := make(chan bool)

	go wsclient.StartWsClient("localhost:8081", "/ws", "/Users/m4pro/Downloads", statusChan)

	tui.BrewTui(statusChan)
}
