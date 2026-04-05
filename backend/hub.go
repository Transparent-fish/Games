package main

// Hub 负责管理所有 WebSocket 客户端的注册和注销
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		}
	}
}
