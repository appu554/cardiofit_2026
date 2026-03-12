-- Inter-KB Referential Integrity and Data Lineage
-- Part III: Inter-KB Dependency Tracking System

-- Dependencies tracking between KBs
CREATE TABLE IF NOT EXISTS kb_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source dependency (the KB that depends on something)
    source_kb VARCHAR(50) NOT NULL,
    source_artifact_type VARCHAR(50) NOT NULL, -- 'rule', 'guideline', 'data', 'model', 'service'
    source_artifact_id VARCHAR(200) NOT NULL,
    source_version VARCHAR(50) NOT NULL,
    source_endpoint VARCHAR(200),
    
    -- Target dependency (what the source depends on)
    target_kb VARCHAR(50) NOT NULL,
    target_artifact_type VARCHAR(50) NOT NULL,
    target_artifact_id VARCHAR(200) NOT NULL,
    target_version VARCHAR(50) NOT NULL,
    target_endpoint VARCHAR(200),
    
    -- Dependency details
    dependency_type VARCHAR(50) NOT NULL CHECK (
        dependency_type IN ('references', 'extends', 'conflicts', 'overrides', 'validates', 'transforms')
    ),
    dependency_strength VARCHAR(20) NOT NULL DEFAULT 'medium' CHECK (
        dependency_strength IN ('critical', 'strong', 'medium', 'weak', 'optional')
    ),
    
    -- Relationship metadata
    relationship_description TEXT,
    relationship_context JSONB DEFAULT '{}',
    /* Structure:
    {
      "usage_pattern": "real_time|batch|occasional",
      "data_flow_direction": "bidirectional|source_to_target|target_to_source",
      "failure_impact": "cascading|isolated|degraded",
      "business_criticality": "critical|high|medium|low",
      "integration_type": "api_call|data_reference|rule_inheritance|validation_check"
    }
    */
    
    -- Performance characteristics
    typical_usage_frequency INTEGER, -- calls per hour
    average_response_time_ms INTEGER,
    failure_rate_percent DECIMAL(5,2),
    
    -- Validation status
    validated BOOLEAN DEFAULT FALSE,
    validation_timestamp TIMESTAMPTZ,
    validation_method VARCHAR(50),
    validation_errors JSONB DEFAULT '[]',
    validation_warnings JSONB DEFAULT '[]',
    
    -- Health monitoring
    last_verified TIMESTAMPTZ,
    health_status VARCHAR(20) DEFAULT 'unknown' CHECK (
        health_status IN ('healthy', 'degraded', 'failing', 'broken', 'unknown')
    ),
    health_check_details JSONB DEFAULT '{}',
    
    -- Discovery metadata
    discovered_by VARCHAR(50) NOT NULL, -- 'manual', 'automated_scan', 'runtime_detection', 'static_analysis'
    discovered_at TIMESTAMPTZ DEFAULT NOW(),
    discovery_confidence DECIMAL(3,2) DEFAULT 0.5,
    
    -- Lifecycle
    active BOOLEAN DEFAULT TRUE,
    deprecated BOOLEAN DEFAULT FALSE,
    deprecated_reason TEXT,
    deprecated_at TIMESTAMPTZ,
    replacement_dependency_id UUID REFERENCES kb_dependencies(id),
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100) NOT NULL,
    last_modified_by VARCHAR(100),
    
    UNIQUE(source_kb, source_artifact_id, source_version, target_kb, target_artifact_id, target_version)
);

