-- =============================================================================
-- KB-6 Formulary: Policy Binding Tables
-- Enhancement #1: Program Binding + Policy Anchors
-- =============================================================================

-- Policy Binding Registry
-- Stores policy bindings that can be attached to PA/ST/QL evaluations
CREATE TABLE IF NOT EXISTS policy_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Program Context
    payer_id VARCHAR(50),
    payer_name TEXT,
    program_id VARCHAR(50),
    program_name TEXT,
    program_type VARCHAR(50),  -- Medicare, Medicaid, Commercial, Exchange
    plan_id VARCHAR(50),
    contract_id VARCHAR(100),
    benefit_phase VARCHAR(50), -- For Medicare Part D: deductible, initial, gap, catastrophic

    -- Policy Reference
    policy_id VARCHAR(100) NOT NULL,
    policy_name TEXT NOT NULL,
    policy_version VARCHAR(50) NOT NULL,
    policy_effective_date DATE NOT NULL,
    policy_expiration_date DATE,
    policy_document_url TEXT,
    policy_section_ref TEXT,
    policy_last_review_date DATE,

    -- Policy Type
    policy_type VARCHAR(50) NOT NULL,  -- PRIOR_AUTHORIZATION, STEP_THERAPY, QUANTITY_LIMIT, FORMULARY, etc.

    -- Jurisdiction
    jurisdiction_type VARCHAR(20) NOT NULL,  -- US, INDIA, AUSTRALIA, UK, etc.
    jurisdiction_code VARCHAR(10),  -- State/region code
    jurisdiction_name TEXT,
    jurisdiction_authority TEXT,  -- FDA, CMS, NLEM, PBS, TGA, etc.

    -- Binding Enforcement
    binding_level VARCHAR(30) NOT NULL,  -- FEDERAL, STATE, PAYER, PBM, HOSPITAL, NETWORK, FORMULARY, INSTITUTION
    compliance_enforce_mode VARCHAR(30) NOT NULL DEFAULT 'WARN',  -- HARD_BLOCK, SOFT_BLOCK, WARN, NOTIFY, AUDIT, ADVISORY

    -- Governance Context
    governance_rule_id VARCHAR(100),  -- Reference to Tier-7 Governance Engine rule
    compliance_category VARCHAR(100),  -- Category for compliance reporting
    audit_required BOOLEAN DEFAULT false,

    -- Hierarchy & Precedence
    parent_binding_id UUID REFERENCES policy_bindings(id),
    precedence INTEGER DEFAULT 1,  -- Priority when multiple bindings apply

    -- Override Configuration
    override_allowed BOOLEAN DEFAULT true,
    override_approval_level VARCHAR(50),  -- Required approval level for override

    -- Metadata
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_policy_binding UNIQUE (policy_id, policy_version, payer_id, plan_id)
);

-- Index for fast lookups
CREATE INDEX idx_policy_bindings_payer ON policy_bindings(payer_id, plan_id);
CREATE INDEX idx_policy_bindings_policy ON policy_bindings(policy_id, policy_version);
CREATE INDEX idx_policy_bindings_jurisdiction ON policy_bindings(jurisdiction_type, jurisdiction_authority);
CREATE INDEX idx_policy_bindings_type ON policy_bindings(policy_type);
CREATE INDEX idx_policy_bindings_active ON policy_bindings(is_active) WHERE is_active = true;

-- Policy Violations Log
-- Records all detected policy violations for audit and compliance reporting
CREATE TABLE IF NOT EXISTS policy_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    binding_id UUID REFERENCES policy_bindings(id),

    -- Context
    patient_id VARCHAR(100),
    provider_id VARCHAR(100),
    drug_rxnorm VARCHAR(20),
    drug_name TEXT,

    -- Violation Details
    violation_type VARCHAR(100) NOT NULL,
    violation_code VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,  -- critical, high, medium, low
    enforcement_action VARCHAR(30) NOT NULL,  -- Action taken: HARD_BLOCK, SOFT_BLOCK, WARN, etc.

    -- Resolution
    requires_override BOOLEAN DEFAULT false,
    override_requested BOOLEAN DEFAULT false,
    override_approved BOOLEAN,
    override_by VARCHAR(100),
    override_at TIMESTAMPTZ,
    override_reason TEXT,

    -- Audit
    audit_log_id VARCHAR(100),
    correlation_id VARCHAR(100),  -- For tracing across services
    request_id VARCHAR(100),

    -- Timestamps
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast lookups
CREATE INDEX idx_policy_violations_binding ON policy_violations(binding_id);
CREATE INDEX idx_policy_violations_patient ON policy_violations(patient_id);
CREATE INDEX idx_policy_violations_drug ON policy_violations(drug_rxnorm);
CREATE INDEX idx_policy_violations_type ON policy_violations(violation_type);
CREATE INDEX idx_policy_violations_detected ON policy_violations(detected_at);

