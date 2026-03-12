-- Migration: Enhanced TOML Support for KB-Drug-Rules Service (FIXED)
-- Version: 004
-- Description: Add TOML format support, versioning, audit trails, and performance optimizations
-- Author: System
-- Date: 2025-01-20

BEGIN;

-- ============================================================================
-- STEP 1: Add new columns for TOML support (using DO block for safety)
-- ============================================================================

DO $$
BEGIN
    -- Add original_format column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'original_format') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN original_format VARCHAR(10) DEFAULT 'json';
    END IF;
    
    -- Add toml_content column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'toml_content') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN toml_content TEXT;
    END IF;
    
    -- Add json_content column (for enhanced storage)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'json_content') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN json_content JSONB;
    END IF;
    
    -- Add previous_version column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'previous_version') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN previous_version VARCHAR(50);
    END IF;
    
    -- Add version_history column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'version_history') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN version_history JSONB DEFAULT '[]';
    END IF;
    
    -- Add deployment_status column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'deployment_status') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN deployment_status JSONB DEFAULT '{}';
    END IF;
    
    -- Add created_by column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'created_by') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN created_by VARCHAR(255) DEFAULT 'system';
    END IF;
    
    -- Add last_modified_by column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'last_modified_by') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN last_modified_by VARCHAR(255) DEFAULT 'system';
    END IF;
    
    -- Add tags column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'drug_rule_packs' AND column_name = 'tags') THEN
        ALTER TABLE drug_rule_packs ADD COLUMN tags TEXT[] DEFAULT '{}';
    END IF;
END $$;

-- ============================================================================
-- STEP 2: Migrate existing data to new format
-- ============================================================================

-- Update existing records to populate json_content from content
UPDATE drug_rule_packs 
SET 
    original_format = 'json',
    json_content = content,
    deployment_status = '{"staging": "deployed", "production": "deployed"}',
    version_history = '[]',
    tags = '{}'
WHERE json_content IS NULL;

-- ============================================================================
-- STEP 3: Create drug_rule_snapshots table for version history
-- ============================================================================

CREATE TABLE IF NOT EXISTS drug_rule_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    snapshot_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    content_snapshot JSONB NOT NULL,
    toml_snapshot TEXT,
    created_by VARCHAR(255) NOT NULL,
    reason VARCHAR(500)
);

-- ============================================================================
-- STEP 4: Create optimized indexes for performance
-- ============================================================================

-- Indexes for TOML support
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_format 
ON drug_rule_packs(original_format);

CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_drug_version 
ON drug_rule_packs(drug_id, version);

-- Indexes for deployment tracking
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_deployment 
ON drug_rule_packs USING GIN (deployment_status);

-- Indexes for audit and search
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_tags 
ON drug_rule_packs USING GIN (tags);

CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_created_by 
ON drug_rule_packs(created_by);

CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_modified_by 
ON drug_rule_packs(last_modified_by);

-- Indexes for JSON content search
CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_json_content 
ON drug_rule_packs USING GIN (json_content);

-- Indexes for snapshots table
CREATE INDEX IF NOT EXISTS idx_drug_rule_snapshots_drug_version 
ON drug_rule_snapshots(drug_id, version);

CREATE INDEX IF NOT EXISTS idx_drug_rule_snapshots_date 
ON drug_rule_snapshots(snapshot_date DESC);

CREATE INDEX IF NOT EXISTS idx_drug_rule_snapshots_created_by 
ON drug_rule_snapshots(created_by);

-- ============================================================================
-- STEP 5: Add constraints and validation
-- ============================================================================

-- Add format constraint (only if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_original_format') THEN
        ALTER TABLE drug_rule_packs 
        ADD CONSTRAINT chk_original_format 
        CHECK (original_format IN ('toml', 'json'));
    END IF;
END $$;

-- Add unique constraint for drug_id + version (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'drug_rule_packs_drug_id_version_key'
    ) THEN
        ALTER TABLE drug_rule_packs 
        ADD CONSTRAINT drug_rule_packs_drug_id_version_key 
        UNIQUE (drug_id, version);
    END IF;
END $$;

-- ============================================================================
-- STEP 6: Create functions and triggers for automatic snapshot creation
-- ============================================================================

-- Function to create automatic snapshots on version updates
CREATE OR REPLACE FUNCTION create_rule_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    -- Create snapshot of old version before update
    IF TG_OP = 'UPDATE' AND (OLD.version != NEW.version OR OLD.json_content != NEW.json_content) THEN
        INSERT INTO drug_rule_snapshots (
            drug_id, 
            version, 
            content_snapshot, 
            toml_snapshot, 
            created_by, 
            reason
        ) VALUES (
            OLD.drug_id, 
            OLD.version, 
            COALESCE(OLD.json_content, OLD.content), 
            OLD.toml_content,
            NEW.last_modified_by, 
            CASE 
                WHEN OLD.version != NEW.version THEN 'Version update: ' || OLD.version || ' -> ' || NEW.version
                ELSE 'Content update for version ' || OLD.version
            END
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic snapshots (drop first if exists)
DROP TRIGGER IF EXISTS trigger_create_rule_snapshot ON drug_rule_packs;
CREATE TRIGGER trigger_create_rule_snapshot
    BEFORE UPDATE ON drug_rule_packs
    FOR EACH ROW
    EXECUTE FUNCTION create_rule_snapshot();

-- ============================================================================
-- STEP 7: Create helper functions for version management
-- ============================================================================

-- Function to get latest version of a drug
CREATE OR REPLACE FUNCTION get_latest_drug_version(p_drug_id VARCHAR)
RETURNS VARCHAR AS $$
DECLARE
    latest_version VARCHAR;
BEGIN
    SELECT version INTO latest_version
    FROM drug_rule_packs
    WHERE drug_id = p_drug_id
    ORDER BY updated_at DESC
    LIMIT 1;
    
    RETURN latest_version;
END;
$$ LANGUAGE plpgsql;

-- Function to check if version exists
CREATE OR REPLACE FUNCTION version_exists(p_drug_id VARCHAR, p_version VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM drug_rule_packs 
        WHERE drug_id = p_drug_id AND version = p_version
    );
END;
$$ LANGUAGE plpgsql;

COMMIT;

-- ============================================================================
-- VERIFICATION QUERIES (for testing)
-- ============================================================================

-- Verify new columns exist
DO $$
BEGIN
    ASSERT (SELECT COUNT(*) FROM information_schema.columns 
            WHERE table_name = 'drug_rule_packs' AND column_name = 'original_format') = 1,
           'original_format column not created';
    
    ASSERT (SELECT COUNT(*) FROM information_schema.columns 
            WHERE table_name = 'drug_rule_packs' AND column_name = 'toml_content') = 1,
           'toml_content column not created';
           
    ASSERT (SELECT COUNT(*) FROM information_schema.columns 
            WHERE table_name = 'drug_rule_packs' AND column_name = 'version_history') = 1,
           'version_history column not created';
    
    RAISE NOTICE 'Migration 004 completed successfully - All columns created';
END $$;

-- Verify indexes exist
DO $$
BEGIN
    ASSERT (SELECT COUNT(*) FROM pg_indexes 
            WHERE tablename = 'drug_rule_packs' AND indexname = 'idx_drug_rule_packs_format') = 1,
           'Format index not created';
           
    ASSERT (SELECT COUNT(*) FROM pg_indexes 
            WHERE tablename = 'drug_rule_snapshots' AND indexname = 'idx_drug_rule_snapshots_drug_version') = 1,
           'Snapshot index not created';
    
    RAISE NOTICE 'Migration 004 completed successfully - All indexes created';
END $$;
