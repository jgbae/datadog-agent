#include "classifier.h"
#include "bpf_helpers.h"
#include "ip.h"
#include "tls.h"
#include "classifier-telemetry.h"

#define PROTO_PROG_TLS 0
struct bpf_map_def SEC("maps/proto_progs") proto_progs = {
    .type = BPF_MAP_TYPE_PROG_ARRAY,
    .key_size = sizeof(u32),
    .value_size = sizeof(u32),
    .max_entries = 1,
};

static __always_inline int fingerprint_proto(conn_tuple_t *tup, skb_info_t* skb_info, struct __sk_buff* skb) {
    if (is_tls(skb_info, skb))
        return PROTO_PROG_TLS;

    return 0;
}

SEC("socket/classifier_filter")
int socket__classifier_filter(struct __sk_buff* skb) {
    proto_args_t args;
    skb_info_t* skb_info = &args.skb_info;
    conn_tuple_t* tup = &args.tup;
    __builtin_memset(&args, 0, sizeof(proto_args_t));
    if (!read_conn_tuple_skb(skb, skb_info, tup))
        return 0;

    if (!(tup->metadata&CONN_TYPE_TCP))
        return 0;

    if (skb_info->tcp_flags & TCPHDR_FIN) {
	    bpf_map_delete_elem(&proto_in_flight, tup);
	    return 0;
    }

    cnx_info_t info = bpf_map_lookup_elem(&proto_in_flight, tup);
    if (info != NULL) {
        if ((info->done) || (info->failed))
            return 0;
    }

    normalize_tuple(args.tup);
    int protocol = fingerprint_proto(tup, skb_info, skb);
    u32 cpu = bpf_get_smp_processor_id();
    if (protocol) {
        int err = bpf_map_update_elem(&protocol_args, &cpu, &args, BPF_ANY);
        if (err < 0)
            return 0;

        bpf_tail_call_compat(skb, &proto_progs, protocol);
        increment_classifier_telemetry_count(tail_call_failed);
    }

    return 0;
}

// This number will be interpreted by elf-loader to set the current running kernel version
__u32 _version SEC("version") = 0xFFFFFFFE; // NOLINT(bugprone-reserved-identifier)

char _license[] SEC("license") = "GPL"; // NOLINT(bugprone-reserved-identifier)
