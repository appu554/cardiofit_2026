-- =============================================================================
-- KB-5 Phase 4: Governance and Attribution Tables
-- Migration: 005_governance_attribution.sql
-- Purpose: Severity → Governance mapping and attribution/provenance tracking
-- =============================================================================

-- Create governance action enum type
DO $$ BEGIN
    CREATE TYPE governance_action AS ENUM (
        'ignore',
        'notify',
        'warn_acknowledge',
        'hard_block',
        'hard_block_governance_override',
        'mandatory_escalation'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Create evidence strength enum type
DO $$ BEGIN
    CREATE TYPE evidence_strength AS ENUM (
        'HIGH',
        'MODERATE',
        'LOW',
        'VERY_LOW',
        'UNGRADED'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Create clinical source enum type
DO $$ BEGIN
    CREATE TYPE clinical_source AS ENUM (
        'FDA_LABEL',
        'DRUGBANK',
        'RXNORM',
        'CPIC',
        'PHARMGKB',
        'MICROMEDEX',
        'LEXICOMP',
        'UPTODATE',
        'CLINICAL_TRIALS',
        'PEER_REVIEWED',
        'EXPERT_CONSENSUS',
        'INSTITUTIONAL',
        'UNKNOWN'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Create governance relevance enum type
DO $$ BEGIN
    CREATE TYPE governance_relevance AS ENUM (
        'REGULATORY',
        'LEGAL',
        'ACCREDITATION',
        'FORMULARY',
        'PROTOCOL',
        'ADVISORY',
        'INFORMATIONAL'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- =============================================================================
-- Severity → Governance Mapping Policies Table
-- Allows institutional customization of severity-to-action translation
-- =============================================================================

CREATE TABLE IF NOT EXISTS ddi_severity_governance_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_name VARCHAR(100) NOT NULL,
    institution_id VARCHAR(100),

    -- Severity → Governance action mappings
    contraindicated_action governance_action NOT NULL DEFAULT 'hard_block',
    major_action governance_action NOT NULL DEFAULT 'warn_acknowledge',
    moderate_action governance_action NOT NULL DEFAULT 'notify',
    minor_action governance_action NOT NULL DEFAULT 'ignore',
    unknown_action governance_action NOT NULL DEFAULT 'notify',

    -- Context-based escalation flags
    pediatric_escalation BOOLEAN DEFAULT TRUE,
    geriatric_escalation BOOLEAN DEFAULT TRUE,
    renal_impairment_upgrade BOOLEAN DEFAULT TRUE,
    hepatic_impairment_upgrade BOOLEAN DEFAULT TRUE,

    -- Policy metadata
    effective_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expiry_date TIMESTAMPTZ,
    active BOOLEAN DEFAULT TRUE,
    approved_by VARCHAR(100),
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_active_policy_per_institution UNIQUE (institution_id, active) WHERE active = TRUE
);

-- Create unique index for policy name
CREATE UNIQUE INDEX IF NOT EXISTS idx_governance_policy_name
    ON ddi_severity_governance_mappings(policy_name);

-- Create index for institution lookup
CREATE INDEX IF NOT EXISTS idx_governance_institution
    ON ddi_severity_governance_mappings(institution_id, active);

-- =============================================================================
-- Attribution Audit Trail Table
-- Stores provenance/audit information for medico-legal compliance
-- =============================================================================

CREATE TABLE IF NOT EXISTS ddi_attribution_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Transaction reference
    transaction_id VARCHAR(100) NOT NULL,
    request_hash VARCHAR(64),

    -- Interaction reference
    interaction_id VARCHAR(100),
    drug1_code VARCHAR(100),
    drug2_code VARCHAR(100),

    -- Governance decision
    clinical_severity VARCHAR(20) NOT NULL,
    governance_action governance_action NOT NULL,
    is_blocking BOOLEAN DEFAULT FALSE,
    requires_acknowledgment BOOLEAN DEFAULT FALSE,
    escalation_required BOOLEAN DEFAULT FALSE,
    escalation_target VARCHAR(50),

    -- Attribution envelope
    dataset_version VARCHAR(50) NOT NULL,
    dataset_date TIMESTAMPTZ NOT NULL,
    clinical_sources TEXT[], -- Array of clinical source enums as text
    evidence_level VARCHAR(20),
    evidence_strength evidence_strength,
    confidence_score DECIMAL(3,2),
    governance_relevance TEXT[], -- Array of relevance types
    program_flags TEXT[], -- Array of program flags

    -- Policy applied
    policy_name VARCHAR(100),
    institution_id VARCHAR(100),

    -- Processing metadata
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_time_ms DECIMAL(10,2),
    engines_used TEXT[],

    -- Audit metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    client_ip VARCHAR(45),
    user_agent TEXT
);

-- Create indexes for audit lookup
CREATE INDEX IF NOT EXISTS idx_attribution_audit_transaction
    ON ddi_attribution_audit(transaction_id);

CREATE INDEX IF NOT EXISTS idx_attribution_audit_drugs
    ON ddi_attribution_audit(drug1_code, drug2_code);

CREATE INDEX IF NOT EXISTS idx_attribution_audit_date
    ON ddi_attribution_audit(processed_at);

CREATE INDEX IF NOT EXISTS idx_attribution_audit_governance
    ON ddi_attribution_audit(governance_action, is_blocking);

-- =============================================================================
-- Governance Override Requests Table
-- Tracks requests to override hard-blocked interactions
-- =============================================================================

CREATE TABLE IF NOT EXISTS ddi_governance_override_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Reference to the blocked interaction
    audit_id UUID REFERENCES ddi_attribution_audit(id),
    transaction_id VARCHAR(100) NOT NULL,

    -- Requester information
    requester_id VARCHAR(100) NOT NULL,
    requester_role VARCHAR(50) NOT NULL,
    department VARCHAR(100),

    -- Patient context
    patient_id VARCHAR(100),
    encounter_id VARCHAR(100),

    -- Override request details
    governance_action governance_action NOT NULL,
    override_justification TEXT NOT NULL,
    clinical_rationale TEXT,
    alternative_monitoring_plan TEXT,

    -- Request status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'expired')),

    -- Approver information (if approved/denied)
    approver_id VARCHAR(100),
    approver_role VARCHAR(50),
    approval_decision_at TIMESTAMPTZ,
    approval_notes TEXT,

    -- Validity
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for override requests
CREATE INDEX IF NOT EXISTS idx_override_requests_status
    ON ddi_governance_override_requests(status, created_at);

