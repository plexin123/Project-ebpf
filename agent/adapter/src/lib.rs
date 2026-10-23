use std::fs;
use std::os::fd::AsRawFd;
use profiler_core::callstack;
use anyhow::{Context, Result};
use aya::Ebpf;
use aya::maps::RingBuf;
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
    fn reader(&mut self, callstack : callstack::CallStack) -> Result<()>; // fn reader(...) -> ???   (pendiente: ¿cero, uno o muchos eventos?)
}

const ENTER_EVENT : u8 = 0;
const EXIT_EVENT : u8 = 1;

pub struct FunctionEvent{
    event_type:  u8,
    pid_tgid:  u64,
    func_address: u64,
    time_stamp: u64,
}

pub struct RustAdapter {
    ebpf: Ebpf, // si se destruye, se van los probes
    binary_path: String,
    crate_prefix: String,
    functions: Vec<FunctionInfo>,
}




impl RustAdapter {
    pub fn new(binary_path: String, bpf_path: &str, crate_prefix: String) -> Result<Self> {
        let ebpf = Ebpf::load_file(bpf_path)
            .with_context(|| format!("no se pudo cargar {bpf_path}"))?;
        Ok(Self {
            ebpf,
            binary_path,
            crate_prefix,
            functions: Vec::new(),
        })
    }
}

pub fn wait_readable(fd : i32){
        let mut fds = [pollfd{
            fd : fd,
            events : POLLIN,
            revents : 0,
        }];
        let timeout = 5000;
        let ret = unsafe {
            poll(fds.as_mut_ptr(), fds.len() as libc::nfds_t, timeout)
        };
        Ok(());
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

    fn reader(&mut self,  callstack : callstack::CallStack) ->  Result<()>{
        let mut ring = RingBuf::try_from(self.ebpf.take_map("events").context("there is not map events")?)?;
        let mut fd_ring = ring.as_raw_fd();
        loop{
            // sleep when there is no data
            wait_readable(fd_ring);
            
            while let Some(item) = ring.next(){
                let event_type = item[0];
                let thread_id  = u32::from_le_bytes(item[1..5].try_into()?);
                let process_id = u32::from_le_bytes(item[5..9].try_into()?);
                let cookie = u64::from_le_bytes(item[9..17].try_into()?);
                let time_stamp = u64::from_le_bytes(item[17..25].try_into()?);
                let identifier = callstack::UniqueKeyIdentifier { pid: process_id, generic_key: thread_id };
                let func_name = self.functions.get(cookie as usize).unwrap().demangled_name;
                // separate the information according to the type
                match event_type {
                        ENTER_EVENT => {
                 
                            if let Some((call_event, trace_id)) = callstack.Enter(identifier, func_name, time_stamp){
                                // send grpc 
                            }
    
                    }   
                        EXIT_EVENT => {
                            let self_time: u64 = callstack.Exit(identifier, time_stamp);
                            let current_trace_id = callstack.map_trace_id.get(&identifier);
                            // send grpc
                            

                    }
                }
                    // enter type -> CallEvent send gRPC
                    // exit type ->  FunctionEvent / Core / send gRPC
            }

        }
        
    }

    
}
