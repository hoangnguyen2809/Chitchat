package main

import (
	"bufio"
	"encoding/json"
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
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*websocket.Conn]*Client),
		rooms:      make(map[string]*Room),
	}
}

func (s *Server) CreateRoom(name string, client *Client, password string) *Room {
	room := NewRoom(name)
	room.register <- client
	room.broadcast <- []byte(client.name + " has joined the room")
	log.Printf("Room created")
	go room.Run()
	return room
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

	// Read the initial message to get the username
	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	var initMsg struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(message, &initMsg); err != nil {
		log.Println("json unmarshal:", err)
		return
	}

	client := &Client{name: initMsg.Name, conn: c, send: make(chan []byte, 256)}
	s.clients[c] = client

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
	http.HandleFunc("/ws", server.HandleConnection)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./static/welcome.html")
		} else if r.URL.Path == "/chatbox" {
			http.ServeFile(w, r, "./static/chatbox.html")
		} else {
			http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
		}
	})

	log.Printf("Starting server on %s", *addr)
	go func() {
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
		case "list clients":
			log.Println(server.clientStat())
		}
	}
}
