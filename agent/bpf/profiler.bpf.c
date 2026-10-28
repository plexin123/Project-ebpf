#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include "ebpf_structures.h"
#include <bpf/bpf_tracing.h>  

SEC("uprobe")
int trace_enter(struct pt_regs *ctx){
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 ts = bpf_ktime_get_ns();
    __u64 cookie = bpf_get_attach_cookie(ctx);
    struct event *enter_event = bpf_ringbuf_reserve(&events, sizeof(struct event),0);
    if(!enter_event){
        return 0;
    }
    __builtin_memset(enter_event, 0, sizeof(*enter_event));
    enter_event->event_type = 0;
    enter_event->time_stamp = ts;
    enter_event->pid_tgid = pid_tgid;
    enter_event->cookie = cookie;
    bpf_ringbuf_submit(enter_event, 0);
    
    return 0;   
}

SEC("uretprobe")
int trace_exit(struct pt_regs *ctx){
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u64 exit_time = bpf_ktime_get_ns();
    __u64 cookie = bpf_get_attach_cookie(ctx);
    struct event *exit_event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
    if(!exit_event){
        return 0;
    }               // target, fill, size
    __builtin_memset(exit_event, 0, sizeof(*exit_event));
    exit_event->event_type = 1;
    exit_event->time_stamp = exit_time;
    exit_event->pid_tgid = pid_tgid;
    exit_event->cookie = cookie;
    bpf_ringbuf_submit(exit_event, 0);
    return 0;

}

char LICENSE[] SEC("license") = "GPL";
