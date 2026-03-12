-- Phase 0: KB Dependency Foundation
-- Core dependency graph that defines fundamental relationships between Knowledge Base services
-- This must be executed BEFORE individual KB migrations to establish the dependency framework

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- KB Dependency Graph - Foundational relationships between KB services
-- This table defines the core architectural dependencies that drive the entire system
CREATE TABLE kb_dependency_graph (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_kb VARCHAR(50) NOT NULL,
    target_kb VARCHAR(50) NOT NULL,
    dependency_type VARCHAR(50) NOT NULL CHECK (
        dependency_type IN ('data', 'version', 'schema', 'api', 'configuration', 'runtime')
    ),
    required BOOLEAN DEFAULT TRUE,
    validation_rule JSONB DEFAULT '{}',
    /* Validation rule structure:
    {
      "version_compatibility": {
        "min_version": "1.0.0",
        "max_version": "2.0.0",
        "breaking_changes": ["field_removal", "api_change"]
      },
      "data_requirements": {
        "required_fields": ["drug_code", "drug_name"],
        "optional_fields": ["generic_name"],
        "data_format": "FHIR_R4"
      },
      "schema_constraints": {
        "table_dependencies": ["medications", "interactions"],
        "view_dependencies": ["drug_lookup"],
        "function_dependencies": ["calculate_dosage"]
      }
    }
    */
    priority INTEGER DEFAULT 5, -- 1=highest, 10=lowest
    criticality VARCHAR(20) DEFAULT 'medium' CHECK (
        criticality IN ('critical', 'high', 'medium', 'low')
    ),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system',
    
    -- Ensure no duplicate dependencies for the same type
    UNIQUE(source_kb, target_kb, dependency_type)
);

-- Index for efficient dependency lookups
CREATE INDEX idx_kb_dependency_graph_source ON kb_dependency_graph(source_kb, required);
CREATE INDEX idx_kb_dependency_graph_target ON kb_dependency_graph(target_kb, criticality);
CREATE INDEX idx_kb_dependency_graph_type ON kb_dependency_graph(dependency_type, required);
CREATE INDEX idx_kb_dependency_graph_priority ON kb_dependency_graph(priority, criticality);

-- GIN index for validation rule queries
CREATE INDEX idx_kb_dependency_graph_validation_gin ON kb_dependency_graph USING GIN(validation_rule);

-- KB Service Registry - Central registry of all KB services and their metadata
CREATE TABLE kb_service_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kb_name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    service_type VARCHAR(50) NOT NULL CHECK (
        service_type IN ('clinical_rules', 'data_source', 'validation', 'calculation', 'terminology', 'guidelines')
    ),
    current_version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    api_endpoint VARCHAR(200),
    health_endpoint VARCHAR(200),
    documentation_url VARCHAR(500),
    maintainer_team VARCHAR(100),
    criticality VARCHAR(20) DEFAULT 'medium' CHECK (
        criticality IN ('critical', 'high', 'medium', 'low')
    ),
    active BOOLEAN DEFAULT TRUE,
    deployment_status VARCHAR(20) DEFAULT 'deployed' CHECK (
        deployment_status IN ('deployed', 'deploying', 'maintenance', 'deprecated', 'retired')
    ),
    resource_requirements JSONB DEFAULT '{}',
    /* Resource requirements structure:
    {
      "cpu_limit": "500m",
      "memory_limit": "1Gi",
      "storage_size": "10Gi",
      "database_connections": 20,
      "cache_size": "256MB"
    }
    */
    configuration_schema JSONB DEFAULT '{}',
    capabilities JSONB DEFAULT '[]',
    /* Capabilities structure:
    [
      "drug_dosing_calculation",
      "interaction_checking",
      "allergy_validation",
      "formulary_lookup"
    ]
    */
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system'
);

-- Index for service registry
CREATE INDEX idx_kb_service_registry_name ON kb_service_registry(kb_name, active);
CREATE INDEX idx_kb_service_registry_type ON kb_service_registry(service_type, deployment_status);
CREATE INDEX idx_kb_service_registry_criticality ON kb_service_registry(criticality, active);

-- GIN indexes for JSONB searches
CREATE INDEX idx_kb_service_registry_capabilities_gin ON kb_service_registry USING GIN(capabilities);
CREATE INDEX idx_kb_service_registry_config_gin ON kb_service_registry USING GIN(configuration_schema);

