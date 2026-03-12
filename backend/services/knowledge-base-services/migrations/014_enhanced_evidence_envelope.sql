-- Enhanced Evidence Envelope Schema
-- Part I: Central Versioned API Layer - Enhanced Evidence Envelope with new tracking

-- Enhanced Evidence Envelopes table with comprehensive tracking
CREATE TABLE IF NOT EXISTS evidence_envelopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id VARCHAR(100) UNIQUE NOT NULL,
    
    -- Version snapshot at time of transaction
    version_set_id UUID REFERENCES kb_version_sets(id),
    kb_versions JSONB NOT NULL,
    
    -- Enhanced decision tracking
    decision_chain JSONB NOT NULL DEFAULT '[]',
    /* Enhanced structure:
    [
      {
        "phase": "ORB",
        "timestamp": "2024-01-15T10:30:00Z",
        "input": {...},
        "output": {...},
        "kb_calls": ["kb_3_guidelines", "kb_7_terminology"],
        "duration_ms": 45,
        "decisions": [...],
        "confidence": 0.95,
        "evidence_quality": "high"
      }
    ]
    */
    
    safety_attestations JSONB NOT NULL DEFAULT '[]',
    /* Enhanced structure:
    [
      {
        "type": "drug_interaction_check",
        "result": "no_major_interactions",
        "confidence": 0.98,
        "evidence": [...],
        "reviewer": "kb_4_safety",
        "timestamp": "2024-01-15T10:30:00Z",
        "attestation_id": "att_123456"
      }
    ]
    */
    
    -- Enhanced performance metrics
    performance_metrics JSONB DEFAULT '{}',
    /* Structure:
    {
      "total_requests": 7,
      "total_latency_ms": 250,
      "avg_latency_ms": 35.7,
      "max_latency_ms": 85,
      "cache_hit_rate": 0.86,
      "error_rate": 0.0,
      "kb_performance": {
        "kb_1_dosing": {"latency_ms": 12, "cache_hit": true},
        "kb_3_guidelines": {"latency_ms": 85, "cache_hit": false}
      }
    }
    */
    
    -- Clinical context
    patient_id VARCHAR(100),
    encounter_id VARCHAR(100),
    clinical_domain VARCHAR(50),
    request_type VARCHAR(50),
    
    -- Enhanced orchestration metadata
    orchestrator_version VARCHAR(50),
    orchestrator_node VARCHAR(100),
    workflow_version VARCHAR(50),
    
    -- Request metadata
    user_id VARCHAR(100),
    user_role VARCHAR(50),
    session_id VARCHAR(100),
    client_ip INET,
    user_agent TEXT,
    
    -- Timing with microsecond precision
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    total_duration_ms INTEGER,
    
    -- Enhanced immutability and signatures
    content_hash VARCHAR(64),
    checksum VARCHAR(64) NOT NULL,
    signed BOOLEAN DEFAULT FALSE,
    signature TEXT,
    signature_algorithm VARCHAR(50) DEFAULT 'Ed25519',
    signed_by VARCHAR(100),
    signed_at TIMESTAMPTZ,
    
    -- Quality metrics
    completeness_score DECIMAL(3,2),
    consistency_score DECIMAL(3,2),
    
    -- Compliance and governance
    regulatory_flags JSONB DEFAULT '{}',
    privacy_classification VARCHAR(50),
    retention_policy VARCHAR(100),
    
    -- Partitioning key
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for evidence envelopes (current year)
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_01 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_02 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_03 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_04 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_05 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_06 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_07 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_08 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_09 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_10 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_11 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE IF NOT EXISTS evidence_envelopes_2025_12 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- KB response log with enhanced tracking
CREATE TABLE IF NOT EXISTS kb_response_log (
    id BIGSERIAL PRIMARY KEY,
    envelope_id UUID REFERENCES evidence_envelopes(id) ON DELETE CASCADE,
    
    -- KB service details
    kb_name VARCHAR(50) NOT NULL,
    kb_version VARCHAR(50) NOT NULL,
    kb_endpoint VARCHAR(200),
    
    -- Request details
    request_method VARCHAR(10),
    request_payload_size INTEGER,
    request_hash VARCHAR(64),
    
    -- Response details
    response_status_code INTEGER,
    response_size INTEGER,
    response_hash VARCHAR(64),
    
    -- Performance metrics
    latency_ms INTEGER NOT NULL,
    queue_time_ms INTEGER,
    processing_time_ms INTEGER,
    
    -- Caching information
    cache_hit BOOLEAN DEFAULT FALSE,
    cache_key VARCHAR(200),
    cache_ttl INTEGER,
    
    -- Error tracking
    error_count INTEGER DEFAULT 0,
    error_details JSONB,
    retry_count INTEGER DEFAULT 0,
    
    -- Quality metrics
    confidence_score DECIMAL(3,2),
    data_quality_score DECIMAL(3,2),
    
    -- Timestamps
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    request_started_at TIMESTAMPTZ,
    request_completed_at TIMESTAMPTZ
) PARTITION BY RANGE (timestamp);

