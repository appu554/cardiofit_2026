-- KB-21 Behavioral Intelligence Service — Initial Schema
-- Implements: InteractionEvent, AdherenceState, EngagementProfile, QuestionTelemetry,
--             NudgeRecord, DietarySignal, BarrierDetection
-- References: Vaidshala KB-21 Pre-Implementation Review (March 2026)

-- ══════════════════════════════════════════════════
-- INTERACTION EVENTS — Raw patient interaction log
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS interaction_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('WHATSAPP', 'SMS', 'IVR', 'APP', 'CLINIC')),
    type VARCHAR(30) NOT NULL CHECK (type IN (
        'DAILY_CHECKIN', 'MEDICATION_CONFIRM', 'SYMPTOM_REPORT',
        'LAB_REPORT', 'NUDGE_RESPONSE', 'ONBOARDING', 'HPI_SESSION'
    )),

    -- Interaction content
    question_id VARCHAR(100),
    response_value TEXT,
    response_quality VARCHAR(20) CHECK (response_quality IN ('HIGH', 'MODERATE', 'LOW', 'PATA_NAHI')),
    response_latency_ms BIGINT DEFAULT 0,

    -- Medication context
    drug_class VARCHAR(100),
    medication_id VARCHAR(100),  -- FDC-linked tracking (Finding F-07)

    -- Dietary signals (Finding F-05 — Gap 3 Circle 1)
    evening_meal_confirmed BOOLEAN,
    fasting_today BOOLEAN,

    -- Metadata
    session_id VARCHAR(100),
    language_code VARCHAR(10) DEFAULT 'hi',
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_interaction_events_patient ON interaction_events(patient_id);
CREATE INDEX idx_interaction_events_patient_time ON interaction_events(patient_id, timestamp DESC);
CREATE INDEX idx_interaction_events_drug_class ON interaction_events(drug_class) WHERE drug_class IS NOT NULL;
CREATE INDEX idx_interaction_events_session ON interaction_events(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_interaction_events_type_time ON interaction_events(type, timestamp DESC);

-- ══════════════════════════════════════════════════
-- ADHERENCE STATE — Per-drug-class medication adherence
-- Includes 30-day weighted + 7-day unweighted scores (Finding F-08)
-- FDC-aware: single record for FDC, projected to components (Finding F-07)
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS adherence_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,

    -- Drug identification
    drug_class VARCHAR(100) NOT NULL,
    medication_id VARCHAR(100),
    is_fdc BOOLEAN DEFAULT FALSE,
    fdc_components TEXT, -- JSON array of component drug classes

    -- Adherence scores
    adherence_score DECIMAL(5,4) NOT NULL DEFAULT 0,     -- 30-day recency-weighted
    adherence_score_7d DECIMAL(5,4) NOT NULL DEFAULT 0,  -- 7-day unweighted (V-MCU consumes)
    data_quality VARCHAR(20) DEFAULT 'LOW' CHECK (data_quality IN ('HIGH', 'MODERATE', 'LOW')),

    -- Trend analysis
    adherence_trend VARCHAR(20) DEFAULT 'STABLE' CHECK (adherence_trend IN ('IMPROVING', 'STABLE', 'DECLINING', 'CRITICAL')),
    trend_slope_per_week DECIMAL(5,4) DEFAULT 0,

    -- Computation metadata
    total_check_ins INT DEFAULT 0,
    responded_check_ins INT DEFAULT 0,
    confirmed_doses INT DEFAULT 0,
    missed_doses INT DEFAULT 0,
    last_confirmed_at TIMESTAMPTZ,
    last_missed_at TIMESTAMPTZ,

    -- Barrier tracking
    primary_barrier VARCHAR(30),
    barrier_codes TEXT, -- JSON array

    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_adherence_patient_drug UNIQUE (patient_id, drug_class)
);

CREATE INDEX idx_adherence_patient ON adherence_states(patient_id);
CREATE INDEX idx_adherence_score ON adherence_states(adherence_score) WHERE adherence_score < 0.50;

-- ══════════════════════════════════════════════════
-- ENGAGEMENT PROFILE — Overall behavioral profile per patient
-- Contains loop_trust_score (Finding F-01 — Gap 1)
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS engagement_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) UNIQUE NOT NULL,

    -- Engagement score
    engagement_score DECIMAL(5,4) NOT NULL DEFAULT 0,

    -- Behavioral phenotype
    phenotype VARCHAR(20) NOT NULL DEFAULT 'STEADY'
        CHECK (phenotype IN ('CHAMPION', 'STEADY', 'SPORADIC', 'DECLINING', 'DORMANT', 'CHURNED')),
    phenotype_since TIMESTAMPTZ,
    previous_phenotype VARCHAR(20),

    -- Loop trust score (Finding F-01)
    loop_trust_score DECIMAL(5,4) NOT NULL DEFAULT 0,
    data_quality_weight DECIMAL(5,4) DEFAULT 1.0,
    phenotype_weight DECIMAL(5,4) DEFAULT 1.0,
    temporal_stability DECIMAL(5,4) DEFAULT 1.0,

    -- Engagement metrics
    total_interactions INT DEFAULT 0,
    interactions_last_7d INT DEFAULT 0,
    interactions_last_30d INT DEFAULT 0,
    avg_response_latency_ms BIGINT DEFAULT 0,
    preferred_channel VARCHAR(20),
    preferred_language VARCHAR(10) DEFAULT 'hi',
    last_interaction_at TIMESTAMPTZ,
    days_since_last_interaction INT DEFAULT 0,

    -- Decay prediction
    decay_risk_score DECIMAL(5,4) DEFAULT 0,
    predicted_churn_at TIMESTAMPTZ,

    -- Onboarding
    onboarding_status VARCHAR(20) DEFAULT 'NOT_STARTED'
        CHECK (onboarding_status IN ('NOT_STARTED', 'IN_PROGRESS', 'COMPLETED')),

    -- Device change detection (Finding F-09)
    device_change_suspected BOOLEAN DEFAULT FALSE,
    last_verified_at TIMESTAMPTZ,

    -- Privacy / DPDPA (Finding F-10)
    consent_for_festival_adapt BOOLEAN DEFAULT FALSE,
    retention_policy_months INT DEFAULT 24,

    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_engagement_phenotype ON engagement_profiles(phenotype);
