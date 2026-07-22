
/* definition structure
    __EXECVE_H => define the structure to use as a template to be then used across all the the c files:
    if this template has already been defined then used
*/

#ifndef __EXECVE_H
#define __EXECVE_H
struct process_event {
    __u32 process_id;
    __u32 process_parent_id;
    char name_of_process[16];
};
#endif

#ifndef __LATENCY_H
#define __LATENCY_H
struct latency_event{
    __u64 durations_ns;
    __u64 memory_id;
    __u32 pid;
    char name_of_process[16];
};
#endif

#ifndef __MEMORY_MAP
#define __MEMORY_MAP
struct {
	__uint(type,BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, __u64);
	__type(value,__u64);
} memory_map SEC(".maps");
#endif



/* ring buffer map
    template pattern 
    declaration of a kernel map object 
    RING_BUFFER structure where is gonna be passed the process information
    kernel_object_map
    transport kernel -> userspace 
    streaming buffer
    __uint(type, BPF_MAP_TYPE_RINGBUF) -> sets the map type -> ring buffer
    __uint(max_entries, 1 << 24) -> is the maximum memory allocated for events
    
    event SEC -> Tells the compiler to put in the ELF section ".maps"
*/
#ifndef __RING_BUFFER_H
#define __RING_BUFFER_H
struct  {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps"); 
#endif

#ifndef __BPF_MAP_TYPE_HASH
#define __BPF_MAP_TYPE_HASH
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key,   __u64);
    __type(value, __u64);
} start_times SEC(".maps");
#endif
