package main

import wsclient "webrtc/websocket"

func main() {
	wsclient.StartWsClient("localhost:8081", "/ws", "/Users/m4pro/Downloads")
	select {}
}
