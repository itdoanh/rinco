# Multi-Tenant Security & Zero-Trust — Tóm tắt

> Tài liệu tóm tắt `docs/09-security/README.md`. Triết lý cốt lõi: **"Zero-Trust — không tin tưởng bất kỳ request nào mặc định; mọi thứ phải verify"**. Mục tiêu: cách ly tuyệt đối dữ liệu giữa các tenant, chống DDoS, chống Bot, chống Prompt Injection, Admin vô hình với thế giới bên ngoài.

---

## 1. Nguyên tắc Zero-Trust

1. **Never trust, always verify** — không có request nào mặc định được chấp nhận.
2. **Assume breach** — luôn giả định attacker đã ở trong hệ thống.
3. **Least privilege** — mỗi actor chỉ có đúng quyền cần thiết.
4. **Defense in depth** — chồng nhiều lớp bảo vệ để một lớp lỡ hổng vẫn còn lớp khác.
5. **Continuous verification** — xác thực liên tục, không chỉ lúc đăng nhập.

**Mục tiêu SLA:**

| ID | Mục tiêu | Đo lường |
|----|---------|---------|
| SEC-1 | 0 cross-tenant data leak | Audit test 100% |
| SEC-2 | DDoS resistance 10 Gbps | Load test |
| SEC-3 | Bot block 99% | Headless browser test |
| SEC-4 | Admin không lộ public | Nmap scan |
| SEC-5 | RLS bypass impossible | Pen test |

---

## 2. Kiến trúc 6 tầng bảo mật

```
[Tầng 1] Network     — Anti-DDoS eBPF/XDP ở NIC driver
[Tầng 2] Frontend    — Wasm Attestation + Argon2 PoW
[Tầng 3] Application — PASETO v4, RBAC, Rate Limit
[Tầng 4] Database    — PostgreSQL RLS, ScyllaDB isolation
[Tầng 5] Dark Admin  — WireGuard + SPA + FIDO2 + 2-of-3 Quorum
[Tầng 6] Kernel      — eBPF RASP runtime self-protection
```

---

## 3. Tầng 1 — Network: eBPF/XDP Anti-DDoS

### 3.1. Kiến trúc

```
[Internet]
    │
    ▼
[NIC] ─── eBPF/XDP (filter ở driver level)
    │            ├─ JA4+ TLS Fingerprint
    │            ├─ Rate Limit Map (token bucket per-IP)
    │            ├─ Geo-block
    │            ├─ IP blacklist (auto-update)
    │            └─ SYN/UDP/ICMP flood detection
    │
    ▼ (pass)
[Linux Kernel TCP/IP Stack]
    │
    ▼
[io_uring Gateway]
```

Lọc ở **XDP hook** (trước khi packet vào kernel TCP/IP stack) cho phép drop hàng triệu packet/giây trên mỗi CPU core, không tốn syscall, không qua `iptables` userspace.

### 3.2. XDP Rate Limit Program (C)

```c
struct rate_limit_key { u32 src_ip; };
struct rate_limit_val { u64 tokens; u64 last_refill_ns; };

struct { __uint(type, BPF_MAP_TYPE_LRU_HASH); ... } rate_limit_map SEC(".maps");

SEC("xdp")
int xdp_antiddos(struct xdp_md *ctx) {
    // Parse ethernet + IP header
    // 1. Check blacklist → XDP_DROP
    // 2. Lookup token bucket theo src_ip
    // 3. Refill tokens theo elapsed_ns (100 req/s, burst 200)
    // 4. tokens == 0 ? XDP_DROP : tokens-- → XDP_PASS
}
```

- **Token bucket** per source IP: 100 req/s, burst 200, refill bằng nanosecond precision.
- **LRU hash map** 100K entries tự đẩy IP cũ ra khi đầy.
- **Config map** cho phép userspace Go controller update rate/burst real-time mà không cần recompile eBPF.

### 3.3. JA4+ TLS Fingerprint

Phân tích TLS Client Hello (version, cipher suites, extensions) → tính hash JA4+. So khớp với `known_bots_map` để phát hiện bot Python requests, Go net/http, curl mà không cần dựa vào IP.

