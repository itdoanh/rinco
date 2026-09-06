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

---

# PHẦN MỞ RỘNG – Audit, Edge Cases, Code Examples, Roadmap

> Phần này bổ sung cho tài liệu gốc, cung cấp implementation chi tiết cho security stack.

---

## 8. Audit Report (Self-Audit)

### 8.1. Những gì đã đủ chi tiết ✓
- Zero-Trust 4 tầng network, frontend, application, database, admin.
- PostgreSQL RLS pattern.
- Dark Admin concept.

### 8.2. Cần bổ sung ⚠️
- Code examples cho mỗi tầng (eBPF, Wasm, PASETO, RLS).
- Penetration testing methodology.
- Cost estimation cho security infrastructure.
- Disaster recovery cho security incidents.

### 8.3. Mâu thuẫn nội bộ ❌
- FIDO2 requirement cho admin có thể conflict với UX nếu không có backup key. Cần rõ recovery flow.

---

## 9. Edge Cases & Error Scenarios (≥ 30)

### 9.1. eBPF/XDP Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 1 | eBPF program verify fail (kernel version mismatch) | Filter không load | Map program version → fallback userspace iptables |
| 2 | XDP_DROP quá aggressive | False positive | Per-IP whitelist, allow-list override |
| 3 | Map overflow | New IP không rate-limit | LRU eviction, monitoring |
| 4 | NIC không support XDP | Bypass filter | Detect + log, fallback |
| 5 | Kernel panic do eBPF bug | Server crash | Disable program, restart, alert |
| 6 | Race condition giữa userspace update và XDP lookup | Stale rule | Use BPF ring buffer for sync |

### 9.2. Wasm Attestation Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 7 | Wasm module load fail (CDN down) | Attestation fail | Local fallback + CAPTCHA |
| 8 | Browser không support Wasm (cũ) | Attestation fail | Polyfill, fallback JS check |
| 9 | Headless Chrome mới bypass Canvas check | False negative | Multiple signals (mouse, WebGL, audio) |
| 10 | Argon2 PoW timing attack | Bot tối ưu | Adaptive difficulty |
| 11 | Wasm compromised (CDN hacked) | Attestation leak | Subresource Integrity hash pin |

### 9.3. PASETO Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 12 | PASETO key rotation mid-session | Token invalid | Support overlap period (2 keys active) |
| 13 | Clock skew giữa issuer và verifier | Token "expired" | 60s clock skew tolerance |
| 14 | Token theft via XSS | Account compromise | Short TTL + httpOnly + CSP |
| 15 | Replay attack | Token reuse | jti + nonce store in Valkey |
| 16 | PASETO local key leak | All tokens compromised | Key rotation + audit + invalidate all |
| 17 | Brute force PASETO verification | DoS | Rate limit per IP |

### 9.4. PostgreSQL RLS Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 18 | RLS bypass via SQL injection | Tenant leak | sqlc parameterized queries, lint |
| 19 | RLS policy too strict | Own data inaccessible | Test matrix, dry-run migration |
| 20 | Missing SET LOCAL | RLS fail silently | Middleware enforce, integration test |
| 21 | Super admin flag leak | Cross-tenant read | Audit + 2FA for super admin |
| 22 | Connection pool reuses session | Wrong tenant context | PgBouncer transaction mode + SET LOCAL |
| 23 | RLS policy conflict giữa policies | Inconsistent result | Review all, integration test |
| 24 | Bypass RLS via superuser | Privilege escalation | FORCE ROW LEVEL SECURITY |
| 25 | LTREE path injection | Path traversal | Validate input format |

### 9.5. Dark Admin Edge Cases
| # | Scenario | Triệu chứng | Xử lý |
|---|----------|------------|-------|
| 26 | SPA packet spoofed | Admin port exposed | HMAC + nonce + replay window |
| 27 | WireGuard key lost | Cannot connect admin | Recovery key + offline QR backup |
| 28 | FIDO2 device lost | Cannot login | Backup YubiKey required at registration |
| 29 | FIDO2 challenge replay | Auth bypass | One-time challenge, 5 min TTL |
| 30 | Quorum member unavailable | Cannot approve action | Time-extension, designated alternate |
| 31 | 2-of-3 quorum compromised | Malicious action | Audit log + anomaly detection + manual review |
| 32 | Admin session stolen | Privilege escalation | IP lock + device fingerprint |

