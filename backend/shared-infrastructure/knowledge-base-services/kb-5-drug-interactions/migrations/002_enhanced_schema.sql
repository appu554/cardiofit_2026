-- KB-5 Enhanced Schema Migration: Production-Grade Drug Interactions
-- Implements the enhanced data model from the Final Proposal (31_8.2)
-- This migration adds pharmacogenomic rules, class interactions, evidence tracking, and dataset versioning

-- Create enhanced data types
CREATE TYPE ddi_severity AS ENUM ('contraindicated','major','moderate','minor','unknown');
CREATE TYPE evidence_level AS ENUM ('A','B','C','D','ExpertOpinion','Unknown');
CREATE TYPE mechanism_type AS ENUM ('PK','PD','PK_PD','Unknown');
CREATE TYPE context_use AS ENUM ('acute','chronic','unspecified');

-- Add new columns to existing drug_interactions table
ALTER TABLE drug_interactions 
ADD COLUMN IF NOT EXISTS dataset_version TEXT NOT NULL DEFAULT '2025Q3.initial',
ADD COLUMN IF NOT EXISTS evidence evidence_level NOT NULL DEFAULT 'Unknown',
ADD COLUMN IF NOT EXISTS confidence NUMERIC(3,2) DEFAULT 0.80,
ADD COLUMN IF NOT EXISTS route_restriction TEXT[],
ADD COLUMN IF NOT EXISTS min_dose1_mg NUMERIC,
ADD COLUMN IF NOT EXISTS min_dose2_mg NUMERIC,
ADD COLUMN IF NOT EXISTS context context_use DEFAULT 'unspecified',
ADD COLUMN IF NOT EXISTS pgx_required BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS pgx_markers JSONB,
ADD COLUMN IF NOT EXISTS food_alcohol_herbal JSONB,
ADD COLUMN IF NOT EXISTS contraindication_reason TEXT,
ADD COLUMN IF NOT EXISTS source_vendor_ids TEXT[];

-- Add unique constraint on dataset version + drug pair
ALTER TABLE drug_interactions 
ADD CONSTRAINT unique_dataset_drug_pair 
UNIQUE (dataset_version, drug_a_code, drug_b_code);

-- Drug class interaction rules
CREATE TABLE ddi_class_rules (
  id BIGSERIAL PRIMARY KEY,
  dataset_version TEXT NOT NULL,
  object_type TEXT NOT NULL CHECK (object_type IN ('drug','class')),
  object_code TEXT NOT NULL,
  subject_type TEXT NOT NULL CHECK (subject_type IN ('drug','class')),
  subject_code TEXT NOT NULL,
  severity ddi_severity NOT NULL,
  mechanism mechanism_type NOT NULL,
  clinical_effects TEXT,
  management_strategy TEXT NOT NULL,
  qualifiers JSONB,
  evidence evidence_level NOT NULL,
  confidence NUMERIC(3,2) DEFAULT 0.75,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (dataset_version, object_type, object_code, subject_type, subject_code)
);

