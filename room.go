package main

type Room struct {
	Name string
	Clients map[*Client]bool
	Slots int
	Password string
	broadcast chan []byte
	register chan *Client
	unregister chan *Client
}
