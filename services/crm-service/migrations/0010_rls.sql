-- Migration 0010: Enhanced RLS Policies
-- =============================================================================

-- Drop existing RLS policies and recreate with subtree access
DROP POLICY IF EXISTS contacts_tenant_isolation ON contacts;
DROP POLICY IF EXISTS companies_tenant_isolation ON companies;
DROP POLICY IF EXISTS deals_tenant_isolation ON deals;
DROP POLICY IF EXISTS activities_tenant_isolation ON activities;
DROP POLICY IF EXISTS notes_tenant_isolation ON notes;
DROP POLICY IF EXISTS tags_tenant_isolation ON tags;
DROP POLICY IF EXISTS custom_fields_tenant_isolation ON custom_fields;
DROP POLICY IF EXISTS users_tenant_isolation ON users;
DROP POLICY IF EXISTS invite_links_tenant_isolation ON invite_links;

-- Helper function to get user's subtree path
CREATE OR REPLACE FUNCTION get_user_subtree_path(user_id UUID)
RETURNS LTREE AS $$
DECLARE
    user_path LTREE;
BEGIN
    SELECT u.path INTO user_path
    FROM users u
    WHERE u.id = user_id AND u.deleted_at IS NULL;
    RETURN user_path;
END;
$$ LANGUAGE plpgsql STABLE;

-- Users: can see users in own subtree
CREATE POLICY users_tenant_isolation ON users
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            -- User is admin/owner
            (current_setting('app.is_admin', true) = 'true')
            OR
            -- User can see their own subtree
            (path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID))
        )
    )
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (current_setting('app.is_admin', true) = 'true' OR 
             parent_id = current_setting('app.current_user_id', true)::UUID)
    );

-- Contacts: tenant isolation + owner or subtree
CREATE POLICY contacts_tenant_isolation ON contacts
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Companies: all in tenant for managers+
CREATE POLICY companies_tenant_isolation ON companies
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Deals: same as contacts
CREATE POLICY deals_tenant_isolation ON deals
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Activities: tenant isolation + owner or subtree
CREATE POLICY activities_tenant_isolation ON activities
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (owner_user_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (owner_user_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Notes: tenant isolation + owner or subtree
CREATE POLICY notes_tenant_isolation ON notes
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (
            (current_setting('app.is_admin', true) = 'true')
            OR
            (author_id = current_setting('app.current_user_id', true)::UUID)
            OR
            (author_id IN (
                SELECT id FROM users 
                WHERE path <@ get_user_subtree_path(current_setting('app.current_user_id', true)::UUID)
            ))
        )
    );

-- Tags: all in tenant
CREATE POLICY tags_tenant_isolation ON tags
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Custom fields: all in tenant
CREATE POLICY custom_fields_tenant_isolation ON custom_fields
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
    );

-- Invite links: admin only
CREATE POLICY invite_links_tenant_isolation ON invite_links
    FOR ALL
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::UUID
        AND (current_setting('app.is_admin', true) = 'true')
    );

-- Function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_users_materialized_view()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY users_with_depth;
END;
$$ LANGUAGE plpgsql;