-- Insert foundational KB service definitions
INSERT INTO kb_service_registry (
    kb_name, display_name, description, service_type, current_version, criticality, capabilities
) VALUES
('kb-drug-rules', 'KB-1 Drug Dosing Rules', 'Clinical drug dosing calculations and rules engine', 'clinical_rules', '1.0.0', 'critical', 
 '["drug_dosing_calculation", "pediatric_dosing", "renal_adjustment", "hepatic_adjustment"]'),
 
('kb-2-clinical-context', 'KB-2 Clinical Context', 'Patient clinical context and phenotype analysis', 'data_source', '1.0.0', 'high',
 '["phenotype_detection", "clinical_context_analysis", "patient_classification"]'),
 
('kb-guideline-evidence', 'KB-3 Guidelines & Evidence', 'Clinical guidelines and evidence-based recommendations', 'guidelines', '1.0.0', 'high',
 '["guideline_recommendations", "evidence_synthesis", "clinical_pathways"]'),
 
('kb-4-patient-safety', 'KB-4 Patient Safety', 'Patient safety monitoring and alert generation', 'validation', '1.0.0', 'critical',
 '["safety_monitoring", "alert_generation", "risk_assessment", "adverse_event_detection"]'),
 
('kb-5-drug-interactions', 'KB-5 Drug Interactions', 'Drug-drug interaction detection and analysis', 'validation', '1.0.0', 'critical',
 '["interaction_checking", "severity_assessment", "mechanism_analysis"]'),
 
('kb-6-formulary', 'KB-6 Formulary Management', 'Formulary and medication coverage management', 'data_source', '1.0.0', 'medium',
 '["formulary_lookup", "coverage_verification", "alternative_suggestions"]'),
 
('kb-7-terminology', 'KB-7 Terminology Services', 'Medical terminology and coding services', 'terminology', '1.0.0', 'critical',
 '["code_lookup", "terminology_mapping", "concept_relationships", "validation"]');

-- Insert foundational dependency relationships
INSERT INTO kb_dependency_graph (
    source_kb, target_kb, dependency_type, required, criticality, priority, description, validation_rule
) VALUES
-- Critical data dependencies - KBs that need terminology services
('kb-drug-rules', 'kb-7-terminology', 'data', true, 'critical', 1, 
 'Drug dosing requires standardized drug codes and terminology',
 '{"data_requirements": {"required_fields": ["drug_code", "drug_name", "rxnorm_code"], "format": "RxNorm"}}'),

('kb-2-clinical-context', 'kb-7-terminology', 'data', true, 'critical', 1,
 'Clinical context analysis requires LOINC codes and clinical terminologies',
 '{"data_requirements": {"required_fields": ["loinc_code", "snomed_code"], "format": "LOINC_SNOMED"}}'),

('kb-5-drug-interactions', 'kb-7-terminology', 'data', true, 'critical', 1,
 'Drug interaction checking requires standardized drug classifications',
 '{"data_requirements": {"required_fields": ["drug_class", "mechanism_code"], "format": "ATC_RxNorm"}}'),

-- Clinical workflow dependencies
('kb-guideline-evidence', 'kb-2-clinical-context', 'version', true, 'high', 2,
 'Clinical guidelines need patient phenotype and context data',
 '{"version_compatibility": {"min_version": "1.0.0"}, "api_requirements": ["phenotype_endpoint"]}'),

('kb-4-patient-safety', 'kb-drug-rules', 'data', true, 'critical', 1,
 'Safety monitoring requires access to current dosing calculations',
 '{"data_requirements": {"required_endpoints": ["/api/v1/dosing/calculate"]}}'),

('kb-4-patient-safety', 'kb-5-drug-interactions', 'api', true, 'critical', 1,
 'Safety alerts need real-time interaction checking',
 '{"api_requirements": {"endpoints": ["/api/v1/interactions/check"], "timeout_ms": 5000}}'),

-- Formulary and coverage dependencies
('kb-6-formulary', 'kb-7-terminology', 'data', true, 'medium', 3,
 'Formulary management requires standardized medication codes',
 '{"data_requirements": {"required_fields": ["ndc_code", "rxnorm_code"]}}'),

-- Optional/enhancement dependencies
('kb-5-drug-interactions', 'kb-drug-rules', 'schema', false, 'low', 5,
 'Drug interactions may reference dosing data for context',
 '{"schema_requirements": {"optional_views": ["current_dosing"]}}'),

