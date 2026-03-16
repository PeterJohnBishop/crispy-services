package main

import (
	wsclient "webrtc/websocket"
)

func main() {
	wsclient.StartInternalClient("localhost:8081", "/ws")
	select {}
}
