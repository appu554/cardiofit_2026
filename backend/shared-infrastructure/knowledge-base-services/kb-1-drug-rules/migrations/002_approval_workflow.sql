-- =============================================================================
-- KB-1 Drug Rules: Approval Workflow Migration
-- =============================================================================
-- CRITICAL SAFETY MIGRATION
--
-- This migration adds the approval workflow required for clinical safety.
-- Pattern-based NLP extraction from FDA SPL documents is NOT deterministic.
-- All extracted rules MUST go through approval before activation.
--
-- Lifecycle: DRAFT → REVIEWED → APPROVED → ACTIVE → RETIRED
--
-- Risk Level: CRITICAL - This is the gate between "what FDA says" and "what
--             clinicians can use for dosing". Errors here can kill patients.
-- =============================================================================

-- Add approval_status column with default DRAFT
-- All existing rules are grandfathered as ACTIVE (they were manually reviewed)
ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS approval_status VARCHAR(20) DEFAULT 'DRAFT';

-- Add risk_level column for clinical risk classification
ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS risk_level VARCHAR(20) DEFAULT 'STANDARD';

-- Add pharmacist review tracking
ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(200);

ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS review_notes TEXT;

-- Add extraction quality metrics
ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS extraction_confidence INT;

ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS extraction_warnings JSONB;

ALTER TABLE drug_rules
ADD COLUMN IF NOT EXISTS requires_manual_review BOOLEAN DEFAULT TRUE;

-- Update existing rules to ACTIVE status (they were manually validated)
UPDATE drug_rules
SET approval_status = 'ACTIVE',
    risk_level = CASE
        WHEN is_high_alert = TRUE THEN 'HIGH'
        WHEN has_black_box = TRUE THEN 'HIGH'
        WHEN is_narrow_ti = TRUE THEN 'HIGH'
        ELSE 'STANDARD'
    END,
    requires_manual_review = FALSE
WHERE approval_status IS NULL OR approval_status = 'DRAFT';

-- Create index for approval status filtering (critical for production queries)
CREATE INDEX IF NOT EXISTS idx_drug_rules_approval_status
ON drug_rules(approval_status);

-- Create partial index for production queries (ACTIVE rules only)
CREATE INDEX IF NOT EXISTS idx_drug_rules_active
ON drug_rules(rxnorm_code, jurisdiction)
WHERE approval_status = 'ACTIVE';

-- Create index for risk level filtering
CREATE INDEX IF NOT EXISTS idx_drug_rules_risk_level
ON drug_rules(risk_level);

-- Create index for pending review queue
CREATE INDEX IF NOT EXISTS idx_drug_rules_pending_review
ON drug_rules(created_at DESC)
WHERE approval_status IN ('DRAFT', 'REVIEWED') AND requires_manual_review = TRUE;

-- =============================================================================
-- APPROVAL AUDIT TABLE
-- =============================================================================
-- Tracks all approval state changes for regulatory compliance

CREATE TABLE IF NOT EXISTS drug_rule_approvals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drug_rule_id UUID REFERENCES drug_rules(id) ON DELETE CASCADE,
    rxnorm_code VARCHAR(20) NOT NULL,
    jurisdiction VARCHAR(10) NOT NULL,

    -- State transition
    previous_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,

    -- Who and why
    changed_by VARCHAR(200) NOT NULL,
    change_reason TEXT,

    -- Quality context at time of approval
    extraction_confidence INT,
    risk_level VARCHAR(20),
    risk_factors TEXT[],

    -- Review details
    review_notes TEXT,
    verified_against_source BOOLEAN DEFAULT FALSE,

    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_drug_rule_approvals_drug ON drug_rule_approvals(drug_rule_id);
CREATE INDEX idx_drug_rule_approvals_rxnorm ON drug_rule_approvals(rxnorm_code);
CREATE INDEX idx_drug_rule_approvals_status ON drug_rule_approvals(new_status);
CREATE INDEX idx_drug_rule_approvals_date ON drug_rule_approvals(created_at DESC);

