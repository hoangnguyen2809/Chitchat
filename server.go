package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

// type Server struct {
// 	Rooms map[string]*Room
// }

//handle the WebSocket protocol upgrade.
var upgrader = websocket.Upgrader{}


func HandleConnections(w http.ResponseWriter, r *http.Request) {
	//calls the Upgrader.Upgrade method from an HTTP request handler to get a *Conn
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
	// Log client connection
    log.Printf("Client connected: %s", c.RemoteAddr().String()) 
    defer c.Close()
    log.Printf("Client disconnected: %s", c.RemoteAddr().String()) // Log client disconnection
}


func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", HandleConnections)
	http.HandleFunc("/", HandleConnections)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

