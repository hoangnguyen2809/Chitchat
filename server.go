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

func (s *Server) CreateRoom(name string, client *Client, password string) *Room {
	room := NewRoom(name, client, password)
	s.rooms[room.id] = room
	room.register <- client
	room.broadcast <- []byte(client.name + " has joined the room")
	log.Printf("Room created: %s", room.id)
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
	//defer the closing of the connection
	defer c.Close()


    log.Printf("Client connected: %s", c.RemoteAddr().String()) 
	err = c.WriteMessage(websocket.TextMessage, []byte("Welcome to the chat server!\nPlease enter your name:"))
	if err != nil {
		log.Println("write:", err)
	}
	
	_, msg, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}
	log.Printf("Received client's username: %s", msg)
	clientName := string(msg)
	if (clientName == "") {
		clientName = "Anonymous"
	}
	client := &Client{name: clientName, conn: c, send: make(chan []byte, 256)}
	s.register <- client

	//defer the unregistering of the client
	defer func() {
		s.unregister <- client
		log.Printf("Client disconnected: %s", c.RemoteAddr().String())
	}()

	welcomeMsg := []byte("Welcome " + client.name + "!\n" +
		"List of commands:\n" +
		"1. /list clients - List all clients\n" +
		"2. /list rooms - List all rooms\n" +
		"3. /create - Create a room\n" +
		"4. /join <room_id> - Join a room\n" + 
		"(This message is only shown once. Type /help to see it again)")
	err = c.WriteMessage(websocket.TextMessage, welcomeMsg)
	if err != nil {
		log.Println("write:", err)
		return
	}
	
	// loop to read messages from the client 
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("Received message from %s: %s", client.name ,msg)

		//handle the message
		switch string(msg) {
		case "/help":
			err = c.WriteMessage(websocket.TextMessage, []byte(s.help()))
		case "/list rooms":
			err = c.WriteMessage(websocket.TextMessage, []byte(s.listRoom()))
		case "/list clients":
			err = c.WriteMessage(websocket.TextMessage, []byte(s.listClients(client)))
		case "/create":
			err = c.WriteMessage(websocket.TextMessage, []byte("Enter password  : "))
			_, msg, err = c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			room := s.CreateRoom(client.name, client, string(msg))
			err = c.WriteMessage(websocket.TextMessage, []byte("Room created: "+room.id))
		default:
			err = c.WriteMessage(websocket.TextMessage, []byte("Unknown command: "+string(msg)))
		}
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func (s *Server) help() string {
	return "List of commands:\n" +
		"1. /list rooms - List all rooms\n" +
		"2. /list clients - List all clients\n" +
		"3. /create - Create a room\n" +
		"4. /join <room_id> - Join a room\n"
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

func (s *Server) clientStat() string {
	clientCount := len(s.clients)
	clientList := fmt.Sprintf("There are currently %d clients connected:\n", clientCount)

	for _, c := range s.clients {
		clientList += fmt.Sprintf("%-20s %s\n", c.name, c.conn.RemoteAddr().String())
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
			log.Println(server.clientStat())
		}
		
			
	}
}