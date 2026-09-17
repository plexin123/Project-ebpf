package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ConnectionStructure struct {
	connectionMu  sync.Mutex
	connectionMap map[*websocket.Conn]bool
	upgrader      websocket.Upgrader
}

func NewConnectionStructure() *ConnectionStructure {
	connectionStructure := &ConnectionStructure{
		connectionMap: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	return connectionStructure
}

func (cs *ConnectionStructure) handleWS(w http.ResponseWriter, r *http.Request) {
	// Connect to websocket server
	c, err := cs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("There has been an error: %v", err)
		return
	}
	fmt.Printf("Client connected: %v", c.RemoteAddr())
	defer c.Close()
	// client is connected to the websocket server then send data
	cs.connectionMu.Lock()
	cs.connectionMap[c] = true
	cs.connectionMu.Unlock()

	defer func() {
		cs.connectionMu.Lock()
		delete(cs.connectionMap, c)
		cs.connectionMu.Unlock()
	}()

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

func (cs *ConnectionStructure) Broadcast(data any) {
	// creating a new struct
	cs.connectionMu.Lock()
	defer cs.connectionMu.Unlock()
	fmt.Printf("map of connections %v ", cs.connectionMap)
	for conn := range cs.connectionMap {
		fmt.Printf("Sending data %v \n", data)
		if err := conn.WriteJSON(data); err != nil {
			fmt.Printf("There has been an error %v \n", err)
			conn.Close()
			delete(cs.connectionMap, conn)
		}
		fmt.Printf("Data has been sent successfully: %v \n", data)
	}
}
