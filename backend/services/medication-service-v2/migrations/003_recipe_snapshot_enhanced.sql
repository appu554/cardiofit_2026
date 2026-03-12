-- +migrate Up
-- Enhanced Recipe & Snapshot architecture with Google FHIR integration
-- This migration updates the core Recipe & Snapshot tables with FHIR integration

-- Enhanced recipes table with Google FHIR integration
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    protocol_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    indication VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('draft', 'review', 'approved', 'active', 'deprecated', 'archived')),
    -- Context requirements and rules (our internal processing logic)
    context_requirements JSONB NOT NULL DEFAULT '{}',
    calculation_rules JSONB NOT NULL DEFAULT '[]',
    safety_rules JSONB NOT NULL DEFAULT '[]',
    monitoring_rules JSONB NOT NULL DEFAULT '[]',
    ttl_hours INTEGER DEFAULT 24,
    clinical_evidence JSONB,
    approval_metadata JSONB,
    -- Google FHIR integration (references, not data duplication)
    fhir_protocol_reference JSONB, -- Reference to PlanDefinition or other FHIR resources
    fhir_evidence_references JSONB DEFAULT '[]', -- Array of FHIRResourceReference for evidence
    fhir_medication_references JSONB DEFAULT '[]', -- Array of medication references
    -- Recipe metadata and optimization
    complexity_score DECIMAL(3,2) DEFAULT 0.0,
    cache_priority INTEGER DEFAULT 5,
    average_execution_ms INTEGER DEFAULT 0,
    estimated_cost DECIMAL(10,2), -- Cost estimation for resource planning
    -- Clinical validation and quality
    clinical_validation_status VARCHAR(50) DEFAULT 'pending' 
        CHECK (clinical_validation_status IN ('pending', 'validated', 'requires_review', 'failed')),
    evidence_quality_score DECIMAL(4,3), -- 0.000 to 1.000
    peer_review_required BOOLEAN DEFAULT TRUE,
    regulatory_approval_required BOOLEAN DEFAULT FALSE,
    -- Audit and compliance
    last_validated_at TIMESTAMP WITH TIME ZONE,
    validation_checksum VARCHAR(64),
    compliance_flags JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    -- Version control with enhanced tracking
    parent_recipe_id UUID REFERENCES recipes(id),
    version_sequence INTEGER DEFAULT 1,
    version_notes TEXT,
    -- Performance and usage tracking
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,4) DEFAULT 0.0000, -- Success rate as decimal
    last_used_at TIMESTAMP WITH TIME ZONE,
    -- Data governance
    data_classification VARCHAR(50) DEFAULT 'clinical_protocol',
    retention_period_days INTEGER DEFAULT 2555, -- 7 years
    -- Unique constraints
    CONSTRAINT recipes_protocol_version_unique UNIQUE (protocol_id, version)
);

-- Enhanced clinical snapshots with Google FHIR integration
CREATE TABLE IF NOT EXISTS clinical_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL,
    recipe_id UUID NOT NULL REFERENCES recipes(id),
    workflow_execution_id UUID REFERENCES workflow_executions(id),
    snapshot_type VARCHAR(50) NOT NULL CHECK (snapshot_type IN ('calculation', 'validation', 'commit', 'monitoring', 'emergency', 'context_assembly')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'active', 'superseded', 'expired', 'invalid')),
    version INTEGER NOT NULL DEFAULT 1,
    -- Google FHIR references (not data duplication - references only)
    patient_fhir_reference JSONB NOT NULL, -- FHIRResourceReference
    clinical_context_fhir_references JSONB DEFAULT '[]', -- Array of FHIRResourceReference
    source_fhir_bundle_reference JSONB, -- Reference to source FHIR Bundle if applicable
    -- Core snapshot operational data (OUR processing results)
    clinical_data JSONB NOT NULL DEFAULT '{}', -- Our processed clinical context
    freshness_metadata JSONB NOT NULL DEFAULT '{}', -- Data age and validity information
    validation_results JSONB NOT NULL DEFAULT '{}', -- Our validation results
    processing_results JSONB DEFAULT '{}', -- Results of our calculations and analysis
    -- Cryptographic integrity (Recipe & Snapshot Architecture requirement)
    content_hash VARCHAR(64) NOT NULL, -- SHA-256 of clinical_data
    integrity_signature VARCHAR(512), -- Digital signature for non-repudiation
    signature_method VARCHAR(50) DEFAULT 'sha256_hmac',
    signature_key_id VARCHAR(255), -- Reference to signing key
    -- Lifecycle management
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    -- Snapshot chaining for audit trail and versioning
    previous_snapshot_id UUID REFERENCES clinical_snapshots(id),
    change_reason TEXT,
    change_type VARCHAR(50) CHECK (change_type IN ('update', 'correction', 'supersede', 'expire')),
    -- Performance and quality tracking
    assembly_duration_ms INTEGER DEFAULT 0,
    completeness_score DECIMAL(4,3) DEFAULT 0.000, -- 0.000 to 1.000
    quality_score DECIMAL(4,3) DEFAULT 0.000,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    -- Evidence envelope for clinical safety and decision support
    evidence_envelope JSONB DEFAULT '{}',
    clinical_reasoning JSONB DEFAULT '{}', -- Our clinical reasoning logic
    safety_assessment JSONB DEFAULT '{}', -- Safety evaluation results
    -- FHIR integration metadata (not full resources)
    fhir_narrative TEXT, -- Human-readable narrative for FHIR compliance
    fhir_composition_reference JSONB, -- Reference to created FHIR Composition if needed
    -- Audit trail and compliance
    audit_trail JSONB DEFAULT '[]',
    compliance_flags JSONB DEFAULT '[]',
    hipaa_tracking JSONB DEFAULT '{}', -- HIPAA access tracking
    -- Data governance
    data_classification VARCHAR(50) DEFAULT 'clinical_snapshot',
    retention_period_days INTEGER DEFAULT 2555, -- 7 years
    -- Metadata
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    environment VARCHAR(50) DEFAULT 'production'
);

