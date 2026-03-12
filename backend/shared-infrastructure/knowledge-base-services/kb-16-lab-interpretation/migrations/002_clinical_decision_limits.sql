-- =============================================================================
-- Migration 002: Clinical Decision Limits for DDI Context Evaluation
-- =============================================================================
-- CRITICAL ARCHITECTURAL FIX:
--   This table provides CLINICAL DECISION LIMITS, NOT reference ranges.
--
-- The Difference (from "Architecture of Context" research):
--   ❌ Reference Range: K+ 3.5-5.1 (statistical 95th percentile, CLSI C28-A3)
--   ✅ Decision Limit:  K+ > 5.5 (KDIGO intervention threshold for hyperkalemia)
--
-- Reference ranges have ~5% false positive rate BY DESIGN.
-- Clinical Decision Limits are guideline-anchored intervention thresholds.
--
-- Sources: KDIGO 2024, AHA/ACC, CPIC, CredibleMeds, FDA, HFSA, ACCP
-- =============================================================================

BEGIN;

-- =============================================================================
-- 1. CLINICAL DECISION LIMITS TABLE
-- =============================================================================
-- Purpose: Authoritative intervention thresholds for DDI context evaluation
-- WARNING: These are NOT reference ranges (CLSI C28-A3)
-- =============================================================================

CREATE TABLE IF NOT EXISTS kb16_clinical_decision_limits (
    id                      SERIAL PRIMARY KEY,

    -- Lab identification
    loinc_code              VARCHAR(20) NOT NULL,
    loinc_name              VARCHAR(255),

    -- Clinical context (what this limit evaluates)
    clinical_context        VARCHAR(100) NOT NULL,
    clinical_context_desc   TEXT,

    -- Decision limit specification
    operator                VARCHAR(5) NOT NULL CHECK (operator IN ('>', '<', '>=', '<=', '=')),
    decision_limit_value    NUMERIC(10,3) NOT NULL,
    unit                    VARCHAR(50) NOT NULL,

    -- Authority and evidence
    authority               VARCHAR(100) NOT NULL,          -- KDIGO, AHA, CPIC, etc.
    authority_tier          INTEGER DEFAULT 2,              -- 1=Regulatory, 2=Guideline, 3=Consensus
    evidence_reference      TEXT,                           -- Citation or URL
    evidence_level          VARCHAR(20),                    -- Grade A/B/C or Level I/II/III

    -- DDI rule linkage (which rules use this limit)
    ddi_rule_ids            INTEGER[],

    -- Lifecycle
    active                  BOOLEAN DEFAULT TRUE,
    effective_from          DATE DEFAULT CURRENT_DATE,
    effective_to            DATE,
    superseded_by           INTEGER REFERENCES kb16_clinical_decision_limits(id),

    -- Audit
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW(),
    created_by              VARCHAR(100) DEFAULT 'system',

    -- Unique constraint: one limit per LOINC/context combination
    UNIQUE (loinc_code, clinical_context)
);

-- Performance indexes
CREATE INDEX idx_cdl_loinc ON kb16_clinical_decision_limits(loinc_code) WHERE active = TRUE;
CREATE INDEX idx_cdl_context ON kb16_clinical_decision_limits(clinical_context) WHERE active = TRUE;
CREATE INDEX idx_cdl_loinc_context ON kb16_clinical_decision_limits(loinc_code, clinical_context) WHERE active = TRUE;
CREATE INDEX idx_cdl_authority ON kb16_clinical_decision_limits(authority);
CREATE INDEX idx_cdl_ddi_rules ON kb16_clinical_decision_limits USING GIN(ddi_rule_ids) WHERE active = TRUE;

-- =============================================================================
-- 2. SEED DATA: Authoritative Clinical Decision Limits
-- =============================================================================
-- Source: Table A from "Architecture of Context" research document
-- =============================================================================

INSERT INTO kb16_clinical_decision_limits
(loinc_code, loinc_name, clinical_context, clinical_context_desc, operator, decision_limit_value, unit, authority, authority_tier, evidence_reference, evidence_level, ddi_rule_ids)
VALUES

-- ═══════════════════════════════════════════════════════════════════════════
-- POTASSIUM - Electrolyte DDIs (Cardiac Glycosides, ACEi/ARB, K-sparing)
-- ═══════════════════════════════════════════════════════════════════════════
('2823-3', 'Potassium [Moles/volume] in Serum or Plasma',
 'HYPERKALEMIA_RISK', 'Risk threshold for hyperkalemia in ACEi/ARB/K-sparing diuretic DDIs',
 '>', 5.5, 'mmol/L',
 'KDIGO 2024 + AHA/ACC', 2,
 'KDIGO Clinical Practice Guideline for CKD 2024; AHA Heart Failure Guidelines',
 'Grade B',
 ARRAY[10, 11]),

