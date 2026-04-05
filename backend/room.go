package main

import (
	"fmt"
	"sync"
)

// ─── Room ─────────────────────────────────────────────

type Room struct {
	ID      string
	Hub     *Hub
	Game    *Game
	GameMu  sync.Mutex
	Clients map[*Client]bool // 本房间的所有客户端
	mu      sync.Mutex
	nextPID int // 自增玩家 ID 计数器
}

func NewRoom(id string) *Room {
	hub := NewHub()
	go hub.Run()

	return &Room{
		ID:      id,
		Hub:     hub,
		Game:    &Game{Status: "waiting", Direction: 1},
		Clients: make(map[*Client]bool),
		nextPID: 1,
	}
}

// AddPlayer 加入房间，返回分配的玩家 ID
func (r *Room) AddPlayer(name string) (*Player, error) {
	if r.Game.Status != "waiting" {
		return nil, fmt.Errorf("游戏已开始，无法加入")
	}
	if len(r.Game.Players) >= 10 {
		return nil, fmt.Errorf("房间已满（最多10人）")
	}
	// 检查昵称重复
	for _, p := range r.Game.Players {
		if p.Name == name {
			return nil, fmt.Errorf("昵称 \"%s\" 已被使用", name)
		}
	}

	pid := fmt.Sprintf("p%d", r.nextPID)
	r.nextPID++

	player := &Player{
		ID:        pid,
		Name:      name,
		Cards:     nil,
		CardCount: 0,
		IsHost:    len(r.Game.Players) == 0, // 第一个加入的是房主
		Online:    true,
	}
	r.Game.Players = append(r.Game.Players, player)
	return player, nil
}

// RemovePlayer 玩家离开房间
func (r *Room) RemovePlayer(playerID string) {
	for _, p := range r.Game.Players {
		if p.ID == playerID {
			p.Online = false
			break
		}
	}
	// 检查是否全员离线
	allOffline := true
	for _, p := range r.Game.Players {
		if p.Online {
			allOffline = false
			break
		}
	}
	if allOffline && len(r.Game.Players) > 0 {
		// 重置房间
		r.Game = &Game{Status: "waiting", Direction: 1}
		r.nextPID = 1
	}
}

// FindPlayer 根据 ID 查找玩家
func (r *Room) FindPlayer(playerID string) *Player {
	for _, p := range r.Game.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// StartGame 房主开始游戏
func (r *Room) StartGame(playerID string) error {
	if r.Game.Status != "waiting" {
		return fmt.Errorf("游戏已经在进行中")
	}
	// 检查是否是房主
	p := r.FindPlayer(playerID)
	if p == nil || !p.IsHost {
		return fmt.Errorf("只有房主可以开始游戏")
	}
	if len(r.Game.Players) < 2 {
		return fmt.Errorf("至少需要2名玩家才能开始")
	}

	InitGame(r.Game)
	return nil
}

// BroadcastState 向房间内每个客户端发送其个人视图
func (r *Room) BroadcastState() {
	r.mu.Lock()
	clients := make(map[*Client]bool)
	for c := range r.Clients {
		clients[c] = true
	}
	r.mu.Unlock()

	for c := range clients {
		c.SendGameState()
	}
}

// RegisterClient 将客户端注册到房间
func (r *Room) RegisterClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Clients[c] = true
}

// UnregisterClient 注销客户端
func (r *Room) UnregisterClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Clients, c)
}

// ─── RoomManager ──────────────────────────────────────

type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewRoomManager() *RoomManager {
	return &RoomManager{rooms: make(map[string]*Room)}
}

func (m *RoomManager) CreateRoom(id string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	if room, ok := m.rooms[id]; ok {
		return room
	}
	room := NewRoom(id)
	m.rooms[id] = room
	return room
}

func (m *RoomManager) GetRoom(id string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id]
}

// ListRooms 返回所有房间的摘要信息
type RoomInfo struct {
	ID         string `json:"id"`
	PlayerNum  int    `json:"playerNum"`
	Status     string `json:"status"`
}

func (m *RoomManager) ListRooms() []RoomInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []RoomInfo
	for _, room := range m.rooms {
		list = append(list, RoomInfo{
			ID:        room.ID,
			PlayerNum: len(room.Game.Players),
			Status:    room.Game.Status,
		})
	}
	return list
}
