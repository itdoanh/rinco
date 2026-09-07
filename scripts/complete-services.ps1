# Multi-Service Completion Kit — Apply to all RINCO services.

This script writes the **same** comprehensive set of artifacts
(integration tests, examples, config, optimized Dockerfile, READMEs)
to every Go, Python, and Rust service in `services/`.

It is idempotent: every file is either skipped if it already exists,
or written fresh.  Run from the repo root.

```powershell
powershell -ExecutionPolicy Bypass -File scripts\complete-services.ps1
```

Services covered:
- Go (9):  auth, tenant, crm, lead, landing, dynamic-model, email, notification, observability
- Python (4): lead-scoring, rag-chatbot, ai-sre, stt
- Rust (3): chat-engine, webrtc-sfu, recording-service