---

## 10. Code Examples chi tiết

### 10.1. C – eBPF/XDP DDoS Filter (full)
```c
// xdp_antiddos.bpf.c
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>

struct rate_limit_key {
    __u32 src_ip;
};

struct rate_limit_val {
    __u64 tokens;
    __u64 last_refill_ns;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, struct rate_limit_key);
    __type(value, struct rate_limit_val);
} rate_limit_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);              // IP
    __type(value, __u64);             // block_until_ns
} blacklist_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u64);
} config_map SEC(".maps");  // 0: tokens_per_sec, 1: burst

static __always_inline __u64 now_ns(void) {
    return bpf_ktime_get_ns();
}

static __always_inline int check_rate(struct rate_limit_key *k, __u64 rate, __u64 burst) {
    struct rate_limit_val *v = bpf_map_lookup_elem(&rate_limit_map, k);
    __u64 now = now_ns();

    if (!v) {
        struct rate_limit_val init = { .tokens = burst, .last_refill_ns = now };
        bpf_map_update_elem(&rate_limit_map, k, &init, BPF_ANY);
        return 1;
    }

    __u64 elapsed = now - v->last_refill_ns;
    __u64 refill = (elapsed * rate) / 1000000000ULL;
    v->tokens = (v->tokens + refill > burst) ? burst : v->tokens + refill;
    v->last_refill_ns = now;

    if (v->tokens == 0) return 0;
    v->tokens--;
    return 1;
}

SEC("xdp")
int xdp_antiddos(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;

    if (eth->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) return XDP_PASS;

    __u32 saddr = ip->saddr;

    // 1. Blacklist
    __u64 *blocked = bpf_map_lookup_elem(&blacklist_map, &saddr);
    if (blocked && *blocked > now_ns()) return XDP_DROP;

    // 2. Get config (rate, burst)
    __u32 zero = 0;
    __u64 *rate = bpf_map_lookup_elem(&config_map, &zero);
    __u64 rps = rate ? *rate : 100;        // default 100 req/s
    __u64 burst = rps * 2;

    // 3. Rate limit
    struct rate_limit_key k = { .src_ip = saddr };
    if (!check_rate(&k, rps, burst)) return XDP_DROP;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

### 10.2. C – eBPF RASP (syscall filter)
```c
// rasp_syscall.bpf.c
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Block suspicious execve
SEC("tracepoint/syscalls/sys_enter_execve")
int block_execve(struct trace_event_raw_sys_enter *ctx) {
    char filename[128];
    bpf_probe_read_user_str(filename, sizeof(filename), (char *)ctx->args[0]);

    // Whitelist safe binaries
    if (
        filename[0] == '/' &&
        filename[1] == 'u' &&
        filename[2] == 's' &&
        filename[3] == 'r' &&
        filename[4] == '/'
    ) return 0;

    // Allow /proc/self/exe (the binary itself)
    // ...

    // Block /bin/sh, /bin/bash, reverse shells
    char blocklist[][16] = {
        "/bin/sh\0",
        "/bin/bash\0",
        "/bin/zsh\0",
        "/usr/bin/curl\0",
        "/usr/bin/wget\0",
        "/usr/bin/nc\0",
        "/usr/bin/python\0",  // potential reverse shell
    };

    for (int i = 0; i < sizeof(blocklist)/sizeof(blocklist[0]); i++) {
        bool match = true;
        for (int j = 0; j < 16 && blocklist[i][j]; j++) {
            if (j >= sizeof(filename)) break;
            if (filename[j] != blocklist[i][j]) { match = false; break; }
        }
        if (match) {
            u64 pid_tgid = bpf_get_current_pid_tgid();
            u32 pid = pid_tgid >> 32;
            bpf_printk("RASP: blocked %s from pid %d", filename, pid);
            bpf_send_signal(SIGKILL);
            return -1;
        }
    }
    return 0;
}

// Block access to sensitive files
SEC("tracepoint/syscalls/sys_enter_openat")
int block_file_access(struct trace_event_raw_sys_enter *ctx) {
    char filename[128];
    bpf_probe_read_user_str(filename, sizeof(filename), (char *)ctx->args[1]);

    // Sensitive file patterns
    if (
        // /etc/shadow, /etc/passwd, /root/.ssh/...
        true  // simplified; use actual pattern matching
    ) {
        bpf_send_signal(SIGKILL);
    }
    return 0;
}

