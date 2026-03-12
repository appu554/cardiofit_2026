-- Clinical Policy Engine Database Schema
-- KB-7 Terminology Service Phase 1 Implementation
-- Policy validation and enforcement system

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =====================================================
-- POLICY ENGINE CORE TABLES
-- =====================================================

-- Policy Rules - Detailed policy rule definitions
CREATE TABLE IF NOT EXISTS policy_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id VARCHAR(100) UNIQUE NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    rule_version VARCHAR(20) NOT NULL,

    -- Rule classification
    rule_type VARCHAR(50) NOT NULL, -- 'validation', 'approval', 'notification', 'blocking'
    rule_category VARCHAR(50) NOT NULL, -- 'safety', 'quality', 'compliance', 'operational'
    priority INTEGER NOT NULL DEFAULT 100, -- Lower numbers = higher priority

    -- Rule definition
    rule_description TEXT NOT NULL,
    rule_expression JSONB NOT NULL, -- JSONLogic or similar rule expression
    rule_conditions JSONB, -- Additional conditions

    -- Scope and triggers
    trigger_events JSONB NOT NULL, -- Event types that trigger this rule
    resource_types JSONB, -- Resource types this rule applies to
    clinical_domains JSONB, -- Clinical domains this rule covers

    -- Actions
    action_type VARCHAR(50) NOT NULL, -- 'allow', 'warn', 'block', 'require_review'
    action_parameters JSONB,
    escalation_rules JSONB,

    -- Lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'draft', 'active', 'deprecated', 'retired'
    effective_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiry_date TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    metadata JSONB,

    CONSTRAINT valid_rule_type CHECK (rule_type IN ('validation', 'approval', 'notification', 'blocking')),
    CONSTRAINT valid_rule_category CHECK (rule_category IN ('safety', 'quality', 'compliance', 'operational')),
    CONSTRAINT valid_action_type CHECK (action_type IN ('allow', 'warn', 'block', 'require_review')),
    CONSTRAINT valid_rule_status CHECK (status IN ('draft', 'active', 'deprecated', 'retired')),
    CONSTRAINT positive_priority CHECK (priority > 0),
    CONSTRAINT valid_date_range CHECK (expiry_date IS NULL OR expiry_date > effective_date)
);

-- Policy Rule Sets - Group related rules
CREATE TABLE IF NOT EXISTS policy_rule_sets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_set_id VARCHAR(100) UNIQUE NOT NULL,
    rule_set_name VARCHAR(255) NOT NULL,
    rule_set_version VARCHAR(20) NOT NULL,

    -- Rule set details
    description TEXT NOT NULL,
    rule_set_type VARCHAR(50) NOT NULL, -- 'clinical_safety', 'data_quality', 'compliance'

    -- Scope
    applicable_domains JSONB,
    applicable_resources JSONB,

    -- Configuration
    evaluation_order VARCHAR(20) NOT NULL DEFAULT 'priority', -- 'priority', 'sequential', 'parallel'
    stop_on_first_match BOOLEAN DEFAULT FALSE,
    default_action VARCHAR(50) DEFAULT 'allow',

    -- Lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    effective_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiry_date TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    metadata JSONB,

    CONSTRAINT valid_rule_set_type CHECK (rule_set_type IN ('clinical_safety', 'data_quality', 'compliance', 'operational')),
    CONSTRAINT valid_evaluation_order CHECK (evaluation_order IN ('priority', 'sequential', 'parallel')),
    CONSTRAINT valid_default_action CHECK (default_action IN ('allow', 'warn', 'block', 'require_review')),
    CONSTRAINT valid_rule_set_status CHECK (status IN ('draft', 'active', 'deprecated', 'retired'))
);

