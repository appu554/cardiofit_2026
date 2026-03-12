-- +migrate Up
-- 4-Phase Workflow execution tracking with Google FHIR integration
-- This migration adds tables for tracking workflow execution state and operational data

-- Workflow executions table for 4-Phase Workflow tracking
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Workflow identification
    workflow_id VARCHAR(255) NOT NULL UNIQUE,
    parent_workflow_id VARCHAR(255), -- For nested workflows
    patient_id UUID NOT NULL,
    -- Google FHIR resource references (not data duplication)
    patient_fhir_reference JSONB NOT NULL, -- FHIRResourceReference
    clinical_context_fhir_references JSONB DEFAULT '[]', -- Array of FHIRResourceReference
    -- Workflow specification
    workflow_type VARCHAR(100) NOT NULL CHECK (workflow_type IN ('medication_recommendation', 'medication_adjustment', 'safety_review', 'clinical_validation')),
    protocol_id VARCHAR(255),
    recipe_id UUID,
    requested_by VARCHAR(255) NOT NULL,
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent', 'emergency')),
    -- 4-Phase execution tracking
    current_phase INTEGER DEFAULT 1 CHECK (current_phase BETWEEN 1 AND 4),
    phase_1_recipe_resolution JSONB DEFAULT '{}', -- {status, start_time, end_time, results, errors}
    phase_2_context_assembly JSONB DEFAULT '{}',
    phase_3_clinical_intelligence JSONB DEFAULT '{}', 
    phase_4_proposal_generation JSONB DEFAULT '{}',
    -- Overall execution status
    execution_status VARCHAR(50) NOT NULL DEFAULT 'initializing' 
        CHECK (execution_status IN ('initializing', 'in_progress', 'completed', 'failed', 'cancelled', 'timeout')),
    -- Performance tracking (targeting <250ms end-to-end)
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    total_execution_time_ms INTEGER,
    performance_target_ms INTEGER DEFAULT 250,
    performance_target_met BOOLEAN,
    -- Resource usage tracking
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_mb INTEGER,
    database_queries_count INTEGER DEFAULT 0,
    external_api_calls_count INTEGER DEFAULT 0,
    -- Quality assurance
    quality_score DECIMAL(3,2), -- 0.00 to 1.00
    validation_passed BOOLEAN,
    safety_checks_passed BOOLEAN,
    clinical_review_required BOOLEAN DEFAULT FALSE,
    -- Error handling and recovery
    error_count INTEGER DEFAULT 0,
    warnings_count INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    recovery_actions JSONB DEFAULT '[]',
    -- Results and outputs
    medication_proposals_generated INTEGER DEFAULT 0,
    clinical_recommendations_count INTEGER DEFAULT 0,
    safety_alerts_count INTEGER DEFAULT 0,
    -- Audit and compliance
    audit_trail JSONB DEFAULT '[]',
    compliance_flags JSONB DEFAULT '[]',
    data_access_log JSONB DEFAULT '[]',
    -- Integration tracking
    rust_engine_session_id VARCHAR(255),
    go_engine_session_id VARCHAR(255),
    context_gateway_session_id VARCHAR(255),
    fhir_api_request_ids JSONB DEFAULT '[]',
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    environment VARCHAR(50) DEFAULT 'production'
);

-- Clinical calculation results table - stores OUR processing results
CREATE TABLE IF NOT EXISTS clinical_calculation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Calculation identification
    calculation_id VARCHAR(255) NOT NULL UNIQUE,
    workflow_execution_id UUID REFERENCES workflow_executions(id),
    patient_id UUID NOT NULL,
    -- Google FHIR references for input data (not data duplication)
    input_fhir_references JSONB NOT NULL DEFAULT '[]', -- Array of FHIRResourceReference
    -- Calculation specification
    calculation_type VARCHAR(100) NOT NULL, -- dosage_calculation, drug_interaction_check, etc.
    rule_engine_used VARCHAR(100) NOT NULL, -- rust_engine, knowledge_base, flow2_go, etc.
    algorithm_version VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}',
    -- Calculation results (OUR processed data, not FHIR duplication)
    results JSONB NOT NULL DEFAULT '{}',
    confidence_score DECIMAL(4,3) CHECK (confidence_score BETWEEN 0 AND 1),
    risk_assessment JSONB DEFAULT '{}',
    safety_flags JSONB DEFAULT '[]',
    clinical_recommendations JSONB DEFAULT '[]',
    -- Performance metrics
    calculation_started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    calculation_completed_at TIMESTAMP WITH TIME ZONE,
    processing_time_ms INTEGER,
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_mb INTEGER,
    -- Validation and quality
    validation_status VARCHAR(50) DEFAULT 'pending' 
        CHECK (validation_status IN ('pending', 'validated', 'failed', 'requires_review')),
    validation_results JSONB DEFAULT '{}',
    quality_score DECIMAL(4,3),
    evidence_strength VARCHAR(20) CHECK (evidence_strength IN ('low', 'moderate', 'high', 'very_high')),
    -- Clinical context
    clinical_indication VARCHAR(500),
    patient_demographics_hash VARCHAR(64), -- For change detection without storing PHI
    contraindications JSONB DEFAULT '[]',
    drug_interactions JSONB DEFAULT '[]',
    -- Error handling
    calculation_errors JSONB DEFAULT '[]',
    warnings JSONB DEFAULT '[]',
    retry_count INTEGER DEFAULT 0,
    -- Audit and compliance
    calculated_by VARCHAR(255) NOT NULL,
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    audit_trail JSONB DEFAULT '[]',
    -- Data governance
    data_classification VARCHAR(50) DEFAULT 'clinical_calculation',
    retention_period_days INTEGER DEFAULT 2555, -- 7 years
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Create indexes for workflow executions
CREATE INDEX IF NOT EXISTS idx_workflow_executions_patient ON workflow_executions(patient_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(execution_status, current_phase);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_performance ON workflow_executions(performance_target_met, total_execution_time_ms);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_priority ON workflow_executions(priority, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_type ON workflow_executions(workflow_type, execution_status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_quality ON workflow_executions(quality_score DESC, validation_passed);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_review ON workflow_executions(clinical_review_required, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_errors ON workflow_executions(error_count, warnings_count) WHERE error_count > 0;

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_workflow_executions_audit_trail ON workflow_executions USING GIN (audit_trail);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_compliance_flags ON workflow_executions USING GIN (compliance_flags);

-- Create indexes for clinical calculation results
CREATE INDEX IF NOT EXISTS idx_clinical_calc_patient ON clinical_calculation_results(patient_id, calculation_started_at DESC);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_workflow ON clinical_calculation_results(workflow_execution_id);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_type ON clinical_calculation_results(calculation_type, algorithm_version);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_status ON clinical_calculation_results(validation_status, calculation_completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_performance ON clinical_calculation_results(processing_time_ms, confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_quality ON clinical_calculation_results(quality_score DESC, evidence_strength);

-- GIN indexes for JSONB analysis
CREATE INDEX IF NOT EXISTS idx_clinical_calc_results ON clinical_calculation_results USING GIN (results);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_safety_flags ON clinical_calculation_results USING GIN (safety_flags);
CREATE INDEX IF NOT EXISTS idx_clinical_calc_recommendations ON clinical_calculation_results USING GIN (clinical_recommendations);

-- +migrate Down
-- Drop tables and indexes in reverse order
DROP TABLE IF EXISTS clinical_calculation_results CASCADE;
DROP TABLE IF EXISTS workflow_executions CASCADE;