# eBPF / XDP Architecture for RINCO

> **Status:** Documentation only — no eBPF/XDP runtime is shipped in
> this milestone.  The notes below describe the architecture that the
> platform team will deploy in a follow-up loop (WS-E2).  The current
> observability + security stack already meets the requirements
> through user-space rate limiting (Valkey + in-memory token bucket)
> and Traefik middleware.  This document is the design record for the
> future kernel-level DDoS mitigation.

---

## 1. Goals

| ID    | Goal                                                          | Today (Traefik + Valkey) | With eBPF/XDP            |
| ----- | ------------------------------------------------------------- | ------------------------ | ------------------------ |
| DD-1  | 10 Gbps line-rate SYN flood absorption                        | ~3 Gbps                  | 25 Gbps (NIC-bound)      |
| DD-2  | Per-source token bucket at < 1 ms latency                     | ~10 ms (usercopy)        | < 50 µs (kernel)         |
| DD-3  | JA4+ TLS fingerprint at edge                                  | n/a                      | yes                      |
| DD-4  | Atomic counter updates (no race conditions)                   | approximate              | yes (BPF atomic)         |
| DD-5  | No third-party kernel modules (no Suricata / Cilium coupling) | n/a                      | yes                      |

---

## 2. Architecture

```
                              ┌────────────────────────────────────────────┐
                              │            User-space controller            │
                              │   (rinco-ebpf-ctl — Go binary in cluster)  │
                              │   - Reads IP threat feeds (CSV / Valkey)    │
                              │   - Updates BPF maps via `bpf(2)`           │
                              │   - Exposes :9091/metrics → Prometheus     │
                              │   - Hot-swaps XDP program version          │
                              └─────────────────┬──────────────────────────┘
                                                │
                                                │ bpf_map_update_elem
                                                ▼
┌──────────────────────────────────────────────────────────────────────┐
│                       Linux 6.x kernel                                │
│                                                                       │
│   ┌──────────────────────┐   ┌──────────────────────┐                │
│   │   XDP program        │   │   TC ingress clsact   │                │
│   │  (xdp_rinco.bpf.c)   │   │ (tc_rinco.bpf.c)      │                │
│   │  NIC-driver level    │   │  (post-XDP, pre-stack)│                │
│   │  - blacklist lookup  │   │  - L7 fingerprint     │                │
│   │  - rate limit map    │   │  - geo-block          │                │
│   │  - syncookie         │   │  - JA4+ extraction    │                │
│   └─────────┬────────────┘   └─────────┬────────────┘                │
│             │  XDP_PASS               │                              │
│             ▼                          ▼                              │
│   ┌──────────────────────────────────────────────────┐               │
│   │   Linux TCP/IP stack                              │               │
│   └──────────────────────────────────────────────────┘               │
└──────────────────────────────────────────────────────────────────────┘
                │
                ▼
        Traefik (L7) / Service pods
```

### 2.1 Component responsibilities

| Component                | Lives in           | Responsibility                                                     |
| ------------------------ | ------------------ | ------------------------------------------------------------------ |
| `xdp_rinco.bpf.c`        | NIC driver (XDP)   | First-line filter: blacklist, rate limit, SYN cookies              |
| `tc_rinco.bpf.c`         | `tc` clsact        | L4-L7 fingerprint, geo-block, JA4+ extraction                      |
| `rinco-ebpf-ctl`         | Pod / DaemonSet    | Reads threat feeds, updates BPF maps, exposes metrics              |
| `rinco-ebpf-pin`         | `/sys/fs/bpf/rinco`| Stable pinned BPF objects across reboots                           |

### 2.2 Maps layout (BPF)

```c
struct rate_limit_key {
    __u32 src_ip;          // IPv4 only initially
    __u32 prefixlen;       // /32 or /24 for dynamic aggregation
};

struct rate_limit_val {
    __u64 tokens;
    __u64 last_refill_ns;
    __u32 hits;            // for visibility
    __u32 drops;           // for visibility
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 256 * 1024);  // 256k active sources
    __type(key, struct rate_limit_key);
    __type(value, struct rate_limit_val);
} rinco_rate_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 1024);
    __type(key, __u32);              // ASN or country code
    __type(value, __u64);            // block until ns
} rinco_geo_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 4);
    __type(key, __u32);
    __type(value, __u64);            // tokens_per_sec, burst, syncookie_enabled, version
} rinco_config_map SEC(".maps");
```

---

## 3. XDP DDoS filter (full source — design only)

The C source below is the design target.  It is **not compiled into
the current repo**; it ships in WS-E2 once the cluster has been
provisioned with kernel 6.6+ and `libbpf`/`bpftool` available.

