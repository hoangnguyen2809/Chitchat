package main

import (
	"flag"
	"log"
	"net/http"

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

	// loop to read messages from the client 
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("Received message from client: %s", msg)

		err = c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
	
    log.Printf("Client disconnected: %s", c.RemoteAddr().String())
}

func (s *Server) listRoom() string {
	var roomList string
	for i, room := range s.rooms {
		roomList += i + room.name + "\n"
	}

	return roomList
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	server := NewServer()
	go server.Run()
	http.HandleFunc("/", server.HandleConnection)
	log.Printf("Starting server on %s", *addr)
	err := (http.ListenAndServe(*addr, nil))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	} else {
		log.Println("Server started, waiting for connection")
	}
}