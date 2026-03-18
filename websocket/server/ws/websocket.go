package ws

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Event struct {
	Type     string `json:"type"`
	SenderID string `json:"sender_id"`
	TargetID string `json:"target_id"`
	Content  any    `json:"content"`
}

type Client struct {
	ID          string
	DisplayName string
	Conn        *websocket.Conn
	Send        chan Event
	PC          *webrtc.PeerConnection
}

type Hub struct {
	Clients    map[*Client]bool
	Directory  map[string]*Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan Event
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateID(length int) string {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "error0"
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Directory:  make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan Event),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			if client.ID != "" {
				h.Directory[client.ID] = client
			}

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				if client.ID != "" {
					delete(h.Directory, client.ID)
				}
				close(client.Send)
			}

		case event := <-h.Broadcast:
			// if targetID is set, only send to the target
			if event.TargetID != "" {
				if target, ok := h.Directory[event.TargetID]; ok {
					target.Send <- event
				}
				continue
			}
			// if targetID is not set, broadcast
			for client := range h.Clients {
				if client.ID == event.SenderID {
					continue
				}
				select {
				case client.Send <- event:
				default:
					close(client.Send)
					delete(h.Clients, client)
					if client.ID != "" {
						delete(h.Directory, client.ID)
					}
				}

			}
		}
	}
}

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.PC.Close()
		c.Conn.Close()
	}()

	for {
		var event Event
		err := c.Conn.ReadJSON(&event)
		if err != nil {
			break
		}
		HandleEvent(hub, event, c.PC)
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for event := range c.Send {
		err := c.Conn.WriteJSON(event)
		if err != nil {
			return
		}
	}
}

func HandleWebsocket(hub *Hub, ctx *gin.Context) {
	id := GenerateID(8)

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		conn.Close()
		return
	}

	client := &Client{
		ID:   id,
		Conn: conn,
		Send: make(chan Event),
		PC:   pc,
	}

	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		// Send the candidate to the other peer via the hub
		// (Note: You'll need logic to know who the 'target' is)
	})

	go client.WritePump()
	go client.ReadPump(hub)

	client.Send <- Event{
		Type:     "connected",
		SenderID: "SERVER",
		Content:  id,
	}

	hub.Register <- client
}

func HandleEvent(hub *Hub, event Event, pc *webrtc.PeerConnection) {
	// convert map[string]any -> WebRTC structs
	convertToStruct := func(src any, dst any) {
		b, _ := json.Marshal(src)
		json.Unmarshal(b, dst)
	}

	switch event.Type {
	case "identify":
		if name, ok := event.Content.(string); ok {
			if client, exists := hub.Directory[event.SenderID]; exists {
				client.DisplayName = name
				fmt.Printf("User %s identified as %s\n", event.SenderID, name)
			}
		} else {
			log.Printf("Identify event received, but Content was not a string: %v", event.Content)
			// send error event and remove the client from the hub???
		}
	case "offer":
		var offer webrtc.SessionDescription
		convertToStruct(event.Content, &offer)

		pc.SetRemoteDescription(offer)
		answer, _ := pc.CreateAnswer(nil)
		pc.SetLocalDescription(answer)

		hub.Broadcast <- Event{
			Type:     "answer",
			SenderID: "SERVER", // Or the client's ID
			TargetID: event.SenderID,
			Content:  answer,
		}

	case "answer":
		var answer webrtc.SessionDescription
		convertToStruct(event.Content, &answer)
		pc.SetRemoteDescription(answer)

	case "candidate":
		var candidate webrtc.ICECandidateInit
		convertToStruct(event.Content, &candidate)
		pc.AddICECandidate(candidate)

	default:
		hub.Broadcast <- event
	}
}