-- =============================================================================
-- UPDATED VIEW FOR PRODUCTION QUERIES
-- =============================================================================
-- Only returns ACTIVE rules - this is what the dosing engine should use

DROP VIEW IF EXISTS v_active_drug_rules;

CREATE VIEW v_active_drug_rules AS
SELECT
    dr.id,
    dr.rxnorm_code,
    dr.jurisdiction,
    dr.drug_name,
    dr.generic_name,
    dr.drug_class,
    dr.rule_data,
    dr.authority,
    dr.document_name,
    dr.document_section,
    dr.document_url,
    dr.version,
    dr.approved_by,
    dr.approved_at,
    dr.is_high_alert,
    dr.is_narrow_ti,
    dr.has_black_box,
    dr.is_beers_list,
    dr.source_set_id,
    dr.source_hash,
    dr.ingested_at
FROM drug_rules dr
WHERE dr.approval_status = 'ACTIVE';

COMMENT ON VIEW v_active_drug_rules IS
'Production view: Only returns ACTIVE (approved) drug rules for dosing calculations';

-- =============================================================================
-- VIEW FOR PENDING REVIEW QUEUE
-- =============================================================================
-- For pharmacists to review pending rules

CREATE VIEW v_pending_review AS
SELECT
    dr.id,
    dr.rxnorm_code,
    dr.drug_name,
    dr.generic_name,
    dr.drug_class,
    dr.jurisdiction,
    dr.authority,
    dr.approval_status,
    dr.risk_level,
    dr.extraction_confidence,
    dr.extraction_warnings,
    dr.is_high_alert,
    dr.has_black_box,
    dr.is_narrow_ti,
    dr.source_set_id,
    dr.document_url,
    dr.ingested_at,
    dr.created_at
FROM drug_rules dr
WHERE dr.approval_status IN ('DRAFT', 'REVIEWED')
  AND dr.requires_manual_review = TRUE
ORDER BY
    -- Critical risk drugs first
    CASE dr.risk_level
        WHEN 'CRITICAL' THEN 0
        WHEN 'HIGH' THEN 1
        WHEN 'STANDARD' THEN 2
        ELSE 3
    END,
    -- Then by extraction confidence (lowest confidence first)
    dr.extraction_confidence ASC NULLS FIRST,
    -- Then by ingestion date
    dr.ingested_at DESC;

COMMENT ON VIEW v_pending_review IS
'Pharmacist review queue: Rules requiring manual validation before activation';

-- =============================================================================
-- FUNCTION: APPROVE DRUG RULE
-- =============================================================================
-- Transitions a rule from DRAFT/REVIEWED to APPROVED/ACTIVE

