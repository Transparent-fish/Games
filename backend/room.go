package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Room struct {
	ID          string
	Hub         *Hub
	Game        *Game
	gameMu      sync.Mutex
	clientsMu   sync.Mutex
	boundPlayer map[string]*Client
}

func NewRoom(id string) *Room {
	hub := NewHub()
	go hub.Run()

	game := &Game{}
	game.Players = append(game.Players, &Player{ID: "p1", Name: "Player 1"})
	game.Players = append(game.Players, &Player{ID: "p2", Name: "Player 2"})
	InitGame(game)

	return &Room{
		ID:          id,
		Hub:         hub,
		Game:        game,
		boundPlayer: make(map[string]*Client),
	}
}

func (r *Room) BindClientID(c *Client, requested string) error {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()

	requested = strings.TrimSpace(requested)
	if requested != "" {
		if requested != "p1" && requested != "p2" {
			return fmt.Errorf("playerId 只支持 p1 或 p2")
		}
		if existing, ok := r.boundPlayer[requested]; ok && existing != c {
			return fmt.Errorf("%s 已连接，请先断开旧连接", requested)
		}
		r.boundPlayer[requested] = c
		c.id = requested
		return nil
	}

	for _, id := range []string{"p1", "p2"} {
		if _, ok := r.boundPlayer[id]; !ok {
			r.boundPlayer[id] = c
			c.id = id
			return nil
		}
	}

	c.id = fmt.Sprintf("guest-%s", time.Now().Format("150405"))
	return nil
}

func (r *Room) UnbindClientID(c *Client) {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()
	for id, client := range r.boundPlayer {
		if client == c {
			delete(r.boundPlayer, id)
			return
		}
	}
}

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
