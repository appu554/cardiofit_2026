-- KB-4 Patient Safety: Enhanced Schema Implementation
-- Advanced safety schema with rule versioning, signal detection, and override audit

-- Drug safety profiles with comprehensive versioning and dependencies
CREATE TABLE IF NOT EXISTS drug_safety_profiles (
    profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code         TEXT NOT NULL,
    drug_name         TEXT NOT NULL,
    drug_class        TEXT NOT NULL,
    version           TEXT NOT NULL,
    status            TEXT NOT NULL CHECK (status IN ('active','deprecated','draft','suspended')),
    effective_from    TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to      TIMESTAMPTZ,
    
    -- Rule bundles and compilation
    rule_bundle_json  JSONB NOT NULL DEFAULT '{}',
    compiled_rules    BYTEA,
    rule_dependencies JSONB NOT NULL DEFAULT '{}',
    
    -- Monitoring and alerting
    monitoring_json   JSONB NOT NULL DEFAULT '{}',
    alert_thresholds  JSONB NOT NULL DEFAULT '{}',
    
    -- Integration references
    kb2_phenotypes    TEXT[] DEFAULT '{}',
    kb3_guidelines    TEXT[] DEFAULT '{}',
    kb5_interactions  TEXT[] DEFAULT '{}',
    
    -- Governance and ownership
    governance_tag    TEXT NOT NULL,
    clinical_owner    TEXT NOT NULL,
    review_date       DATE NOT NULL DEFAULT (current_date + interval '3 months'),
    risk_tier         INTEGER NOT NULL DEFAULT 2 CHECK (risk_tier BETWEEN 1 AND 3),
    
    -- Audit fields
    created_by        TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints
    UNIQUE (drug_code, version)
);

-- Indexes for drug safety profiles
CREATE INDEX IF NOT EXISTS idx_profiles_code_active 
    ON drug_safety_profiles (drug_code) WHERE status='active';
CREATE INDEX IF NOT EXISTS idx_profiles_class_active 
    ON drug_safety_profiles (drug_class) WHERE status='active';
CREATE INDEX IF NOT EXISTS idx_profiles_review_date 
    ON drug_safety_profiles (review_date) WHERE status='active';
CREATE INDEX IF NOT EXISTS idx_profiles_risk_tier 
    ON drug_safety_profiles (risk_tier);

-- GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_profiles_rules_gin 
    ON drug_safety_profiles USING GIN (rule_bundle_json);
CREATE INDEX IF NOT EXISTS idx_profiles_dependencies_gin 
    ON drug_safety_profiles USING GIN (rule_dependencies);

-- Rule evaluation cache with dependency tracking
CREATE TABLE IF NOT EXISTS rule_evaluation_cache (
    cache_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key         TEXT UNIQUE NOT NULL,
    drug_code         TEXT NOT NULL,
    rule_version      TEXT NOT NULL,
    
    -- Evaluation result
    result            JSONB NOT NULL DEFAULT '{}',
    status            TEXT NOT NULL CHECK (status IN ('PASS','WARN','VETO')),
    
    -- Dependencies for intelligent invalidation
    depends_on_fields TEXT[] DEFAULT '{}',
    depends_on_rules  TEXT[] DEFAULT '{}',
    depends_on_kb_services TEXT[] DEFAULT '{}',
    
    -- Performance metadata
    evaluation_time_ms INTEGER NOT NULL DEFAULT 0,
    confidence_score  DECIMAL(3,2) DEFAULT 1.0,
    
    -- Cache management
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ NOT NULL,
    access_count      INTEGER DEFAULT 1,
    last_accessed     TIMESTAMPTZ DEFAULT now()
);

-- Indexes for cache table
CREATE INDEX IF NOT EXISTS idx_cache_expires 
    ON rule_evaluation_cache (expires_at);
CREATE INDEX IF NOT EXISTS idx_cache_drug_version 
    ON rule_evaluation_cache (drug_code, rule_version);
CREATE INDEX IF NOT EXISTS idx_cache_key_hash 
    ON rule_evaluation_cache USING HASH (cache_key);