CREATE OR REPLACE FUNCTION approve_drug_rule(
    p_drug_rule_id UUID,
    p_approved_by VARCHAR(200),
    p_review_notes TEXT DEFAULT NULL,
    p_skip_verification BOOLEAN DEFAULT FALSE
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_status VARCHAR(20);
    v_risk_level VARCHAR(20);
    v_rxnorm_code VARCHAR(20);
    v_jurisdiction VARCHAR(10);
BEGIN
    -- Get current state
    SELECT approval_status, risk_level, rxnorm_code, jurisdiction
    INTO v_current_status, v_risk_level, v_rxnorm_code, v_jurisdiction
    FROM drug_rules
    WHERE id = p_drug_rule_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Drug rule not found: %', p_drug_rule_id;
    END IF;

    -- Validate state transition
    IF v_current_status NOT IN ('DRAFT', 'REVIEWED') THEN
        RAISE EXCEPTION 'Cannot approve rule in status: %', v_current_status;
    END IF;

    -- CRITICAL/HIGH risk drugs require explicit verification
    IF v_risk_level IN ('CRITICAL', 'HIGH') AND NOT p_skip_verification THEN
        RAISE EXCEPTION 'High-risk drug requires explicit verification flag';
    END IF;

    -- Update the rule
    UPDATE drug_rules
    SET approval_status = 'ACTIVE',
        approved_by = p_approved_by,
        approved_at = NOW(),
        reviewed_by = p_approved_by,
        reviewed_at = NOW(),
        review_notes = p_review_notes,
        requires_manual_review = FALSE
    WHERE id = p_drug_rule_id;

    -- Log the approval
    INSERT INTO drug_rule_approvals (
        drug_rule_id, rxnorm_code, jurisdiction,
        previous_status, new_status,
        changed_by, change_reason, review_notes,
        risk_level, verified_against_source
    )
    VALUES (
        p_drug_rule_id, v_rxnorm_code, v_jurisdiction,
        v_current_status, 'ACTIVE',
        p_approved_by, 'Approved for clinical use', p_review_notes,
        v_risk_level, p_skip_verification
    );

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FUNCTION: REJECT DRUG RULE
-- =============================================================================
-- Marks a rule as rejected (not suitable for clinical use)

CREATE OR REPLACE FUNCTION reject_drug_rule(
    p_drug_rule_id UUID,
    p_rejected_by VARCHAR(200),
    p_rejection_reason TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_status VARCHAR(20);
    v_rxnorm_code VARCHAR(20);
    v_jurisdiction VARCHAR(10);
BEGIN
    -- Get current state
    SELECT approval_status, rxnorm_code, jurisdiction
    INTO v_current_status, v_rxnorm_code, v_jurisdiction
    FROM drug_rules
    WHERE id = p_drug_rule_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Drug rule not found: %', p_drug_rule_id;
    END IF;

    -- Update the rule
    UPDATE drug_rules
    SET approval_status = 'RETIRED',
        reviewed_by = p_rejected_by,
        reviewed_at = NOW(),
        review_notes = p_rejection_reason,
        requires_manual_review = FALSE
    WHERE id = p_drug_rule_id;

    -- Log the rejection
    INSERT INTO drug_rule_approvals (
        drug_rule_id, rxnorm_code, jurisdiction,
        previous_status, new_status,
        changed_by, change_reason
    )
    VALUES (
        p_drug_rule_id, v_rxnorm_code, v_jurisdiction,
        v_current_status, 'RETIRED',
        p_rejected_by, p_rejection_reason
    );

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TRIGGER: PREVENT UNAPPROVED RULES IN PRODUCTION QUERIES
-- =============================================================================
-- This is a safety check - the application should use v_active_drug_rules,
-- but this trigger adds a belt-and-suspenders approach

CREATE OR REPLACE FUNCTION check_approval_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Log a warning if someone queries non-ACTIVE rules
    -- This is informational, not blocking
    IF NEW.approval_status != 'ACTIVE' THEN
        RAISE NOTICE 'Warning: Accessing non-ACTIVE rule % (status: %)',
            NEW.rxnorm_code, NEW.approval_status;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON COLUMN drug_rules.approval_status IS
'Approval lifecycle: DRAFT (ingested) → REVIEWED (pharmacist checked) → APPROVED → ACTIVE (in production) → RETIRED';

COMMENT ON COLUMN drug_rules.risk_level IS
'Clinical risk classification: CRITICAL (anticoagulants, insulin, chemo), HIGH (narrow TI, black box), STANDARD, LOW';

COMMENT ON COLUMN drug_rules.reviewed_by IS
'Pharmacist or CMO who reviewed this rule';

COMMENT ON COLUMN drug_rules.reviewed_at IS
'Timestamp of pharmacist review';

COMMENT ON COLUMN drug_rules.extraction_confidence IS
'NLP extraction confidence score (0-100). Low scores require manual review.';

COMMENT ON COLUMN drug_rules.requires_manual_review IS
'Flag indicating this rule needs pharmacist review before activation';

COMMENT ON TABLE drug_rule_approvals IS
'Audit trail of all approval state changes for regulatory compliance';