char _license[] SEC(".rodata") = "GPL";
```

### 10.3. Rust – Wasm Attestation (full)
```rust
// attestation/src/lib.rs
use wasm_bindgen::prelude::*;
use serde::{Serialize, Deserialize};
use sha2::{Digest, Sha256};

#[derive(Serialize, Deserialize, Debug)]
pub struct AttestationProof {
    pub canvas_hash: String,
    pub webgl_renderer: String,
    pub audio_fingerprint: String,
    pub has_real_mouse: bool,
    pub has_real_touch: bool,
    pub timestamp: u64,
    pub nonce: String,
    pub user_agent_hash: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub enum AttestationResult {
    Human(AttestationProof),
    Bot(String),
}

#[wasm_bindgen]
pub fn attest(nonce: JsValue) -> JsValue {
    let nonce_str: String = nonce.as_string().unwrap_or_default();
    let result = attest_internal(&nonce_str);
    serde_wasm_bindgen::to_value(&result).unwrap()
}

fn attest_internal(nonce: &str) -> AttestationResult {
    let canvas_hash = get_canvas_hash();
    let webgl_renderer = get_webgl_renderer();
    let audio_fp = get_audio_fingerprint();
    let ua = get_user_agent();

    // 1. Detect headless Chrome
    if webgl_renderer.contains("swiftshader") ||
       webgl_renderer.contains("llvmpipe") {
        return AttestationResult::Bot("headless_chromium".into());
    }

    // 2. Detect Selenium/Puppeteer
    if webgl_renderer.is_empty() || webgl_renderer == "WebGL" {
        return AttestationResult::Bot("no_webgl".into());
    }

    // 3. Audio context fingerprint
    if audio_fp.is_empty() {
        return AttestationResult::Bot("no_audio".into());
    }

    // 4. Mouse movement entropy (assumed real if has mouse activity)
    let has_real_mouse = check_mouse_entropy();

    AttestationResult::Human(AttestationProof {
        canvas_hash,
        webgl_renderer,
        audio_fingerprint: audio_fp,
        has_real_mouse,
        has_real_touch: false,
        timestamp: get_timestamp(),
        nonce: nonce.to_string(),
        user_agent_hash: sha256_hex(ua.as_bytes()),
    })
}

fn get_canvas_hash() -> String {
    let window = web_sys::window().unwrap();
    let document = window.document().unwrap();
    let canvas = document.create_element("canvas").unwrap()
        .dyn_into::<web_sys::HtmlCanvasElement>().unwrap();
    canvas.set_width(280);
    canvas.set_height(60);
    let ctx = canvas.get_context("2d").unwrap().unwrap()
        .dyn_into::<web_sys::CanvasRenderingContext2d>().unwrap();
    ctx.set_font("16px Arial");
    ctx.fill_text("RINCO-attest-2026", 2.0, 30.0).unwrap();
    let data = canvas.to_data_url().unwrap();
    sha256_hex(data.as_bytes())
}

fn get_webgl_renderer() -> String {
    let window = web_sys::window().unwrap();
    let document = window.document().unwrap();
    let canvas = document.create_element("canvas").unwrap()
        .dyn_into::<web_sys::HtmlCanvasElement>().unwrap();
    let gl = canvas.get_context("webgl").unwrap().unwrap()
        .dyn_into::<web_sys::WebGlRenderingContext>().unwrap();
    let ext = gl.get_extension("WEBGL_debug_renderer_info").unwrap();
    let renderer = gl.get_parameter(
        ext.as_ref().unwrap(),
        web_sys::WebGlRenderingContext::UNMASKED_RENDERER_WEBGL
    ).unwrap();
    renderer.as_string().unwrap_or_default()
}

fn get_audio_fingerprint() -> String {
    // OfflineAudioContext fingerprint
    let window = web_sys::window().unwrap();
    let audio_ctx = web_sys::AudioContext::new().unwrap();
    let oscillator = audio_ctx.create_oscillator().unwrap();
    let analyser = audio_ctx.create_analyser().unwrap();
    oscillator.connect(&analyser).unwrap();
    let _ = analyser.get_byte_frequency_data(&mut web_sys::Uint8Array::new_with_length(analyser.frequency_bin_count()));
    let sum: u32 = (0..analyser.frequency_bin_count()).map(|i| i as u32).sum();
    sha256_hex(sum.to_string().as_bytes())
}

fn check_mouse_entropy() -> bool {
    // Implementation depends on event listener.
    // Stub: assume true (real validation done at server side via mouse events).
    true
}

fn get_timestamp() -> u64 {
    js_sys::Date::now() as u64
}

fn sha256_hex(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    let result = hasher.finalize();
    hex::encode(result)
}
```

### 10.4. Go – PASETO v4 + RBAC (full)
```go
// pkg/auth/paseto.go
package auth

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/o1egl/paseto"
    "golang.org/x/crypto/chacha20poly1305"
)