CREATE INDEX IF NOT EXISTS idx_override_requests_patient
    ON ddi_governance_override_requests(patient_id, encounter_id);

-- =============================================================================
-- Seed Default Governance Policy
-- =============================================================================

INSERT INTO ddi_severity_governance_mappings (
    policy_name,
    institution_id,
    contraindicated_action,
    major_action,
    moderate_action,
    minor_action,
    unknown_action,
    pediatric_escalation,
    geriatric_escalation,
    renal_impairment_upgrade,
    hepatic_impairment_upgrade,
    active,
    approved_by,
    approved_at
) VALUES (
    'default_clinical_safety',
    NULL, -- Global default (no institution)
    'hard_block',
    'warn_acknowledge',
    'notify',
    'ignore',
    'notify',
    TRUE,
    TRUE,
    TRUE,
    TRUE,
    TRUE,
    'system',
    NOW()
) ON CONFLICT (policy_name) DO NOTHING;

-- =============================================================================
-- Comments for documentation
-- =============================================================================

COMMENT ON TABLE ddi_severity_governance_mappings IS
    'Stores severity → governance action mapping policies. Each institution can have custom policies.';

COMMENT ON TABLE ddi_attribution_audit IS
    'Audit trail for all interaction checks with full attribution/provenance for medico-legal compliance.';

COMMENT ON TABLE ddi_governance_override_requests IS
    'Tracks requests to override hard-blocked interactions, requiring governance approval.';

COMMENT ON COLUMN ddi_severity_governance_mappings.contraindicated_action IS
    'Action for CONTRAINDICATED severity - typically hard_block';

COMMENT ON COLUMN ddi_attribution_audit.request_hash IS
    'SHA256 hash of the original request for integrity verification';

COMMENT ON COLUMN ddi_attribution_audit.evidence_strength IS
    'GRADE-style evidence strength assessment';

COMMENT ON COLUMN ddi_governance_override_requests.override_justification IS
    'Clinical justification for why the override is medically necessary';
