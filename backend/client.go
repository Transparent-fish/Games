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
	room     *Room
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	playerID string // 分配的玩家 ID
}

// ─── 消息类型定义 ──────────────────────────────────────

type InMessage struct {
	Type        string `json:"type"`                  // JOIN / START / PLAY / DRAW / PASS
	Name        string `json:"name,omitempty"`         // JOIN 时的昵称
	CardID      int    `json:"cardId,omitempty"`       // PLAY 时的牌 ID
	ChosenColor string `json:"chosenColor,omitempty"`  // PLAY 黑牌时选颜色
}

type OutMessage struct {
	Type string      `json:"type"`          // ROOM_STATE / ERROR / GAME_OVER / JOINED
	Data interface{} `json:"data,omitempty"`
	Msg  string      `json:"msg,omitempty"`
}

// ─── 发送辅助方法 ──────────────────────────────────────

func (c *Client) sendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) sendError(msg string) {
	c.sendJSON(OutMessage{Type: "ERROR", Msg: msg})
}

func (c *Client) SendGameState() {
	c.room.GameMu.Lock()
	view := GameViewForPlayer(c.room.Game, c.playerID)
	c.room.GameMu.Unlock()
	c.sendJSON(OutMessage{Type: "ROOM_STATE", Data: view})
}

// ─── 读取消息 ─────────────────────────────────────────

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.room.UnregisterClient(c)
		if c.playerID != "" {
			c.room.GameMu.Lock()
			c.room.RemovePlayer(c.playerID)
			c.room.GameMu.Unlock()
			c.room.BroadcastState()
		}
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("读取消息失败: %v", err)
			break
		}

		var action InMessage
		if err := json.Unmarshal(message, &action); err != nil {
			c.sendError("消息格式错误")
			continue
		}

		switch strings.ToUpper(action.Type) {
		case "JOIN":
			c.handleJoin(action)
		case "START":
			c.handleStart()
		case "PLAY":
			c.handlePlay(action)
		case "DRAW":
			c.handleDraw()
		default:
			c.sendError("未知的消息类型: " + action.Type)
		}
	}
}

func (c *Client) handleJoin(action InMessage) {
	name := strings.TrimSpace(action.Name)
	if name == "" {
		c.sendError("昵称不能为空")
		return
	}
	if c.playerID != "" {
		c.sendError("你已经加入了房间")
		return
	}

	c.room.GameMu.Lock()
	player, err := c.room.AddPlayer(name)
	c.room.GameMu.Unlock()

	if err != nil {
		c.sendError(err.Error())
		return
	}

	c.playerID = player.ID
	c.sendJSON(OutMessage{Type: "JOINED", Data: map[string]string{
		"playerId": player.ID,
		"name":     player.Name,
	}})
	c.room.BroadcastState()
}

func (c *Client) handleStart() {
	if c.playerID == "" {
		c.sendError("请先加入房间")
		return
	}

	c.room.GameMu.Lock()
	err := c.room.StartGame(c.playerID)
	c.room.GameMu.Unlock()

	if err != nil {
		c.sendError(err.Error())
		return
	}
	c.room.BroadcastState()
}

func (c *Client) handlePlay(action InMessage) {
	if c.playerID == "" {
		c.sendError("请先加入房间")
		return
	}

	c.room.GameMu.Lock()
	defer c.room.GameMu.Unlock()

	p := c.room.FindPlayer(c.playerID)
	if p == nil {
		c.sendError("玩家不存在")
		return
	}

	ok, msg := CheckCard(c.room.Game, p, action.CardID)
	if !ok {
		c.sendError(msg)
		return
	}

	PlayCard(c.room.Game, p, action.CardID, action.ChosenColor)

	// 解锁后再广播（BroadcastState 内部会加锁）
	go func() {
		c.room.BroadcastState()
	}()
}

func (c *Client) handleDraw() {
	if c.playerID == "" {
		c.sendError("请先加入房间")
		return
	}

	c.room.GameMu.Lock()
	p := c.room.FindPlayer(c.playerID)
	if p == nil {
		c.room.GameMu.Unlock()
		c.sendError("玩家不存在")
		return
	}

	game := c.room.Game
	if game.Players[game.NowIdx].ID != p.ID {
		c.room.GameMu.Unlock()
		c.sendError("还没轮到你摸牌")
		return
	}

	drawn := DrawCard(game, p)
	c.room.GameMu.Unlock()

	if drawn == nil {
		c.sendError("摸牌堆已空")
		return
	}

	c.room.BroadcastState()
}



// ─── 写入消息 ─────────────────────────────────────────

func (c *Client) WritePump() {
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.conn.WriteMessage(websocket.TextMessage, message)
	}
}

// ─── WebSocket 入口 ──────────────────────────────────

func serveWs(manager *RoomManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	if roomID == "" {
		errMsg, _ := json.Marshal(OutMessage{Type: "ERROR", Msg: "缺少 roomId 参数"})
		_ = conn.WriteMessage(websocket.TextMessage, errMsg)
		_ = conn.Close()
		return
	}

	room := manager.GetRoom(roomID)
	if room == nil {
		errMsg, _ := json.Marshal(OutMessage{Type: "ERROR", Msg: "房间不存在"})
		_ = conn.WriteMessage(websocket.TextMessage, errMsg)
		_ = conn.Close()
		return
	}

	client := &Client{
		room: room,
		hub:  room.Hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	room.Hub.register <- client
	room.RegisterClient(client)

	go client.WritePump()
	go client.ReadPump()
}
