package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ClientInfo struct {
	Name string
}

var upgrader = websocket.Upgrader{

	CheckOrigin: func(r *http.Request) bool { return true },
}

// REGISTER CLIENTS
// HASHMAP
var connectionMap = make(map[*websocket.Conn]bool)
var connectionMu sync.Mutex

// LISTENING FOR CLIENTS

func handleWS(w http.ResponseWriter, r *http.Request) {
	// Connect to websocket server
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("There has been an error: %v", err)
	}
	defer c.Close()
	// client is connected to the websocket server then send data
	connectionMu.Lock()
	connectionMap[c] = true
	connectionMu.Unlock()
}

func broadcast(data any) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	for conn := range connectionMap {
		fmt.Printf("Sending data %v \n", data)
		if err := conn.WriteJSON(data); err != nil {
			fmt.Printf("There has been an error %v \n", err)
			conn.Close()
			delete(connectionMap, conn)
		}
		fmt.Printf("Data has been sent successfully: %v \n", data)
	}
}
