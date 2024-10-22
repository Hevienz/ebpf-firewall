//go:build ignore

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/in.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

struct packet_info {
    __be32 src_ip;
    __be32 dst_ip;
    __be32 src_ipv6[4];
    __be32 dst_ipv6[4];
    __be16 src_port;
    __be16 dst_port;
    unsigned char src_mac[ETH_ALEN];
    unsigned char dst_mac[ETH_ALEN];
    __u16 eth_proto;
    __u16 ip_proto;
    __u32 pkt_size;
} __attribute__((packed));

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(int));
    __uint(value_size, sizeof(int));
    __uint(max_entries, 128);
} events SEC(".maps");

struct black_event {
    unsigned char mac[ETH_ALEN];
    union {
        __be32 ipv4;
        __be32 ipv6[4];
    } ip;
    __u16 ip_version;
} __attribute__((packed));

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __be32);
    __type(value, __u8);
    __uint(max_entries, 512);
} ipv4_blacklist SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __be32[4]);
    __type(value, __u8);
    __uint(max_entries, 256);
} ipv6_blacklist SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, unsigned char[ETH_ALEN]);
    __type(value, __u8);
    __uint(max_entries, 256);
} mac_blacklist SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32));
} black_events SEC(".maps");


static __always_inline void parse_transport(struct packet_info *pkt_info, void *data, void *data_end, __u8 proto) {
    pkt_info->ip_proto = proto;

    switch (proto) {
    case IPPROTO_TCP: {
        struct tcphdr *tcp = data;
        if ((void *)(tcp + 1) > data_end) return;
        pkt_info->src_port = tcp->source;
        pkt_info->dst_port = tcp->dest;
        break;
    }
    case IPPROTO_UDP: {
        struct udphdr *udp = data;
        if ((void *)(udp + 1) > data_end) return;
        pkt_info->src_port = udp->source;
        pkt_info->dst_port = udp->dest;
        break;
    }
    default:
        break;
    }
}

SEC("xdp")
int xdp_prog(struct xdp_md *ctx)
{
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    struct ethhdr *eth = data;

    if (data + sizeof(struct ethhdr) > data_end)
        return XDP_PASS;

    struct packet_info pkt_info = {};

    // 复制MAC地址
    bpf_probe_read_kernel(pkt_info.src_mac, ETH_ALEN, eth->h_source);
    __u8 *mac_blocked = bpf_map_lookup_elem(&mac_blacklist, pkt_info.src_mac);
    // 如果MAC地址在黑名单中
    if (mac_blocked) {
        struct black_event evt = {};
        evt.ip_version = 0; // Indicate MAC-based block
        bpf_probe_read_kernel(evt.mac, ETH_ALEN, eth->h_source); // copy mac to evt
        bpf_perf_event_output(ctx, &black_events, BPF_F_CURRENT_CPU, &evt, sizeof(evt));
        return XDP_DROP;
    }

    bpf_probe_read_kernel(pkt_info.dst_mac, ETH_ALEN, eth->h_dest);

    pkt_info.eth_proto = bpf_ntohs(eth->h_proto);
    pkt_info.pkt_size = data_end - data;

    switch (pkt_info.eth_proto) {
    case ETH_P_IP: {
        struct iphdr *ip = data + sizeof(struct ethhdr);
        if ((void *)(ip + 1) > data_end)
            goto submit;

        pkt_info.src_ip = ip->saddr;
        pkt_info.dst_ip = ip->daddr;

        __u8 *blocked = bpf_map_lookup_elem(&ipv4_blacklist, &pkt_info.src_ip);
        if (blocked) {
            struct black_event evt = {};
            evt.ip_version = 1;
            evt.ip.ipv4 = pkt_info.src_ip;
            bpf_perf_event_output(ctx, &black_events, BPF_F_CURRENT_CPU, &evt, sizeof(evt));
            return XDP_DROP;
        }

        parse_transport(&pkt_info, (void *)(ip + 1), data_end, ip->protocol);
        break;
    }
    case ETH_P_IPV6: {
        struct ipv6hdr *ip6 = data + sizeof(struct ethhdr);
        if ((void *)(ip6 + 1) > data_end)
            goto submit;

        bpf_probe_read_kernel(pkt_info.src_ipv6, sizeof(pkt_info.src_ipv6), &ip6->saddr);
        bpf_probe_read_kernel(pkt_info.dst_ipv6, sizeof(pkt_info.dst_ipv6), &ip6->daddr);

        __u8 *blocked = bpf_map_lookup_elem(&ipv6_blacklist, &pkt_info.src_ipv6);
        if (blocked) {
            struct black_event evt = {};
            evt.ip_version = 2;
            bpf_probe_read_kernel(evt.ip.ipv6, sizeof(pkt_info.src_ipv6), &ip6->saddr);
            bpf_perf_event_output(ctx, &black_events, BPF_F_CURRENT_CPU, &evt, sizeof(evt));
            return XDP_DROP;
        }

        parse_transport(&pkt_info, (void *)(ip6 + 1), data_end, ip6->nexthdr);
        break;
    }
    default:
        break;
    }

submit:
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &pkt_info, sizeof(pkt_info));

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