-- =============================================================================
-- Enhancement #2: Cross-Service Events Table
-- =============================================================================

-- Event Log for Cross-Service Signals
-- Stores all emitted events for audit, replay, and debugging
CREATE TABLE IF NOT EXISTS formulary_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Event Identity
    event_type VARCHAR(50) NOT NULL,  -- PA_REQUIRED, ST_NON_COMPLIANT, QL_VIOLATION, etc.
    category VARCHAR(50) NOT NULL,     -- PRIOR_AUTHORIZATION, STEP_THERAPY, QUANTITY_LIMIT, OVERRIDE, COVERAGE, GOVERNANCE
    severity VARCHAR(20) NOT NULL,     -- CRITICAL, HIGH, MEDIUM, LOW, INFO

    -- Source
    source_service VARCHAR(50) DEFAULT 'KB6_FORMULARY',
    source_version VARCHAR(20),
    correlation_id VARCHAR(100) NOT NULL,
    request_id VARCHAR(100),

    -- Context (stored as JSONB for flexibility)
    drug_context JSONB,
    patient_context JSONB,
    provider_context JSONB,
    payer_context JSONB,
    policy_binding_id UUID REFERENCES policy_bindings(id),

    -- Event Details
    reason TEXT NOT NULL,
    details JSONB,
    recommendations JSONB,

    -- Target Services
    target_services TEXT[],

    -- Acknowledgment
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(100),

    -- Timestamps
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for event queries
CREATE INDEX idx_formulary_events_type ON formulary_events(event_type);
CREATE INDEX idx_formulary_events_category ON formulary_events(category);
CREATE INDEX idx_formulary_events_correlation ON formulary_events(correlation_id);
CREATE INDEX idx_formulary_events_timestamp ON formulary_events(timestamp);
CREATE INDEX idx_formulary_events_unacked ON formulary_events(acknowledged) WHERE acknowledged = false;

-- =============================================================================
-- Seed Data: Predefined Policy Bindings
-- =============================================================================

-- India NLEM (National List of Essential Medicines)
INSERT INTO policy_bindings (
    policy_id, policy_name, policy_version, policy_effective_date,
    policy_type, jurisdiction_type, jurisdiction_name, jurisdiction_authority,
    binding_level, compliance_enforce_mode, compliance_category, audit_required
) VALUES (
    'INDIA_NLEM_2022', 'National List of Essential Medicines 2022', '2022.1', '2022-09-01',
    'FORMULARY', 'INDIA', 'India', 'NLEM',
    'FEDERAL', 'HARD_BLOCK', 'NLEM_COMPLIANCE', true
) ON CONFLICT (policy_id, policy_version, payer_id, plan_id) DO NOTHING;

-- Australia PBS (Pharmaceutical Benefits Scheme)
INSERT INTO policy_bindings (
    policy_id, policy_name, policy_version, policy_effective_date,
    policy_type, jurisdiction_type, jurisdiction_name, jurisdiction_authority,
    binding_level, compliance_enforce_mode, compliance_category, audit_required
) VALUES (
    'AUSTRALIA_PBS_2024', 'Pharmaceutical Benefits Scheme', '2024.Q3', '2024-07-01',
    'REIMBURSEMENT', 'AUSTRALIA', 'Australia', 'PBS',
    'FEDERAL', 'SOFT_BLOCK', 'PBS_COMPLIANCE', true
) ON CONFLICT (policy_id, policy_version, payer_id, plan_id) DO NOTHING;

-- US Medicare Part D
INSERT INTO policy_bindings (
    policy_id, policy_name, policy_version, policy_effective_date,
    policy_type, jurisdiction_type, jurisdiction_name, jurisdiction_authority,
    binding_level, compliance_enforce_mode, compliance_category, audit_required,
    program_type
) VALUES (
    'US_MEDICARE_PART_D_2025', 'Medicare Part D Coverage', '2025.1', '2025-01-01',
    'PRIOR_AUTHORIZATION', 'US', 'United States', 'CMS',
    'FEDERAL', 'HARD_BLOCK', 'MEDICARE_PART_D', true,
    'Medicare'
) ON CONFLICT (policy_id, policy_version, payer_id, plan_id) DO NOTHING;

