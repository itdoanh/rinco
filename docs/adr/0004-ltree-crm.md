# ADR-0004: LTREE for CRM Hierarchy

> **Status**: Accepted · **Date**: 2026-09-18 · **Authors**: RINCO Platform Team

## Context

CRM trong RINCO có **cây tổ chức** (organizational tree) phân cấp:

```
Tenant (root)
├── Manager (Miền Bắc)
│   ├── Team Lead (Hà Nội)
│   │   ├── Agent 1
│   │   └── Agent 2
│   └── Team Lead (Hải Phòng)
│       └── Agent 3
└── Manager (Miền Nam)
    └── Team Lead (HCM)
        └── Agent 4
```

Mỗi node có thể là User, hoặc có thể là **node trung gian** (phòng ban / chi nhánh).

Yêu cầu:

1. **Lưu trữ cây vô hạn cấp** — depth không cố định.
2. **Query subtree hiệu quả** — "tất cả Agent dưới Manager X" phải nhanh.
3. **Move subtree** — chuyển nhánh mà không phải update nhiều.
4. **Cycle prevention** — không thể move parent vào con của nó.
5. **Path uniqueness** — path trong cây phải unique.
6. **Multi-tenant isolation** — mỗi tenant có cây riêng, RLS enforce.
7. **Permission check** — RBAC theo vị trí trong cây.

## Decision

Dùng **PostgreSQL LTREE extension** để lưu trữ và query cây CRM.

### LTREE là gì?

PostgreSQL extension lưu path dạng `lquery` / `ltxtquery` (vd: `1.5.23.7`), hỗ trợ:

- `<@` "is descendant of"
- `@>` "is ancestor of"
- `~` "matches lquery"
- `?` (any descendant)
- Index `GIST` / `SP-GiST` cho sub-millisecond query.

### Schema

```sql
-- Node bất kỳ (user, team, department) trong cây CRM
CREATE TABLE crm.nodes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    parent_id    UUID REFERENCES crm.nodes(id),
    -- LTREE path: ví dụ 'r.a1.b2.c3' (tenant root auto-prepend 'r')
    path         LTREE NOT NULL,
    ntype        TEXT NOT NULL,           -- 'user' | 'team' | 'department'
    name         TEXT NOT NULL,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now(),
    UNIQUE (tenant_id, path)
);

CREATE INDEX idx_crm_nodes_path_gist ON crm.nodes USING GIST (path);
CREATE INDEX idx_crm_nodes_path_spgist ON crm.nodes USING SPGIST (path);
CREATE INDEX idx_crm_nodes_parent ON crm.nodes (parent_id);
CREATE INDEX idx_crm_nodes_tenant ON crm.nodes (tenant_id);

-- RLS: chỉ truy cập node trong tenant của mình
ALTER TABLE crm.nodes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON crm.nodes
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
```

### Common queries

```sql
-- Tất cả descendant của node X
SELECT * FROM crm.nodes WHERE path <@ 'r.a1.b2';

-- Tất cả ancestor của node Y
SELECT * FROM crm.nodes WHERE path @> 'r.a1.b2.c3';

-- Tất cả user (leaf) trong subtree của manager Y
SELECT * FROM crm.nodes
WHERE path <@ 'r.a1.b2'
  AND ntype = 'user';

-- Move subtree: update path của tất cả descendant
UPDATE crm.nodes
SET path = 'r.a99.b2' || subpath(path, nlevel('r.a1.b2') - 1)
WHERE path <@ 'r.a1.b2'
  AND path != 'r.a1.b2';

-- Cycle prevention (move không vào con của chính nó)
SELECT EXISTS (
    SELECT 1 FROM crm.nodes
    WHERE path @> 'r.a1.b2.c3'  -- node Y là ancestor của new_parent
      AND id = 'a1.b2'           -- current node đang move
);
```

### Sync với `auth.users`

Mỗi user có entry trong `crm.nodes` với `ntype = 'user'`, liên kết qua `user_id`:

```sql
ALTER TABLE auth.users ADD COLUMN crm_node_id UUID REFERENCES crm.nodes(id);
```

Khi tạo user → insert `crm.nodes` trong cùng transaction.

### RBAC subtree check

```sql
-- User U (Agent) có quyền read Lead của Manager M không?
SELECT EXISTS (
    SELECT 1 FROM crm.nodes
    WHERE id = :user_node_id              -- node của U
      AND :manager_node_path <@ path      -- M là ancestor của U
);
```

Hoặc đơn giản: `M.path @> U.path` ⇒ U là descendant của M ⇒ M có quyền read Lead của U.

### Materialized view cho hot path

