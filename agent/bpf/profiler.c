#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include "ebpf_structures.h"


SEC("uprobe/trace_center")
int trace_center(struct pt_regs *ctx){
    __u64 pid = bpf_get_current_pid_tgid() >> 32
    
    __u64 ts = bpf_ktime_get_ns();

    bpf_map_update_elem(&start_times , &pid, &ts, BPF_ANY);
    return 0;   
}

SEC("uretprobe/trace_exit")
int trace_exit(struct pt_regs *ctx){
    __u64  pid = bpf_get_current_pid_tgid() >> 32;
    __u64 *ts = bpf_map_lookup_elem(&start_times,&pid);
    if(!ts){
        return 0;
    }
    __u64 duration := bpf_ktime_get_ns() - *ts;
    bpf_map_delete_elem(&start_times, &pid)

    struct latency_event *event = bpf_ringbuf_reserve(&latency_event, sizeof(*event), 0);
    
    if(!event){
        return 0;
    }

    event->pid = (__u32)pid;
    event->durations_ns = duration;
    bpf_get_current_comm(&event->name_of_process, sizeof(event->name_of_process))
    
    bpf_ringbuf_submit(e, 0);
    return 0;

}

char LICENSE[] SEC("license") = "GPL"