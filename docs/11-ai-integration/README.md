# Phần 11 – Tích Hợp AI Toàn Hệ Thống (AI Integration Architecture)

> **Phân hệ:** Trí tuệ nhân tạo được tích hợp sâu vào 3 phân vùng chức năng.  
> **Mục tiêu:** AI SRE tự vận hành, AI Predictive phân tích khách hàng, AI Conversational phục vụ user.  
> **Triết lý:** Zero-Trust AI – mỗi AI agent có quyền giới hạn, cách ly tuyệt đối giữa các tenant.

---

## Mục lục
1. [Tổng quan 3 Phân Vùng AI](#1-tổng-quan-3-phân-vùng-ai)
2. [AI SRE & Code Intelligence](#2-ai-sre--code-intelligence)
3. [AI Predictive CRM & Ads Optimizer](#3-ai-predict-crm--ads-optimizer)
4. [AI Conversational & Media](#4-ai-conversational--media)
5. [Multi-Tenant AI Isolation](#5-multi-tenant-ai-isolation)
6. [Vector DB Comparison & Choice](#6-vector-db-comparison--choice)
7. [AI Operations (AIOps)](#7-ai-operations-aiops)
8. [Danh sách tính năng (≥ 100)](#8-danh-sách-tính-năng)
9. [Database Schema](#9-database-schema)
10. [API Surface](#10-api-surface)

---

## 1. Tổng quan 3 Phân Vùng AI

### 1.1. Sơ đồ
```
┌─────────────────────────────────────────────────────────────────┐
│  System Observability & eBPF Signals                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────┴────────────────────────────────────┐
│                  [ NATS JetStream ]                              │
└─────┬──────────────────────────┬─────────────────────────┬───────┘
      │                          │                         │
      ▼                          ▼                         ▼
┌────────────────────┐  ┌─────────────────────┐  ┌──────────────────────┐
│  1. AI SRE         │  │  2. AI Predictive   │  │  3. AI Conversation  │
│  Code-LLM          │  │  XGBoost/LightGBM   │  │  vLLM + Whisper.cpp │
│  AST + Code RAG    │  │  Lead Scoring/pLTV  │  │  RAG Multi-tenant   │
│  Auto RCA          │  │  FB CAPI Smart      │  │  STT + Summary      │
└────────────────────┘  └─────────────────────┘  └──────────────────────┘
```

### 1.2. Nguyên tắc tích hợp
- **Zero-Trust:** Mỗi AI agent có quyền riêng, không có AI nào có quyền super-admin.
- **Tenant-scoped:** Cache key phải có tenant_id hash.
- **Bounded:** AI action phải qua approval layer cho critical action.
- **Observable:** Mọi AI inference đều log + metric.

---

## 2. AI SRE & Code Intelligence

### 2.1. Mục tiêu
- Tự động Root Cause Analysis khi có lỗi.
- Đề xuất Hotfix Patch.
- Phát hiện code regression.
- Capacity planning.

### 2.2. Pipeline
```
[Error in Sentry/GlitchTip]
        ↓
[Webhook → AI SRE Worker]
        ↓
[1. Query ClickHouse logs by trace_id]
        ↓
[2. Query Jaeger trace spans]
        ↓
[3. Read source code (Git repo at file:line)]
        ↓
[4. AST Parsing → code structure]
        ↓
[5. Code-LLM Analysis (DeepSeek-Coder-V2)]
        ↓
[6. Generate RCA report + suggested fix]
        ↓
[7. Send to Telegram/Slack with link]
        ↓
[Optional: Auto-create PR with patch]
```

### 2.3. Code-LLM Models

| Model | Size | Use case |
|-------|------|----------|
| DeepSeek-Coder-V2-Lite | 16B | Quick analysis, RCA |
| DeepSeek-Coder-V2 | 236B | Deep code review |
| Llama-3-70B-Code | 70B | Alternative |
| Codestral-22B | 22B | Fast inference |

### 2.4. Implementation (Python)
```python
# ai-sre-worker/main.py
import asyncio
from transformers import AutoModelForCausalLM
from code_parser import parse_codebase

class AISREWorker:
    def __init__(self):
        self.model = AutoModelForCausalLM.from_pretrained(
            "deepseek-ai/deepseek-coder-v2-lite-instruct",
            device_map="auto",
            torch_dtype=torch.bfloat16,
        )
    
    async def analyze_incident(self, trace_id: str, error: Exception) -> RCAReport:
        # 1. Fetch logs
        logs = await clickhouse.query(f"SELECT * FROM app_logs WHERE trace_id = '{trace_id}'")
        
        # 2. Fetch trace
        spans = await jaeger.get_trace(trace_id)
        
        # 3. Fetch source code
        stack_trace = self.extract_stack_trace(error)
        source_code = await self.fetch_source(stack_trace)
        
        # 4. Parse AST
        ast = parse_codebase(source_code)
        
        # 5. Build prompt
        prompt = f"""
        Analyze this production incident.
        
        Error: {error.message}
        Stack: {error.stack}
        
        Recent logs:
        {logs[:10]}
        
        Source code (file: {stack_trace.file}, line {stack_trace.line}):
        {source_code}
        
        Provide JSON:
        {{
          "root_cause": "...",
          "why": "...",
          "fix": "code snippet",
          "prevention": "...",
          "severity": "P0|P1|P2|P3"
        }}
        """
        
        response = self.model.generate(prompt, max_new_tokens=2000)
        return RCAReport.parse_raw(response)
    
    async def fetch_source(self, stack: StackFrame) -> str:
        # Use GitHub API or local clone
        return github.get_file_content(stack.repo, stack.file, ref=stack.commit_sha)
```

### 2.5. Auto Hotfix PR
```python
async def create_hotfix_pr(self, rca: RCAReport) -> PRInfo:
    branch = f"hotfix/{rca.error_code}-{uuid.uuid4().hex[:8]}"
    
    # Clone repo
    repo = git.clone(rca.repo_url, branch=branch)
    
    # Apply fix
    repo.edit_file(
        path=rca.error_file,
        original=rca.original_code,
        new=rca.fix_code,
    )
    
    # Commit
    repo.commit(
        message=f"fix: {rca.root_cause}\n\n{rca.why}\n\nAuto-generated by AI SRE",
        author=AI_BOT_SIGNATURE,
    )
    
    # Push & create PR
    repo.push()
    pr = github.create_pull_request(
        repo=rca.repo_url,
        head=branch,
        base="main",
        title=f"🤖 AI Hotfix: {rca.root_cause}",
        body=f"""
## AI RCA
{rca.root_cause}

## Why
{rca.why}

## Suggested Fix
```diff
{rca.diff}
```

## Prevention
{rca.prevention}

Severity: {rca.severity}
Trace: {rca.trace_id}

⚠️ Please review carefully before merge.
        """,
        labels=["ai-generated", "needs-review"]
    )
    return pr
```

### 2.6. Performance
- Inference: < 3 giây (DeepSeek-Coder-V2-Lite, 1x A100).
- Hotfix proposal: < 30 giây.
- Chạy async qua NATS, không block main thread.

---

## 3. AI Predictive CRM & Ads Optimizer

### 3.1. Use Cases
1. **Lead Scoring:** Điểm tiềm năng 0-100.
2. **pLTV (predicted Lifetime Value):** Giá trị vòng đời ước tính.
3. **Conversion Probability:** Xác suất chuyển đổi.
4. **Next Best Action:** Hành động tiếp theo nên làm.
5. **Churn Risk:** Nguy cơ mất khách.
6. **Smart FB Feedback:** Lead nào nên bắn về Facebook.

### 3.2. Pipeline
```
[Lead Form Submit]
        ↓
[Go Ingestion API → ScyllaDB + NATS]
        ↓
[ai-scoring-worker consumes NATS]
        ↓
[1. Feature extraction (clickstream, UTM, time, ...)]
        ↓
[2. XGBoost/LightGBM inference]
        ↓
[3. Score = probability(converted | features)]
        ↓
[4. Write to PostgreSQL lead.score, lead.pLTV]
        ↓
[5. Emit "lead.scored" event to NATS]
        ↓
[meta-capi-worker consume → send to FB]
        ↓
[CRM Service consume → auto-assign if score > 80]
```

### 3.3. Feature Engineering
```python
# ai-scoring/features.py
def extract_features(lead: Lead, clickstream: List[Event]) -> Features:
    return Features(
        # Demographics
        age=extract_age(lead),
        gender=lead.gender,
        location=lead.city,
        
        # Source
        utm_source=lead.utm_source,
        utm_campaign=lead.utm_campaign,
        fbclid_present=bool(lead.fbclid),
        
        # Engagement
        page_views=count(clickstream, 'page_view'),
        time_on_site=sum_duration(clickstream),
        scroll_depth=max(clickstream.scroll_depth),
        video_watched=count(clickstream, 'video_play'),
        
        # Behavioral
        form_time_to_submit=lead.submit_time - lead.first_interaction,
        fields_filled_ratio=lead.filled / lead.total_fields,
        
        # Time
        hour_of_day=lead.created_at.hour,
        day_of_week=lead.created_at.weekday(),
        
        # Historical (per tenant)
        tenant_avg_conversion=tenant_stats.avg_conversion,
        similar_lead_conversion=tenant_stats.similar_score_conversion,
    )
```

### 3.4. Model Training
```python
# ai-scoring/training/train.py
import xgboost as xgb
from sklearn.model_selection import train_test_split

def train_model(tenant_id: str):
    # Load data
    df = postgres.query(f"""
        SELECT features, converted
        FROM lead_training_data
        WHERE tenant_id = '{tenant_id}'
          AND created_at > now() - interval '6 months'
          AND converted IS NOT NULL
    """)
    
    X = df['features']
    y = df['converted']
    
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2)
    
    model = xgb.XGBClassifier(
        n_estimators=200,
        max_depth=6,
        learning_rate=0.1,
        scale_pos_weight=sum(y_train == 0) / sum(y_train == 1),
    )
    model.fit(X_train, y_train)
    
    # Evaluate
    score = model.score(X_test, y_test)
    
    # Save as ONNX
    model.save_model(f'models/{tenant_id}.json')
    convert_to_onnx(model, f'models/{tenant_id}.onnx')
    
    return score
```

### 3.5. ONNX Runtime Inference
```python
import onnxruntime as ort

class ScoringService:
    def __init__(self, model_path: str):
        self.session = ort.InferenceSession(model_path)
    
    def score(self, features: np.ndarray) -> float:
        input_name = self.session.get_inputs()[0].name
        result = self.session.run(None, {input_name: features})[0]
        # result[0][1] = probability of converted
        return float(result[0][1])

# Latency: < 5ms per inference
```

### 3.6. FB CAPI Smart Feedback
```go
// Chỉ bắn những Lead có score cao về Facebook
func SendToCAPI(lead *Lead) error {
    if lead.Score < 70 && lead.Status != "won" {
        return nil  // Skip
    }
    
    event := CAPIEvent{
        EventName: "Lead",
        EventID:   lead.EventID,
        UserData:  buildUserData(lead),
        CustomData: CustomData{
            Value: lead.Score,  // Use score as value
            Currency: "VND",
            ContentName: fmt.Sprintf("score_%d", lead.Score),
        },
    }
    
    // Add predicted LTV if available
    if lead.PredictedLTV > 0 {
        event.CustomData.PredictedLTV = lead.PredictedLTV
    }
    
    return metaCAPI.Send(event)
}

// Khi Lead converted
func SendPurchaseEvent(deal *Deal) error {
    event := CAPIEvent{
        EventName: "Purchase",
        EventID:   deal.EventID,
        UserData:  buildUserData(deal.Lead),
        CustomData: CustomData{
            Value:    deal.Value,
            Currency: deal.Currency,
        },
    }
    return metaCAPI.Send(event)
}
```

### 3.7. Performance
- Training: Daily batch job.
- Inference: < 5ms per lead.
- ONNX Runtime trên CPU, không cần GPU.

---

## 4. AI Conversational & Media

### 4.1. Use Cases
1. **Chatbot:** Trả lời câu hỏi user.
2. **RAG:** Search knowledge base trả lời theo context.
3. **Meeting Transcription:** Whisper.cpp STT.
4. **Meeting Summary:** Llama-3 tóm tắt.
5. **Action Items Extraction:** Auto tạo task.
6. **Email Reply Suggestion:** Gợi ý email trả lời.
7. **Voice Message Transcription:** Voice → text.

### 4.2. Pipeline
```
[User Query]
    ↓
[ai-conversation-service]
    ↓
[1. Tenant resolution]
    ↓
[2. Embedding (BGE-M3)]
    ↓
[3. Vector search (Qdrant/pgvector)]
    ↓
[4. Top-K documents retrieved]
    ↓
[5. Build RAG prompt]
    ↓
[6. vLLM inference (Llama-3-70B)]
    ↓
[7. Stream response back to user]
```

### 4.3. vLLM Deployment
```bash
# Start vLLM with Llama-3-70B-AWQ
python -m vllm.entrypoints.openai.api_server \
  --model meta-llama/Meta-Llama-3-70B-Instruct-AWQ \
  --quantization awq \
  --tensor-parallel-size 2 \
  --gpu-memory-utilization 0.95 \
  --max-model-len 8192 \
  --enable-prefix-caching \
  --port 8000
```

### 4.4. RAG Implementation
```python
from langchain.embeddings import HuggingFaceEmbeddings
from langchain.vectorstores import Qdrant
from langchain.llms import VLLM
from langchain.chains import RetrievalQA

class ConversationEngine:
    def __init__(self, tenant_id: str):
        self.tenant_id = tenant_id
        self.embedding = HuggingFaceEmbeddings(model_name="BAAI/bge-m3")
        self.vectorstore = Qdrant(
            client=qdrant_client,
            collection_name=f"kb_{tenant_id}",
            embeddings=self.embedding,
        )
        self.llm = VLLM(
            model="meta-llama/Meta-Llama-3-70B-Instruct-AWQ",
            vllm_kwargs={...},
        )
        self.qa_chain = RetrievalQA.from_chain_type(
            llm=self.llm,
            retriever=self.vectorstore.as_retriever(search_kwargs={"k": 5}),
            return_source_documents=True,
        )
    
    async def ask(self, question: str, context: List[Message] = None) -> Response:
        # Prefix caching key với tenant_id
        cache_key = hashlib.sha256(
            f"{self.tenant_id}:{self.qa_chain.retriever.get_relevant_documents.__qualname__}".encode()
        ).hexdigest()
        
        # Token admission control
        if not self.admission_control.allow(self.tenant_id):
            raise QueueFull("tenant rate limit exceeded")
        
        result = await self.qa_chain.acall({
            "query": question,
            "chat_history": context or [],
        })
        return Response(
            text=result["result"],
            sources=result["source_documents"],
        )
```

### 4.5. Whisper.cpp STT
```python
# ai-media/stt_worker.py
from pywhispercpp.model import Model

class STTWorker:
    def __init__(self):
        self.model = Model(
            "large-v3",
            n_threads=8,
            print_progress=False,
        )
    
    async def transcribe(self, audio_path: str, language: str = "auto") -> Transcript:
        segments = self.model.transcribe(
            audio_path,
            language=language,
            print_progress=False,
        )
        return Transcript(
            language=language,
            segments=[
                Segment(start=s.t0, end=s.t1, text=s.text, speaker=None)
                for s in segments
            ],
            full_text=" ".join(s.text for s in segments),
        )
```

### 4.6. Meeting Summary
```python
async def summarize_meeting(transcript: Transcript) -> Summary:
    prompt = f"""
    Bạn là trợ lý AI tóm tắt cuộc họp.
    
    Transcript:
    {transcript.full_text}
    
    Provide JSON:
    {{
      "summary": "3-5 câu tóm tắt",
      "key_points": ["point 1", "point 2", ...],
      "action_items": [
        {{"owner": "Tên", "task": "...", "deadline": "YYYY-MM-DD"}}
      ],
      "decisions": ["..."],
      "next_steps": ["..."]
    }}
    """
    response = await llama3.generate(prompt, max_tokens=2000)
    return Summary.parse_raw(response)
```

### 4.7. Real-time Live Caption
- Streaming Whisper từng chunk 2 giây.
- Speaker diarization (pyannote.audio).
- Output qua WebSocket.
- Sub-200ms latency với GPU.

---

## 5. Multi-Tenant AI Isolation

### 5.1. Vấn đề
- **PROMPTPEEK Attack:** Đo First-Token Latency để suy đoán prompt của tenant khác.
- **KV-cache sharing:** vLLM/SGLang chia sẻ KV-cache có thể leak.
- **GPU contention:** 1 tenant spam request → ảnh hưởng tenant khác.

### 5.2. Tenant-Scoped Cache Key
```python
# vLLM scheduler hook
def compute_cache_key(prompt: str, tenant_id: str) -> str:
    prefix = prompt[:200]  # First 200 chars
    salted = f"{tenant_id}:{prefix}"
    return hashlib.sha256(salted.encode()).hexdigest()
```

### 5.3. GPU Token Pool Admission Control
```python
class TokenPool:
    def __init__(self, total_tokens_per_sec: int = 1000):
        self.tenants = {}  # tenant_id -> (rate, capacity, current)
    
    def allow(self, tenant_id: str, tokens: int) -> bool:
        rate, capacity, current = self.tenants.get(tenant_id, (10, 100, 0))
        
        if current + tokens > capacity:
            return False
        
        self.tenants[tenant_id] = (rate, capacity, current + tokens)
        return True
    
    def release(self, tenant_id: str, tokens: int):
        _, _, current = self.tenants[tenant_id]
        self.tenants[tenant_id] = (rate, capacity, max(0, current - tokens))
```

### 5.4. Routing per Tenant
```python
# Allocate vLLM instance per tenant cluster
def route_request(tenant_id: str, model: str) -> vLLMInstance:
    tenant_tier = get_tenant_tier(tenant_id)
    
    if tenant_tier == "enterprise":
        return get_dedicated_instance(tenant_id, model)  # 1 instance per tenant
    elif tenant_tier == "business":
        return get_pooled_instance(tenant_id, model)  # Shared but isolated prefix
    else:
        return get_shared_instance()  # Shared best-effort
```

### 5.5. PII Redaction trước khi gửi AI
```python
def redact_pii(text: str) -> str:
    patterns = {
        'phone': r'\+?\d{9,15}',
        'email': r'[\w\.-]+@[\w\.-]+',
        'ssn': r'\d{3}-\d{2}-\d{4}',
        'credit_card': r'\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}',
    }
    for ptype, pattern in patterns.items():
        text = re.sub(pattern, f'[REDACTED_{ptype.upper()}]', text)
    return text
```

---

## 6. Vector DB Comparison & Choice

### 6.1. Ma trận so sánh

| Tiêu chí | pgvector + pgvectorscale | Qdrant |
|---------|------------------------|--------|
| **Kiến trúc Index** | HNSW / DiskANN | Dynamic HNSW + Quantization |
| **P99 Latency (< 10M)** | 5-15ms | 3-8ms |
| **Throughput (QPS)** | 300-500 | 1,500-2,500+ |
| **Multi-tenant filter** | WHERE tenant_id (SQL) | Payload filter (engine) |
| **Operational cost** | Thấp (reuse Postgres) | Trung bình (cluster riêng) |
| **Cost optimization** | 0 USD | Tối ưu RAM via On-Disk HNSW |
| **Best for** | < 10M vectors, ACID + vector | > 10M vectors, pure vector |

### 6.2. Routing Strategy
```python
def get_vector_store(tenant_id: str) -> VectorStore:
    tenant_vector_count = get_tenant_vector_count(tenant_id)
    
    if tenant_vector_count < 10_000_000:
        return PGVectorStore(tenant_id=tenant_id)  # pgvector
    else:
        return QdrantStore(tenant_id=tenant_id)  # Qdrant
```

---

## 7. AI Operations (AIOps)

### 7.1. AI SRE Workflow
```
[Alert fires]
    ↓
[AI SRE Worker]
    ↓
[1. Pull context: trace_id, logs, source code]
    ↓
[2. Code-LLM analysis]
    ↓
[3. RCA report]
    ↓
[4. Hotfix proposal (if confident)]
    ↓
[5. Notify on-call via Telegram]
    ↓
[Human reviews PR → merges if good]
```

### 7.2. Anomaly Detection
```python
# ai-anomaly/detector.py
from prophet import Prophet
import numpy as np

class AnomalyDetector:
    def __init__(self):
        self.models = {}  # metric -> Prophet
    
    def train(self, metric_name: str, history: pd.DataFrame):
        model = Prophet(interval_width=0.99, daily_seasonality=True)
        model.fit(history)
        self.models[metric_name] = model
    
    def detect(self, metric_name: str, recent: pd.DataFrame) -> bool:
        model = self.models[metric_name]
        forecast = model.predict(recent)
        
        actual = recent['y'].values
        predicted = forecast['yhat'].values
        upper = forecast['yhat_upper'].values
        
        is_anomaly = np.any(actual > upper)
        return is_anomaly
```

### 7.3. Capacity Planning
- Predict traffic 7/30/90 days ahead.
- Recommend scale up/down.
- Cost projection.

### 7.4. Cost Optimization
- Identify idle resources.
- Recommend right-sizing.
- Spot instance suggestions.

---

## 8. Danh sách tính năng (≥ 100)

### 8.1. AI SRE (1-30)
1. Auto RCA on error.
2. Auto Hotfix PR.
3. AI weekly report.
4. AI capacity plan.
5. AI cost optimization.
6. AI anomaly detection.
7. AI security threat scoring.
8. AI tenant health score.
9. AI alert dedup.
10. AI runbook generation.
11. AI runbook execution.
12. AI incident post-mortem.
13. AI change impact analysis.
14. AI dependency map.
15. AI log pattern clustering.
16. AI performance regression.
17. AI SLO recommendations.
18. AI error budget tracking.
19. AI on-call suggestion.
20. AI documentation generator.
22. AI code review assistant.
23. AI test generation.
25. AI vulnerability detection.
26. AI config optimization.
27. AI database tuning.
28. AI query optimization.
29. AI index recommendation.
30. AI backup strategy.

### 8.2. Predictive CRM (31-60)
31. Lead Scoring 0-100.
32. pLTV prediction.
33. Conversion probability.
34. Churn risk.
35. Next best action.
36. Auto-assign Lead.
37. Smart routing (best fit agent).
38. Duplicate detection.
39. Cross-sell/upsell.
40. Win probability.
41. Sales forecast.
42. Revenue projection.
43. Campaign ROI predict.
44. Optimal send time (email).
46. Optimal call time.
47. Lead temperature (hot/warm/cold).
48. Engagement score.
49. Quality score (EMQ feedback).
50. Custom scoring (per tenant).
51. Model retraining.
52. A/B test scoring.
53. Segment auto-clustering.
54. Persona detection.
55. Lookalike audience (FB sync).
56. High-value flag (FB CAPI).
57. Bot likelihood score.
58. Fraud risk.
59. Spam detection.
60. Sentiment analysis.

### 8.3. Conversational AI (61-85)
61. Chatbot RAG.
62. Multi-language chatbot.
63. Custom knowledge base.
64. Custom system prompt per tenant.
65. Conversation memory.
66. Streaming response.
67. Function calling (tools).
68. Web search integration.
69. Code interpreter.
70. Image generation.
71. Voice synthesis.
72. Voice cloning.
73. Custom persona.
74. Guardrails (PII, harmful).
75. Token budget per tenant.
76. Rate limit per tenant.
77. Audit log per conversation.
78. Feedback collection.
79. Hallucination detection.
81. Re-ranking.
82. Hybrid search (vector + BM25).
83. Query rewrite.
84. Step-back prompting.
85. Chain-of-thought.

### 8.4. Media AI (86-110)
86. Real-time STT (Whisper.cpp).
87. Live caption.
88. Speaker diarization.
89. Language detection.
90. Translation.
91. Meeting summary.
92. Action items extraction.
93. Decision tracking.
94. Follow-up email generation.
95. Task auto-create.
96. Sentiment analysis (meeting).
97. Topic segmentation.
98. Topic labeling.
99. Engagement scoring.
100. Talk time per participant.
101. Custom vocabulary.
102. Profanity filter.
103. Compliance check.
104. Meeting quality score.
105. Audio enhancement.
106. Noise removal.
107. Echo cancellation (post-process).
108. Voice cloning (opt-in).
109. Subtitle generation.
110. Multi-language subtitles.

### 8.5. AI Operations (111-130)
111. AI scaling decision.
112. AI cost alerting.
113. AI resource right-sizing.
114. AI spot instance recommendation.
115. AI reserved capacity suggestion.
116. AI query optimization.
117. AI index recommendation.
118. AI cache strategy.
119. AI CDN strategy.
120. AI geo-distribution.
121. AI load balancer config.
122. AI rate limit recommendation.
123. AI retention policy.
124. AI backup strategy.
125. AI disaster recovery drill.
126. AI compliance check.
127. AI GDPR audit.
128. AI security scan.
129. AI penetration test.
130. AI documentation maintenance.

---

## 9. Database Schema

### 9.1. PostgreSQL
```sql
-- AI Models registry
CREATE TABLE ai_models (
  id UUID PRIMARY KEY,
  model_code TEXT UNIQUE NOT NULL,
  model_type TEXT,                  -- 'scoring','rag','stt','summary'
  base_model TEXT,                  -- 'xgboost','llama-3','whisper-large'
  tenant_id UUID,                   -- NULL = system model
  version INT,
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Lead scores (cached for fast read)
CREATE TABLE lead_scores (
  lead_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  score INT,
  p_ltv DECIMAL(18, 2),
  conversion_probability DECIMAL(5, 4),
  churn_risk DECIMAL(5, 4),
  model_version TEXT,
  scored_at TIMESTAMPTZ DEFAULT now()
);

-- Conversation memory
CREATE TABLE ai_conversations (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  user_id UUID NOT NULL,
  session_id UUID,
  messages JSONB NOT NULL,
  token_used INT,
  cost DECIMAL(10, 6),
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Meeting transcripts
CREATE TABLE meeting_ai_summary (
  meeting_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  transcript JSONB,
  summary TEXT,
  action_items JSONB,
  decisions JSONB,
  key_points JSONB,
  sentiment JSONB,
  model_version TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- AI audit log
CREATE TABLE ai_audit_log (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  service TEXT,
  model TEXT,
  input_tokens INT,
  output_tokens INT,
  cost DECIMAL(10, 6),
  latency_ms INT,
  trace_id UUID,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Tenant AI quota
CREATE TABLE tenant_ai_quotas (
  tenant_id UUID PRIMARY KEY,
  monthly_token_limit BIGINT,
  monthly_request_limit BIGINT,
  monthly_cost_limit DECIMAL(10, 2),
  used_tokens BIGINT DEFAULT 0,
  used_requests BIGINT DEFAULT 0,
  used_cost DECIMAL(10, 2) DEFAULT 0,
  reset_at TIMESTAMPTZ
);
```

### 9.2. Qdrant Collections
- `kb_{tenant_id}` – Knowledge base per tenant.
- `documents_{tenant_id}` – Document embeddings.

### 9.3. Valkey
```
ai:ratelimit:{tenant_id} → Token bucket
ai:cache:{tenant_id}:{prefix_hash} → Cached response
ai:tenant:{tenant_id}:quota → Quota counter
```

---

## 10. API Surface

### 10.1. Predictive CRM
```
POST   /api/ai/v1/score/lead              # Trigger scoring
GET    /api/ai/v1/score/lead/:id          # Get score
POST   /api/ai/v1/predict/pltv            # pLTV
POST   /api/ai/v1/predict/conversion
POST   /api/ai/v1/predict/churn
GET    /api/ai/v1/models                  # List models
POST   /api/ai/v1/train                   # Trigger retraining
```

### 10.2. Conversational AI
```
POST   /api/ai/v1/chat/ask
POST   /api/ai/v1/chat/stream             # SSE stream
POST   /api/ai/v1/chat/feedback
GET    /api/ai/v1/chat/history/:session_id
POST   /api/ai/v1/knowledge/upload        # Add to KB
DELETE /api/ai/v1/knowledge/:id
GET    /api/ai/v1/knowledge/search
```

### 10.3. Media AI
```
POST   /api/ai/v1/stt/transcribe          # Upload audio
GET    /api/ai/v1/stt/:id
POST   /api/ai/v1/summarize/meeting
POST   /api/ai/v1/summarize/text
POST   /api/ai/v1/extract/action-items
WS     /ws/ai/stt/live                    # Live caption
```

### 10.4. SRE AI
```
POST   /api/ai-sre/v1/analyze             # Trigger RCA
GET    /api/ai-sre/v1/incidents
POST   /api/ai-sre/v1/hotfix              # Create PR
GET    /api/ai-sre/v1/anomaly/detect
GET    /api/ai-sre/v1/capacity-plan
GET    /api/ai-sre/v1/cost-optimization
```

---

## Phụ lục: Acceptance Criteria

| AC | Tiêu chí | Đo lường |
|----|---------|---------|
| AC-AI-01 | Lead scoring < 5ms | p95 |
| AC-AI-02 | First token latency < 200ms | p95 |
| AC-AI-03 | RCA < 3 giây | p95 |
| AC-AI-04 | Whisper STT < 200ms | Sub-200ms |
| AC-AI-05 | Tenant cache isolation 100% | Pen test |
| AC-AI-06 | PROMPTPEEK defense | Timing attack |

---

**Hoàn thành tài liệu thiết kế chi tiết.**

Xem lại: [`docs/00-master/README.md`](../00-master/README.md) – Bản thiết kế tổng thể.