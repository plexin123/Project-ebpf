#include "vmlinux.h"   
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>             
#include "ebpf_structures.h"

//  Capture packets
//  Add that those metrics in a map
//  Counters by key
//  Verifier safe
//  Then this is consumed in go

// hooks into the exact moment a process attempts to execute a new program on Linux
// define an struct of the event 
SEC("tracepoint/syscalls/sys_enter_execve")
int new_program(struct trace_event_raw_sys_enter *ctx){

    // pid_tgid packs both values: [ tgid (upper 32) | tid (lower 32) ]
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    
    __u32 pid = pid_tgid >> 32;  // tgid = what userspace calls pid
	
    struct task_struct *task = (struct task_struct *)bpf_get_current_task_btf();
    __u32 ppid = BPF_CORE_READ(task,real_parent, tgid);
    //Declare as a pointer
    struct process_event *process_event;
    //Use this variable that will the memory address of the 
    process_event = bpf_ringbuf_reserve(&events, sizeof(struct process_event), 0);


    if(!process_event){
        return 0;
    }
        
    process_event->process_id = pid;
    process_event->process_parent_id = ppid;

    bpf_get_current_comm(&process_event->name_of_process, sizeof(process_event->name_of_process));

    bpf_ringbuf_submit(process_event,0);
    
    return 0;
}


char LICENSE[] SEC("license") = "GPL";

