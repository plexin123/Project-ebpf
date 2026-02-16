#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>


struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __unit(max_entries,1024);
    __type(key, __u32);
    __type(value, __u64);
} lumen_trace SEC(".maps");

SEC("xdp")
int lumen_trace(struct xdp_md *ctx){
    __u64 packet_size = ctx->data_end - ctx->data;
    __u32 dummy_ip = 1234
    // update map
    // idea 
    // first we need to check if the key exists in the map
    // if the key doesnt exists we add in the map
    // then we need to add the value which is the packets size
    __u64 *value = bpf_map_lookup_elem(&lumen_trace, &dummy_ip)

    if (!value){
            bpf_map_update_elem(&lumen_trace,&dummy_ip, &packet_size, BPF_ANY)
    }
    else{
        __sync_fetch_and_add(value, packet_size)
    }
    return XDP_PASS;
}
