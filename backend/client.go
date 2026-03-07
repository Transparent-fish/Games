package main

import "log"

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}
