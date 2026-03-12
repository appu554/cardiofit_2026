package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"database/sql/driver"

	"go.uber.org/zap"
)

// GoogleFHIRIntegration handles FHIR resource references and metadata
type GoogleFHIRIntegration struct {
	ProjectID    string `json:"project_id"`
	Location     string `json:"location"`
	DatasetID    string `json:"dataset_id"`
	FHIRStoreID  string `json:"fhir_store_id"`
}

// FHIRResourceReference represents a reference to a Google FHIR resource
type FHIRResourceReference struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	VersionID    string `json:"version_id,omitempty"`
	FullURL      string `json:"full_url"`
	LastUpdated  *time.Time `json:"last_updated,omitempty"`
}

// Value implements the driver.Valuer interface for database storage
func (f FHIRResourceReference) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements the sql.Scanner interface for database retrieval
func (f *FHIRResourceReference) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, f)
	case string:
		return json.Unmarshal([]byte(v), f)
	default:
		return fmt.Errorf("cannot scan %T into FHIRResourceReference", value)
	}
}

// SchemaManager manages database schema operations with Google FHIR integration
type SchemaManager struct {
	db           *PostgreSQL
	logger       *zap.Logger
	googleFHIR   GoogleFHIRIntegration
	migrationMgr *MigrationManager
}

// NewSchemaManager creates a new schema manager with Google FHIR integration
func NewSchemaManager(db *PostgreSQL, logger *zap.Logger, googleFHIR GoogleFHIRIntegration) *SchemaManager {
	migrationMgr := NewMigrationManager(db, logger)
	return &SchemaManager{
		db:           db,
		logger:       logger,
		googleFHIR:   googleFHIR,
		migrationMgr: migrationMgr,
	}
}

