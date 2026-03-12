-- KB-13 Quality Measures Engine - Initial Schema
-- Migration 001: Core tables for quality measure calculations
--
-- Critical Architecture Notes (CTO/CMO Gate):
--   🔴 Care gaps have SOURCE field - KB-13 gaps are DERIVED, not authoritative
--   🟡 ExecutionContextVersion tracked for all calculations
--   🟡 Benchmarks versioned separately by year

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- MEASURE DEFINITIONS (cached from YAML, not source of truth)
-- =====================================================
CREATE TABLE IF NOT EXISTS measures (
    id VARCHAR(100) PRIMARY KEY,
    version VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    description TEXT,
    type VARCHAR(50) NOT NULL,  -- PROCESS, OUTCOME, STRUCTURE, etc.
    scoring VARCHAR(50) NOT NULL,  -- proportion, ratio, continuous
    domain VARCHAR(100) NOT NULL,  -- DIABETES, CARDIOVASCULAR, etc.
    program VARCHAR(50) NOT NULL,  -- HEDIS, CMS, MIPS, etc.
    nqf_number VARCHAR(50),
    cms_number VARCHAR(50),
    hedis_code VARCHAR(50),
    improvement_notation VARCHAR(50) DEFAULT 'increase',
    active BOOLEAN DEFAULT true,
    definition_yaml JSONB,  -- Full YAML definition for reference
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_measures_program ON measures(program);
CREATE INDEX idx_measures_domain ON measures(domain);
CREATE INDEX idx_measures_active ON measures(active);
CREATE INDEX idx_measures_cms_number ON measures(cms_number);

-- =====================================================
-- BENCHMARKS (Versioned by year per CTO/CMO architecture)
-- =====================================================
CREATE TABLE IF NOT EXISTS benchmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    measure_id VARCHAR(100) NOT NULL REFERENCES measures(id),
    year INTEGER NOT NULL,
    source VARCHAR(100) NOT NULL,  -- NCQA, CMS, etc.
    effective_date DATE NOT NULL,
    percentile_25 DECIMAL(5,2),
    percentile_50 DECIMAL(5,2),
    percentile_75 DECIMAL(5,2),
    percentile_90 DECIMAL(5,2),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(measure_id, year, source)
);

CREATE INDEX idx_benchmarks_measure_year ON benchmarks(measure_id, year);

-- =====================================================
-- CALCULATION RESULTS
-- =====================================================
CREATE TABLE IF NOT EXISTS calculation_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    measure_id VARCHAR(100) NOT NULL REFERENCES measures(id),
    report_type VARCHAR(50) NOT NULL,  -- individual, subject-list, summary
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Population counts
    initial_population INTEGER NOT NULL DEFAULT 0,
    denominator INTEGER NOT NULL DEFAULT 0,
    denominator_exclusion INTEGER NOT NULL DEFAULT 0,
    denominator_exception INTEGER NOT NULL DEFAULT 0,
    numerator INTEGER NOT NULL DEFAULT 0,
    numerator_exclusion INTEGER NOT NULL DEFAULT 0,

    -- Calculated score
    score DECIMAL(5,4),  -- 0.0000 to 1.0000

    -- Stratifications stored as JSONB
    stratifications JSONB,

    -- Execution context (🟡 REQUIRED per CTO/CMO gate)
    execution_context JSONB NOT NULL,
    -- Example: {
    --   "kb13_version": "1.0.0",
    --   "cql_library_version": "1.0.0",
    --   "terminology_version": "2024-01",
    --   "measure_yaml_version": "12.0.0",
    --   "executed_at": "2024-01-15T10:30:00Z"
    -- }

    execution_time_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_calc_results_measure ON calculation_results(measure_id);
CREATE INDEX idx_calc_results_period ON calculation_results(period_start, period_end);
CREATE INDEX idx_calc_results_created ON calculation_results(created_at DESC);

