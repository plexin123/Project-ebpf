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
	callStackTracer := callstack.New()
	connectionStructure := broadcast.New()
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
