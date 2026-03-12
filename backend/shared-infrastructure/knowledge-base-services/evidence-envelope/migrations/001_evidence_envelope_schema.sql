-- Evidence Envelope Schema - Audit trail and governance tracking
-- This schema provides comprehensive audit trails for all clinical decisions and data transformations

-- Transaction audit log - tracks all system interactions
CREATE TABLE evidence_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(50),
    session_id VARCHAR(100),
    source_service VARCHAR(50) NOT NULL,
    target_service VARCHAR(50),
    operation_type VARCHAR(30) NOT NULL, -- query, mutation, transformation, decision
    graphql_operation TEXT,
    request_payload JSONB,
    response_payload JSONB,
    http_status INTEGER,
    processing_time_ms INTEGER,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    correlation_id VARCHAR(100),
    trace_id VARCHAR(100),
    span_id VARCHAR(100)
);

-- Data lineage tracking - tracks data transformations and origins
CREATE TABLE data_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(64) REFERENCES evidence_transactions(transaction_id),
    source_system VARCHAR(50) NOT NULL,
    source_entity VARCHAR(100) NOT NULL,
    source_id VARCHAR(100) NOT NULL,
    target_system VARCHAR(50) NOT NULL,
    target_entity VARCHAR(100) NOT NULL,
    target_id VARCHAR(100),
    transformation_type VARCHAR(50), -- mapping, enrichment, validation, normalization
    transformation_rules JSONB,
    data_quality_score DECIMAL(3,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Clinical decision audit - tracks all clinical reasoning decisions
CREATE TABLE clinical_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(64) REFERENCES evidence_transactions(transaction_id),
    decision_id VARCHAR(100) UNIQUE NOT NULL,
    patient_id VARCHAR(50),
    decision_type VARCHAR(50) NOT NULL, -- drug_interaction, phenotype_detection, formulary_check
    knowledge_source VARCHAR(50) NOT NULL, -- which KB service made the decision
    input_data JSONB NOT NULL,
    decision_outcome JSONB NOT NULL,
    confidence_score DECIMAL(3,2),
    evidence_sources JSONB, -- references to supporting evidence
    overridden_by VARCHAR(50),
    override_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Knowledge base version tracking
CREATE TABLE kb_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kb_service VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    schema_version VARCHAR(10) NOT NULL,
    data_sources JSONB NOT NULL, -- external sources with versions
    deployment_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    validation_status VARCHAR(20) DEFAULT 'pending',
    validation_results JSONB,
    is_active BOOLEAN DEFAULT FALSE,
    deactivated_at TIMESTAMP WITH TIME ZONE
);

-- Data provenance - tracks source of clinical data
CREATE TABLE data_provenance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    source_system VARCHAR(50) NOT NULL,
    source_timestamp TIMESTAMP WITH TIME ZONE,
    ingestion_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    data_quality_flags JSONB,
    validation_status VARCHAR(20) DEFAULT 'valid',
    retention_policy VARCHAR(50),
    gdpr_consent_status VARCHAR(20)
);

-- Performance and error tracking
CREATE TABLE system_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    metric_value DECIMAL(10,4) NOT NULL,
    metric_unit VARCHAR(20),
    tags JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_evidence_transactions_timestamp ON evidence_transactions(timestamp);
CREATE INDEX idx_evidence_transactions_user ON evidence_transactions(user_id);
CREATE INDEX idx_evidence_transactions_service ON evidence_transactions(source_service);
CREATE INDEX idx_evidence_transactions_correlation ON evidence_transactions(correlation_id);

CREATE INDEX idx_data_lineage_transaction ON data_lineage(transaction_id);
CREATE INDEX idx_data_lineage_source ON data_lineage(source_system, source_entity, source_id);

CREATE INDEX idx_clinical_decisions_patient ON clinical_decisions(patient_id);
CREATE INDEX idx_clinical_decisions_type ON clinical_decisions(decision_type);
CREATE INDEX idx_clinical_decisions_timestamp ON clinical_decisions(created_at);

CREATE INDEX idx_kb_versions_service ON kb_versions(kb_service, version);
CREATE INDEX idx_kb_versions_active ON kb_versions(is_active) WHERE is_active = TRUE;

CREATE INDEX idx_data_provenance_entity ON data_provenance(entity_type, entity_id);
CREATE INDEX idx_system_metrics_service_time ON system_metrics(service_name, timestamp);