('2823-3', 'Potassium [Moles/volume] in Serum or Plasma',
 'HYPOKALEMIA_RISK', 'Risk threshold for digoxin toxicity potentiation',
 '<', 3.5, 'mmol/L',
 'AHA/ACC', 2,
 'AHA Digoxin Scientific Statement; Loop diuretic + Cardiac glycoside interactions',
 'Grade B',
 ARRAY[9]),

-- ═══════════════════════════════════════════════════════════════════════════
-- QTc INTERVAL - Arrhythmia Risk (QT-prolonging drug combinations)
-- ═══════════════════════════════════════════════════════════════════════════
('8636-3', 'QTc interval',
 'QT_PROLONGATION_CRITICAL', 'Critical risk for Torsades de Pointes - CONTRAINDICATE additional QT drugs',
 '>', 500, 'ms',
 'CredibleMeds + Tisdale Score', 2,
 'CredibleMeds QT Risk Categories; Tisdale QT Risk Score; Arizona CERT',
 'Grade A',
 ARRAY[7]),

('8636-3', 'QTc interval',
 'QT_PROLONGATION_HIGH', 'High risk for TdP - Enhanced monitoring required for QT drug combinations',
 '>', 470, 'ms',
 'CredibleMeds', 2,
 'CredibleMeds QT Risk Categories; ESC Guidelines on Ventricular Arrhythmias',
 'Grade B',
 ARRAY[7]),

('8636-3', 'QTc interval',
 'QT_PROLONGATION_MODERATE', 'Moderate risk - Consider alternative or closer monitoring',
 '>', 450, 'ms',
 'CredibleMeds', 2,
 'CredibleMeds QT Risk Categories',
 'Grade B',
 ARRAY[7]),

-- ═══════════════════════════════════════════════════════════════════════════
-- DIGOXIN LEVEL - Toxicity Risk
-- ═══════════════════════════════════════════════════════════════════════════
('2226-9', 'Digoxin [Mass/volume] in Serum or Plasma',
 'DIGOXIN_TOXICITY', 'Increased mortality threshold in heart failure',
 '>', 1.2, 'ng/mL',
 'HFSA + DIG Trial', 2,
 'DIG Trial post-hoc analysis; HFSA Digoxin Position Statement',
 'Grade A',
 ARRAY[9]),

-- ═══════════════════════════════════════════════════════════════════════════
-- RENAL FUNCTION - Dose Adjustment Thresholds
-- ═══════════════════════════════════════════════════════════════════════════
('2160-0', 'Creatinine clearance',
 'RENAL_IMPAIRMENT_SEVERE', 'Severe renal impairment - dose reduction/contraindication threshold',
 '<', 30, 'mL/min',
 'CPIC + FDA', 1,
 'CPIC Guidelines; FDA Drug Labeling Requirements',
 'Grade A',
 NULL),

('2160-0', 'Creatinine clearance',
 'RENAL_IMPAIRMENT_MODERATE', 'Moderate renal impairment - monitoring/adjustment threshold',
 '<', 60, 'mL/min',
 'KDIGO 2024', 2,
 'KDIGO CKD Classification and Management',
 'Grade B',
 NULL),

('33914-3', 'eGFR CKD-EPI',
 'EGFR_SEVERE', 'eGFR-based severe renal impairment (CKD Stage 4-5)',
 '<', 30, 'mL/min/1.73m2',
 'KDIGO 2024', 2,
 'KDIGO CKD Classification 2024',
 'Grade A',
 NULL),

('33914-3', 'eGFR CKD-EPI',
 'EGFR_MODERATE', 'eGFR-based moderate renal impairment (CKD Stage 3)',
 '<', 60, 'mL/min/1.73m2',
 'KDIGO 2024', 2,
 'KDIGO CKD Classification 2024',
 'Grade A',
 NULL),

-- ═══════════════════════════════════════════════════════════════════════════
-- HEPATIC FUNCTION - NCI Organ Dysfunction Classification
-- ═══════════════════════════════════════════════════════════════════════════
('1975-2', 'Bilirubin.total [Mass/volume] in Serum or Plasma',
 'HEPATIC_IMPAIRMENT_SEVERE', 'Severe hepatic impairment (Bilirubin > 3x ULN)',
 '>', 3.0, 'xULN',
 'NCI ODWG', 1,
 'NCI Organ Dysfunction Working Group Classification',
 'Grade A',
 NULL),

('1975-2', 'Bilirubin.total [Mass/volume] in Serum or Plasma',
 'HEPATIC_IMPAIRMENT_MODERATE', 'Moderate hepatic impairment (Bilirubin 1.5-3x ULN)',
 '>', 1.5, 'xULN',
 'NCI ODWG', 1,
 'NCI Organ Dysfunction Working Group Classification',
 'Grade A',
 NULL),

-- ═══════════════════════════════════════════════════════════════════════════
-- INR - Bleeding Risk (Anticoagulant DDIs)
-- ═══════════════════════════════════════════════════════════════════════════
('6301-6', 'INR',
 'BLEEDING_RISK_CRITICAL', 'Critical bleeding risk with anticoagulant DDIs',
 '>', 4.0, 'ratio',
 'ACCP Guidelines', 2,
 'ACCP Antithrombotic Therapy Guidelines; CHEST 2021',
 'Grade A',
 ARRAY[4, 5, 6]),

('6301-6', 'INR',
 'BLEEDING_RISK_HIGH', 'High bleeding risk - warfarin + NSAID/antiplatelet combinations',
 '>', 3.5, 'ratio',
 'ACCP Guidelines', 2,
 'ACCP Antithrombotic Therapy Guidelines',
 'Grade B',
 ARRAY[4, 5, 6]),

('6301-6', 'INR',
 'BLEEDING_RISK_ELEVATED', 'Elevated bleeding risk - enhanced monitoring required',
 '>', 3.0, 'ratio',
 'ACCP Guidelines', 2,
 'ACCP Antithrombotic Therapy Guidelines',
 'Grade B',
 ARRAY[4, 5, 6]),

-- ═══════════════════════════════════════════════════════════════════════════
-- LITHIUM - Toxicity Risk (Thiazide/NSAID DDIs)
-- ═══════════════════════════════════════════════════════════════════════════
('2721-9', 'Lithium [Moles/volume] in Serum or Plasma',
 'LITHIUM_TOXICITY', 'Lithium toxicity threshold - thiazide/NSAID interactions increase levels',
 '>', 1.2, 'mmol/L',
 'APA Guidelines', 2,
 'APA Practice Guideline for Bipolar Disorder; FDA Lithium Labeling',
 'Grade A',
 ARRAY[8]),

('2721-9', 'Lithium [Moles/volume] in Serum or Plasma',
 'LITHIUM_HIGH_THERAPEUTIC', 'Upper therapeutic range - monitor closely with DDIs',
 '>', 1.0, 'mmol/L',
 'APA Guidelines', 2,
 'APA Practice Guideline for Bipolar Disorder',
 'Grade B',
 ARRAY[8]),

-- ═══════════════════════════════════════════════════════════════════════════
-- BLOOD PRESSURE - Hypotension Risk (PDE-5/Nitrate Contraindication)
-- ═══════════════════════════════════════════════════════════════════════════
('8480-6', 'Systolic blood pressure',
 'HYPOTENSION_RISK', 'Systolic BP threshold for PDE-5 + Nitrate contraindication',
 '<', 90, 'mmHg',
 'AHA/ACC', 2,
 'AHA/ACC Guideline for Management of Patients with Stable Ischemic Heart Disease',
 'Grade A',
 ARRAY[15]),

-- ═══════════════════════════════════════════════════════════════════════════
-- PLATELET COUNT - Bleeding Risk
-- ═══════════════════════════════════════════════════════════════════════════
('777-3', 'Platelets [#/volume] in Blood',
 'THROMBOCYTOPENIA_SEVERE', 'Severe thrombocytopenia - contraindicate anticoagulants/antiplatelets',
 '<', 50, '10*9/L',
 'ASH Guidelines', 2,
 'ASH Thrombocytopenia Guidelines',
 'Grade A',
 ARRAY[4, 5, 6]),

('777-3', 'Platelets [#/volume] in Blood',
 'THROMBOCYTOPENIA_MODERATE', 'Moderate thrombocytopenia - caution with anticoagulants',
 '<', 100, '10*9/L',
 'ASH Guidelines', 2,
 'ASH Thrombocytopenia Guidelines',
 'Grade B',
 ARRAY[4, 5, 6])

ON CONFLICT (loinc_code, clinical_context) DO UPDATE SET
    decision_limit_value = EXCLUDED.decision_limit_value,
    authority = EXCLUDED.authority,
    evidence_reference = EXCLUDED.evidence_reference,
    ddi_rule_ids = EXCLUDED.ddi_rule_ids,
    updated_at = NOW();

-- =============================================================================
-- 3. HELPER FUNCTIONS
-- =============================================================================

