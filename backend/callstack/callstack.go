package callstack

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

// type Broadcaster interface {
// 	Broadcast(data any)
// }

type CallStackTracer struct {
	Stack_mu          sync.Mutex
	Map_trace_id      map[uint64]uuid.UUID
	Map_pid_gid_stack map[uint64][]Record
	// BroadCaster       Broadcaster
}

type CallEvent struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}

type Record struct {
	Name            string
	AccumulatedTime uint64
}

func New() *CallStackTracer {
	newCallStackTracer := &CallStackTracer{
		Map_trace_id:      make(map[uint64]uuid.UUID),
		Map_pid_gid_stack: make(map[uint64][]Record),
		// BroadCaster:       br,
	}

	return newCallStackTracer
}

func (cst *CallStackTracer) SelfTimeCalculation(pid_gid, total_time uint64) uint64 {
	// TO DO: Implementation of self function time without children/dependent
	// each parent_func,sum_children_time
	current_stack := cst.Map_pid_gid_stack[pid_gid]
	if len(current_stack) >= 2 {
		current_stack[len(current_stack)-2].AccumulatedTime += total_time
	}
	accumulated_children_time := current_stack[len(current_stack)-1].AccumulatedTime
	selftime := total_time - accumulated_children_time
	return selftime

}

// instead of just funcName send the whole structure
// or just send the time
// P
func (cst *CallStackTracer) HandleEnterEvent(pid_gid uint64, funcName string) (CallEvent, uuid.UUID) {
	cst.Stack_mu.Lock()
	defer cst.Stack_mu.Unlock()
	get_current_stack := cst.Map_pid_gid_stack[pid_gid]
	get_current_stack = append(get_current_stack, Record{funcName, 0})
	cst.Map_pid_gid_stack[pid_gid] = get_current_stack
	if len(get_current_stack) > 1 {
		current_father := get_current_stack[len(get_current_stack)-2].Name
		current_trace_id := cst.Map_trace_id[pid_gid]
		//Separation of responsabilities to avoid cycle dependency
		return CallEvent{Caller: current_father, Callee: funcName}, current_trace_id
		// cst.BroadCaster.Broadcast(broadcast.WsMessage{Type: "connection", Payload: CallEvent{Caller: current_father, Callee: funcName}, TraceId: current_trace_id.String()})
	}
	if len(get_current_stack) == 1 {
		// when it just enters a new trace_id, then there is a new empty stack to fill
		traceId, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Failed to create traceId %v", err)
		}
		cst.Map_trace_id[pid_gid] = traceId
	}
	fmt.Printf("This is the current stack for this pid %v: %v", pid_gid, get_current_stack)
	return CallEvent{}, uuid.Nil
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
