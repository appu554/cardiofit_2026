-- +migrate Up
-- Initial schema for Google FHIR Healthcare API integration
-- This migration sets up the foundation for complementing (not replacing) Google FHIR

-- Enable UUID extension for PostgreSQL
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Google FHIR configuration table
CREATE TABLE IF NOT EXISTS google_fhir_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(255) NOT NULL UNIQUE,
    project_id VARCHAR(255) NOT NULL,
    location VARCHAR(100) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    fhir_store_id VARCHAR(255) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    -- Authentication configuration
    service_account_key_path VARCHAR(500),
    access_token_url VARCHAR(500),
    scope VARCHAR(500) DEFAULT 'https://www.googleapis.com/auth/cloud-healthcare',
    -- Integration settings
    sync_enabled BOOLEAN DEFAULT TRUE,
    batch_size INTEGER DEFAULT 100,
    rate_limit_per_minute INTEGER DEFAULT 1000,
    timeout_seconds INTEGER DEFAULT 30,
    retry_attempts INTEGER DEFAULT 3,
    -- Performance optimization
    enable_caching BOOLEAN DEFAULT TRUE,
    cache_ttl_minutes INTEGER DEFAULT 30,
    -- Compliance and audit
    enable_audit_logging BOOLEAN DEFAULT TRUE,
    data_residency_region VARCHAR(100),
    encryption_key_name VARCHAR(500),
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    environment VARCHAR(50) DEFAULT 'production' CHECK (environment IN ('development', 'staging', 'production'))
);

-- FHIR resource mappings - maps our internal resources to Google FHIR resources
CREATE TABLE IF NOT EXISTS fhir_resource_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Our internal resource identification
    internal_resource_type VARCHAR(100) NOT NULL, -- recipe, snapshot, proposal, workflow
    internal_resource_id UUID NOT NULL,
    -- Google FHIR resource reference (DO NOT duplicate data, only reference)
    fhir_resource_type VARCHAR(100) NOT NULL, -- Patient, Medication, MedicationRequest, etc.
    fhir_resource_id VARCHAR(255) NOT NULL,
    fhir_version_id VARCHAR(255),
    fhir_full_url VARCHAR(1000) NOT NULL,
    fhir_last_updated TIMESTAMP WITH TIME ZONE,
    -- Mapping metadata
    mapping_type VARCHAR(50) NOT NULL CHECK (mapping_type IN ('primary', 'derived', 'referenced', 'composite')),
    mapping_purpose VARCHAR(100) NOT NULL, -- patient_context, medication_data, clinical_context, etc.
    sync_status VARCHAR(50) DEFAULT 'synchronized' CHECK (sync_status IN ('synchronized', 'pending', 'failed', 'stale')),
    -- Data integrity and freshness
    content_hash VARCHAR(64), -- SHA-256 hash for change detection
    last_synchronized_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    synchronization_attempts INTEGER DEFAULT 0,
    last_sync_error TEXT,
    -- Performance optimization
    access_frequency INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    cache_priority INTEGER DEFAULT 5 CHECK (cache_priority BETWEEN 1 AND 10),
    -- Audit and compliance
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    -- Data governance
    data_classification VARCHAR(50) DEFAULT 'clinical' CHECK (data_classification IN ('clinical', 'administrative', 'operational')),
    retention_period_days INTEGER DEFAULT 2555, -- 7 years for HIPAA
    -- Unique constraint to prevent duplicate mappings
    CONSTRAINT unique_internal_fhir_mapping UNIQUE (internal_resource_type, internal_resource_id, fhir_resource_type, fhir_resource_id)
);

-- FHIR synchronization status tracking
CREATE TABLE IF NOT EXISTS fhir_sync_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Sync batch identification
    sync_batch_id VARCHAR(255) NOT NULL,
    sync_type VARCHAR(50) NOT NULL CHECK (sync_type IN ('full_sync', 'incremental_sync', 'resource_sync', 'validation_sync')),
    resource_type VARCHAR(100), -- Patient, Medication, etc. (NULL for full sync)
    -- Sync execution details
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(50) NOT NULL DEFAULT 'in_progress' 
        CHECK (sync_status IN ('in_progress', 'completed', 'failed', 'partial', 'cancelled')),
    -- Performance metrics
    total_resources INTEGER DEFAULT 0,
    processed_resources INTEGER DEFAULT 0,
    successful_resources INTEGER DEFAULT 0,
    failed_resources INTEGER DEFAULT 0,
    skipped_resources INTEGER DEFAULT 0,
    -- Error tracking
    error_summary TEXT,
    detailed_errors JSONB DEFAULT '[]',
    -- Performance data
    processing_time_ms INTEGER,
    average_processing_per_resource_ms DECIMAL(10,2),
    throughput_resources_per_second DECIMAL(10,2),
    -- Google FHIR API usage
    api_requests_made INTEGER DEFAULT 0,
    api_quota_consumed INTEGER DEFAULT 0,
    api_rate_limit_hits INTEGER DEFAULT 0,
    -- Data integrity verification
    content_validation_passed BOOLEAN,
    schema_validation_passed BOOLEAN,
    business_rule_validation_passed BOOLEAN,
    -- Next sync scheduling
    next_sync_scheduled_at TIMESTAMP WITH TIME ZONE,
    sync_frequency_minutes INTEGER DEFAULT 60,
    -- Metadata
    triggered_by VARCHAR(255) NOT NULL,
    configuration_used VARCHAR(255) REFERENCES google_fhir_config(config_name),
    environment VARCHAR(50) DEFAULT 'production'
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_google_fhir_config_active ON google_fhir_config(is_active, environment);
CREATE INDEX IF NOT EXISTS idx_google_fhir_config_project ON google_fhir_config(project_id, dataset_id, fhir_store_id);

CREATE INDEX IF NOT EXISTS idx_fhir_mappings_internal ON fhir_resource_mappings(internal_resource_type, internal_resource_id);
CREATE INDEX IF NOT EXISTS idx_fhir_mappings_fhir ON fhir_resource_mappings(fhir_resource_type, fhir_resource_id);
CREATE INDEX IF NOT EXISTS idx_fhir_mappings_sync_status ON fhir_resource_mappings(sync_status, last_synchronized_at);
CREATE INDEX IF NOT EXISTS idx_fhir_mappings_purpose ON fhir_resource_mappings(mapping_purpose, mapping_type);

CREATE INDEX IF NOT EXISTS idx_fhir_sync_batch ON fhir_sync_status(sync_batch_id);
CREATE INDEX IF NOT EXISTS idx_fhir_sync_status ON fhir_sync_status(sync_status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_fhir_sync_type_resource ON fhir_sync_status(sync_type, resource_type, started_at DESC);

-- +migrate Down
-- Drop tables in reverse order to handle dependencies
DROP TABLE IF EXISTS fhir_sync_status CASCADE;
DROP TABLE IF EXISTS fhir_resource_mappings CASCADE;
DROP TABLE IF EXISTS google_fhir_config CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp";