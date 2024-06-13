package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	//calls the Upgrader.Upgrade method from an HTTP request handler to get a *Conn
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	
    log.Printf("Client connected: %s", c.RemoteAddr().String()) 

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

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", HandleConnection)
	log.Fatal(http.ListenAndServe(*addr, nil))
}