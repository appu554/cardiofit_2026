-- KB-14 Care Navigator: Escalation Schema Update Migration
-- Migration: 006_add_escalation_columns
-- Description: Adds missing status and resolution columns to escalations table

-- Add status column for escalation workflow
ALTER TABLE escalations ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'PENDING';

-- Add resolution tracking columns
ALTER TABLE escalations ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE escalations ADD COLUMN IF NOT EXISTS resolved_by UUID;
ALTER TABLE escalations ADD COLUMN IF NOT EXISTS resolution VARCHAR(500);

-- Create index for status queries
CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status);

-- Add check constraint for valid status values
ALTER TABLE escalations DROP CONSTRAINT IF EXISTS chk_escalations_status;
ALTER TABLE escalations ADD CONSTRAINT chk_escalations_status
    CHECK (status IN ('PENDING', 'ACKNOWLEDGED', 'RESOLVED'));

-- Add comment
COMMENT ON COLUMN escalations.status IS 'Escalation status: PENDING, ACKNOWLEDGED, RESOLVED';
COMMENT ON COLUMN escalations.resolution IS 'Resolution notes when escalation is resolved';
