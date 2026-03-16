package wsclient

import (
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type Event struct {
	Type     string `json:"type"`
	SenderID string `json:"sender_id"`
	Content  any    `json:"content"`
}

var (
	assignedID string
	idMu       sync.RWMutex // Protects assignedID from race conditions
)

func StartInternalClient(addr string, path string) {
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	log.Printf("Client connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Dial error: %v", err)
		return
	}

	send := make(chan Event)

	// receiving from the server
	go func() {
		defer conn.Close()
		for {
			var event Event
			err := conn.ReadJSON(&event)
			if err != nil {
				log.Printf("Client read error: %v", err)
				return
			}

			switch event.Type {
			case "connected":
				idMu.Lock()
				assignedID = fmt.Sprint(event.Content)
				idMu.Unlock()
				log.Printf("Client assigned ID: %s", assignedID)

			case "broadcast":
				log.Printf("[%s says]: %v", event.SenderID, event.Content)

			default:
				log.Printf("Client received unknown event: %s", event.Type)
			}
		}
	}()

	// sending to the server
	go func() {
		defer conn.Close()
		for event := range send {
			err := conn.WriteJSON(event)
			if err != nil {
				log.Printf("Client write error: %v", err)
				return
			}
		}
	}()
}

// GetAssignedID provides thread-safe access to the global ID
func GetAssignedID() string {
	idMu.RLock()
	defer idMu.RUnlock()
	return assignedID
}
