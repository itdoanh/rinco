-- 0005_rls.sql
-- Row Level Security policies

-- Enable RLS
ALTER TABLE landing.landing_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE landing.form_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE landing.form_submissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE landing.tracking_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE landing.fb_pixel_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE landing.conversion_goals ENABLE ROW LEVEL SECURITY;

-- Create policies (requires app.current_tenant_id to be set)
-- For now, we use a simplified approach where policies check tenant_id directly

CREATE POLICY landing_pages_tenant_policy ON landing.landing_pages
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY form_definitions_tenant_policy ON landing.form_definitions
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY form_submissions_tenant_policy ON landing.form_submissions
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tracking_events_tenant_policy ON landing.tracking_events
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY fb_pixel_configs_tenant_policy ON landing.fb_pixel_configs
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY conversion_goals_tenant_policy ON landing.conversion_goals
    FOR ALL
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