-- Safety signals for population-level monitoring
CREATE TABLE IF NOT EXISTS safety_signals (
    signal_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code         TEXT NOT NULL,
    drug_class        TEXT,
    signal_type       TEXT NOT NULL CHECK (signal_type IN (
        'adverse_event', 'near_miss', 'override_pattern', 
        'statistical_anomaly', 'trend_deviation', 'control_violation'
    )),
    
    -- Statistical analysis
    baseline_rate     DECIMAL(7,4),
    current_rate      DECIMAL(7,4),
    z_score           DECIMAL(6,2),
    p_value           DECIMAL(8,6),
    confidence_interval JSONB, -- [lower, upper]
    
    -- Detection metadata
    detection_window  INTERVAL NOT NULL,
    sample_size       INTEGER NOT NULL,
    confidence_level  DECIMAL(3,2) NOT NULL DEFAULT 0.95,
    detection_method  TEXT NOT NULL CHECK (detection_method IN ('spc', 'cusum', 'ml_anomaly')),
    
    -- Signal management
    status            TEXT NOT NULL DEFAULT 'detected' CHECK (status IN ('detected','investigating','confirmed','dismissed','resolved')),
    severity          TEXT NOT NULL DEFAULT 'moderate' CHECK (severity IN ('critical','high','moderate','low')),
    detected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    investigated_by   TEXT,
    investigation_notes TEXT,
    resolution        TEXT,
    resolved_at       TIMESTAMPTZ,
    
    -- Metadata
    signal_metadata   JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for safety signals
CREATE INDEX IF NOT EXISTS idx_signals_drug_detected 
    ON safety_signals (drug_code, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_status_active 
    ON safety_signals (status) WHERE status IN ('detected','investigating');
CREATE INDEX IF NOT EXISTS idx_signals_severity_high 
    ON safety_signals (severity) WHERE severity IN ('critical','high');
CREATE INDEX IF NOT EXISTS idx_signals_type_method 
    ON safety_signals (signal_type, detection_method);

-- Override audit trail with complete state machine
CREATE TABLE IF NOT EXISTS override_audit (
    override_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          UUID NOT NULL,
    parent_override_id UUID, -- For chained overrides
    
    -- Override levels and transitions
    override_level    TEXT NOT NULL CHECK (override_level IN ('L1','L2','L3')),
    previous_level    TEXT,
    transition_type   TEXT CHECK (transition_type IN ('initial','escalate','approve','deny','expire')),
    
    -- L1: Acknowledge warning
    acknowledged_by   TEXT,
    acknowledged_at   TIMESTAMPTZ,
    acknowledgment_notes TEXT,
    
    -- L2: Justify override with clinical reasoning
    justification     TEXT,
    justified_by      TEXT,
    justified_at      TIMESTAMPTZ,
    clinical_rationale TEXT,
    risk_acceptance   BOOLEAN DEFAULT FALSE,
    
    -- L3: Break glass with dual authorization
    break_glass_token TEXT,
    primary_authorizer TEXT,
    secondary_authorizer TEXT,
    authorized_by     TEXT[],
    authorized_at     TIMESTAMPTZ,
    emergency_context JSONB DEFAULT '{}',
    
    -- Outcome tracking and follow-up
    patient_outcome   JSONB DEFAULT '{}',
    outcome_recorded_at TIMESTAMPTZ,
    adverse_event     BOOLEAN DEFAULT FALSE,
    follow_up_required BOOLEAN DEFAULT FALSE,
    follow_up_completed BOOLEAN DEFAULT FALSE,
    
    -- Audit and compliance
    approval_chain    JSONB DEFAULT '{}', -- Complete approval history
    witness_signatures TEXT[],
    compliance_notes  TEXT,
    
    -- Timestamps
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ, -- For time-limited overrides
    
    -- References
    FOREIGN KEY (event_id) REFERENCES safety_alerts_v2(event_id) ON DELETE CASCADE
);

-- Indexes for override audit
CREATE INDEX IF NOT EXISTS idx_override_event_id 
    ON override_audit (event_id);
CREATE INDEX IF NOT EXISTS idx_override_level_time 
    ON override_audit (override_level, authorized_at DESC);
CREATE INDEX IF NOT EXISTS idx_override_break_glass 
    ON override_audit (break_glass_token) WHERE break_glass_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_override_adverse_events 
    ON override_audit (adverse_event) WHERE adverse_event = TRUE;
CREATE INDEX IF NOT EXISTS idx_override_follow_up 
    ON override_audit (follow_up_required, follow_up_completed) 
    WHERE follow_up_required = TRUE;

-- Patient allergy profiles for personalized safety checks
CREATE TABLE IF NOT EXISTS patient_allergies (
    allergy_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id        TEXT NOT NULL,
    substance_code    TEXT NOT NULL,
    substance_name    TEXT NOT NULL,
    reaction_terms    TEXT[] DEFAULT '{}',
    severity          TEXT CHECK (severity IN ('mild','moderate','severe','anaphylaxis')),
    verification_status TEXT CHECK (verification_status IN ('confirmed','suspected','family_history')),
    onset_date        DATE,
    
    -- Clinical context
    reaction_description TEXT,
    treating_clinician TEXT,
    source_system     TEXT DEFAULT 'kb4',
    
    -- Audit trail
    recorded_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    recorded_by       TEXT NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Data quality
    confidence_score  DECIMAL(3,2) DEFAULT 1.0,
    data_source       TEXT NOT NULL DEFAULT 'manual_entry',
    
    UNIQUE (patient_id, substance_code)
);

-- Indexes for patient allergies
CREATE INDEX IF NOT EXISTS idx_allergies_patient 
    ON patient_allergies (patient_id);
CREATE INDEX IF NOT EXISTS idx_allergies_substance 
    ON patient_allergies (substance_code);
CREATE INDEX IF NOT EXISTS idx_allergies_severity 
    ON patient_allergies (severity) WHERE severity IN ('severe','anaphylaxis');

-- Multi-factor risk assessment rules
CREATE TABLE IF NOT EXISTS risk_assessment_rules (
    rule_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              TEXT NOT NULL,
    description       TEXT,
    version           TEXT NOT NULL,
    
    -- Rule definition
    dsl               JSONB NOT NULL,
    compiled_function BYTEA,
    input_schema      JSONB NOT NULL DEFAULT '{}',
    output_schema     JSONB NOT NULL DEFAULT '{}',
    
    -- Rule metadata
    rule_type         TEXT NOT NULL CHECK (rule_type IN ('risk_scoring','threshold','pattern_detection')),
    complexity_score  INTEGER DEFAULT 1 CHECK (complexity_score BETWEEN 1 AND 10),
    execution_priority INTEGER DEFAULT 5 CHECK (execution_priority BETWEEN 1 AND 10),
    
    -- Status and lifecycle
    status            TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('active','deprecated','draft','testing')),
    test_cases        JSONB DEFAULT '[]',
    validation_results JSONB DEFAULT '{}',
    
    -- Governance
    created_by        TEXT NOT NULL,
    approved_by       TEXT,
    clinical_reviewer TEXT,
    
    -- Timestamps
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at       TIMESTAMPTZ,
    last_tested       TIMESTAMPTZ,
    next_review       DATE DEFAULT (current_date + interval '6 months')
);

-- Indexes for risk assessment rules
CREATE INDEX IF NOT EXISTS idx_risk_rules_status_priority 
    ON risk_assessment_rules (status, execution_priority) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_risk_rules_type 
    ON risk_assessment_rules (rule_type);
CREATE INDEX IF NOT EXISTS idx_risk_rules_review 
    ON risk_assessment_rules (next_review) WHERE status = 'active';

-- Statistical control charts for SPC monitoring
CREATE TABLE IF NOT EXISTS statistical_control_charts (
    chart_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code         TEXT NOT NULL,
    metric_type       TEXT NOT NULL,
    chart_type        TEXT NOT NULL CHECK (chart_type IN ('xbar','p','c','u','ewma','cusum')),
    
    -- Control limits
    center_line       DECIMAL(10,4) NOT NULL,
    upper_control_limit DECIMAL(10,4) NOT NULL,
    lower_control_limit DECIMAL(10,4) NOT NULL,
    upper_warning_limit DECIMAL(10,4),
    lower_warning_limit DECIMAL(10,4),
    
    -- Chart configuration
    subgroup_size     INTEGER DEFAULT 1,
    sigma_multiplier  DECIMAL(3,1) DEFAULT 3.0,
    
    -- Statistical parameters
    baseline_mean     DECIMAL(10,4),
    baseline_stddev   DECIMAL(10,4),
    baseline_period   INTERVAL DEFAULT '30 days',
    baseline_calculated_at TIMESTAMPTZ,
    
    -- Chart metadata
    status            TEXT DEFAULT 'active' CHECK (status IN ('active','suspended','archived')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_violation    TIMESTAMPTZ,
    violation_count   INTEGER DEFAULT 0,
    
    UNIQUE (drug_code, metric_type, chart_type)
);

-- Indexes for control charts
CREATE INDEX IF NOT EXISTS idx_charts_drug_metric 
    ON statistical_control_charts (drug_code, metric_type);
CREATE INDEX IF NOT EXISTS idx_charts_active 
    ON statistical_control_charts (status) WHERE status = 'active';

-- Chart data points for SPC analysis
CREATE TABLE IF NOT EXISTS chart_data_points (
    point_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chart_id          UUID NOT NULL REFERENCES statistical_control_charts(chart_id) ON DELETE CASCADE,
    ts                TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Data point values
    value             DECIMAL(10,4) NOT NULL,
    subgroup_id       TEXT,
    sample_size       INTEGER DEFAULT 1,
    
    -- Violation detection
    violation_type    TEXT CHECK (violation_type IN ('none','control_limit','warning_limit','trend','run','cycle')),
    violation_rule    TEXT,
    signal_strength   DECIMAL(4,3) DEFAULT 0.0,
    
    -- Context metadata
    context_data      JSONB DEFAULT '{}',
    data_source       TEXT NOT NULL DEFAULT 'safety_assessment',
    quality_score     DECIMAL(3,2) DEFAULT 1.0,
    
    PRIMARY KEY (point_id, ts)
);

-- Create hypertable for chart data points
SELECT create_hypertable(
    'chart_data_points', 
    'ts',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Compression for chart data points
SELECT add_compression_policy(
    'chart_data_points',
    INTERVAL '90 days',
    if_not_exists => TRUE
);

-- Indexes for chart data points
CREATE INDEX IF NOT EXISTS idx_chart_points_chart_ts 
    ON chart_data_points (chart_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_chart_points_violations 
    ON chart_data_points (violation_type) WHERE violation_type != 'none';

-- Evidence envelope store for tamper-evident audit
CREATE TABLE IF NOT EXISTS evidence_envelopes (
    envelope_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          UUID NOT NULL,
    patient_id_hash   TEXT NOT NULL, -- Hashed for privacy
    
    -- Evidence content
    profile_version   TEXT NOT NULL,
    compiled_bundle_id TEXT NOT NULL,
    kb3_snapshot      TEXT,
    evaluator_version TEXT NOT NULL,
    conflict_resolution TEXT,
    
    -- Tamper evidence
    content_hash      TEXT NOT NULL,
    previous_hash     TEXT,
    chain_hash        TEXT NOT NULL,
    
    -- Digital signature
    signature         TEXT,
    signed_by         TEXT,
    signature_algorithm TEXT DEFAULT 'SHA256withRSA',
    
    -- Integrity validation
    integrity_verified BOOLEAN DEFAULT FALSE,
    verification_timestamp TIMESTAMPTZ,
    verification_details JSONB DEFAULT '{}',
    
    -- Metadata
    envelope_version  TEXT NOT NULL DEFAULT '1.0',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    FOREIGN KEY (event_id) REFERENCES safety_alerts_v2(event_id) ON DELETE CASCADE
);

-- Indexes for evidence envelopes
CREATE INDEX IF NOT EXISTS idx_evidence_event_id 
    ON evidence_envelopes (event_id);
CREATE INDEX IF NOT EXISTS idx_evidence_chain_hash 
    ON evidence_envelopes (chain_hash);
CREATE INDEX IF NOT EXISTS idx_evidence_unverified 
    ON evidence_envelopes (integrity_verified) WHERE integrity_verified = FALSE;

-- Clinical decision context for enhanced analysis
CREATE TABLE IF NOT EXISTS clinical_decision_contexts (
    context_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          UUID NOT NULL,
    patient_id        TEXT NOT NULL,
    
    -- Clinical context snapshot
    diagnoses         JSONB DEFAULT '{}',
    lab_values        JSONB DEFAULT '{}',
    vital_signs       JSONB DEFAULT '{}',
    active_medications JSONB DEFAULT '[]',
    allergies         JSONB DEFAULT '[]',
    
    -- Risk factors
    risk_factors      JSONB DEFAULT '{}',
    comorbidities     TEXT[],
    contraindications TEXT[],
    
    -- Data quality metrics
    completeness_score DECIMAL(3,2) DEFAULT 0.0,
    recency_score     DECIMAL(3,2) DEFAULT 0.0,
    reliability_score DECIMAL(3,2) DEFAULT 0.0,
    
    -- Integration metadata
    kb2_context_id    TEXT,
    kb3_guideline_refs TEXT[],
    kb5_interaction_refs TEXT[],
    
    -- Timestamps
    context_timestamp TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    FOREIGN KEY (event_id) REFERENCES safety_alerts_v2(event_id) ON DELETE CASCADE
);

-- Indexes for clinical contexts
CREATE INDEX IF NOT EXISTS idx_contexts_event_id 
    ON clinical_decision_contexts (event_id);
CREATE INDEX IF NOT EXISTS idx_contexts_patient_ts 
    ON clinical_decision_contexts (patient_id, context_timestamp DESC);

-- Notification audit for override workflows
CREATE TABLE IF NOT EXISTS notification_audit (
    notification_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    override_id       UUID REFERENCES override_audit(override_id) ON DELETE CASCADE,
    
    -- Notification details
    notification_type TEXT NOT NULL CHECK (notification_type IN ('email','sms','slack','pager','phone')),
    recipient_id      TEXT NOT NULL,
    recipient_role    TEXT NOT NULL,
    
    -- Message content
    subject           TEXT,
    message_body      TEXT NOT NULL,
    priority          TEXT DEFAULT 'normal' CHECK (priority IN ('low','normal','high','critical')),
    
    -- Delivery tracking
    sent_at           TIMESTAMPTZ DEFAULT now(),
    delivered_at      TIMESTAMPTZ,
    acknowledged_at   TIMESTAMPTZ,
    delivery_status   TEXT DEFAULT 'sent' CHECK (delivery_status IN ('sent','delivered','failed','acknowledged')),
    
    -- Response tracking
    response_required BOOLEAN DEFAULT FALSE,
    response_deadline TIMESTAMPTZ,
    response_received_at TIMESTAMPTZ,
    response_content  TEXT,
    
    -- Metadata
    notification_metadata JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for notification audit
CREATE INDEX IF NOT EXISTS idx_notifications_override 
    ON notification_audit (override_id);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_status 
    ON notification_audit (recipient_id, delivery_status);
CREATE INDEX IF NOT EXISTS idx_notifications_pending_response 
    ON notification_audit (response_required, response_received_at) 
    WHERE response_required = TRUE AND response_received_at IS NULL;

-- Functions for data integrity and validation

-- Function to update timestamps automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_drug_safety_profiles_updated_at 
    BEFORE UPDATE ON drug_safety_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_safety_signals_updated_at 
    BEFORE UPDATE ON safety_signals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_override_audit_updated_at 
    BEFORE UPDATE ON override_audit
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to validate evidence envelope integrity
CREATE OR REPLACE FUNCTION validate_evidence_integrity()
RETURNS TRIGGER AS $$
DECLARE
    computed_hash TEXT;
    content_to_hash TEXT;
BEGIN
    -- Compute hash of critical fields
    content_to_hash := concat(
        NEW.event_id, '|',
        NEW.profile_version, '|', 
        NEW.compiled_bundle_id, '|',
        COALESCE(NEW.kb3_snapshot, ''), '|',
        NEW.evaluator_version
    );
    
    computed_hash := encode(sha256(content_to_hash::bytea), 'hex');
    
    -- Verify content hash matches
    IF NEW.content_hash != computed_hash THEN
        RAISE EXCEPTION 'Evidence envelope integrity violation: hash mismatch';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for evidence integrity validation
CREATE TRIGGER validate_evidence_envelope_integrity
    BEFORE INSERT OR UPDATE ON evidence_envelopes
    FOR EACH ROW EXECUTE FUNCTION validate_evidence_integrity();

-- Comments for documentation
COMMENT ON TABLE drug_safety_profiles IS 'Versioned drug safety profiles with rule bundles and dependencies';
COMMENT ON TABLE rule_evaluation_cache IS 'Intelligent cache with dependency tracking for rule evaluations';
COMMENT ON TABLE safety_signals IS 'Population-level safety signals from statistical analysis';
COMMENT ON TABLE override_audit IS 'Complete audit trail for L1/L2/L3 override state machine';
COMMENT ON TABLE evidence_envelopes IS 'Tamper-evident evidence envelopes for audit compliance';
COMMENT ON TABLE clinical_decision_contexts IS 'Clinical context snapshots for decision analysis';
COMMENT ON TABLE chart_data_points IS 'Time-series data points for SPC chart analysis';
COMMENT ON TABLE notification_audit IS 'Notification delivery and response tracking';

-- Grant permissions for service account
GRANT SELECT, INSERT, UPDATE ON drug_safety_profiles TO kb4_service;
GRANT SELECT, INSERT, UPDATE, DELETE ON rule_evaluation_cache TO kb4_service;
GRANT SELECT, INSERT, UPDATE ON safety_signals TO kb4_service;
GRANT SELECT, INSERT, UPDATE ON override_audit TO kb4_service;
GRANT SELECT, INSERT ON evidence_envelopes TO kb4_service;
GRANT SELECT, INSERT ON clinical_decision_contexts TO kb4_service;
GRANT SELECT, INSERT ON chart_data_points TO kb4_service;
GRANT SELECT, INSERT, UPDATE ON notification_audit TO kb4_service;

-- Grant read-only access for analytics
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb4_analytics;

-- Create dedicated role for rule authoring
CREATE ROLE IF NOT EXISTS kb4_rule_author;
GRANT SELECT, INSERT, UPDATE ON drug_safety_profiles TO kb4_rule_author;
GRANT SELECT, INSERT, UPDATE ON risk_assessment_rules TO kb4_rule_author;