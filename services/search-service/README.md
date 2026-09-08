# Search Service

> **Service:** Full-text search across CRM entities (contacts, companies, leads, deals, documents).
> **Stack:** Go + Echo + Meilisearch (HTTP client)
> **DB:** Reuses PostgreSQL `rinco_crm` for reindexing (read-only)

## Stack

- **Go 1.26+**
- **Echo v4** — HTTP framework
- **Meilisearch** — via REST API (no SDK dependency)
- **pgx v5** — optional PostgreSQL reindex support

## Endpoints

### Tenant Search

```
GET    /api/search/v1/search?q=&page=&hits_per_page=&type=    Search documents
POST   /api/search/v1/documents                              Index single document
POST   /api/search/v1/documents/bulk                         Bulk index documents
DELETE /api/search/v1/documents/:id                          Remove document
POST   /api/search/v1/reindex                                Reindex from PostgreSQL
GET    /health/live                                          Liveness
GET    /health/ready                                         Readiness (Meilisearch ping)
```

## Searchable Types

| Type       | Indexed from                  | Use case                  |
|------------|-------------------------------|---------------------------|
| `contact`  | CRM `contacts`                | Find customer/lead        |
| `company`  | CRM `companies`               | Find business             |
| `lead`     | CRM `leads`                   | Find prospect             |
| `deal`     | CRM `deals`                   | Find deal by description  |
| `activity` | CRM `activities`              | Find task/note            |
| `document` | KB docs                       | Find document             |
| `user`     | Auth `super_admins` + users   | Find user                 |

## Vietnamese Support

- **Synonyms**: `công ty ↔ doanh nghiệp ↔ company`
- `khách hàng ↔ customer ↔ client`
- `nhân viên ↔ staff ↔ employee`
- `hợp đồng ↔ contract ↔ agreement`

(See `models.SynonymsDictionary` for full list.)

## Index Settings

The index is configured with:
- `searchableAttributes`: title, body, tags
- `filterableAttributes`: tenant_id, type, tags
- `sortableAttributes`: created_at, updated_at
- Vietnamese-friendly ranking rules

## Tenant Isolation

Every search query automatically applies `tenant_id = '<X-Tenant-ID>'` filter,
ensuring no cross-tenant data leakage. Admin-only cross-tenant search is
performed by omitting the `X-Tenant-ID` header.

## Configuration

| Environment variable | Default                  | Description          |
|----------------------|--------------------------|----------------------|
| `PORT`               | `8094`                   | Listen port          |
| `DATABASE_URL`       | localhost/rinco_crm      | PostgreSQL for reindex |
| `MEILISEARCH_HOST`   | `http://localhost:7700`  | Meilisearch URL      |
| `MEILISEARCH_API_KEY`| (empty)                  | Master/admin key     |

## Tests

```bash
go test ./...
```

## Setup

```bash
# Run Meilisearch (or use docker)
docker run -d -p 7700:7700 -e MEILI_MASTER_KEY=masterKey getmeili/meilisearch

# Start service
DATABASE_URL=postgres://... MEILISEARCH_HOST=http://localhost:7700 go run ./cmd/main.go
```