### 3.4. BGP Blackhole (Auto)

Khi phát hiện IP tấn công kéo dài (volume lớn, nhiều IP), tự push route blackhole lên upstream router qua BGP session. Phối hợp với ISP để null-route tận nguồn, giảm tải cho hạ tầng RINCO.

### 3.5. Network Headers & Standards

- TLS 1.3 only, strong cipher suite, HSTS preload, OCSP stapling.
- Certificate pinning cho mobile app.
- CSP header, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy.
- Subresource Integrity (SRI) cho mọi script từ CDN.
- DNSSEC + DNS over HTTPS (DoH).

---

## 4. Tầng 2 — Frontend: Wasm Attestation + Argon2 PoW

### 4.1. Wasm Hardware Attestation

Module Rust compile sang WebAssembly chạy trên browser, thu thập nhiều tín hiệu phần cứng để chứng minh đây là trình duyệt thật trên OS thật, không phải headless bot:

```rust
#[wasm_bindgen]
pub fn attest() -> Attestation {
    let canvas_hash = get_canvas_hash();        // render text lên canvas, hash
    let webgl_renderer = get_webgl_renderer();  // "swiftshader"/"llvmpipe" = bot
    let audio_context = get_audio_fingerprint();
    let has_mouse_movement = check_mouse_entropy();

    if webgl_renderer.contains("swiftshader") { return Attestation::bot("headless_chromium"); }
    if !has_mouse_movement { return Attestation::bot("no_human_interaction"); }
    Attestation::human(AttestationProof { canvas_hash, webgl_renderer, timestamp, nonce })
}
```

Tín hiệu thu: canvas fingerprint, WebGL renderer string, audio context fingerprint, mouse entropy, touch support, user-agent hash. Subresource Integrity (SRI) pin Wasm module để chống CDN compromise.

### 4.2. Argon2id Proof-of-Work

Server issue challenge (nonce + salt + difficulty), browser compute Argon2id với cost ~20ms trước khi gửi request. Bot chạy hàng nghìn request/giây sẽ bị throttle vì tốn CPU.

```js
async function solvePoW(challenge) {
    const result = await argon2.hash({
        pass: challenge.nonce, salt: challenge.salt,
        time: challenge.difficulty * 4,  // ~20ms
        mem: 65536, parallelism: 1,
        type: argon2.ArgonType.Argon2id,
    });
    return { nonce: challenge.nonce, proof: result.hash, timeMs: ... };
}
```

### 4.3. Anti-Bot Score

Composite score = Wasm attestation + PoW timing + TLS fingerprint + behavioral (mouse, keystroke). Threshold: `< 0.5 → block`, `0.5–0.7 → CAPTCHA`, `> 0.7 → allow`.

### 4.4. Cookie Consent & GDPR/PDPA

- Explicit opt-in (không pre-ticked).
- Granular consent (analytics, marketing, etc.).
- Easy withdrawal, data export, right to be forgotten.

---

## 5. Tầng 3 — Application: PASETO v4 + RBAC

### 5.1. PASETO v4 Tokens (thay thế JWT)

PASETO (Platform-Agnostic SEcurity TOkens) khắc phục các lỗi thiết kế của JWT (alg=none, weak HMAC). RINCO dùng 2 variant:

- **PASETO v4.local** — ChaCha20-Poly1305 authenticated symmetric encryption (mặc định cho auth token).
- **PASETO v4.public** — Ed25519 asymmetric signing (cho service-to-service).

```go
var pasetoKey = paseto.NewV4SymmetricKey()

func GenerateToken(claims Claims) string {
    return paseto.New().Encrypt(pasetoKey, claims, nil)
}
```

6 loại token:
- **Auth Token** — User session (24h).
- **API Token** — Service-to-service (lâu hơn).
- **Refresh Token** — Renew auth (30d).
- **Invitation Token** — Mời user (7d).
- **Impersonation Token** — Admin impersonate user (24h, có audit log đặc biệt).
- **FIDO Challenge Token** — WebAuthn (5 phút, one-time).

