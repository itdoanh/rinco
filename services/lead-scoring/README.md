# Lead Scoring Service (RINCO)

> **Phân hệ #11.3 — AI Predictive** · Ensemble ML scoring 0-100 với tier + recommended_action + SHAP explanations.
> Multi-tenant (model per tenant + global fallback). Pull features từ PostgreSQL/ClickHouse/MongoDB, publish `lead.scored` qua NATS JetStream, writeback score vào PG.

## 1. Endpoints (7)

| Method | Path | Mô tả |
|--------|------|-------|
| POST | `/v1/score/{lead_id}` | Score 1 lead |
| POST | `/v1/score` | Score 1 lead (no path id) |
| POST | `/v1/score/batch` | Batch ≤ 1000 |
| POST | `/v1/train` | Train lại (admin) |
| GET  | `/v1/model/info` | Metrics hiện tại |
| POST | `/v1/explain/{lead_id}` | SHAP top factors |
| GET  | `/v1/health` / `/v1/metrics` | Self |

## 2. Architecture

```
                 ┌────────────────────────────────────────────────────────┐
                 │                  NATS JetStream                         │
                 │   subscribe "lead.created"  → publish "lead.scored"     │
                 └─────────────────────┬──────────────────────────────────┘
                                       ▼
   ┌──────────────────────────────────────────────────────────────────┐
   │                    Lead Scoring Python                            │
   │   ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐      │
   │   │  Feature    │  │  Ensemble    │  │  Post-processing     │      │
   │   │  Engineer   │─▶│  XGB + NN +  │─▶│  tier / action / SHAP │     │
   │   │  (~40 cols) │  │  Fallback    │  │  writeback PG         │     │
   │   └─────────────┘  └──────────────┘  └──────────────────────┘      │
   └──────────────────────────────────────────────────────────────────┘
```

## 3. Features (40+)

| Group         | Names |
|---------------|-------|
| Demographic   | `has_phone`, `has_email`, `country_vn/us`, `company_length` |
| Behavioral    | `page_views`, `time_on_site_log`, `repeat_visits`, `fbclid_present` |
| Engagement    | `email_opens`, `email_clicks`, `form_fills`, `open_rate`, `ctr` |
| Firmographic  | `is_b2b`, `company_size_sm/md/lg`, `company_revenue_log` |
| Source        | `source_paid/organic/direct/referral`, `source_quality`, `channel_conv_rate` |
| Device        | `is_mobile/tablet/desktop` |

## 4. Models

1. **XGBoost** (primary) — `n_estimators=80, max_depth=4, lr=0.08`
2. **Neural Net ONNX** — loaded via `onnxruntime` if available (share=0.3)
3. **LogisticRegression fallback** — always trained (share=1 - sum(weights))
4. Ensemble: weighted average of available model probabilities.

## 5. Tier + Action matrix

| Score 0-100  | Tier      | recommended_action |
|--------------|-----------|--------------------|
| ≥ 85         | very-hot  | call-now           |
| ≥ 60         | hot       | call-soon          |
| ≥ 30         | warm      | email              |
| < 30         | cold      | nurture            |

## 6. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8092` | HTTP port |
| `MODEL_DIR` | `/models` | dir chứa `<tenant>/model.pkl` |
| `NATS_URL` | `nats://nats:4222` | NATS server |
| `NATS_SUBJECT` | `lead.created` | listen subject |
| `LOG_LEVEL` | `INFO` | - |
| `ALLOW_PUBLIC_TRAIN` | `0` | set `1` để public train |

## 7. Run

```bash
pip install -r services/lead-scoring/requirements.txt
cd services/lead-scoring
uvicorn main:app --host 0.0.0.0 --port 8092 --workers 2
```

Hoặc Docker:
```bash
docker build -t rinco/lead-scoring -f Dockerfile .
```

## 8. Examples

```bash
curl -X POST http://localhost:8092/v1/score/L-001 \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: apex' \
  -d '{
    "email": "alice@example.com",
    "phone": "0981234567",
    "source": "facebook_ads",
    "page_views": 5,
    "time_on_site_seconds": 240,
    "has_phone": true,
    "fbclid": "abc",
    "country": "VN",
    "device_type": "mobile",
    "email_opens": 3,
    "email_clicks": 1,
    "form_fills": 1
  }'
```

```bash
# Train lại (super-admin only)
curl -X POST http://localhost:8092/v1/train \
  -H 'X-Is-Super-Admin: true' \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id": "apex", "notes": "Q3 retrain"}'
```

## 9. Tests

```bash
cd services/lead-scoring
pytest -q
```
