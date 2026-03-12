-- V3 Clinical Guideline Curation Pipeline - Facts Store Schema
-- Database: v3_facts
-- Purpose: Store extracted clinical facts from guidelines

-- ═══════════════════════════════════════════════════════════════════════════════
-- KB-1: Drug Dosing Facts
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS kb1_drug_dosing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rxnorm_code VARCHAR(20) NOT NULL,
    drug_name VARCHAR(255) NOT NULL,

    -- Renal adjustment data (matches Go struct)
    egfr_min DECIMAL(10,2),
    egfr_max DECIMAL(10,2),
    adjustment_factor DECIMAL(5,3),
    max_dose DECIMAL(10,2),
    max_dose_unit VARCHAR(20),
    recommendation TEXT,
    contraindicated BOOLEAN DEFAULT FALSE,
    action_type VARCHAR(50),  -- CONTRAINDICATED, REDUCE_DOSE, MONITOR, NO_CHANGE

    -- Provenance
    source_authority VARCHAR(100) NOT NULL,
    source_document VARCHAR(500),
    source_section VARCHAR(100),
    source_page INT,
    source_snippet TEXT,
    evidence_level VARCHAR(10),
    effective_date DATE,

    -- Extraction metadata
    extraction_timestamp TIMESTAMPTZ DEFAULT NOW(),
    extractor_version VARCHAR(20),
    review_status VARCHAR(50) DEFAULT 'PENDING_REVIEW',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_kb1_rxnorm ON kb1_drug_dosing(rxnorm_code);
CREATE INDEX idx_kb1_drug_name ON kb1_drug_dosing(drug_name);
CREATE INDEX idx_kb1_egfr_range ON kb1_drug_dosing(egfr_min, egfr_max);
CREATE INDEX idx_kb1_review_status ON kb1_drug_dosing(review_status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- KB-4: Patient Safety Facts (Contraindications)
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS kb4_contraindications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rxnorm_code VARCHAR(20) NOT NULL,
    drug_name VARCHAR(255) NOT NULL,

    -- Contraindication data (matches Go struct)
    condition_codes TEXT[],          -- ICD-10 codes
    condition_descriptions TEXT[],
    snomed_codes TEXT[],
    contraindication_type VARCHAR(20) NOT NULL,  -- absolute, relative
    severity VARCHAR(20) NOT NULL,   -- CRITICAL, HIGH, MODERATE, LOW
    clinical_rationale TEXT,
    alternative_considerations TEXT,

    -- Lab-based contraindication
    lab_parameter VARCHAR(100),
    lab_loinc_code VARCHAR(20),
    lab_threshold_operator VARCHAR(10),
    lab_threshold_value DECIMAL(10,2),
    lab_threshold_unit VARCHAR(20),

    -- Provenance
    source_authority VARCHAR(100) NOT NULL,
    source_document VARCHAR(500),
    source_section VARCHAR(100),
    evidence_level VARCHAR(10),
    effective_date DATE,

    -- Extraction metadata
    extraction_timestamp TIMESTAMPTZ DEFAULT NOW(),
    extractor_version VARCHAR(20),
    review_status VARCHAR(50) DEFAULT 'PENDING_REVIEW',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_kb4_rxnorm ON kb4_contraindications(rxnorm_code);
CREATE INDEX idx_kb4_severity ON kb4_contraindications(severity);
CREATE INDEX idx_kb4_type ON kb4_contraindications(contraindication_type);
CREATE INDEX idx_kb4_review_status ON kb4_contraindications(review_status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- KB-16: Lab Monitoring Requirements
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS kb16_lab_monitoring (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rxnorm_code VARCHAR(20) NOT NULL,
    drug_name VARCHAR(255) NOT NULL,

    -- Lab monitoring data (matches Go struct)
    lab_name VARCHAR(255) NOT NULL,
    loinc_code VARCHAR(20),
    frequency VARCHAR(100),
    baseline_required BOOLEAN DEFAULT FALSE,
    monitoring_required BOOLEAN DEFAULT TRUE,

    -- Critical thresholds
    critical_high_value DECIMAL(10,2),
    critical_high_unit VARCHAR(20),
    critical_high_action TEXT,
    critical_low_value DECIMAL(10,2),
    critical_low_unit VARCHAR(20),
    critical_low_action TEXT,

    -- Target range
    target_min DECIMAL(10,2),
    target_max DECIMAL(10,2),
    target_unit VARCHAR(20),

    -- Provenance
    source_authority VARCHAR(100) NOT NULL,
    source_document VARCHAR(500),
    source_section VARCHAR(100),
    evidence_level VARCHAR(10),
    effective_date DATE,

    -- Extraction metadata
    extraction_timestamp TIMESTAMPTZ DEFAULT NOW(),
    extractor_version VARCHAR(20),
    review_status VARCHAR(50) DEFAULT 'PENDING_REVIEW',
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_kb16_rxnorm ON kb16_lab_monitoring(rxnorm_code);
CREATE INDEX idx_kb16_loinc ON kb16_lab_monitoring(loinc_code);
CREATE INDEX idx_kb16_review_status ON kb16_lab_monitoring(review_status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- Extraction Jobs Tracking
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS extraction_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Job info
    job_name VARCHAR(255) NOT NULL,
    source_pdf VARCHAR(500) NOT NULL,
    source_hash VARCHAR(64) NOT NULL,
    target_kb VARCHAR(20) NOT NULL,  -- dosing, safety, monitoring

    -- Status
    status VARCHAR(50) DEFAULT 'PENDING',  -- PENDING, RUNNING, COMPLETED, FAILED
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,

    -- Results
    facts_extracted INT DEFAULT 0,
    facts_validated INT DEFAULT 0,
    facts_rejected INT DEFAULT 0,

    -- Pipeline stages
    l1_status VARCHAR(20),  -- PENDING, COMPLETED, FAILED
    l1_completed_at TIMESTAMPTZ,
    l2_status VARCHAR(20),
    l2_completed_at TIMESTAMPTZ,
    l3_status VARCHAR(20),
    l3_completed_at TIMESTAMPTZ,
    l4_status VARCHAR(20),
    l4_completed_at TIMESTAMPTZ,
    l5_status VARCHAR(20),
    l5_completed_at TIMESTAMPTZ,

    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_status ON extraction_jobs(status);
CREATE INDEX idx_jobs_source ON extraction_jobs(source_hash);

-- ═══════════════════════════════════════════════════════════════════════════════
-- Guideline Version Conflicts (from Plan)
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS guideline_version_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_rxnorm_code VARCHAR(20) NOT NULL,
    parameter_name VARCHAR(100) NOT NULL,
    old_guideline_id VARCHAR(100) NOT NULL,
    old_value JSONB NOT NULL,
    new_guideline_id VARCHAR(100) NOT NULL,
    new_value JSONB NOT NULL,
    conflict_type VARCHAR(50) NOT NULL,  -- THRESHOLD_CHANGE, RECOMMENDATION_CHANGE, REMOVAL
    resolution_status VARCHAR(50) DEFAULT 'PENDING',  -- PENDING, APPROVED, REJECTED
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_version_conflicts_pending
ON guideline_version_conflicts(drug_rxnorm_code)
WHERE resolution_status = 'PENDING';

-- ═══════════════════════════════════════════════════════════════════════════════
-- CQL Gap Tracking
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS cql_gaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gap_type VARCHAR(50) NOT NULL,  -- FORWARD_GAP, BACKWARD_GAP, COVERAGE_GAP

    -- For forward/coverage gaps
    drug_rxnorm_code VARCHAR(20),
    drug_name VARCHAR(255),
    guideline_authority VARCHAR(100),
    guideline_section VARCHAR(100),
    fact_type VARCHAR(50),

    -- For backward gaps
    cql_file VARCHAR(255),
    cql_define VARCHAR(255),
    cql_line INT,

    -- Status
    status VARCHAR(50) DEFAULT 'OPEN',  -- OPEN, IN_PROGRESS, RESOLVED
    action_taken TEXT,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,

    -- Audit
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_gaps_type ON cql_gaps(gap_type);
CREATE INDEX idx_gaps_status ON cql_gaps(status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- Update trigger for updated_at
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_kb1_updated_at BEFORE UPDATE ON kb1_drug_dosing
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kb4_updated_at BEFORE UPDATE ON kb4_contraindications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kb16_updated_at BEFORE UPDATE ON kb16_lab_monitoring
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON extraction_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
