-- KB-16 Lab Interpretation & Trending Service
-- Initial Database Schema
-- Version: 1.0.0

-- =============================================================================
-- LAB RESULTS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS lab_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    value_numeric DECIMAL(15,5),
    value_string TEXT,
    unit VARCHAR(50),
    ref_low DECIMAL(15,5),
    ref_high DECIMAL(15,5),
    ref_text TEXT,
    ref_age_specific BOOLEAN DEFAULT FALSE,
    ref_sex_specific BOOLEAN DEFAULT FALSE,
    collected_at TIMESTAMPTZ NOT NULL,
    reported_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) DEFAULT 'final',
    performer VARCHAR(100),
    encounter_id VARCHAR(100),
    specimen_id VARCHAR(100),
    order_id VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for lab_results
CREATE INDEX IF NOT EXISTS idx_lab_results_patient ON lab_results(patient_id);
CREATE INDEX IF NOT EXISTS idx_lab_results_patient_code ON lab_results(patient_id, code);
CREATE INDEX IF NOT EXISTS idx_lab_results_patient_code_collected ON lab_results(patient_id, code, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_lab_results_collected ON lab_results(collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_lab_results_encounter ON lab_results(encounter_id) WHERE encounter_id IS NOT NULL;

-- =============================================================================
-- INTERPRETATIONS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS interpretations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    result_id UUID REFERENCES lab_results(id) ON DELETE CASCADE,
    flag VARCHAR(20) NOT NULL,
    severity VARCHAR(20),
    is_critical BOOLEAN DEFAULT FALSE,
    is_panic BOOLEAN DEFAULT FALSE,
    requires_action BOOLEAN DEFAULT FALSE,
    deviation_percent DECIMAL(10,2),
    deviation_direction VARCHAR(10),
    delta_check JSONB,
    clinical_comment TEXT,
    recommendations JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for interpretations
CREATE INDEX IF NOT EXISTS idx_interpretations_result ON interpretations(result_id);
CREATE INDEX IF NOT EXISTS idx_interpretations_critical ON interpretations(is_critical) WHERE is_critical = TRUE;
CREATE INDEX IF NOT EXISTS idx_interpretations_panic ON interpretations(is_panic) WHERE is_panic = TRUE;
CREATE INDEX IF NOT EXISTS idx_interpretations_flag ON interpretations(flag);

-- =============================================================================
-- PATIENT BASELINES TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS patient_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    mean DECIMAL(15,5) NOT NULL,
    std_dev DECIMAL(15,5),
    min_value DECIMAL(15,5),
    max_value DECIMAL(15,5),
    sample_count INT DEFAULT 0,
    source VARCHAR(20) DEFAULT 'CALCULATED',
    notes TEXT,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, code)
);

-- Indexes for patient_baselines
CREATE INDEX IF NOT EXISTS idx_baselines_patient ON patient_baselines(patient_id);
CREATE INDEX IF NOT EXISTS idx_baselines_patient_code ON patient_baselines(patient_id, code);

-- =============================================================================
-- RESULT REVIEWS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS result_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    result_id UUID REFERENCES lab_results(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'PENDING',
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,
    action_taken VARCHAR(100),
    kb14_task_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for result_reviews
CREATE INDEX IF NOT EXISTS idx_reviews_result ON result_reviews(result_id);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON result_reviews(status);
CREATE INDEX IF NOT EXISTS idx_reviews_pending ON result_reviews(status, created_at) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_reviews_kb14_task ON result_reviews(kb14_task_id) WHERE kb14_task_id IS NOT NULL;

-- =============================================================================
-- AUDIT LOG TABLE (for compliance)
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor VARCHAR(100),
    changes JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for audit_log
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at DESC);

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS update_lab_results_updated_at ON lab_results;
CREATE TRIGGER update_lab_results_updated_at
    BEFORE UPDATE ON lab_results
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_result_reviews_updated_at ON result_reviews;
CREATE TRIGGER update_result_reviews_updated_at
    BEFORE UPDATE ON result_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE lab_results IS 'Stores laboratory test results with LOINC codes';
COMMENT ON TABLE interpretations IS 'Stores clinical interpretations of lab results';
COMMENT ON TABLE patient_baselines IS 'Stores patient-specific baseline values for lab tests';
COMMENT ON TABLE result_reviews IS 'Tracks review status and actions for lab results';
COMMENT ON TABLE audit_log IS 'Audit trail for compliance and tracking changes';

COMMENT ON COLUMN lab_results.code IS 'LOINC code for the laboratory test';
COMMENT ON COLUMN lab_results.status IS 'Result status: final, preliminary, corrected, cancelled';
COMMENT ON COLUMN interpretations.flag IS 'Interpretation flag: NORMAL, LOW, HIGH, CRITICAL_LOW, CRITICAL_HIGH, PANIC_LOW, PANIC_HIGH';
COMMENT ON COLUMN interpretations.severity IS 'Clinical severity: LOW, MEDIUM, HIGH, CRITICAL';
COMMENT ON COLUMN patient_baselines.source IS 'Baseline source: CALCULATED, MANUAL, IMPORTED';
COMMENT ON COLUMN result_reviews.status IS 'Review status: PENDING, ACKNOWLEDGED, IN_PROGRESS, COMPLETED, ACTIONED';
