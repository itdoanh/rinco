# Dynamic Model Service

> **Phân hệ #4 — Dynamic Model Engine** · Meta-Schema service cho phép mỗi
> tenant tự định nghĩa entity types, fields, validation rules, workflow, và
> record CRUD + CSV import/export. Multi-tenant qua PostgreSQL RLS.

## 1. Tổng quan

- **Ngôn ngữ:** Go 1.23 + Echo + sql + Prometheus
- **Database:** PostgreSQL 17 (mọi dynamic data trong JSONB + GIN index)
- **Migrations:** `services/dynamic-model-service/migrations/*.sql`
- **Validation engine:** JSON Schema lite + custom (email / phone-VN / tax-code / cccd)
- **Schema generator:** JSON Schema 2020-12 + React JSON Schema Form (rjsf v5) UI hints
- **Versioning:** immutable snapshots trong `model.model_versions`
- **Multi-tenant:** RLS enforc `app.current_tenant_id` + `app.is_super_admin`

## 2. Endpoints (23)

Tất cả endpoint yêu cầu header `X-Tenant-ID`, `X-User-ID`, optional `X-Is-Super-Admin`.

| # | Method | Path                                          | Mô tả                                  |
|---|--------|-----------------------------------------------|----------------------------------------|
| 1 | POST   | `/v1/models`                                  | Tạo dynamic model                      |
| 2 | GET    | `/v1/models`                                  | List + filter (`status`, `slug`)       |
| 3 | GET    | `/v1/models/:id`                              | Retrieve schema metadata               |
| 4 | PUT    | `/v1/models/:id`                              | Update name / description / status     |
| 5 | DELETE | `/v1/models/:id`                              | Soft delete (`status=archived`)        |
| 6 | POST   | `/v1/models/:id/duplicate`                    | Duplicate qua tenant khác              |
| 7 | POST   | `/v1/models/:id/publish`                      | Publish + snapshot                     |
| 8 | GET    | `/v1/models/:id/versions`                     | History versions                       |
| 9 | POST   | `/v1/models/:id/fields`                       | Add field                              |
|10 | DELETE | `/v1/models/:id/fields/:field_id`             | Delete field                           |
|11 | GET    | `/v1/models/:id/fields`                       | List fields (helper)                   |
|12 | POST   | `/v1/models/:id/records`                      | Create record (validate trước khi save) |
|13 | GET    | `/v1/models/:id/records`                      | Query records (JSONB containment)      |
|14 | GET    | `/v1/models/:id/records/:record_id`           | Retrieve record                        |
|15 | PUT    | `/v1/models/:id/records/:record_id`           | Update record (validate lại)           |
|16 | DELETE | `/v1/models/:id/records/:record_id`           | Delete record                          |
|17 | POST   | `/v1/models/:id/import`                       | CSV import (multipart `file=`)         |
|18 | GET    | `/v1/models/:id/export`                       | CSV export                             |
|19 | POST   | `/v1/models/:id/validate`                     | Validate payload (no persistence)      |
|20 | POST   | `/v1/models/:id/migrate`                      | Bump version + snapshot                |
|21 | GET    | `/v1/models/:id/ui-schema`                    | UI hints (rjsf v5 + `ui:order`)        |
|22 | GET    | `/v1/models/:id/json-schema`                  | JSON Schema 2020-12 output             |
|23 | POST   | `/v1/models/:id/restore/:version`             | Khôi phục schema từ version cũ         |

Plus: `/health`, `/ready`, `/metrics`.

## 3. Schema & Tables

| Table | Purpose |
|-------|---------|
| `model.models` | Header metadata, status (`draft`/`published`/`archived`) |
| `model.model_fields` | Per-field defs: name, type, required, default, validation_rules (JSONB), ui_config (JSONB) |
| `model.model_records` | Records với `data JSONB` + GIN index |
| `model.model_versions` | Immutable snapshots cho audit / rollback |
| `model.model_import_jobs` | Theo dõi CSV import progress + errors |

