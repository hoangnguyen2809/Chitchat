package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

type Client struct {
	name string
	room *Room
	conn *websocket.Conn
	send chan []byte
}

type Server struct {
	clients    map[*websocket.Conn]*Client
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*websocket.Conn]*Client),
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (s *Server) CreateRoom(name string) *Room {
	room := NewRoom(name)
	s.rooms[room.id] = room
	go room.Run()
	return room
}


func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client.conn] = client
		case client := <-s.unregister:
			if _, ok := s.clients[client.conn]; ok {
				delete(s.clients, client.conn)
				client.conn.Close()
			}
		}
	}
}


func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	//calls the Upgrader.Upgrade method from an HTTP request handler to get a *Conn
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

    log.Printf("Client connected: %s", c.RemoteAddr().String()) 
	err = c.WriteMessage(websocket.TextMessage, []byte("Welcome to the chat server! \n Please enter your name:"))
	if err != nil {
		log.Println("write:", err)
	}
	_, msg, err := c.ReadMessage()
	log.Printf("Received client's username: %s", msg)
	client := &Client{name: string(msg), conn: c, send: make(chan []byte, 256)}
	s.register <- client

	// loop to read messages from the client 
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("Received message from client: %s", msg)

		//handle the message
		switch string(msg) {
		case "list rooms":
			err = c.WriteMessage(websocket.TextMessage, []byte(s.listRoom()))
			if err != nil {
				log.Println("write:", err)
			}
		case "list clients":
			err = c.WriteMessage(websocket.TextMessage, []byte(s.listClients(client)))
			if err != nil {
				log.Println("write:", err)
			}
		}
		// err = c.WriteMessage(websocket.TextMessage, msg)
		// if err != nil {
		// 	log.Println("write:", err)
		// 	break
		// }
	}

	s.unregister <- client
    log.Printf("Client disconnected: %s", c.RemoteAddr().String())
}

func (s *Server) listRoom() string {
	var roomList string
	for i, room := range s.rooms {
		roomList += i + room.name + "\n"
	}

	return roomList
}

func (s *Server) listClients(client *Client) string {
	clientCount := len(s.clients)
	clientList := fmt.Sprintf("There are currently %d clients connected:\n", clientCount)

	for _, c := range s.clients {
		if client != nil && c == client {
			clientList += c.name + " (You)\n"
		} else {
			clientList += c.name + "\n"
		}
	}

	return clientList
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	server := NewServer()
	go server.Run()
	http.HandleFunc("/", server.HandleConnection)
	log.Printf("Starting server on %s", *addr)
	go func ()  {
		err := (http.ListenAndServe(*addr, nil))
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		} else {
			log.Println("Server started, waiting for connection")
		}
	}()
	
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		switch text {
		case "list rooms":
			log.Println(server.listRoom())
		case "list clients":
			log.Println(server.listClients(nil))
		}
	}
}