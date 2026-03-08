package main

import (
	"log"
	"net/http"
)

func main() {
	hub := NewHub()
	go hub.Run()

	game := &Game{}
	// 初始化几个玩家用于测试
	game.Players = append(game.Players, &Player{ID: "p1", Name: "Player 1"})
	game.Players = append(game.Players, &Player{ID: "p2", Name: "Player 2"})
	InitGame(game)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, game, w, r)
	})

	log.Println("Server starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