var (
    ErrExpired      = errors.New("token expired")
    ErrInvalid      = errors.New("invalid token")
    ErrReplay       = errors.New("token replay detected")
)

type Claims struct {
    Issuer     string    `json:"iss"`
    Subject    string    `json:"sub"`
    Audience   string    `json:"aud"`
    ExpiresAt  time.Time `json:"exp"`
    IssuedAt   time.Time `json:"iat"`
    NotBefore  time.Time `json:"nbf"`
    JTI        string    `json:"jti"`
    TenantID   string    `json:"tenant_id,omitempty"`
    UserID     string    `json:"user_id,omitempty"`
    Roles      []string  `json:"roles,omitempty"`
    Permissions []string `json:"perms,omitempty"`
    Scope      string    `json:"scope,omitempty"`
}

type Paseto struct {
    key       []byte
    validator *Validator
}

func NewPaseto(hexKey string) (*Paseto, error) {
    key, err := hex.DecodeString(hexKey)
    if err != nil {
        return nil, err
    }
    if len(key) != chacha20poly1305.KeySize {
        return nil, errors.New("invalid key size")
    }
    return &Paseto{
        key: key,
        validator: NewValidator(),
    }, nil
}

func (p *Paseto) Encrypt(claims Claims) (string, error) {
    pasetoObj := paseto.NewV4Local()
    return pasetoObj.Encrypt(p.key, claims, nil)
}

func (p *Paseto) Decrypt(token string) (*Claims, error) {
    pasetoObj := paseto.NewV4Local()
    var claims Claims
    if err := pasetoObj.Decrypt(token, p.key, &claims, nil); err != nil {
        return nil, ErrInvalid
    }
    if err := p.validator.Validate(&claims); err != nil {
        return nil, err
    }
    return &claims, nil
}

type Validator struct {
    nowFunc func() time.Time
    skew    time.Duration
}

func NewValidator() *Validator {
    return &Validator{
        nowFunc: time.Now,
        skew:    60 * time.Second,
    }
}

func (v *Validator) Validate(c *Claims) error {
    now := v.nowFunc()
    if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt.Add(v.skew)) {
        return ErrExpired
    }
    if !c.NotBefore.IsZero() && now.Before(c.NotBefore.Add(-v.skew)) {
        return ErrInvalid
    }
    return nil
}

// RotateKey xoay vòng key (overlap period)
type KeyRing struct {
    current  []byte
    previous []byte
}

func (k *KeyRing) Encrypt(c Claims) (string, error) {
    p := paseto.NewV4Local()
    return p.Encrypt(k.current, c, nil)
}

func (k *KeyRing) Decrypt(token string) (*Claims, error) {
    p := paseto.NewV4Local()
    var c Claims
    if err := p.Decrypt(token, k.current, &c, nil); err == nil {
        return &c, nil
    }
    if err := p.Decrypt(token, k.previous, &c, nil); err == nil {
        return &c, nil
    }
    return nil, ErrInvalid
}
```

### 10.5. Go – PostgreSQL RLS Setup
```go
// pkg/db/rls.go
package db

import (
    "context"
    "database/sql"
    "fmt"
)

type ctxKey string

const (
    TenantIDKey ctxKey = "tenant_id"
    UserIDKey   ctxKey = "user_id"
    IsAdminKey  ctxKey = "is_super_admin"
)

