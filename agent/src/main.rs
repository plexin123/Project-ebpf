use adapter::adapter::{Adapter, EventType, RustAdapter};
use anyhow::{Context, Result};
use profiler::{AgentEvent, CallEdge, CallResult, agent_event::Kind, routing_functions_client::RoutingFunctionsClient};
use profiler_core::callstack::CallStack;
use tokio::sync::mpsc;
use tokio_stream::wrappers::ReceiverStream;

pub mod profiler {
    tonic::include_proto!("profiler");
}

// usage: agent <target-binary> <profiler.bpf.o> [crate-prefix] [collector-addr]
#[tokio::main]
async fn main() -> Result<()> {
    let mut args = std::env::args().skip(1);
    let binary_path = args.next().context("missing <target-binary>")?;
    let bpf_path = args.next().context("missing <profiler.bpf.o>")?;
    let crate_prefix = args.next().unwrap_or_else(|| "main".to_string());
    let collector_addr = args.next().unwrap_or_else(|| "http://127.0.0.1:50051".to_string());

    // attach before connecting: if the probes fail there is nothing to stream
    let mut rust_adapter = RustAdapter::new(binary_path, &bpf_path, crate_prefix)?;
    rust_adapter.scan()?;
    rust_adapter.map()?;

    let mut client = RoutingFunctionsClient::connect(collector_addr.clone())
        .await
        .with_context(|| format!("could not connect to {collector_addr}"))?;

    // reader() blocks on poll(), so it gets its own OS thread instead of a tokio task
    let (tx, rx) = mpsc::channel::<AgentEvent>(1024);
    let reader = std::thread::spawn(move || -> Result<()> {
        let mut callstack = CallStack::new();
        loop {
            let data = rust_adapter.reader()?;
            let kind = match data.event_type {
                EventType::Enter => {
                    let edge = callstack.enter(data.unique_identifier, data.func_name, data.time_stamp);
                    Kind::Edge(CallEdge {
                        trace_id: edge.trace_id,
                        caller: edge.caller,
                        callee: edge.callee,
                        pid: data.process_id,
                        tid: data.thread_id,
                    })
                }
                EventType::Exit => {
                    // no matching enter: the function was already running when we attached
                    let Some(result) = callstack.exit(data.unique_identifier, data.time_stamp) else {
                        continue;
                    };
                    Kind::Result(CallResult {
                        trace_id: result.trace_id,
                        function: result.function,
                        time_stamp: data.time_stamp,
                        self_time_ns: result.self_time,
                        pid: data.process_id,
                        tid: data.thread_id,
                    })
                }
            };
            // Err means the gRPC side dropped the receiver -> stop reading
            if tx.blocking_send(AgentEvent { kind: Some(kind) }).is_err() {
                return Ok(());
            }
        }
    });

    // keeps main alive for as long as the stream is open
    client.streaming_data(ReceiverStream::new(rx)).await?;

    reader.join().expect("reader thread panicked")?;
    Ok(())
}
