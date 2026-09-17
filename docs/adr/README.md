# ADR Index

Architecture Decision Records (ADR) — log mọi quyết định kiến trúc quan trọng trong RINCO Platform.

> Format theo [Michael Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions).

## Status legend

- 🟢 **Accepted** — đang áp dụng
- 🟡 **Proposed** — đang xem xét
- 🔴 **Superseded** — đã thay thế bằng ADR khác

---

## Danh sách

| # | Title | Status | Date | Critical? |
|---|-------|--------|------|-----------|
| [0001](0001-polyglot-persistence.md) | Polyglot Persistence | 🟢 Accepted | 2026-09-18 | — |
| [0002](0002-paseto-vs-jwt.md) | PASETO over JWT | 🟢 Accepted | 2026-09-18 | — |
| [0003](0003-nats-event-bus.md) | NATS JetStream as Event Bus | 🟢 Accepted | 2026-09-18 | — |
| [0004](0004-ltree-crm.md) | LTREE for CRM Hierarchy | 🟢 Accepted | 2026-09-18 | — |
| [0005](0005-rls-stickness-mitigation.md) | RLS Stickiness Mitigation | 🟢 Accepted | 2026-09-18 | 🔴 **Critical Issue #1** |
| [0006](0006-tiered-storage-minio.md) | Tiered Storage on MinIO | 🟢 Accepted | 2026-09-18 | — |
| [0007](0007-chat-engine-rust.md) | Chat Engine in Rust | 🟢 Accepted | 2026-09-18 | — |
| [0008](0008-frontend-multi-app-strategy.md) | Frontend Multi-app Strategy | 🟢 Accepted | 2026-09-18 | — |

---

## Critical issues

| ID | Title | ADR | Status |
|----|-------|-----|--------|
| **#1** | RLS Stickiness — cross-tenant data leak | [ADR-0005](0005-rls-stickness-mitigation.md) | Mitigated (5 layers) |
| **#2** | (planned) | TBD | — |
| **#3** | (planned) | TBD | — |

---

## Quy trình viết ADR mới

1. Copy template từ [template.md](template.md).
2. Đặt tên file `<NNNN>-<short-title>.md` (số thứ tự tiếp theo).
3. Điền `Status`, `Date`, `Authors`, `Context`, `Decision`, `Consequences`, `Alternatives Considered`, `References`.
4. Update table trên.
5. PR review bởi ít nhất 2 senior engineers.
6. Sau khi merge → thông báo trong Slack `#engineering`.

## Template

```markdown
# ADR-NNNN: <Title>

> **Status**: Proposed | Accepted | Superseded · **Date**: YYYY-MM-DD · **Authors**: ...

## Context
<Problem statement, forces at play>

## Consequences

### Positive
- ...

### Negative
- ...

### Mitigations
- ...

## Alternatives Considered

### A. <Alternative 1>
- **Pro**: ...
- **Con**: ...
- **Verdict**: ✅ Accepted | ❌ Rejected | 🟡 ...

## References
- ...
```