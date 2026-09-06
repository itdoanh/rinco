# RAG Chatbot Service

Retrieval-Augmented Generation chatbot với Qdrant vector database.

## Tính năng

- **Document Ingestion**: Multi-format (PDF, DOCX, TXT, MD, HTML)
- **Vector Embeddings**: sentence-transformers/all-MiniLM-L6-v2
- **Semantic Search**: Find relevant context cho câu hỏi
- **Multi-tenant**: Mỗi tenant có knowledge base riêng
- **LLM Integration**: Qwen2.5, Llama 3.1, OpenAI, Anthropic
- **Source Citation**: Trả lời kèm nguồn tham khảo
- **Conversation Memory**: Multi-turn dialogue
- **Streaming Response**: Server-Sent Events
- **Feedback Loop**: User feedback để improve

## Công nghệ

- **Language**: Python 3.11+
- **Framework**: FastAPI
- **Vector DB**: Qdrant
- **Embeddings**: sentence-transformers
- **LLM**: Qwen2.5-7B-Instruct, Llama 3.1-8B
- **Document Parsing**: PyPDF2, python-docx, BeautifulSoup

## Knowledge Base Pipeline

```
1. Upload documents (PDF, DOCX, etc.)
   ↓
2. Extract text
   ↓
3. Chunk into 512-token segments
   ↓
4. Generate embeddings (384-dim)
   ↓
5. Store in Qdrant with metadata
   ↓
6. Index by tenant_id, document_id
```

## Query Pipeline

```
1. User asks question
   ↓
2. Generate embedding for question
   ↓
3. Search Qdrant for top-K (k=5) similar chunks
   ↓
4. Filter by tenant_id
   ↓
5. Build context (chunks + sources)
   ↓
6. LLM generates answer with context
   ↓
7. Return answer + sources + confidence
```

## API Endpoints

```
POST   /query                  - Ask a question
POST   /query/stream           - Streaming response (SSE)
POST   /ingest                 - Ingest document
POST   /ingest/batch           - Batch ingest
GET    /documents              - List documents
DELETE /documents/:id          - Delete document
POST   /feedback               - Submit feedback
GET    /health                 - Health check
```

## Query Request

```json
POST /query
{
  "tenant_id": "uuid",
  "conversation_id": "uuid",
  "question": "Làm sao để tạo tenant mới?",
  "top_k": 5,
  "temperature": 0.7
}
```

## Query Response

```json
{
  "answer": "Để tạo tenant mới, bạn cần gọi API POST /v1/tenants với...",
  "sources": [
    {
      "document_id": "uuid",
      "title": "API Documentation",
      "chunk_text": "POST /v1/tenants endpoint accepts...",
      "score": 0.92,
      "metadata": {
        "section": "Tenants",
        "page": 15
      }
    }
  ],
  "confidence": 0.89,
  "model": "qwen2.5-7b-instruct",
  "tokens_used": 350
}
```

## Document Ingestion

```json
POST /ingest
Content-Type: multipart/form-data

{
  "tenant_id": "uuid",
  "file": <binary>,
  "metadata": {
    "category": "api-docs",
    "language": "vi"
  }
}
```

Supported formats:
- PDF (with OCR fallback)
- DOCX
- TXT, MD
- HTML
- JSON

## Environment Variables

```bash
RAG_CHATBOT_PORT=8088
QDRANT_URL=http://qdrant:6333
QDRANT_COLLECTION=rinco_kb
EMBEDDING_MODEL=sentence-transformers/all-MiniLM-L6-v2
LLM_API_URL=http://llm-service:8080/v1
LLM_MODEL=qwen2.5-7b-instruct
CHUNK_SIZE=512
CHUNK_OVERLAP=50
TOP_K=5
MAX_CONTEXT_LENGTH=4096
```

## Development

```bash
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8088 --reload
```

## Performance

- **Embedding generation**: < 100ms per chunk
- **Vector search**: < 50ms for 1M vectors
- **End-to-end query**: < 2 seconds
- **Concurrent queries**: 50+/instance

## Architecture

```
┌──────────┐
│  Client  │
└────┬─────┘
     │
     ↓
┌──────────────┐
│ FastAPI      │
│ /query       │
└────┬─────────┘
     │
     ├──→ Embedding Service ──→ Qdrant (search)
     │                              │
     │                              ↓
     │                         Top-K chunks
     │                              │
     └──→ LLM Service ←─────────────┘
              │
              ↓
         Answer + Sources
```
