-- Migration 0015: Add user FK constraints (extracted from 0008_users_tree.sql)
-- Problem: 0008 ran ALTER TABLE ADD COLUMN IF NOT EXISTS + FK in a DO block,
-- which fails when column already exists (IF NOT EXISTS skips col but constraint
-- syntax still evaluated). This migration safely adds constraints idempotently.
-- =============================================================================

DO $$
BEGIN
    -- contacts.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'contacts_owner_user_id_fkey'
    ) THEN
        ALTER TABLE contacts ADD CONSTRAINT contacts_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- deals.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'deals_owner_user_id_fkey'
    ) THEN
        ALTER TABLE deals ADD CONSTRAINT deals_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- deal_stage_history.changed_by
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'deal_stage_history_changed_by_fkey'
    ) THEN
        ALTER TABLE deal_stage_history ADD CONSTRAINT deal_stage_history_changed_by_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- activities.owner_user_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'activities_owner_user_id_fkey'
    ) THEN
        ALTER TABLE activities ADD CONSTRAINT activities_owner_user_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- notes.author_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'notes_author_id_fkey'
    ) THEN
        ALTER TABLE notes ADD CONSTRAINT notes_author_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
    
    -- users.self-referential parent_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'users_parent_id_fkey'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_parent_id_fkey 
            REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;
