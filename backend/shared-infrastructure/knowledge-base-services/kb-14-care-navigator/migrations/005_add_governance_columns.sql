-- KB-14 Care Navigator: Governance Columns Migration
-- Migration: 005_add_governance_columns
-- Description: Adds Tier-7 governance compliance columns to tasks table

-- Add governance fields for compliance tracking
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS reason_code VARCHAR(50);
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS reason_text TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS clinical_justification TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS intelligence_id UUID;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS last_audit_at TIMESTAMPTZ;

-- Add index for intelligence_id lookups
CREATE INDEX IF NOT EXISTS idx_tasks_intelligence_id ON tasks(intelligence_id);

-- Add comments for documentation
COMMENT ON COLUMN tasks.reason_code IS 'Governance reason code for state changes (decline, cancel, escalate)';
COMMENT ON COLUMN tasks.reason_text IS 'Human-readable reason text for state changes';
COMMENT ON COLUMN tasks.clinical_justification IS 'Clinical justification for governance compliance';
COMMENT ON COLUMN tasks.intelligence_id IS 'Reference to KB intelligence source (alerts, gaps, etc.)';
COMMENT ON COLUMN tasks.last_audit_at IS 'Timestamp of last governance audit';
