package main

type Latency_event struct {
	DurationsNS     uint64
	MemoryPointer   uint64
	PidTgid         uint64
	Name_of_process [16]byte
}

type CallEvent struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}

type EnterEvent struct {
	PidTgid     uint64
	FuncAddress uint64
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
