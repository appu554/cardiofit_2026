-- KB-14 Care Navigator: Escalations Table Migration
-- Migration: 003_create_escalations
-- Description: Creates the escalations table for SLA tracking and escalation management

-- Create escalations table
CREATE TABLE IF NOT EXISTS escalations (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Task reference
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,

    -- Escalation details
    level INTEGER NOT NULL, -- 0=NONE, 1=WARNING, 2=URGENT, 3=CRITICAL, 4=EXECUTIVE
    reason VARCHAR(500),

    -- Who was notified
    escalated_to UUID,
    escalated_to_role VARCHAR(50),

    -- Notification tracking
    notification_sent BOOLEAN DEFAULT false,
    notification_channel VARCHAR(20), -- in_app, email, sms, push, pager
    notification_sent_at TIMESTAMPTZ,

    -- Acknowledgment
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID,

    -- SLA information at time of escalation
    sla_elapsed_percent DOUBLE PRECISION,
    time_overdue_minutes INTEGER, -- Negative if before SLA

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_escalations_task_id ON escalations(task_id);
CREATE INDEX idx_escalations_level ON escalations(level);
CREATE INDEX idx_escalations_acknowledged ON escalations(acknowledged);
CREATE INDEX idx_escalations_created_at ON escalations(created_at);
CREATE INDEX idx_escalations_escalated_to ON escalations(escalated_to);

-- Composite indexes for common queries
CREATE INDEX idx_escalations_task_level ON escalations(task_id, level DESC);
CREATE INDEX idx_escalations_unacknowledged ON escalations(acknowledged, level DESC)
    WHERE acknowledged = false;

-- Add check constraint for escalation levels
ALTER TABLE escalations ADD CONSTRAINT chk_escalations_level
    CHECK (level >= 0 AND level <= 4);

-- Create trigger for updated_at
CREATE TRIGGER update_escalations_updated_at
    BEFORE UPDATE ON escalations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE escalations IS 'Escalation events for tasks based on SLA thresholds';
COMMENT ON COLUMN escalations.level IS 'Escalation level: 0=NONE, 1=WARNING, 2=URGENT, 3=CRITICAL, 4=EXECUTIVE';
COMMENT ON COLUMN escalations.reason IS 'Human-readable reason for escalation';
COMMENT ON COLUMN escalations.notification_channel IS 'Channel used: in_app, email, sms, push, pager';
COMMENT ON COLUMN escalations.sla_elapsed_percent IS 'Percentage of SLA elapsed at time of escalation';
COMMENT ON COLUMN escalations.time_overdue_minutes IS 'Minutes past SLA (negative if before deadline)';

-- Create view for active escalations with task details
CREATE OR REPLACE VIEW v_active_escalations AS
SELECT
    e.id AS escalation_id,
    e.level,
    e.reason,
    e.sla_elapsed_percent,
    e.time_overdue_minutes,
    e.created_at AS escalated_at,
    e.acknowledged,
    t.id AS task_id,
    t.task_id AS task_number,
    t.type AS task_type,
    t.priority AS task_priority,
    t.title AS task_title,
    t.patient_id,
    t.assigned_to,
    t.team_id,
    t.due_date
FROM escalations e
JOIN tasks t ON e.task_id = t.id
WHERE e.acknowledged = false
  AND t.status NOT IN ('COMPLETED', 'VERIFIED', 'CANCELLED')
ORDER BY e.level DESC, e.created_at ASC;

COMMENT ON VIEW v_active_escalations IS 'Active unacknowledged escalations with task details for monitoring dashboard';

-- Create view for escalation analytics
CREATE OR REPLACE VIEW v_escalation_stats AS
SELECT
    DATE_TRUNC('day', created_at) AS date,
    level,
    COUNT(*) AS total_escalations,
    COUNT(*) FILTER (WHERE acknowledged = true) AS acknowledged_count,
    AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at)) / 60)
        FILTER (WHERE acknowledged = true) AS avg_response_minutes
FROM escalations
GROUP BY DATE_TRUNC('day', created_at), level
ORDER BY date DESC, level;

COMMENT ON VIEW v_escalation_stats IS 'Daily escalation statistics for trend analysis';

-- Create function to get current escalation level for a task
CREATE OR REPLACE FUNCTION get_task_escalation_level(p_task_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_level INTEGER;
BEGIN
    SELECT COALESCE(MAX(level), 0) INTO v_level
    FROM escalations
    WHERE task_id = p_task_id;
    RETURN v_level;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_task_escalation_level IS 'Returns the highest escalation level for a task';

-- Create indexes for SLA-based queries on tasks
CREATE INDEX idx_tasks_escalation_level ON tasks(escalation_level) WHERE escalation_level > 0;

-- Create function to calculate SLA elapsed percentage
CREATE OR REPLACE FUNCTION calculate_sla_elapsed(
    p_created_at TIMESTAMPTZ,
    p_sla_minutes INTEGER
)
RETURNS DOUBLE PRECISION AS $$
DECLARE
    v_elapsed_minutes DOUBLE PRECISION;
BEGIN
    IF p_sla_minutes <= 0 THEN
        RETURN 0;
    END IF;

    v_elapsed_minutes := EXTRACT(EPOCH FROM (NOW() - p_created_at)) / 60;
    RETURN v_elapsed_minutes / p_sla_minutes;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_sla_elapsed IS 'Calculates SLA elapsed percentage for escalation determination';
