package collector

import (
	"ebpf-project/backend/broadcast"
	"ebpf-project/backend/callstack"
	"ebpf-project/backend/database"
	pb "ebpf-project/backend/pb"
	"ebpf-project/backend/stats"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
)

type FunctionStats struct {
	FunctionName string
	Window       []uint64
	baselineflag bool
	baselinep95  uint64
}

// what is send to the Frontend structure
type FunctionEvent struct {
	FuncName   string  `json:"funcName"`
	Duration   uint64  `json:"duration"`
	SelfTime   uint64  `json:"selfTime"`
	Status     Status  `json:"status"`
	Baseline   uint64  `json:"baseline"`
	Current    uint64  `json:"current"`
	DriftPct   float64 `json:"driftPct"`
	TimeStamp  int64   `json:"timestamp"`
	CommitHash string  `json:"commithash"`
}

type Status string

const (
	StatusOk          Status = "ok"
	StatusBaselineSet Status = "baseline_set"
	StatusRegression  Status = "regression"
)

// gRPC server: receives the events already paired by the agent (enter/exit, self time)
// and only does the calculations, the storage and the broadcast
type Server struct {
	pb.UnimplementedRoutingFunctionsServer
	db         *database.Database
	connection *broadcast.ConnectionStructure
	commit     string
	// every agent stream runs in its own goroutine, functions is shared between them
	mu        sync.Mutex
	functions map[string]*FunctionStats
}

func NewServer(db *database.Database, connection *broadcast.ConnectionStructure, commit string) *Server {
	return &Server{
		db:         db,
		connection: connection,
		commit:     commit,
		functions:  make(map[string]*FunctionStats),
	}
}

func (s *Server) StreamingData(stream pb.RoutingFunctions_StreamingDataServer) error {
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// the agent closed the stream
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
		switch kind := event.Kind.(type) {
		case *pb.AgentEvent_Edge:
			s.handleEdge(kind.Edge)
		case *pb.AgentEvent_Result:
			s.handleResult(kind.Result)
		}
	}
}

func (s *Server) handleEdge(edge *pb.CallEdge) {
	// root of the stack: there is no caller, so no connection to draw
	if edge.Caller == "" {
		return
	}
	callEvent := callstack.CallEvent{Caller: edge.Caller, Callee: edge.Callee}
	s.connection.Broadcast(broadcast.WsMessage{Type: "connection", Payload: callEvent, TraceId: edge.TraceId})
}

func (s *Server) handleResult(result *pb.CallResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	funcName := result.Function
	if s.functions[funcName] == nil {
		s.functions[funcName] = &FunctionStats{FunctionName: funcName}
	}
	functionStats := s.functions[funcName]
	new_window := append(functionStats.Window, result.TotalTime)
	validated_window := stats.ValidateWindow(new_window)
	if len(validated_window) < 1 {
		return
	}

	currentbaselinep95 := stats.P95(validated_window)
	baselinep95 := functionStats.baselinep95

	event_data := FunctionEvent{
		FuncName: funcName,
		Duration: result.TotalTime,
		Current:  currentbaselinep95,
		SelfTime: result.SelfTime,
	}

	if functionStats.baselineflag == false {
		functionStats.baselinep95 = currentbaselinep95
		functionStats.baselineflag = true
		event_data.Status = StatusBaselineSet
		event_data.Baseline = currentbaselinep95
	} else {
		// float64 before subtracting: with uint64, current < baseline wraps around to a huge drift
		drift := 0.0
		if baselinep95 != 0 {
			drift = (float64(currentbaselinep95) - float64(baselinep95)) / float64(baselinep95)
		}
		event_data.Baseline = baselinep95
		event_data.DriftPct = drift * 100
		if drift > 0.2 {
			event_data.Status = StatusRegression
		} else {
			event_data.Status = StatusOk
		}
	}
	functionStats.Window = validated_window

	// wall clock: the agent time_stamp is bpf_ktime_get_ns (time since boot)
	timeStamp := time.Now()
	timeStampUnix := timeStamp.UnixMilli()
	if _, err := s.db.InsertEvent(database.FunctionEvent{
		FuncName:   event_data.FuncName,
		Duration:   event_data.Duration,
		Status:     database.Status(event_data.Status),
		Baseline:   event_data.Baseline,
		Current:    event_data.Current,
		DriftPct:   event_data.DriftPct,
		TimeStamp:  timeStampUnix,
		CommitHash: s.commit,
	}); err != nil {
		log.Printf("failed to insert event for %s: %v", funcName, err)
	}

	event_data.TimeStamp = timeStampUnix
	event_data.CommitHash = s.commit
	s.connection.Broadcast(broadcast.WsMessage{Type: "event", Payload: event_data, TraceId: result.TraceId})
}
