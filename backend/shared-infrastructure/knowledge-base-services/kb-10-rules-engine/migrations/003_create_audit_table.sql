-- KB-10 Clinical Rules Engine - Rule Executions Audit Table
-- Version: 1.0.0
-- Date: 2025-01-05

-- Rule executions table - audit trail for all rule evaluations
CREATE TABLE IF NOT EXISTS rule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id VARCHAR(100) NOT NULL,
    rule_name VARCHAR(255),
    patient_id VARCHAR(100) NOT NULL,
    encounter_id VARCHAR(100),
    triggered BOOLEAN NOT NULL,
    context JSONB,
    result JSONB,
    execution_time_ms FLOAT,
    cache_hit BOOLEAN DEFAULT FALSE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying and analytics
CREATE INDEX IF NOT EXISTS idx_executions_rule ON rule_executions(rule_id);
CREATE INDEX IF NOT EXISTS idx_executions_patient ON rule_executions(patient_id);
CREATE INDEX IF NOT EXISTS idx_executions_created ON rule_executions(created_at);
CREATE INDEX IF NOT EXISTS idx_executions_triggered ON rule_executions(triggered);
CREATE INDEX IF NOT EXISTS idx_executions_encounter ON rule_executions(encounter_id);

-- Partitioning recommendation for high-volume deployments:
-- CREATE TABLE rule_executions_2025 PARTITION OF rule_executions
--     FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

COMMENT ON TABLE rule_executions IS 'KB-10 Audit trail for all rule evaluations - used for compliance and analytics';
COMMENT ON COLUMN rule_executions.triggered IS 'Whether the rule conditions evaluated to true';
COMMENT ON COLUMN rule_executions.context IS 'JSONB snapshot of evaluation context (labs, vitals, etc.)';
COMMENT ON COLUMN rule_executions.result IS 'JSONB result of the evaluation including any actions taken';
COMMENT ON COLUMN rule_executions.cache_hit IS 'Whether result was served from cache';
