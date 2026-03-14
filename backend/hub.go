package main

type Hub struct {
	clients    map[*Client]bool //所有在线客户端
	broadcast  chan []byte      //广播通道
	register   chan *Client     //注册
	unregister chan *Client     //注销
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		// 新人加入
		case client := <-h.register:
			h.clients[client] = true
			// 调用 Uno.go

		// 离开
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// 通知 Uno.go 该玩家掉线3
			}

		// 处理广播消息
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 如果发送失败，说明连接已断开，清理该客户端
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
