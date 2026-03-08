package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte // 待发送
	id   string      // 唯一ID
}

func (x *Client) ReadPump(game *Game) {
	defer func() {
		x.hub.unregister <- x
		x.conn.Close()
	}()
	for {
		_, message, err := x.conn.ReadMessage()
		if err != nil {
			log.Printf("err:connet err:%v", err)
			break
		}
		var action struct {
			Type   string `json:"type"`
			CardId int    `json:"cardId"` //哪张牌
		}
		if err := json.Unmarshal(message, &action); err != nil {
			continue
		}
		switch action.Type {
		case "PLAY":
			var p *Player
			for _, player := range game.Players {
				if player.ID == x.id {
					p = player
					break
				}
			}
			if p == nil {
				continue
			}
			ok, msg := CheckCard(game, p, action.CardId)
			if ok { //合法
				PlayAction(game, p, action.CardId)
				state, _ := json.Marshal(game)
				x.hub.broadcast <- state
			} else {
				errMsg, _ := json.Marshal(map[string]string{"error": msg})
				x.send <- errMsg
			}
		case "DRWA":
			var p *Player
			for _, player := range game.Players {
				if player.ID == x.id {
					p = player
					break
				}
			}
			if p == nil {
				continue
			}
			DrawCard(game, p)
			state, _ := json.Marshal(game)
			x.hub.broadcast <- state
		}
	}
}

func (c *Client) WritePump() {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func serveWs(hub *Hub, game *Game, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// 暂时用时间戳当 ID
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		id:   time.Now().Format("150405"),
	}

	client.hub.register <- client

	go client.WritePump()
	go client.ReadPump(game)
}
