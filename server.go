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

// Client represents a connected client
type Client struct {
    name    string
    conn    *websocket.Conn
    partner *Client
    send    chan []byte
}

// Server manages all connected clients and waiting clients
type Server struct {
    clients      map[*websocket.Conn]*Client
    waiting      []*Client
    clientsMutex sync.Mutex
}

// NewServer creates a new Server instance
func NewServer() *Server {
    return &Server{
        clients:  make(map[*websocket.Conn]*Client),
        waiting:  make([]*Client, 0),
    }
}

// HandleConnection upgrades the HTTP connection to a WebSocket and manages client communication
func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
    //Upgrades HTTP connection to WebSocket and manages client communication
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    // Close the connection when the function returns
    defer c.Close()

    // Getting username
    _, username, err := c.ReadMessage()
    if err != nil {
        log.Println("read:", err)
        return
    }

    // Create a new client
    client := &Client{name: string(username), conn: c, send: make(chan []byte, 256)}
    s.clientsMutex.Lock()
    s.clients[c] = client
    s.clientsMutex.Unlock()

    //Update the client count and pair the client
    s.broadcastClientCount()
    s.pairClient(client)

    // Handle incoming messages and send them to the partner
    for {
        _, message, err := c.ReadMessage()
        if err != nil {
            break
        }

        if string(message) == "[STOP]" {
            s.handleStopMessage(client)
			s.pairWaitingClients()
			continue
        } else if client.partner != nil {
            formattedMessage := fmt.Sprintf("[MSG]: %s", message)
            client.partner.conn.WriteMessage(websocket.TextMessage, []byte(formattedMessage))
        }
    }

    // Remove the client from the server
    delete(s.clients, c)
    s.removeFromWaitingList(client)

    // Update the client count and pair the waiting clients
    s.clientsMutex.Lock()
    if client.partner != nil {
        formatNoti := fmt.Sprintf("[NOTI1]: %s", client.name)
        client.partner.conn.WriteMessage(websocket.TextMessage, []byte(formatNoti))
        client.partner.partner = nil
        s.waiting = append(s.waiting, client.partner)
    }
    s.clientsMutex.Unlock()
    s.pairWaitingClients()
    s.broadcastClientCount()
}

// handleStopMessage processes the STOP order from a client, adding them to the waiting list and notifying their partner
func (s *Server) handleStopMessage(client *Client) {
    s.clientsMutex.Lock()
    defer s.clientsMutex.Unlock()

    if client.partner != nil {
        partnerNoti := fmt.Sprintf("[NOTI1]: %s ", client.name)
        client.partner.conn.WriteMessage(websocket.TextMessage, []byte(partnerNoti))
        client.partner.partner = nil

        s.waiting = append(s.waiting, client.partner)
    }

    if client != nil {
        client.partner = nil
        s.waiting = append(s.waiting, client)
    }
}

// pairClient pairs a client with another waiting client or adds them to the waiting list
func (s *Server) pairClient(client *Client) {
    s.clientsMutex.Lock()
    defer s.clientsMutex.Unlock()

    if len(s.waiting) > 0 {
        partner := s.waiting[0]
        s.waiting = s.waiting[1:]

        client.partner = partner
        partner.partner = client

        notifyClient := fmt.Sprintf("[CONNECT]: %s.", partner.name)
        notifyPartner := fmt.Sprintf("[CONNECT]: %s.", client.name)
        client.conn.WriteMessage(websocket.TextMessage, []byte(notifyClient))
        partner.conn.WriteMessage(websocket.TextMessage, []byte(notifyPartner))
    } else {
        s.waiting = append(s.waiting, client)
        client.conn.WriteMessage(websocket.TextMessage, []byte("[ONWAIT]"))
    }
}

// pairWaitingClients pairs clients from the waiting list
func (s *Server) pairWaitingClients() {
    s.clientsMutex.Lock()
    defer s.clientsMutex.Unlock()

    for len(s.waiting) > 1 {
        client := s.waiting[0]
        partner := s.waiting[1]
        s.waiting = s.waiting[2:]

        client.partner = partner
        partner.partner = client

        notifyClient := fmt.Sprintf("[CONNECT]: %s.", partner.name)
        notifyPartner := fmt.Sprintf("[CONNECT]: %s.", client.name)
        client.conn.WriteMessage(websocket.TextMessage, []byte(notifyClient))
        partner.conn.WriteMessage(websocket.TextMessage, []byte(notifyPartner))
    }
}

// removeFromWaitingList removes a client from the waiting list
func (s *Server) removeFromWaitingList(client *Client) {
    s.clientsMutex.Lock()
    defer s.clientsMutex.Unlock()

    for i, c := range s.waiting {
        if c == client {
            s.waiting = append(s.waiting[:i], s.waiting[i+1:]...)
            break
        }
    }
}

// waitingList returns a string with the current waiting clients
func (s *Server) waitingList() string {
	waitingCount := len(s.waiting)
	waitingList := fmt.Sprintf("There are currently %d clients waiting:\n", waitingCount)

	for _, c := range s.waiting {
		waitingList += fmt.Sprintf("%s\n", c.name)
	}

	return waitingList
}

// broadcastClientCount sends the current client count to all connected clients
func (s *Server) broadcastClientCount() int{
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	count := len(s.clients)
	message := fmt.Sprintf("[COUNT]:%d", count)
	for _, client := range s.clients {
		client.conn.WriteMessage(websocket.TextMessage, []byte(message))
	}

	return count
}


// clientStat returns a string with the current connected clients and their statuses
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


