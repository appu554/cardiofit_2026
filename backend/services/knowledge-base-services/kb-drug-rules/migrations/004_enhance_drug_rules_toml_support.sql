-- Migration: Enhanced TOML Support for KB-Drug-Rules Service
-- Version: 004
-- Description: Add TOML format support, versioning, audit trails, and performance optimizations
-- Author: System
-- Date: 2025-01-20

BEGIN;

-- ============================================================================
-- STEP 1: Add new columns for TOML support and enhanced features
-- ============================================================================

-- Add TOML support columns
ALTER TABLE drug_rule_packs 
ADD COLUMN IF NOT EXISTS original_format VARCHAR(10) DEFAULT 'json',
ADD COLUMN IF NOT EXISTS toml_content TEXT,
ADD COLUMN IF NOT EXISTS json_content JSONB;

-- Add versioning support columns
ALTER TABLE drug_rule_packs 
ADD COLUMN IF NOT EXISTS previous_version VARCHAR(50),
ADD COLUMN IF NOT EXISTS version_history JSONB DEFAULT '[]';

-- Add deployment tracking columns
ALTER TABLE drug_rule_packs 
ADD COLUMN IF NOT EXISTS deployment_status JSONB DEFAULT '{}';

-- Add audit columns
ALTER TABLE drug_rule_packs 
ADD COLUMN IF NOT EXISTS created_by VARCHAR(255) DEFAULT 'system',
ADD COLUMN IF NOT EXISTS last_modified_by VARCHAR(255) DEFAULT 'system',
ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- ============================================================================
-- STEP 2: Migrate existing data to new format
-- ============================================================================

-- Update existing records to new format
UPDATE drug_rule_packs 
SET 
    original_format = 'json',
    json_content = content::jsonb,
    deployment_status = '{"staging": "deployed", "production": "deployed"}',
    version_history = '[]',
    tags = '{}'
WHERE original_format IS NULL;

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

-- Add format constraint
ALTER TABLE drug_rule_packs 
ADD CONSTRAINT IF NOT EXISTS chk_original_format 
CHECK (original_format IN ('toml', 'json'));

-- Ensure json_content is not null for new records
ALTER TABLE drug_rule_packs 
ALTER COLUMN json_content SET NOT NULL;

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
            OLD.json_content, 
            OLD.toml_content,
            NEW.last_modified_by, 
            CASE 
                WHEN OLD.version != NEW.version THEN 'Version update: ' || OLD.version || ' -> ' || NEW.version
                ELSE 'Content update for version ' || OLD.version
            END
        );
        
        -- Update version history
        NEW.version_history = COALESCE(OLD.version_history, '[]'::jsonb) || 
            jsonb_build_object(
                'version', OLD.version,
                'modified_date', OLD.updated_at,
                'modified_by', OLD.last_modified_by,
                'change_summary', 'Automated snapshot before update',
                'snapshot_id', (
                    SELECT id::text FROM drug_rule_snapshots 
                    WHERE drug_id = OLD.drug_id AND version = OLD.version 
                    ORDER BY snapshot_date DESC LIMIT 1
                )
            );
            
        -- Keep only last 10 version history entries
        IF jsonb_array_length(NEW.version_history) > 10 THEN
            NEW.version_history = NEW.version_history #> '{-10,-1}';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic snapshots
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
    ORDER BY 
        string_to_array(version, '.')::int[] DESC,
        updated_at DESC
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

-- ============================================================================
-- STEP 8: Create views for common queries
-- ============================================================================

-- View for active drug rules (latest versions)
CREATE OR REPLACE VIEW active_drug_rules AS
SELECT DISTINCT ON (drug_id) 
    id,
    drug_id,
    version,
    original_format,
    json_content,
    toml_content,
    deployment_status,
    created_at,
    updated_at,
    created_by,
    last_modified_by,
    tags
FROM drug_rule_packs
WHERE deployment_status->>'production' = 'deployed'
ORDER BY drug_id, 
         string_to_array(version, '.')::int[] DESC,
         updated_at DESC;

-- View for version history with snapshots
CREATE OR REPLACE VIEW drug_version_history AS
SELECT 
    drp.drug_id,
    drp.version,
    drp.updated_at as version_date,
    drp.last_modified_by,
    drs.id as snapshot_id,
    drs.snapshot_date,
    drs.reason as change_reason
FROM drug_rule_packs drp
LEFT JOIN drug_rule_snapshots drs ON drp.drug_id = drs.drug_id AND drp.version = drs.version
ORDER BY drp.drug_id, drp.updated_at DESC;

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