// CreateAllTables creates all required tables for the Recipe & Snapshot architecture with Google FHIR integration
func (sm *SchemaManager) CreateAllTables(ctx context.Context) error {
	// Initialize migration manager first
	if err := sm.migrationMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration manager: %w", err)
	}

	// Create Google FHIR integration tables first
	integrationTables := []func(context.Context) error{
		sm.CreateGoogleFHIRConfigTable,
		sm.CreateFHIRResourceMappingsTable,
		sm.CreateFHIRSyncStatusTable,
	}

	// Core Recipe & Snapshot tables
	coreTables := []func(context.Context) error{
		sm.CreateRecipesTable,
		sm.CreateClinicalSnapshotsTable,
		sm.CreateMedicationProposalsTable,
		sm.CreateWorkflowStatesTable,
	}

	// Operational and compliance tables
	operationalTables := []func(context.Context) error{
		sm.CreateWorkflowExecutionsTable,
		sm.CreateClinicalCalculationResultsTable,
		sm.CreateMedicationProposalWorkflowsTable,
		sm.CreateAuditTrailTable,
		sm.CreateCacheableDataTable,
		sm.CreatePerformanceMetricsTable,
		sm.CreateSecurityEventsTable,
		sm.CreateFHIRIntegrationLogsTable,
	}

	// Execute in order: integration → core → operational
	allTables := append(append(integrationTables, coreTables...), operationalTables...)

	for _, createTable := range allTables {
		if err := createTable(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes and functions
	if err := sm.CreateIndexes(ctx); err != nil {
		return err
	}

	if err := sm.CreateFunctions(ctx); err != nil {
		return err
	}

	return sm.CreateTriggers(ctx)
}

// CreateGoogleFHIRConfigTable creates the configuration table for Google FHIR integration
func (sm *SchemaManager) CreateGoogleFHIRConfigTable(ctx context.Context) error {
	query := `
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
		
		-- Indexes for Google FHIR config
		CREATE INDEX IF NOT EXISTS idx_google_fhir_config_active ON google_fhir_config(is_active, environment);
		CREATE INDEX IF NOT EXISTS idx_google_fhir_config_project ON google_fhir_config(project_id, dataset_id, fhir_store_id);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create google_fhir_config table: %w", err)
	}

	// Insert default configuration if not exists
	defaultConfig := `
		INSERT INTO google_fhir_config (config_name, project_id, location, dataset_id, fhir_store_id, base_url, created_by)
		VALUES ('default', $1, $2, $3, $4, $5, 'system')
		ON CONFLICT (config_name) DO NOTHING
	`
	
	baseURL := fmt.Sprintf("https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		sm.googleFHIR.ProjectID, sm.googleFHIR.Location, sm.googleFHIR.DatasetID, sm.googleFHIR.FHIRStoreID)
	
	_, err = sm.db.DB.ExecContext(ctx, defaultConfig, 
		sm.googleFHIR.ProjectID, sm.googleFHIR.Location, 
		sm.googleFHIR.DatasetID, sm.googleFHIR.FHIRStoreID, baseURL)
	if err != nil {
		sm.logger.Warn("Failed to insert default Google FHIR config", zap.Error(err))
	}

	sm.logger.Info("Created google_fhir_config table successfully")
	return nil
}

// CreateFHIRResourceMappingsTable creates the table for mapping our resources to Google FHIR resources
func (sm *SchemaManager) CreateFHIRResourceMappingsTable(ctx context.Context) error {
	query := `
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
		
		-- Performance indexes for FHIR resource mappings
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_internal ON fhir_resource_mappings(internal_resource_type, internal_resource_id);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_fhir ON fhir_resource_mappings(fhir_resource_type, fhir_resource_id);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_sync_status ON fhir_resource_mappings(sync_status, last_synchronized_at);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_purpose ON fhir_resource_mappings(mapping_purpose, mapping_type);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_freshness ON fhir_resource_mappings(fhir_last_updated DESC, sync_status);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_access ON fhir_resource_mappings(access_frequency DESC, last_accessed_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_hash ON fhir_resource_mappings(content_hash) WHERE content_hash IS NOT NULL;
		-- Cleanup index for expired mappings
		CREATE INDEX IF NOT EXISTS idx_fhir_mappings_retention ON fhir_resource_mappings(created_at, retention_period_days) 
			WHERE created_at + (retention_period_days * INTERVAL '1 day') < NOW();
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create fhir_resource_mappings table: %w", err)
	}

	sm.logger.Info("Created fhir_resource_mappings table successfully")
	return nil
}

// CreateFHIRSyncStatusTable creates the table for tracking synchronization status with Google FHIR
func (sm *SchemaManager) CreateFHIRSyncStatusTable(ctx context.Context) error {
	query := `
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
		
		-- Indexes for FHIR sync status tracking
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_batch ON fhir_sync_status(sync_batch_id);
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_status ON fhir_sync_status(sync_status, started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_type_resource ON fhir_sync_status(sync_type, resource_type, started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_completed ON fhir_sync_status(completed_at DESC) WHERE completed_at IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_performance ON fhir_sync_status(throughput_resources_per_second DESC, processing_time_ms);
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_next_scheduled ON fhir_sync_status(next_sync_scheduled_at) WHERE next_sync_scheduled_at IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_fhir_sync_errors ON fhir_sync_status(sync_status, started_at DESC) WHERE sync_status IN ('failed', 'partial');
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create fhir_sync_status table: %w", err)
	}

	sm.logger.Info("Created fhir_sync_status table successfully")
	return nil
}

// CreateWorkflowExecutionsTable creates the table for 4-Phase Workflow execution tracking
func (sm *SchemaManager) CreateWorkflowExecutionsTable(ctx context.Context) error {
	query := `
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
			recipe_id UUID REFERENCES recipes(id),
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
		
		-- Performance indexes for workflow executions
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_patient ON workflow_executions(patient_id, started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(execution_status, current_phase);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_performance ON workflow_executions(performance_target_met, total_execution_time_ms);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_priority ON workflow_executions(priority, started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_recipe ON workflow_executions(recipe_id, execution_status);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_type ON workflow_executions(workflow_type, execution_status, started_at DESC);
		-- Quality and compliance indexes
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_quality ON workflow_executions(quality_score DESC, validation_passed);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_review ON workflow_executions(clinical_review_required, completed_at DESC);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_errors ON workflow_executions(error_count, warnings_count) WHERE error_count > 0;
		-- GIN indexes for JSONB columns
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_audit_trail ON workflow_executions USING GIN (audit_trail);
		CREATE INDEX IF NOT EXISTS idx_workflow_executions_compliance_flags ON workflow_executions USING GIN (compliance_flags);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create workflow_executions table: %w", err)
	}

	sm.logger.Info("Created workflow_executions table successfully")
	return nil
}

// CreateClinicalCalculationResultsTable creates the table for storing our clinical calculation results
func (sm *SchemaManager) CreateClinicalCalculationResultsTable(ctx context.Context) error {
	query := `
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
		
		-- Performance indexes for clinical calculation results
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_patient ON clinical_calculation_results(patient_id, calculation_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_workflow ON clinical_calculation_results(workflow_execution_id);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_type ON clinical_calculation_results(calculation_type, algorithm_version);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_status ON clinical_calculation_results(validation_status, calculation_completed_at DESC);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_performance ON clinical_calculation_results(processing_time_ms, confidence_score DESC);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_quality ON clinical_calculation_results(quality_score DESC, evidence_strength);
		-- Clinical analysis indexes
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_indication ON clinical_calculation_results(clinical_indication);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_demographics_hash ON clinical_calculation_results(patient_demographics_hash, calculation_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_review ON clinical_calculation_results(reviewed_by, reviewed_at) WHERE reviewed_by IS NOT NULL;
		-- GIN indexes for JSONB analysis
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_results ON clinical_calculation_results USING GIN (results);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_safety_flags ON clinical_calculation_results USING GIN (safety_flags);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_recommendations ON clinical_calculation_results USING GIN (clinical_recommendations);
		CREATE INDEX IF NOT EXISTS idx_clinical_calc_contraindications ON clinical_calculation_results USING GIN (contraindications);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create clinical_calculation_results table: %w", err)
	}

	sm.logger.Info("Created clinical_calculation_results table successfully")
	return nil
}

// CreateMedicationProposalWorkflowsTable creates the table for medication proposal workflow state
func (sm *SchemaManager) CreateMedicationProposalWorkflowsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS medication_proposal_workflows (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			-- Workflow identification
			proposal_workflow_id VARCHAR(255) NOT NULL UNIQUE,
			workflow_execution_id UUID REFERENCES workflow_executions(id),
			patient_id UUID NOT NULL,
			-- Google FHIR references (not data duplication)
			patient_fhir_reference JSONB NOT NULL,
			medication_fhir_references JSONB DEFAULT '[]', -- Array of medication references
			-- Proposal details 
			proposal_type VARCHAR(100) NOT NULL CHECK (proposal_type IN ('new_prescription', 'dose_adjustment', 'medication_switch', 'discontinuation', 'alternative_therapy')),
			clinical_indication VARCHAR(500) NOT NULL,
			urgency_level VARCHAR(20) DEFAULT 'routine' CHECK (urgency_level IN ('routine', 'urgent', 'emergent')),
			-- Workflow state management
			current_state VARCHAR(50) NOT NULL DEFAULT 'draft' 
				CHECK (current_state IN ('draft', 'clinical_review', 'safety_review', 'pharmacist_review', 'physician_approval', 'patient_consent', 'ready_for_execution', 'executed', 'cancelled', 'expired')),
			state_history JSONB DEFAULT '[]', -- Array of state transitions with timestamps
			-- Generated proposals (OUR analysis results, not FHIR duplication)
			proposed_medications JSONB NOT NULL DEFAULT '[]',
			dosing_recommendations JSONB DEFAULT '[]',
			administration_instructions JSONB DEFAULT '{}',
			monitoring_requirements JSONB DEFAULT '[]',
			-- Clinical decision support
			clinical_reasoning JSONB DEFAULT '{}',
			evidence_summary JSONB DEFAULT '{}',
			risk_benefit_analysis JSONB DEFAULT '{}',
			alternative_options JSONB DEFAULT '[]',
			-- Safety and quality assurance
			safety_review_status VARCHAR(50) DEFAULT 'pending' 
				CHECK (safety_review_status IN ('pending', 'approved', 'flagged', 'rejected')),
			safety_concerns JSONB DEFAULT '[]',
			drug_interactions_checked BOOLEAN DEFAULT FALSE,
			allergy_screening_passed BOOLEAN DEFAULT FALSE,
			contraindication_screening_passed BOOLEAN DEFAULT FALSE,
			-- Approval workflow
			clinical_reviewer VARCHAR(255),
			clinical_review_timestamp TIMESTAMP WITH TIME ZONE,
			clinical_review_notes TEXT,
			safety_reviewer VARCHAR(255),
			safety_review_timestamp TIMESTAMP WITH TIME ZONE,
			safety_review_notes TEXT,
			final_approver VARCHAR(255),
			final_approval_timestamp TIMESTAMP WITH TIME ZONE,
			-- Patient engagement
			patient_consent_required BOOLEAN DEFAULT TRUE,
			patient_consent_status VARCHAR(50) DEFAULT 'pending' 
				CHECK (patient_consent_status IN ('pending', 'obtained', 'declined', 'not_required')),
			patient_education_provided BOOLEAN DEFAULT FALSE,
			-- Performance tracking
			workflow_started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			workflow_completed_at TIMESTAMP WITH TIME ZONE,
			total_processing_time_ms INTEGER,
			state_transition_count INTEGER DEFAULT 0,
			-- Quality metrics
			proposal_quality_score DECIMAL(4,3),
			clinical_appropriateness_score DECIMAL(4,3),
			safety_score DECIMAL(4,3),
			patient_preference_alignment DECIMAL(4,3),
			-- Integration and execution
			fhir_medication_request_id VARCHAR(255), -- Reference to created MedicationRequest
			execution_system VARCHAR(100), -- EHR system, pharmacy system, etc.
			execution_timestamp TIMESTAMP WITH TIME ZONE,
			execution_confirmation VARCHAR(255),
			-- Audit and compliance
			created_by VARCHAR(255) NOT NULL,
			audit_trail JSONB DEFAULT '[]',
			compliance_checked BOOLEAN DEFAULT FALSE,
			regulatory_flags JSONB DEFAULT '[]',
			-- Metadata
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '7 days'),
			version INTEGER DEFAULT 1
		);
		
		-- Indexes for medication proposal workflows
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_patient ON medication_proposal_workflows(patient_id, workflow_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_execution ON medication_proposal_workflows(workflow_execution_id);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_state ON medication_proposal_workflows(current_state, urgency_level);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_type ON medication_proposal_workflows(proposal_type, clinical_indication);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_review_status ON medication_proposal_workflows(safety_review_status, clinical_reviewer);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_consent ON medication_proposal_workflows(patient_consent_status, patient_consent_required);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_approval ON medication_proposal_workflows(final_approver, final_approval_timestamp);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_quality ON medication_proposal_workflows(proposal_quality_score DESC, safety_score DESC);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_execution ON medication_proposal_workflows(fhir_medication_request_id, execution_timestamp);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_expires ON medication_proposal_workflows(expires_at) WHERE current_state NOT IN ('executed', 'cancelled', 'expired');
		-- GIN indexes for JSONB columns
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_medications ON medication_proposal_workflows USING GIN (proposed_medications);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_safety_concerns ON medication_proposal_workflows USING GIN (safety_concerns);
		CREATE INDEX IF NOT EXISTS idx_med_proposal_workflows_alternatives ON medication_proposal_workflows USING GIN (alternative_options);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create medication_proposal_workflows table: %w", err)
	}

	sm.logger.Info("Created medication_proposal_workflows table successfully")
	return nil
}

// CreateFHIRIntegrationLogsTable creates the table for detailed Google FHIR integration logging
func (sm *SchemaManager) CreateFHIRIntegrationLogsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS fhir_integration_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			-- Log event identification
			log_event_id VARCHAR(255) NOT NULL,
			correlation_id VARCHAR(255), -- For tracing across multiple operations
			-- FHIR operation details
			operation_type VARCHAR(50) NOT NULL CHECK (operation_type IN ('read', 'search', 'create', 'update', 'delete', 'batch', 'transaction')),
			fhir_resource_type VARCHAR(100),
			fhir_resource_id VARCHAR(255),
			fhir_endpoint VARCHAR(500) NOT NULL,
			-- HTTP request/response details
			http_method VARCHAR(10) NOT NULL,
			http_status_code INTEGER,
			request_headers JSONB DEFAULT '{}',
			response_headers JSONB DEFAULT '{}',
			request_body_size_bytes INTEGER DEFAULT 0,
			response_body_size_bytes INTEGER DEFAULT 0,
			-- Performance metrics
			request_started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			response_received_at TIMESTAMP WITH TIME ZONE,
			total_latency_ms INTEGER,
			dns_resolution_ms INTEGER,
			tcp_connection_ms INTEGER,
			tls_handshake_ms INTEGER,
			time_to_first_byte_ms INTEGER,
			-- Authentication and authorization
			auth_method VARCHAR(50), -- oauth2, service_account, api_key
			token_used_hash VARCHAR(64), -- Hash of token for audit without exposing
			scopes_used JSONB DEFAULT '[]',
			-- Error tracking
			success BOOLEAN NOT NULL DEFAULT FALSE,
			error_code VARCHAR(100),
			error_message TEXT,
			fhir_operation_outcome JSONB, -- FHIR OperationOutcome resource
			retry_attempt INTEGER DEFAULT 0,
			-- Data governance
			data_accessed_classification VARCHAR(50), -- clinical, administrative, operational
			patient_id_accessed UUID, -- If patient data was accessed
			data_sensitivity_level VARCHAR(20) DEFAULT 'high' CHECK (data_sensitivity_level IN ('low', 'medium', 'high', 'critical')),
			-- Rate limiting and quotas
			rate_limit_remaining INTEGER,
			quota_used INTEGER,
			quota_limit INTEGER,
			quota_reset_at TIMESTAMP WITH TIME ZONE,
			-- Integration context
			triggered_by_service VARCHAR(100), -- medication-service-v2, etc.
			triggered_by_operation VARCHAR(100), -- workflow_execution, sync_patient_data, etc.
			workflow_execution_id UUID,
			-- Compliance and audit
			hipaa_compliant BOOLEAN DEFAULT TRUE,
			audit_required BOOLEAN DEFAULT TRUE,
			data_retention_days INTEGER DEFAULT 2555, -- 7 years
			-- Environment and configuration
			environment VARCHAR(50) DEFAULT 'production',
			fhir_config_used VARCHAR(255),
			api_version VARCHAR(20),
			-- Metadata
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		
		-- Performance indexes for FHIR integration logs
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_timestamp ON fhir_integration_logs(request_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_correlation ON fhir_integration_logs(correlation_id, request_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_operation ON fhir_integration_logs(operation_type, fhir_resource_type, request_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_status ON fhir_integration_logs(http_status_code, success);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_performance ON fhir_integration_logs(total_latency_ms DESC, request_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_errors ON fhir_integration_logs(success, error_code, request_started_at DESC) WHERE success = FALSE;
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_patient ON fhir_integration_logs(patient_id_accessed, request_started_at DESC) WHERE patient_id_accessed IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_workflow ON fhir_integration_logs(workflow_execution_id, request_started_at DESC) WHERE workflow_execution_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_service ON fhir_integration_logs(triggered_by_service, triggered_by_operation);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_rate_limit ON fhir_integration_logs(rate_limit_remaining, quota_used, request_started_at DESC);
		-- Compliance indexes
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_audit ON fhir_integration_logs(audit_required, hipaa_compliant, request_started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_sensitivity ON fhir_integration_logs(data_sensitivity_level, data_accessed_classification);
		-- Cleanup index for retention
		CREATE INDEX IF NOT EXISTS idx_fhir_logs_retention ON fhir_integration_logs(created_at, data_retention_days) 
			WHERE created_at + (data_retention_days * INTERVAL '1 day') < NOW();
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create fhir_integration_logs table: %w", err)
	}

	sm.logger.Info("Created fhir_integration_logs table successfully")
	return nil
}

// CreateRecipesTable creates the recipes table with FHIR compliance and Google FHIR integration
func (sm *SchemaManager) CreateRecipesTable(ctx context.Context) error {
	query := `
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
		
		-- Performance indexes for recipes
		CREATE INDEX IF NOT EXISTS idx_recipes_protocol_id ON recipes(protocol_id);
		CREATE INDEX IF NOT EXISTS idx_recipes_status ON recipes(status, clinical_validation_status);
		CREATE INDEX IF NOT EXISTS idx_recipes_indication ON recipes(indication);
		CREATE INDEX IF NOT EXISTS idx_recipes_created_at ON recipes(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_recipes_complexity ON recipes(complexity_score, cache_priority);
		CREATE INDEX IF NOT EXISTS idx_recipes_version_sequence ON recipes(protocol_id, version_sequence DESC);
		CREATE INDEX IF NOT EXISTS idx_recipes_usage ON recipes(usage_count DESC, success_rate DESC);
		CREATE INDEX IF NOT EXISTS idx_recipes_quality ON recipes(evidence_quality_score DESC, clinical_validation_status);
		CREATE INDEX IF NOT EXISTS idx_recipes_performance ON recipes(average_execution_ms, estimated_cost);
		-- GIN indexes for JSONB columns for fast queries
		CREATE INDEX IF NOT EXISTS idx_recipes_context_requirements ON recipes USING GIN (context_requirements);
		CREATE INDEX IF NOT EXISTS idx_recipes_calculation_rules ON recipes USING GIN (calculation_rules);
		CREATE INDEX IF NOT EXISTS idx_recipes_safety_rules ON recipes USING GIN (safety_rules);
		CREATE INDEX IF NOT EXISTS idx_recipes_fhir_references ON recipes USING GIN (fhir_medication_references);
		CREATE INDEX IF NOT EXISTS idx_recipes_compliance_flags ON recipes USING GIN (compliance_flags);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create recipes table: %w", err)
	}

	sm.logger.Info("Created recipes table successfully")
	return nil
}

// CreateClinicalSnapshotsTable creates the clinical snapshots table for Recipe & Snapshot architecture with Google FHIR integration
func (sm *SchemaManager) CreateClinicalSnapshotsTable(ctx context.Context) error {
	query := `
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
		
		-- TTL index for automatic cleanup (Recipe & Snapshot Architecture requirement)
		CREATE INDEX IF NOT EXISTS idx_snapshots_ttl ON clinical_snapshots(expires_at);
		
		-- Performance indexes
		CREATE INDEX IF NOT EXISTS idx_snapshots_patient_created ON clinical_snapshots(patient_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_snapshots_recipe_status ON clinical_snapshots(recipe_id, status);
		CREATE INDEX IF NOT EXISTS idx_snapshots_type_status ON clinical_snapshots(snapshot_type, status);
		CREATE INDEX IF NOT EXISTS idx_snapshots_hash ON clinical_snapshots(hash);
		CREATE INDEX IF NOT EXISTS idx_snapshots_access ON clinical_snapshots(access_count DESC, last_accessed_at DESC);
		
		-- Chain tracking index
		CREATE INDEX IF NOT EXISTS idx_snapshots_chain ON clinical_snapshots(previous_snapshot_id);
		
		-- GIN indexes for JSONB columns
		CREATE INDEX IF NOT EXISTS idx_snapshots_clinical_data ON clinical_snapshots USING GIN (clinical_data);
		CREATE INDEX IF NOT EXISTS idx_snapshots_validation_results ON clinical_snapshots USING GIN (validation_results);
		CREATE INDEX IF NOT EXISTS idx_snapshots_evidence_envelope ON clinical_snapshots USING GIN (evidence_envelope);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create clinical_snapshots table: %w", err)
	}

	sm.logger.Info("Created clinical_snapshots table successfully")
	return nil
}

// CreateMedicationProposalsTable creates the medication proposals table
func (sm *SchemaManager) CreateMedicationProposalsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS medication_proposals (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			patient_id UUID NOT NULL,
			protocol_id VARCHAR(255) NOT NULL,
			indication VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL CHECK (status IN ('draft', 'proposed', 'validated', 'rejected', 'committed', 'expired')),
			-- Linked snapshot for Recipe & Snapshot Architecture
			snapshot_id UUID REFERENCES clinical_snapshots(id),
			-- Clinical context and medication details
			clinical_context JSONB NOT NULL DEFAULT '{}',
			medication_details JSONB NOT NULL DEFAULT '{}',
			dosage_recommendations JSONB NOT NULL DEFAULT '[]',
			safety_constraints JSONB NOT NULL DEFAULT '[]',
			-- Calculation results from Rust Engine
			calculation_results JSONB DEFAULT '{}',
			confidence_score DECIMAL(3,2) DEFAULT 0.0,
			processing_time_ms INTEGER DEFAULT 0,
			-- Lifecycle
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '24 hours'),
			created_by VARCHAR(255) NOT NULL,
			validated_by VARCHAR(255),
			validation_timestamp TIMESTAMP WITH TIME ZONE,
			-- FHIR compliance
			fhir_medication_request JSONB,
			fhir_dosage_instruction JSONB,
			-- Decision support metadata
			originating_workflow VARCHAR(100),
			rule_engine_version VARCHAR(50),
			knowledge_base_version VARCHAR(50),
			-- Audit and safety
			safety_review_required BOOLEAN DEFAULT FALSE,
			safety_reviewer VARCHAR(255),
			safety_review_timestamp TIMESTAMP WITH TIME ZONE,
			override_reasons JSONB DEFAULT '[]'
		);
		
		-- Performance indexes
		CREATE INDEX IF NOT EXISTS idx_proposals_patient_created ON medication_proposals(patient_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_proposals_status ON medication_proposals(status);
		CREATE INDEX IF NOT EXISTS idx_proposals_snapshot ON medication_proposals(snapshot_id);
		CREATE INDEX IF NOT EXISTS idx_proposals_protocol ON medication_proposals(protocol_id);
		CREATE INDEX IF NOT EXISTS idx_proposals_expires ON medication_proposals(expires_at);
		CREATE INDEX IF NOT EXISTS idx_proposals_confidence ON medication_proposals(confidence_score DESC);
		
		-- Safety and review indexes
		CREATE INDEX IF NOT EXISTS idx_proposals_safety_review ON medication_proposals(safety_review_required, safety_review_timestamp);
		CREATE INDEX IF NOT EXISTS idx_proposals_validation ON medication_proposals(validated_by, validation_timestamp);
		
		-- GIN indexes for JSONB search
		CREATE INDEX IF NOT EXISTS idx_proposals_clinical_context ON medication_proposals USING GIN (clinical_context);
		CREATE INDEX IF NOT EXISTS idx_proposals_dosage_recommendations ON medication_proposals USING GIN (dosage_recommendations);
		CREATE INDEX IF NOT EXISTS idx_proposals_safety_constraints ON medication_proposals USING GIN (safety_constraints);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create medication_proposals table: %w", err)
	}

	sm.logger.Info("Created medication_proposals table successfully")
	return nil
}

// CreateWorkflowStatesTable creates the workflow states table for 4-Phase Workflow
func (sm *SchemaManager) CreateWorkflowStatesTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS workflow_states (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			workflow_id VARCHAR(255) NOT NULL,
			patient_id UUID NOT NULL,
			proposal_id UUID REFERENCES medication_proposals(id),
			snapshot_id UUID REFERENCES clinical_snapshots(id),
			-- 4-Phase Workflow state management
			current_phase VARCHAR(50) NOT NULL CHECK (current_phase IN ('recipe_resolution', 'context_assembly', 'calculation_execution', 'validation_commit')),
			phase_status VARCHAR(50) NOT NULL CHECK (phase_status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped')),
			overall_status VARCHAR(50) NOT NULL CHECK (overall_status IN ('initializing', 'processing', 'completed', 'failed', 'cancelled')),
			-- Phase execution tracking
			phase_1_recipe_resolution JSONB DEFAULT '{}',
			phase_2_context_assembly JSONB DEFAULT '{}',
			phase_3_calculation_execution JSONB DEFAULT '{}',
			phase_4_validation_commit JSONB DEFAULT '{}',
			-- Timing and performance
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE,
			total_processing_ms INTEGER DEFAULT 0,
			phase_timings JSONB DEFAULT '{}',
			-- Error handling
			error_details JSONB DEFAULT '{}',
			retry_count INTEGER DEFAULT 0,
			last_retry_at TIMESTAMP WITH TIME ZONE,
			-- Context for resumption
			execution_context JSONB DEFAULT '{}',
			checkpoint_data JSONB DEFAULT '{}',
			-- Audit and monitoring
			created_by VARCHAR(255) NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			-- Integration points
			go_engine_session_id VARCHAR(255),
			rust_engine_request_id VARCHAR(255),
			context_gateway_request_id VARCHAR(255)
		);
		
		-- Workflow tracking indexes
		CREATE INDEX IF NOT EXISTS idx_workflow_states_workflow_id ON workflow_states(workflow_id);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_patient ON workflow_states(patient_id, started_at DESC);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_phase ON workflow_states(current_phase, phase_status);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_overall_status ON workflow_states(overall_status);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_proposal ON workflow_states(proposal_id);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_snapshot ON workflow_states(snapshot_id);
		
		-- Performance monitoring indexes
		CREATE INDEX IF NOT EXISTS idx_workflow_states_performance ON workflow_states(total_processing_ms, completed_at);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_errors ON workflow_states(retry_count, last_retry_at) WHERE retry_count > 0;
		
		-- Integration tracking indexes
		CREATE INDEX IF NOT EXISTS idx_workflow_states_go_session ON workflow_states(go_engine_session_id);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_rust_request ON workflow_states(rust_engine_request_id);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create workflow_states table: %w", err)
	}

	sm.logger.Info("Created workflow_states table successfully")
	return nil
}

// CreateAuditTrailTable creates comprehensive audit trail for HIPAA compliance
func (sm *SchemaManager) CreateAuditTrailTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_trail (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			-- Audit event identification
			event_id VARCHAR(255) NOT NULL UNIQUE,
			event_type VARCHAR(100) NOT NULL,
			event_category VARCHAR(50) NOT NULL CHECK (event_category IN ('access', 'modification', 'creation', 'deletion', 'calculation', 'validation', 'security', 'system')),
			-- User and session tracking
			user_id VARCHAR(255) NOT NULL,
			user_role VARCHAR(100),
			session_id VARCHAR(255),
			ip_address INET,
			user_agent TEXT,
			-- Resource identification
			resource_type VARCHAR(100) NOT NULL, -- recipe, snapshot, proposal, workflow
			resource_id UUID NOT NULL,
			patient_id UUID, -- For patient-related events
			-- Event details
			action VARCHAR(100) NOT NULL,
			description TEXT NOT NULL,
			details JSONB DEFAULT '{}',
			-- Changes tracking
			old_values JSONB,
			new_values JSONB,
			changes_summary TEXT,
			-- Clinical context
			clinical_context JSONB DEFAULT '{}',
			safety_impact VARCHAR(50), -- none, low, medium, high, critical
			compliance_flags JSONB DEFAULT '[]',
			-- Timing
			event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			processing_time_ms INTEGER,
			-- Status and outcome
			status VARCHAR(50) NOT NULL CHECK (status IN ('success', 'failure', 'warning', 'partial')),
			error_message TEXT,
			error_code VARCHAR(50),
			-- Metadata
			service_name VARCHAR(100) DEFAULT 'medication-service-v2',
			service_version VARCHAR(50),
			request_id VARCHAR(255),
			correlation_id VARCHAR(255),
			-- Retention and archiving
			retention_policy VARCHAR(100) DEFAULT 'standard_7_years',
			archived BOOLEAN DEFAULT FALSE,
			archived_at TIMESTAMP WITH TIME ZONE
		);
		
		-- Primary audit indexes
		CREATE INDEX IF NOT EXISTS idx_audit_event_timestamp ON audit_trail(event_timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_user_events ON audit_trail(user_id, event_timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_patient_events ON audit_trail(patient_id, event_timestamp DESC) WHERE patient_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_trail(resource_type, resource_id, event_timestamp DESC);
		
		-- Event classification indexes
		CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_trail(event_type, event_category);
		CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_trail(action, status);
		CREATE INDEX IF NOT EXISTS idx_audit_safety_impact ON audit_trail(safety_impact, event_timestamp DESC) WHERE safety_impact IN ('high', 'critical');
		
		-- Session and request tracking
		CREATE INDEX IF NOT EXISTS idx_audit_session ON audit_trail(session_id, event_timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_trail(request_id, correlation_id);
		
		-- HIPAA compliance indexes
		CREATE INDEX IF NOT EXISTS idx_audit_compliance_flags ON audit_trail USING GIN (compliance_flags);
		CREATE INDEX IF NOT EXISTS idx_audit_retention ON audit_trail(retention_policy, archived, event_timestamp);
		
		-- Performance monitoring
		CREATE INDEX IF NOT EXISTS idx_audit_performance ON audit_trail(processing_time_ms DESC, event_timestamp DESC) WHERE processing_time_ms IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_audit_errors ON audit_trail(status, error_code, event_timestamp DESC) WHERE status = 'failure';
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create audit_trail table: %w", err)
	}

	sm.logger.Info("Created audit_trail table successfully")
	return nil
}

// CreateCacheableDataTable creates table for caching frequently accessed data
func (sm *SchemaManager) CreateCacheableDataTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS cacheable_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			cache_key VARCHAR(255) NOT NULL UNIQUE,
			cache_type VARCHAR(100) NOT NULL, -- patient_demographics, lab_results, medication_history, etc.
			entity_id UUID NOT NULL, -- patient_id, medication_id, etc.
			data_payload JSONB NOT NULL,
			-- Cache metadata
			computed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			access_count INTEGER DEFAULT 0,
			hit_rate DECIMAL(5,4) DEFAULT 0.0,
			-- Invalidation tracking
			version VARCHAR(50) DEFAULT '1.0',
			dependent_resources JSONB DEFAULT '[]', -- Resources this cache depends on
			invalidation_triggers JSONB DEFAULT '[]',
			-- Performance optimization
			computation_cost_ms INTEGER DEFAULT 0,
			data_size_bytes INTEGER DEFAULT 0,
			compression_ratio DECIMAL(4,3) DEFAULT 1.0,
			-- Quality metrics
			freshness_score DECIMAL(3,2) DEFAULT 1.0,
			accuracy_score DECIMAL(3,2) DEFAULT 1.0,
			data_source VARCHAR(255) NOT NULL
		);
		
		-- TTL index for automatic cache expiration
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_ttl ON cacheable_data(expires_at);
		
		-- Cache lookup indexes
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_key ON cacheable_data(cache_key);
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_type_entity ON cacheable_data(cache_type, entity_id);
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_entity_type ON cacheable_data(entity_id, cache_type);
		
		-- Performance monitoring indexes
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_access ON cacheable_data(last_accessed DESC, access_count DESC);
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_hit_rate ON cacheable_data(hit_rate DESC, access_count DESC);
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_cost ON cacheable_data(computation_cost_ms DESC, data_size_bytes DESC);
		
		-- Data quality indexes
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_quality ON cacheable_data(freshness_score DESC, accuracy_score DESC);
		CREATE INDEX IF NOT EXISTS idx_cacheable_data_version ON cacheable_data(version, computed_at DESC);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create cacheable_data table: %w", err)
	}

	sm.logger.Info("Created cacheable_data table successfully")
	return nil
}

// CreatePerformanceMetricsTable creates table for performance monitoring
func (sm *SchemaManager) CreatePerformanceMetricsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS performance_metrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			metric_name VARCHAR(255) NOT NULL,
			metric_type VARCHAR(100) NOT NULL, -- timing, throughput, error_rate, resource_usage
			service_component VARCHAR(100) NOT NULL, -- recipe_resolver, context_gateway, rust_engine, etc.
			-- Measurement data
			metric_value DECIMAL(15,6) NOT NULL,
			metric_unit VARCHAR(50) NOT NULL, -- ms, rps, percentage, bytes, etc.
			measured_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			-- Context information
			request_id VARCHAR(255),
			workflow_id VARCHAR(255),
			patient_id UUID,
			recipe_id UUID,
			snapshot_id UUID,
			-- Performance targets (for comparison)
			target_value DECIMAL(15,6),
			target_met BOOLEAN,
			performance_delta DECIMAL(15,6), -- Difference from target
			-- Aggregation support
			aggregation_period VARCHAR(50), -- minute, hour, day
			bucket_start TIMESTAMP WITH TIME ZONE,
			bucket_end TIMESTAMP WITH TIME ZONE,
			sample_count INTEGER DEFAULT 1,
			-- Additional metadata
			metadata JSONB DEFAULT '{}',
			environment VARCHAR(50) DEFAULT 'production',
			version VARCHAR(50)
		);
		
		-- Time-series indexes for performance analysis
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_time ON performance_metrics(measured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_name_time ON performance_metrics(metric_name, measured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_component ON performance_metrics(service_component, metric_type, measured_at DESC);
		
		-- Request correlation indexes
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_request ON performance_metrics(request_id, measured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_workflow ON performance_metrics(workflow_id, measured_at DESC);
		
		-- Resource correlation indexes
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_patient ON performance_metrics(patient_id, measured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_recipe ON performance_metrics(recipe_id, measured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_snapshot ON performance_metrics(snapshot_id, measured_at DESC);
		
		-- Performance analysis indexes
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_targets ON performance_metrics(target_met, performance_delta DESC) WHERE target_value IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_aggregation ON performance_metrics(aggregation_period, bucket_start, bucket_end) WHERE aggregation_period IS NOT NULL;
		
		-- Environment and version tracking
		CREATE INDEX IF NOT EXISTS idx_performance_metrics_env_version ON performance_metrics(environment, version, measured_at DESC);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create performance_metrics table: %w", err)
	}

	sm.logger.Info("Created performance_metrics table successfully")
	return nil
}

// CreateSecurityEventsTable creates table for security event tracking
func (sm *SchemaManager) CreateSecurityEventsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS security_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			-- Event identification
			event_id VARCHAR(255) NOT NULL UNIQUE,
			event_type VARCHAR(100) NOT NULL, -- authentication, authorization, data_access, integrity_check, etc.
			severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
			status VARCHAR(50) NOT NULL CHECK (status IN ('detected', 'investigating', 'confirmed', 'false_positive', 'mitigated', 'resolved')),
			-- User and session information
			user_id VARCHAR(255),
			user_role VARCHAR(100),
			session_id VARCHAR(255),
			ip_address INET,
			user_agent TEXT,
			-- Event details
			description TEXT NOT NULL,
			event_details JSONB DEFAULT '{}',
			detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			-- Resource involved
			resource_type VARCHAR(100),
			resource_id UUID,
			patient_id UUID,
			-- Security context
			threat_indicators JSONB DEFAULT '[]',
			risk_score DECIMAL(4,2), -- 0.00 to 10.00
			mitre_attack_ids JSONB DEFAULT '[]', -- MITRE ATT&CK framework references
			-- Detection metadata
			detection_method VARCHAR(100), -- rule_based, ml_based, signature, behavioral, etc.
			detection_confidence DECIMAL(3,2), -- 0.00 to 1.00
			false_positive_probability DECIMAL(3,2),
			-- Response and mitigation
			response_actions JSONB DEFAULT '[]',
			mitigated_at TIMESTAMP WITH TIME ZONE,
			mitigation_details TEXT,
			resolved_at TIMESTAMP WITH TIME ZONE,
			resolution_details TEXT,
			-- Investigation tracking
			assigned_to VARCHAR(255),
			investigation_notes TEXT,
			last_updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			-- Compliance and reporting
			compliance_impact VARCHAR(100), -- HIPAA, SOX, FDA, etc.
			reporting_required BOOLEAN DEFAULT FALSE,
			reported_to_authorities BOOLEAN DEFAULT FALSE,
			reported_at TIMESTAMP WITH TIME ZONE,
			-- Metadata
			service_name VARCHAR(100) DEFAULT 'medication-service-v2',
			version VARCHAR(50),
			environment VARCHAR(50) DEFAULT 'production'
		);
		
		-- Security monitoring indexes
		CREATE INDEX IF NOT EXISTS idx_security_events_detected_at ON security_events(detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_severity ON security_events(severity, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_status ON security_events(status, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_type ON security_events(event_type, severity, detected_at DESC);
		
		-- Risk assessment indexes
		CREATE INDEX IF NOT EXISTS idx_security_events_risk_score ON security_events(risk_score DESC, detected_at DESC) WHERE risk_score IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_security_events_high_risk ON security_events(detected_at DESC) WHERE severity IN ('high', 'critical') AND status NOT IN ('false_positive', 'resolved');
		
		-- User and session tracking
		CREATE INDEX IF NOT EXISTS idx_security_events_user ON security_events(user_id, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_session ON security_events(session_id, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_ip ON security_events(ip_address, detected_at DESC);
		
		-- Resource correlation indexes
		CREATE INDEX IF NOT EXISTS idx_security_events_resource ON security_events(resource_type, resource_id, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_patient ON security_events(patient_id, detected_at DESC) WHERE patient_id IS NOT NULL;
		
		-- Investigation and response indexes
		CREATE INDEX IF NOT EXISTS idx_security_events_assigned ON security_events(assigned_to, status, detected_at DESC);
		CREATE INDEX IF NOT EXISTS idx_security_events_response_time ON security_events(detected_at, mitigated_at, resolved_at);
		
		-- Compliance and reporting indexes
		CREATE INDEX IF NOT EXISTS idx_security_events_compliance ON security_events(compliance_impact, reporting_required, reported_to_authorities);
		CREATE INDEX IF NOT EXISTS idx_security_events_reporting ON security_events(reporting_required, reported_at) WHERE reporting_required = TRUE;
		
		-- GIN indexes for JSON arrays
		CREATE INDEX IF NOT EXISTS idx_security_events_threat_indicators ON security_events USING GIN (threat_indicators);
		CREATE INDEX IF NOT EXISTS idx_security_events_mitre_attacks ON security_events USING GIN (mitre_attack_ids);
	`

	_, err := sm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create security_events table: %w", err)
	}

	sm.logger.Info("Created security_events table successfully")
	return nil
}

// CreateIndexes creates additional performance and specialized indexes
func (sm *SchemaManager) CreateIndexes(ctx context.Context) error {
	indexQueries := []string{
		// Cross-table relationship indexes
		`CREATE INDEX IF NOT EXISTS idx_cross_recipe_snapshots ON clinical_snapshots(recipe_id, patient_id, status, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_cross_snapshot_proposals ON medication_proposals(snapshot_id, status, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_cross_proposal_workflows ON workflow_states(proposal_id, overall_status, started_at DESC);`,
		
		// Performance optimization indexes
		`CREATE INDEX IF NOT EXISTS idx_recipes_performance_lookup ON recipes(status, complexity_score, cache_priority) WHERE status = 'active';`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_active_recent ON clinical_snapshots(status, expires_at, created_at DESC) WHERE status = 'active';`,
		`CREATE INDEX IF NOT EXISTS idx_proposals_pending_validation ON medication_proposals(status, confidence_score DESC) WHERE status IN ('proposed', 'draft');`,
		
		// Analytics and reporting indexes
		`CREATE INDEX IF NOT EXISTS idx_audit_daily_summary ON audit_trail(date_trunc('day', event_timestamp), event_category, status);`,
		`CREATE INDEX IF NOT EXISTS idx_performance_hourly_avg ON performance_metrics(date_trunc('hour', measured_at), service_component, metric_name);`,
		`CREATE INDEX IF NOT EXISTS idx_security_events_daily ON security_events(date_trunc('day', detected_at), severity, status);`,
		
		// Cleanup and maintenance indexes
		`CREATE INDEX IF NOT EXISTS idx_expired_snapshots ON clinical_snapshots(expires_at, status) WHERE status != 'expired';`,
		`CREATE INDEX IF NOT EXISTS idx_expired_proposals ON medication_proposals(expires_at, status) WHERE status != 'expired';`,
		`CREATE INDEX IF NOT EXISTS idx_old_audit_records ON audit_trail(event_timestamp, archived) WHERE archived = FALSE;`,
	}

	for _, query := range indexQueries {
		_, err := sm.db.DB.ExecContext(ctx, query)
		if err != nil {
			sm.logger.Warn("Failed to create index", zap.String("query", query), zap.Error(err))
			// Continue with other indexes even if one fails
			continue
		}
	}

	sm.logger.Info("Created additional performance indexes")
	return nil
}

// CreateFunctions creates PostgreSQL functions for optimization
func (sm *SchemaManager) CreateFunctions(ctx context.Context) error {
	functionQueries := []string{
		// Function to calculate snapshot hash
		`
		CREATE OR REPLACE FUNCTION calculate_snapshot_hash(clinical_data JSONB)
		RETURNS TEXT AS $$
		BEGIN
			RETURN encode(digest(clinical_data::text, 'sha256'), 'hex');
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;
		`,
		
		// Function to check snapshot expiry
		`
		CREATE OR REPLACE FUNCTION is_snapshot_expired(expires_at TIMESTAMP WITH TIME ZONE)
		RETURNS BOOLEAN AS $$
		BEGIN
			RETURN expires_at < NOW();
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;
		`,
		
		// Function to update snapshot access tracking
		`
		CREATE OR REPLACE FUNCTION update_snapshot_access(snapshot_uuid UUID)
		RETURNS VOID AS $$
		BEGIN
			UPDATE clinical_snapshots 
			SET access_count = access_count + 1,
				last_accessed_at = NOW()
			WHERE id = snapshot_uuid;
		END;
		$$ LANGUAGE plpgsql;
		`,
		
		// Function for efficient recipe lookup with caching preference
		`
		CREATE OR REPLACE FUNCTION get_cached_recipe(protocol_id_param VARCHAR, version_param VARCHAR DEFAULT NULL)
		RETURNS TABLE(recipe_data JSONB) AS $$
		BEGIN
			IF version_param IS NULL THEN
				RETURN QUERY
				SELECT row_to_json(r)::jsonb as recipe_data
				FROM recipes r
				WHERE r.protocol_id = protocol_id_param
				  AND r.status = 'active'
				ORDER BY r.cache_priority DESC, r.version_sequence DESC
				LIMIT 1;
			ELSE
				RETURN QUERY
				SELECT row_to_json(r)::jsonb as recipe_data
				FROM recipes r
				WHERE r.protocol_id = protocol_id_param
				  AND r.version = version_param
				  AND r.status = 'active'
				LIMIT 1;
			END IF;
		END;
		$$ LANGUAGE plpgsql;
		`,
	}

	for _, query := range functionQueries {
		_, err := sm.db.DB.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create function: %w", err)
		}
	}

	sm.logger.Info("Created database functions successfully")
	return nil
}

// CreateTriggers creates triggers for automatic data management
func (sm *SchemaManager) CreateTriggers(ctx context.Context) error {
	triggerQueries := []string{
		// Auto-update timestamps
		`
		CREATE OR REPLACE FUNCTION update_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		`,
		
		`CREATE TRIGGER recipes_update_timestamp
			BEFORE UPDATE ON recipes
			FOR EACH ROW EXECUTE FUNCTION update_timestamp();`,
		
		`CREATE TRIGGER medication_proposals_update_timestamp
			BEFORE UPDATE ON medication_proposals
			FOR EACH ROW EXECUTE FUNCTION update_timestamp();`,
		
		`CREATE TRIGGER workflow_states_update_timestamp
			BEFORE UPDATE ON workflow_states
			FOR EACH ROW EXECUTE FUNCTION update_timestamp();`,
		
		// Auto-expire snapshots
		`
		CREATE OR REPLACE FUNCTION auto_expire_snapshot()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NEW.expires_at < NOW() AND NEW.status != 'expired' THEN
				NEW.status = 'expired';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		`,
		
		`CREATE TRIGGER clinical_snapshots_auto_expire
			BEFORE UPDATE ON clinical_snapshots
			FOR EACH ROW EXECUTE FUNCTION auto_expire_snapshot();`,
	}

	for _, query := range triggerQueries {
		_, err := sm.db.DB.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	sm.logger.Info("Created database triggers successfully")
	return nil
}

// DropAllTables drops all tables (for testing/development)
func (sm *SchemaManager) DropAllTables(ctx context.Context) error {
	dropQueries := []string{
		"DROP TABLE IF EXISTS security_events CASCADE;",
		"DROP TABLE IF EXISTS performance_metrics CASCADE;",
		"DROP TABLE IF EXISTS cacheable_data CASCADE;",
		"DROP TABLE IF EXISTS audit_trail CASCADE;",
		"DROP TABLE IF EXISTS workflow_states CASCADE;",
		"DROP TABLE IF EXISTS medication_proposals CASCADE;",
		"DROP TABLE IF EXISTS clinical_snapshots CASCADE;",
		"DROP TABLE IF EXISTS recipes CASCADE;",
		"DROP TABLE IF EXISTS schema_migrations CASCADE;",
	}

	for _, query := range dropQueries {
		_, err := sm.db.DB.ExecContext(ctx, query)
		if err != nil {
			sm.logger.Warn("Failed to drop table", zap.String("query", query), zap.Error(err))
		}
	}

	sm.logger.Info("Dropped all tables")
	return nil
}

// GetTableInfo returns information about database tables
func (sm *SchemaManager) GetTableInfo(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			tableowner,
			hasindexes,
			hasrules,
			hastriggers,
			rowsecurity
		FROM pg_tables 
		WHERE schemaname = 'public'
		ORDER BY tablename;
	`

	rows, err := sm.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	tables := make([]map[string]interface{}, 0)
	for rows.Next() {
		var schemaname, tablename, tableowner string
		var hasindexes, hasrules, hastriggers, rowsecurity bool
		
		err := rows.Scan(&schemaname, &tablename, &tableowner, &hasindexes, &hasrules, &hastriggers, &rowsecurity)
		if err != nil {
			continue
		}
		
		table := map[string]interface{}{
			"schema": schemaname,
			"name": tablename,
			"owner": tableowner,
			"has_indexes": hasindexes,
			"has_rules": hasrules,
			"has_triggers": hastriggers,
			"row_security": rowsecurity,
		}
		tables = append(tables, table)
	}

	return map[string]interface{}{
		"tables": tables,
		"count": len(tables),
	}, nil
}