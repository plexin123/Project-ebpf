package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"golang.org/x/arch/arm64/arm64asm"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"slices"
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
	DurationsNS     uint64
	MemoryPointer   uint64
	PID             uint32
	_               [4]byte
	Name_of_process [16]byte
}





// for each functionality there is going to be an average
// func_name - avg = (time end - time start) // amount of calls, amount of calls
// left pointer starting with the first average time
// "average_time" : (average_time + DurationNS) / amount_of_calls
// "amount_calls" : amount_of_calls
// // for each func_name ->{
// 		"array_of_time": [times],
// 		"left_pointer" : 0
// 		"amount_calls" : 0
// }

type FunctionStats struct {
	FunctionName string
	Window       []uint64
	baselineflag bool
	baselinep95  uint64
}

// map -> each fuction will have the struct FunctionStats

func p95(window []uint64) uint64 {
	// sort the value
	sorted := make([]uint64, len(window))
	copy(sorted, window)
	// do operation -> 95 // size(window)
	slices.Sort(sorted)
	index_of_95 := int(float64(len(window)) * 0.95)
	// sorted_value[ans]
	value := sorted[index_of_95]
	// return a
	return value
}

func validateWindow(window []uint64) []uint64 {
	if len(window) > 20 {
		new_window := window[1:]
		return new_window
	}
	return window
}

func findRetOffsets(data []byte) []uint64 {
	var offsets []uint64
	offset := 0
	for offset + 4 <=  len(data) {
		inst, err := arm64asm.Decode(data[offset: offset+4])
		if err != nil {
			offset += 4
			continue
		}
		if inst.Op == arm64asm.RET {
			offsets = append(offsets, uint64(offset))
		}
		offset += 4
	}
	return offsets
}	


func main() {
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
		log.Fatalf("faled to read symbols: %v", err)
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

		ret,err := ex.Uretprobe(sym.Name, coll.Programs["trace_exit"],nil)

		if err != nil{
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
		fmt.Printf("[RAW EVENT] PID: %d | MemPtr: 0x%x | Duration: %dns\n", 
		        event.PID, event.MemoryPointer, event.DurationsNS)
		funcName, ok := register_map[event.MemoryPointer]
		if !ok {
			fmt.Printf("MemoryPointer 0x%x not found in register_map!\n", event.MemoryPointer)
			continue
		}
		if map_of_functions[funcName] == nil {
			map_of_functions[funcName] = &FunctionStats{FunctionName: funcName}
		}
		current_window := map_of_functions[funcName].Window
		new_window := append(current_window, event.DurationsNS)
		map_of_functions[funcName].Window = validateWindow(new_window)

		fmt.Printf("EVENT RECV -> func: %-20s duration: %dms (window size: %d)\n",
	funcName, event.DurationsNS/1_000_000, len(map_of_functions[funcName].Window))
		if len(map_of_functions[funcName].Window) >= 7 {

			currentbaselinep95 := p95(map_of_functions[funcName].Window)

			baselinep95 := map_of_functions[funcName].baselinep95

			if map_of_functions[funcName].baselineflag == false {
				map_of_functions[funcName].baselinep95 = currentbaselinep95
				map_of_functions[funcName].baselineflag = true
			} else if map_of_functions[funcName].baselineflag {
				drift := (float64(currentbaselinep95)-float64(baselinep95)) / float64(baselinep95)
				if drift > 0.00005 {
					fmt.Printf("⚠ regression: %s baseline=%dms current=%dms +%.0f%%\n",
						funcName, baselinep95/1_000_000, currentbaselinep95/1_000_000, drift*100)
				}
			}

			fmt.Printf("func: a%-40s  duration: %dms\n", funcName, event.DurationsNS/1_000_000)
		}
	}
	// func 10
	// func 20
	// use function name
	// [20, 30, 10, 40, 50, 29]
	// [10,20,29,30,40,500000,] ->40
	// 40,100,120,140,150,100, -> 150
	//
}
