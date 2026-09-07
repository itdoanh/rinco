# Contributing to RINCO Platform

Cảm ơn bạn quan tâm đến việc đóng góp cho RINCO! Tài liệu này hướng dẫn cách contribute code, docs, hoặc báo lỗi.

> Bằng việc đóng góp, bạn đồng ý tuân thủ [Code of Conduct](CODE_OF_CONDUCT.md) và phát hành đóng góp dưới [MIT License](LICENSE).

---

## 1. Quick Links

- **Issues**: https://github.com/itdoanh/rinco/issues
- **Discussions**: https://github.com/itdoanh/rinco/discussions
- **Pull Requests**: https://github.com/itdoanh/rinco/pulls
- **Security Issues**: security@rinco.app (xem [SECURITY.md](SECURITY.md))

---

## 2. Workflow: Fork + Branch + PR

### 2.1 Fork repo

```bash
# Trên GitHub, click "Fork"
git clone https://github.com/<your-username>/rinco.git
cd rinco
git remote add upstream https://github.com/itdoanh/rinco.git
```

### 2.2 Tạo branch

```bash
git checkout -b <type>/<scope>-<short-desc>

# Ví dụ:
git checkout -b feat/auth-add-webauthn-challenge
git checkout -b fix/crm-tree-query-performance
git checkout -b docs/architecture-update
```

### 2.3 Commit (Conventional Commits)

Format: `<type>(<scope>): <description>`

**Types:**

| Type | Mục đích |
|------|----------|
| `feat` | New feature (minor version bump) |
| `fix` | Bug fix (patch version bump) |
| `docs` | Documentation only |
| `style` | Formatting, missing semi-colons (no code change) |
| `refactor` | Code change without feature/fix |
| `test` | Add/modify tests |
| `chore` | Build, CI, tooling |
| `perf` | Performance improvement |
| `revert` | Revert previous commit |

**Scope** (optional): `auth`, `tenant`, `crm`, `chat`, `webrtc`, `frontend`, `docs`, ...

**Examples:**
```bash
git commit -m "feat(auth): thêm WebAuthn registration flow"
git commit -m "fix(crm): sửa off-by-one khi tính lead score"
git commit -m "docs: cập nhật ARCHITECTURE.md với service mesh section"
git commit -m "test(chat): thêm property-based test cho signal protocol"
```

**Breaking changes** — thêm `!` sau type/scope và giải thích trong body:
```bash
git commit -m "feat(api)!: đổi /api/v1/users sang /api/v2/users

BREAKING CHANGE: API path đã thay đổi. Xem migration guide."
```

### 2.4 Push + tạo PR

```bash
git push origin <branch-name>
# Mở Pull Request trên GitHub → chọn base branch: main
```

---

## 3. PR Template

Khi tạo PR, điền vào template sau:

```markdown
## Mô tả

<!-- Mô tả ngắn gọn thay đổi. -->

## Loại thay đổi

- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (fix/feature ảnh hưởng API hiện tại)
- [ ] Documentation only

## Liên kết issue

<!-- Ví dụ: Closes #123 -->

## Cách test

<!-- Mô tả cách reviewer có thể verify. -->

## Checklist

- [ ] Code follow style guide của ngôn ngữ
- [ ] Tests pass locally (`make test`)
- [ ] Lint pass (`make lint`)
- [ ] Docs updated (nếu cần)
- [ ] CHANGELOG.md updated (cho `feat`/`fix`)
- [ ] Không commit secrets / `.env`
- [ ] PR title theo Conventional Commits
```

---

## 4. Testing Requirements

### 4.1 Unit tests

- Mỗi PR phải có test cho code mới.
- Coverage ≥ 80% cho mỗi service (CI sẽ check).
- Test phải **deterministic** — không flaky.

### 4.2 Integration tests

- Service mới cần ít nhất 1 integration test (qua Docker Compose stack).
- Test database phải dùng schema `*_test` riêng.

### 4.3 E2E tests

- UI changes cần Playwright E2E test.
- API breaking changes cần test script theo `scripts/test-lead-flow.ps1`.

### 4.4 Chạy tests

```bash
make test         # Tất cả
make go-test      # Go services
make rust-test    # Rust services
make python-test  # Python services
make e2e          # Playwright
```

---

## 5. Coding Style

| Language | Style | Auto-format |
|----------|-------|-------------|
| Go | `gofmt` + `go vet` + `golangci-lint` | `make lint` |
| Rust | `cargo fmt` + `cargo clippy -- -D warnings` | `make lint` |
| Python | `black` + `isort` + `ruff` + `mypy --strict` | `make lint` |
| TypeScript | `prettier` + `eslint` | `make lint` |
| SQL | lower-case keywords + 4-space indent | (manual) |

### 5.1 Đặt tên

- **Go**: PascalCase cho exported, camelCase cho unexported.
- **Rust**: snake_case functions, PascalCase types.
- **Python**: snake_case functions/variables, PascalCase classes.
- **TypeScript**: camelCase variables, PascalCase components.

### 5.2 Comments

- Mọi exported function cần doc comment.
- Explain **why**, không phải **what**.
- TODO comments kèm `// TODO(username): <description>`.

---

## 6. Documentation

- Mỗi feature mới cần update docs (README + relevant section trong `docs/`).
- Doc cùng ngôn ngữ với code (tiếng Anh preferred, tiếng Việt acceptable).
- Diagrams: dùng Mermaid syntax trong markdown (render tự động trên GitHub).

---

## 7. Review Process

1. **Automated checks** — CI chạy lint + test + security scan.
2. **Code review** — ít nhất 1 maintainer approve.
3. **Merge** — squash commit vào main, giữ Conventional Commit message.

Maintainers sẽ review trong vòng 2-5 ngày làm việc.

---

## 8. Báo lỗi

### 8.1 Bug reports

Mở issue với template:
- Mô tả bug
- Steps to reproduce
- Expected vs Actual
- Environment (OS, version, config)
- Logs / screenshots (nếu có)

### 8.2 Feature requests

Mở issue với template:
- Use case (vấn đề gì cần giải)
- Proposed solution
- Alternatives considered
- Impact (scope lớn nhỏ?)

### 8.3 Security issues

**KHÔNG** mở public issue. Gửi email **security@rinco.app** — xem [SECURITY.md](SECURITY.md).

---

## 9. Community

- **Discord**: (sắp ra mắt)
- **GitHub Discussions**: cho Q&A, ideas, show-and-tell
- **Monthly call**: maintainers sync mỗi tháng (notes công khai)

---

## 10. Recognition

Contributors được list trong [README.md](README.md) → Contributors section và trong release notes.

Maintainers cấp quyền **commit** cho contributors đáng tin cậy sau 3+ merged PRs.

---

Cảm ơn bạn đã contribute! 🎉