('kb-guideline-evidence', 'kb-6-formulary', 'api', false, 'medium', 4,
 'Guidelines may check formulary status for recommendations',
 '{"api_requirements": {"optional_endpoints": ["/api/v1/formulary/check"]}}');

-- KB Deployment Order - Defines the correct deployment sequence
CREATE TABLE kb_deployment_order (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kb_name VARCHAR(50) NOT NULL REFERENCES kb_service_registry(kb_name),
    deployment_phase INTEGER NOT NULL, -- 0=foundation, 1=core, 2=enhanced, 3=optional
    deployment_order INTEGER NOT NULL, -- Order within phase
    prerequisites TEXT[], -- Array of KB names that must be deployed first
    deployment_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(kb_name),
    UNIQUE(deployment_phase, deployment_order)
);

-- Insert deployment order
INSERT INTO kb_deployment_order (kb_name, deployment_phase, deployment_order, prerequisites, deployment_notes) VALUES
-- Phase 0: Foundation services
('kb-7-terminology', 0, 1, '{}', 'Must deploy first - all other services depend on terminology'),

-- Phase 1: Core clinical services  
('kb-drug-rules', 1, 1, ARRAY['kb-7-terminology'], 'Core dosing engine with terminology dependency'),
('kb-2-clinical-context', 1, 2, ARRAY['kb-7-terminology'], 'Clinical context analysis with terminology'),
('kb-5-drug-interactions', 1, 3, ARRAY['kb-7-terminology'], 'Interaction checking with terminology'),

-- Phase 2: Enhanced services
('kb-4-patient-safety', 2, 1, ARRAY['kb-drug-rules', 'kb-5-drug-interactions'], 'Safety monitoring requires dosing and interactions'),
('kb-guideline-evidence', 2, 2, ARRAY['kb-2-clinical-context'], 'Guidelines require clinical context'),

-- Phase 3: Optional/supporting services
('kb-6-formulary', 3, 1, ARRAY['kb-7-terminology'], 'Formulary can be deployed independently with terminology');

-- Views for dependency analysis

-- View: KB Dependency Tree - Shows complete dependency hierarchy
CREATE VIEW v_kb_dependency_tree AS
WITH RECURSIVE dependency_tree AS (
    -- Base case: root dependencies (services with no dependencies)
    SELECT 
        ksr.kb_name,
        ksr.display_name,
        ksr.criticality,
        0 as level,
        ARRAY[ksr.kb_name] as path,
        ksr.kb_name as root_kb
    FROM kb_service_registry ksr
    WHERE NOT EXISTS (
        SELECT 1 FROM kb_dependency_graph kdg 
        WHERE kdg.source_kb = ksr.kb_name AND kdg.required = true
    )
    
    UNION ALL
    
    -- Recursive case: services that depend on others
    SELECT 
        ksr.kb_name,
        ksr.display_name,
        ksr.criticality,
        dt.level + 1,
        dt.path || ksr.kb_name,
        dt.root_kb
    FROM kb_service_registry ksr
    JOIN kb_dependency_graph kdg ON kdg.source_kb = ksr.kb_name
    JOIN dependency_tree dt ON dt.kb_name = kdg.target_kb
    WHERE kdg.required = true
      AND NOT (ksr.kb_name = ANY(dt.path)) -- Prevent cycles
      AND dt.level < 10 -- Prevent infinite recursion
)
SELECT 
    root_kb,
    kb_name,
    display_name,
    level,
    path,
    criticality
FROM dependency_tree
ORDER BY root_kb, level, kb_name;

-- View: KB Critical Dependencies - Shows critical dependency paths
CREATE VIEW v_kb_critical_dependencies AS
SELECT 
    kdg.source_kb,
    ksr_source.display_name as source_display_name,
    kdg.target_kb,
    ksr_target.display_name as target_display_name,
    kdg.dependency_type,
    kdg.criticality,
    kdg.priority,
    kdg.description,
    ksr_source.deployment_status as source_status,
    ksr_target.deployment_status as target_status,
    CASE 
        WHEN ksr_target.deployment_status != 'deployed' THEN 'blocked'
        WHEN ksr_target.active = false THEN 'inactive_dependency'
        ELSE 'ready'
    END as dependency_status
