package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"ebpf-project/backend/broadcast"
	"ebpf-project/backend/collector"
	"ebpf-project/backend/database"
	pb "ebpf-project/backend/pb"

	"debug/buildinfo"

	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <target-binary>", os.Args[0])
	}
	target_path := os.Args[1]
	info, err := buildinfo.ReadFile(target_path)
	if err != nil {
		log.Fatalf("Could not read the targeted file %s: %v", target_path, err)
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
		log.Fatalf("Could not create an instance of the database %v", err)
	}
	handler := database.New(databaseInstance)

	http.HandleFunc("/ws", connectionStructure.HandleWS)
	http.HandleFunc("/history/{name}", handler.GetHistoryByName)

	fmt.Printf("Websocket server starting.. on 8080")
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("websocket server failed %v", err)
			sig <- os.Interrupt
		}
	}()

	// gRPC server that receives the events from the agent
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("could not listen on :50051 %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterRoutingFunctionsServer(grpcServer, collector.NewServer(databaseInstance, connectionStructure, commit_hash))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("collector has failed %v", err)
			sig <- os.Interrupt
		}
	}()
	// wait for all the process to finish and then exit, most important ->  ws server.
	answer := <-sig
	handler.Close()
	log.Fatalf("There has been an error %v", answer)
	// wait for thread to finish and join or is it bad practice
}
