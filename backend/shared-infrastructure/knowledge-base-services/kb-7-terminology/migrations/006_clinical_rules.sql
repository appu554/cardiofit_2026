-- KB-7 Terminology Service - Clinical Rules Schema
-- Provides persistent storage for CDSS clinical rules with versioning and audit

-- ============================================================================
-- Clinical Rules Table
-- ============================================================================
-- Stores clinical decision support rules that can be managed via API

CREATE TABLE IF NOT EXISTS clinical_rules (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL DEFAULT '1.0',

    -- Clinical metadata
    domain VARCHAR(50) NOT NULL,              -- sepsis, renal, cardiac, diabetes, respiratory, medication, general
    severity VARCHAR(20) NOT NULL,            -- critical, high, moderate, low, informational
    category VARCHAR(50) NOT NULL,            -- diagnosis, threshold, medication, protocol, monitoring

    -- Rule definition (JSON)
    conditions JSONB NOT NULL,                -- Array of RuleCondition objects
    /*
    Example structure:
    [
        {
            "type": "compound",
            "compound_operator": "AND",
            "sub_conditions": [
                {"type": "value_set", "value_set_id": "SepsisDiagnosis"},
                {"type": "threshold", "loinc_code": "2524-7", "operator": ">", "value": 2.0, "unit": "mmol/L"}
            ]
        }
    ]
    */

    -- Alert configuration
    alert_title VARCHAR(500) NOT NULL,
    alert_description TEXT,
    recommendations TEXT[],                   -- Array of recommendation strings
    guideline_references TEXT[],              -- Array of guideline citations

    -- Rule state
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 5,      -- 0 = highest priority

    -- Source tracking
    source VARCHAR(50) DEFAULT 'system',      -- system, user, import
    source_reference VARCHAR(255),            -- Original source (e.g., guideline name)

    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system',
    updated_by VARCHAR(100) DEFAULT 'system'
);

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_clinical_rules_domain ON clinical_rules(domain);
CREATE INDEX IF NOT EXISTS idx_clinical_rules_severity ON clinical_rules(severity);
CREATE INDEX IF NOT EXISTS idx_clinical_rules_enabled ON clinical_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_clinical_rules_priority ON clinical_rules(priority);
CREATE INDEX IF NOT EXISTS idx_clinical_rules_domain_enabled ON clinical_rules(domain, enabled);
CREATE INDEX IF NOT EXISTS idx_clinical_rules_conditions ON clinical_rules USING GIN(conditions);

-- ============================================================================
-- Rule Audit History Table
-- ============================================================================
-- Tracks all changes to clinical rules for compliance and audit

CREATE TABLE IF NOT EXISTS clinical_rules_history (
    history_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id VARCHAR(100) NOT NULL,
    action VARCHAR(20) NOT NULL,              -- INSERT, UPDATE, DELETE

    -- Snapshot of rule at time of change
    rule_snapshot JSONB NOT NULL,

    -- Change metadata
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    changed_by VARCHAR(100) DEFAULT 'system',
    change_reason TEXT,

    -- Reference to current rule (may be null if deleted)
    CONSTRAINT fk_rule_history_rule FOREIGN KEY (rule_id)
        REFERENCES clinical_rules(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_rules_history_rule_id ON clinical_rules_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_rules_history_changed_at ON clinical_rules_history(changed_at);

-- ============================================================================
-- Trigger for updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_clinical_rules_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_clinical_rules_updated_at ON clinical_rules;
CREATE TRIGGER trigger_clinical_rules_updated_at
    BEFORE UPDATE ON clinical_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_clinical_rules_timestamp();

-- ============================================================================
-- Trigger for Audit History
-- ============================================================================

CREATE OR REPLACE FUNCTION audit_clinical_rules_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO clinical_rules_history (rule_id, action, rule_snapshot, changed_by)
        VALUES (NEW.id, 'INSERT', to_jsonb(NEW), COALESCE(NEW.created_by, 'system'));
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO clinical_rules_history (rule_id, action, rule_snapshot, changed_by)
        VALUES (NEW.id, 'UPDATE', to_jsonb(NEW), COALESCE(NEW.updated_by, 'system'));
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO clinical_rules_history (rule_id, action, rule_snapshot, changed_by)
        VALUES (OLD.id, 'DELETE', to_jsonb(OLD), 'system');
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_clinical_rules_audit ON clinical_rules;
CREATE TRIGGER trigger_clinical_rules_audit
    AFTER INSERT OR UPDATE OR DELETE ON clinical_rules
    FOR EACH ROW
    EXECUTE FUNCTION audit_clinical_rules_changes();

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Get enabled rules for a specific domain
CREATE OR REPLACE FUNCTION get_enabled_rules_by_domain(p_domain VARCHAR)
RETURNS SETOF clinical_rules AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM clinical_rules
    WHERE domain = p_domain AND enabled = true
    ORDER BY priority ASC, created_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Get all enabled rules ordered by priority
CREATE OR REPLACE FUNCTION get_all_enabled_rules()
RETURNS SETOF clinical_rules AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM clinical_rules
    WHERE enabled = true
    ORDER BY priority ASC, domain ASC, created_at ASC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE clinical_rules IS 'CDSS clinical decision support rules with JSON conditions';
COMMENT ON TABLE clinical_rules_history IS 'Audit trail for all clinical rule changes';
COMMENT ON COLUMN clinical_rules.conditions IS 'JSONB array of RuleCondition objects defining when rule fires';
COMMENT ON COLUMN clinical_rules.priority IS 'Rule priority: 0=highest (life-threatening), higher=lower priority';