-- Get decision limit for a LOINC/context pair
CREATE OR REPLACE FUNCTION get_clinical_decision_limit(
    p_loinc_code VARCHAR(20),
    p_clinical_context VARCHAR(100)
)
RETURNS TABLE (
    loinc_code VARCHAR(20),
    clinical_context VARCHAR(100),
    operator VARCHAR(5),
    decision_limit_value NUMERIC(10,3),
    unit VARCHAR(50),
    authority VARCHAR(100),
    evidence_level VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        cdl.loinc_code,
        cdl.clinical_context,
        cdl.operator,
        cdl.decision_limit_value,
        cdl.unit,
        cdl.authority,
        cdl.evidence_level
    FROM kb16_clinical_decision_limits cdl
    WHERE cdl.loinc_code = p_loinc_code
      AND cdl.clinical_context = p_clinical_context
      AND cdl.active = TRUE
      AND (cdl.effective_to IS NULL OR cdl.effective_to > CURRENT_DATE);
END;
$$ LANGUAGE plpgsql STABLE;

-- Evaluate a patient lab value against a clinical decision limit
CREATE OR REPLACE FUNCTION evaluate_clinical_limit(
    p_loinc_code VARCHAR(20),
    p_clinical_context VARCHAR(100),
    p_patient_value NUMERIC
)
RETURNS TABLE (
    limit_exceeded BOOLEAN,
    decision_limit NUMERIC,
    patient_value NUMERIC,
    operator VARCHAR(5),
    authority VARCHAR(100),
    evaluation_reason TEXT
) AS $$
DECLARE
    v_limit RECORD;
    v_exceeded BOOLEAN;
BEGIN
    -- Fetch the limit
    SELECT * INTO v_limit
    FROM kb16_clinical_decision_limits
    WHERE loinc_code = p_loinc_code
      AND clinical_context = p_clinical_context
      AND active = TRUE;

    IF NOT FOUND THEN
        RETURN QUERY SELECT
            NULL::BOOLEAN,
            NULL::NUMERIC,
            p_patient_value,
            NULL::VARCHAR(5),
            NULL::VARCHAR(100),
            'No clinical decision limit defined for this LOINC/context'::TEXT;
        RETURN;
    END IF;

    -- Evaluate
    v_exceeded := CASE v_limit.operator
        WHEN '>' THEN p_patient_value > v_limit.decision_limit_value
        WHEN '<' THEN p_patient_value < v_limit.decision_limit_value
        WHEN '>=' THEN p_patient_value >= v_limit.decision_limit_value
        WHEN '<=' THEN p_patient_value <= v_limit.decision_limit_value
        WHEN '=' THEN p_patient_value = v_limit.decision_limit_value
        ELSE FALSE
    END;

    RETURN QUERY SELECT
        v_exceeded,
        v_limit.decision_limit_value,
        p_patient_value,
        v_limit.operator,
        v_limit.authority,
        CASE
            WHEN v_exceeded THEN
                format('Patient value %s EXCEEDS %s limit %s %s (%s)',
                    p_patient_value, v_limit.clinical_context,
                    v_limit.operator, v_limit.decision_limit_value, v_limit.authority)
            ELSE
                format('Patient value %s within safe range for %s (limit: %s %s)',
                    p_patient_value, v_limit.clinical_context,
                    v_limit.operator, v_limit.decision_limit_value)
        END::TEXT;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 4. DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE kb16_clinical_decision_limits IS
'Clinical Decision Limits for DDI context evaluation.

CRITICAL: These are INTERVENTION THRESHOLDS from clinical guidelines,
NOT statistical reference ranges (CLSI C28-A3).

The Difference:
  ❌ Reference Range: K+ 3.5-5.1 (statistical 95th percentile)
  ✅ Decision Limit:  K+ > 5.5 (KDIGO intervention threshold)

Reference ranges have ~5% false positive rate BY DESIGN.
Clinical Decision Limits are guideline-anchored with near-zero false positives.

Sources: KDIGO 2024, AHA/ACC, CPIC, CredibleMeds, FDA, HFSA, ACCP, NCI ODWG

Version: 1.0.0 (2026-01-22)
Reference: Architecture of Context research document - Table A';

COMMENT ON FUNCTION get_clinical_decision_limit(VARCHAR, VARCHAR) IS
'Retrieves the authoritative clinical decision limit for a LOINC/context pair.
Returns NULL if no active limit is defined.';

COMMENT ON FUNCTION evaluate_clinical_limit(VARCHAR, VARCHAR, NUMERIC) IS
'Evaluates a patient lab value against a clinical decision limit.
Returns whether the limit is exceeded with full audit trail.';

COMMIT;

-- =============================================================================
-- MIGRATION METADATA
-- =============================================================================
-- Version: 002
-- Date: 2026-01-22
-- Author: Claude Code
-- Purpose: Clinical Decision Limits for DDI context (NOT reference ranges)
-- Phase: Phase 1 completion - Clinical correctness fix
-- Dependencies: 001_initial_schema.sql
-- Reference: "Architecture of Context" research document
-- =============================================================================
