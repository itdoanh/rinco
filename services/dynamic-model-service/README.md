# Dynamic Model Service

Service quản lý dynamic schemas - cho phép mỗi tenant định nghĩa fields riêng cho entities.

## Tính năng

- **JSON Schema Generation**: Tự động tạo JSON Schema từ field definitions
- **Per-tenant Schemas**: Mỗi tenant có schema riêng cho mỗi entity type
- **Field Types**: string, number, boolean, date, enum, json, file, relation
- **Validation Rules**: required, min/max, regex, custom
- **Multi-language Labels**: Hỗ trợ i18n
- **Display Order**: Configurable field ordering
- **Default Values**: Set defaults cho fields
- **Versioning**: Track schema changes
- **Backward Compatibility**: Migrate old data to new schema

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **Database**: PostgreSQL với JSONB
- **Schema Engine**: Custom + JSON Schema draft-07

## API Endpoints

```
POST   /v1/models/fields                      - Tạo field definition
GET    /v1/models/fields                      - List fields (filterable)
GET    /v1/models/fields/:id                  - Field details
PATCH  /v1/models/fields/:id                  - Update field
DELETE /v1/models/fields/:id                  - Delete field

POST   /v1/models/workflows                   - Tạo workflow
GET    /v1/models/workflows                   - List workflows
GET    /v1/models/workflows/:id               - Workflow details
PATCH  /v1/models/workflows/:id               - Update workflow
DELETE /v1/models/workflows/:id               - Delete workflow

GET    /v1/models/schema/:entity_type         - Get JSON Schema
POST   /v1/models/schema/validate             - Validate data
```

## Field Types

- `string` - Text field
- `number` - Numeric field (integer/float)
- `boolean` - True/False
- `date` - Date/datetime
- `enum` - Single select from options
- `multi_select` - Multiple values from options
- `json` - JSON object/array
- `file` - File upload
- `relation` - Link to another entity
- `formula` - Computed field
- `lookup` - Reference to other entity's field

## Example: Field Definition

```json
{
  "entity_type": "lead",
  "field_name": "budget",
  "field_label": "Ngân sách dự kiến",
  "field_type": "number",
  "is_required": true,
  "is_searchable": true,
  "validation_rules": {
    "min": 0,
    "max": 1000000000
  },
  "display_order": 5
}
```

## Example: Generated JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "full_name": {
      "type": "string",
      "title": "Họ và tên"
    },
    "email": {
      "type": "string",
      "format": "email"
    },
    "budget": {
      "type": "number",
      "minimum": 0,
      "maximum": 1000000000
    },
    "industry": {
      "type": "string",
      "enum": ["tech", "finance", "healthcare", "other"]
    }
  },
  "required": ["full_name", "email", "budget"],
  "additionalProperties": false
}
```

## Workflow Definition

```json
{
  "name": "High-value Lead Auto-assign",
  "trigger_type": "lead.created",
  "trigger_config": {
    "conditions": [
      {"field": "budget", "operator": ">", "value": 100000000}
    ]
  },
  "steps": [
    {
      "type": "assign",
      "config": {"user_role": "sales_manager"}
    },
    {
      "type": "notification",
      "config": {
        "channel": "telegram",
        "template": "high_value_lead"
      }
    },
    {
      "type": "ai_score",
      "config": {"priority": "high"}
    }
  ],
  "is_active": true
}
```

## Environment Variables

```bash
DYNAMIC_MODEL_SERVICE_PORT=8086
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
SCHEMA_CACHE_TTL=300
```

## Development

```bash
go build -o bin/dynamic-model-service ./cmd/main.go
./bin/dynamic-model-service
```
