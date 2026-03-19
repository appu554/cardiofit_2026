-- BCE v1.0: Technique effectiveness (Bayesian learning), motivation phases, intake profiles

CREATE TABLE IF NOT EXISTS technique_effectiveness (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    technique VARCHAR(10) NOT NULL,
    alpha DECIMAL(8,4) NOT NULL DEFAULT 1.0,
    beta DECIMAL(8,4) NOT NULL DEFAULT 1.0,
    posterior_mean DECIMAL(5,4) NOT NULL DEFAULT 0.5,
    deliveries INT DEFAULT 0,
    successes INT DEFAULT 0,
    last_delivered TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(patient_id, technique)
);

CREATE INDEX idx_tech_eff_patient ON technique_effectiveness(patient_id);

CREATE TABLE IF NOT EXISTS patient_motivation_phases (
    patient_id VARCHAR(100) PRIMARY KEY,
    phase VARCHAR(20) NOT NULL DEFAULT 'INITIATION',
    phase_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cycle_day_start INT NOT NULL DEFAULT 1,
    cycle_day INT NOT NULL DEFAULT 1,
    previous_phase VARCHAR(20),
    transitioned_at TIMESTAMPTZ,
    pre_recovery_phase VARCHAR(20),
    recovery_count INT DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS intake_profiles (
    patient_id VARCHAR(100) PRIMARY KEY,
    age_band VARCHAR(10),
    education_level VARCHAR(20),
    smartphone_literacy VARCHAR(20),
    self_efficacy DECIMAL(3,2) DEFAULT 0.5,
    family_structure VARCHAR(20),
    employment_status VARCHAR(20),
    prior_program_success BOOLEAN,
    first_response_latency_ms BIGINT DEFAULT 0,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
