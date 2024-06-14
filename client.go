package main

import (
	"bufio"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	// Set up interrupt handler
	interrupt := make(chan os.Signal, 1)
	// Notify sends os.Interrupt (in this case CtrlC) to interrupt channel
	signal.Notify(interrupt, os.Interrupt)

	// Create a new URL struct
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	// Dial the server
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err) // if dial fails, log the error and exit
	}
	defer c.Close() 

	// channel to signal when the connection is closed
	done := make(chan struct{})

	// Read messages from the server
	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s, type: %s", message, websocket.FormatMessageType(mt))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			err := c.WriteMessage(websocket.TextMessage, []byte(text))
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			log.Println("scanner error:", err)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Send close message to server
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}

			// Wait for server to close the connection
			select {
			// if connection is closed, return
			case <-done:
			// set timeout, if connection isnt closed in 1 second, close it manually
			case <-time.After(time.Second):
			}
			return
		}
	}
}