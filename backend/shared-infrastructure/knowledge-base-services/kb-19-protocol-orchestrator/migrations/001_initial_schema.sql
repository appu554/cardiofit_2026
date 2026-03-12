-- KB-19 Protocol Orchestrator - Initial Schema
-- This schema stores decision audit trails for regulatory compliance.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Decision bundles (main audit record)
CREATE TABLE IF NOT EXISTS recommendation_bundles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id UUID NOT NULL,
    encounter_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED',
    narrative_summary TEXT,
    processing_time_ms INTEGER,
    protocols_evaluated INTEGER DEFAULT 0,
    protocols_applicable INTEGER DEFAULT 0,
    conflicts_detected INTEGER DEFAULT 0,
    safety_blocks INTEGER DEFAULT 0,
    highest_urgency VARCHAR(20),
    service_versions JSONB,
    executive_summary JSONB,
    risk_assessment JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for patient lookups
CREATE INDEX idx_bundles_patient_id ON recommendation_bundles(patient_id);
CREATE INDEX idx_bundles_encounter_id ON recommendation_bundles(encounter_id);
CREATE INDEX idx_bundles_timestamp ON recommendation_bundles(timestamp DESC);

-- Arbitrated decisions
CREATE TABLE IF NOT EXISTS arbitrated_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bundle_id UUID NOT NULL REFERENCES recommendation_bundles(id) ON DELETE CASCADE,
    decision_type VARCHAR(20) NOT NULL,
    target VARCHAR(255) NOT NULL,
    target_rxnorm VARCHAR(50),
    target_snomed VARCHAR(50),
    rationale TEXT,
    urgency VARCHAR(20) NOT NULL,
    source_protocol VARCHAR(100),
    source_protocol_id VARCHAR(100),
    arbitration_reason TEXT,
    conflicted_with VARCHAR(100),
    conflict_type VARCHAR(50),
    recommendation_class VARCHAR(10),
    evidence_level VARCHAR(10),
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_decisions_bundle_id ON arbitrated_decisions(bundle_id);
CREATE INDEX idx_decisions_patient ON arbitrated_decisions(bundle_id, decision_type);

-- Safety flags
CREATE TABLE IF NOT EXISTS safety_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id UUID NOT NULL REFERENCES arbitrated_decisions(id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    reason TEXT,
    source VARCHAR(100),
    overridden BOOLEAN DEFAULT FALSE,
    override_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_safety_flags_decision ON safety_flags(decision_id);

-- Evidence envelopes
CREATE TABLE IF NOT EXISTS evidence_envelopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id UUID NOT NULL REFERENCES arbitrated_decisions(id) ON DELETE CASCADE,
    recommendation_class VARCHAR(10),
    evidence_level VARCHAR(10),
    guideline_source VARCHAR(100),
    guideline_version VARCHAR(50),
    guideline_year INTEGER,
    citation_anchor TEXT,
    citation_text TEXT,
    inference_chain JSONB,
    kb_versions JSONB,
    cql_engine_version VARCHAR(50),
    patient_context_id UUID,
    checksum VARCHAR(64),
    finalized BOOLEAN DEFAULT FALSE,
    digital_signature TEXT,
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_evidence_decision ON evidence_envelopes(decision_id);

-- Protocol evaluations
CREATE TABLE IF NOT EXISTS protocol_evaluations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bundle_id UUID NOT NULL REFERENCES recommendation_bundles(id) ON DELETE CASCADE,
    protocol_id VARCHAR(100) NOT NULL,
    protocol_name VARCHAR(255),
    is_applicable BOOLEAN DEFAULT FALSE,
    applicability_reason TEXT,
    contraindicated BOOLEAN DEFAULT FALSE,
    contraindication_reasons TEXT[],
    priority_class INTEGER,
    risk_score_impact FLOAT,
    cql_facts_used TEXT[],
    calculators_used JSONB,
    confidence FLOAT DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_evaluations_bundle ON protocol_evaluations(bundle_id);

-- Conflict resolutions
CREATE TABLE IF NOT EXISTS conflict_resolutions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bundle_id UUID NOT NULL REFERENCES recommendation_bundles(id) ON DELETE CASCADE,
    protocol_a VARCHAR(100) NOT NULL,
    protocol_b VARCHAR(100) NOT NULL,
    conflict_type VARCHAR(50) NOT NULL,
    winner VARCHAR(100),
    loser VARCHAR(100),
    resolution_rule TEXT,
    explanation TEXT,
    loser_outcome VARCHAR(20),
    confidence FLOAT DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conflicts_bundle ON conflict_resolutions(bundle_id);

-- Safety gates applied
CREATE TABLE IF NOT EXISTS safety_gates_applied (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bundle_id UUID NOT NULL REFERENCES recommendation_bundles(id) ON DELETE CASCADE,
    gate_name VARCHAR(100) NOT NULL,
    source VARCHAR(100),
    triggered BOOLEAN DEFAULT FALSE,
    result VARCHAR(20),
    details TEXT,
    affected_decisions UUID[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_safety_gates_bundle ON safety_gates_applied(bundle_id);

-- Alerts generated
CREATE TABLE IF NOT EXISTS clinical_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bundle_id UUID NOT NULL REFERENCES recommendation_bundles(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    decision_ref UUID,
    requires_ack BOOLEAN DEFAULT FALSE,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_bundle ON clinical_alerts(bundle_id);
CREATE INDEX idx_alerts_unacked ON clinical_alerts(bundle_id) WHERE acknowledged = FALSE;

-- Protocol definitions (cached from YAML)
CREATE TABLE IF NOT EXISTS protocol_definitions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    priority_class INTEGER,
    trigger_criteria TEXT[],
    contraindication_rules TEXT[],
    required_calculators TEXT[],
    guideline_source VARCHAR(100),
    guideline_version VARCHAR(50),
    citation_reference TEXT,
    applicable_settings TEXT[],
    target_population TEXT[],
    is_active BOOLEAN DEFAULT TRUE,
    version VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Conflict matrix entries (cached from YAML)
CREATE TABLE IF NOT EXISTS conflict_matrix (
    id VARCHAR(100) PRIMARY KEY,
    protocol_a VARCHAR(100) NOT NULL,
    protocol_b VARCHAR(100) NOT NULL,
    conflict_type VARCHAR(50) NOT NULL,
    description TEXT,
    resolution_rule JSONB,
    severity VARCHAR(20),
    clinical_rationale TEXT,
    citation TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conflict_matrix_protocols ON conflict_matrix(protocol_a, protocol_b);

-- Audit log for all operations
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    patient_id UUID,
    user_id VARCHAR(100),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_patient ON audit_log(patient_id, timestamp DESC);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_bundles_updated_at
    BEFORE UPDATE ON recommendation_bundles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_protocols_updated_at
    BEFORE UPDATE ON protocol_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conflicts_updated_at
    BEFORE UPDATE ON conflict_matrix
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE recommendation_bundles IS 'Main audit table for KB-19 recommendation bundles';
COMMENT ON TABLE arbitrated_decisions IS 'Individual clinical decisions from arbitration engine';
COMMENT ON TABLE evidence_envelopes IS 'FDA 21 CFR Part 11 compliant evidence trail';
COMMENT ON TABLE conflict_resolutions IS 'Protocol conflicts identified and resolved';
COMMENT ON TABLE safety_gates_applied IS 'Safety checks applied during arbitration';
