# Security Policy

Cảm ơn bạn đã quan tâm đến security của RINCO Platform. Tài liệu này mô tả cách báo cáo lỗ hổng, security policy, và supported versions.

---

## 1. Reporting a Vulnerability

**Xin vui lòng KHÔNG mở public GitHub issue cho security vulnerabilities.**

Gửi báo cáo đến: **security@rinco.app**

Bao gồm:
- Mô tả chi tiết lỗ hổng
- Steps to reproduce
- Potential impact (data leak? RCE? privilege escalation?)
- Affected versions / services
- (Optional) Proof-of-concept code

Bạn sẽ nhận được acknowledgment trong vòng **48 giờ**.

### 1.1 Disclosure timeline

| Phase | Thời gian |
|-------|-----------|
| Initial acknowledgment | 48 giờ |
| Triage & severity assessment | 5 ngày làm việc |
| Patch development (Critical/High) | 7-14 ngày |
| Coordinated disclosure (sau khi patch) | Public sau 30 ngày (hoặc theo thỏa thuận) |

### 1.2 Severity rating

| Severity | Ví dụ | Response time |
|----------|-------|---------------|
| **Critical** | RCE, auth bypass toàn hệ thống | 24 giờ patch |
| **High** | Data leak giữa tenants, SQLi, XSS persistent | 7 ngày patch |
| **Medium** | CSRF, limited info disclosure | 30 ngày patch |
| **Low** | Information leak không nhạy cảm | Best-effort |

---

## 2. Supported Versions

| Version | Supported | Notes |
|---------|-----------|-------|
| `main` branch | ✅ Active development | Luôn nhận security patches |
| `v1.x.x` (release tags) | ✅ Cho đến 6 tháng sau release | LTS |
| `v0.x.x` | ⚠️ Best-effort | Có thể yêu cầu upgrade |
| Older commits | ❌ Not supported | Phải nâng cấp |

Cập nhật thường xuyên — chạy trên latest stable release.

---

## 3. Security Architecture (tóm tắt)

### 3.1 Authentication & Authorization

- **PASETO v4** (preferred) hoặc JWT cho user sessions.
- **FIDO2/WebAuthn** cho passwordless login.
- **mTLS** cho service-to-service communication (K8s cert-manager).
- **API key + HMAC** cho 3rd-party integrations.
- **RBAC + ABAC** với PostgreSQL Row-Level Security (RLS) theo `tenant_id`.

### 3.2 Encryption

- **In transit**: TLS 1.3 (public), mTLS (internal).
- **At rest**: pgcrypto (PostgreSQL), SSE-KMS (MinIO), TDE (ScyllaDB).
- **E2EE**: chat-engine dùng Signal Protocol (X3DH + Double Ratchet).
- **Backups**: AES-256-GCM encrypted trước khi upload S3.

### 3.3 Network

- Public subnet (Traefik + Cloudflare) → Application subnet → Data subnet → Observability subnet.
- NetworkPolicy Kubernetes enforce strict ingress/egress.
- WireGuard mesh cho private cross-node communication.

### 3.4 Audit

- Mọi admin action logged to ClickHouse (immutable).
- Tamper-evident qua hash chain.
- Retention: 365 ngày cho audit log, 30 ngày cho app logs.

---

## 4. Security Best Practices cho Contributors

### 4.1 Code review checklist

Trước khi merge, reviewer check:
- [ ] Không commit secrets, `.env`, `*.pem`, `*.key`.
- [ ] User input validation (allowlist, không denylist).
- [ ] Parameterized queries (không string concatenation cho SQL/CQL).
- [ ] Output encoding cho HTML (chống XSS).
- [ ] CSRF protection cho state-changing endpoints.
- [ ] Rate-limit áp dụng cho public endpoints.
- [ ] Tenant isolation: mọi query filter `tenant_id`.
- [ ] PII fields đã được redact trong logs.
- [ ] Dependencies không có known CVE (`govulncheck` / `npm audit` / `pip-audit`).
- [ ] Không disable security features (CORS, CSP, HSTS) trong prod config.

### 4.2 Reporting security issues trong code

Nếu bạn phát hiện vấn đề security trong code:
1. **KHÔNG** mở public PR với fix ngay.
2. Email **security@rinco.app** để coordinate.
3. Maintainers sẽ quyết định disclosure timeline.

### 4.3 Dependencies

- Mọi dependency update phải pass `gosec` / `cargo audit` / `pip-audit` / `npm audit`.
- Critical CVE → patch trong 7 ngày.
- High CVE → patch trong 30 ngày.
- Auto-update Dependabot enabled cho `go.mod`, `Cargo.toml`, `package.json`, `requirements.txt`.

---

## 5. Security Tools

| Tool | Mục đích | CI |
|------|----------|-----|
| **Trivy** | Image + filesystem CVE scan | ✅ |
| **gosec** | Go static analysis | ✅ |
| **cargo-audit** | Rust dependency CVE check | ✅ |
| **bandit** | Python security lint | ✅ |
| **npm audit** | JS/TS dependency CVE | ✅ |
| **OWASP ZAP** | DAST (optional, weekly) | ⚠️ |
| **govulncheck** | Go vuln database | ✅ |

---

## 6. Bug Bounty

Hiện chưa có chương trình bug bounty chính thức. Maintainers sẽ ghi nhận contributors trong:

- GitHub contributors list
- Release notes (Security Credits section)
- Hall of Fame (sắp ra mắt)

---

## 7. Compliance

### 7.1 Data privacy

- **GDPR**: Right to erasure, data portability implemented trong `auth-service` + `crm-service`.
- **CCPA**: Opt-out tracking trong `landing-service`.
- **PDPA** (Vietnam): Data minimization + consent management.

### 7.2 Audit log retention

- App logs: 30 ngày (Loki).
- Audit logs: 365 ngày (ClickHouse).
- Backups: 90 ngày (S3 immutable bucket).

---

## 8. Contact

- **Security email**: security@rinco.app (PGP key: [link])
- **General**: dev@rinco.app
- **GitHub**: https://github.com/itdoanh/rinco/security/advisories

---

## 9. Acknowledgments

Cảm ơn các security researchers đã đóng góp:
<!-- Maintainers sẽ update list này sau mỗi coordinated disclosure -->

_(chưa có reported vulnerabilities)_
