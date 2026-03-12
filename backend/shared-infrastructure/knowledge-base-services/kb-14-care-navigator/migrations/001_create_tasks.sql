-- KB-14 Care Navigator: Tasks Table Migration
-- Migration: 001_create_tasks
-- Description: Creates the tasks table for clinical task management

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(50) NOT NULL UNIQUE,

    -- Task classification
    type VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'CREATED',
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    source VARCHAR(30) NOT NULL,
    source_id VARCHAR(100),

    -- Patient context
    patient_id VARCHAR(50) NOT NULL,
    encounter_id VARCHAR(50),

    -- Task details
    title VARCHAR(200) NOT NULL,
    description TEXT,
    instructions TEXT,
    clinical_note TEXT,

    -- Assignment
    assigned_to UUID,
    assigned_role VARCHAR(50),
    team_id UUID,

    -- SLA & Timing
    due_date TIMESTAMPTZ,
    sla_minutes INTEGER DEFAULT 0,
    escalation_level INTEGER DEFAULT 0,

    -- Completion
    completed_by UUID,
    completed_at TIMESTAMPTZ,
    verified_by UUID,
    verified_at TIMESTAMPTZ,
    outcome VARCHAR(50),

    -- JSONB fields for flexible data
    actions JSONB DEFAULT '[]'::jsonb,
    notes JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ
);

-- Create indexes for common query patterns
CREATE INDEX idx_tasks_task_id ON tasks(task_id);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_source ON tasks(source);
CREATE INDEX idx_tasks_patient_id ON tasks(patient_id);
CREATE INDEX idx_tasks_assigned_to ON tasks(assigned_to);
CREATE INDEX idx_tasks_team_id ON tasks(team_id);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);

-- Composite indexes for worklist queries
CREATE INDEX idx_tasks_assignee_status ON tasks(assigned_to, status);
CREATE INDEX idx_tasks_team_status ON tasks(team_id, status);
CREATE INDEX idx_tasks_status_priority ON tasks(status, priority);
CREATE INDEX idx_tasks_status_due_date ON tasks(status, due_date);
CREATE INDEX idx_tasks_patient_status ON tasks(patient_id, status);

-- Index for overdue task queries
CREATE INDEX idx_tasks_overdue ON tasks(due_date, status)
    WHERE status NOT IN ('COMPLETED', 'VERIFIED', 'CANCELLED');

-- Index for unassigned task queries
CREATE INDEX idx_tasks_unassigned ON tasks(status, created_at)
    WHERE assigned_to IS NULL AND status = 'CREATED';

-- GIN index for JSONB metadata search
CREATE INDEX idx_tasks_metadata ON tasks USING GIN (metadata);

-- Add check constraints for valid enum values
ALTER TABLE tasks ADD CONSTRAINT chk_tasks_status
    CHECK (status IN ('CREATED', 'ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'VERIFIED', 'DECLINED', 'BLOCKED', 'ESCALATED', 'CANCELLED'));

ALTER TABLE tasks ADD CONSTRAINT chk_tasks_priority
    CHECK (priority IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW'));

ALTER TABLE tasks ADD CONSTRAINT chk_tasks_source
    CHECK (source IN ('KB3_TEMPORAL', 'KB9_CARE_GAPS', 'KB12_ORDER_SETS', 'MANUAL'));

-- Create trigger function for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for tasks
CREATE TRIGGER update_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE tasks IS 'Clinical tasks from KB-3 Temporal, KB-9 Care Gaps, KB-12 Order Sets, and manual entry';
COMMENT ON COLUMN tasks.type IS 'Task type: CRITICAL_LAB_REVIEW, MEDICATION_REVIEW, CARE_GAP_CLOSURE, etc.';
COMMENT ON COLUMN tasks.status IS 'Lifecycle: CREATED → ASSIGNED → IN_PROGRESS → COMPLETED → VERIFIED';
COMMENT ON COLUMN tasks.priority IS 'Priority: CRITICAL (1hr SLA), HIGH (4hr), MEDIUM (24hr), LOW (7day)';
COMMENT ON COLUMN tasks.source IS 'Origin: KB3_TEMPORAL, KB9_CARE_GAPS, KB12_ORDER_SETS, MANUAL';
COMMENT ON COLUMN tasks.actions IS 'JSONB array of TaskAction objects for task checklist items';
COMMENT ON COLUMN tasks.notes IS 'JSONB array of TaskNote objects for clinical notes';
COMMENT ON COLUMN tasks.metadata IS 'JSONB for source-specific metadata and extensions';
