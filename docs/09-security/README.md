# Phần 9 – Multi-Tenant Security & Zero-Trust

> **Phân hệ:** Bảo mật đa tầng cho hệ thống Hyper-Scale Multi-Tenant.  
> **Mục tiêu:** Cách ly tuyệt đối dữ liệu giữa các tenant, chống DDoS, chống Bot, chống Prompt Injection, Admin vô hình.  
> **Triết lý:** Zero-Trust – không tin tưởng bất kỳ request nào mặc định; mọi thứ phải verify.

---

## Mục lục
1. [Mục tiêu & Triết lý](#1-mục-tiêu--triết-lý)
2. [Tầng Network – Anti-DDoS eBPF/XDP](#2-tầng-network--anti-ddos-ebpfxdp)
3. [Tầng Frontend & Ingestion](#3-tầng-frontend--ingestion)
4. [Tầng Application – PASETO & RBAC](#4-tầng-application--paseto--rbac)
5. [Tầng Database – RLS](#5-tầng-database--rls)
6. [Dark Admin](#6-dark-admin)
7. [Tầng Kernel – eBPF RASP](#7-tầng-kernel--ebpf-rasp)
8. [AI Security](#8-ai-security)
9. [Danh sách tính năng (≥ 100)](#9-danh-sách-tính-năng)
10. [Audit & Compliance](#10-audit--compliance)
11. [Incident Response](#11-incident-response)

---

## 1. Mục tiêu & Triết lý

### 1.1. Mục tiêu
| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| SEC-1 | Zero cross-tenant data leak | Audit test 100% |
| SEC-2 | DDoS resistance 10 Gbps | Load test |
| SEC-3 | Bot block 99% | Headless browser test |
| SEC-4 | Admin không lộ public | Nmap scan |
| SEC-5 | RLS bypass impossible | Pen test |

### 1.2. Zero-Trust Principles
1. **Never trust, always verify.**
2. **Assume breach.**
3. **Least privilege.**
4. **Defense in depth.**
5. **Continuous verification.**

---

## 2. Tầng Network – Anti-DDoS eBPF/XDP

### 2.1. Kiến trúc
```
[Internet] 
    │
    ▼
[NIC] ─── eBPF/XDP (filter ở driver level)
    │            ├─ JA4+ Fingerprint
    │            ├─ Rate Limit Map
    │            ├─ Geo-block
    │            └─ Known bot blacklist
    │
    ▼ (pass)
[Linux Kernel TCP/IP Stack]
    │
    ▼
[io_uring Gateway]
```

### 2.2. XDP Program (C)
```c
#include <linux/bpf.h>

struct rate_limit_key {
    u32 src_ip;
};

struct rate_limit_val {
    u64 tokens;
    u64 last_refill_ns;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, struct rate_limit_key);
    __type(value, struct rate_limit_val);
} rate_limit_map SEC(".maps");

SEC("xdp")
int xdp_antiddos(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;
    
    if (eth->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;
    
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) return XDP_PASS;
    
    u32 saddr = ip->saddr;
    
    // Check blacklist
    u32 *blacklist = bpf_map_lookup_elem(&blacklist_map, &saddr);
    if (blacklist) {
        return XDP_DROP;
    }
    
    // Token bucket rate limit
    struct rate_limit_key key = { .src_ip = saddr };
    struct rate_limit_val *val = bpf_map_lookup_elem(&rate_limit_map, &key);
    
    u64 now = bpf_ktime_get_ns();
    u64 rate_per_sec = 100;  // 100 req/s per IP
    u64 burst = 200;          // Allow burst 200
    
    if (!val) {
        struct rate_limit_val init = {
            .tokens = burst,
            .last_refill_ns = now
        };
        bpf_map_update_elem(&rate_limit_map, &key, &init, BPF_ANY);
        return XDP_PASS;
    }
    
    // Refill
    u64 elapsed_ns = now - val->last_refill_ns;
    u64 refill = (elapsed_ns * rate_per_sec) / 1000000000;
    val->tokens = min(burst, val->tokens + refill);
    val->last_refill_ns = now;
    
    if (val->tokens == 0) {
        return XDP_DROP;
    }
    val->tokens--;
    
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

### 2.3. JA4+ Fingerprint
- TLS Client Hello fingerprinting.
- Detect bot dựa trên TLS negotiation pattern.

```c
SEC("xdp")
int xdp_ja4(struct xdp_md *ctx) {
    // Parse TLS ClientHello
    // Extract: version, ciphers, extensions
    // Compute JA4+ hash
    // Lookup in known_bots_map
}
```

### 2.4. BGP Blackhole (Auto)
- Khi phát hiện IP tấn công kéo dài → auto push lên upstream router.
- Phối hợp với ISP để blackhole route.

---

## 3. Tầng Frontend & Ingestion

### 3.1. Wasm Hardware Attestation

```rust
// Rust compiled to Wasm
#[wasm_bindgen]
pub fn attest() -> Attestation {
    let canvas_hash = get_canvas_hash();
    let webgl_renderer = get_webgl_renderer();
    let audio_context = get_audio_fingerprint();
    let has_mouse_movement = check_mouse_entropy();
    
    // Real Chrome on real OS
    if webgl_renderer.contains("swiftshader") {
        return Attestation::bot("headless_chromium");
    }
    if !has_mouse_movement {
        return Attestation::bot("no_human_interaction");
    }
    
    Attestation::human(AttestationProof {
        canvas_hash,
        webgl_renderer,
        timestamp: get_timestamp(),
        nonce: get_session_nonce(),
    })
}
```

### 3.2. Argon2 Proof-of-Work
```go
// Server-side
func IssuePoWChallenge(ip string) Challenge {
    nonce := uuid.NewV7().String()
    salt := randomBytes(16)
    
    return Challenge{
        Nonce:  nonce,
        Salt:   base64(salt),
        Difficulty: 3,  // Argon2id iterations
        Algorithm: "Argon2id",
    }
}

// Client-side compute (browser)
async function solvePoW(challenge) {
    const start = performance.now();
    const result = await argon2.hash({
        pass: challenge.nonce,
        salt: challenge.salt,
        time: challenge.difficulty * 4,  // ~20ms
        mem: 65536,
        parallelism: 1,
        type: argon2.ArgonType.Argon2id,
    });
    return {
        nonce: challenge.nonce,
        proof: result.hash,
        timeMs: performance.now() - start,
    };
}
```

### 3.3. Anti-Bot Score
- Composite score: Wasm attestation + PoW time + Fingerprint + Behavioral.
- Threshold: < 0.5 → Block.
- 0.5-0.7 → CAPTCHA.
- > 0.7 → Allow.

### 3.4. Cookie Consent & GDPR/PDPA
- Explicit opt-in (không pre-ticked).
- Granular consent (analytics, marketing, etc.).
- Easy withdrawal.
- Data export & delete (right to be forgotten).

---

## 4. Tầng Application – PASETO & RBAC

### 4.1. PASETO v4 (thay JWT)
- **PASETO v4.local:** Authenticated symmetric encryption.
- **PASETO v4.public:** Asymmetric signing.

```go
import "github.com/o1egl/paseto"

var pasetoKey = paseto.NewV4SymmetricKey()

func GenerateToken(claims Claims) string {
    token := paseto.New()
    return token.Encrypt(pasetoKey, claims, nil)
}

func VerifyToken(tokenStr string) (Claims, error) {
    var claims Claims
    token := paseto.New()
    err := token.Decrypt(tokenStr, pasetoKey, &claims, nil)
    return claims, err
}
```

#### Token Types
- **Auth Token:** User session (24h).
- **API Token:** Service-to-service (longer).
- **Refresh Token:** Renew auth (30d).
- **Invitation Token:** Mời user (7d).
- **Impersonation Token:** Admin impersonate (24h).
- **FIDO Challenge Token:** WebAuthn (5 min).

### 4.2. RBAC System

#### Permission Format
```
{resource}.{action}[.{scope}]

Ví dụ:
    lead.create
    lead.read.subtree
    lead.update.own
    deal.delete
    user.invite
    report.export
    admin.tenant.delete
```

#### Role Definition
```sql
CREATE TABLE roles (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  permissions TEXT[] NOT NULL,
  inherits_from TEXT REFERENCES roles(code),
  is_system BOOLEAN DEFAULT false
);
```

#### Permission Check
```go
func HasPermission(user *User, resource, action, scope string) bool {
    // 1. Direct permission on role
    for _, p := range user.Role.Permissions {
        if matchPermission(p, fmt.Sprintf("%s.%s.%s", resource, action, scope)) {
            return true
        }
        if matchPermission(p, fmt.Sprintf("%s.%s", resource, action)) {
            return true
        }
    }
    return false
}
```

### 4.3. Tenant Context Middleware
```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            // Resolve từ Host header
            tenantID = resolveTenantFromHost(r.Host)
        }
        
        // Validate user có trong tenant
        user := getUserFromContext(r.Context())
        if user.TenantID != tenantID && !isSuperAdmin(user) {
            http.Error(w, "Forbidden", 403)
            return
        }
        
        ctx := r.Context()
        ctx = setTenantID(ctx, tenantID)
        ctx = setUserID(ctx, user.ID)
        
        // Set DB session variable for RLS
        db.Exec("SET LOCAL app.current_tenant_id = ?", tenantID)
        db.Exec("SET LOCAL app.current_user_id = ?", user.ID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 4.4. Rate Limiting (per tenant)
```go
// Valkey token bucket per tenant
func RateLimit(tenantID string, action string) bool {
    key := fmt.Sprintf("ratelimit:%s:%s", tenantID, action)
    result, _ := valkey.Eval(`
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local current = redis.call('INCR', key)
        if current == 1 then
            redis.call('EXPIRE', key, window)
        end
        if current > limit then
            return 0
        end
        return 1
    `, []string{key}, limit, window).Result()
    return result.(int64) == 1
}

// Usage
if !RateLimit(tenantID, "api_call") {
    return 429 Too Many Requests
}
```

---

## 5. Tầng Database – RLS

### 5.1. PostgreSQL Row-Level Security

```sql
-- Enable RLS
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

-- Policy: tenant isolation
CREATE POLICY leads_tenant_isolation ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

-- Policy: user scope (own or subtree)
CREATE POLICY leads_user_scope ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM users 
      WHERE path <@ (
        SELECT path FROM users 
        WHERE id = current_setting('app.current_user_id', true)::UUID
      )
    )
    OR current_setting('app.is_super_admin', true) = 'true'
  );
```

### 5.2. Connection Pool Setup
```go
// PgBouncer-style middleware
func SetTenantContext(db *sql.DB, tenantID, userID string) error {
    tx, err := db.Begin()
    if err != nil { return err }
    defer tx.Rollback()
    
    _, err = tx.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)
    if err != nil { return err }
    
    _, err = tx.Exec("SET LOCAL app.current_user_id = $1", userID)
    if err != nil { return err }
    
    return tx.Commit()
}
```

### 5.3. Testing RLS
```sql
-- Verify RLS bằng cách switch tenant
SET app.current_tenant_id = 'tenant_a';
SET app.current_user_id = 'user_in_a';
SELECT count(*) FROM leads;  -- Chỉ thấy leads của A

SET app.current_tenant_id = 'tenant_b';
SET app.current_user_id = 'user_in_b';
SELECT count(*) FROM leads;  -- Chỉ thấy leads của B, KHÔNG thấy A
```

### 5.4. NoSQL Isolation (ScyllaDB)
- Partition key bao gồm `tenant_id`.
- Query luôn có `WHERE tenant_id = ?`.

```cql
CREATE TABLE rinco.messages (
  channel_id text,
  tenant_id text,    -- Partition key prefix
  event_time bigint,
  ...
  PRIMARY KEY ((channel_id, tenant_id), event_time)
);

-- Mọi query phải có tenant_id
SELECT * FROM messages WHERE channel_id = ? AND tenant_id = ?;
```

### 5.5. ClickHouse Isolation
- User riêng cho mỗi tenant với quyền filter theo `tenant_id`.

```sql
CREATE USER tenant_apex IDENTIFIED BY 'xxx';
GRANT SELECT ON rinco.analytics TO tenant_apex;
-- Row policy tự động filter
CREATE ROW POLICY tenant_apex_filter ON rinco.analytics
  USING tenant_id = 'apex' TO tenant_apex;
```

---

## 6. Dark Admin

### 6.1. Khái niệm
- Admin Portal **không có DNS record** (subdomain admin không tồn tại).
- **Không có IP public** – chỉ bind trên WireGuard interface.
- Admin phải qua VPN + SPA + Auth mới truy cập được.

### 6.2. WireGuard + SPA

```
[Admin] ──► [SPA Tool: sends 1 packet]
              │
              ▼
[WireGuard Server] ──► verify packet signature
              │ (valid)
              ▼
[Open ephemeral UDP port] ──► return to SPA tool
              │
              ▼
[Admin: open WireGuard tunnel to that port]
              │
              ▼
[Admin Gateway :8891 (private)]
              │
              ▼
[Auth: PASETO + FIDO2]
```

#### SPA Implementation
```go
// SPA Tool (Client)
func Authenticate(serverPubKey []byte, sharedSecret []byte) error {
    // 1. Compute HMAC of timestamp with shared secret
    ts := time.Now().Unix()
    sig := hmac.New(sha256.New, sharedSecret)
    sig.Write([]byte(strconv.FormatInt(ts, 10)))
    signature := sig.Sum(nil)
    
    // 2. Send UDP packet to server
    packet := append([]byte{0x01}, signature...)
    packet = append(packet, []byte(strconv.FormatInt(ts, 10))...)
    
    _, err := net.DialUDP("udp", nil, &net.UDPAddr{
        IP:   net.ParseIP("wireguard.rinco.app"),
        Port: 51820,
    })
    if err != nil { return err }
    
    conn.Write(packet)
    
    // 3. Server opens ephemeral port, returns via same UDP
    // 4. Client opens WireGuard tunnel
    return nil
}
```

#### WireGuard Server Config
```ini
[Interface]
PrivateKey = <server_private_key>
ListenPort = 51820

[Peer]
PublicKey = <admin_pub_key>
PresharedKey = <pre_shared_key>
AllowedIPs = 10.99.0.2/32
```

### 6.3. FIDO2 / YubiKey
- WebAuthn challenge.
- Required cho Admin.
- Backup key bắt buộc.

```go
import "github.com/go-webauthn/webauthn"

func (a *AdminAuth) VerifyYubiKey(session *WebAuthnSession, response *CredentialAssertionResponse) error {
    expectedChallenge := session.Challenge
    
    valid, err := webauthn.ValidateLogin(
        session.User,
        *session,
        expectedChallenge,
        response,
    )
    if !valid { return errors.New("invalid signature") }
    
    return nil
}
```

### 6.4. 2-of-3 Quorum
```go
type QuorumRequest struct {
    Action         string
    Payload        json.RawMessage
    InitiatorID    string
    RequiredSigs   int
    CollectedSigs  []Signature
    ExpiresAt      time.Time
}

func (q *QuorumRequest) AddSignature(adminID string, signature []byte) error {
    if q.IsExpired() {
        return errors.New("expired")
    }
    if len(q.CollectedSigs) >= q.RequiredSigs {
        return errors.New("already satisfied")
    }
    
    // Verify signature with admin's public key
    if !verifySignature(adminID, signature, q.Payload) {
        return errors.New("invalid signature")
    }
    
    q.CollectedSigs = append(q.CollectedSigs, Signature{
        AdminID:   adminID,
        Signature: signature,
        Timestamp: time.Now(),
    })
    
    if len(q.CollectedSigs) >= q.RequiredSigs {
        q.Execute()
    }
    return nil
}
```

---

## 7. Tầng Kernel – eBPF RASP

### 7.1. Runtime Application Self-Protection

```c
SEC("tracepoint/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx) {
    char filename[256];
    bpf_probe_read_user_str(filename, sizeof(filename), (char *)ctx->args[0]);
    
    // Block suspicious commands
    if (strstr(filename, "/bin/sh") || 
        strstr(filename, "/bin/bash") ||
        strstr(filename, "/usr/bin/curl")) {
        u64 pid_tgid = bpf_get_current_pid_tgid();
        bpf_printk("BLOCKED execve: %s from pid %d", filename, pid_tgid >> 32);
        
        // Kill process
        bpf_send_signal(SIGKILL);
        return -1;
    }
    return 0;
}

SEC("tracepoint/syscalls/sys_enter_ptrace")
int trace_ptrace(struct trace_event_raw_sys_enter *ctx) {
    u64 pid_tgid = bpf_get_current_pid_tgid();
    u32 target_pid = ctx->args[0];
    
    // Only allow ptrace from parent or tracer process
    u32 current_pid = pid_tgid >> 32;
    if (current_pid != target_pid && current_pid != 1) {
        bpf_send_signal(SIGKILL);
    }
    return 0;
}
```

### 7.2. File Integrity Monitoring
```c
SEC("tracepoint/syscalls/sys_enter_openat")
int trace_openat(struct trace_event_raw_sys_enter *ctx) {
    char filename[256];
    bpf_probe_read_user_str(filename, sizeof(filename), (char *)ctx->args[1]);
    
    // Block access to sensitive files
    if (strstr(filename, "/etc/shadow") ||
        strstr(filename, "/etc/passwd") ||
        strstr(filename, "/proc/self/mem")) {
        bpf_send_signal(SIGKILL);
    }
    return 0;
}
```

### 7.3. Network Anomaly Detection
```c
SEC("xdp")
int xdp_net_anomaly(struct xdp_md *ctx) {
    // Detect port scanning
    // Detect SYN flood
    // Detect UDP amplification
}
```

---

## 8. AI Security

### 8.1. Prompt Injection Defense

#### Isolation
- AI agents có quyền giới hạn (RBAC).
- AI đọc log → chỉ metadata, không đọc user content trực tiếp.
- AI chỉ output: text response, không được gọi destructive action.

#### Input Sanitization
```python
def sanitize_prompt(user_input: str) -> str:
    # Remove potential injection patterns
    patterns = [
        r"ignore\s+previous\s+instructions",
        r"you\s+are\s+now",
        r"system\s+prompt",
        r"disregard",
    ]
    for p in patterns:
        if re.search(p, user_input, re.IGNORECASE):
            log_security_event("prompt_injection_attempt", user_input)
            raise SecurityException("Invalid input")
    return user_input
```

#### Multi-Tenant Cache Isolation
```python
# Tenant-scoped cache key
def get_cache_key(tenant_id: str, prompt_prefix: str) -> str:
    return hashlib.sha256(
        f"{tenant_id}:{prompt_prefix}".encode()
    ).hexdigest()
```

### 8.2. AI Tenant Isolation
- AI inference pool theo tenant.
- KV-cache isolated per tenant.
- Token bucket admission control.

### 8.3. Adversarial Input Detection
- Detect unusual input patterns.
- Rate limit AI per tenant.
- Auto-suspend tenant nếu có attack pattern.

---

## 9. Danh sách tính năng (≥ 100)

### 9.1. Network Security (1-25)
1. eBPF/XDP DDoS filter.
2. eBPF/XDP rate limit.
3. eBPF/XDP JA4+ fingerprint.
4. Geo-blocking.
5. IP blacklist (auto-update).
6. IP whitelist.
7. BGP blackhole integration.
8. SYN flood protection.
9. UDP flood protection.
10. ICMP flood protection.
11. Slowloris protection.
12. TLS 1.3 only.
13. Strong cipher suite.
14. HSTS preload.
15. OCSP stapling.
16. Certificate pinning (mobile).
17. CSP header.
19. X-Frame-Options.
20. X-Content-Type-Options.
21. Referrer-Policy.
22. Permissions-Policy.
23. Subresource Integrity.
24. DNSSEC.
25. DNS over HTTPS (DoH).

### 9.2. Application Security (26-50)
26. PASETO v4 token.
27. WebAuthn/FIDO2.
28. TOTP 2FA.
29. Backup codes.
30. Session management.
31. CSRF protection.
32. XSS protection (CSP).
33. SQL Injection protection (parameterized).
34. Path traversal protection.
35. Command injection protection.
36. SSRF protection.
37. XXE protection.
38. Deserialization protection.
39. Rate limiting per IP.
40. Rate limiting per user.
41. Rate limiting per tenant.
42. Brute force protection.
43. Account lockout.
44. Password complexity rules.
45. Password breach check (HaveIBeenPwned).
46. Secure password storage (Argon2id).
47. Secure session cookie (httpOnly, secure, sameSite).
48. JWT/PASETO signing key rotation.
49. API key scoping.
50. OAuth 2.0 / OIDC support.

### 9.3. Multi-Tenant Security (51-75)
51. PostgreSQL RLS enabled.
52. RLS policy testing.
53. Tenant context middleware.
54. Connection pool per tenant.
55. Resource quota per tenant.
56. Cross-tenant query blocking.
57. Data leakage prevention.
58. Per-tenant encryption key.
59. Per-tenant audit log.
60. Tenant impersonation (audited).
61. Tenant data export.
62. Tenant data deletion (GDPR).
63. Tenant isolation test (chaos).
64. Tenant throttling.
65. Tenant circuit breaker.
66. Tenant DDoS protection.
67. Per-tenant Valkey namespace.
68. Per-tenant ScyllaDB keyspace.
69. Per-tenant MinIO bucket.
70. Cross-region replication (tenant opt-in).
71. Backup encryption (per-tenant).
72. Backup retention (per-tenant).
73. Encryption at rest.
74. Encryption in transit (TLS).
75. Bring Your Own Key (BYOK).

### 9.4. Admin Security (76-95)
76. Dark admin (no public).
77. WireGuard-only access.
78. Single Packet Authorization.
79. FIDO2 required.
80. YubiKey required.
81. 2-of-3 quorum for dangerous ops.
82. IP whitelist for admin.
83. Time-based access (chỉ trong giờ HC).
84. Action approval chain.
85. Audit log 5 năm.
86. Audit log search.
87. Audit log export.
88. Audit log monitoring.
89. Admin session timeout ngắn (30 phút).
90. Admin IP lock (chỉ IP whitelisted).
91. Admin device fingerprint.
92. Admin geolocation alert.
93. Admin activity anomaly.
94. Admin role separation.
95. Admin emergency lockout.

### 9.6. Kernel Security (96-110)
96. eBPF RASP enabled.
97. Syscall filtering.
98. File integrity monitoring.
99. Process whitelist.
100. Container seccomp profile.
101. AppArmor profile.
102. SELinux policy.
103. No-new-privileges.
104. Read-only root filesystem.
105. Drop ALL capabilities.
106. Network policy (Kubernetes).
107. Pod Security Standards.
108. Kernel version pinning.
109. Vulnerability scanning (Trivy).
110. CIS benchmark compliance.

### 9.7. AI Security (111-125)
111. Tenant-scoped AI cache key.
112. AI token bucket admission.
113. Prompt injection detection.
114. PII redaction in AI input.
115. AI output validation.
116. AI action authorization.
117. AI audit log.
118. AI rate limit per tenant.
119. AI model version control.
120. AI training data isolation.
121. AI rollback.
122. AI shadow mode (test before deploy).
123. Adversarial input detection.
124. AI emergency stop.
125. AI explainability log.

---

## 10. Audit & Compliance

### 10.1. Audit Log Schema
```sql
CREATE TABLE audit_log (
  id UUID PRIMARY KEY,
  tenant_id UUID,
  actor_type TEXT,        -- 'user','admin','system','api_key'
  actor_id UUID,
  actor_email TEXT,
  action TEXT,            -- 'lead.create','tenant.delete'
  resource_type TEXT,
  resource_id UUID,
  ip_address INET,
  user_agent TEXT,
  geo_country TEXT,
  geo_city TEXT,
  trace_id UUID,
  request_payload JSONB,
  response_status INT,
  success BOOLEAN,
  error_message TEXT,
  signature TEXT,         -- Chữ ký số nếu action quan trọng
  created_at TIMESTAMPTZ DEFAULT now()
) PARTITION BY RANGE (created_at);
```

### 10.2. Compliance
- **GDPR (EU):** Right to access, rectify, delete, portability.
- **PDPA (VN):** Consent, purpose limitation.
- **SOC 2 Type II:** Security controls audit.
- **ISO 27001:** Information security management.
- **HIPAA-ready (optional):** Healthcare data protection.

### 10.3. Data Retention
| Loại data | Retention | Sau đó |
|-----------|-----------|--------|
| Audit log | 5 năm | Archive S3 Glacier |
| Application log | 90 ngày | Delete |
| ClickHouse aggregated | 2 năm | Archive |
| PII data | Theo tenant config | Delete hoặc anonymize |
| Recordings | Theo tenant (30-365 ngày) | Delete |

---

## 11. Incident Response

### 11.1. Playbook
1. **Detect:** Alert fires.
2. **Triage:** AI SRE auto-analyze.
3. **Contain:** Circuit breaker, isolate.
4. **Eradicate:** Patch, restart.
5. **Recover:** Verify, restore.
6. **Learn:** Post-mortem, document.

### 11.2. Auto-Remediation
- Restart unhealthy pod (K3s).
- Block attacker IP (eBPF).
- Rollback deploy (ArgoCD).
- Scale up healthy nodes (HPA).
- Drain traffic from bad node.

### 11.3. War Room
- Slack channel auto-create cho P0.
- AI SRE bot join.
- Real-time trace/log shared.
- Decision logging.

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SEC-01 | RLS block 100% cross-tenant | Pen test |
| AC-SEC-02 | DDoS resistance 10 Gbps | Load test |
| AC-SEC-03 | Bot block 99% | Selenium test |
| AC-SEC-04 | Admin không có DNS | Nmap |
| AC-SEC-05 | Quorum 2-of-3 hoạt động | E2E test |
| AC-SEC-06 | FIDO2 only login | Test |
| AC-SEC-07 | eBPF block shell exec | Chaos test |

---

**Tiếp theo:** [`docs/10-database/README.md`](../10-database/README.md) – Database Schema chi tiết & Polyglot Persistence.