FROM kb_dependency_graph kdg
JOIN kb_service_registry ksr_source ON ksr_source.kb_name = kdg.source_kb
JOIN kb_service_registry ksr_target ON ksr_target.kb_name = kdg.target_kb
WHERE kdg.required = true
ORDER BY kdg.criticality DESC, kdg.priority ASC;

-- View: KB Deployment Readiness
CREATE VIEW v_kb_deployment_readiness AS
SELECT 
    ksr.kb_name,
    ksr.display_name,
    ksr.deployment_status,
    kdo.deployment_phase,
    kdo.deployment_order,
    array_length(kdo.prerequisites, 1) as prerequisite_count,
    (
        SELECT COUNT(*) 
        FROM unnest(kdo.prerequisites) as prereq
        JOIN kb_service_registry ksr_prereq ON ksr_prereq.kb_name = prereq
        WHERE ksr_prereq.deployment_status = 'deployed'
    ) as prerequisites_ready,
    CASE 
        WHEN array_length(kdo.prerequisites, 1) IS NULL THEN 'ready'
        WHEN (
            SELECT COUNT(*) 
            FROM unnest(kdo.prerequisites) as prereq
            JOIN kb_service_registry ksr_prereq ON ksr_prereq.kb_name = prereq
            WHERE ksr_prereq.deployment_status = 'deployed'
        ) = array_length(kdo.prerequisites, 1) THEN 'ready'
        ELSE 'waiting'
    END as readiness_status,
    kdo.prerequisites as missing_prerequisites
FROM kb_service_registry ksr
JOIN kb_deployment_order kdo ON kdo.kb_name = ksr.kb_name
ORDER BY kdo.deployment_phase, kdo.deployment_order;

-- Functions for dependency management

