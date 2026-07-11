package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserId int
	Conn   *websocket.Conn
	Send   chan []byte
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Hub struct {
	clients    map[int]*Client
	mu         sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.UserId] = client
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, exists := h.clients[client.UserId]; exists {
				delete(h.clients, client.UserId)
				close(client.Send)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) SendToUser(userId int, msgType string, payload interface{}) {
	h.mu.RLock()
	client, exists := h.clients[userId]
	h.mu.RUnlock()

	if !exists {
		return
	}

	msg := Message{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
		h.mu.Lock()
		delete(h.clients, userId)
		close(client.Send)
		h.mu.Unlock()
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}