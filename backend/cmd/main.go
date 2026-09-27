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
	"ebpf-project/backend/database"
)

func main() {
	PATH := "in-memory"
	connectionStructure := broadcast.New()
	databaseInstance, err := database.Open(PATH)
	if err != nil {
		fmt.Printf("Could not created a instance of a database %v", err)
	}
	handler := database.New(databaseInstance)

	// databaseStructure, err := database.Open("path")
	// if err != nil {
	// 	log.Fatalf("database initialization has failed %v", err)
	// }
	callStackTracer := callstack.New(connectionStructure)

	http.HandleFunc("/ws", connectionStructure.HandleWS)
	http.HandleFunc("/history/{name}", handler.GetHistoryByName)

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
