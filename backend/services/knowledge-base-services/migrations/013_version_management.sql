-- Enhanced KB Version Management Schema
-- Part I: Central Versioned API Layer - Version Management Tables

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- KB Version Sets table - Central version coordination
CREATE TABLE IF NOT EXISTS kb_version_sets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version_set_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    
    -- Version mapping for all KBs
    kb_versions JSONB NOT NULL DEFAULT '{}',
    /* Example structure:
    {
      "kb_1_dosing": "3.2.0+sha.9c1d8ab",
      "kb_2_context": "2.4.1+sha.77fe1",
      "kb_3_guidelines": "1.9.0+sha.af12d",
      "kb_4_safety": "3.0.0+sha.0e1b2",
      "kb_5_ddi": "2.6.3+sha.2a77e",
      "kb_6_formulary": "1.5.0+sha.55ef1",
      "kb_7_terminology": "2.2.0+sha.d1aa7"
    }
    */
    
    -- Validation status
    validated BOOLEAN DEFAULT FALSE,
    validation_results JSONB,
    validation_timestamp TIMESTAMPTZ,
    
    -- Deployment tracking
    environment VARCHAR(50) NOT NULL CHECK (environment IN ('dev', 'staging', 'production')),
    active BOOLEAN DEFAULT FALSE,
    activated_at TIMESTAMPTZ,
    deactivated_at TIMESTAMPTZ,
    
    -- Governance
    created_by VARCHAR(100) NOT NULL,
    approved_by VARCHAR(100),
    approval_timestamp TIMESTAMPTZ,
    approval_notes TEXT,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Ensure only one active version set per environment
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_version_sets_active_env 
ON kb_version_sets(environment) WHERE active = true;

-- Version deployment history
CREATE TABLE IF NOT EXISTS kb_version_deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version_set_id UUID REFERENCES kb_version_sets(id) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    
    -- Deployment details
    deployed_by VARCHAR(100) NOT NULL,
    deployed_at TIMESTAMPTZ DEFAULT NOW(),
    rollback_version_set_id UUID REFERENCES kb_version_sets(id),
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'deploying' CHECK (
        status IN ('deploying', 'deployed', 'failed', 'rolled_back')
    ),
    
    -- Deployment process log
    deployment_log JSONB NOT NULL DEFAULT '[]',
    /* Structure:
    [
      {
        "timestamp": "2024-01-15T10:30:00Z",
        "level": "info",
        "message": "Deployment started",
        "details": {...}
      }
    ]
    */
    
    -- Validation results post-deployment
    validation_results JSONB DEFAULT '[]',
    
    -- Performance metrics
    pre_deployment_health JSONB,
    post_deployment_health JSONB,
    rollback_reason TEXT
);

-- KB service version registry
CREATE TABLE IF NOT EXISTS kb_service_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kb_name VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    
    -- Version metadata
    git_sha VARCHAR(40),
    git_branch VARCHAR(100),
    git_tag VARCHAR(100),
    build_number INTEGER,
    build_timestamp TIMESTAMPTZ,
    
    -- Service information
    service_url VARCHAR(255),
    health_check_url VARCHAR(255),
    api_schema_url VARCHAR(255),
    
    -- Compatibility information
    min_compatible_version VARCHAR(50),
    max_compatible_version VARCHAR(50),
    breaking_changes JSONB DEFAULT '[]',
    
    -- Deployment status
    status VARCHAR(20) DEFAULT 'built' CHECK (
        status IN ('built', 'tested', 'deployed', 'deprecated', 'retired')
    ),
    
    -- Quality gates
    tests_passed BOOLEAN DEFAULT FALSE,
    security_scan_passed BOOLEAN DEFAULT FALSE,
    performance_benchmarks JSONB,
    
    -- Lifecycle dates
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deployed_at TIMESTAMPTZ,
    deprecated_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ,
    
    -- Governance
    created_by VARCHAR(100) NOT NULL,
    approved_by VARCHAR(100),
    approval_timestamp TIMESTAMPTZ,
    
    UNIQUE(kb_name, version)
);

