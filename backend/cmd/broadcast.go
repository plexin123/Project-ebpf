package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	)


var upgrader = websocket.Upgrader{
	
	CheckOrigin: func(r *http.Request) bool { return true}
	
}


var map_of_clients := map[id]WebsocketInstance*
