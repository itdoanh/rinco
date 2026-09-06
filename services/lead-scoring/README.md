# Lead Scoring Service

ML-based lead scoring sử dụng LightGBM.

## Endpoints

- `POST /score` - Score a lead (returns 0-100)
- `POST /batch-score` - Batch scoring (max 1000)
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics
- `POST /retrain` - Trigger model retrain (admin only)

## Request

```json
{
  "email": "test@example.com",
  "phone": "0912345678",
  "full_name": "Nguyen Van A",
  "company": "ABC Corp",
  "job_title": "CEO",
  "industry": "fintech",
  "source": "facebook_ads",
  "campaign_id": "uuid",
  "page_views": 5,
  "time_on_site_seconds": 300,
  "has_phone": true,
  "has_email": true,
  "fbclid": "...",
  "gclid": "...",
  "country": "VN",
  "device_type": "desktop",
  "custom_fields": {}
}
```

## Response

```json
{
  "score": 0.78,
  "score_band": "hot",
  "model_version": "1.0.0",
  "feature_importances": {...},
  "explanation": "Lead scored as 'hot'. Top factors: page_views (0.25), has_phone (0.18)",
  "latency_ms": 4
}
```

## Environment

- `MODEL_DIR` - Path to models (default `/models`)
- `PRELOAD_TENANTS` - Comma-separated tenant IDs to preload
- `MLFLOW_URI` - MLflow tracking URI
