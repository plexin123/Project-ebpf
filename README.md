# Lumentrace

Lumentrace is an eBPF-powered, function-level profiler and regression
detector. It attaches to an unmodified binary (currently Rust), measures
every call to the binary's own functions, streams a live call graph to a
browser UI, and flags functions whose latency drifts away from their
baseline — tagged with the commit that produced the binary.

```
target binary ──uprobes──▶ agent (Rust/aya) ──gRPC :50051──▶ backend (Go) ──WebSocket :8080──▶ UI (React/D3)
```

## How it works

1. **Scan** — the agent (`agent/adapter`) reads the target's ELF symbol
   table, demangles the names, and keeps only the functions that belong to
   the target crate (e.g. `orders::`).
2. **Attach** — it loads `agent/bpf/profiler.bpf.o` and attaches a uprobe
   (function entry) and a uretprobe (function exit) to each of those
   functions. Every probe carries a *cookie*: the function's index in the
   agent's table.
3. **Capture (kernel)** — on every entry/exit, the BPF program writes a
   25-byte event `{type, pid/tid, cookie, timestamp_ns}` to a ring buffer.
   No logic runs in the kernel; it only records.
4. **Reconstruct (agent)** — `agent/profiler-core/src/callstack.rs` keeps one
   call stack per thread. An entry pushes a frame and emits a
   `caller → callee` edge; an exit pops it and computes **total time** and
   **self time** (total minus time spent in children). Each root call gets
   its own trace id. Events are streamed to the backend over gRPC
   (`proto/`).
5. **Detect (backend)** — `backend/collector` keeps the last 20 durations of
   each function and computes their p95. The first value becomes the
   baseline; a drift of more than 20% is reported as a `regression`.
   Every event is stored in SQLite with the commit hash.
6. **Visualize** — the React app receives live events over
   `ws://<host>:8080/ws`, draws the call graph with D3, and queries
   `/history/{function}` for past runs.

A second, independent probe — `agent/bpf/monitor.bpf.c` — hooks
`sys_enter_execve` to capture process creation (pid, ppid, process name).
It isn't wired into the profiler pipeline above.

## Key eBPF concepts

- **eBPF** — small programs that run *inside the Linux kernel*, triggered by
  events, without recompiling the kernel or loading a module. Here:
  `trace_enter` and `trace_exit` in `profiler.bpf.c`.
- **uprobe / uretprobe** — the "u" means the *observed* function lives in
  user space; the BPF code itself always runs in the kernel. A uprobe
  replaces the function's first instruction with a breakpoint (entry). A
  uretprobe swaps the return address on the stack for a kernel trampoline
  (exit).
- **Verifier** — checks every BPF program before it loads (bounded loops,
  safe memory access). That's why `bpf_ringbuf_reserve` results are
  null-checked.
- **Maps / ring buffer** — memory shared between kernel and user space. The
  profiler uses a 16 MB `BPF_MAP_TYPE_RINGBUF`; the agent `poll()`s it.
- **Attach cookie** — one BPF program serves every function, so each attach
  passes an index that the program reads back with
  `bpf_get_attach_cookie()`. Cheaper than sending names from the kernel.
- **`pid_tgid`** — `bpf_get_current_pid_tgid()` returns the process id
  (upper 32 bits) and thread id (lower 32 bits); call stacks are keyed by
  that pair, which makes recursion and multi-threading work.
- **`bpf_ktime_get_ns()`** — monotonic nanoseconds since boot; used for
  durations, not wall-clock dates.
- **Logic in user space** — the kernel side only timestamps; pairing,
  self time and detection live in testable user-space code that can be
  reused across language adapters.

## Repository structure

- `agent/` — Rust workspace (uses [aya](https://aya-rs.dev))
  - `bpf/` — kernel-side C programs: `profiler.bpf.c`, `monitor.bpf.c`,
    shared structs/maps in `ebpf_structures.h`, `Makefile`
  - `src/main.rs` — agent binary: wires adapter → call stack → gRPC
  - `adapter/` — language adapter (symbol scan, attach, ring buffer reader)
  - `profiler-core/` — language-agnostic call stack reconstruction
- `proto/` — gRPC contract between agent and backend
- `backend/` — Go server
  - `cmd/` — entry point: WebSocket/HTTP on `:8080`, gRPC on `:50051`
  - `collector/` — receives agent events, p95 baselines, regression status
  - `stats/` — sliding window and p95
  - `database/` — SQLite storage and the `/history/{name}` handler
  - `broadcast/` — WebSocket fan-out to connected clients
  - `pb/` — generated gRPC code
- `frontend/ebpf-project/` — React + Vite + D3 dashboard
- `test/` — targets to profile
  - `rust_target/src/main.rs` — minimal `main → func1 → func2 → func3` chain
  - `rust_target/src/bin/orders.rs` — multi-threaded order pipeline with
    recursion, a shared callee, and a `SLOW_LOOKUP` regression toggle
  - `testapp`, `target_service/` — older Go targets

## Running it

Requires Linux with BTF support, clang, Rust, Go and Node. The BPF
`Makefile` targets arm64; change `-D__TARGET_ARCH_arm64` for x86.
Build the target **on Linux** — uprobes need an ELF binary.

1. **Build the eBPF object and the agent**

   ```
   cd agent/bpf && make
   cd .. && cargo build
   ```

2. **Build a target.** Functions you want as separate frames need
   `#[inline(never)]`.

   ```
   cd test/rust_target && cargo build --bin orders
   ```

3. **Start the backend** with the commit hash of the target:

   ```
   cd backend && go run ./cmd $(git rev-parse HEAD)
   ```

4. **Start the agent** (root is needed to load eBPF):

   ```
   sudo ./agent/target/debug/agent \
     "$PWD/test/rust_target/target/debug/orders" \
     agent/bpf/profiler.bpf.o \
     orders:: \
     [collector-addr, default http://127.0.0.1:50051]
   ```

5. **Run the target** — `./test/rust_target/target/debug/orders`

6. **Start the frontend**

   ```
   cd frontend/ebpf-project && npm install && npm run dev
   ```

### Regression demo

Set `SLOW_LOOKUP = true` in `test/rust_target/src/bin/orders.rs`, commit,
rebuild, then restart **both** the backend (new commit hash) and the agent
(probes are bound to the old binary file) and run `orders` again.
`find_customer` should be flagged as a regression.

## Limitations / status

Work in progress. Known limitations:

- **Baselines are in memory only** — they reset when the backend restarts,
  so comparisons across commits are not persisted yet.
- **Overhead** — each probe costs roughly 1–2 µs, which distorts very small,
  hot functions.
- **async code** — with futures, entry/exit marks a `poll`, not a logical
  call, so durations and call edges aren't meaningful yet.
- **Rebuilds** — probes attach to the binary's inode; restart the agent
  after rebuilding the target.
- **Attaching mid-run** — calls already in progress are skipped (orphan
  exits are ignored).
- **Ring buffer full** — events are dropped silently.
- The frontend's history URL is hardcoded in
  `frontend/ebpf-project/src/components/Dashboard.tsx`.
