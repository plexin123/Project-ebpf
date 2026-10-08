package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"ebpf-project/backend/broadcast"
	"ebpf-project/backend/callstack"
	"ebpf-project/backend/collector"
	"ebpf-project/backend/database"

	"debug/buildinfo"
)

func main() {
	target_path := os.Args[1]
	info, err := buildinfo.ReadFile(target_path)
	if err != nil {
		fmt.Printf("Could read the targeted file")
		os.Exit(1)
	}
	commit_hash := ""
	for _, val := range info.Settings {
		if val.Key == "vcs.revision" {
			commit_hash = val.Value
		}
	}
	PATH := "./data/app.db"
	connectionStructure := broadcast.New()
	databaseInstance, err := database.Open(PATH)
	if err != nil {
		fmt.Printf("Could not created a instance of a database %v", err)
	}
	handler := database.New(databaseInstance)
	callStackTracer := callstack.New()

	http.HandleFunc("/ws", connectionStructure.HandleWS)
	http.HandleFunc("/history/{name}", handler.GetHistoryByName)

	fmt.Printf("Websocket server starting.. on 8080")

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("websocket server failed %v", err)
		}
	}()

	if err := collector.Collector(callStackTracer, databaseInstance, connectionStructure, commit_hash); err != nil {
		log.Fatalf("collector failed %v", err)
	}
}
