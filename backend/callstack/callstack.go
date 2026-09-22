package callstack

import (
	"fmt"
	"log"
	"sync"

	"ebpf-project/backend/types"

	"github.com/google/uuid"
)

type Broadcaster interface {
	Broadcast(data any)
}

type CallStackTracer struct {
	Stack_mu          sync.Mutex
	Map_trace_id      map[uint64]uuid.UUID
	Map_pid_gid_stack map[uint64][]string
	BroadCaster       Broadcaster
}

func New() *CallStackTracer {
	newCallStackTracer := &CallStackTracer{
		Map_trace_id:      make(map[uint64]uuid.UUID),
		Map_pid_gid_stack: make(map[uint64][]string),
	}

	return newCallStackTracer
}
func (cst *CallStackTracer) HandleEnterEvent(pid_gid uint64, funcName string) {
	cst.Stack_mu.Lock()
	defer cst.Stack_mu.Unlock()
	get_current_stack := cst.Map_pid_gid_stack[pid_gid]
	get_current_stack = append(get_current_stack, funcName)
	cst.Map_pid_gid_stack[pid_gid] = get_current_stack
	if len(get_current_stack) > 1 {
		current_father := get_current_stack[len(get_current_stack)-2]
		current_trace_id := cst.Map_trace_id[pid_gid]
		cst.BroadCaster.Broadcast(types.WsMessage{Type: "connection", Payload: types.CallEvent{Caller: current_father, Callee: funcName}, TraceId: current_trace_id.String()})
	}
	if len(get_current_stack) == 1 {
		traceId, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Failed to create traceId %v", err)
		}
		cst.Map_trace_id[pid_gid] = traceId
	}
	fmt.Printf("This is the current stack for this pid %v: %v", pid_gid, get_current_stack)
}

func (cst *CallStackTracer) HandleExitEvent(pid_gid uint64) {
	cst.Stack_mu.Lock()
	defer cst.Stack_mu.Unlock()
	get_current_stack := cst.Map_pid_gid_stack[pid_gid]
	if len(get_current_stack) > 0 {
		cst.Map_pid_gid_stack[pid_gid] = get_current_stack[:len(get_current_stack)-1]
	}
	if len(cst.Map_pid_gid_stack[pid_gid]) == 0 {
		delete(cst.Map_trace_id, pid_gid)
	}

}