-- Pharmacogenomic interaction rules
CREATE TABLE ddi_pharmacogenomic_rules (
  id BIGSERIAL PRIMARY KEY,
  dataset_version TEXT NOT NULL,
  drug_code TEXT NOT NULL,
  gene TEXT NOT NULL,
  phenotype TEXT NOT NULL,
  interaction_with TEXT,
  severity ddi_severity NOT NULL,
  clinical_effects TEXT,
  management_strategy TEXT NOT NULL,
  evidence evidence_level NOT NULL,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Expression-based unique index for nullable column
CREATE UNIQUE INDEX idx_pgx_rules_unique ON ddi_pharmacogenomic_rules
  (dataset_version, drug_code, gene, phenotype, COALESCE(interaction_with, 'NA'));

-- Drug interaction modifiers (food, alcohol, herbal)
CREATE TABLE ddi_modifiers (
  id BIGSERIAL PRIMARY KEY,
  dataset_version TEXT NOT NULL,
  modifier_type TEXT NOT NULL CHECK (modifier_type IN ('food','alcohol','herbal','disease')),
  modifier_code TEXT,
  drug_code TEXT NOT NULL,
  effect TEXT NOT NULL,
  management_strategy TEXT NOT NULL,
  severity ddi_severity NOT NULL,
  evidence evidence_level NOT NULL,
  source_vendor_ids TEXT[],
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Institutional overrides for P&T Committee modifications
CREATE TABLE ddi_overrides (
  id BIGSERIAL PRIMARY KEY,
  dataset_version TEXT NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('pair','class','pgx','modifier')),
  selector JSONB NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('replace','suppress','amend')),
  payload JSONB NOT NULL,
  rationale TEXT NOT NULL,
  approver TEXT NOT NULL,
  approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Dataset version tracking for Evidence Envelope integration
CREATE TABLE ddi_dataset_versions (
  id BIGSERIAL PRIMARY KEY,
  version_name TEXT UNIQUE NOT NULL,
  vendor_source TEXT NOT NULL,
  harmonized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  record_count BIGINT DEFAULT 0,
  quality_score NUMERIC(3,2) DEFAULT 0.0,
  is_current BOOLEAN DEFAULT FALSE,
  metadata JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  created_by TEXT
);

-- Performance-critical materialized view for fast lookups
CREATE MATERIALIZED VIEW ddi_interaction_matrix AS
SELECT 
    dataset_version,
    drug_a_code AS drug1_code,
    drug_b_code AS drug2_code,
    severity,
    management_strategy,
    evidence,
    confidence,
    mechanism,
    clinical_effect,
    pgx_required,
    pgx_markers,
    route_restriction,
    context
FROM drug_interactions 
WHERE active = TRUE;

-- Indexes for materialized view
CREATE INDEX idx_ddi_matrix_version_drugs ON ddi_interaction_matrix (dataset_version, drug1_code, drug2_code);
CREATE INDEX idx_ddi_matrix_severity ON ddi_interaction_matrix (severity);
CREATE INDEX idx_ddi_matrix_pgx ON ddi_interaction_matrix (pgx_required) WHERE pgx_required = TRUE;

-- Indexes for new tables
CREATE INDEX idx_ddi_class_rules_version ON ddi_class_rules (dataset_version);
CREATE INDEX idx_ddi_class_rules_object ON ddi_class_rules (object_type, object_code);
CREATE INDEX idx_ddi_class_rules_subject ON ddi_class_rules (subject_type, subject_code);
CREATE INDEX idx_ddi_class_rules_active ON ddi_class_rules (active) WHERE active = TRUE;

CREATE INDEX idx_ddi_pgx_drug ON ddi_pharmacogenomic_rules (drug_code);
CREATE INDEX idx_ddi_pgx_gene_phenotype ON ddi_pharmacogenomic_rules (gene, phenotype);
CREATE INDEX idx_ddi_pgx_version ON ddi_pharmacogenomic_rules (dataset_version);
CREATE INDEX idx_ddi_pgx_active ON ddi_pharmacogenomic_rules (active) WHERE active = TRUE;

CREATE INDEX idx_ddi_modifiers_drug ON ddi_modifiers (drug_code);
CREATE INDEX idx_ddi_modifiers_type ON ddi_modifiers (modifier_type);
CREATE INDEX idx_ddi_modifiers_version ON ddi_modifiers (dataset_version);

CREATE INDEX idx_ddi_overrides_version ON ddi_overrides (dataset_version);
CREATE INDEX idx_ddi_overrides_scope ON ddi_overrides (scope);
CREATE INDEX idx_ddi_overrides_active ON ddi_overrides (active) WHERE active = TRUE;

-- Enhanced PostgreSQL functions for high-performance interaction checking
CREATE OR REPLACE FUNCTION check_drug_interactions_enhanced(
    p_dataset_version TEXT,
    p_drug_codes TEXT[],
    p_patient_pgx JSONB DEFAULT NULL,
    p_patient_context JSONB DEFAULT NULL
) RETURNS TABLE (
    interaction_id TEXT,
    drug1_code TEXT,
    drug2_code TEXT,
    severity TEXT,
    mechanism TEXT,
    clinical_effects TEXT,
    management_strategy TEXT,
    evidence TEXT,
    confidence NUMERIC,
    pgx_applicable BOOLEAN,
    route_specific BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        CONCAT(m.drug1_code, '_', m.drug2_code, '_', m.dataset_version) AS interaction_id,
        m.drug1_code::TEXT,
        m.drug2_code::TEXT,
        m.severity::TEXT,
        m.mechanism::TEXT,
        m.clinical_effect::TEXT,
        m.management_strategy::TEXT,
        m.evidence::TEXT,
        m.confidence,
        (m.pgx_required AND p_patient_pgx IS NOT NULL) AS pgx_applicable,
        (m.route_restriction IS NOT NULL) AS route_specific
    FROM ddi_interaction_matrix m
    WHERE m.dataset_version = p_dataset_version
        AND m.drug1_code = ANY(p_drug_codes)
        AND m.drug2_code = ANY(p_drug_codes)
        AND m.drug1_code != m.drug2_code
        -- Apply PGx filtering if patient context provided
        AND (NOT m.pgx_required OR p_patient_pgx IS NOT NULL)
    ORDER BY 
        CASE m.severity
            WHEN 'contraindicated' THEN 4
            WHEN 'major' THEN 3
            WHEN 'moderate' THEN 2
            WHEN 'minor' THEN 1
            ELSE 0
        END DESC,
        m.confidence DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function for class-based interaction checking
CREATE OR REPLACE FUNCTION check_class_interactions(
    p_dataset_version TEXT,
    p_drug_codes TEXT[],
    p_class_codes TEXT[]
) RETURNS TABLE (
    object_code TEXT,
    subject_code TEXT,
    interaction_type TEXT,
    severity TEXT,
    management_strategy TEXT,
    confidence NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        dcr.object_code::TEXT,
        dcr.subject_code::TEXT,
        CONCAT(dcr.object_type, '_to_', dcr.subject_type)::TEXT AS interaction_type,
        dcr.severity::TEXT,
        dcr.management_strategy::TEXT,
        dcr.confidence
    FROM ddi_class_rules dcr
    WHERE dcr.dataset_version = p_dataset_version
        AND dcr.active = TRUE
        AND (
            (dcr.object_type = 'drug' AND dcr.object_code = ANY(p_drug_codes))
            OR (dcr.object_type = 'class' AND dcr.object_code = ANY(p_class_codes))
        )
        AND (
            (dcr.subject_type = 'drug' AND dcr.subject_code = ANY(p_drug_codes))
            OR (dcr.subject_type = 'class' AND dcr.subject_code = ANY(p_class_codes))
        )
    ORDER BY 
        CASE dcr.severity
            WHEN 'contraindicated' THEN 4
            WHEN 'major' THEN 3
            WHEN 'moderate' THEN 2
            WHEN 'minor' THEN 1
            ELSE 0
        END DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function for pharmacogenomic interaction checking
CREATE OR REPLACE FUNCTION check_pgx_interactions(
    p_dataset_version TEXT,
    p_drug_codes TEXT[],
    p_patient_pgx JSONB
) RETURNS TABLE (
    drug_code TEXT,
    gene TEXT,
    phenotype TEXT,
    severity TEXT,
    clinical_effects TEXT,
    management_strategy TEXT,
    applicable BOOLEAN
) AS $$
DECLARE
    pgx_key TEXT;
    pgx_value TEXT;
BEGIN
    -- If no PGx data provided, return empty
    IF p_patient_pgx IS NULL THEN
        RETURN;
    END IF;
    
    RETURN QUERY
    SELECT 
        pgxr.drug_code::TEXT,
        pgxr.gene::TEXT,
        pgxr.phenotype::TEXT,
        pgxr.severity::TEXT,
        pgxr.clinical_effects::TEXT,
        pgxr.management_strategy::TEXT,
        (p_patient_pgx ->> pgxr.gene = pgxr.phenotype) AS applicable
    FROM ddi_pharmacogenomic_rules pgxr
    WHERE pgxr.dataset_version = p_dataset_version
        AND pgxr.active = TRUE
        AND pgxr.drug_code = ANY(p_drug_codes)
        AND p_patient_pgx ? pgxr.gene
        AND p_patient_pgx ->> pgxr.gene = pgxr.phenotype
    ORDER BY 
        CASE pgxr.severity
            WHEN 'contraindicated' THEN 4
            WHEN 'major' THEN 3
            WHEN 'moderate' THEN 2
            WHEN 'minor' THEN 1
            ELSE 0
        END DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_ddi_matrix() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY ddi_interaction_matrix;
END;
$$ LANGUAGE plpgsql;

-- Insert sample enhanced data with new schema
INSERT INTO ddi_dataset_versions (version_name, vendor_source, record_count, quality_score, is_current, metadata) VALUES
('2025Q3.harmonized', 'VendorA_Clinical_Database', 1500, 0.95, TRUE, '{"harmonizer_version": "2.1.0", "validation_passed": true}');

-- Insert sample pharmacogenomic interactions
INSERT INTO ddi_pharmacogenomic_rules (
    dataset_version, drug_code, gene, phenotype, severity, 
    clinical_effects, management_strategy, evidence
) VALUES
('2025Q3.harmonized', 'RxCUI:2670', 'CYP2D6', 'ultrarapid', 'major',
 'Toxic morphine levels due to rapid codeine conversion',
 'Avoid codeine; use non-CYP2D6 metabolized opioid (morphine, oxycodone)',
 'A'),
('2025Q3.harmonized', 'RxCUI:36567', 'SLCO1B1', 'poor', 'major',
 'Increased simvastatin exposure and myopathy risk',
 'Reduce simvastatin dose by 50% or consider pravastatin/rosuvastatin',
 'A'),
('2025Q3.harmonized', 'RxCUI:1154343', 'CYP2C19', 'poor', 'moderate',
 'Reduced clopidogrel activation, decreased antiplatelet effect',
 'Consider alternative antiplatelet (prasugrel, ticagrelor) or increase dose',
 'A');

-- Insert sample drug class rules
INSERT INTO ddi_class_rules (
    dataset_version, object_type, object_code, subject_type, subject_code,
    severity, mechanism, clinical_effects, management_strategy, evidence, confidence
) VALUES
('2025Q3.harmonized', 'class', 'ATC:C09AA', 'class', 'ATC:M01AE', 
 'moderate', 'PD', 'Renal perfusion reduction, decreased BP control',
 'Avoid chronic combination; monitor BP and renal function; prefer acetaminophen',
 'B', 0.80),
('2025Q3.harmonized', 'class', 'ATC:N05BA', 'class', 'ATC:N02AA',
 'major', 'PD', 'Additive CNS depression, respiratory depression risk',
 'Avoid combination when possible; if necessary, use lowest doses with close monitoring',
 'A', 0.90),
('2025Q3.harmonized', 'class', 'ATC:B01AA', 'class', 'ATC:B01AC',
 'major', 'PD', 'Additive bleeding risk through different hemostatic pathways',
 'Use combination only when benefits outweigh risks; consider gastroprotection',
 'A', 0.85);

-- Insert sample food/drug interaction modifiers
INSERT INTO ddi_modifiers (
    dataset_version, modifier_type, modifier_code, drug_code, 
    effect, management_strategy, severity, evidence
) VALUES
('2025Q3.harmonized', 'food', 'SNOMED:102259006', 'RxCUI:5781',
 'Hypertensive crisis from tyramine interaction with MAOI',
 'Avoid tyramine-rich foods (aged cheese, wine, cured meats) during MAOI therapy',
 'major', 'A'),
('2025Q3.harmonized', 'alcohol', 'SNOMED:53041004', 'RxCUI:4493',
 'Enhanced CNS depression and respiratory depression',
 'Avoid alcohol consumption; counsel patient on risks',
 'major', 'A'),
('2025Q3.harmonized', 'herbal', 'SNOMED:726542003', 'RxCUI:11289',
 'Ginkgo increases bleeding risk with warfarin',
 'Discontinue ginkgo or monitor INR more frequently',
 'moderate', 'B');

-- Insert sample institutional overrides
INSERT INTO ddi_overrides (
    dataset_version, scope, selector, action, payload, rationale, approver
) VALUES
('2025Q3.harmonized', 'pair', 
 '{"drug1_code": "RxCUI:203496", "drug2_code": "ATC:N06AB"}',
 'amend',
 '{"management_strategy": "CONTRAINDICATED unless no alternatives AND inpatient monitoring with ICU capability"}',
 'Institutional safety policy for linezolid + SSRI combinations',
 'P&T-Committee-2025-08-31');

-- Update existing data to include dataset version
UPDATE drug_interactions 
SET dataset_version = '2025Q3.initial', 
    evidence = 'B',
    confidence = 0.80
WHERE dataset_version IS NULL;

-- Refresh the materialized view
REFRESH MATERIALIZED VIEW ddi_interaction_matrix;

-- Add indexes for enhanced performance
CREATE INDEX idx_drug_interactions_dataset_version ON drug_interactions(dataset_version);
CREATE INDEX idx_drug_interactions_pgx_required ON drug_interactions(pgx_required) WHERE pgx_required = TRUE;
CREATE INDEX idx_drug_interactions_confidence ON drug_interactions(confidence DESC);
CREATE INDEX idx_drug_interactions_evidence ON drug_interactions(evidence);

-- GIN indexes for JSONB columns
CREATE INDEX idx_drug_interactions_pgx_markers ON drug_interactions USING GIN (pgx_markers);
CREATE INDEX idx_drug_interactions_food_interactions ON drug_interactions USING GIN (food_alcohol_herbal);
CREATE INDEX idx_ddi_class_rules_qualifiers ON ddi_class_rules USING GIN (qualifiers);
CREATE INDEX idx_ddi_overrides_selector ON ddi_overrides USING GIN (selector);

-- Function to get current dataset version
CREATE OR REPLACE FUNCTION get_current_dataset_version() RETURNS TEXT AS $$
BEGIN
    RETURN (SELECT version_name FROM ddi_dataset_versions WHERE is_current = TRUE LIMIT 1);
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to atomically switch dataset versions (for ETL updates)
CREATE OR REPLACE FUNCTION switch_dataset_version(new_version TEXT) RETURNS BOOLEAN AS $$
BEGIN
    -- Verify new version exists
    IF NOT EXISTS (SELECT 1 FROM ddi_dataset_versions WHERE version_name = new_version) THEN
        RAISE EXCEPTION 'Dataset version % does not exist', new_version;
    END IF;
    
    -- Atomic switch
    BEGIN
        UPDATE ddi_dataset_versions SET is_current = FALSE WHERE is_current = TRUE;
        UPDATE ddi_dataset_versions SET is_current = TRUE WHERE version_name = new_version;
        
        -- Refresh materialized view with new version
        REFRESH MATERIALIZED VIEW CONCURRENTLY ddi_interaction_matrix;
        
        RETURN TRUE;
    EXCEPTION WHEN OTHERS THEN
        -- Rollback on any error
        RAISE;
    END;
END;
$$ LANGUAGE plpgsql;