-- Create monthly partitions for KB response log
CREATE TABLE IF NOT EXISTS kb_response_log_2025_01 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_02 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_03 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_04 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_05 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_06 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_07 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_08 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_09 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_10 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_11 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE IF NOT EXISTS kb_response_log_2025_12 PARTITION OF kb_response_log 
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Enhanced audit trail table
CREATE TABLE IF NOT EXISTS kb_audit_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Transaction tracking
    transaction_id VARCHAR(100),
    evidence_envelope_id UUID,
    
    -- Request information
    operation_type VARCHAR(50) NOT NULL,
    operation_name VARCHAR(100),
    user_id VARCHAR(100),
    user_role VARCHAR(50),
    session_id VARCHAR(100),
    
    -- Network information
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    
    -- Request details
    query TEXT,
    variables JSONB,
    kb_services_used JSONB DEFAULT '[]',
    
    -- Response information
    response_time_ms INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Version information
    version_set_id UUID,
    kb_versions JSONB,
    
    -- Clinical context
    clinical_domain VARCHAR(50),
    patient_id VARCHAR(100),
    
    -- Compliance and governance
    data_classification VARCHAR(50),
    retention_category VARCHAR(50),
    
    -- Additional metadata
    metadata JSONB DEFAULT '{}'
) PARTITION BY RANGE (timestamp);

-- Create monthly partitions for audit log
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_01 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_02 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_03 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_04 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_05 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_06 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_07 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_08 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_09 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_10 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_11 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE IF NOT EXISTS kb_audit_log_2025_12 PARTITION OF kb_audit_log 
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_transaction_id ON evidence_envelopes(transaction_id);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_patient_id ON evidence_envelopes(patient_id);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_created_at ON evidence_envelopes(created_at);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_user_id ON evidence_envelopes(user_id);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_clinical_domain ON evidence_envelopes(clinical_domain);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_version_set ON evidence_envelopes(version_set_id);