-- =====================================================
-- CARE GAPS (🔴 CRITICAL: Source annotation required)
-- =====================================================
CREATE TABLE IF NOT EXISTS care_gaps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    measure_id VARCHAR(100) NOT NULL REFERENCES measures(id),
    subject_id VARCHAR(255) NOT NULL,  -- Patient or organization ID
    gap_type VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',  -- critical, high, medium, low
    status VARCHAR(50) NOT NULL DEFAULT 'open',  -- open, in-progress, closed, deferred
    due_date DATE,
    intervention TEXT,

    -- 🔴 CRITICAL: Source annotation (KB-13 gaps are DERIVED)
    source VARCHAR(50) NOT NULL DEFAULT 'QUALITY_MEASURE',
    -- Valid values: 'QUALITY_MEASURE' (KB-13), 'PATIENT_CDS' (KB-9)

    -- 🔴 CRITICAL: KB-13 gaps are NOT authoritative
    is_authoritative BOOLEAN NOT NULL DEFAULT false,
    -- KB-9 care gaps have is_authoritative = true
    -- KB-13 care gaps MUST have is_authoritative = false

    -- Calculation reference
    calculation_result_id UUID REFERENCES calculation_results(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- Constraint: KB-13 care gaps must be non-authoritative
ALTER TABLE care_gaps ADD CONSTRAINT chk_kb13_non_authoritative
    CHECK (source != 'QUALITY_MEASURE' OR is_authoritative = false);

CREATE INDEX idx_care_gaps_measure ON care_gaps(measure_id);
CREATE INDEX idx_care_gaps_subject ON care_gaps(subject_id);
CREATE INDEX idx_care_gaps_status ON care_gaps(status);
CREATE INDEX idx_care_gaps_priority ON care_gaps(priority);
CREATE INDEX idx_care_gaps_source ON care_gaps(source);

-- =====================================================
-- CALCULATION JOBS (for async batch processing)
-- =====================================================
CREATE TABLE IF NOT EXISTS calculation_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    measure_id VARCHAR(100) NOT NULL REFERENCES measures(id),
    report_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, running, completed, failed
    progress INTEGER DEFAULT 0,  -- 0-100
    result_id UUID REFERENCES calculation_results(id),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_calc_jobs_status ON calculation_jobs(status);
CREATE INDEX idx_calc_jobs_created ON calculation_jobs(created_at DESC);

-- =====================================================
-- DASHBOARD CACHE (for performance)
-- =====================================================
CREATE TABLE IF NOT EXISTS dashboard_cache (
    id VARCHAR(100) PRIMARY KEY,  -- e.g., 'summary', 'trends:30d'
    data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dashboard_cache_expires ON dashboard_cache(expires_at);

-- =====================================================
-- AUDIT LOG (for compliance)
-- =====================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255),
    user_id VARCHAR(255),
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_log_event ON audit_log(event_type);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at DESC);

-- =====================================================
-- FUNCTIONS
-- =====================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_measures_updated_at
    BEFORE UPDATE ON measures
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_care_gaps_updated_at
    BEFORE UPDATE ON care_gaps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- COMMENTS (documentation)
-- =====================================================
COMMENT ON TABLE measures IS 'Quality measure definitions (cached from YAML)';
COMMENT ON TABLE benchmarks IS 'Versioned performance benchmarks by year';
COMMENT ON TABLE calculation_results IS 'Results of quality measure calculations';
COMMENT ON TABLE care_gaps IS 'Identified care gaps (KB-13 = DERIVED, KB-9 = AUTHORITATIVE)';
COMMENT ON COLUMN care_gaps.source IS 'QUALITY_MEASURE (KB-13) or PATIENT_CDS (KB-9)';
COMMENT ON COLUMN care_gaps.is_authoritative IS 'KB-13 gaps must be false, KB-9 gaps are true';
COMMENT ON COLUMN calculation_results.execution_context IS 'Required version tracking per CTO/CMO gate';
