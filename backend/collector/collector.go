package collector

import (
	"bytes"
	"debug/elf"
	"ebpf-project/backend/broadcast"
	"ebpf-project/backend/callstack"
	"ebpf-project/backend/stats"

	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type Latency_event struct {
	DurationsNS     uint64
	Name_of_process [16]byte
}

type EnterEvent struct {
	PidTgid     uint64
	FuncAddress uint64
}

type EnvelopedEvent struct {
	EventType     uint8
	PidTgid       uint64
	FuncAddress   uint64
	Latency_event Latency_event
}

type FunctionStats struct {
	FunctionName string
	Window       []uint64
	baselineflag bool
	baselinep95  uint64
}

type FunctionEvent struct {
	FuncName string  `json:"funcName"`
	Duration uint64  `json:"duration"`
	Status   Status  `json:"status"`
	Baseline uint64  `json:"baseline"`
	Current  uint64  `json:"current"`
	DriftPct float64 `json:"driftPct"`
}

type Status string

const (
	StatusOk          Status = "ok"
	StatusBaselineSet Status = "baseline_set"
	StatusRegression  Status = "regression"
)

// TO DO: Separation between the mapping functionality and calculation
func Collector(callStackTracer *callstack.CallStackTracer) error {
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
			currentTraceId := callStackTracer.Map_trace_id[event.PidTgid]
			callStackTracer.HandleExitEvent(event.PidTgid)
			if !ok {
				continue
			}
			if map_of_functions[funcName] == nil {
				map_of_functions[funcName] = &FunctionStats{FunctionName: funcName}
			}
			current_window := map_of_functions[funcName].Window
			new_window := append(current_window, event.Latency_event.DurationsNS)
			validated_window := stats.ValidateWindow(new_window)
			fmt.Printf("SENDING_DATA_PAUL")
			if len(validated_window) >= 1 {

				currentbaselinep95 := stats.P95(validated_window)

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
				callStackTracer.BroadCaster.Broadcast(broadcast.WsMessage{Type: "event", Payload: event_data, TraceId: currentTraceId.String()})
			}
		} else if event.EventType == 0 {
			funcName, ok := register_map[event.FuncAddress]
			if !ok {
				continue
			}

			callStackTracer.HandleEnterEvent(event.PidTgid, funcName)
		}
	}

	return nil
}
