-- =====================================================================
-- Lead scoring seed — sample lead scores + features for tenant
-- `demo-tenant`.  These rows live alongside the lead pipeline tables and
-- are intended for demo dashboards + integration tests.
-- =====================================================================

INSERT INTO rinco.ai.lead_scores (
    id, tenant_id, lead_id, score, tier, recommended_action,
    confidence, model_version, features, shap_top, scored_at
) VALUES
    ('11111111-1111-1111-1111-000000000001',
     'demo-tenant', 'lead-001', 87.5, 'very-hot', 'call-now', 0.91, '2.0.0',
     '{"page_views":12,"time_on_site_log":4.9,"is_mobile":1,"email_opens":3,"email_clicks":2,"is_b2b":1}',
     '{"page_views":0.32,"is_b2b":0.21,"email_clicks":0.18}',
     now()),

    ('11111111-1111-1111-1111-000000000002',
     'demo-tenant', 'lead-002', 71.0, 'hot', 'call-soon', 0.88, '2.0.0',
     '{"page_views":7,"time_on_site_log":4.3,"is_mobile":0,"has_phone":1,"email_clicks":1,"is_b2b":1}',
     '{"page_views":0.27,"is_b2b":0.17,"email_clicks":0.13}',
     now()),

    ('11111111-1111-1111-1111-000000000003',
     'demo-tenant', 'lead-003', 48.2, 'warm', 'email', 0.83, '2.0.0',
     '{"page_views":3,"time_on_site_log":3.0,"is_mobile":1,"email_opens":1,"form_fills":0,"is_b2b":0}',
     '{"page_views":0.18,"email_opens":0.10,"is_b2b":-0.12}',
     now()),

    ('11111111-1111-1111-1111-000000000004',
     'demo-tenant', 'lead-004', 22.7, 'cold', 'nurture', 0.78, '2.0.0',
     '{"page_views":1,"time_on_site_log":1.1,"repeat_visits":0,"email_opens":0,"is_b2b":0}',
     '{"page_views":-0.05,"is_b2b":-0.10,"time_on_site_log":-0.08}',
     now()),

    ('11111111-1111-1111-1111-000000000005',
     'demo-tenant', 'lead-005', 92.1, 'very-hot', 'call-now', 0.93, '2.0.0',
     '{"page_views":25,"time_on_site_log":6.1,"form_fills":3,"email_clicks":5,"is_b2b":1,"company_size_lg":1}',
     '{"page_views":0.36,"company_size_lg":0.25,"form_fills":0.21}',
     now()),

    ('11111111-1111-1111-1111-000000000006',
     'demo-tenant', 'lead-006', 64.4, 'hot', 'call-soon', 0.86, '2.0.0',
     '{"page_views":9,"time_on_site_log":4.6,"email_opens":4,"email_clicks":2,"is_b2b":1}',
     '{"page_views":0.24,"email_opens":0.13,"is_b2b":0.15}',
     now()),

    ('11111111-1111-1111-1111-000000000007',
     'demo-tenant', 'lead-007', 38.8, 'warm', 'email', 0.81, '2.0.0',
     '{"page_views":4,"time_on_site_log":3.4,"email_opens":2,"is_mobile":1,"is_b2b":0}',
     '{"page_views":0.14,"email_opens":0.09,"is_mobile":0.05}',
     now()),

    ('11111111-1111-1111-1111-000000000008',
     'demo-tenant', 'lead-008', 15.3, 'cold', 'nurture', 0.74, '2.0.0',
     '{"page_views":0,"time_on_site_log":0.5,"is_mobile":1,"repeat_visits":0,"is_b2b":0}',
     '{"is_b2b":-0.16,"page_views":-0.05,"time_on_site_log":-0.07}',
     now()),

    ('11111111-1111-1111-1111-000000000009',
     'demo-tenant', 'lead-009', 79.2, 'hot', 'call-soon', 0.89, '2.0.0',
     '{"page_views":18,"time_on_site_log":5.5,"form_fills":2,"email_clicks":3,"is_b2b":1}',
     '{"page_views":0.31,"form_fills":0.22,"is_b2b":0.19}',
     now()),

    ('11111111-1111-1111-1111-000000000010',
     'demo-tenant', 'lead-010', 56.7, 'warm', 'email', 0.85, '2.0.0',
     '{"page_views":6,"time_on_site_log":3.9,"email_opens":3,"form_fills":1,"is_b2b":1}',
     '{"page_views":0.21,"form_fills":0.13,"is_b2b":0.12}',
     now())
ON CONFLICT (id) DO NOTHING;