CREATE INDEX IF NOT EXISTS idx_kb_response_log_envelope ON kb_response_log(envelope_id);
CREATE INDEX IF NOT EXISTS idx_kb_response_log_kb_name ON kb_response_log(kb_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_kb_response_log_timestamp ON kb_response_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_kb_response_log_latency ON kb_response_log(latency_ms);
CREATE INDEX IF NOT EXISTS idx_kb_response_log_cache_hit ON kb_response_log(cache_hit);

CREATE INDEX IF NOT EXISTS idx_kb_audit_log_transaction ON kb_audit_log(transaction_id);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_timestamp ON kb_audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_user ON kb_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_operation ON kb_audit_log(operation_type, operation_name);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_success ON kb_audit_log(success, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_patient ON kb_audit_log(patient_id);

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_decision_chain_gin ON evidence_envelopes USING GIN(decision_chain);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_safety_attestations_gin ON evidence_envelopes USING GIN(safety_attestations);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_performance_metrics_gin ON evidence_envelopes USING GIN(performance_metrics);
CREATE INDEX IF NOT EXISTS idx_kb_response_log_error_details_gin ON kb_response_log USING GIN(error_details);
CREATE INDEX IF NOT EXISTS idx_kb_audit_log_metadata_gin ON kb_audit_log USING GIN(metadata);

-- Functions for enhanced evidence envelope management

-- Function to calculate evidence envelope completeness score
CREATE OR REPLACE FUNCTION calculate_completeness_score(envelope_data JSONB)
RETURNS DECIMAL(3,2) AS $$
DECLARE
    score DECIMAL(3,2) := 0.0;
    total_fields INTEGER := 10;
    completed_fields INTEGER := 0;
BEGIN
    -- Check required fields
    IF envelope_data ? 'decision_chain' AND jsonb_array_length(envelope_data->'decision_chain') > 0 THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'safety_attestations' AND jsonb_array_length(envelope_data->'safety_attestations') > 0 THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'performance_metrics' AND envelope_data->'performance_metrics' != '{}' THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'patient_id' AND envelope_data->>'patient_id' IS NOT NULL THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'clinical_domain' AND envelope_data->>'clinical_domain' IS NOT NULL THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'user_id' AND envelope_data->>'user_id' IS NOT NULL THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'checksum' AND envelope_data->>'checksum' IS NOT NULL THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'kb_versions' AND envelope_data->'kb_versions' != '{}' THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'total_duration_ms' AND (envelope_data->>'total_duration_ms')::INTEGER > 0 THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    IF envelope_data ? 'version_set_id' AND envelope_data->>'version_set_id' IS NOT NULL THEN
        completed_fields := completed_fields + 1;
    END IF;
    
    score := completed_fields::DECIMAL / total_fields;
    RETURN LEAST(score, 1.0);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to validate evidence envelope integrity
CREATE OR REPLACE FUNCTION validate_evidence_envelope(envelope_id UUID)
RETURNS TABLE (
    valid BOOLEAN,
    validation_errors TEXT[],
    validation_warnings TEXT[]
) AS $$
DECLARE
    envelope_record RECORD;
    errors TEXT[] := '{}';
    warnings TEXT[] := '{}';
    is_valid BOOLEAN := TRUE;
BEGIN
    -- Get envelope record
    SELECT * INTO envelope_record
    FROM evidence_envelopes
    WHERE id = envelope_id;
    
    IF NOT FOUND THEN
        RETURN QUERY SELECT FALSE, ARRAY['Evidence envelope not found'], ARRAY[]::TEXT[];
        RETURN;
    END IF;
    
    -- Validate required fields
    IF envelope_record.transaction_id IS NULL OR envelope_record.transaction_id = '' THEN
        errors := errors || 'Missing transaction_id';
        is_valid := FALSE;
    END IF;
    
    IF envelope_record.kb_versions IS NULL OR envelope_record.kb_versions = '{}' THEN
        errors := errors || 'Missing kb_versions';
        is_valid := FALSE;
    END IF;
    
    IF envelope_record.checksum IS NULL OR envelope_record.checksum = '' THEN
        errors := errors || 'Missing checksum';
        is_valid := FALSE;
    END IF;
    
    -- Validate decision chain
    IF envelope_record.decision_chain IS NULL OR jsonb_array_length(envelope_record.decision_chain) = 0 THEN
        warnings := warnings || 'Empty decision chain';
    END IF;
    
    -- Validate performance metrics
    IF envelope_record.total_duration_ms IS NULL THEN
        warnings := warnings || 'Missing total duration';
    ELSIF envelope_record.total_duration_ms > 10000 THEN
        warnings := warnings || 'High total duration: ' || envelope_record.total_duration_ms::TEXT || 'ms';
    END IF;
    
    -- Validate completeness
    IF envelope_record.completeness_score < 0.8 THEN
        warnings := warnings || 'Low completeness score: ' || envelope_record.completeness_score::TEXT;
    END IF;
    
    RETURN QUERY SELECT is_valid, errors, warnings;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get evidence envelope statistics
CREATE OR REPLACE FUNCTION get_evidence_envelope_stats(
    start_date TIMESTAMPTZ DEFAULT NOW() - INTERVAL '24 hours',
    end_date TIMESTAMPTZ DEFAULT NOW()
)
RETURNS TABLE (
    total_envelopes BIGINT,
    successful_envelopes BIGINT,
    avg_duration_ms DECIMAL,
    p95_duration_ms DECIMAL,
    avg_completeness_score DECIMAL,
    signed_envelopes BIGINT,
    unique_patients BIGINT,
    unique_users BIGINT,
    top_clinical_domains JSONB
) AS $$
BEGIN
    RETURN QUERY
    WITH envelope_stats AS (
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE completed_at IS NOT NULL) as successful,
            AVG(total_duration_ms) as avg_duration,
            PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY total_duration_ms) as p95_duration,
            AVG(completeness_score) as avg_completeness,
            COUNT(*) FILTER (WHERE signed = TRUE) as signed_count,
            COUNT(DISTINCT patient_id) as unique_patients_count,
            COUNT(DISTINCT user_id) as unique_users_count,
            jsonb_object_agg(
                clinical_domain,
                domain_count
            ) as domains
        FROM evidence_envelopes,
        LATERAL (
            SELECT clinical_domain, COUNT(*) as domain_count
            FROM evidence_envelopes e2
            WHERE e2.clinical_domain = evidence_envelopes.clinical_domain
              AND e2.created_at BETWEEN start_date AND end_date
            GROUP BY clinical_domain
            ORDER BY domain_count DESC
            LIMIT 5
        ) domain_stats
        WHERE created_at BETWEEN start_date AND end_date
    )
    SELECT 
        total,
        successful,
        avg_duration,
        p95_duration,
        avg_completeness,
        signed_count,
        unique_patients_count,
        unique_users_count,
        domains
    FROM envelope_stats;
END;
$$ LANGUAGE plpgsql STABLE;

-- Comments for documentation
COMMENT ON TABLE evidence_envelopes IS 'Enhanced evidence envelopes with comprehensive clinical decision tracking and governance';
COMMENT ON TABLE kb_response_log IS 'Detailed log of all KB service responses with performance and quality metrics';
COMMENT ON TABLE kb_audit_log IS 'Comprehensive audit trail for all GraphQL operations with compliance tracking';

COMMENT ON COLUMN evidence_envelopes.decision_chain IS 'Enhanced JSONB array of decision points with confidence and evidence quality scores';
COMMENT ON COLUMN evidence_envelopes.safety_attestations IS 'JSONB array of safety checks and attestations with reviewer information';
COMMENT ON COLUMN evidence_envelopes.performance_metrics IS 'JSONB object with detailed performance metrics across all KB services';
COMMENT ON COLUMN evidence_envelopes.completeness_score IS 'Calculated score (0.0-1.0) indicating data completeness';
COMMENT ON COLUMN evidence_envelopes.consistency_score IS 'Calculated score (0.0-1.0) indicating data consistency across KB responses';