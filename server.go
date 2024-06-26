package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
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
	s.broadcastClientCount()
	s.pairClient(client)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		log.Printf("recv: %s", message)
		if client.partner != nil {
			formattedMessage := fmt.Sprintf("[MSG]:[%s]: %s", client.name, message)
			client.partner.conn.WriteMessage(websocket.TextMessage, []byte(formattedMessage))
		}
	}
	s.clientsMutex.Lock()
	delete(s.clients, c)
	log.Printf("Client %s disconnected", client.name)
	if client.partner != nil {
		formatNoti := fmt.Sprintf("[NOTI]: %s has disconnected.", client.name)
		client.partner.conn.WriteMessage(websocket.TextMessage, []byte(formatNoti))
		client.partner.partner = nil
		log.Printf("Putting %s back in the waiting list", client.partner.name)
		s.waiting = append(s.waiting, client.partner)
	}
	s.clientsMutex.Unlock()
	s.pairWaitingClients(client)
	s.broadcastClientCount()
	log.Println(s.clientStat())
}

func (s *Server) pairClient(client *Client) {
	log.Print("Pairing client")
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if len(s.waiting) > 0 {
		partner := s.waiting[0]
		s.waiting = s.waiting[1:]

		client.partner = partner
		partner.partner = client

		//Log
		log.Printf("Pairing %s with %s", client.name, partner.name)

		notifyClient := fmt.Sprintf("[NOTI]:You are now connected to %s.", partner.name)
		notifyPartner := fmt.Sprintf("[NOTI]:You are now connected to %s.", client.name)
		client.conn.WriteMessage(websocket.TextMessage, []byte(notifyClient))
		partner.conn.WriteMessage(websocket.TextMessage, []byte(notifyPartner))
	} else {
		s.waiting = append(s.waiting, client)
		//Log
		log.Printf("%s is waiting", client.name)
		client.conn.WriteMessage(websocket.TextMessage, []byte("[NOTI]:Waiting for a stranger to connect..."))
	}
}

func (s *Server) pairWaitingClients(client *Client) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if(len(s.waiting) == 1) {
		s.waiting = append(s.waiting, client)
		log.Printf("%s is waiting", client.name)
		client.conn.WriteMessage(websocket.TextMessage, []byte("[NOTI]:Waiting for a stranger to connect..."))
	}
	for len(s.waiting) > 1 {
		client := s.waiting[0]
		partner := s.waiting[1]
		s.waiting = s.waiting[2:]

		client.partner = partner
		partner.partner = client

		// Log
		log.Printf("Pairing %s with %s from waiting list", client.name, partner.name)

		notifyClient := fmt.Sprintf("[NOTI]:You are now connected to %s.", partner.name)
		notifyPartner := fmt.Sprintf("[NOTI]:You are now connected to %s.", client.name)
		client.conn.WriteMessage(websocket.TextMessage, []byte(notifyClient))
		partner.conn.WriteMessage(websocket.TextMessage, []byte(notifyPartner))
	}

	
	
}

func (s *Server) waitingList() string {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	waitingCount := len(s.waiting)
	waitingList := fmt.Sprintf("There are currently %d clients waiting:\n", waitingCount)

	for _, c := range s.waiting {
		waitingList += fmt.Sprintf("%s\n", c.name)
	}

	return waitingList
}

func (s *Server) broadcastClientCount() {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	count := len(s.clients)
	message := fmt.Sprintf("[COUNT]:%d", count)
	for _, client := range s.clients {
		client.conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}

func (s *Server) clientStat() string {
	clientCount := len(s.clients)
	clientList := fmt.Sprintf("There are currently %d clients connected:\n", clientCount)

	for _, c := range s.clients {
		var partnerName string
		if c.partner != nil {
			partnerName = c.partner.name
		} else {
			partnerName = "No partner"
		}
		clientList += fmt.Sprintf("%-20s %s %s\n", c.name, c.conn.RemoteAddr().String(), partnerName)
	}

	return clientList
}