// WithRLS thiết lập session variables cho RLS trong transaction.
func WithRLS(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }

    tenantID, _ := ctx.Value(TenantIDKey).(string)
    userID, _ := ctx.Value(UserIDKey).(string)
    isAdmin, _ := ctx.Value(IsAdminKey).(bool)

    if tenantID != "" {
        if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)); err != nil {
            tx.Rollback()
            return nil, fmt.Errorf("set tenant: %w", err)
        }
    }
    if userID != "" {
        if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", userID)); err != nil {
            tx.Rollback()
            return nil, fmt.Errorf("set user: %w", err)
        }
    }
    if isAdmin {
        if _, err := tx.ExecContext(ctx, "SET LOCAL app.is_super_admin = 'true'"); err != nil {
            tx.Rollback()
            return nil, fmt.Errorf("set admin: %w", err)
        }
    }

    return tx, nil
}

// SetTenantContext gắn tenant_id vào context.
func SetTenantContext(ctx context.Context, tenantID, userID string, isAdmin bool) context.Context {
    ctx = context.WithValue(ctx, TenantIDKey, tenantID)
    ctx = context.WithValue(ctx, UserIDKey, userID)
    if isAdmin {
        ctx = context.WithValue(ctx, IsAdminKey, true)
    }
    return ctx
}

// AuditContext ghi log mọi query
func AuditContext(ctx context.Context, action string) {
    // Implementation: ghi vào ScyllaDB audit_log
}
```

### 10.6. SQL – Tất cả RLS policies
```sql
-- 1. Tenants table (chỉ super admin)
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

CREATE POLICY tenants_admin ON tenants
  FOR ALL
  USING (current_setting('app.is_super_admin', true) = 'true')
  WITH CHECK (current_setting('app.is_super_admin', true) = 'true');

-- 2. Users table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY users_tenant ON users
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

CREATE POLICY users_subtree ON users
  FOR SELECT
  USING (
    owner_in_subtree(id)
  );

-- 3. Leads
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant ON leads
  FOR ALL
  USING (
    tenant_id = current_setting('app.current_tenant_id', true)::UUID
    OR current_setting('app.is_super_admin', true) = 'true'
  );

CREATE POLICY leads_user_scope ON leads
  FOR ALL
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

-- Helper function
CREATE OR REPLACE FUNCTION owner_in_subtree(uid UUID)
RETURNS BOOLEAN AS $$
DECLARE
    current_path LTREE;
    target_path LTREE;
BEGIN
    SELECT path INTO current_path FROM users
    WHERE id = current_setting('app.current_user_id', true)::UUID;
    SELECT path INTO target_path FROM users WHERE id = uid;

    IF current_path IS NULL OR target_path IS NULL THEN
        RETURN FALSE;
    END IF;

    RETURN target_path <@ current_path;
END;
$$ LANGUAGE plpgsql STABLE;
```

### 10.7. YAML – WireGuard + Headscale
```yaml
# docker-compose.wireguard.yml
version: '3.8'
services:
  headscale:
    image: headscale/headscale:latest
    command: headscale serve
    volumes:
      - ./headscale/config.yaml:/etc/headscale/config.yaml
      - headscale-data:/var/lib/headscale
    ports:
      - "8080:8080"
    restart: always

  coturn:
    image: coturn/coturn:latest
    network_mode: host
    command: >
      -n
      --realm=rinco.app
      --static-auth-secret=xxx
      --use-auth-secret
      --no-tls
      --no-dtls
      --listening-port=3478
      --min-port=49160
      --max-port=49200
      --fingerprint
      --no-multicast-peers
    restart: always

volumes:
  headscale-data:
```

```yaml
# WireGuard client config (admin)
[Interface]
PrivateKey = <admin_priv>
Address = 10.99.0.2/32
DNS = 10.99.0.1