-- Change impact analysis tracking
CREATE TABLE IF NOT EXISTS change_impact_analysis (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Change details
    changed_kb VARCHAR(50) NOT NULL,
    changed_artifact_type VARCHAR(50) NOT NULL,
    changed_artifact_id VARCHAR(200) NOT NULL,
    change_type VARCHAR(50) NOT NULL CHECK (
        change_type IN ('create', 'update', 'delete', 'deprecate', 'version_upgrade', 'configuration_change')
    ),
    old_version VARCHAR(50),
    new_version VARCHAR(50),
    change_description TEXT,
    
    -- Change metadata
    change_scope VARCHAR(20) DEFAULT 'minor' CHECK (
        change_scope IN ('major', 'minor', 'patch', 'hotfix', 'breaking')
    ),
    breaking_change BOOLEAN DEFAULT FALSE,
    backward_compatible BOOLEAN DEFAULT TRUE,
    
    -- Impact assessment results
    analysis_status VARCHAR(20) DEFAULT 'pending' CHECK (
        analysis_status IN ('pending', 'running', 'completed', 'failed', 'cancelled')
    ),
    
    direct_impacts JSONB NOT NULL DEFAULT '[]',
    /* Structure:
    [
      {
        "target_kb": "kb_guidelines",
        "target_artifact": "htn_treatment_pathway",
        "impact_type": "reference_broken|dependency_updated|validation_required",
        "impact_severity": "critical|major|minor|negligible",
        "required_action": "immediate_fix|scheduled_update|no_action",
        "estimated_effort_hours": 4,
        "mitigation_strategy": "update_reference|add_fallback|deprecate_gracefully"
      }
    ]
    */
    
    indirect_impacts JSONB DEFAULT '[]',
    /* Structure: similar to direct_impacts but for downstream dependencies */
    
    cascade_impacts JSONB DEFAULT '[]',
    /* Multi-level cascading impact analysis */
    
    -- Scope and metrics
    total_affected_artifacts INTEGER DEFAULT 0,
    affected_kb_services TEXT[] DEFAULT '{}',
    estimated_patient_impact INTEGER DEFAULT 0,
    estimated_downtime_minutes INTEGER DEFAULT 0,
    
    -- Risk assessment
    risk_score DECIMAL(3,2) DEFAULT 0.0,
    risk_level VARCHAR(20) DEFAULT 'low' CHECK (
        risk_level IN ('critical', 'high', 'medium', 'low', 'minimal')
    ),
    risk_factors JSONB DEFAULT '[]',
    /* Structure:
    [
      {
        "factor": "high_usage_dependency",
        "description": "Dependency used in >1000 requests/hour",
        "weight": 0.8,
        "mitigation": "staged_rollout_recommended"
      }
    ]
    */
    
    -- Recommendations
    recommended_actions JSONB DEFAULT '[]',
    rollback_plan TEXT,
    testing_requirements JSONB DEFAULT '[]',
    
    -- Approval workflow
    requires_approval BOOLEAN DEFAULT FALSE,
    approval_status VARCHAR(20) DEFAULT 'not_required' CHECK (
        approval_status IN ('not_required', 'pending', 'approved', 'rejected', 'conditional')
    ),
    approved_by VARCHAR(100),
    approval_timestamp TIMESTAMPTZ,
    approval_conditions TEXT,
    
    -- Execution tracking
    execution_status VARCHAR(20) DEFAULT 'not_started' CHECK (
        execution_status IN ('not_started', 'planned', 'in_progress', 'completed', 'failed', 'rolled_back')
    ),
    execution_started_at TIMESTAMPTZ,
    execution_completed_at TIMESTAMPTZ,
    execution_notes TEXT,
    
    -- Validation
    pre_change_validation JSONB DEFAULT '{}',
    post_change_validation JSONB DEFAULT '{}',
    validation_passed BOOLEAN,
    
    -- Metadata
    created_by VARCHAR(100) NOT NULL,
    assigned_to VARCHAR(100),
    priority INTEGER DEFAULT 5, -- 1=highest, 10=lowest
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Automated cascade update tracking
CREATE TABLE IF NOT EXISTS cascade_updates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trigger_change_id UUID REFERENCES change_impact_analysis(id) NOT NULL,
    
    -- Update details
    target_kb VARCHAR(50) NOT NULL,
    target_artifact_type VARCHAR(50) NOT NULL,
    target_artifact_id VARCHAR(200) NOT NULL,
    current_version VARCHAR(50),
    target_version VARCHAR(50),
    
    update_type VARCHAR(50) NOT NULL CHECK (
        update_type IN ('version_update', 'reference_fix', 'configuration_sync', 'data_migration', 'validation_update')
    ),
    update_priority INTEGER DEFAULT 5,
    
    -- Automation details
    automated BOOLEAN DEFAULT FALSE,
    automation_rule_id UUID,
    manual_override BOOLEAN DEFAULT FALSE,
    
    -- Execution tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (
        status IN ('pending', 'scheduled', 'running', 'completed', 'failed', 'skipped', 'cancelled')
    ),
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Execution details
    execution_method VARCHAR(50), -- 'api_call', 'database_update', 'configuration_reload', 'service_restart'
    execution_result JSONB DEFAULT '{}',
    execution_log JSONB DEFAULT '[]',
    
    -- Validation
    pre_update_validation JSONB DEFAULT '{}',
    post_update_validation JSONB DEFAULT '{}',
    validation_passed BOOLEAN,
    
    -- Error handling
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    
    -- Dependencies
    depends_on UUID[] DEFAULT '{}', -- Other cascade_updates this depends on
    blocks UUID[] DEFAULT '{}', -- Other cascade_updates this blocks
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Data lineage tracking for Apache Atlas integration
CREATE TABLE IF NOT EXISTS data_lineage_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Event identification
    event_type VARCHAR(50) NOT NULL CHECK (
        event_type IN ('data_read', 'data_write', 'data_transform', 'data_validate', 'data_delete')
    ),
    transaction_id VARCHAR(100),
    evidence_envelope_id UUID,
    
    -- Source data
    source_kb VARCHAR(50) NOT NULL,
    source_artifact_type VARCHAR(50) NOT NULL,
    source_artifact_id VARCHAR(200) NOT NULL,
    source_version VARCHAR(50),
    source_location TEXT, -- database, file path, API endpoint
    
    -- Target data (for transforms and writes)
    target_kb VARCHAR(50),
    target_artifact_type VARCHAR(50),
    target_artifact_id VARCHAR(200),
    target_version VARCHAR(50),
    target_location TEXT,
    
    -- Transformation details
    transformation_type VARCHAR(50), -- 'aggregation', 'mapping', 'enrichment', 'validation', 'normalization'
    transformation_logic JSONB DEFAULT '{}',
    data_quality_metrics JSONB DEFAULT '{}',
    
    -- Data characteristics
    data_schema JSONB DEFAULT '{}',
    data_size_bytes BIGINT,
    record_count INTEGER,
    data_hash VARCHAR(64), -- For integrity checking
    
    -- Processing metadata
    processing_node VARCHAR(100),
    processing_duration_ms INTEGER,
    cpu_time_ms INTEGER,
    memory_used_mb DECIMAL(10,2),
    
    -- Quality and governance
    data_classification VARCHAR(50), -- 'public', 'internal', 'confidential', 'restricted'
    pii_detected BOOLEAN DEFAULT FALSE,
    phi_detected BOOLEAN DEFAULT FALSE,
    compliance_tags TEXT[] DEFAULT '{}',
    
    -- Lineage relationships
    parent_events UUID[] DEFAULT '{}',
    child_events UUID[] DEFAULT '{}',
    related_events UUID[] DEFAULT '{}',
    
    -- Apache Atlas integration
    atlas_entity_guid VARCHAR(100),
    atlas_process_guid VARCHAR(100),
    atlas_lineage_id VARCHAR(100),
    
    -- Error handling
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    warning_count INTEGER DEFAULT 0,
    warnings JSONB DEFAULT '[]'
);

