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
)

// we need to create a udp server to listen for incoming packets parse it and then print that
// to achive first goal

type ProcessEvent struct {
	PID  uint32
	PPID uint32
	Comm [16]byte
}

func main() {

	spec, err := ebpf.LoadCollectionSpec("monitor.bpf.o")

	if err != nil {
		log.Fatalf("failed to load eBPF spec: %v", err)
	}

	coll, err := ebpf.NewCollection(spec)

	if err != nil {
		log.Fatalf("failed to create collection %v", err)
	}

	defer coll.Close()

	tp, err := link.Tracepoint("syscalls", "sys_enter_execve", coll.Programs["new_program"], nil)

	if err != nil {
		log.Fatalf("failed to attach tracepoint %v", err)
	}

	defer tp.Close()

	reader, err := ringbuf.NewReader(coll.Maps["events"])

	if err != nil {
		log.Fatalf("failed to create a new reader %v", err)
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

		var event ProcessEvent

		if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
			continue
		}
		name := string(bytes.TrimRight(event.Comm[:], "\x00"))

		fmt.Printf("pid: %-6d  ppid: %-6d  comm: %s\n", event.PID, event.PPID, name)

	}

}