-- Version compatibility matrix
CREATE TABLE IF NOT EXISTS kb_version_compatibility (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source KB version
    source_kb VARCHAR(50) NOT NULL,
    source_version VARCHAR(50) NOT NULL,
    
    -- Target KB version
    target_kb VARCHAR(50) NOT NULL,
    target_version VARCHAR(50) NOT NULL,
    
    -- Compatibility assessment
    compatibility_status VARCHAR(20) NOT NULL CHECK (
        compatibility_status IN ('compatible', 'warning', 'incompatible', 'unknown')
    ),
    
    -- Details
    compatibility_notes TEXT,
    test_results JSONB,
    performance_impact JSONB,
    
    -- Validation
    validated BOOLEAN DEFAULT FALSE,
    validated_by VARCHAR(100),
    validated_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(source_kb, source_version, target_kb, target_version)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_kb_version_sets_environment ON kb_version_sets(environment);
CREATE INDEX IF NOT EXISTS idx_kb_version_sets_active ON kb_version_sets(active);
CREATE INDEX IF NOT EXISTS idx_kb_version_sets_created_at ON kb_version_sets(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kb_version_deployments_version_set ON kb_version_deployments(version_set_id);
CREATE INDEX IF NOT EXISTS idx_kb_version_deployments_environment ON kb_version_deployments(environment, deployed_at DESC);
CREATE INDEX IF NOT EXISTS idx_kb_version_deployments_status ON kb_version_deployments(status);

CREATE INDEX IF NOT EXISTS idx_kb_service_versions_kb ON kb_service_versions(kb_name, version);
CREATE INDEX IF NOT EXISTS idx_kb_service_versions_status ON kb_service_versions(status);
CREATE INDEX IF NOT EXISTS idx_kb_service_versions_created_at ON kb_service_versions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kb_compatibility_source ON kb_version_compatibility(source_kb, source_version);
CREATE INDEX IF NOT EXISTS idx_kb_compatibility_target ON kb_version_compatibility(target_kb, target_version);
CREATE INDEX IF NOT EXISTS idx_kb_compatibility_status ON kb_version_compatibility(compatibility_status);

-- Functions for version management

-- Function to get active version set for environment
CREATE OR REPLACE FUNCTION get_active_version_set(env VARCHAR(50))
RETURNS TABLE (
    id UUID,
    version_set_name VARCHAR,
    kb_versions JSONB,
    validated BOOLEAN,
    activated_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        kvs.id,
        kvs.version_set_name,
        kvs.kb_versions,
        kvs.validated,
        kvs.activated_at
    FROM kb_version_sets kvs
    WHERE kvs.environment = env 
      AND kvs.active = TRUE
    ORDER BY kvs.activated_at DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to validate version set compatibility
CREATE OR REPLACE FUNCTION validate_version_set_compatibility(version_set_data JSONB)
RETURNS TABLE (
    compatible BOOLEAN,
    warnings TEXT[],
    errors TEXT[]
) AS $$
DECLARE
    kb_name TEXT;
    kb_version TEXT;
    compatibility_issues TEXT[] := '{}';
    compatibility_warnings TEXT[] := '{}';
    is_compatible BOOLEAN := TRUE;
BEGIN
    -- Check each KB version in the set
    FOR kb_name, kb_version IN 
        SELECT * FROM jsonb_each_text(version_set_data)
    LOOP
        -- Check if version exists
        IF NOT EXISTS (
            SELECT 1 FROM kb_service_versions 
            WHERE kb_name = kb_name AND version = kb_version
        ) THEN
            compatibility_issues := compatibility_issues || (kb_name || ' version ' || kb_version || ' not found');
            is_compatible := FALSE;
        END IF;
        
        -- Check for deprecated versions
        IF EXISTS (
            SELECT 1 FROM kb_service_versions 
            WHERE kb_name = kb_name 
              AND version = kb_version 
              AND status = 'deprecated'
        ) THEN
            compatibility_warnings := compatibility_warnings || (kb_name || ' version ' || kb_version || ' is deprecated');
        END IF;
    END LOOP;
    
    RETURN QUERY SELECT is_compatible, compatibility_warnings, compatibility_issues;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to deploy version set
CREATE OR REPLACE FUNCTION deploy_version_set(
    version_set_id UUID,
    target_environment VARCHAR(50),
    deployed_by VARCHAR(100)
)
RETURNS UUID AS $$
DECLARE
    deployment_id UUID;
    current_active_id UUID;
BEGIN
    -- Generate deployment ID
    deployment_id := uuid_generate_v4();
    
    -- Get current active version set for rollback
    SELECT id INTO current_active_id
    FROM kb_version_sets
    WHERE environment = target_environment AND active = TRUE;
    
    -- Deactivate current version set
    IF current_active_id IS NOT NULL THEN
        UPDATE kb_version_sets
        SET active = FALSE, deactivated_at = NOW()
        WHERE id = current_active_id;
    END IF;
    
    -- Activate new version set
    UPDATE kb_version_sets
    SET active = TRUE, activated_at = NOW()
    WHERE id = version_set_id AND environment = target_environment;
    
    -- Create deployment record
    INSERT INTO kb_version_deployments (
        id, version_set_id, environment, deployed_by,
        rollback_version_set_id, status, deployment_log
    ) VALUES (
        deployment_id, version_set_id, target_environment, deployed_by,
        current_active_id, 'deployed',
        '[{"timestamp": "' || NOW()::TEXT || '", "level": "info", "message": "Deployment completed"}]'::JSONB
    );
    
    RETURN deployment_id;
END;
$$ LANGUAGE plpgsql;

-- Function to rollback version set
CREATE OR REPLACE FUNCTION rollback_version_set(
    deployment_id UUID,
    rollback_reason TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    deployment_record RECORD;
    rollback_successful BOOLEAN := FALSE;
BEGIN
    -- Get deployment details
    SELECT * INTO deployment_record
    FROM kb_version_deployments
    WHERE id = deployment_id;
    
    IF deployment_record.rollback_version_set_id IS NOT NULL THEN
        -- Deactivate current version set
        UPDATE kb_version_sets
        SET active = FALSE, deactivated_at = NOW()
        WHERE id = deployment_record.version_set_id;
        
        -- Reactivate previous version set
        UPDATE kb_version_sets
        SET active = TRUE, activated_at = NOW()
        WHERE id = deployment_record.rollback_version_set_id;
        
        -- Update deployment record
        UPDATE kb_version_deployments
        SET 
            status = 'rolled_back',
            rollback_reason = rollback_reason,
            deployment_log = deployment_log || 
                ('{"timestamp": "' || NOW()::TEXT || '", "level": "info", "message": "Rollback completed", "reason": "' || COALESCE(rollback_reason, 'Manual rollback') || '"}')::JSONB
        WHERE id = deployment_id;
        
        rollback_successful := TRUE;
    END IF;
    
    RETURN rollback_successful;
END;
$$ LANGUAGE plpgsql;

-- Insert default version set for development
INSERT INTO kb_version_sets (
    version_set_name, description, environment, created_by, active, activated_at,
    kb_versions
) VALUES (
    'default_dev_2025_01',
    'Default development version set for January 2025',
    'dev',
    'system',
    TRUE,
    NOW(),
    '{
        "kb_1_dosing": "1.0.0+sha.initial",
        "kb_2_context": "1.0.0+sha.initial",
        "kb_3_guidelines": "1.0.0+sha.initial",
        "kb_4_safety": "1.0.0+sha.initial",
        "kb_5_ddi": "1.0.0+sha.initial",
        "kb_6_formulary": "1.0.0+sha.initial",
        "kb_7_terminology": "1.0.0+sha.initial"
    }'::JSONB
) ON CONFLICT (version_set_name) DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE kb_version_sets IS 'Central registry of KB version sets with deployment and validation tracking';
COMMENT ON TABLE kb_version_deployments IS 'History of version set deployments across environments';
COMMENT ON TABLE kb_service_versions IS 'Registry of individual KB service versions with metadata';
COMMENT ON TABLE kb_version_compatibility IS 'Compatibility matrix between different KB versions';

COMMENT ON COLUMN kb_version_sets.kb_versions IS 'JSONB mapping of KB service names to their versions';
COMMENT ON COLUMN kb_version_deployments.deployment_log IS 'JSONB array of deployment process events and logs';
COMMENT ON COLUMN kb_service_versions.breaking_changes IS 'JSONB array of breaking changes introduced in this version';