**Key rotation** với overlap period (2 key cùng active trong thời gian rollover) để không break session. `KeyRing` thử `current` trước, fall back `previous` nếu fail.

### 5.2. RBAC (Role-Based Access Control)

Permission format: `{resource}.{action}[.{scope}]`

```
lead.create
lead.read.subtree
lead.update.own
deal.delete
user.invite
report.export
admin.tenant.delete
```

Role kế thừa (`inherits_from`), system role không thể xóa. Permission check match `lead.create`, `lead.create.own`, hoặc `lead.*`. Scope `own` (chỉ của mình), `subtree` (cả nhánh con trong user tree), `tenant` (toàn tenant), `*` (super admin).

### 5.3. Tenant Context Middleware

```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" { tenantID = resolveTenantFromHost(r.Host) }

        user := getUserFromContext(r.Context())
        if user.TenantID != tenantID && !isSuperAdmin(user) {
            http.Error(w, "Forbidden", 403); return
        }

        // Set DB session variable cho RLS
        db.Exec("SET LOCAL app.current_tenant_id = ?", tenantID)
        db.Exec("SET LOCAL app.current_user_id = ?", user.ID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Tenant được resolve từ header `X-Tenant-ID` hoặc subdomain. Khi có user rồi, set session var cho PostgreSQL RLS (xem Tầng 4).

### 5.4. Rate Limiting (multi-tier)

| Tier | Key | Mục đích |
|------|-----|---------|
| Per-IP | `ratelimit:ip:<ip>:<action>` | Chống DDoS |
| Per-user | `ratelimit:user:<user_id>:<action>` | Chống abuse |
| Per-tenant | `ratelimit:tenant:<tenant_id>:<action>` | Chia sẻ công bằng |

Dùng Valkey token bucket với Lua script atomic. Brute force protection (login fail 5 lần → lockout 15 phút), account lockout, password complexity, HaveIBeenPwned breach check, Argon2id password storage.

### 5.5. Other Application Hardening

- **CSRF protection**, **XSS** (CSP), **SQL injection** (parameterized queries qua sqlc), **Path traversal**, **Command injection**, **SSRF**, **XXE**, **Deserialization protection**.
- OAuth 2.0 / OIDC support, API key scoping, signing key rotation, secure cookie (httpOnly + secure + sameSite).

---

## 6. Tầng 4 — Database: Row-Level Security

### 6.1. PostgreSQL RLS

Mọi bảng chứa dữ liệu tenant đều `ENABLE ROW LEVEL SECURITY` + `FORCE ROW LEVEL SECURITY` (kể cả table owner cũng bị ép tuân thủ):

```sql
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
ALTER TABLE leads FORCE ROW LEVEL SECURITY;

CREATE POLICY leads_tenant_isolation ON leads
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY leads_user_scope ON leads
  USING (
    owner_user_id = current_setting('app.current_user_id', true)::UUID
    OR owner_user_id IN (
      SELECT id FROM users
      WHERE path <@ (SELECT path FROM users
                     WHERE id = current_setting('app.current_user_id', true)::UUID)
    )
    OR current_setting('app.is_super_admin', true) = 'true'
  );
