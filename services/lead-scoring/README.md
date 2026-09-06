# Lead Scoring Service

ML-based lead scoring service sử dụng LightGBM, viết bằng Python.

## Tính năng

- **Multi-tenant Models**: Mỗi tenant có model riêng, fallback về default
- **LightGBM**: Gradient boosting cho high accuracy
- **Feature Engineering**: Auto-extract features từ lead data
- **Real-time Scoring**: Single lead scoring < 50ms
- **Batch Scoring**: Bulk scoring với throughput cao
- **Model Retraining**: Auto-trigger retrain khi có data mới
- **Feature Importance**: Explain tại sao lead có điểm cao/thấp
- **GPU Support**: Optional CUDA acceleration
- **A/B Testing**: Multiple model variants

## Công nghệ

- **Language**: Python 3.11+
- **Framework**: FastAPI
- **ML**: LightGBM, scikit-learn, pandas, numpy
- **Model Storage**: PostgreSQL (binary) + S3/MinIO
- **API**: REST + GraphQL

## Model Features

```python
FEATURES = [
    "company_size",        # Number of employees
    "industry",            # Encoded industry
    "position_seniority",  # C-level, VP, Manager, IC
    "email_quality",       # Free vs corporate domain
    "phone_valid",         # Phone validation result
    "utm_source_score",    # Historical conversion by source
    "page_views_30d",      # Behavioral
    "time_on_site_avg",    # Engagement
    "previous_interactions",
    "company_revenue",
    "country",
    "device_type",
]
```

## API Endpoints

```
POST   /score                  - Score single lead
POST   /batch-score            - Score batch of leads
POST   /retrain                - Trigger model retraining (admin)
GET    /model/info             - Model metadata
GET    /model/features         - Feature importance
GET    /health                 - Health check
GET    /metrics                - Prometheus metrics
```

## Scoring Output

```json
{
  "lead_id": "uuid",
  "score": 87,
  "p_ltv": 15000.00,
  "confidence": 0.92,
  "features_importance": {
    "email_quality": 0.28,
    "company_size": 0.22,
    "position_seniority": 0.18,
    ...
  },
  "tier": "high",  // high/medium/low
  "recommendation": "Prioritize immediate outreach"
}
```

## Model Training

```python
# Auto-retrain weekly via cron
# Or manual trigger:
POST /retrain
{
  "tenant_id": "uuid",
  "lookback_days": 90,
  "min_samples": 100
}
```

Training pipeline:
1. Fetch leads + outcomes (won/lost) from PostgreSQL
2. Engineer features
3. Train LightGBM model
4. Validate on holdout set
5. Save model to S3
6. Update model registry
7. Hot-swap model in production

## Performance

- **Single scoring latency**: < 50ms p99
- **Batch throughput**: 1000 leads/sec
- **Model accuracy**: AUC > 0.85
- **Training time**: < 30 minutes for 100K samples

## Environment Variables

```bash
LEAD_SCORING_PORT=8084
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
MODEL_STORAGE_URL=s3://models/lead-scoring/
MODEL_DEFAULT_PATH=/models/default_lgbm_v3.pkl
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
RETRAIN_CRON=0 0 * * 0  # Weekly
GPU_ENABLED=false
```

## Development

```bash
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8084 --reload

# Test
pytest test_main.py
```

## Architecture

```
┌─────────────┐
│ Lead Created│
└──────┬──────┘
       │
       ↓
┌──────────────┐      ┌─────────────┐
│ Feature Eng. │ ←──→ │ PostgreSQL  │
└──────┬───────┘      └─────────────┘
       │
       ↓
┌──────────────┐      ┌─────────────┐
│ LightGBM     │ ←──→ │ S3/MinIO    │
│ Inference    │      │ (Model)     │
└──────┬───────┘      └─────────────┘
       │
       ↓
┌──────────────┐
│ Score + pLTV │
└──────────────┘
```
