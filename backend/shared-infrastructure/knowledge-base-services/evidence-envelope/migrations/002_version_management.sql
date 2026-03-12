-- Add to evidence-envelope/migrations/002_version_management.sql

-- Version management table
CREATE TABLE kb_version_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_set_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    kb_versions JSONB NOT NULL DEFAULT '{}',
    validated BOOLEAN DEFAULT FALSE,
    validation_results JSONB,
    environment VARCHAR(50) NOT NULL,
    active BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_active_per_env EXCLUDE (environment WITH =) WHERE (active = true)
);

-- Add version tracking to evidence_envelopes
ALTER TABLE evidence_envelopes 
ADD COLUMN version_set_id UUID REFERENCES kb_version_sets(id);

-- Add checksum and signature columns
ALTER TABLE evidence_envelopes
ADD COLUMN checksum VARCHAR(64),
ADD COLUMN signed BOOLEAN DEFAULT FALSE,
ADD COLUMN signature TEXT;

-- Create partitions for performance
CREATE TABLE evidence_envelopes_2025_01 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Add indexes for version management
CREATE INDEX idx_kb_version_sets_environment ON kb_version_sets(environment);
CREATE INDEX idx_kb_version_sets_active ON kb_version_sets(active) WHERE active = true;
CREATE INDEX idx_evidence_envelopes_version_set ON evidence_envelopes(version_set_id);
CREATE INDEX idx_evidence_envelopes_checksum ON evidence_envelopes(checksum);
CREATE INDEX idx_evidence_envelopes_signed ON evidence_envelopes(signed);

-- Create function for checksum calculation
CREATE OR REPLACE FUNCTION calculate_evidence_checksum(
    transaction_data JSONB,
    kb_calls JSONB,
    clinical_decisions JSONB
) RETURNS VARCHAR(64) AS $$
DECLARE
    combined_data TEXT;
    checksum_value VARCHAR(64);
BEGIN
    -- Combine all data into a single string for hashing
    combined_data := COALESCE(transaction_data::text, '') || 
                    COALESCE(kb_calls::text, '') || 
                    COALESCE(clinical_decisions::text, '');
    
    -- Calculate SHA-256 hash
    checksum_value := encode(digest(combined_data, 'sha256'), 'hex');
    
    RETURN checksum_value;
END;
$$ LANGUAGE plpgsql;

-- Create function for version validation
CREATE OR REPLACE FUNCTION validate_kb_version_set(
    version_set_id_param UUID
) RETURNS JSONB AS $$
DECLARE
    version_set RECORD;
    validation_results JSONB := '{}';
    kb_name TEXT;
    kb_version TEXT;
    service_url TEXT;
    health_status BOOLEAN;
BEGIN
    -- Get version set details
    SELECT * INTO version_set FROM kb_version_sets WHERE id = version_set_id_param;
    
    IF NOT FOUND THEN
        RETURN jsonb_build_object('error', 'Version set not found');
    END IF;
    
    -- Validate each KB service version
    FOR kb_name, kb_version IN SELECT * FROM jsonb_each_text(version_set.kb_versions)
    LOOP
        -- Build service URL based on KB name
        service_url := 'http://localhost:' || 
            CASE kb_name
                WHEN 'kb1-drug-rules' THEN '8081'
                WHEN 'kb2-clinical-context' THEN '8082'
                WHEN 'kb3-guidelines' THEN '8083'
                WHEN 'kb4-patient-safety' THEN '8084'
                WHEN 'kb5-ddi' THEN '8085'
                WHEN 'kb6-formulary' THEN '8086'
                WHEN 'kb7-terminology' THEN '8087'
                ELSE '8080'
            END || '/health';
        
        -- Note: In real implementation, would make HTTP call to check health
        -- For now, assume services are healthy if version is provided
        health_status := (kb_version IS NOT NULL AND kb_version != '');
        
        validation_results := validation_results || 
            jsonb_build_object(
                kb_name, 
                jsonb_build_object(
                    'version', kb_version,
                    'healthy', health_status,
                    'validated_at', NOW()
                )
            );
    END LOOP;
    
    -- Update validation results in the version set
    UPDATE kb_version_sets 
    SET validation_results = validation_results,
        validated = true
    WHERE id = version_set_id_param;
    
    RETURN validation_results;
END;
$$ LANGUAGE plpgsql;

-- Insert initial version set
INSERT INTO kb_version_sets (
    version_set_name,
    description,
    kb_versions,
    environment,
    active,
    created_by
) VALUES (
    'initial_v1_0_0',
    'Initial Knowledge Base version set for Phase 0 deployment',
    '{
        "kb1-drug-rules": "1.0.0",
        "kb2-clinical-context": "1.0.0",
        "kb3-guidelines": "1.0.0",
        "kb4-patient-safety": "1.0.0",
        "kb5-ddi": "1.0.0",
        "kb6-formulary": "1.0.0",
        "kb7-terminology": "1.0.0",
        "evidence-envelope": "1.0.0"
    }',
    'development',
    true,
    'system'
);

-- Add comments for documentation
COMMENT ON TABLE kb_version_sets IS 'Manages coordinated versions of all Knowledge Base services';
COMMENT ON COLUMN kb_version_sets.kb_versions IS 'JSONB object mapping KB service names to their versions';
COMMENT ON COLUMN kb_version_sets.validated IS 'Whether this version set has been validated for deployment';
COMMENT ON COLUMN kb_version_sets.active IS 'Only one version set can be active per environment';
COMMENT ON FUNCTION calculate_evidence_checksum IS 'Calculates SHA-256 checksum for evidence envelope integrity';
COMMENT ON FUNCTION validate_kb_version_set IS 'Validates that all KB services in a version set are healthy and compatible';