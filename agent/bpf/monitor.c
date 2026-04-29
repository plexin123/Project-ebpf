#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>


struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries,1024);
    __type(key, __u32);
    __type(value, __u64);
} lumen_trace SEC(".maps");


//  Capture packets
//  Add that those metrics in a map
//  Counters by key
//  Verifier safe
//  Then this is consumed in go

SEC("xdp")
int lumen_trace(struct xdp_md *ctx){
    
    // data_end __u32
    // data __u32
    __u32 packet_size = ctx->data_end - ctx->data;
    
    // convert the data a __u32 type ->  64 bits -> convert into a pointer
    void *data = (void *)(long) ctx->data;
    // pointer to a pointer -> basically explaining that its pointing to a structure
    struct ethhdr *eth = data
    
    void *ipdata = (void *) data + sizeof(struct ethhdr)
 
    struct iphdr *iph = ipdata
    
    char *type_of_protocol = NULL

    if (iph->protocol == 6){
        type_of_protocol = "TCP"
    }
    else if(iph->protocol == 17){
        type_of_protocol = "UDP"
    }
    
    // get the the type of communication layer TCP or UDP
    
    __u32 dummy_ip = 1234;
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