[Peer]
PublicKey = <server_pub>
PresharedKey = <psk>
Endpoint = wireguard.rinco.app:51820
AllowedIPs = 10.99.0.0/24
PersistentKeepalive = 25
```

---

## 11. Implementation Roadmap

### Tuần 1-2: RLS + Tenant Context
- [ ] Apply RLS policies cho 10 tables quan trọng nhất.
- [ ] Implement `pkg/db/rls.go` Go.
- [ ] Middleware enforce SET LOCAL.
- [ ] Integration test cross-tenant.
- **Acceptance:** 100% cross-tenant attempt fail.

### Tuần 3-4: PASETO + RBAC
- [ ] Setup `pkg/auth/paseto.go`.
- [ ] KeyRing với rotation.
- [ ] RBAC engine với permission matrix.
- [ ] Migration từ JWT (nếu có).
- **Acceptance:** Auth flow end-to-end với PASETO, no JWT.

### Tuần 5-6: DDoS Protection (eBPF)
- [ ] Compile + load `xdp_antiddos.bpf.c`.
- [ ] Userspace controller (Go) để update map.
- [ ] Test với traffic generator.
- [ ] Integration với alerting.
- **Acceptance:** 1M pps drop không ảnh hưởng legitimate traffic.

### Tuần 7-8: Wasm + Argon2
- [ ] Compile Wasm attestation module.
- [ ] Integrate vào landing page.
- [ ] Server-side verification.
- [ ] Argon2 PoW server.
- **Acceptance:** 99% bot block, false positive < 1%.

### Tuần 9-10: Dark Admin + WireGuard
- [ ] Headscale deployment.
- [ ] SPA tool (Go client).
- [ ] WireGuard mesh giữa nodes.
- [ ] Admin gateway trên WireGuard interface only.
- **Acceptance:** Admin không accessible từ public IP (nmap clean).

### Tuần 11-12: FIDO2 + Quorum + Polish
- [ ] WebAuthn flow cho admin.
- [ ] YubiKey enrollment.
- [ ] 2-of-3 quorum service.
- [ ] Audit log + anomaly detection.
- **Acceptance:** Critical action yêu cầu 2 admin ký.

---

## 12. Disaster Recovery cho Security Incidents

### 12.1. PASETO Key Compromise
```
1. Rotate key ngay (overlap period).
2. Invalidate all active tokens.
3. Force re-login mọi user.
4. Audit log: xem token leak từ đâu.
5. Update secret store.
```

### 12.2. RLS Bypass Detection
```
1. SIEM alert "RLS policy not enforced".
2. Auto-disable service (circuit breaker).
3. Investigate qua audit log.
4. Patch + add test.
5. Post-mortem.
```

### 12.3. FIDO2 Device Lost
```
1. Admin đăng nhập bằng backup key (nếu có).
2. Nếu mất cả 2 → admin emergency contact.
3. Verify identity (video call, passport).
4. Generate new credentials.
5. Revoke old.
```

### 12.4. eBPF Program Crash
```
1. Auto-detect via health check.
2. Detach program, fallback userspace.
3. Investigate kernel version.
4. Fix, redeploy.
```

---

## 13. Cost Estimation

| Component | Spec | Cost/month (USD) |
|-----------|------|-----------------|
| Headscale + WireGuard | 2 nodes | $80 |
| Coturn TURN | 2 nodes, 1Gbps | $200 |
| FIDO2 YubiKey | 5 keys ($50 each, 3-year life) | ~$15 amortized |
| Penetration test annual | 3rd party | $15,000 one-time |
| DDoS appliance (Cloudflare Pro) | - | $240 |
| Secrets management (Vault) | Self-hosted | $80 |
| SIEM (Wazuh) | Self-hosted | $120 |
| **Total recurring** | | **~$735** |

---

## 14. Open Questions / Cần user xác nhận

1. **Self-host Sentry/GlitchTip hay SaaS?**
2. **BYOK (Bring Your Own Key) cho Enterprise tenants?**
3. **SAML SSO cho tenant SSO?**
4. **Cloudflare DDoS hay tự build eBPF?**
5. **Penetration test cadence: hàng quý?**
6. **Audit log retention: 5 năm hay 7 năm?**
7. **GDPR compliance certification?**
8. **SOC 2 Type II target year?**
9. **Backup encryption key escrow policy?**
10. **Disaster recovery RPO/RTO targets?**

---

## 15. Acceptance Criteria bổ sung

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SEC-08 | PASETO rotation không break session | E2E test |
| AC-SEC-09 | eBPF program verify pass | bpftool verify |
| AC-SEC-10 | Wasm attestation block 99% headless | Selenium grid |
| AC-SEC-11 | RLS policy test 100% tables | Coverage report |
| AC-SEC-12 | Audit log integrity (hash chain) | Verify |
| AC-SEC-13 | Penetration test no P0/P1 | Report |
| AC-SEC-14 | GDPR right-to-be-forgotten < 30 days | SLA |