-- Link rules to rule sets
CREATE TABLE IF NOT EXISTS policy_rule_set_rules (
    rule_set_id UUID REFERENCES policy_rule_sets(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES policy_rules(id) ON DELETE CASCADE,
    execution_order INTEGER NOT NULL,
    is_mandatory BOOLEAN DEFAULT TRUE,
    rule_parameters JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (rule_set_id, rule_id),
    CONSTRAINT positive_execution_order CHECK (execution_order > 0)
);

-- =====================================================
-- POLICY EVALUATION AND EXECUTION
-- =====================================================

-- Policy Evaluations - Track policy rule evaluations
CREATE TABLE IF NOT EXISTS policy_evaluations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evaluation_id VARCHAR(255) UNIQUE NOT NULL,

    -- Evaluation context
    evaluation_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rule_set_id UUID REFERENCES policy_rule_sets(id) NOT NULL,
    rule_id UUID REFERENCES policy_rules(id),

    -- Input context
    event_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    actor_id VARCHAR(100) NOT NULL,

    -- Evaluation input
    input_data JSONB NOT NULL,
    context_data JSONB,

    -- Evaluation result
    evaluation_result VARCHAR(20) NOT NULL, -- 'allow', 'warn', 'block', 'require_review', 'error'
    rule_matched BOOLEAN NOT NULL,
    match_confidence DECIMAL(3,2), -- 0.00 to 1.00

    -- Decision details
    decision_reason TEXT,
    triggered_actions JSONB,
    escalation_triggered BOOLEAN DEFAULT FALSE,

    -- Performance metrics
    evaluation_duration_ms INTEGER,

    -- References
    audit_event_id UUID, -- Link to audit event if available
    audit_session_id UUID, -- Link to audit session if available

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_evaluation_result CHECK (evaluation_result IN ('allow', 'warn', 'block', 'require_review', 'error')),
    CONSTRAINT valid_match_confidence CHECK (match_confidence IS NULL OR (match_confidence >= 0.00 AND match_confidence <= 1.00))
);

-- Policy Actions - Track actions taken based on policy decisions
CREATE TABLE IF NOT EXISTS policy_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    action_id VARCHAR(255) UNIQUE NOT NULL,

    -- Action context
    evaluation_id UUID REFERENCES policy_evaluations(id) NOT NULL,
    action_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Action details
    action_type VARCHAR(50) NOT NULL, -- 'notification', 'block_request', 'require_approval', 'log_warning'
    action_target VARCHAR(100) NOT NULL, -- 'user', 'system', 'reviewer', 'administrator'
    action_parameters JSONB,

    -- Execution
    execution_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'executed', 'failed', 'cancelled'
    execution_result JSONB,
    execution_error TEXT,
    execution_timestamp TIMESTAMPTZ,

    -- Follow-up
    follow_up_required BOOLEAN DEFAULT FALSE,
    follow_up_deadline TIMESTAMPTZ,
    follow_up_status VARCHAR(20), -- 'pending', 'completed', 'overdue'

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_action_type CHECK (action_type IN ('notification', 'block_request', 'require_approval', 'log_warning', 'escalate', 'audit_log')),
    CONSTRAINT valid_execution_status CHECK (execution_status IN ('pending', 'executed', 'failed', 'cancelled')),
    CONSTRAINT valid_follow_up_status CHECK (follow_up_status IS NULL OR follow_up_status IN ('pending', 'completed', 'overdue'))
);

-- =====================================================
-- CLINICAL SAFETY POLICIES
-- =====================================================

