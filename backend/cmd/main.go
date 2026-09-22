package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"fmt"
	"log"
	"net/http"

	"ebpf-project/backend/broadcast"
	"ebpf-project/backend/callstack"
	"ebpf-project/backend/collector"
)

func main() {
	connectionStructure := broadcast.New()
	// the functions that appeared in Broadcaster => are gonna be implemented in the instance connection Structure
	// quiero que uses estas funciones de esta instance en la que mi interface ha definido
	callStackTracer := callstack.New(connectionStructure)
	http.HandleFunc("/ws", connectionStructure.HandleWS)

	fmt.Printf("Websocket server starting.. on 8080")

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("websocket server failed %v", err)
		}
	}()

	if err := collector.Collector(callStackTracer); err != nil {
		log.Fatalf("collector failed %v", err)
	}
}
