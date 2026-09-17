-- =====================================================================
-- RAG chatbot seed — sample knowledge base documents.  These are the
-- canonical "starter" KB every tenant receives at onboarding.  The
-- actual Qdrant points are populated by the service via POST /v1/ingest
-- (see services/rag-chatbot/app/services/knowledge_base.py).
-- =====================================================================

-- A source-of-truth table that operators can read to know what was
-- seeded; the actual vector points live in Qdrant.
CREATE TABLE IF NOT EXISTS rinco.ai.kb_documents (
    id           UUID PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    title        TEXT NOT NULL,
    source_url   TEXT,
    doc_type     TEXT NOT NULL,
    chunk_count  INT NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO rinco.ai.kb_documents (id, tenant_id, title, source_url, doc_type, chunk_count)
VALUES
    ('55555555-5555-5555-5555-000000000001',
     'demo-tenant', 'Getting started with RINCO CRM',
     'https://kb.rinco.example/demo-tenant/docs/getting-started', 'doc', 6),
    ('55555555-5555-5555-5555-000000000002',
     'demo-tenant', 'Lead scoring tiers',
     'https://kb.rinco.example/demo-tenant/docs/lead-scoring-tiers', 'doc', 3),
    ('55555555-5555-5555-5555-000000000003',
     'demo-tenant', 'Creating a meeting room',
     'https://kb.rinco.example/demo-tenant/docs/meeting-rooms', 'doc', 5),
    ('55555555-5555-5555-5555-000000000004',
     'demo-tenant', 'AI SRE and incident analysis',
     'https://kb.rinco.example/demo-tenant/docs/ai-sre', 'doc', 7),
    ('55555555-5555-5555-5555-000000000005',
     'demo-tenant', 'RAG chatbot and knowledge base',
     'https://kb.rinco.example/demo-tenant/docs/rag', 'doc', 5),
    ('55555555-5555-5555-5555-000000000006',
     'demo-tenant', 'Speech-to-text transcription',
     'https://kb.rinco.example/demo-tenant/docs/stt', 'doc', 4),
    ('55555555-5555-5555-5555-000000000007',
     'demo-tenant', 'Observability stack',
     'https://kb.rinco.example/demo-tenant/docs/observability', 'doc', 6),
    ('55555555-5555-5555-5555-000000000008',
     'demo-tenant', 'Privacy and PII handling',
     'https://kb.rinco.example/demo-tenant/docs/privacy', 'doc', 4)
ON CONFLICT (id) DO NOTHING;