```sql
CREATE MATERIALIZED VIEW crm.user_subtree AS
SELECT
    u.id AS user_id,
    u.tenant_id,
    n.id AS node_id,
    n.path,
    -- Ancestor chain
    array(
        SELECT node_id FROM crm.nodes
        WHERE path @> n.path AND path != n.path
        ORDER BY nlevel(path)
    ) AS ancestor_node_ids,
    -- Subtree size
    (SELECT count(*) FROM crm.nodes WHERE path <@ n.path) AS subtree_size
FROM auth.users u
JOIN crm.nodes n ON u.crm_node_id = n.id;

CREATE UNIQUE INDEX ON crm.user_subtree (user_id);
REFRESH MATERIALIZED VIEW CONCURRENTLY crm.user_subtree;
```

REFRESH mỗi giờ hoặc sau batch operation.

### Move operation safety

```go
// pseudo-code
func MoveSubtree(ctx context.Context, nodeID, newParentID UUID) error {
    tx, _ := db.Begin(ctx)
    defer tx.Rollback()

    var oldPath, newParentPath ltree
    tx.QueryRow("SELECT path FROM crm.nodes WHERE id = $1", nodeID).Scan(&oldPath)
    tx.QueryRow("SELECT path FROM crm.nodes WHERE id = $1", newParentID).Scan(&newParentPath)

    // Cycle check: newParent không được là descendant của nodeID
    if newParentPath <@ oldPath {
        return ErrCycleDetected
    }

    // Compute new path
    newPath := newParentPath + subpath(oldPath, nlevel(oldPath) - nlevel(oldPath) + 1)
    // Hoặc nếu move toàn subtree:
    // newPath := newParentPath || subpath(oldPath, nlevel(oldPath))

    // Update tất cả descendant
    tx.Exec(`
        UPDATE crm.nodes
        SET path = $1 || subpath(path, nlevel($2))
        WHERE path <@ $2
    `, newParentPath, oldPath)

    tx.Commit()
    // Publish NATS event: crm.node.moved
    return nil
}
```

### Giới hạn

- LTREE label tối đa **256 chars**, không chứa ký tự `.`, `/`, `\`.
- Path depth practical: < 100 (PostgreSQL giới hạn 65535/8).
- Mỗi label phải unique trong parent (giải quyết qua UNIQUE(tenant_id, path)).

## Consequences

### Positive

- **Native PostgreSQL** — không cần extension thứ 3 ngoài `ltree` (built-in).
- **Subtree query cực nhanh** — O(log n) với GIST index.
- **Cycle detection** — `path <@ path` check tự nhiên.
- **Materialized view** giải quyết hot-path RBAC check.
- **Audit trail** — toàn bộ path được log, dễ reconstruct.

### Negative

- **Move subtree expensive** — phải UPDATE tất cả descendant path (mitigate qua async job + temporary shadow path).
- **Không support cây với cạnh có weight khác nhau** (LTREE là unweighted tree).
- **Khó migrate ra khỏi PostgreSQL** nếu sau này muốn (mitigate qua ORM abstraction).

### Mitigations

- **Batch move** trong 1 transaction + off-peak window.
- **Shadow path**: insert node mới với new_path trước, switch sau.
- **Giới hạn subtree size** ở mức 10,000 nodes (cảnh báo nếu vượt).

## Alternatives Considered

### A. Adjacency list (`parent_id` only)

- **Pro**: schema đơn giản.
- **Con**: subtree query cần recursive CTE — chậm với deep tree.
- **Verdict**: ❌ Rejected — không scale.

### B. Nested set model

- **Pro**: subtree query nhanh (1 query).
- **Con**: insert/move rất expensive (update nhiều rows), cycle detection khó.
- **Verdict**: ❌ Rejected — CRM thay đổi cấu trúc thường xuyên.

### C. Closure table

- **Pro**: query nhanh, dễ move.
- **Con**: storage O(n²) trong worst case, maintenance cost cao.
- **Verdict**: ❌ Rejected — không scale cho 100k+ users.

### D. Path enumeration (Materialized path)

- **Pro**: như LTREE nhưng custom impl.
- **Con**: phải tự implement sort, query.
- **Verdict**: 🟡 PostgreSQL LTREE chính là materialized path có index support.

### E. Graph database (Neo4j)

- **Pro**: relationship query mạnh.
- **Con**: thêm 1 database (overhead), RLS khó.
- **Verdict**: ❌ Rejected — không cần graph power cho CRM tree.

## References

- [PostgreSQL LTREE docs](https://www.postgresql.org/docs/current/ltree.html)
- [ARCHITECTURE.md §4.2](../ARCHITECTURE.md#42-authorization-rbac--abac)
- [services/crm-service](../../services/crm-service) — implementation
- [docs/03-crm-tree/](../03-crm-tree/) — design chi tiết
- [ADR-0001 Polyglot Persistence](0001-polyglot-persistence.md)
- [ADR-0005 RLS Stickiness](0005-rls-stickness-mitigation.md)