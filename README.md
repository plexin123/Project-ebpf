# Lumentrace

Lumentrace is an eBPF-powered function-level profiler for Go binaries. It
attaches to a running process with no code changes required, watches every
call to its exported functions, and streams a live call graph to a browser
UI — flagging functions the moment their latency drifts away from their own
baseline.

## How it works

1. **Trace** — `agent/bpf/profiler.bpf.c` attaches a uprobe/uretprobe pair to
   each `main.*` function in the target binary. On entry and exit it pushes
   timing events (pid/tgid, function address, duration) through a BPF ring
   buffer.
2. **Collect** — `backend/cmd/main.go` loads the compiled eBPF program,
   resolves the target binary's symbols, and reads the ring buffer. For each
   function it keeps a rolling window of recent call durations and computes
   a p95 baseline.
3. **Detect** — once a baseline is established, every new call is compared
   against it. A p95 drift of more than 20% is flagged as a `regression`;
   otherwise the function reports `ok`.
4. **Visualize** — the backend broadcasts call-graph edges (caller → callee)
   and per-function stats over a WebSocket (`:8080/ws`). The
   `frontend/ebpf-project` React app consumes the stream and renders it as a
   live, per-trace call graph.

A second, independent probe — `agent/bpf/monitor.bpf.c` — hooks
`sys_enter_execve` to capture process creation (pid, ppid, process name).
It's a general-purpose exec monitor and isn't wired into the profiler
pipeline above.

## Repository structure

- `agent/` — the eBPF (kernel-side) programs
  - `bpf/profiler.bpf.c` — uprobe/uretprobe latency tracer used by the profiler
  - `bpf/monitor.bpf.c` — `execve` tracepoint, reports new processes
  - `bpf/ebpf_structures.h` — shared event structs and BPF map definitions
  - `bpf/Makefile` — compiles the `.bpf.c` files to `.bpf.o` objects with clang
- `backend/cmd/` — the Go collector/server
  - `main.go` — loads `profiler.bpf.o`, attaches uprobes to the target
    binary, computes p95 baselines, detects regressions
  - `broadcast.go` — WebSocket server that fans events out to connected clients
  - `types.go` — shared event/message types
  - `test.go` — a synthetic target program that calls
    `main → handleRequestA → handleRequestB → handleRequestC` in a loop and
    injects an artificial 100x latency spike partway through, for exercising
    the profiler end-to-end
- `frontend/ebpf-project/` — React + Vite app that connects to the
  WebSocket and renders the live call graph
- `test/` — a standalone build of the synthetic target binary used for
  manual testing

## Running it

Requires Linux with BTF support (kernel headers/vmlinux), clang, and Go.

1. **Build the eBPF objects**

   ```
   cd agent/bpf
   make
   ```

2. **Build (or reuse) a target binary to profile.** For a quick smoke test,
   build the synthetic target in `test/`:

   ```
   cd test
   go build -gcflags="all=-l" -o target_service .
   ./target_service
   ```

   (`-gcflags="all=-l"` disables inlining so the uprobes actually fire.)

3. **Run the collector** (needs root to load eBPF programs), pointing it at
   the binary you want to profile:

   ```
   cd backend/cmd
   sudo go run . ../../test/target_service
   ```

   This starts the WebSocket server on `:8080/ws` and begins tracing.

4. **Run the frontend**

   ```
   cd frontend/ebpf-project
   npm install
   npm run dev
   ```

   Open the printed local URL and connect it to the collector's WebSocket
   to watch the call graph update live, including the regression flag when
   `handleRequestC`'s injected latency spike kicks in.

## Status

Actively evolving / work in progress. Expect rough edges (e.g. the
WebSocket URL the frontend connects to is currently hardcoded and will need
to point at wherever the collector is running).
