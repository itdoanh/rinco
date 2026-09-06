# RAG Chatbot (RINCO)

> **Phân hệ #11.4 — AI Conversational** · RAG chatbot đa tenant với Qdrant
> vector store + BGE-M3 embeddings + BGE-reranker + vLLM LLM streaming
> (SSE).  Hỗ trợ ingest file (PDF/DOCX/XLSX/MD/TXT) và URL, prompt
> injection defense, PII redaction, rate-limit sẵn qua header.

## 1. Endpoints (8)

| Method | Path | Mô tả |
|--------|------|-------|
| POST | `/v1/chat` | RAG chat (SSE stream nếu `stream=true`) |
| POST | `/v1/ingest` | Ingest từ URL/file/PDF/DOCX/XLSX/MD/TXT |
| GET  | `/v1/collections` | List Qdrant collections |
| POST | `/v1/collections` | Create (auto đặt `tenant_<id>_kb`) |
| DELETE | `/v1/collections/:name` | Delete (chỉ của tenant) |
| POST | `/v1/search` | Semantic search (no LLM) |
| POST | `/v1/feedback` | RLHF (+1/-1/0 + comment) |
| GET  | `/v1/health` / `/v1/metrics` | Self |

## 2. Pipeline

```
USER QUERY
    │
    ▼  PII redaction + prompt-injection strip
[BGE-M3 embed] → Qdrant search top_k=20 (filter tenant_id)
                                       │
                                       ▼
                          [BGE-reranker-base] → top 6
                                       │
                                       ▼
                  [vLLM Llama-3-8B-Instruct / Qwen2.5]
                                       │
                                       ▼
                            Stream tokens (SSE) + citations
```

## 3. Multi-tenant Isolation

- Collection per tenant: `tenant_<tenant_id>_kb`
- Plus payload filter `tenant_id = <self>` ở mọi query
- Delete collection yêu cầu prefix `tenant_<id>` (403 nếu khác)

## 4. Ingestion

| Kind    | Backend                                    |
|---------|--------------------------------------------|
| `text`  | raw                                        |
| `url`   | httpx + HTML strip (production: trafilatura) |
| `pdf`   | pdfplumber                                  |
| `docx`  | python-docx                                 |
| `xlsx`  | openpyxl                                    |
| `md`    | raw                                         |
| `txt`   | raw                                         |

Chunking: recursive char splitter (512 chars / 64 overlap) — tương đương
`RecursiveCharacterTextSplitter` của LangChain.

## 5. Security

- PII redaction trước khi embed & store (email, phone-VN, cccd, tax-code)
- `vLLM prompt` tách riêng system channel (system message trước, user
  sau), reject các pattern `ignore previous instructions / system: / assistant:`
- Rate limit per-user: thêm ở gateway/API layer.

## 6. ENV

| Var | Default | Purpose |
|-----|---------|---------|
| `VLLM_URL` | `http://vllm:8000/v1` | OpenAI-compatible endpoint |
| `VLLM_API_KEY` | `` | vLLM API key |
| `EMBEDDING_MODEL` | `BAAI/bge-m3` | HF / vLLM model id |
| `RERANKER_MODEL` | `BAAI/bge-reranker-base` | cross-encoder |
| `LLM_MODEL` | `meta-llama/Llama-3-8B-Instruct` | vLLM served model |
| `QDRANT_URL` | `http://qdrant:6333` | Qdrant |
| `CHUNK_SIZE` | `512` | chunk size |
| `CHUNK_OVERLAP` | `64` | chunk overlap |
| `TOP_K` | `20` | initial search size |
| `FEEDBACK_LOG_PATH` | `/var/log/rinco/rlhf.jsonl` | JSONL append |

## 7. Run

```bash
pip install -r services/rag-chatbot/requirements.txt
cd services/rag-chatbot
uvicorn main:app --host 0.0.0.0 --port 8091
```

## 8. Examples

### Ingest
```bash
curl -X POST http://localhost:8091/v1/ingest \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: apex' \
  -d '{"source":"https://docs.example.com/pricing.pdf","kind":"url"}'
```

### Chat (SSE)
```bash
curl -N -X POST http://localhost:8091/v1/chat \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: apex' \
  -H 'X-User-ID: u-1' \
  -d '{"query":"Bảng giá gói Pro?","stream":true}'
```

### Search (no LLM)
```bash
curl -X POST http://localhost:8091/v1/search \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: apex' \
  -d '{"query":"trial period","top_k":5}'
```

### Feedback (RLHF)
```bash
curl -X POST http://localhost:8091/v1/feedback \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: apex' \
  -d '{"query":"Bảng giá?", "answer":"...","rating":1,"comment":"ok"}'
```
