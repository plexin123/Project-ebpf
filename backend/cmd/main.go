package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

// Latency time for each functionality
// Keep it 1 function
// compile one test golang file
// run it against the lumentrace -> see function how long does it take

// type ProcessEvent struct {
// 	PID  uint32
// 	PPID uint32
// 	Comm [16]byte
// }

// func main() {

// 	spec, err := ebpf.LoadCollectionSpec("monitor.bpf.o")

// 	if err != nil {
// 		log.Fatalf("failed to load eBPF spec: %v", err)
// 	}

// 	coll, err := ebpf.NewCollection(spec)

// 	if err != nil {
// 		log.Fatalf("failed to create collection %v", err)
// 	}

// 	defer coll.Close()

// 	tp, err := link.Tracepoint("syscalls", "sys_enter_execve", coll.Programs["new_program"], nil)

// 	if err != nil {
// 		log.Fatalf("failed to attach tracepoint %v", err)
// 	}

// 	defer tp.Close()

// 	reader, err := ringbuf.NewReader(coll.Maps["events"])

// 	if err != nil {
// 		log.Fatalf("failed to create a new reader %v", err)
// 	}

// 	defer reader.Close()

// 	sig := make(chan os.Signal, 1)

// 	signal.Notify(sig, os.Interrupt)

// 	go func() {
// 		<-sig
// 		reader.Close()
// 	}()

// 	for {
// 		record, err := reader.Read()

// 		if err != nil {
// 			break
// 		}

// 		var event ProcessEvent

// 		if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
// 			continue
// 		}
// 		name := string(bytes.TrimRight(event.Comm[:], "\x00"))

// 		fmt.Printf("pid: %-6d  ppid: %-6d  comm: %s\n", event.PID, event.PPID, name)

// 	}

// }

type Latency_event struct {
	DurationNS uint64
	PID        uint32
	Comm       [16]byte
}

func main() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("failed to remove memlock: %v", err)
	}
	if len(os.Args) < 3 {
		log.Fatalf("usage: profiler <binary> <function>")
	}
	binaryPath := os.Args[1]
	funcName := os.Args[2]

	spec, err := ebpf.LoadCollectionSpec("profiler.bpf.o")
	if err != nil {
		log.Fatalf("failed to load spec: %v", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Fatalf("failed to create collection: %v", err)
	}
	defer coll.Close()

	// open binary

	ex, err := link.OpenExecutable(binaryPath)

	if err != nil {
		log.Fatalf("failed to open binary: %v", err)
	}

	// attach entry probe

	up, err := ex.Uprobe(funcName, coll.Programs["trace_enter"], nil)

	if err != nil {
		log.Fatalf("failed to attached uprobe: %v", err)
	}

	defer up.Close()

	// exit probe
	ret, err := ex.Uretprobe(funcName, coll.Programs["trace_exit"], nil)

	if err != nil {
		log.Fatalf("Failed to attach uretprobe: %v", err)
	}
	defer ret.Close()

	reader, err := ringbuf.NewReader(coll.Maps["latency_events"])

	if err != nil {
		log.Fatalf("failed to open ring buffer: %v", err)
	}

	defer reader.Close()

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt)

	go func() {
		<-sig
		reader.Close()
	}()

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		var event Latency_event
		if err := binary.Read(
			bytes.NewReader(record.RawSample),
			binary.LittleEndian,
			&event,
		); err != nil {
			log.Printf("Failed to parse event: %v", err)
			continue
		}
		name := string(bytes.TrimRight(event.Comm[:], "\x00"))
		fmt.Printf("pid: %-6d duration: %dms\n name: %s\n", event.PID, event.DurationNS/1_000_000, name)
	}

	// use function name

}
