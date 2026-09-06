-- 001_seed_default_models.sql
-- Optional system-wide template models (idempotent).

INSERT INTO model.models (id, tenant_id, name, slug, version, status, created_by, description)
SELECT '00000000-0000-0000-0000-000000000001'::uuid,
       '00000000-0000-0000-0000-000000000000'::uuid,
       'Product', 'product', 1, 'published',
       '00000000-0000-0000-0000-000000000000'::uuid,
       'Sample product entity'
WHERE NOT EXISTS (
    SELECT 1 FROM model.models WHERE slug = 'product'
);
