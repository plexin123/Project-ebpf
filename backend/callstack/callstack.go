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
	stack_mu          sync.Mutex
	map_trace_id      map[uint64]uuid.UUID
	map_pid_gid_stack map[uint64][]string
	broadCaster       Broadcaster
}

func New() *CallStackTracer {
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
		cst.broadCaster.Broadcast(types.WsMessage{Type: "connection", Payload: types.CallEvent{Caller: current_father, Callee: funcName}, TraceId: current_trace_id.String()})
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

func (cst *CallStackTracer) HandleExitEvent(pid_gid uint64) {
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
