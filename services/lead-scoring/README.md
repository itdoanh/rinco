# lead-scoring

RINCO AI service: predicts a 0–100 score, tier and recommended action
for inbound leads.  Backed by an ensemble (XGBoost + ONNX NN + sklearn
LogisticRegression baseline) per tenant.

## Endpoints

| Method | Path                       | Description |
|--------|----------------------------|-------------|
| POST   | `/v1/score`                | Score a single lead (no id) |
| POST   | `/v1/score/{lead_id}`      | Score + persist lead_id |
| POST   | `/v1/score/batch`          | Up to 1000 leads per request |
| POST   | `/v1/train`                | (admin) re-train tenant model |
| GET    | `/v1/model/info`           | Current model metadata |
| POST   | `/v1/explain/{lead_id}`    | SHAP feature contributions |
| GET    | `/v1/health`               | Service health |
| GET    | `/v1/metrics`              | Prometheus exposition |

## Pipeline

```
NATS lead.created → feature store (PG/CH/Mongo) → engineer ~40 features
   → ensemble (XGBoost + NN ONNX + LogReg baseline)
   → score + tier + recommended_action + confidence
   → publish lead.scored → writeback PG
```

## Configuration

| Env var                  | Default            |
|--------------------------|--------------------|
| `LEAD_SCORING_ENV`       | `development`      |
| `LEAD_SCORING_HTTP_ADDR` | `:8092`            |
| `LEAD_SCORING_MODEL_DIR` | `/models`          |
| `LEAD_SCORING_NATS_URL`  | `nats://nats:4222` |
| `LEAD_SCORING_NATS_SUBJECT` | `lead.created`  |
| `LEAD_SCORING_POSTGRES_URL` | —                |
| `LEAD_SCORING_CLICKHOUSE_URL` | —             |
| `LEAD_SCORING_MONGO_URL` | —                  |
| `LEAD_SCORING_MAX_BATCH_SIZE` | `1000`        |
| `MLFLOW_TRACKING_URI`    | —                  |
| `MLFLOW_EXPERIMENT`      | `lead-scoring`     |

## Build

```bash
docker build -f services/lead-scoring/Dockerfile -t rinco/lead-scoring .
```

## Test

```bash
cd services/lead-scoring
pytest -q tests/
```