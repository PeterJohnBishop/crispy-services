package main

import (
	"log"
	"webrtc/tui"
	wsclient "webrtc/websocket"
)

func main() {
	statusChan := make(chan bool)

	wsc, err := wsclient.StartWsClient("localhost:8081", "/ws", "/Users/m4pro/Downloads", statusChan)
	if err != nil {
		log.Fatalf("Error starting Websocket Client: %w", err)
	}

	tui.BrewTui(statusChan, wsc)
}
