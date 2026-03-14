package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	room *Room
	hub  *Hub
	conn *websocket.Conn
	send chan []byte // 待发送
	id   string      // 唯一ID
}

func (x *Client) ReadPump() {
	defer func() {
		x.hub.unregister <- x
		x.room.UnbindClientID(x)
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
		switch strings.ToUpper(action.Type) {
		case "PLAY":
			x.room.gameMu.Lock()
			game := x.room.Game
			var p *Player
			for _, player := range game.Players {
				if player.ID == x.id {
					p = player
					break
				}
			}
			if p == nil {
				x.room.gameMu.Unlock()
				errMsg, _ := json.Marshal(map[string]string{"error": "当前连接未绑定到有效玩家(p1/p2)"})
				x.send <- errMsg
				continue
			}
			ok, msg := CheckCard(game, p, action.CardId)
			if ok { //合法
				PlayAction(game, p, action.CardId)
				state, _ := json.Marshal(game)
				x.room.gameMu.Unlock()
				x.hub.broadcast <- state
			} else {
				x.room.gameMu.Unlock()
				errMsg, _ := json.Marshal(map[string]string{"error": msg})
				x.send <- errMsg
			}
		case "DRAW", "DRWA":
			x.room.gameMu.Lock()
			game := x.room.Game
			var p *Player
			for _, player := range game.Players {
				if player.ID == x.id {
					p = player
					break
				}
			}
			if p == nil {
				x.room.gameMu.Unlock()
				errMsg, _ := json.Marshal(map[string]string{"error": "当前连接未绑定到有效玩家(p1/p2)"})
				x.send <- errMsg
				continue
			}
			if game.Players[game.NowID].ID != p.ID {
				x.room.gameMu.Unlock()
				errMsg, _ := json.Marshal(map[string]string{"error": "没到你摸牌"})
				x.send <- errMsg
				continue
			}
			DrawCard(game, p)
			state, _ := json.Marshal(game)
			x.room.gameMu.Unlock()
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

func serveWs(manager *RoomManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	if roomID == "" {
		errMsg, _ := json.Marshal(map[string]string{"error": "缺少 roomId"})
		_ = conn.WriteMessage(websocket.TextMessage, errMsg)
		_ = conn.Close()
		return
	}
	room := manager.GetRoom(roomID)
	if room == nil {
		errMsg, _ := json.Marshal(map[string]string{"error": "房间不存在"})
		_ = conn.WriteMessage(websocket.TextMessage, errMsg)
		_ = conn.Close()
		return
	}

	client := &Client{
		room: room,
		hub:  room.Hub,
		conn: conn,
		send: make(chan []byte, 256),
		id:   "",
	}

	if err := room.BindClientID(client, r.URL.Query().Get("playerId")); err != nil {
		errMsg, _ := json.Marshal(map[string]string{"error": err.Error()})
		_ = conn.WriteMessage(websocket.TextMessage, errMsg)
		_ = conn.Close()
		return
	}

	client.hub.register <- client

	room.gameMu.Lock()
	state, _ := json.Marshal(room.Game)
	room.gameMu.Unlock()
	client.send <- state

	go client.WritePump()
	go client.ReadPump()
}