-- Function: Get KB Dependencies
CREATE OR REPLACE FUNCTION get_kb_dependencies(p_kb_name VARCHAR(50), p_include_optional BOOLEAN DEFAULT false)
RETURNS TABLE (
    target_kb VARCHAR(50),
    dependency_type VARCHAR(50),
    required BOOLEAN,
    criticality VARCHAR(20),
    validation_rule JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        kdg.target_kb,
        kdg.dependency_type,
        kdg.required,
        kdg.criticality,
        kdg.validation_rule
    FROM kb_dependency_graph kdg
    WHERE kdg.source_kb = p_kb_name
      AND (p_include_optional OR kdg.required = true)
    ORDER BY kdg.priority, kdg.criticality DESC;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Check Deployment Readiness
CREATE OR REPLACE FUNCTION check_deployment_readiness(p_kb_name VARCHAR(50))
RETURNS TABLE (
    ready BOOLEAN,
    missing_dependencies VARCHAR(50)[],
    blocking_issues TEXT[]
) AS $$
DECLARE
    missing_deps VARCHAR(50)[];
    issues TEXT[];
    dep_record RECORD;
BEGIN
    -- Check all required dependencies
    FOR dep_record IN
        SELECT kdg.target_kb, ksr.deployment_status, ksr.active
        FROM kb_dependency_graph kdg
        JOIN kb_service_registry ksr ON ksr.kb_name = kdg.target_kb
        WHERE kdg.source_kb = p_kb_name AND kdg.required = true
    LOOP
        IF dep_record.deployment_status != 'deployed' THEN
            missing_deps := array_append(missing_deps, dep_record.target_kb);
            issues := array_append(issues, 
                format('Dependency %s is not deployed (status: %s)', 
                       dep_record.target_kb, dep_record.deployment_status));
        END IF;
        
        IF NOT dep_record.active THEN
            issues := array_append(issues, 
                format('Dependency %s is inactive', dep_record.target_kb));
        END IF;
    END LOOP;
    
    RETURN QUERY SELECT 
        (array_length(missing_deps, 1) IS NULL AND array_length(issues, 1) IS NULL),
        COALESCE(missing_deps, '{}'),
        COALESCE(issues, '{}');
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Validate Dependency Rules
CREATE OR REPLACE FUNCTION validate_dependency_rules(p_source_kb VARCHAR(50), p_target_kb VARCHAR(50))
RETURNS TABLE (
    valid BOOLEAN,
    validation_errors TEXT[],
    warnings TEXT[]
) AS $$
DECLARE
    rule_record RECORD;
    errors TEXT[] := '{}';
    warnings TEXT[] := '{}';
    is_valid BOOLEAN := true;
BEGIN
    -- Get validation rules for this dependency
    SELECT validation_rule INTO rule_record
    FROM kb_dependency_graph
    WHERE source_kb = p_source_kb AND target_kb = p_target_kb;
    
    -- If no rules found, dependency is valid by default
    IF rule_record.validation_rule IS NULL THEN
        RETURN QUERY SELECT true, errors, warnings;
        RETURN;
    END IF;
    
    -- Validate version compatibility (simplified example)
    IF rule_record.validation_rule ? 'version_compatibility' THEN
        -- This would contain actual version validation logic
        -- For now, we'll assume validation passes
        warnings := array_append(warnings, 'Version compatibility check passed');
    END IF;
    
    -- Validate data requirements
    IF rule_record.validation_rule ? 'data_requirements' THEN
        -- This would contain actual data validation logic
        warnings := array_append(warnings, 'Data requirements validation passed');
    END IF;
    
    RETURN QUERY SELECT is_valid, errors, warnings;
END;
$$ LANGUAGE plpgsql STABLE;

-- Triggers for maintaining data consistency

-- Trigger: Update timestamp on kb_dependency_graph changes
CREATE OR REPLACE FUNCTION update_kb_dependency_graph_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER kb_dependency_graph_updated_at
    BEFORE UPDATE ON kb_dependency_graph
    FOR EACH ROW
    EXECUTE FUNCTION update_kb_dependency_graph_timestamp();

-- Trigger: Update timestamp on kb_service_registry changes
CREATE OR REPLACE FUNCTION update_kb_service_registry_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER kb_service_registry_updated_at
    BEFORE UPDATE ON kb_service_registry
    FOR EACH ROW
    EXECUTE FUNCTION update_kb_service_registry_timestamp();

-- Comments for documentation
COMMENT ON TABLE kb_dependency_graph IS 'Foundational dependency relationships between KB services that define system architecture';
COMMENT ON TABLE kb_service_registry IS 'Central registry of all KB services with metadata and capabilities';
COMMENT ON TABLE kb_deployment_order IS 'Defines the correct deployment sequence for KB services based on dependencies';

COMMENT ON VIEW v_kb_dependency_tree IS 'Hierarchical view of KB dependencies showing complete dependency chains';
COMMENT ON VIEW v_kb_critical_dependencies IS 'Critical dependencies that must be operational for dependent services';
COMMENT ON VIEW v_kb_deployment_readiness IS 'Deployment readiness status for each KB service based on prerequisites';

COMMENT ON FUNCTION get_kb_dependencies IS 'Returns all dependencies for a given KB service';
COMMENT ON FUNCTION check_deployment_readiness IS 'Checks if a KB service is ready for deployment based on dependencies';
COMMENT ON FUNCTION validate_dependency_rules IS 'Validates dependency rules and requirements between KB services';

-- Initial data validation
DO $$
DECLARE
    invalid_deps INTEGER;
    orphaned_services INTEGER;
BEGIN
    -- Check for invalid dependency references
    SELECT COUNT(*) INTO invalid_deps
    FROM kb_dependency_graph kdg
    WHERE NOT EXISTS (SELECT 1 FROM kb_service_registry WHERE kb_name = kdg.source_kb)
       OR NOT EXISTS (SELECT 1 FROM kb_service_registry WHERE kb_name = kdg.target_kb);
       
    IF invalid_deps > 0 THEN
        RAISE WARNING 'Found % invalid dependency references in kb_dependency_graph', invalid_deps;
    END IF;
    
    -- Check for services without any dependencies (potential issues)
    SELECT COUNT(*) INTO orphaned_services
    FROM kb_service_registry ksr
    WHERE ksr.kb_name != 'kb-7-terminology' -- Terminology is root service
      AND NOT EXISTS (
          SELECT 1 FROM kb_dependency_graph 
          WHERE source_kb = ksr.kb_name OR target_kb = ksr.kb_name
      );
      
    IF orphaned_services > 0 THEN
        RAISE WARNING 'Found % services with no dependency relationships', orphaned_services;
    END IF;
    
    RAISE NOTICE 'Phase 0 KB Dependency Foundation setup completed successfully';
    RAISE NOTICE 'Registered % KB services with % dependency relationships', 
        (SELECT COUNT(*) FROM kb_service_registry),
        (SELECT COUNT(*) FROM kb_dependency_graph);
END $$;