-- KB-26 Metabolic Digital Twin — Initial Schema
-- Applied by GORM AutoMigrate in development; this file is the canonical DDL for production.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- twin_states: append-only versioned snapshots of patient metabolic state
-- =============================================================================
CREATE TABLE IF NOT EXISTS twin_states (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id            UUID NOT NULL,
    state_version         INT NOT NULL,
    update_source         VARCHAR(50) NOT NULL,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Tier 1: Directly Measured
    fbg7d_mean            DOUBLE PRECISION,
    fbg14d_trend          VARCHAR(255),
    ppbg7d_mean           DOUBLE PRECISION,
    hb_a1c                DOUBLE PRECISION,
    hb_a1c_date           TIMESTAMPTZ,
    sbp14d_mean           DOUBLE PRECISION,
    dbp14d_mean           DOUBLE PRECISION,
    egfr                  DOUBLE PRECISION,
    egfr_date             TIMESTAMPTZ,
    waist_cm              DOUBLE PRECISION,
    weight_kg             DOUBLE PRECISION,
    bmi                   DOUBLE PRECISION,
    daily_steps7d_mean    DOUBLE PRECISION,
    resting_hr            DOUBLE PRECISION,

    -- Tier 2: Reliably Derived
    visceral_fat_proxy    DOUBLE PRECISION,
    visceral_fat_trend    VARCHAR(255),
    renal_slope           DOUBLE PRECISION,
    renal_classification  VARCHAR(255),
    map_value             DOUBLE PRECISION,
    glycemic_variability  DOUBLE PRECISION,
    dawn_phenomenon       BOOLEAN,
    trig_hdl_ratio        DOUBLE PRECISION,

    -- Tier 3: Estimated (JSONB)
    insulin_sensitivity   JSONB,
    hepatic_glucose_output JSONB,
    muscle_mass_proxy     JSONB,
    beta_cell_function    JSONB,
    sympathetic_tone      JSONB
);

CREATE INDEX IF NOT EXISTS idx_twin_patient
    ON twin_states (patient_id, updated_at DESC);

-- =============================================================================
-- calibrated_effects: patient-specific Bayesian-calibrated treatment effects
-- =============================================================================
CREATE TABLE IF NOT EXISTS calibrated_effects (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id        UUID NOT NULL,
    kb25_edge_type    VARCHAR(50) NOT NULL,
    intervention_code VARCHAR(50) NOT NULL,
    target_variable   VARCHAR(50) NOT NULL,
    population_effect DOUBLE PRECISION NOT NULL,
    patient_effect    DOUBLE PRECISION NOT NULL,
    observations      INT NOT NULL DEFAULT 0,
    confidence        DOUBLE PRECISION NOT NULL DEFAULT 0,
    prior_mean        DOUBLE PRECISION,
    prior_sd          DOUBLE PRECISION,
    posterior_mean    DOUBLE PRECISION,
    posterior_sd      DOUBLE PRECISION,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_calibrated_patient
    ON calibrated_effects (patient_id);

-- =============================================================================
-- simulation_runs: recorded simulation results for audit and comparison
-- =============================================================================
CREATE TABLE IF NOT EXISTS simulation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    intervention    JSONB NOT NULL,
    projection_days INT NOT NULL,
    results         JSONB NOT NULL,
    twin_state_id   UUID,
    requested_by    VARCHAR(50)
);

CREATE INDEX IF NOT EXISTS idx_sim_patient
    ON simulation_runs (patient_id, requested_at DESC);
