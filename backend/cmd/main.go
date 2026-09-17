package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/google/uuid"
)

func p95(window []uint64) uint64 {
	sorted := make([]uint64, len(window))
	copy(sorted, window)
	slices.Sort(sorted)
	index_of_95 := int(float64(len(window)) * 0.95)
	value := sorted[index_of_95]
	return value
}

func validateWindow(window []uint64) []uint64 {
	if len(window) > 20 {
		new_window := window[1:]
		return new_window
	}
	return window
}

type Broadcaster interface {
	Broadcast(data any)
}

type CallStackTracer struct {
	stack_mu          sync.Mutex
	map_trace_id      map[uint64]uuid.UUID
	map_pid_gid_stack map[uint64][]string
	broadCaster       Broadcaster
}

func newCallStackTracer() *CallStackTracer {
	newCallStackTracer := &CallStackTracer{
		map_trace_id:      make(map[uint64]uuid.UUID),
		map_pid_gid_stack: make(map[uint64][]string),
	}

	return newCallStackTracer
}
func (cst *CallStackTracer) HandleEnterEvent(pid_gid uint64, funcName string) {
	cst.stack_mu.Lock()
	defer cst.stack_mu.Unlock()
	get_current_stack := cst.map_pid_gid_stack[pid_gid]
	get_current_stack = append(get_current_stack, funcName)
	cst.map_pid_gid_stack[pid_gid] = get_current_stack
	if len(get_current_stack) > 1 {
		current_father := get_current_stack[len(get_current_stack)-2]
		current_trace_id := cst.map_trace_id[pid_gid]
		cst.broadCaster.Broadcast(WsMessage{Type: "connection", Payload: CallEvent{Caller: current_father, Callee: funcName}, TraceId: current_trace_id.String()})
	}
	if len(get_current_stack) == 1 {
		traceId, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Failed to create traceId %v", err)
		}
		cst.map_trace_id[pid_gid] = traceId
	}
	fmt.Printf("This is the current stack for this pid %v: %v", pid_gid, get_current_stack)

}
func (cst *CallStackTracer) handleExitEvent(pid_gid uint64) {
	cst.stack_mu.Lock()
	defer cst.stack_mu.Unlock()
	get_current_stack := cst.map_pid_gid_stack[pid_gid]
	if len(get_current_stack) > 0 {
		cst.map_pid_gid_stack[pid_gid] = get_current_stack[:len(get_current_stack)-1]
	}
	if len(cst.map_pid_gid_stack[pid_gid]) == 0 {
		delete(cst.map_trace_id, pid_gid)
	}

}
func (cst *CallStackTracer) collector(cn *ConnectionStructure) error {
	if err := rlimit.RemoveMemlock(); err != nil {

		log.Fatalf("failed to remove memlock: %v", err)
	}
	if len(os.Args) < 2 {
		log.Fatalf("usage: profiler <binary> <function>")
	}

	binaryPath := os.Args[1]

	spec, err := ebpf.LoadCollectionSpec("../../agent/bpf/profiler.bpf.o")
	if err != nil {
		log.Fatalf("failed to load spec: %v", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Fatalf("failed to create collection: %v", err)
	}
	defer coll.Close()
	// add these right after ebpf.NewCollection(spec)
	fmt.Printf("programs found: %v\n", coll.Programs)
	fmt.Printf("maps found: %v\n", coll.Maps)

	// open binary

	ex, err := link.OpenExecutable(binaryPath)

	if err != nil {
		log.Fatalf("failed to open binary: %v", err)
	}
	f, err := elf.Open(binaryPath)

	if err != nil {
		log.Fatalf("failed to open ELF: %v", err)
	}
	syms, err := f.Symbols()
	if err != nil {
		log.Fatalf("failed to read symbols: %v", err)
	}
	f.Close()

	register_map := make(map[uint64]string)
	map_of_functions := make(map[string]*FunctionStats)
	var links []link.Link
	for _, sym := range syms {

		// filter the according to the name main.*
		if elf.ST_TYPE(sym.Info) != elf.STT_FUNC {
			continue
		}

		if !strings.HasPrefix(sym.Name, "main.") {

			continue

		}

		up, err := ex.Uprobe(sym.Name, coll.Programs["trace_enter"], nil)

		if err != nil {
			continue
		}

		ret, err := ex.Uretprobe(sym.Name, coll.Programs["trace_exit"], nil)

		if err != nil {
			up.Close()
			continue
		}

		links = append(links, up, ret)

		register_map[sym.Value] = sym.Name
		map_of_functions[sym.Name] = &FunctionStats{
			FunctionName: sym.Name,
			Window:       []uint64{},
			baselinep95:  0,
			baselineflag: false,
		}
	}
	defer func() {
		for _, l := range links {
			l.Close()
		}
	}()
	reader, err := ringbuf.NewReader(coll.Maps["events"])

	if err != nil {
		log.Fatalf("failed to open ring buffer: %v", err)
	}

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
		var event EnvelopedEvent
		if err := binary.Read(
			bytes.NewReader(record.RawSample),
			binary.LittleEndian,
			&event,
		); err != nil {
			log.Printf("Failed to parse event: %v", err)
			continue
		}
		// 0 -> entrance event

		if event.EventType == 1 {

			funcName, ok := register_map[event.FuncAddress]
			currentTraceId := cst.map_trace_id[event.PidTgid]
			cst.handleExitEvent(event.PidTgid)
			if !ok {
				continue
			}
			if map_of_functions[funcName] == nil {
				map_of_functions[funcName] = &FunctionStats{FunctionName: funcName}
			}
			current_window := map_of_functions[funcName].Window
			new_window := append(current_window, event.Latency_event.DurationsNS)
			validated_window := validateWindow(new_window)
			fmt.Printf("SENDING_DATA_PAUL")
			if len(validated_window) >= 1 {

				currentbaselinep95 := p95(validated_window)

				baselinep95 := map_of_functions[funcName].baselinep95

				event_data := FunctionEvent{
					FuncName: funcName,
					Duration: event.Latency_event.DurationsNS,
					Current:  currentbaselinep95,
				}

				if map_of_functions[funcName].baselineflag == false {
					map_of_functions[funcName].baselinep95 = currentbaselinep95
					map_of_functions[funcName].baselineflag = true
					event_data.Status = StatusBaselineSet
					event_data.Baseline = currentbaselinep95
				} else {
					drift := float64(currentbaselinep95-baselinep95) / float64(baselinep95)
					event_data.Baseline = baselinep95
					event_data.DriftPct = drift * 100
					if drift > 0.2 {
						event_data.Status = StatusRegression
					} else {
						event_data.Status = StatusOk
					}

				}
				map_of_functions[funcName].Window = validated_window
				cst.broadCaster.Broadcast(WsMessage{Type: "event", Payload: event_data, TraceId: currentTraceId.String()})
			}
		} else if event.EventType == 0 {
			funcName, ok := register_map[event.FuncAddress]
			if !ok {
				continue
			}

			cst.HandleEnterEvent(event.PidTgid, funcName)
		}
	}

	return nil
}
func main() {
	callStackTracer := newCallStackTracer()
	connectionSructure := NewConnectionStructure()
	http.HandleFunc("/ws", connectionSructure.HandleWS)

	fmt.Printf("Websocket server starting.. on 8080")

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("websocket server failed %v", err)
		}
	}()

	if err := callStackTracer.collector(connectionSructure); err != nil {
		log.Fatalf("collector failed %v", err)
	}
}