-- Create performance indexes for recipes
CREATE INDEX IF NOT EXISTS idx_recipes_protocol_id ON recipes(protocol_id);
CREATE INDEX IF NOT EXISTS idx_recipes_status ON recipes(status, clinical_validation_status);
CREATE INDEX IF NOT EXISTS idx_recipes_indication ON recipes(indication);
CREATE INDEX IF NOT EXISTS idx_recipes_created_at ON recipes(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recipes_complexity ON recipes(complexity_score, cache_priority);
CREATE INDEX IF NOT EXISTS idx_recipes_version_sequence ON recipes(protocol_id, version_sequence DESC);
CREATE INDEX IF NOT EXISTS idx_recipes_usage ON recipes(usage_count DESC, success_rate DESC);
CREATE INDEX IF NOT EXISTS idx_recipes_quality ON recipes(evidence_quality_score DESC, clinical_validation_status);
CREATE INDEX IF NOT EXISTS idx_recipes_performance ON recipes(average_execution_ms, estimated_cost);

-- GIN indexes for JSONB columns in recipes
CREATE INDEX IF NOT EXISTS idx_recipes_context_requirements ON recipes USING GIN (context_requirements);
CREATE INDEX IF NOT EXISTS idx_recipes_calculation_rules ON recipes USING GIN (calculation_rules);
CREATE INDEX IF NOT EXISTS idx_recipes_safety_rules ON recipes USING GIN (safety_rules);
CREATE INDEX IF NOT EXISTS idx_recipes_fhir_references ON recipes USING GIN (fhir_medication_references);
CREATE INDEX IF NOT EXISTS idx_recipes_compliance_flags ON recipes USING GIN (compliance_flags);

-- Create indexes for clinical snapshots
-- TTL index for automatic cleanup (Recipe & Snapshot Architecture requirement)
CREATE INDEX IF NOT EXISTS idx_snapshots_ttl ON clinical_snapshots(expires_at);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_snapshots_patient_created ON clinical_snapshots(patient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_recipe_status ON clinical_snapshots(recipe_id, status);
CREATE INDEX IF NOT EXISTS idx_snapshots_type_status ON clinical_snapshots(snapshot_type, status);
CREATE INDEX IF NOT EXISTS idx_snapshots_workflow ON clinical_snapshots(workflow_execution_id) WHERE workflow_execution_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_snapshots_hash ON clinical_snapshots(content_hash);
CREATE INDEX IF NOT EXISTS idx_snapshots_access ON clinical_snapshots(access_count DESC, last_accessed_at DESC);

-- Chain tracking index
CREATE INDEX IF NOT EXISTS idx_snapshots_chain ON clinical_snapshots(previous_snapshot_id);

-- Quality and performance indexes
CREATE INDEX IF NOT EXISTS idx_snapshots_quality ON clinical_snapshots(quality_score DESC, completeness_score DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_performance ON clinical_snapshots(assembly_duration_ms, access_count DESC);

-- GIN indexes for JSONB columns in snapshots
CREATE INDEX IF NOT EXISTS idx_snapshots_clinical_data ON clinical_snapshots USING GIN (clinical_data);
CREATE INDEX IF NOT EXISTS idx_snapshots_validation_results ON clinical_snapshots USING GIN (validation_results);
CREATE INDEX IF NOT EXISTS idx_snapshots_evidence_envelope ON clinical_snapshots USING GIN (evidence_envelope);
CREATE INDEX IF NOT EXISTS idx_snapshots_clinical_reasoning ON clinical_snapshots USING GIN (clinical_reasoning);
CREATE INDEX IF NOT EXISTS idx_snapshots_safety_assessment ON clinical_snapshots USING GIN (safety_assessment);
CREATE INDEX IF NOT EXISTS idx_snapshots_fhir_context ON clinical_snapshots USING GIN (clinical_context_fhir_references);

-- +migrate Down
-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS clinical_snapshots CASCADE;
DROP TABLE IF EXISTS recipes CASCADE;