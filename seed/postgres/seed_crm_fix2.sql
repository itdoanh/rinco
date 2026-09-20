-- =============================================================
-- Seed public.leads (maps from leads.leads data)
-- then public.lead_stage_history
-- =============================================================

-- Insert into public.leads (stage_id references public.pipeline_stages)
INSERT INTO public.leads (id, tenant_id, source_id, owner_user_id, full_name, email, phone, company_name, status, score, score_tier, estimated_value, utm, stage_id, custom_fields)
SELECT
    id,
    tenant_id,
    NULL,
    owner_user_id,
    full_name,
    email,
    phone,
    (custom_fields->>'company')::text,
    stage,
    score,
    CASE WHEN score >= 80 THEN 'hot' WHEN score >= 50 THEN 'warm' WHEN score >= 20 THEN 'cold' ELSE 'low' END,
    (custom_fields->>'estimated_value')::numeric,
    jsonb_build_object(
        'utm_source', utm_source,
        'utm_medium', utm_medium,
        'utm_campaign', utm_campaign,
        'utm_content', utm_content,
        'utm_term', utm_term
    ),
    CASE
        WHEN stage = 'new' THEN '21000001-0000-0000-0000-000000000001'::uuid
        WHEN stage = 'contacted' THEN '21000001-0000-0000-0000-000000000002'::uuid
        WHEN stage = 'qualified' THEN '21000001-0000-0000-0000-000000000003'::uuid
        WHEN stage = 'proposal' THEN '21000001-0000-0000-0000-000000000004'::uuid
        WHEN stage = 'won' THEN '21000001-0000-0000-0000-000000000005'::uuid
        WHEN stage = 'lost' THEN '21000001-0000-0000-0000-000000000006'::uuid
        ELSE NULL
    END,
    custom_fields
FROM leads.leads
WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
ON CONFLICT (id) DO NOTHING;

-- Now insert lead_stage_history (references public.leads)
INSERT INTO public.lead_stage_history (id, tenant_id, lead_id, from_stage, to_stage, changed_by, notes) VALUES
  ('42000001-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000001', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000010', 'Initial call completed, interested in loan'),
  ('42000001-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000001', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000010', 'Documents verified, income confirmed'),
  ('42000001-0000-0000-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000004', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000011', 'Phone consultation done'),
  ('42000001-0000-0000-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000004', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000011', 'Referred by existing client, high intent'),
  ('42000001-0000-0000-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000012', 'Email inquiry'),
  ('42000001-0000-0000-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000012', 'Site visit completed'),
  ('42000001-0000-0000-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000005', 'qualified', 'proposal', 'a0000001-0000-0000-0000-000000000012', 'Proposal sent, awaiting response'),
  ('42000001-0000-0000-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000014', 'Walk-in customer'),
  ('42000001-0000-0000-0000-000000000009', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'contacted', 'qualified', 'a0000001-0000-0000-0000-000000000014', 'Documents submitted'),
  ('42000001-0000-0000-0000-000000000010', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'qualified', 'proposal', 'a0000001-0000-0000-0000-000000000014', 'Custom loan package presented'),
  ('42000001-0000-0000-0000-000000000011', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000010', 'proposal', 'won', 'a0000001-0000-0000-0000-000000000014', 'Contract signed!'),
  ('42000001-0000-0000-0000-000000000012', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000003', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000011', 'Zalo contact made'),
  ('42000001-0000-0000-0000-000000000013', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000003', 'contacted', 'lost', 'a0000001-0000-0000-0000-000000000011', 'Not interested anymore, budget concerns'),
  ('42000001-0000-0000-0000-000000000014', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000007', 'new', 'contacted', 'a0000001-0000-0000-0000-000000000013', 'TikTok ad response'),
  ('42000001-0000-0000-0000-000000000015', 'aaaaaaaa-0000-0000-0000-000000000001', '41000001-0000-0000-0000-000000000007', 'contacted', 'lost', 'a0000001-0000-0000-0000-000000000013', 'Chose competitor offering lower rate')
ON CONFLICT (id) DO NOTHING;

-- Verify
SELECT 'public.leads' AS table_name, COUNT(*) AS row_count FROM public.leads WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001'
UNION ALL SELECT 'lead_stage_history', COUNT(*) FROM public.lead_stage_history WHERE tenant_id = 'aaaaaaaa-0000-0000-0000-000000000001';
