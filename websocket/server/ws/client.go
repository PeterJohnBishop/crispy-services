package ws

import (
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

func StartInternalClient(addr string, path string) {
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	log.Printf("Internal client connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Dial error: %v", err)
		return
	}

	send := make(chan Event)

	// recieving from the server
	go func() {
		defer conn.Close()
		for {
			var event Event
			err := conn.ReadJSON(&event)
			if err != nil {
				log.Printf("Internal client read error: %v", err)
				return
			}
			log.Printf("Internal client received: %s - %s", event.Type, event.Content)
		}
	}()

	// sending to the server
	go func() {
		defer conn.Close()
		for event := range send {
			err := conn.WriteJSON(event)
			if err != nil {
				log.Printf("Internal client write error: %v", err)
				return
			}
		}
	}()

	send <- Event{
		Type:    "greeting",
		Content: "Hello from the internal goroutine!",
	}
}
