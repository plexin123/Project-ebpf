package main

// Standalone target binary for testing the eBPF profiler end-to-end.
//
// It calls a fixed chain of functions sequentially, over and over:
//   main -> handleRequestA -> handleRequestB -> handleRequestC
//
// The first ~25 calls run at a stable "normal" latency (5ms), which lets
// the profiler establish a p95 baseline. After that, handleRequestC's
// latency jumps to 500ms -- 100x baseline -- on purpose, so the regression
// is unmissable on the graph instead of a subtle percentage shift.
//
// BUILD (disable inlining, or uprobes on small funcs may never fire):
//   go build -gcflags="all=-l" -o target_service target_service.go
//
// RUN:
//   ./target_service
//
// Then, in another terminal, point your profiler at it:
//   sudo ./profiler ./target_service
//
// Watch the graph: you should see main.main -> main.handleRequestA ->
// main.handleRequestB -> main.handleRequestC edges pulse on every call,
// and main.handleRequestC should flip to Regression once the spike kicks in.

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	totalIterations = 60
	spikeStartsAt   = 25 // after this many calls, handleRequestC latency spikes

	baselineLatency = 5 * time.Millisecond
	spikeLatency    = 500 * time.Millisecond // 100x baseline -- unmissable on the graph
)

//go:noinline
func handleRequestA(i int) {
	// light, stable work
	time.Sleep(2 * time.Millisecond)
	handleRequestB(i)
}

//go:noinline
func handleRequestB(i int) {
	// light, stable work with a touch of jitter so p95 isn't perfectly flat
	time.Sleep(time.Duration(3+rand.Intn(2)) * time.Millisecond)
	handleRequestC(i)
}

//go:noinline
func handleRequestC(i int) {
	if i < spikeStartsAt {
		// normal latency -- this is what establishes the baseline
		time.Sleep(baselineLatency)
	} else {
		// deliberate regression: 100x baseline, impossible to miss on the graph
		time.Sleep(spikeLatency)
	}
	validate(i)
}

//go:noinline
func validate(i int) {
	// cheap leaf call so the graph shows a 4-deep chain, not just 3
	_ = i * i
}

func test() {
	fmt.Printf("target_service starting: %d sequential calls, latency spike after call #%d\n",
		totalIterations, spikeStartsAt)

	for i := 0; i < totalIterations; i++ {
		handleRequestA(i)
		if i == spikeStartsAt {
			fmt.Println("!! REGRESSION INJECTED -- handleRequestC just went 100x slower !!")
		}
		// small pause between calls so events are easy to watch land one at a time
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("target_service done")
}
