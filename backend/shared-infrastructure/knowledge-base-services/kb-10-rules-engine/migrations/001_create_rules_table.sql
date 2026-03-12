-- KB-10 Clinical Rules Engine - Rules Table Migration
-- Version: 1.0.0
-- Date: 2025-01-05

-- Rules table - stores configurable clinical rules
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    priority INTEGER DEFAULT 100,
    version VARCHAR(50),
    conditions JSONB NOT NULL,
    condition_logic VARCHAR(10) DEFAULT 'AND',
    actions JSONB NOT NULL,
    evidence JSONB,
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_rules_type ON rules(type);
CREATE INDEX IF NOT EXISTS idx_rules_category ON rules(category);
CREATE INDEX IF NOT EXISTS idx_rules_severity ON rules(severity);
CREATE INDEX IF NOT EXISTS idx_rules_status ON rules(status);
CREATE INDEX IF NOT EXISTS idx_rules_tags ON rules USING GIN(tags);

-- Rule types:
-- ALERT - Critical value alerts (e.g., K > 6.5)
-- INFERENCE - Clinical inference rules (e.g., sepsis detection)
-- VALIDATION - Data validation rules
-- ESCALATION - Escalation pathway rules
-- SUPPRESSION - Alert suppression rules
-- DERIVATION - Calculated value derivation (e.g., AKI stage)
-- RECOMMENDATION - Clinical recommendations
-- CONFLICT - Conflict detection rules

COMMENT ON TABLE rules IS 'KB-10 Clinical Rules Engine - configurable YAML-driven business rules';
COMMENT ON COLUMN rules.type IS 'Rule type: ALERT, INFERENCE, VALIDATION, ESCALATION, SUPPRESSION, DERIVATION, RECOMMENDATION, CONFLICT';
COMMENT ON COLUMN rules.conditions IS 'JSONB array of conditions with field, operator, value, unit';
COMMENT ON COLUMN rules.condition_logic IS 'Logic for combining conditions: AND, OR, or complex expression like ((1 AND 2) OR 3)';
COMMENT ON COLUMN rules.actions IS 'JSONB array of actions to execute when rule triggers';