```

User tree (LTREE) cho phép user thấy data của subtree dưới mình (cấp bậc sales/manager). Biến session `app.current_tenant_id`, `app.current_user_id`, `app.is_super_admin` được set qua `SET LOCAL` ở đầu mỗi transaction.

### 6.2. Multi-Database Isolation

| DB | Cơ chế isolation |
|----|------------------|
| **PostgreSQL** | RLS policy + `SET LOCAL` + `FORCE ROW LEVEL SECURITY` |
| **ScyllaDB** | Partition key luôn prefix `tenant_id`, mọi query kèm `WHERE tenant_id = ?` |
| **ClickHouse** | User riêng mỗi tenant + Row Policy `USING tenant_id = '...'` |
| **Valkey** | Key prefix `tenant:<id>:...` |
| **MinIO** | Bucket riêng mỗi tenant |

Per-tenant Valkey namespace, per-tenant ScyllaDB keyspace, per-tenant MinIO bucket, per-tenant encryption key (BYOK cho Enterprise).

### 6.3. RLS Testing

Switch tenant bằng `SET app.current_tenant_id = 'tenant_a'` rồi đếm row → verify chỉ thấy data của A. Integration test tự động quét 100% table.

---

## 7. Tầng 5 — Dark Admin (Admin vô hình)

### 7.1. Khái niệm

Admin Portal có:
- **Không DNS record** — subdomain admin không tồn tại trong DNS public.
- **Không IP public** — bind chỉ trên WireGuard interface.
- **Bắt buộc VPN + SPA + FIDO2** trước khi vào được.

Nmap scan trả về "host down". Không có cách nào biết admin đang chạy ở đâu.

### 7.2. Single Packet Authorization (SPA)

Thay vì mở port WireGuard 24/7, server chỉ mở port khi nhận được 1 UDP packet có chữ ký HMAC hợp lệ:

```
[Admin] ──► [SPA Tool: sends 1 signed UDP packet]
              │
              ▼
[WireGuard Server] ──► verify HMAC signature
              │ (valid)
              ▼
[Open ephemeral UDP port] ──► return to SPA tool
              │
              ▼
[Admin: open WireGuard tunnel to that ephemeral port]
              │
              ▼
[Admin Gateway :8891 (chỉ trên interface 10.99.0.x)]
              │
              ▼
[Auth: PASETO + FIDO2]
```

SPA packet chứa HMAC-SHA256(timestamp, shared_secret), replay window 30 giây. Sau khi verify, server mở 1 port ephemeral, return port number cho client, client mở WireGuard tunnel đến port đó. Không có port nào open mặc định → kẻ quét không biết tấn công vào đâu.

### 7.3. FIDO2 / YubiKey / WebAuthn

Mọi admin login bắt buộc WebAuthn (YubiKey hoặc platform authenticator):
- Server sinh challenge, browser/YubiKey ký bằng private key trong hardware.
- Server verify signature + challenge + counter (chống replay).
- Backup key bắt buộc đăng ký lúc onboarding (mất key chính dùng key phụ).

### 7.4. 2-of-3 Quorum Multi-Party Authorization

Mọi action nguy hiểm (xóa tenant, rotate key production, restore backup) yêu cầu **ít nhất 2 trong 3 admin** ký:

```go
type QuorumRequest struct {
    Action        string
    Payload       json.RawMessage
    InitiatorID   string
    RequiredSigs  int          // 2
    CollectedSigs []Signature
    ExpiresAt     time.Time
}

