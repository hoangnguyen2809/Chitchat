package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	name string
	conn *websocket.Conn
	partner *Client
	send chan []byte
}

type Server struct {
	clients    map[*websocket.Conn]*Client
	waiting    []*Client
	clientsMutex sync.Mutex
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*websocket.Conn]*Client),
		waiting:    make([]*Client, 0),
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
	

	// Read the initial message to get the username
	_, username, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	client := &Client{name: string(username), conn: c, send: make(chan []byte, 256)}
	s.clientsMutex.Lock()
	s.clients[c] = client
	log.Printf("New client: %s", username)
	s.clientsMutex.Unlock()

	s.pairClient(client)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		log.Printf("recv: %s", message)
		if client.partner != nil {
			client.partner.conn.WriteMessage(websocket.TextMessage, message)
		}
	}

	s.clientsMutex.Lock()
	delete(s.clients, c)
	if client.partner != nil {
		client.partner.conn.WriteMessage(websocket.TextMessage, []byte("Stranger has disconnected."))
		client.partner.partner = nil
		s.pairClient(client.partner)
	}
	s.clientsMutex.Unlock()

	log.Printf("Client %s disconnected", client.name)
	log.Println(s.clientStat())

}

func (s *Server) pairClient(client *Client) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if len(s.waiting) > 0 {
		partner := s.waiting[0]
		s.waiting = s.waiting[1:]

		client.partner = partner
		partner.partner = client

		client.conn.WriteMessage(websocket.TextMessage, []byte("You are now connected to a stranger."))
		partner.conn.WriteMessage(websocket.TextMessage, []byte("You are now connected to a stranger."))
	} else {
		s.waiting = append(s.waiting, client)
		client.conn.WriteMessage(websocket.TextMessage, []byte("Waiting for a stranger to connect..."))
	}
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
		} else if r.URL.Path == "/chatbox.html" {
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