-- Hospital Formulary Template
INSERT INTO policy_bindings (
    policy_id, policy_name, policy_version, policy_effective_date,
    policy_type, jurisdiction_type, jurisdiction_name, jurisdiction_authority,
    binding_level, compliance_enforce_mode, compliance_category, audit_required,
    override_allowed, override_approval_level
) VALUES (
    'HOSPITAL_FORMULARY_TEMPLATE', 'Hospital P&T Committee Formulary', '1.0', '2024-01-01',
    'FORMULARY', 'US', 'Institutional', 'Hospital P&T Committee',
    'HOSPITAL', 'WARN', 'HOSPITAL_FORMULARY', false,
    true, 'PHARMACIST'
) ON CONFLICT (policy_id, policy_version, payer_id, plan_id) DO NOTHING;

-- =============================================================================
-- Add policy binding reference columns to existing tables
-- =============================================================================

-- Add policy binding to PA requirements
ALTER TABLE pa_requirements
ADD COLUMN IF NOT EXISTS policy_binding_id UUID REFERENCES policy_bindings(id);

-- Add policy binding to ST rules
ALTER TABLE step_therapy_rules
ADD COLUMN IF NOT EXISTS policy_binding_id UUID REFERENCES policy_bindings(id);

-- Add policy binding to PA submissions
ALTER TABLE pa_submissions
ADD COLUMN IF NOT EXISTS policy_binding_id UUID REFERENCES policy_bindings(id);

-- Add policy binding to ST overrides
ALTER TABLE step_therapy_overrides
ADD COLUMN IF NOT EXISTS policy_binding_id UUID REFERENCES policy_bindings(id);

-- =============================================================================
-- Functions for Policy Binding
-- =============================================================================

-- Function to get applicable policy binding for a drug/payer/plan
CREATE OR REPLACE FUNCTION get_applicable_policy_binding(
    p_policy_type VARCHAR,
    p_payer_id VARCHAR DEFAULT NULL,
    p_plan_id VARCHAR DEFAULT NULL,
    p_jurisdiction_type VARCHAR DEFAULT 'US'
) RETURNS UUID AS $$
DECLARE
    v_binding_id UUID;
BEGIN
    -- Find most specific applicable binding
    SELECT id INTO v_binding_id
    FROM policy_bindings
    WHERE policy_type = p_policy_type
      AND is_active = true
      AND (policy_expiration_date IS NULL OR policy_expiration_date > CURRENT_DATE)
      AND (
          -- Exact match for payer/plan
          (payer_id = p_payer_id AND plan_id = p_plan_id)
          -- Payer-level match
          OR (payer_id = p_payer_id AND plan_id IS NULL)
          -- Jurisdiction-level match
          OR (payer_id IS NULL AND jurisdiction_type = p_jurisdiction_type)
      )
    ORDER BY
        CASE
            WHEN payer_id = p_payer_id AND plan_id = p_plan_id THEN 1
            WHEN payer_id = p_payer_id THEN 2
            ELSE 3
        END,
        precedence DESC
    LIMIT 1;

    RETURN v_binding_id;
END;
$$ LANGUAGE plpgsql;

-- Function to log policy violation
CREATE OR REPLACE FUNCTION log_policy_violation(
    p_binding_id UUID,
    p_patient_id VARCHAR,
    p_drug_rxnorm VARCHAR,
    p_drug_name VARCHAR,
    p_violation_type VARCHAR,
    p_violation_code VARCHAR,
    p_message TEXT,
    p_severity VARCHAR,
    p_enforcement_action VARCHAR,
    p_correlation_id VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_violation_id UUID;
BEGIN
    INSERT INTO policy_violations (
        binding_id, patient_id, drug_rxnorm, drug_name,
        violation_type, violation_code, message, severity,
        enforcement_action, correlation_id, requires_override
    ) VALUES (
        p_binding_id, p_patient_id, p_drug_rxnorm, p_drug_name,
        p_violation_type, p_violation_code, p_message, p_severity,
        p_enforcement_action, p_correlation_id,
        p_enforcement_action IN ('HARD_BLOCK', 'SOFT_BLOCK')
    )
    RETURNING id INTO v_violation_id;

    RETURN v_violation_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update timestamps
CREATE OR REPLACE FUNCTION update_policy_binding_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_policy_binding_timestamp
    BEFORE UPDATE ON policy_bindings
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_binding_timestamp();