-- KB service health and availability tracking
CREATE TABLE IF NOT EXISTS kb_service_health (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Service identification
    kb_name VARCHAR(50) NOT NULL,
    kb_version VARCHAR(50) NOT NULL,
    service_endpoint VARCHAR(200) NOT NULL,
    
    -- Health check results
    status VARCHAR(20) NOT NULL CHECK (
        status IN ('healthy', 'degraded', 'unhealthy', 'unreachable', 'unknown')
    ),
    response_time_ms INTEGER,
    http_status_code INTEGER,
    
    -- Detailed health metrics
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_percent DECIMAL(5,2),
    disk_usage_percent DECIMAL(5,2),
    active_connections INTEGER,
    queue_length INTEGER,
    
    -- Performance indicators
    requests_per_second DECIMAL(8,2),
    errors_per_minute DECIMAL(8,2),
    average_latency_ms DECIMAL(8,2),
    cache_hit_rate DECIMAL(3,2),
    
    -- Dependencies health
    dependency_health JSONB DEFAULT '{}',
    /* Structure:
    {
      "database": {"status": "healthy", "latency_ms": 5},
      "cache": {"status": "degraded", "hit_rate": 0.85},
      "external_apis": {"status": "healthy", "avg_response_ms": 150}
    }
    */
    
    -- Service-specific metrics
    custom_metrics JSONB DEFAULT '{}',
    
    -- Error details
    error_message TEXT,
    warning_messages TEXT[],
    
    -- Check metadata
    check_type VARCHAR(30) DEFAULT 'automated' CHECK (
        check_type IN ('automated', 'manual', 'synthetic', 'user_reported')
    ),
    check_source VARCHAR(50), -- monitoring system, user, etc.
    
    INDEX idx_kb_health_timestamp (check_timestamp DESC),
    INDEX idx_kb_health_service (kb_name, kb_version),
    INDEX idx_kb_health_status (status, check_timestamp DESC)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_source ON kb_dependencies(source_kb, source_artifact_id, source_version);
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_target ON kb_dependencies(target_kb, target_artifact_id, target_version);
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_type ON kb_dependencies(dependency_type, dependency_strength);
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_active ON kb_dependencies(active, health_status);
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_discovery ON kb_dependencies(discovered_by, discovered_at DESC);

CREATE INDEX IF NOT EXISTS idx_change_impact_analysis_kb ON change_impact_analysis(changed_kb, analysis_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_change_impact_analysis_status ON change_impact_analysis(analysis_status, execution_status);
CREATE INDEX IF NOT EXISTS idx_change_impact_analysis_approval ON change_impact_analysis(approval_status, requires_approval);
CREATE INDEX IF NOT EXISTS idx_change_impact_analysis_risk ON change_impact_analysis(risk_level, risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_change_impact_analysis_assigned ON change_impact_analysis(assigned_to, priority);

CREATE INDEX IF NOT EXISTS idx_cascade_updates_trigger ON cascade_updates(trigger_change_id);
CREATE INDEX IF NOT EXISTS idx_cascade_updates_target ON cascade_updates(target_kb, target_artifact_id);
CREATE INDEX IF NOT EXISTS idx_cascade_updates_status ON cascade_updates(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_cascade_updates_dependencies ON cascade_updates USING GIN(depends_on);

CREATE INDEX IF NOT EXISTS idx_data_lineage_timestamp ON data_lineage_events(event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_data_lineage_source ON data_lineage_events(source_kb, source_artifact_id);
CREATE INDEX IF NOT EXISTS idx_data_lineage_target ON data_lineage_events(target_kb, target_artifact_id);
CREATE INDEX IF NOT EXISTS idx_data_lineage_transaction ON data_lineage_events(transaction_id);
CREATE INDEX IF NOT EXISTS idx_data_lineage_atlas ON data_lineage_events(atlas_entity_guid, atlas_process_guid);

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_kb_dependencies_context_gin ON kb_dependencies USING GIN(relationship_context);
CREATE INDEX IF NOT EXISTS idx_change_impact_direct_gin ON change_impact_analysis USING GIN(direct_impacts);
CREATE INDEX IF NOT EXISTS idx_change_impact_indirect_gin ON change_impact_analysis USING GIN(indirect_impacts);
CREATE INDEX IF NOT EXISTS idx_cascade_updates_result_gin ON cascade_updates USING GIN(execution_result);
CREATE INDEX IF NOT EXISTS idx_data_lineage_transform_gin ON data_lineage_events USING GIN(transformation_logic);

-- Functions for dependency management

-- Function to discover dependencies from runtime transactions
CREATE OR REPLACE FUNCTION discover_dependencies_from_transactions(
    lookback_hours INTEGER DEFAULT 24
)
RETURNS INTEGER AS $$
DECLARE
    transaction_record RECORD;
    kb_call_record JSONB;
    dependency_count INTEGER := 0;
    kb1_name VARCHAR(50);
    kb2_name VARCHAR(50);
    kb1_version VARCHAR(50);
    kb2_version VARCHAR(50);
BEGIN
    -- Analyze recent evidence envelopes to discover KB interactions
    FOR transaction_record IN
        SELECT transaction_id, kb_versions, decision_chain
        FROM evidence_envelopes
        WHERE created_at >= NOW() - (lookback_hours || ' hours')::INTERVAL
          AND jsonb_array_length(decision_chain) > 1
    LOOP
        -- Extract KB interactions from decision chain
        FOR kb_call_record IN
            SELECT * FROM jsonb_array_elements(transaction_record.decision_chain)
        LOOP
            -- Check if this represents a KB dependency
            IF kb_call_record ? 'kb_calls' AND jsonb_array_length(kb_call_record->'kb_calls') > 1 THEN
                -- Multiple KB calls in one decision phase suggest dependencies
                FOR kb1_name, kb2_name IN
                    SELECT 
                        kc1.value::text as kb1,
                        kc2.value::text as kb2
                    FROM jsonb_array_elements(kb_call_record->'kb_calls') kc1,
                         jsonb_array_elements(kb_call_record->'kb_calls') kc2
                    WHERE kc1.value::text < kc2.value::text
                LOOP
                    kb1_version := transaction_record.kb_versions->>kb1_name;
                    kb2_version := transaction_record.kb_versions->>kb2_name;
                    
                    -- Insert or update dependency record
                    INSERT INTO kb_dependencies (
                        source_kb, source_artifact_type, source_artifact_id, source_version,
                        target_kb, target_artifact_type, target_artifact_id, target_version,
                        dependency_type, dependency_strength, discovered_by, discovery_confidence,
                        relationship_context, created_by
                    ) VALUES (
                        kb1_name, 'service', 'runtime_interaction', kb1_version,
                        kb2_name, 'service', 'runtime_interaction', kb2_version,
                        'references', 'medium', 'runtime_detection', 0.7,
                        jsonb_build_object(
                            'usage_pattern', 'real_time',
                            'data_flow_direction', 'bidirectional',
                            'integration_type', 'api_call',
                            'discovered_in_transaction', transaction_record.transaction_id
                        ),
                        'system'
                    ) ON CONFLICT (source_kb, source_artifact_id, source_version, target_kb, target_artifact_id, target_version)
                    DO UPDATE SET
                        last_verified = NOW(),
                        discovery_confidence = LEAST(kb_dependencies.discovery_confidence + 0.1, 1.0),
                        relationship_context = relationship_context || jsonb_build_object(
                            'last_seen_transaction', transaction_record.transaction_id,
                            'verification_count', COALESCE((relationship_context->>'verification_count')::INTEGER, 0) + 1
                        );
                    
                    dependency_count := dependency_count + 1;
                END LOOP;
            END IF;
        END LOOP;
    END LOOP;
    
    RETURN dependency_count;
END;
$$ LANGUAGE plpgsql;

-- Function to analyze change impact
CREATE OR REPLACE FUNCTION analyze_change_impact(
    p_kb_name VARCHAR(50),
    p_artifact_id VARCHAR(200),
    p_change_type VARCHAR(50),
    p_old_version VARCHAR(50) DEFAULT NULL,
    p_new_version VARCHAR(50) DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    analysis_id UUID;
    direct_deps RECORD;
    impact_score DECIMAL(3,2) := 0.0;
    direct_impacts JSONB := '[]';
    risk_level VARCHAR(20) := 'low';
BEGIN
    -- Create analysis record
    INSERT INTO change_impact_analysis (
        changed_kb, changed_artifact_id, change_type,
        old_version, new_version, created_by
    ) VALUES (
        p_kb_name, p_artifact_id, p_change_type,
        p_old_version, p_new_version, 'system'
    ) RETURNING id INTO analysis_id;
    
    -- Find direct dependencies
    FOR direct_deps IN
        SELECT * FROM kb_dependencies
        WHERE target_kb = p_kb_name
          AND target_artifact_id = p_artifact_id
          AND active = TRUE
    LOOP
        -- Calculate impact based on dependency strength
        CASE direct_deps.dependency_strength
            WHEN 'critical' THEN impact_score := impact_score + 1.0;
            WHEN 'strong' THEN impact_score := impact_score + 0.7;
            WHEN 'medium' THEN impact_score := impact_score + 0.5;
            WHEN 'weak' THEN impact_score := impact_score + 0.3;
            WHEN 'optional' THEN impact_score := impact_score + 0.1;
        END CASE;
        
        -- Add to direct impacts
        direct_impacts := direct_impacts || jsonb_build_object(
            'target_kb', direct_deps.source_kb,
            'target_artifact', direct_deps.source_artifact_id,
            'impact_type', CASE p_change_type
                WHEN 'delete' THEN 'reference_broken'
                WHEN 'version_upgrade' THEN 'dependency_updated'
                ELSE 'validation_required'
            END,
            'impact_severity', CASE direct_deps.dependency_strength
                WHEN 'critical' THEN 'critical'
                WHEN 'strong' THEN 'major'
                ELSE 'minor'
            END,
            'dependency_strength', direct_deps.dependency_strength
        );
    END LOOP;
    
    -- Determine risk level based on impact score
    risk_level := CASE 
        WHEN impact_score >= 3.0 THEN 'critical'
        WHEN impact_score >= 2.0 THEN 'high'
        WHEN impact_score >= 1.0 THEN 'medium'
        ELSE 'low'
    END;
    
    -- Update analysis with results
    UPDATE change_impact_analysis
    SET 
        direct_impacts = direct_impacts,
        risk_score = impact_score,
        risk_level = risk_level,
        requires_approval = (risk_level IN ('critical', 'high')),
        analysis_status = 'completed'
    WHERE id = analysis_id;
    
    RETURN analysis_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create cascade updates
CREATE OR REPLACE FUNCTION create_cascade_updates(p_change_analysis_id UUID)
RETURNS INTEGER AS $$
DECLARE
    analysis_record RECORD;
    impact_record JSONB;
    cascade_count INTEGER := 0;
BEGIN
    -- Get the change analysis details
    SELECT * INTO analysis_record
    FROM change_impact_analysis
    WHERE id = p_change_analysis_id;
    
    -- Process each direct impact
    FOR impact_record IN
        SELECT * FROM jsonb_array_elements(analysis_record.direct_impacts)
    LOOP
        -- Create cascade update for each impacted dependency
        INSERT INTO cascade_updates (
            trigger_change_id,
            target_kb,
            target_artifact_type,
            target_artifact_id,
            update_type,
            update_priority,
            automated
        ) VALUES (
            p_change_analysis_id,
            impact_record->>'target_kb',
            'service', -- Default to service type
            impact_record->>'target_artifact',
            CASE impact_record->>'impact_type'
                WHEN 'reference_broken' THEN 'reference_fix'
                WHEN 'dependency_updated' THEN 'version_update'
                ELSE 'validation_update'
            END,
            CASE impact_record->>'impact_severity'
                WHEN 'critical' THEN 1
                WHEN 'major' THEN 3
                WHEN 'minor' THEN 7
                ELSE 10
            END,
            (impact_record->>'impact_severity' IN ('minor', 'negligible'))
        );
        
        cascade_count := cascade_count + 1;
    END LOOP;
    
    RETURN cascade_count;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE kb_dependencies IS 'Tracks dependencies and relationships between KB services and their artifacts';
COMMENT ON TABLE change_impact_analysis IS 'Analyzes the impact of changes across the KB ecosystem with risk assessment';
COMMENT ON TABLE cascade_updates IS 'Manages automated and manual cascade updates triggered by KB changes';
COMMENT ON TABLE data_lineage_events IS 'Records data lineage events for compliance and debugging purposes';
COMMENT ON TABLE kb_service_health IS 'Tracks health and performance metrics for KB services and their dependencies';

COMMENT ON COLUMN kb_dependencies.relationship_context IS 'JSONB containing detailed relationship metadata and usage patterns';
COMMENT ON COLUMN change_impact_analysis.direct_impacts IS 'JSONB array of immediately affected dependencies and their impact details';
COMMENT ON COLUMN cascade_updates.execution_result IS 'JSONB containing detailed results of the cascade update execution';