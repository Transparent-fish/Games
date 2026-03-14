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
	defaultRoom := manager.CreateRoom("ROOM01")
	log.Printf("Default room created: %s", defaultRoom.ID)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(manager, w, r)
	})
	http.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		roomID := randomRoomID()
		for manager.GetRoom(roomID) != nil {
			roomID = randomRoomID()
		}
		room := manager.CreateRoom(roomID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"roomId": room.ID})
	})

	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	log.Println("Server starting on :8800...")
	err := http.ListenAndServe(":8800", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
