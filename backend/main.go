package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func randomRoomID() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	manager := NewRoomManager()

	// 创建默认房间
	defaultRoom := manager.CreateRoom("ROOM01")
	log.Printf("默认房间已创建: %s", defaultRoom.ID)

	// WebSocket 入口
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(manager, w, r)
	})

	// 获取房间列表
	http.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			rooms := manager.ListRooms()
			if rooms == nil {
				rooms = []RoomInfo{}
			}
			_ = json.NewEncoder(w).Encode(rooms)
		case http.MethodPost:
			roomID := randomRoomID()
			for manager.GetRoom(roomID) != nil {
				roomID = randomRoomID()
			}
			room := manager.CreateRoom(roomID)
			_ = json.NewEncoder(w).Encode(map[string]string{"roomId": room.ID})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 静态文件服务
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	log.Println("服务器启动在 :8800 ...")
	err := http.ListenAndServe(":8800", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
