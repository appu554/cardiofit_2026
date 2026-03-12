-- KB-21 Migration 002: OutcomeCorrelation entity (Finding F-04 / Gap 5)
-- The most architecturally significant addition from the pre-implementation review.
-- Enables: pharmacological vs. behavioral differential diagnosis
-- Data flow: KB-20 LAB_RESULT events → KB-21 correlation computation → V-MCU consumption

CREATE TABLE IF NOT EXISTS outcome_correlations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,

    -- Correlation period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Behavioral input (from KB-21 AdherenceState)
    mean_adherence_score DECIMAL(5,4) NOT NULL,
    adherence_trend VARCHAR(20) CHECK (adherence_trend IN ('IMPROVING', 'STABLE', 'DECLINING', 'CRITICAL')),
    dominant_phenotype VARCHAR(20) CHECK (dominant_phenotype IN ('CHAMPION', 'STEADY', 'SPORADIC', 'DECLINING', 'DORMANT', 'CHURNED')),

    -- Clinical output (from KB-20 LAB_RESULT events)
    hba1c_start DECIMAL(5,2),
    hba1c_end DECIMAL(5,2),
    hba1c_delta DECIMAL(5,2),         -- negative = improvement
    fbg_mean DECIMAL(6,2),
    fbg_trend VARCHAR(20),             -- IMPROVING, STABLE, WORSENING
    bp_systolic_mean DECIMAL(5,1),

    -- Correlation result
    treatment_response_class VARCHAR(30) NOT NULL CHECK (treatment_response_class IN (
        'CONCORDANT',        -- adherence↑ + outcome↑ → treatment working
        'DISCORDANT',        -- adherence↑ + outcome flat → pharmacological issue
        'BEHAVIORAL_GAP',    -- adherence↓ + outcome↓ → fix behavior first
        'INSUFFICIENT_DATA'  -- not enough data points
    )),
    correlation_strength DECIMAL(5,4),  -- 0.0–1.0
    confidence_level VARCHAR(20) DEFAULT 'LOW' CHECK (confidence_level IN ('LOW', 'MODERATE', 'HIGH')),

    -- Reinforcement (Gap 5 Q3: when should the system celebrate?)
    celebration_eligible BOOLEAN DEFAULT FALSE,
    celebration_message TEXT,

    computed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_outcome_patient ON outcome_correlations(patient_id);
CREATE INDEX idx_outcome_patient_period ON outcome_correlations(patient_id, period_end DESC);
CREATE INDEX idx_outcome_response_class ON outcome_correlations(treatment_response_class);
CREATE INDEX idx_outcome_celebration ON outcome_correlations(patient_id)
    WHERE celebration_eligible = TRUE;

-- View: Latest correlation per patient for V-MCU quick lookup
CREATE OR REPLACE VIEW v_latest_outcome_correlation AS
SELECT DISTINCT ON (patient_id)
    id, patient_id, period_start, period_end,
    mean_adherence_score, adherence_trend, dominant_phenotype,
    hba1c_start, hba1c_end, hba1c_delta, fbg_mean, fbg_trend,
    treatment_response_class, correlation_strength, confidence_level,
    celebration_eligible, computed_at
FROM outcome_correlations
ORDER BY patient_id, period_end DESC;
