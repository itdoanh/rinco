"""Extended mock data for the AI-SRE service (WS-B Loop 9).

30+ incidents, 50+ runbooks, 20+ hotfixes.
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

__all__ = [
    "EXTENDED_INCIDENTS",
    "EXTENDED_RUNBOOKS",
    "EXTENDED_HOTFIXES",
    "EXTENDED_DEPLOY_HISTORY",
    "EXTENDED_SERVICE_CATALOG",
]


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


# ---------------------------------------------------------------------------
# Extended Incidents (30+) — across RINCO services
# ---------------------------------------------------------------------------
_SERVICES = [
    "auth-service", "tenant-service", "crm-service", "lead-service",
    "lead-scoring", "notification-service", "email-service", "billing-service",
    "analytics-service", "search-service", "recording-service", "stt-service",
    "chat-engine", "webrtc-sfu", "landing-service", "meta-capi-service",
    "integration-tests", "dynamic-model-service", "rag-chatbot", "ai-sre",
    "observability-service",
]

_INCIDENT_TYPES = [
    ("JWT verification failed: signature mismatch", "P1"),
    ("WebRTC ICE connection failed: STUN timeout", "P2"),
    ("ClickHouse insert query timeout", "P3"),
    ("PostgreSQL connection pool exhausted", "P1"),
    ("Redis cache miss spike: 95% miss rate", "P2"),
    ("MongoDB replica lag: 30s behind primary", "P2"),
    ("ScyllaDB write timeout", "P1"),
    ("MinIO upload failed: signature mismatch", "P2"),
    ("NATS message backlog: 50K pending", "P2"),
    ("Kafka consumer lag spike: 1M behind", "P3"),
    ("K8s pod OOMKilled", "P1"),
    ("K8s pod CrashLoopBackOff", "P2"),
    ("Service response time > 5s (p99)", "P2"),
    ("API 5xx error rate > 5%", "P1"),
    ("Database deadlock detected", "P2"),
    ("Slow query: 30s execution time", "P3"),
    ("Memory leak detected", "P1"),
    ("Goroutine leak: 10K goroutines", "P2"),
    ("File descriptor exhaustion", "P1"),
    ("Network packet loss > 5%", "P2"),
    ("DNS resolution slow: 2s", "P3"),
    ("SSL certificate expiring in 7 days", "P2"),
    ("Disk usage > 90%", "P2"),
    ("CPU usage > 95% sustained", "P2"),
    ("Tenant quota exceeded: API calls", "P3"),
    ("Webhook delivery failed: 500 errors", "P3"),
    ("User session storage full", "P2"),
    ("AI model inference timeout", "P2"),
    ("Vector search slow: 1.5s p99", "P3"),
    ("Email send queue backed up: 100K", "P2"),
]


def _build_extended_incidents() -> List[Dict[str, Any]]:
    incidents = []
    for i, (error, severity) in enumerate(_INCIDENT_TYPES):
        svc = _SERVICES[i % len(_SERVICES)]
        s_idx = _SERVICES.index(svc)
        incidents.append({
            "incident_id": f"inc-ext-{i:04d}",
            "service": svc,
            "trace_id": f"trace-ext-{uuid.uuid4().hex[:8]}",
            "error": error,
            "severity": severity,
            "status": (["open", "investigating", "mitigating", "resolved", "resolved", "resolved"])[i % 6],
            "created_at": _now(),
            "files": [
                {
                    "path": f"{svc}/internal/handler/{['main','process','service','client','repository'][i % 5]}.go",
                    "line": (10 + i * 13) % 500,
                    "function": (["main","Init","Process","HandleRequest","Save"])[i % 5],
                }
            ],
            "context": {
                "environment": (["production", "staging", "production", "production"])[i % 4],
                "tenant_id": (["demo-tenant-apexfintech","demo-tenant-hct-consulting","demo-tenant"])[i % 3],
                "request_id": f"req-{uuid.uuid4().hex[:12]}",
                "user_agent": "Mozilla/5.0" if i % 2 == 0 else "PostmanRuntime/7.32",
            },
            "tags": [
                (["high-priority", "recurring", "automated"])[i % 3],
                (["backend", "infrastructure", "ai", "data"])[i % 4],
            ],
        })
    return incidents


EXTENDED_INCIDENTS: List[Dict[str, Any]] = _build_extended_incidents()


# ---------------------------------------------------------------------------
# Extended Runbooks (50+)
# ---------------------------------------------------------------------------
_RUNBOOK_TOPICS = [
    ("JWT Verification Failures", "auth-service"),
    ("WebRTC ICE / TURN failures", "video-service"),
    ("Database connection pool exhaustion", "auth-service"),
    ("Redis cache miss spike", "crm-service"),
    ("MongoDB replica lag", "lead-service"),
    ("ScyllaDB write timeout", "chat-engine"),
    ("MinIO upload signature mismatch", "recording-service"),
    ("NATS message backlog", "notification-service"),
    ("Kafka consumer lag spike", "analytics-service"),
    ("K8s pod OOMKilled", "auth-service"),
    ("K8s pod CrashLoopBackOff", "tenant-service"),
    ("API 5xx error rate spike", "crm-service"),
    ("PostgreSQL deadlock", "crm-service"),
    ("Slow query optimization", "analytics-service"),
    ("Memory leak detection", "lead-scoring"),
    ("Goroutine leak investigation", "chat-engine"),
    ("File descriptor exhaustion", "recording-service"),
    ("Network packet loss", "webrtc-sfu"),
    ("DNS resolution slow", "tenant-service"),
    ("SSL certificate renewal", "tenant-service"),
    ("Disk usage high", "observability-service"),
    ("CPU usage high", "lead-scoring"),
    ("Tenant quota exceeded", "billing-service"),
    ("Webhook delivery failures", "integration-tests"),
    ("Session storage full", "auth-service"),
    ("AI model timeout", "rag-chatbot"),
    ("Vector search slow", "search-service"),
    ("Email queue backed up", "email-service"),
    ("Service deployment rollback", "dynamic-model-service"),
    ("Database migration failed", "tenant-service"),
    ("Rate limit exceeded", "api-gateway"),
    ("OAuth flow broken", "auth-service"),
    ("JWT signing key rotation", "auth-service"),
    ("Tenant isolation breach (RLS)", "tenant-service"),
    ("PII data leak detection", "audit-service"),
    ("SOC2 audit logging gaps", "audit-service"),
    ("GDPR data export", "tenant-service"),
    ("Backup restoration drill", "observability-service"),
    ("Disaster recovery failover", "observability-service"),
    ("Multi-region routing", "tenant-service"),
    ("CDN cache invalidation", "landing-service"),
    ("DDoS mitigation", "tenant-service"),
    ("Brute force login detection", "auth-service"),
    ("API quota exhaustion (per tenant)", "billing-service"),
    ("Storage bucket size limit", "recording-service"),
    ("Encryption key rotation", "auth-service"),
    ("Audit log retention", "audit-service"),
    ("Database vacuum and analyze", "crm-service"),
    ("Index rebuild", "search-service"),
    ("Materialized view refresh", "analytics-service"),
    ("Data pipeline backfill", "analytics-service"),
    ("Webhook signature verification", "integration-tests"),
    ("OAuth token refresh", "auth-service"),
    ("AI SRE self-monitoring", "ai-sre"),
]


def _build_extended_runbooks() -> List[Dict[str, Any]]:
    runbooks = []
    for i, (title, service) in enumerate(_RUNBOOK_TOPICS):
        runbooks.append({
            "page_id": f"runbook-ext-{i:03d}",
            "title": title,
            "url": f"https://confluence.example.com/runbook/{title.lower().replace(' ', '-').replace('/', '-')}",
            "service": service,
            "summary": (
                f"Step-by-step runbook for {title}. "
                f"1) Check recent deployments, 2) Review error logs, 3) Run diagnostic commands, "
                f"4) Identify root cause, 5) Apply mitigation, 6) Verify resolution, 7) Post-mortem."
            ),
            "severity_filter": (["P1","P2","P3"])[i % 3],
            "estimated_resolution_time_min": (15 + (i * 7) % 120),
            "tags": [
                (["high-priority", "runbook", "playbook"])[i % 3],
                service,
                (["oncall","backend","database","infrastructure","ai","security"])[i % 6],
            ],
            "updated_at": _now(),
        })
    return runbooks


EXTENDED_RUNBOOKS: List[Dict[str, Any]] = _build_extended_runbooks()


# ---------------------------------------------------------------------------
# Extended Hotfixes (20+)
# ---------------------------------------------------------------------------
EXTENDED_HOTFIXES: List[Dict[str, Any]] = [
    {
        "hotfix_id": f"hf-ext-{i:03d}",
        "incident_id": f"inc-ext-{i:04d}",
        "title": f"Hotfix for {_INCIDENT_TYPES[i % len(_INCIDENT_TYPES)][0][:50]}",
        "diff": (
            f"--- a/{_SERVICES[i % len(_SERVICES)]}/internal/handler/main.go\n"
            f"+++ b/{_SERVICES[i % len(_SERVICES)]}/internal/handler/main.go\n"
            f"@@ -{(20 + i*5) % 200},1 +{(20 + i*5) % 200},1 @@\n"
            f"-    if err := checkOld(); err != nil {{\n"
            f"+    if err := checkNew(); err != nil {{\n"
        ),
        "file": f"{_SERVICES[i % len(_SERVICES)]}/internal/handler/main.go",
        "line": (20 + i * 5) % 200,
        "function": "main",
        "confidence": round(0.75 + (i % 5) * 0.05, 2),
        "created_at": _now(),
        "test_coverage_pct": 80 + (i % 20),
        "risk_level": (["low", "medium", "low", "high"])[i % 4],
    }
    for i in range(22)
]


# ---------------------------------------------------------------------------
# Deployment history (15 entries)
# ---------------------------------------------------------------------------
EXTENDED_DEPLOY_HISTORY: List[Dict[str, Any]] = [
    {
        "deployment_id": f"deploy-ext-{i:03d}",
        "service": _SERVICES[i % len(_SERVICES)],
        "version": f"v{2 + i // 5}.{i % 5}.{i % 3}",
        "environment": (["production","staging","production"])[i % 3],
        "commit_sha": uuid.uuid4().hex[:8],
        "deployed_at": _now(),
        "deployer": "cicd-pipeline",
        "status": (["success","success","success","rolled_back"])[i % 4],
        "duration_sec": 90 + (i * 13) % 180,
        "rollout_strategy": (["blue-green","canary","rolling"])[i % 3],
    }
    for i in range(15)
]


# ---------------------------------------------------------------------------
# Service catalog (RINCO services for SRE monitoring)
# ---------------------------------------------------------------------------
EXTENDED_SERVICE_CATALOG: List[Dict[str, Any]] = [
    {
        "service": svc,
        "language": (["Go","Rust","Python","TypeScript"])[i % 4],
        "tier": (["critical","high","medium","low"])[i % 4],
        "oncall_team": (["platform","data","ai","frontend","infra"])[i % 5],
        "slos": {
            "availability": 0.999 if i % 4 == 0 else 0.99,
            "latency_p99_ms": 500 if i % 3 == 0 else 1000,
            "error_rate": 0.001 if i % 4 == 0 else 0.01,
        },
        "dependencies": [_SERVICES[(i + j) % len(_SERVICES)] for j in range(1, 4)],
        "monitoring": {
            "metrics_endpoint": f"/metrics",
            "health_endpoint": f"/health",
            "logs": f"stdout (json)",
            "traces": f"otlp",
        },
        "runbook_count": 3 + (i % 8),
        "open_incidents": i % 3,
    }
    for i, svc in enumerate(_SERVICES)
]
