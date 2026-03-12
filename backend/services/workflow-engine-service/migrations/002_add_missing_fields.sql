-- Migration: Add missing fields to workflow tables
-- This script adds the missing fields that are causing monitoring errors

-- Add updated_at field to workflow_instances table
ALTER TABLE workflow_instances 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add escalated field to workflow_tasks table  
ALTER TABLE workflow_tasks 
ADD COLUMN IF NOT EXISTS escalated BOOLEAN DEFAULT FALSE;

-- Update existing records to have proper timestamps
UPDATE workflow_instances 
SET updated_at = start_time 
WHERE updated_at IS NULL;

-- Update existing tasks to have escalated = false
UPDATE workflow_tasks 
SET escalated = FALSE 
WHERE escalated IS NULL;

-- Create trigger to automatically update updated_at field for workflow_instances
CREATE OR REPLACE FUNCTION update_workflow_instance_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop trigger if it exists and create new one
DROP TRIGGER IF EXISTS trigger_update_workflow_instance_updated_at ON workflow_instances;
CREATE TRIGGER trigger_update_workflow_instance_updated_at
    BEFORE UPDATE ON workflow_instances
    FOR EACH ROW
    EXECUTE FUNCTION update_workflow_instance_updated_at();

-- Create indexes for the new fields
CREATE INDEX IF NOT EXISTS idx_workflow_instances_updated_at ON workflow_instances(updated_at);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalated ON workflow_tasks(escalated);

-- Add comments for documentation
COMMENT ON COLUMN workflow_instances.updated_at IS 'Timestamp when the workflow instance was last updated';
COMMENT ON COLUMN workflow_tasks.escalated IS 'Flag indicating if the task has been escalated';
