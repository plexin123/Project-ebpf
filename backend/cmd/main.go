package main

//  now we need to capture the packets, from the websockets and then convert that into object and display that
// 1 websocket capture

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"debug/elf"
	"log"
	"os"
	"strings"
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
	DurationsNS uint64
	MemoryPointer uint64
	PID        uint32
	_             [4]byte
	Name_of_process       [16]byte
}

func main() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("failed to remove memlock: %v", err)
	}
	if len(os.Args) < 2 {
		log.Fatalf("usage: profiler <binary> <function>")
	}
	binaryPath := os.Args[1]
	// there is not going to be a function name
	// Instead we read from the binary, read the table of functions
	// 	getFunctions() -> []string : all function names
	// 	filter those functions according to the main.*
	// 	attach uprobe
	//	create a map key = memory pointer -> latency_event struct
	//	
		
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

	if err != nil{
		log.Fatalf("failed to open ELF: %v", err)	
	}
	syms, err := f.Symbols()
	if err != nil{
		log.Fatalf("faled to read symbols: %v", err)
	}
	f.Close()
	
	register_map :=  make(map[uint64]string)
	var links []link.Link
	for _ ,sym := range syms{
		
		// filter the according to the name main.*
		if elf.ST_TYPE(sym.Info) != elf.STT_FUNC{
			continue
		}

		if !strings.HasPrefix(sym.Name,"main."){
			
			continue
			
		}


		up, err := ex.Uprobe(sym.Name, coll.Programs["trace_enter"],nil)
		
		if err != nil{
			continue
		}
		
		ret, err := ex.Uretprobe(sym.Name,coll.Programs["trace_exit"], nil)

		if err != nil{
			up.Close()
			continue	
		}
		
		links = append(links, up,ret)
		
		register_map[sym.Value] = sym.Name
	}

	defer func(){
		for _,l := range(links){
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

		funcName, ok := register_map[event.MemoryPointer]
		if !ok{
			continue
		}
		fmt.Printf("func: %-40s  duration: %dms\n", funcName, event.DurationsNS/1_000_000)
	}

	// use function name

}
