package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os"
)

func startServer(server *Server) {
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
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	server := NewServer()

	go startServer(server)

	// Command line interface
	// 1. Print client statistics
	// 2. Print waiting list
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		switch text {
		case "1":
			log.Println(server.clientStat())
		case "2":
			log.Println(server.waitingList())
		}
	}
}