```c
// xdp_rinco.bpf.c — design target (WS-E2)
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define ETH_P_IP 0x0800

struct rate_limit_key { __u32 src_ip; };
struct rate_limit_val { __u64 tokens; __u64 last_refill_ns; __u32 hits; __u32 drops; };

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 256 * 1024);
    __type(key, struct rate_limit_key);
    __type(value, struct rate_limit_val);
} rinco_rate_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 4);
    __type(key, __u32);
    __type(value, __u64);
} rinco_config_map SEC(".maps");

static __always_inline __u64 now_ns(void) { return bpf_ktime_get_ns(); }

SEC("xdp")
int xdp_rinco_antiddos(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;
    if (eth->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) return XDP_PASS;

    __u32 saddr = ip->saddr;

    // 1. Token bucket rate limit
    __u32 zero = 0;
    __u64 *rate = bpf_map_lookup_elem(&rinco_config_map, &zero);
    if (!rate) return XDP_PASS;
    __u64 burst_key = 1;
    __u64 *burst = bpf_map_lookup_elem(&rinco_config_map, &burst_key);
    if (!burst) return XDP_PASS;

    struct rate_limit_key k = { .src_ip = saddr };
    struct rate_limit_val *v = bpf_map_lookup_elem(&rinco_rate_map, &k);
    __u64 now = now_ns();
    if (!v) {
        struct rate_limit_val init = {
            .tokens = *burst,
            .last_refill_ns = now,
        };
        bpf_map_update_elem(&rinco_rate_map, &k, &init, BPF_ANY);
        return XDP_PASS;
    }
    __u64 elapsed_ns = now - v->last_refill_ns;
    __u64 refill = (elapsed_ns * *rate) / 1000000000ULL;
    v->tokens = (v->tokens + refill > *burst) ? *burst : v->tokens + refill;
    v->last_refill_ns = now;
    v->hits += 1;
    if (v->tokens == 0) {
        v->drops += 1;
        return XDP_DROP;
    }
    v->tokens -= 1;
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

---

## 4. DDoS mitigation flow

### 4.1 Layered response

```
   packet
     │
     ▼
┌──────────────────────┐
│ XDP — rate limit     │  < 1 ms p99 — drops most garbage
└──────────┬───────────┘
           ▼  pass
┌──────────────────────┐
│ tc — JA4+ / geo      │  < 1 ms p99 — fingerprint / geo
└──────────┬───────────┘
           ▼  pass
┌──────────────────────┐
│ Traefik — L7         │  < 5 ms p99 — arg-2 / CAPI / bot scoring
└──────────┬───────────┘
           ▼  pass
┌──────────────────────┐
│ Service              │  application logic
└──────────────────────┘
```

### 4.2 Threat feed → BPF map updates

```mermaid
flowchart LR
    A[Threat feed<br/>(Cloudflare / Spamhaus)] --> B[rinco-ebpf-ctl]
    C[Prometheus alert] --> B
    D[Operator CLI] --> B
    B -->|bpf_map_update_elem| E[BPF maps]
    E --> F[XDP / TC programs]
    F -->|drops metric| G[Prometheus]
    G -->|alert| C
```

The `rinco-ebpf-ctl` binary:

1. Polls `https://www.spamhaus.org/drop/drop.txt` every 60 s.
2. Polls Cloudflare's API for confirmed botnets every 5 min.
3. Watches `rinco_threat_feed_updated` metric for operator-driven
   updates.
4. Batches 1000 IPs per `bpf_map_update_elem` syscall to amortise
   the kernel trip.

### 4.3 Auto-blackhole

When the same /24 IP block generates > 10,000 drops in 60 s:

1. `rinco-ebpf-ctl` marks the /24 with `block_until = now + 24h`.
2. Emits `rinco_autoblock_total{prefix="x.y.z.0/24"}` counter.
3. Optionally POSTs to upstream provider (Cloudflare BGP / path)
   to push blackhole route.

---

## 5. Integration with observability stack

| Metric                                 | Source          | Panel             |
| -------------------------------------- | --------------- | ----------------- |
| `rinco_xdp_drops_total`                | BPF map         | System Overview   |
| `rinco_xdp_passes_total`               | BPF map         | System Overview   |
| `rinco_tc_j4a_total{verdict="drop"}`   | tc program      | WebRTC / SFU      |
| `rinco_ebpf_map_size{map="rate"}`      | userspace ctl   | Infrastructure    |
| `rinco_autoblock_total{prefix="..."}`  | userspace ctl   | System Overview   |

All metrics are scraped from `rinco-ebpf-ctl:9091/metrics` and
forwarded via the existing Prometheus pipeline.  No new
infrastructure is required.

---

## 6. Build / deploy (for WS-E2 reference)

```bash
# Compile
clang -O2 -g -target bpf -D__TARGET_ARCH_x86 \
    -I/usr/include/x86_64-linux-gnu \
    -c xdp_rinco.bpf.c -o xdp_rinco.bpf.o

# Generate skeleton
bpftool gen skeleton xdp_rinco.bpf.o > xdp_rinco.skel.h

# Build userspace controller
go build -o bin/rinco-ebpf-ctl ./cmd/ebpf-ctl

# Deploy as DaemonSet (one pod per node)
kubectl apply -f infra/k8s/observability/ebpf-ds.yaml
```

---

## 7. Acceptance criteria

| AC      | Criterion                                  | Measurement                |
| ------- | ------------------------------------------ | -------------------------- |
| WS-E-AC-DD-1 | 25 Gbps SYN flood absorbed   | iperf3 + hping3 in chaos   |
| WS-E-AC-DD-2 | p99 < 50 µs in XDP hook       | bpftool prof               |
| WS-E-AC-DD-3 | Auto-blackhole < 60 s         | e2e drill                  |
| WS-E-AC-DD-4 | No kernel panic for 24 h      | soak test                  |
| WS-E-AC-DD-5 | map updates < 10 ms / 1k IPs  | bench                      |

---

## 8. Open questions

1. Do we run XDP in `driver` or `generic` mode?  Driver mode
   requires driver support; generic mode is portable but slower.
2. Should we additionally ship a `cilium`-based layer for L7
   policy enforcement, or stay kernel-module-free?
3. Do we need to integrate with Wanguard / RTBH upstream
   providers, or is Cloudflare sufficient?

---

**Related documents:**

- [09-security/README.md](./README.md) — full security stack.
- [../../DEV-PLAN.md §10](../../DEV-PLAN.md) — observability &
  security hardening scope.