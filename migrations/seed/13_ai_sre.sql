-- =====================================================================
-- AI SRE seed — sample anomaly + RCA + scaling events used by demo
-- dashboards and chaos tests.
-- Schema mirrors the in-memory store at services/ai-sre/app/services/incident_store.py
-- plus the Prometheus side of the data model.
-- =====================================================================

-- AI SRE incidents
INSERT INTO rinco.ai.aisre_incidents (
    id, tenant_id, service, trace_id, severity, error, status,
    root_cause, confidence, pr_url, runbook_url, created_at, resolved_at
) VALUES
    ('22222222-2222-2222-2222-000000000001',
     'demo-tenant', 'crm-core', 'trace-aaaa-0001', 'P1',
     'pq: deadlock detected', 'resolved',
     'Two concurrent update_lead flows held the row lock; introduce row-level ORDER BY id and retry with backoff.',
     0.87,
     'https://github.com/itdoanh/rinco/pull/482',
     'https://wiki.rinco.app/runbooks/crm-deadlock',
     now() - INTERVAL '6 hours',
     now() - INTERVAL '5 hours 50 minutes'),

    ('22222222-2222-2222-2222-000000000002',
     'demo-tenant', 'chat-engine', 'trace-aaaa-0002', 'P2',
     'scylla timeout during presence update', 'open',
     'Presence batch size too large under load; reduce from 200 to 50 and add timeout.',
     0.74, NULL, NULL,
     now() - INTERVAL '40 minutes',
     NULL),

    ('22222222-2222-2222-2222-000000000003',
     'demo-tenant', 'meeting-ui', 'trace-aaaa-0003', 'P3',
     'transient WebRTC peer connection close', 'resolved',
     'Network jitter on the user's side; auto-reconnect handled by the SFU client.',
     0.55, NULL, NULL,
     now() - INTERVAL '2 days',
     now() - INTERVAL '2 days' + INTERVAL '10 minutes')
ON CONFLICT (id) DO NOTHING;

-- AI SRE anomaly observations (rolling 24h window)
INSERT INTO rinco.ai.aisre_anomalies (
    id, tenant_id, service, metric, value, expected_value,
    deviation, severity, detector, observed_at
) VALUES
    ('33333333-3333-3333-3333-000000000001',
     'demo-tenant', 'crm-core', 'error_rate', 0.082, 0.012, 6.83,
     'critical', 'zscore', now() - INTERVAL '4 hours'),
    ('33333333-3333-3333-3333-000000000002',
     'demo-tenant', 'chat-engine', 'rpc_latency_p99', 1.85, 0.45, 4.11,
     'high', 'ewma', now() - INTERVAL '2 hours'),
    ('33333333-3333-3333-3333-000000000003',
     'demo-tenant', 'meeting-ui', 'cpu_usage', 0.91, 0.55, 1.65,
     'medium', 'iqr', now() - INTERVAL '90 minutes'),
    ('33333333-3333-3333-3333-000000000004',
     'demo-tenant', 'crm-core', 'request_rate', 0.21, 0.62, -2.04,
     'medium', 'zscore', now() - INTERVAL '15 minutes')
ON CONFLICT (id) DO NOTHING;

-- Auto-scaling recommendations
INSERT INTO rinco.ai.aisre_scaling_recommendations (
    id, tenant_id, service, metric, action,
    current_replicas, recommended_replicas, severity,
    rationale, predicted_peak, generated_at
) VALUES
    ('44444444-4444-4444-4444-000000000001',
     'demo-tenant', 'crm-core', 'cpu', 'scale_up',
     4, 7, 'high',
     'Predicted peak 0.91 over 30 minutes exceeds target 0.7 with safety margin 1.25; scale from 4 to 7.',
     0.91, now() - INTERVAL '4 hours'),
    ('44444444-4444-4444-4444-000000000002',
     'demo-tenant', 'chat-engine', 'memory', 'add_capacity',
     8, 12, 'critical',
     'Predicted peak 0.88 + safety margin exceeds max_replicas=12; provision extra nodes.',
     0.88, now() - INTERVAL '2 hours'),
    ('44444444-4444-4444-4444-000000000003',
     'demo-tenant', 'meeting-ui', 'cpu', 'hold',
     6, 6, 'low',
     'Predicted peak 0.61 within target 0.7; no change needed.',
     0.61, now() - INTERVAL '90 minutes'),
    ('44444444-4444-4444-4444-000000000004',
     'demo-tenant', 'lead-scoring', 'latency_p95', 'scale_up',
     3, 5, 'high',
     'Backlog depth above threshold; scale up to 5 replicas to keep p95 below 500ms.',
     0.78, now() - INTERVAL '45 minutes')
ON CONFLICT (id) DO NOTHING;
