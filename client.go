package main

import "github.com/gorilla/websocket"

type Client struct {
	Name string
	conn *websocket.Conn
	send chan []byte
	Room *Room
}

