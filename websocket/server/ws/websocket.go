package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan Event
}

type Hub struct {
	Clients    map[*Client]bool
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

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
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

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}

		case event := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- event:
				default:
					close(client.Send)
					delete(h.Clients, client)
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
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	client := &Client{
		Conn: conn,
		Send: make(chan Event),
	}

	hub.Register <- client

	go client.WritePump()
	go client.ReadPump(hub)
}

func HandleEvent(hub *Hub, event Event) {
	// add switch cases here for different event types
	hub.Broadcast <- event
}
