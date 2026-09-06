# AI SRE Service

AI-powered Site Reliability Engineering service - tự động phân tích incidents, RCA và đề xuất hotfixes.

## Tính năng

- **Incident Detection**: Nhận alerts từ Sentry, Prometheus Alertmanager
- **Log Aggregation**: Tự động fetch logs liên quan từ ClickHouse/Loki
- **Trace Analysis**: Pull distributed traces từ Jaeger/Tempo
- **Root Cause Analysis (RCA)**: Sử dụng LLM để phân tích
- **Code Retrieval**: Fetch source code từ GitHub để hiểu context
- **Auto Hotfix PR**: Tạo Pull Request với code fix đề xuất
- **Slack/Telegram Alerts**: Thông báo cho team
- **Knowledge Base**: RAG trên past incidents
- **Runbook Suggestions**: Đề xuất actions từ past incidents

## Công nghệ

- **Language**: Python 3.11+
- **Framework**: FastAPI
- **LLM**: DeepSeek-Coder, Qwen2.5, Llama 3.1
- **Vector DB**: Qdrant
- **Embeddings**: sentence-transformers
- **External**: Sentry API, GitHub API, ClickHouse, Jaeger

## Workflow

```
1. Alert triggered (Sentry/Prometheus)
   ↓
2. AI SRE receives incident
   ↓
3. Query ClickHouse for related logs (5min window)
   ↓
4. Query Jaeger for failed traces
   ↓
5. Fetch affected source code from GitHub
   ↓
6. Build context (logs + traces + code)
   ↓
7. LLM analyzes and proposes RCA
   ↓
8. LLM generates code fix
   ↓
9. Create draft GitHub PR with fix
   ↓
10. Notify team via Telegram/Slack
   ↓
11. Human reviews and merges
```

## API Endpoints

```
POST   /incident               - Receive new incident
POST   /analyze                - Analyze with logs + traces + code
POST   /hotfix                 - Generate code fix PR
GET    /incidents              - List recent incidents
GET    /incidents/:id          - Incident details
GET    /runbooks               - List runbooks
POST   /runbooks               - Add runbook
GET    /health                 - Health check
```

## Incident Payload

```json
{
  "incident_id": "sentry-12345",
  "service": "auth-service",
  "severity": "high",
  "title": "Database connection timeout",
  "error_message": "pq: connection timeout after 30s",
  "stack_trace": "...",
  "affected_users": 150,
  "timestamp": "2026-09-07T10:30:00Z",
  "metadata": {
    "environment": "production",
    "region": "ap-southeast-1",
    "tenant_id": "uuid"
  }
}
```

## RCA Output

```json
{
  "incident_id": "sentry-12345",
  "root_cause": "Connection pool exhausted due to long-running queries",
  "confidence": 0.92,
  "evidence": [
    "Logs show 'connection pool: 0/100 available' repeated 50+ times",
    "Trace shows 30s+ queries on /v1/auth/login endpoint",
    "Recent deploy changed query timeout from 5s to 30s"
  ],
  "recommendations": [
    "Revert query timeout change",
    "Increase pool size to 200",
    "Add connection pool monitoring"
  ],
  "hotfix": {
    "description": "Increase connection pool size and add timeout",
    "code_diff": "...",
    "files_changed": [
      "internal/db/pool.go"
    ],
    "estimated_risk": "low"
  }
}
```

## Auto Hotfix

AI SRE có thể:
- ✅ Tạo draft PR với code change
- ✅ Add tests
- ✅ Update documentation
- ❌ KHÔNG tự merge (cần human approval)

## Environment Variables

```bash
AI_SRE_PORT=8087
SENTRY_API_URL=https://sentry.io/api/0/
SENTRY_AUTH_TOKEN=sntrys_...
GITHUB_TOKEN=ghp_...
GITHUB_REPO=rinco/platform
CLICKHOUSE_URL=clickhouse://clickhouse:9000
JAEGER_URL=http://jaeger:16686
LLM_API_KEY=sk-...
LLM_MODEL=deepseek-coder
QDRANT_URL=http://qdrant:6333
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=...
AUTO_HOTFIX_ENABLED=false  # Require explicit enable
```

## Development

```bash
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8087 --reload
```

## Safety

- **No auto-merge**: All fixes require human review
- **Sandbox testing**: All generated code tested in isolated env
- **Rollback ready**: Auto-create rollback PR
- **Audit log**: Every action logged for compliance

## Performance

- **RCA generation**: < 60 seconds
- **Hotfix PR creation**: < 2 minutes
- **Knowledge base size**: 10K+ incidents
- **Retrieval accuracy**: > 90%
