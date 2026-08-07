#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include "ebpf_structures.h"
#include <bpf/bpf_tracing.h>  

SEC("uprobe")
int trace_enter(struct pt_regs *ctx){
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ts = bpf_ktime_get_ns();
    __u64 stack_pointer_id = PT_REGS_IP(ctx);


    //send data to ringbuf
    struct enter_event *enter_event = bpf_ringbuf_reserve(&enter_events, sizeof(struct enter_event),0);
    if(enter_event){
    enter_event->pid_tgid = pid_tgid;
    enter_event->func_address = stack_pointer_id;
    bpf_ringbuf_submit(enter_event, 0);
    }
    bpf_map_update_elem(&start_times , &pid_tgid, &ts, BPF_ANY);
    bpf_map_update_elem(&memory_map, &pid_tgid, &stack_pointer_id, BPF_ANY);
    return 0;   
}

SEC("uretprobe")
int trace_exit(struct pt_regs *ctx){
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 *ts = bpf_map_lookup_elem(&start_times,&pid_tgid);
    __u64 *stack_pointer_id = bpf_map_lookup_elem(&memory_map,&pid_tgid);
    if(!ts){
        return 0;
    }
    if(!stack_pointer_id){
	return 0;
    }
    __u64 duration = bpf_ktime_get_ns() - *ts;
    bpf_map_delete_elem(&start_times, &pid_tgid);
    bpf_map_delete_elem(&memory_map, &pid_tgid);
    struct latency_event *event = bpf_ringbuf_reserve(&events, sizeof(struct latency_event), 0);

    
    if(!event){
        return 0;
    }

    event->pid_tgid = pid_tgid;
    // add attribute memory address -> uint64
    event->memory_id = *stack_pointer_id; 
    event->durations_ns = duration;
    bpf_get_current_comm(&event->name_of_process, sizeof(event->name_of_process));
   
    bpf_ringbuf_submit(event, 0);
    return 0;

}

char LICENSE[] SEC("license") = "GPL";
