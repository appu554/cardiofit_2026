-- KB-23 Decision Cards Engine — Initial Schema
-- Tables: decision_cards, card_recommendations, card_templates, summary_fragments,
--         mcu_gate_history, composite_card_signals, hypoglycaemia_alerts

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. decision_cards
CREATE TABLE IF NOT EXISTS decision_cards (
    card_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    session_id UUID,
    snapshot_id UUID,
    template_id VARCHAR(100) NOT NULL,
    node_id VARCHAR(50) NOT NULL,
    primary_differential_id VARCHAR(100),
    primary_posterior DOUBLE PRECISION,
    diagnostic_confidence_tier VARCHAR(20) NOT NULL,  -- FIRM/PROBABLE/POSSIBLE/UNCERTAIN
    confidence_tier_decayed BOOLEAN DEFAULT FALSE,
    confidence_tier_decay_reason TEXT,
    mcu_gate VARCHAR(10) NOT NULL,  -- SAFE/MODIFY/PAUSE/HALT
    mcu_gate_rationale TEXT,
    dose_adjustment_notes TEXT,
    observation_reliability VARCHAR(20) DEFAULT 'HIGH',  -- HIGH/MODERATE/LOW
    secondary_differentials JSONB,
    clinician_summary TEXT,
    patient_summary_en TEXT,
    patient_summary_hi TEXT,
    patient_summary_local TEXT,
    patient_safety_instructions JSONB,
    locale_code VARCHAR(10),
    safety_tier VARCHAR(20) NOT NULL,  -- IMMEDIATE/URGENT/ROUTINE
    recurrence_count INTEGER DEFAULT 0,
    card_source VARCHAR(30) NOT NULL,  -- KB22_SESSION/HYPOGLYCAEMIA_FAST_PATH/PERTURBATION_DECAY/BEHAVIORAL_GAP
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE/SUPERSEDED/PENDING_REAFFIRMATION/ARCHIVED
    pending_reaffirmation BOOLEAN DEFAULT FALSE,
    re_entry_protocol BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    superseded_at TIMESTAMPTZ,
    superseded_by UUID
);

CREATE INDEX idx_decision_cards_patient_id ON decision_cards(patient_id);
CREATE INDEX idx_decision_cards_session_id ON decision_cards(session_id);
CREATE INDEX idx_decision_cards_template_id ON decision_cards(template_id);
CREATE INDEX idx_decision_cards_node_id ON decision_cards(node_id);
CREATE INDEX idx_decision_cards_status ON decision_cards(status);
CREATE INDEX idx_decision_cards_patient_status ON decision_cards(patient_id, status);
CREATE INDEX idx_decision_cards_created_at ON decision_cards(created_at);

-- 2. card_recommendations
CREATE TABLE IF NOT EXISTS card_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id UUID NOT NULL REFERENCES decision_cards(card_id) ON DELETE CASCADE,
    rec_type VARCHAR(30) NOT NULL,
    urgency VARCHAR(20) NOT NULL,
    target VARCHAR(100),
    action_text_en TEXT,
    action_text_hi TEXT,
    rationale_en TEXT,
    guideline_ref VARCHAR(100),
    confidence_tier_required VARCHAR(20),
    bypasses_confidence_gate BOOLEAN DEFAULT FALSE,
    trigger_condition_en TEXT,
    trigger_condition_hi TEXT,
    from_secondary_differential BOOLEAN DEFAULT FALSE,
    conflict_flag BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_card_recommendations_card_id ON card_recommendations(card_id);

-- 3. card_templates (audit reference — data loaded from YAML)
CREATE TABLE IF NOT EXISTS card_templates (
    template_id VARCHAR(100) PRIMARY KEY,
    node_id VARCHAR(50) NOT NULL,
    differential_id VARCHAR(100),
    template_version VARCHAR(20),
    content_sha256 VARCHAR(64),
    confidence_thresholds JSONB,
    mcu_gate_default VARCHAR(10),
    recommendations_count INTEGER DEFAULT 0,
    has_safety_instructions BOOLEAN DEFAULT FALSE,
    requires_dose_adjustment_notes BOOLEAN DEFAULT FALSE,
    clinical_reviewer VARCHAR(100),
    approved_at TIMESTAMPTZ,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_card_templates_node_id ON card_templates(node_id);

-- 4. summary_fragments (audit reference — data loaded from YAML)
CREATE TABLE IF NOT EXISTS summary_fragments (
    fragment_id VARCHAR(100) PRIMARY KEY,
    template_id VARCHAR(100) NOT NULL,
    fragment_type VARCHAR(30) NOT NULL,  -- CLINICIAN/PATIENT/SAFETY_INSTRUCTION
    text_en TEXT NOT NULL,
    text_hi TEXT NOT NULL,
    text_local TEXT,
    locale_code VARCHAR(10),
    patient_advocate_reviewed_by VARCHAR(100),
    reading_level_validated BOOLEAN DEFAULT FALSE,
    guideline_ref VARCHAR(100),
    version VARCHAR(20)
);

CREATE INDEX idx_summary_fragments_template_id ON summary_fragments(template_id);

-- 5. mcu_gate_history (N-01 hysteresis tracking)
CREATE TABLE IF NOT EXISTS mcu_gate_history (
    history_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    card_id UUID NOT NULL,
    gate_value VARCHAR(10) NOT NULL,
    previous_gate VARCHAR(10),
    session_id UUID,
    transition_reason TEXT,
    clinician_resume_by VARCHAR(100),
    clinician_resume_reason TEXT,
    re_entry_protocol BOOLEAN DEFAULT FALSE,
    halt_duration_hours DOUBLE PRECISION,
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mcu_gate_history_patient_id ON mcu_gate_history(patient_id);
CREATE INDEX idx_mcu_gate_history_created_at ON mcu_gate_history(created_at);
CREATE INDEX idx_mcu_gate_history_patient_created ON mcu_gate_history(patient_id, created_at DESC);

-- 6. composite_card_signals (72-hour synthesis)
CREATE TABLE IF NOT EXISTS composite_card_signals (
    composite_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    card_ids JSONB,
    most_restrictive_gate VARCHAR(10),
    recurrence_count INTEGER DEFAULT 0,
    urgency_upgraded BOOLEAN DEFAULT FALSE,
    synthesis_summary_en TEXT,
    synthesis_summary_hi TEXT,
    window_start TIMESTAMPTZ,
    window_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_composite_card_signals_patient_id ON composite_card_signals(patient_id);

-- 7. hypoglycaemia_alerts (V-08 fast-path)
CREATE TABLE IF NOT EXISTS hypoglycaemia_alerts (
    alert_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    source VARCHAR(20) NOT NULL,
    glucose_mmol_l DOUBLE PRECISION,
    duration_minutes INTEGER,
    severity VARCHAR(10) NOT NULL,
    predicted_at_hours DOUBLE PRECISION,
    halt_source VARCHAR(10),
    generated_card_id UUID,
    event_timestamp TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hypoglycaemia_alerts_patient_id ON hypoglycaemia_alerts(patient_id);
CREATE INDEX idx_hypoglycaemia_alerts_event_timestamp ON hypoglycaemia_alerts(event_timestamp);
