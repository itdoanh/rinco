# ADR-0002: PASETO over JWT

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

RINCO cần authentication tokens cho:

1. **User session** (web app, mobile app) — lifetime 15 phút access + 7 ngày refresh.
2. **Service-to-service** (mTLS thay thế khi không có cert).
3. **API client** (third-party integration qua API key).
4. **Webhook** (inbound với HMAC signature).

Lựa chọn chính giữa:

- **JWT (JSON Web Token)**: phổ biến, nhiều library, dễ debug.
- **PASETO (Platform-Agnostic SEcurity TOkens)**: hiện đại hơn, ít footguns hơn.

## Decision

Dùng **PASETO v4.public** cho user session, kết hợp **FIDO2/WebAuthn** cho high-security actions.

### PASETO v4.public

- **Algorithm**: Ed25519 (signature), XChaCha20-Poly1305 (encryption optional).
- **Claims**: standard (`sub`, `iat`, `exp`, `aud`, `iss`) + custom (`tenant_id`, `role`, `device_id`).
- **Library**: `o1egl/paseto` (Go), custom Rust impl.

### So sánh PASETO vs JWT

| Khía cạnh | JWT | PASETO v4.public |
|-----------|-----|-------------------|
| **Algorithms** | Nhiều (HS256, RS256, ES256, none, …) | Chỉ 1 per version (v4 = Ed25519) |
| **`alg: none` attack** | ⚠️ Có (nếu dev quên whitelist) | ✅ Không tồn tại |
| **Header injection** | ⚠️ Có thể (jku, x5u, kid confusion) | ✅ Không có header field nguy hiểm |
| **JWK rotation** | ⚠️ Phức tạp (JWK Set) | ✅ Đơn giản (key version trong payload) |
| **Library quality** | Nhiều nhưng dễ viết sai | Ít nhưng đặc tả chặt |
| **Industry adoption** | 🟢 Phổ biến | 🟡 Mới nổi (v4 từ 2021) |
| **Cryptographic agility** | ✅ Có (nhiều algs) | ❌ Không — phiên bản cố định |

### Refresh strategy

```
┌──────────────────────────────────────────────────────────┐
│  Login → Access Token (15 min) + Refresh Token (7 days)  │
│                                                          │
│  Access: PASETO v4.public, payload = {                   │
│    sub, tenant_id, role, device_id, iat, exp, jti        │
│  }                                                       │
│                                                          │
│  Refresh: PASETO v4.local (encrypted), lưu trong        │
│  httpOnly secure cookie + DB mapping cho revoke          │
│                                                          │
│  Refresh flow:                                           │
│  1. Client gửi refresh token + device fingerprint       │
│  2. Server verify + rotate (cấp refresh mới, revoke cũ)  │
│  3. Cấp access token mới                                 │
└──────────────────────────────────────────────────────────┘
```

### FIDO2/WebAuthn (yubikey)

Cho high-security actions (Quorum signing, delete tenant, rotate master keys), yêu cầu **FIDO2/WebAuthn**:

- Server tạo challenge → user cắm YubiKey → touch → verify signature.
- Public key lưu trong PostgreSQL (`auth.fido_credentials`).
- Counter-based replay protection.

### Service-to-service auth

- **Internal cluster**: mTLS (K8s cert-manager, SPIFFE).
- **Cross-cluster**: short-lived PASETO v4.public (15 min) + service-account claim.

### API client (third-party)

- API key + HMAC-SHA256 signature.
- Không dùng JWT cho API client vì:
  - Không cần expiration (revoke ngay khi cần).
  - Không cần claims (chỉ cần "đúng key").

### Webhook inbound

- HMAC-SHA256 signature header + timestamp + replay protection window.

## Consequences

### Positive

- **Không có `alg: none`** → không bị attack cơ bản nhất.
- **Ít footguns** → dễ audit, dễ onboard dev mới.
- **Spec chặt** → nếu library sai → bug rõ ràng.
- **FIDO2 cho P0** → chống insider attack.

### Negative

- **Ít library hơn JWT** → một số edge case phải tự implement.
- **Industry adoption thấp** → khó debug khi integrate với bên thứ ba.
- **Migration khó** nếu sau này muốn đổi.
- **PASETO không có JWE natively** → encryption phải wrap bằng cách khác.

### Mitigations

- **Wrapper library** trong `packages/go/auth` và `packages/frontend/ui` để abstract.
- **Token introspection** endpoint cho debug.
- **Documentation** rõ ràng cho mỗi use case.

## Alternatives Considered

### A. JWT (HS256/RS256/ES256)

- **Pro**: phổ biến, nhiều tool, debug dễ.
- **Con**: `alg: none` attack, JKU header injection, library quality không đảm bảo.
- **Verdict**: ❌ Rejected — security footprint quá rộng.

### B. Opaque tokens (server-side session)

- **Pro**: dễ revoke, không có crypto risk ở client.
- **Con**: phải lookup DB mỗi request → latency cao hơn.
- **Verdict**: 🟡 Hybrid — dùng cho refresh token (qua `auth.refresh_tokens` table).

### C. Macaroons

- **Pro**: attenuation (giảm quyền), revocation-friendly.
- **Con**: ecosystem còn non trẻ.
- **Verdict**: ❌ Rejected — chưa cần attenuation phức tạp.

## References

- [PASETO spec](https://github.com/paseto-standard/paseto-spec)
- [auth-service README](../../services/auth-service/README.md)
- [packages/go/auth](../../packages/go/auth)
- [ADR-0001 Polyglot Persistence](0001-polyglot-persistence.md)
- [ADR-0008 Frontend Multi-app](0008-frontend-multi-app-strategy.md)