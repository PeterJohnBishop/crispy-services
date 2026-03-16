package ws

import (
	"crypto/rand"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type     string `json:"type"`
	SenderID string `json:"sender_id"`
	Content  any    `json:"content"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan Event
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
		c.Conn.Close()
	}()

	for {
		var event Event
		err := c.Conn.ReadJSON(&event)
		if err != nil {
			break
		}
		HandleEvent(hub, event)
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

	client := &Client{
		ID:   id,
		Conn: conn,
		Send: make(chan Event),
	}

	go client.WritePump()
	go client.ReadPump(hub)

	client.Send <- Event{
		Type:     "connected",
		SenderID: "SERVER",
		Content:  id,
	}

	hub.Register <- client
}

func HandleEvent(hub *Hub, event Event) {
	// add switch cases here for different event types
	hub.Broadcast <- event
}