func (q *QuorumRequest) AddSignature(adminID string, signature []byte) error {
    if !verifySignature(adminID, signature, q.Payload) { return errors.New("invalid") }
    q.CollectedSigs = append(q.CollectedSigs, Signature{...})
    if len(q.CollectedSigs) >= q.RequiredSigs { q.Execute() }
    return nil
}
```

Mỗi admin ký bằng private key riêng (hardware-backed khi có thể). Payload + signature hash chain → audit log. Time-bound (5 phút), không thể replay.

### 7.5. Admin Hardening khác

- Admin session timeout ngắn (30 phút).
- IP whitelist (chỉ IP đã biết), device fingerprint, geolocation alert, activity anomaly detection.
- Time-based access (chỉ trong giờ hành chính).
- Admin role separation (không ai có full quyền).
- Emergency lockout (revoke tất cả admin session ngay).
- Audit log 5 năm, hash chain integrity, search/export.

---

## 8. Tầng 6 — Kernel: eBPF RASP

### 8.1. Runtime Application Self-Protection

eBPF program hook vào các syscall nguy hiểm, kill process nếu phát hiện hành vi bất thường:

```c
SEC("tracepoint/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx) {
    char filename[256];
    bpf_probe_read_user_str(filename, sizeof(filename), (char *)ctx->args[0]);

    // Block shell, reverse shell, curl pipe bash
    if (strstr(filename, "/bin/sh") ||
        strstr(filename, "/bin/bash") ||
        strstr(filename, "/usr/bin/curl")) {
        bpf_send_signal(SIGKILL);  // Kill process ngay lập tức
        return -1;
    }
    return 0;
}
```

### 8.2. File Integrity Monitoring

Hook `sys_enter_openat`: block access đến `/etc/shadow`, `/etc/passwd`, `/proc/self/mem`, `/root/.ssh/`. Mọi file write vào `/usr/bin/`, `/usr/sbin/` đều block. Hash whitelist cho binary hợp lệ.

### 8.3. Process Whitelist & Syscall Filtering

- Chỉ cho phép exec binary trong whitelist (`/usr/local/bin/rinco-*`).
- Block `ptrace` từ process lạ (chỉ parent hoặc PID 1 mới được trace).
- Seccomp profile giới hạn syscall cho mỗi container.

### 8.4. Network Anomaly Detection

XDP hook riêng detect:
- Port scanning (nhiều SYN đến nhiều port từ 1 IP).
- SYN flood (nhiều SYN không ACK).
- UDP amplification (DNS/NTP reflection).
- Outbound connection bất thường từ container (callback C2).

---

## 9. AI Security

### 9.1. Prompt Injection Defense

- **Isolation:** AI agents có RBAC riêng, scope giới hạn. AI chỉ đọc metadata, không đọc user content trực tiếp.
- **AI chỉ được output text response**, không gọi destructive action (xóa, modify, deploy).
- **Input sanitization:** regex loại bỏ pattern nguy hiểm (`ignore previous instructions`, `you are now`, `system prompt`, `disregard`). Nếu match → log security event `prompt_injection_attempt` + block.
- **Multi-tenant cache isolation:** cache key = `sha256(tenant_id + prompt_prefix)`, không bao giờ share cache giữa tenant.

### 9.2. AI Tenant Isolation & Limits

- AI inference pool per tenant.
- KV-cache isolated per tenant.
- Token bucket admission control (rate limit per tenant).
- Adversarial input detection, auto-suspend tenant nếu có attack pattern.

### 9.3. AI Audit & Explainability

Mọi AI call: model version, prompt hash, response hash, tenant_id, user_id đều vào audit log. AI shadow mode (test model mới trên traffic thật nhưng không ảnh hưởng user). AI rollback về version cũ khi drift. AI emergency stop (tắt toàn bộ AI bằng 1 flag).

---

## 10. Audit Log Schema (5 năm retention)

```sql
CREATE TABLE audit_log (
  id UUID PRIMARY KEY,
  tenant_id UUID,
  actor_type TEXT,        -- 'user','admin','system','api_key','ai_agent'
  actor_id UUID,
  actor_email TEXT,
  action TEXT,            -- 'lead.create','tenant.delete','paseto.rotate'
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
  signature TEXT,         -- Chữ ký số cho action quan trọng (tamper-evident)
  prev_hash TEXT,         -- Hash chain: mỗi record hash từ record trước
  created_at TIMESTAMPTZ DEFAULT now()
) PARTITION BY RANGE (created_at);
```

Hash chain (mỗi record chứa hash của record trước) đảm bảo tamper-evident: nếu attacker xóa/sửa 1 record, hash chain bị gãy, hệ thống phát hiện ngay.

---

## 11. Compliance & Data Retention

| Regulation | Yêu cầu chính |
|------------|----------------|
| **GDPR (EU)** | Right to access, rectify, delete, portability, consent |
| **PDPA (VN)** | Explicit consent, purpose limitation |
| **SOC 2 Type II** | Security controls audit hàng năm |
| **ISO 27001** | Information security management system |
| **HIPAA-ready** | Healthcare data protection (optional) |

| Data | Retention | Sau đó |
|------|-----------|--------|
| Audit log | 5 năm | Archive S3 Glacier |
| Application log | 90 ngày | Delete |
| ClickHouse aggregated | 2 năm | Archive |
| PII data | Theo tenant config | Delete hoặc anonymize |
| Recordings | 30–365 ngày (tenant chọn) | Delete |

---

## 12. Incident Response

### 12.1. Playbook

1. **Detect** — Alert fires (Sentry, Prometheus, SIEM).
2. **Triage** — AI SRE auto-analyze.
3. **Contain** — Circuit breaker, isolate, block IP (eBPF).
4. **Eradicate** — Patch, restart.
5. **Recover** — Verify, restore.
6. **Learn** — Post-mortem, document.

### 12.2. Auto-Remediation

- Restart unhealthy pod (K3s).
- Block attacker IP (eBPF XDP_DROP).
- Rollback deploy (ArgoCD).
- Scale up healthy nodes (HPA).
- Drain traffic từ node bad.

### 12.3. War Room

P0 alert → tự tạo Slack channel, AI SRE bot join, share trace/log real-time, decision logging tự động.

---

## 13. Cost Estimation (≈ $735 / tháng + pen-test)

| Component | Spec | USD/tháng |
|-----------|------|-----------|
| Headscale + WireGuard | 2 nodes | $80 |
| Coturn TURN | 2 nodes, 1 Gbps | $200 |
| FIDO2 YubiKey | 5 keys ($50/3 năm) | ~$15 amortized |
| Penetration test | 3rd party, annual | $15,000/năm |
| DDoS appliance (Cloudflare Pro) | - | $240 |
| Secrets management (Vault) | Self-hosted | $80 |
| SIEM (Wazuh) | Self-hosted | $120 |

---

## 14. Danh sách ≥ 25+ tính năng Security

### Network Security (1–25)
1. eBPF/XDP DDoS filter
2. eBPF/XDP rate limit
3. eBPF/XDP JA4+ fingerprint
4. Geo-blocking
5. IP blacklist (auto-update)
6. IP whitelist
7. BGP blackhole integration
8. SYN flood protection
9. UDP flood protection
10. ICMP flood protection
11. Slowloris protection
12. TLS 1.3 only
13. Strong cipher suite
14. HSTS preload
15. OCSP stapling
16. Certificate pinning (mobile)
17. CSP header
18. X-Frame-Options
19. X-Content-Type-Options
20. Referrer-Policy
21. Permissions-Policy
22. Subresource Integrity
23. DNSSEC
24. DNS over HTTPS (DoH)
25. XDP driver-level packet drop

### Application Security (26–50)
26. PASETO v4 token (thay JWT)
27. WebAuthn / FIDO2
28. TOTP 2FA
29. Backup codes
30. Session management
31. CSRF protection
32. XSS protection (CSP)
33. SQL Injection protection (parameterized)
34. Path traversal protection
35. Command injection protection
36. SSRF protection
37. XXE protection
38. Deserialization protection
39. Rate limit per IP
40. Rate limit per user
41. Rate limit per tenant
42. Brute force protection
43. Account lockout
44. Password complexity rules
45. Password breach check (HaveIBeenPwned)
46. Argon2id password storage
47. Secure session cookie (httpOnly + secure + sameSite)
48. PASETO key rotation (overlap)
49. API key scoping
50. OAuth 2.0 / OIDC support

### Multi-Tenant Security (51–75)
51. PostgreSQL RLS enabled
52. RLS policy testing (100% tables)
53. Tenant context middleware
54. Connection pool per tenant
55. Resource quota per tenant
56. Cross-tenant query blocking (RLS)
57. Data leakage prevention (DLP)
58. Per-tenant encryption key
59. Per-tenant audit log
60. Tenant impersonation (audited)
61. Tenant data export (GDPR)
62. Tenant data deletion (right to be forgotten)
63. Tenant isolation test (chaos)
64. Tenant throttling
65. Tenant circuit breaker
66. Tenant DDoS protection
67. Per-tenant Valkey namespace
68. Per-tenant ScyllaDB keyspace
69. Per-tenant MinIO bucket
70. Cross-region replication (opt-in)
71. Backup encryption (per-tenant)
72. Backup retention (per-tenant)
73. Encryption at rest
74. Encryption in transit (TLS)
75. Bring Your Own Key (BYOK)

### Admin Security (76–95)
76. Dark Admin (không có DNS, không IP public)
77. WireGuard-only access
78. Single Packet Authorization (SPA)
79. FIDO2 required
80. YubiKey required
81. 2-of-3 quorum cho dangerous ops
82. IP whitelist cho admin
83. Time-based access (chỉ giờ HC)
84. Action approval chain
85. Audit log 5 năm
86. Audit log search
87. Audit log export
88. Audit log monitoring (SIEM)
89. Admin session timeout ngắn (30 phút)
90. Admin IP lock (chỉ IP whitelisted)
91. Admin device fingerprint
92. Admin geolocation alert
93. Admin activity anomaly detection
94. Admin role separation
95. Admin emergency lockout

### Kernel Security (96–110)
96. eBPF RASP enabled
97. Syscall filtering
98. File integrity monitoring
99. Process whitelist (binary)
100. Container seccomp profile
101. AppArmor profile
102. SELinux policy
103. No-new-privileges
104. Read-only root filesystem
105. Drop ALL capabilities
106. Network policy (Kubernetes)
107. Pod Security Standards
108. Kernel version pinning
109. Vulnerability scanning (Trivy)
110. CIS benchmark compliance

### AI Security (111–125)
111. Tenant-scoped AI cache key
112. AI token bucket admission control
113. Prompt injection detection
114. PII redaction in AI input
115. AI output validation
116. AI action authorization (RBAC)
117. AI audit log
118. AI rate limit per tenant
119. AI model version control
120. AI training data isolation
121. AI rollback
122. AI shadow mode
123. Adversarial input detection
124. AI emergency stop
125. AI explainability log

---

## 15. Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-SEC-01 | RLS block 100% cross-tenant | Pen test |
| AC-SEC-02 | DDoS resistance 10 Gbps | Load test |
| AC-SEC-03 | Bot block 99% | Selenium test |
| AC-SEC-04 | Admin không có DNS | Nmap scan |
| AC-SEC-05 | Quorum 2-of-3 hoạt động | E2E test |
| AC-SEC-06 | FIDO2 only login | Test |
| AC-SEC-07 | eBPF block shell exec | Chaos test |
| AC-SEC-08 | PASETO rotation không break session | E2E test |
| AC-SEC-09 | eBPF program verify pass | bpftool verify |
| AC-SEC-10 | Wasm attestation block 99% headless | Selenium grid |
| AC-SEC-11 | RLS policy test 100% tables | Coverage report |
| AC-SEC-12 | Audit log integrity (hash chain) | Verify |
| AC-SEC-13 | Penetration test no P0/P1 | Report |
| AC-SEC-14 | GDPR right-to-be-forgotten < 30 days | SLA |

---

## 16. Disaster Recovery cho Security Incidents

**PASETO key compromise:** rotate ngay với overlap period → invalidate all tokens → force re-login → audit log review → update secret store. **RLS bypass detected:** SIEM alert → circuit breaker ngừng service → investigate qua audit log → patch + add test. **FIDO2 device lost:** backup key trước; mất cả 2 → video call verify → tạo credentials mới, revoke cũ. **eBPF program crash:** health check detect → detach → fallback userspace → fix kernel compatibility → redeploy.

---

**Tóm lại:** Security của RINCO là một hệ thống **Zero-Trust 6 tầng** (Network → Frontend → Application → Database → Dark Admin → Kernel) với triết lý "không tin ai, xác minh mọi thứ, defense in depth". Điểm khác biệt lớn: anti-DDoS chạy ở **eBPF/XDP ở NIC driver level** (không qua userspace), **Wasm attestation** + **Argon2 PoW** chống bot ở frontend, **PostgreSQL RLS với FORCE** đảm bảo không thể bypass, **Dark Admin** không thể scan được, **2-of-3 Quorum** cho mọi action nguy hiểm, và **eBPF RASP ở kernel** chống RCE ngay cả khi app bị hack. Toàn bộ kết hợp với AI Security (prompt injection defense, tenant cache isolation, AI emergency stop) để bảo vệ cả lớp AI mới.
