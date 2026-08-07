package main

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

var regressionFlag atomic.Bool

//go:noinline
func handleRequestA() {
	ms := rand.Intn(10) + 5
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

//go:noinline
func handleRequestC() {
	var ms int
	if regressionFlag.Load() {
		// Slow path: 100ms - 200ms latency injected
		ms = rand.Intn(100) + 100
	} else {
		// Normal path: 10ms - 20ms latency
		ms = rand.Intn(10) + 10
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func main() {
	fmt.Println("🚀 Target binary running PID:", time.Now().Unix())

	// Trigger regression after 5 seconds
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("\n⚠️  INJECTING REGRESSION INTO handleRequestC ⚠️\n")
		regressionFlag.Store(true)
	}()

	for {
		handleRequestA()
		handleRequestC()
	}
}