CREATE INDEX idx_engagement_loop_trust ON engagement_profiles(loop_trust_score);
CREATE INDEX idx_engagement_decay ON engagement_profiles(decay_risk_score DESC) WHERE decay_risk_score > 0.60;

-- ══════════════════════════════════════════════════
-- QUESTION TELEMETRY — Check-in question effectiveness
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS question_telemetry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id VARCHAR(100) NOT NULL,
    language VARCHAR(10) NOT NULL,

    question_text TEXT NOT NULL,
    category VARCHAR(50), -- ADHERENCE, SYMPTOM, DIETARY, ONBOARDING

    -- Effectiveness metrics
    times_asked INT DEFAULT 0,
    times_answered INT DEFAULT 0,
    times_pata_nahi INT DEFAULT 0,
    response_rate DECIMAL(5,4) DEFAULT 0,
    pata_nahi_rate DECIMAL(5,4) DEFAULT 0,
    avg_latency_ms BIGINT DEFAULT 0,
    information_yield DECIMAL(5,4) DEFAULT 0,

    active BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_question_lang UNIQUE (question_id, language)
);

-- ══════════════════════════════════════════════════
-- NUDGE RECORDS — Automated patient communication tracking
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS nudge_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    nudge_type VARCHAR(40) NOT NULL CHECK (nudge_type IN (
        'MEDICATION_REMINDER', 'BARRIER_SUPPORT', 'POSITIVE_REINFORCEMENT',
        'OUTCOME_LINKED_CELEBRATION', 'RE_ENGAGEMENT', 'EDUCATIONAL'
    )),

    channel VARCHAR(20) NOT NULL,
    message_text TEXT,
    language VARCHAR(10) DEFAULT 'hi',

    trigger_reason TEXT,
    barrier_code VARCHAR(30),

    sent_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    responded_at TIMESTAMPTZ,
    response_type VARCHAR(30), -- POSITIVE, NEGATIVE, IGNORED

    adherence_pre_nudge DECIMAL(5,4) DEFAULT 0,
    adherence_post_nudge DECIMAL(5,4) DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_nudge_patient ON nudge_records(patient_id);
CREATE INDEX idx_nudge_patient_time ON nudge_records(patient_id, sent_at DESC);

-- ══════════════════════════════════════════════════
-- DIETARY SIGNALS — Lightweight meal adherence (Finding F-05)
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS dietary_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    date DATE NOT NULL,

    -- Circle 1 signals
    evening_meal_confirmed BOOLEAN NOT NULL,
    fasting_today BOOLEAN DEFAULT FALSE,
    fasting_reason VARCHAR(30), -- RELIGIOUS, MEDICAL, VOLUNTARY

    -- Circle 2 extension
    meal_regularity_score DECIMAL(5,4),
    carb_estimate_category VARCHAR(20), -- LOW, MODERATE, HIGH
    dietary_barrier_codes TEXT, -- JSON array

    source VARCHAR(20) DEFAULT 'SELF_REPORT',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dietary_patient_date ON dietary_signals(patient_id, date DESC);

-- ══════════════════════════════════════════════════
-- BARRIER DETECTION — Identified adherence barriers
-- ══════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS barrier_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    drug_class VARCHAR(100),
    barrier VARCHAR(30) NOT NULL CHECK (barrier IN (
        'FORGETFULNESS', 'SIDE_EFFECTS', 'COST', 'CULTURAL',
        'FASTING', 'KNOWLEDGE', 'ACCESS', 'POLYPHARMACY'
    )),

    detected_at TIMESTAMPTZ NOT NULL,
    detection_method VARCHAR(30), -- SELF_REPORT, PATTERN_ANALYSIS, CLINICIAN
    confidence DECIMAL(5,4),

    intervention_type VARCHAR(50),
    intervention_sent BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolution_outcome VARCHAR(30), -- RESOLVED, ONGOING, ESCALATED

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_barrier_patient ON barrier_detections(patient_id);
CREATE INDEX idx_barrier_unresolved ON barrier_detections(patient_id) WHERE resolved_at IS NULL;
