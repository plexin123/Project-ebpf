
use std::fs;
use std::io::Error;
use std::os::fd::AsRawFd;
use profiler_core::callstack::UniqueKeyIdentifier;
use anyhow::{Context, Result, bail};
use aya::Ebpf;
use aya::maps::{MapData, RingBuf};
use aya::programs::UProbe;
use aya::programs::uprobe::{UProbeAttachLocation, UProbeAttachPoint, UProbeScope};
use object::{Object, ObjectSymbol, SymbolKind};
use rustc_demangle::demangle;
use libc::{POLLIN, poll, pollfd};

// datos propios (String, u64): no dependen del Vec<u8> que lee scan
pub struct FunctionInfo {
    pub mangled_name: String,   // para attach
    pub demangled_name: String, // para mostrar / filtrar
    pub address: u64,
}


pub trait Adapter {
    fn scan(&mut self) -> Result<()>;
    fn map(&mut self) -> Result<()>;
    // bloquea hasta que llegue un evento
    fn reader(&mut self) -> Result<FunctionEvent>;
}

// valores de event_type escritos por profiler.bpf.c
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EventType {
    Enter,
    Exit,
}

impl TryFrom<u8> for EventType {
    type Error = anyhow::Error;
    fn try_from(value: u8) -> Result<Self> {
        match value {
            0 => Ok(EventType::Enter),
            1 => Ok(EventType::Exit),
            other => bail!("unknown event_type {other}"),
        }
    }
}

pub struct RustAdapter {
    ebpf: Ebpf,
    binary_path: String,
    crate_prefix: String,
    functions: Vec<FunctionInfo>,
    ring_buffer: RingBuf<MapData>,
}

pub struct FunctionEvent{
    pub event_type: EventType,
    pub thread_id : u32,
    pub process_id : u32,
    pub unique_identifier : UniqueKeyIdentifier,
    pub func_name: String,
    pub time_stamp: u64,

}



impl RustAdapter {
    pub fn new(binary_path: String, bpf_path: &str, crate_prefix: String) -> Result<Self> {
        let mut ebpf = Ebpf::load_file(bpf_path)
            .with_context(|| format!("no se pudo cargar {bpf_path}"))?;
        // take_map antes de mover ebpf al struct
        let ring_buffer = RingBuf::try_from(ebpf.take_map("events").context("there is not map events")?)?;
        Ok(Self {
            ebpf,
            binary_path,
            crate_prefix,
            functions: Vec::new(),
            ring_buffer,
        })
    }
}

pub fn wait_readable(fd : i32) -> Result<()>{
        let mut fds: [pollfd; 1] = [pollfd{
            fd : fd,
            events : POLLIN,
            revents : 0,
        }];
        let timeout = 5000;
        let ret = unsafe {
            poll(fds.as_mut_ptr(), fds.len() as libc::nfds_t, timeout)
        };
        if ret == -1{
            let err = Error::last_os_error();
            if err.kind() ==  std::io::ErrorKind::Interrupted{
                return Ok(());
            }
            return Err(err.into());
        }
        // ret == 0 -> timeout, ret > 0 -> data; the caller just tries to read again
        Ok(())
    }




impl Adapter for RustAdapter {
    // leer los simbolos del binario que se va a perfilar
    fn scan(&mut self) -> Result<()> {
        let bytes = fs::read(&self.binary_path)
            .with_context(|| format!("no se pudo leer {}", self.binary_path))?;
        let file = object::File::parse(&*bytes)?;

        for sym in file.symbols() {
            if sym.kind() != SymbolKind::Text || !sym.is_definition() || sym.size() == 0 {
                continue;
            }
            let Ok(name) = sym.name() else { continue };
            // {:#} quita el hash final: target::foo en vez de target::foo::h8a3f...
            let demangled = format!("{:#}", demangle(name));
            if !demangled.starts_with(&self.crate_prefix) {
                continue;
            }
            self.functions.push(FunctionInfo {
                mangled_name: name.to_string(),
                demangled_name: demangled,
                address: sym.address(),
            });
        }
        Ok(())
    }

    fn map(&mut self) -> Result<()> {
        for program_name in ["trace_enter", "trace_exit"] {
            let program: &mut UProbe = self
                .ebpf
                .program_mut(program_name)
                .with_context(|| format!("{program_name} no está en el .o"))?
                .try_into()?;
            program.load()?; // aquí corre el verifier

            for (i, f) in self.functions.iter().enumerate() {
                let point = UProbeAttachPoint {
                    location: UProbeAttachLocation::Symbol(&f.mangled_name),
                    cookie: Some(i as u64),
                };
                program
                    .attach(point, &self.binary_path, UProbeScope::AllProcesses)
                    .with_context(|| format!("attach {program_name} a {}", f.demangled_name))?;
            }
        }
        Ok(())
    }
    fn reader(&mut self) -> Result<FunctionEvent>{
        let fd_ring = self.ring_buffer.as_raw_fd();
        loop{
            // sleep when there is no data
            wait_readable(fd_ring)?;
            if let Some(item) = self.ring_buffer.next(){
                // struct event (packed): u8 event_type | u64 pid_tgid | u64 cookie | u64 time_stamp
                if item.len() < 25 {
                    bail!("short event: {} bytes", item.len());
                }
                let event_type = EventType::try_from(item[0])?;
                // pid_tgid: low 32 bits = tid, high 32 bits = tgid (the pid in user space)
                let thread_id  = u32::from_le_bytes(item[1..5].try_into()?);
                let process_id = u32::from_le_bytes(item[5..9].try_into()?);
                let cookie = u64::from_le_bytes(item[9..17].try_into()?);
                let time_stamp = u64::from_le_bytes(item[17..25].try_into()?);
                let identifier = UniqueKeyIdentifier { pid: process_id, generic_key: thread_id };
                let func_name = self.functions.get(cookie as usize)
                    .with_context(|| format!("cookie {cookie} is not in the function table"))?
                    .demangled_name.clone();
                return Ok(FunctionEvent{
                    event_type,
                    thread_id,
                    process_id,
                    unique_identifier : identifier,
                    func_name,
                    time_stamp,
                });
            }
        }
    }
}


