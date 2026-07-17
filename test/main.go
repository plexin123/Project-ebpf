package main

import (
	"fmt"
	"math/rand"
	"time"
)

func handleRequest() {
	ms := rand.Intn(100) + 10
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func main() {
	fmt.Print("This is a test")
	for {
		handleRequest()
		fmt.Println("its being handle")
	}
}