-- Clinical Safety Rules - Specific clinical safety rules
CREATE TABLE IF NOT EXISTS clinical_safety_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    safety_rule_id VARCHAR(100) UNIQUE NOT NULL,
    rule_name VARCHAR(255) NOT NULL,

    -- Clinical context
    clinical_domain VARCHAR(100) NOT NULL, -- 'medication', 'allergy', 'diagnosis', 'lab'
    safety_category VARCHAR(50) NOT NULL, -- 'drug_interaction', 'dosing', 'contraindication', 'allergy'
    risk_level VARCHAR(20) NOT NULL, -- 'low', 'moderate', 'high', 'critical'

    -- Rule definition
    rule_description TEXT NOT NULL,
    clinical_logic JSONB NOT NULL, -- Clinical decision logic
    trigger_conditions JSONB NOT NULL,

    -- Safety parameters
    patient_impact_level VARCHAR(20) NOT NULL, -- 'minimal', 'moderate', 'significant', 'severe'
    required_reviewer_level VARCHAR(50) NOT NULL, -- 'clinical_specialist', 'clinical_lead', 'medical_director'

    -- Evidence and references
    evidence_level VARCHAR(20), -- 'expert_opinion', 'case_series', 'rct', 'systematic_review'
    clinical_references JSONB,
    regulatory_requirements JSONB,

    -- Lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    effective_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    review_date TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    reviewed_by VARCHAR(100),
    metadata JSONB,

    CONSTRAINT valid_safety_category CHECK (safety_category IN ('drug_interaction', 'dosing', 'contraindication', 'allergy', 'pregnancy', 'pediatric', 'geriatric')),
    CONSTRAINT valid_risk_level CHECK (risk_level IN ('low', 'moderate', 'high', 'critical')),
    CONSTRAINT valid_patient_impact CHECK (patient_impact_level IN ('minimal', 'moderate', 'significant', 'severe')),
    CONSTRAINT valid_reviewer_level CHECK (required_reviewer_level IN ('clinical_specialist', 'clinical_lead', 'medical_director')),
    CONSTRAINT valid_evidence_level CHECK (evidence_level IS NULL OR evidence_level IN ('expert_opinion', 'case_series', 'cohort_study', 'rct', 'systematic_review')),
    CONSTRAINT valid_safety_status CHECK (status IN ('draft', 'active', 'under_review', 'deprecated', 'retired'))
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Policy Rules indexes
CREATE INDEX IF NOT EXISTS idx_policy_rules_status ON policy_rules (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_policy_rules_type_category ON policy_rules (rule_type, rule_category);
CREATE INDEX IF NOT EXISTS idx_policy_rules_priority ON policy_rules (priority ASC);
CREATE INDEX IF NOT EXISTS idx_policy_rules_domains ON policy_rules USING GIN (clinical_domains);
CREATE INDEX IF NOT EXISTS idx_policy_rules_resources ON policy_rules USING GIN (resource_types);
CREATE INDEX IF NOT EXISTS idx_policy_rules_triggers ON policy_rules USING GIN (trigger_events);
CREATE INDEX IF NOT EXISTS idx_policy_rules_effective ON policy_rules (effective_date, expiry_date);

-- Policy Rule Sets indexes
CREATE INDEX IF NOT EXISTS idx_policy_rule_sets_status ON policy_rule_sets (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_policy_rule_sets_type ON policy_rule_sets (rule_set_type);
CREATE INDEX IF NOT EXISTS idx_policy_rule_sets_domains ON policy_rule_sets USING GIN (applicable_domains);

-- Policy Evaluations indexes
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_timestamp ON policy_evaluations (evaluation_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_result ON policy_evaluations (evaluation_result);
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_rule_set ON policy_evaluations (rule_set_id);
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_resource ON policy_evaluations (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_actor ON policy_evaluations (actor_id);
CREATE INDEX IF NOT EXISTS idx_policy_evaluations_event_type ON policy_evaluations (event_type);

-- Policy Actions indexes
CREATE INDEX IF NOT EXISTS idx_policy_actions_status ON policy_actions (execution_status);
CREATE INDEX IF NOT EXISTS idx_policy_actions_type ON policy_actions (action_type);
CREATE INDEX IF NOT EXISTS idx_policy_actions_timestamp ON policy_actions (action_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_policy_actions_follow_up ON policy_actions (follow_up_required, follow_up_status) WHERE follow_up_required = TRUE;

-- Clinical Safety Rules indexes
CREATE INDEX IF NOT EXISTS idx_clinical_safety_rules_domain ON clinical_safety_rules (clinical_domain);
CREATE INDEX IF NOT EXISTS idx_clinical_safety_rules_category ON clinical_safety_rules (safety_category);
CREATE INDEX IF NOT EXISTS idx_clinical_safety_rules_risk ON clinical_safety_rules (risk_level);
CREATE INDEX IF NOT EXISTS idx_clinical_safety_rules_status ON clinical_safety_rules (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_clinical_safety_rules_reviewer ON clinical_safety_rules (required_reviewer_level);

-- =====================================================
-- FUNCTIONS AND TRIGGERS
-- =====================================================

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_policy_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update timestamp trigger to relevant tables
DROP TRIGGER IF EXISTS policy_rules_updated_at ON policy_rules;
CREATE TRIGGER policy_rules_updated_at
    BEFORE UPDATE ON policy_rules
    FOR EACH ROW EXECUTE FUNCTION update_policy_timestamp();

DROP TRIGGER IF EXISTS policy_rule_sets_updated_at ON policy_rule_sets;
CREATE TRIGGER policy_rule_sets_updated_at
    BEFORE UPDATE ON policy_rule_sets
    FOR EACH ROW EXECUTE FUNCTION update_policy_timestamp();

DROP TRIGGER IF EXISTS clinical_safety_rules_updated_at ON clinical_safety_rules;
CREATE TRIGGER clinical_safety_rules_updated_at
    BEFORE UPDATE ON clinical_safety_rules
    FOR EACH ROW EXECUTE FUNCTION update_policy_timestamp();

-- =====================================================
-- INITIAL POLICY DATA
-- =====================================================

-- Insert core rule sets
INSERT INTO policy_rule_sets (
    rule_set_id, rule_set_name, rule_set_version, description,
    rule_set_type, applicable_domains, applicable_resources,
    evaluation_order, created_by
) VALUES
(
    'clinical-safety-core',
    'Core Clinical Safety Rules',
    '1.0.0',
    'Essential clinical safety rules for patient safety',
    'clinical_safety',
    '["medication", "allergy", "diagnosis"]',
    '["terminology", "mapping", "api"]',
    'priority',
    'system'
),
(
    'medication-safety',
    'Medication Safety Rules',
    '1.0.0',
    'Medication-specific safety and validation rules',
    'clinical_safety',
    '["medication"]',
    '["terminology", "mapping"]',
    'priority',
    'system'
),
(
    'allergy-safety',
    'Allergy Management Safety Rules',
    '1.0.0',
    'Allergy and adverse reaction safety rules',
    'clinical_safety',
    '["allergy"]',
    '["terminology", "mapping", "api"]',
    'priority',
    'system'
),
(
    'data-quality',
    'Data Quality Validation Rules',
    '1.0.0',
    'Data quality and consistency validation rules',
    'data_quality',
    '["medication", "allergy", "diagnosis", "lab"]',
    '["terminology", "mapping", "model"]',
    'parallel',
    'system'
)
ON CONFLICT (rule_set_id) DO NOTHING;

-- Insert core policy rules
INSERT INTO policy_rules (
    rule_id, rule_name, rule_version, rule_type, rule_category,
    priority, rule_description, rule_expression, trigger_events,
    resource_types, clinical_domains, action_type, created_by
) VALUES
(
    'medication-high-risk-check',
    'High-Risk Medication Change Detection',
    '1.0.0',
    'validation',
    'safety',
    10,
    'Detects changes to high-risk medication terminologies requiring clinical review',
    '{"and": [{"in": [{"var": "clinical_domain"}, ["medication"]]}, {"or": [{"in": [{"var": "concept_type"}, ["narcotic", "controlled_substance", "high_alert"]]}, {">": [{"var": "risk_score"}, 75]}]}]}',
    '["terminology_change", "mapping_change"]',
    '["terminology", "mapping"]',
    '["medication"]',
    'require_review',
    'system'
),
(
    'allergy-terminology-validation',
    'Allergy Terminology Validation',
    '1.0.0',
    'validation',
    'safety',
    5,
    'Validates all allergy and adverse reaction terminology changes',
    '{"and": [{"in": [{"var": "clinical_domain"}, ["allergy"]]}, {"or": [{"==": [{"var": "change_type"}, "create"]}, {"==": [{"var": "change_type"}, "update"]}, {"==": [{"var": "change_type"}, "map"]}]}]}',
    '["terminology_change", "mapping_change", "api_change"]',
    '["terminology", "mapping", "api"]',
    '["allergy"]',
    'require_review',
    'system'
),
(
    'critical-system-change-block',
    'Critical System Change Blocker',
    '1.0.0',
    'blocking',
    'safety',
    1,
    'Blocks changes that affect critical patient safety systems without proper approval',
    '{"and": [{">=": [{"var": "safety_score"}, 90]}, {"==": [{"var": "approval_status"}, "pending"]}]}',
    '["api_change", "model_change", "system_change"]',
    '["api", "model", "system"]',
    '["medication", "allergy", "diagnosis"]',
    'block',
    'system'
),
(
    'terminology-consistency-check',
    'Terminology Consistency Validation',
    '1.0.0',
    'validation',
    'quality',
    50,
    'Ensures terminology changes maintain consistency across systems',
    '{"and": [{"in": [{"var": "change_type"}, ["create", "update", "map"]]}, {"!=": [{"var": "consistency_score"}, null]}]}',
    '["terminology_change", "mapping_change"]',
    '["terminology", "mapping"]',
    '["medication", "allergy", "diagnosis", "lab"]',
    'warn',
    'system'
),
(
    'deprecation-safety-check',
    'Terminology Deprecation Safety Check',
    '1.0.0',
    'validation',
    'safety',
    20,
    'Ensures deprecated terminologies are safely handled',
    '{"and": [{"==": [{"var": "change_type"}, "deprecate"]}, {"or": [{"in": [{"var": "clinical_domain"}, ["medication", "allergy"]]}, {">": [{"var": "usage_count"}, 100]}]}]}',
    '["terminology_change"]',
    '["terminology"]',
    '["medication", "allergy", "diagnosis"]',
    'require_review',
    'system'
)
ON CONFLICT (rule_id) DO NOTHING;

-- Link rules to rule sets
INSERT INTO policy_rule_set_rules (rule_set_id, rule_id, execution_order)
SELECT
    rs.id,
    r.id,
    ROW_NUMBER() OVER (PARTITION BY rs.id ORDER BY r.priority)
FROM policy_rule_sets rs
CROSS JOIN policy_rules r
WHERE (rs.rule_set_id = 'clinical-safety-core' AND r.rule_category = 'safety')
   OR (rs.rule_set_id = 'medication-safety' AND r.rule_id LIKE '%medication%')
   OR (rs.rule_set_id = 'allergy-safety' AND r.rule_id LIKE '%allergy%')
   OR (rs.rule_set_id = 'data-quality' AND r.rule_category = 'quality')
ON CONFLICT (rule_set_id, rule_id) DO NOTHING;

-- Insert clinical safety rules
INSERT INTO clinical_safety_rules (
    safety_rule_id, rule_name, clinical_domain, safety_category,
    risk_level, rule_description, clinical_logic, trigger_conditions,
    patient_impact_level, required_reviewer_level, evidence_level,
    created_by
) VALUES
(
    'narcotic-medication-safety',
    'Narcotic Medication Safety Rule',
    'medication',
    'drug_interaction',
    'critical',
    'Requires medical director approval for narcotic medication terminology changes',
    '{"and": [{"in": [{"var": "drug_class"}, ["narcotic", "opioid", "controlled_substance"]]}, {"or": [{"==": [{"var": "change_type"}, "create"]}, {"==": [{"var": "change_type"}, "update"]}]}]}',
    '{"change_types": ["create", "update", "map"], "drug_classes": ["narcotic", "opioid", "controlled_substance"]}',
    'severe',
    'medical_director',
    'systematic_review',
    'system'
),
(
    'allergy-cross-reaction-safety',
    'Allergy Cross-Reaction Safety Rule',
    'allergy',
    'allergy',
    'high',
    'Validates cross-reaction mappings for allergy terminologies',
    '{"and": [{"==": [{"var": "terminology_type"}, "allergy"]}, {"!=": [{"var": "cross_reactions"}, null]}]}',
    '{"change_types": ["create", "update", "map"], "has_cross_reactions": true}',
    'significant',
    'clinical_lead',
    'rct',
    'system'
),
(
    'pediatric-dosing-safety',
    'Pediatric Dosing Safety Rule',
    'medication',
    'dosing',
    'high',
    'Special validation for pediatric medication dosing terminologies',
    '{"and": [{"==": [{"var": "patient_population"}, "pediatric"]}, {"in": [{"var": "dosing_parameters"}, ["weight_based", "age_based", "bsa_based"]]}]}',
    '{"patient_populations": ["pediatric", "neonatal"], "dosing_types": ["weight_based", "age_based", "bsa_based"]}',
    'significant',
    'clinical_lead',
    'systematic_review',
    'system'
),
(
    'pregnancy-safety-category',
    'Pregnancy Safety Category Rule',
    'medication',
    'pregnancy',
    'critical',
    'Validates pregnancy safety categories for medication terminologies',
    '{"and": [{"!=": [{"var": "pregnancy_category"}, null]}, {"in": [{"var": "pregnancy_category"}, ["D", "X"]]}]}',
    '{"pregnancy_categories": ["D", "X"], "requires_teratogenicity_data": true}',
    'severe',
    'medical_director',
    'systematic_review',
    'system'
)
ON CONFLICT (safety_rule_id) DO NOTHING;

-- Create useful views
CREATE OR REPLACE VIEW v_active_policy_rules AS
SELECT
    r.*,
    COALESCE(array_agg(DISTINCT rs.rule_set_name) FILTER (WHERE rs.rule_set_name IS NOT NULL), '{}') as rule_sets
FROM policy_rules r
LEFT JOIN policy_rule_set_rules rsr ON r.id = rsr.rule_id
LEFT JOIN policy_rule_sets rs ON rsr.rule_set_id = rs.id AND rs.status = 'active'
WHERE r.status = 'active'
  AND (r.expiry_date IS NULL OR r.expiry_date > NOW())
GROUP BY r.id
ORDER BY r.priority ASC;

CREATE OR REPLACE VIEW v_policy_evaluation_summary AS
SELECT
    DATE_TRUNC('hour', evaluation_timestamp) as evaluation_hour,
    rule_set_id,
    evaluation_result,
    COUNT(*) as evaluation_count,
    AVG(evaluation_duration_ms) as avg_duration_ms,
    COUNT(*) FILTER (WHERE escalation_triggered = TRUE) as escalations_triggered
FROM policy_evaluations
WHERE evaluation_timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', evaluation_timestamp), rule_set_id, evaluation_result
ORDER BY evaluation_hour DESC;

CREATE OR REPLACE VIEW v_clinical_safety_rules_summary AS
SELECT
    clinical_domain,
    safety_category,
    risk_level,
    COUNT(*) as rule_count,
    COUNT(*) FILTER (WHERE status = 'active') as active_rules,
    COUNT(*) FILTER (WHERE review_date < NOW() - INTERVAL '1 year') as rules_needing_review
FROM clinical_safety_rules
GROUP BY clinical_domain, safety_category, risk_level
ORDER BY clinical_domain, risk_level DESC;

-- Grant appropriate permissions
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO kb_policy_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_readonly_user;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO kb_policy_user;