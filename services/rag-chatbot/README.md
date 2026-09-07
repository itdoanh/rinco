# rag-chatbot

RAG chatbot service with semantic search, LLM streaming (vLLM) and
multi-tenant Qdrant-backed knowledge base.

## Endpoints

| Method | Path                        | Description |
|--------|----------------------------|-------------|
| POST   | `/v1/chat`                | RAG chat (streaming SSE) |
| POST   | `/v1/ingest`              | Ingest URL/file |
| GET    | `/v1/collections`         | List tenant collections |
| POST   | `/v1/collections`         | Create collection |
| DELETE | `/v1/collections/:name`   | Delete collection |
| POST   | `/v1/search`              | Pure semantic search |
| POST   | `/v1/feedback`            | RLHF feedback |
| GET    | `/v1/health`              | Service health |
| GET    | `/v1/metrics`            | Prometheus |

## Configuration

| Env var             | Default                       |
|--------------------|-------------------------------|
| `VLLM_URL`        | `http://vllm:8000/v1`       |
| `VLLM_API_KEY`    | —                             |
| `EMBEDDING_MODEL`  | `BAAI/bge-m3`               |
| `RERANKER_MODEL`  | `BAAI/bge-reranker-base`    |
| `LLM_MODEL`        | `meta-llama/Llama-3-8B-Instruct` |
| `QDRANT_URL`       | `http://qdrant:6333`         |
| `CHUNK_SIZE`       | `512`                        |
| `CHUNK_OVERLAP`    | `64`                         |
| `TOP_K`            | `20`                         |

## Build

```bash
docker build -f services/rag-chatbot/Dockerfile -t rinco/rag-chatbot .
```