Indexes:
- `GIN (data)` cho JSONB search
- `(model_id, name)` unique
- `(tenant_id, slug, version)` unique

## 4. Validation Rules (`validation_rules` JSONB)

| Field type  | Build-in checks                                            |
|-------------|-------------------------------------------------------------|
| `string`    | `min_length`, `max_length`, `pattern` (regex), `validator`  |
| `number`    | `min`, `max`                                                |
| `integer`   | `min`, `max`                                                |
| `enum`      | `options` (array of strings)                                |
| `email`     | Built-in regex                                              |
| `phone`     | VN phone regex (`+84|0` prefix, 10-11 digits)                |
| `url`       | http/https scheme                                           |
| `color`     | `#RRGGBB`                                                   |
| any         | `json_schema` (full JSON Schema block qua gojsonschema)     |

Custom validators: `email`, `phone-vn`, `tax-code`, `cccd`.

## 5. UI Config (`ui_config` JSONB)

Các key thông dụng:
- `placeholder`, `help_text`, `tooltip`
- `section` (group nhiều field), `width` (`full`/`half`/`third`)
- `show_in_list` (bool), `order`
- `component` (vd `rjsf` widget: `textarea`, `select`, `date`)
- `lookup_collection` (cho `relation`/`ref`)

## 6. Env vars

| Var                          | Default            | Purpose                          |
|------------------------------|--------------------|----------------------------------|
| `PORT`                       | `8084`             | HTTP listen port                 |
| `ENV`                        | `development`      | `development` / `production`    |
| `DB_HOST`                    | `localhost`        | PostgreSQL                       |
| `DB_USER` / `DB_PASSWORD`    | `rinco` / dev pwd  |                                  |
| `DB_NAME`                    | `rinco`            |                                  |
| `DYNAMIC_MODEL_UPLOAD_DIR`   | *(unset)*          | Lưu lại CSV uploads              |

## 7. Run

```bash
# 1. Migrate
psql -U rinco -d rinco -f services/dynamic-model-service/migrations/000_init_dynamic_model.sql

# 2. Run
cd services/dynamic-model-service
go build ./cmd && ./dynamic-model-service
```

Docker:
```bash
docker build -t rinco/dynamic-model-service -f Dockerfile .
```

## 8. Ví dụ

### Tạo model + 2 fields
```bash
curl -X POST http://localhost:8084/v1/models \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: 00000000-0000-0000-0000-000000000000' \
  -H 'X-User-ID:   00000000-0000-0000-0000-000000000000' \
  -d '{"name":"Product","slug":"product","description":"Hàng hóa"}'
# → id = 018f3a9b-...

curl -X POST http://localhost:8084/v1/models/$ID/fields \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: ...' -H 'X-User-ID: ...' \
  -d '{"name":"sku","label":"SKU","type":"string","required":true,
       "validation_rules":{"pattern":"^[A-Z]{3}\\d{4}$","validator":"email"},
       "ui_config":{"section":"main","width":"half"}}'
```

### Create record (auto-validated)
```bash
curl -X POST http://localhost:8084/v1/models/$ID/records \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: ...' -H 'X-User-ID: ...' \
  -d '{"data":{"sku":"ABC1234","name":"Áo sơ mi"}}'
```

### CSV export
```bash
curl "http://localhost:8084/v1/models/$ID/export" -o products.csv
```

### Build UI Schema (for RJSF / form-builder)
```bash
curl http://localhost:8084/v1/models/$ID/ui-schema | jq
```

## 9. Observability

- Prometheus: `GET /metrics` (`rinco_http_requests_total`, `rinco_http_request_duration_seconds`)
- Structured logs (zap JSON): mọi request kèm `trace_id`, `tenant_id`, `user_id`
- `/health` (liveness), `/ready` (PostgreSQL ping)

## 10. Liên kết

- `docs/04-dynamic-model/README.md` — design doc đầy đủ (200+ tính năng)
- `packages/go/db` — RLS context helper
- `packages/go/logger` — structured logger
- `packages/go/middleware` — Echo middleware (Trace / Logger / Metrics / Recovery / CORS)
