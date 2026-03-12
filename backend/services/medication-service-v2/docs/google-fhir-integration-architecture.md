# Google FHIR Healthcare API Integration Architecture

## Overview

This document describes the comprehensive database schema and integration architecture for Medication Service V2 that **complements** (not replaces) Google FHIR Healthcare API. Our approach provides operational data management while leveraging Google FHIR as the authoritative source for clinical data.

## Architecture Principles

### 1. **Complement, Don't Duplicate** 
- **Google FHIR**: Authoritative source for clinical data (Patient, Medication, Observation, etc.)
- **Our Database**: Operational data, workflow state, processing results, audit trails
- **Integration**: Reference-based mapping with synchronization and caching

### 2. **Reference-Based Integration**
```
┌─────────────────────────┐    ┌─────────────────────────┐
│   Google FHIR Store     │    │   Our PostgreSQL       │
│                         │    │                         │
│ • Patient Resources     │◄──►│ • Workflow Executions   │
│ • Medication Resources  │    │ • Clinical Calculations │  
│ • Observation Resources │    │ • Proposal Workflows    │
│ • FHIR Bundles         │    │ • Audit Trails          │
└─────────────────────────┘    │ • Performance Metrics   │
                               │ • FHIR Resource Refs    │
                               └─────────────────────────┘
```

### 3. **Performance Optimization**
- **Target**: <250ms end-to-end workflow execution
- **Strategy**: Smart caching, reference pre-loading, parallel FHIR calls
- **Monitoring**: Real-time performance tracking with database metrics

## Database Schema Architecture

### Core Integration Tables

#### 1. **Google FHIR Configuration** (`google_fhir_config`)
```sql
-- Configuration for Google FHIR Healthcare API connection
CREATE TABLE google_fhir_config (
    id UUID PRIMARY KEY,
    config_name VARCHAR(255) NOT NULL UNIQUE,
    project_id VARCHAR(255) NOT NULL,          -- Google Cloud Project ID
    location VARCHAR(100) NOT NULL,            -- us-central1, etc.
    dataset_id VARCHAR(255) NOT NULL,          -- FHIR dataset ID
    fhir_store_id VARCHAR(255) NOT NULL,       -- FHIR store ID
    base_url VARCHAR(500) NOT NULL,            -- Full FHIR API base URL
    -- Authentication & Performance
    sync_enabled BOOLEAN DEFAULT TRUE,
    rate_limit_per_minute INTEGER DEFAULT 1000,
    timeout_seconds INTEGER DEFAULT 30,
    enable_caching BOOLEAN DEFAULT TRUE,
    -- Compliance
    enable_audit_logging BOOLEAN DEFAULT TRUE,
    data_residency_region VARCHAR(100),
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    environment VARCHAR(50) DEFAULT 'production'
);
```

#### 2. **FHIR Resource Mappings** (`fhir_resource_mappings`)
```sql
-- Maps our internal resources to Google FHIR resources (NO DATA DUPLICATION)
CREATE TABLE fhir_resource_mappings (
    id UUID PRIMARY KEY,
    -- Our internal resource
    internal_resource_type VARCHAR(100) NOT NULL,  -- 'workflow', 'snapshot', 'proposal'
    internal_resource_id UUID NOT NULL,
    -- Google FHIR resource reference (REFERENCE ONLY, NOT DATA)
    fhir_resource_type VARCHAR(100) NOT NULL,      -- 'Patient', 'Medication', etc.
    fhir_resource_id VARCHAR(255) NOT NULL,
    fhir_version_id VARCHAR(255),
    fhir_full_url VARCHAR(1000) NOT NULL,          -- Full Google FHIR URL
    fhir_last_updated TIMESTAMP WITH TIME ZONE,
    -- Mapping metadata
    mapping_type VARCHAR(50) NOT NULL,             -- 'primary', 'derived', 'referenced'
    mapping_purpose VARCHAR(100) NOT NULL,         -- 'patient_context', 'medication_data'
    sync_status VARCHAR(50) DEFAULT 'synchronized',
    -- Performance optimization
    content_hash VARCHAR(64),                      -- For change detection
    access_frequency INTEGER DEFAULT 0,
    cache_priority INTEGER DEFAULT 5,
    -- UNIQUE constraint prevents duplicate mappings
    CONSTRAINT unique_internal_fhir_mapping UNIQUE (
        internal_resource_type, internal_resource_id, 
        fhir_resource_type, fhir_resource_id
    )
);
```

#### 3. **FHIR Synchronization Status** (`fhir_sync_status`)
```sql
-- Tracks synchronization operations with Google FHIR
CREATE TABLE fhir_sync_status (
    id UUID PRIMARY KEY,
    sync_batch_id VARCHAR(255) NOT NULL,
    sync_type VARCHAR(50) NOT NULL,                -- 'full_sync', 'incremental_sync'
    resource_type VARCHAR(100),                     -- Specific resource type or NULL for full
    -- Execution tracking
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    -- Performance metrics
    total_resources INTEGER DEFAULT 0,
    successful_resources INTEGER DEFAULT 0,
    failed_resources INTEGER DEFAULT 0,
    processing_time_ms INTEGER,
    throughput_resources_per_second DECIMAL(10,2),
    -- Google FHIR API usage tracking
    api_requests_made INTEGER DEFAULT 0,
    api_quota_consumed INTEGER DEFAULT 0,
    api_rate_limit_hits INTEGER DEFAULT 0,
    -- Quality assurance
    content_validation_passed BOOLEAN,
    schema_validation_passed BOOLEAN
);
```

### Operational Data Tables

#### 4. **Workflow Executions** (`workflow_executions`)
```sql
-- 4-Phase Workflow execution tracking with FHIR integration
CREATE TABLE workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id VARCHAR(255) NOT NULL UNIQUE,
    patient_id UUID NOT NULL,
    -- Google FHIR references (NOT data duplication)
    patient_fhir_reference JSONB NOT NULL,         -- FHIRResourceReference
    clinical_context_fhir_references JSONB DEFAULT '[]',  -- Array of references
    -- Workflow specification
    workflow_type VARCHAR(100) NOT NULL,           -- 'medication_recommendation', etc.
    protocol_id VARCHAR(255),
    recipe_id UUID REFERENCES recipes(id),
    priority VARCHAR(20) DEFAULT 'normal',
    -- 4-Phase execution tracking (OUR processing state)
    current_phase INTEGER DEFAULT 1,
    phase_1_recipe_resolution JSONB DEFAULT '{}',  -- Our processing results
    phase_2_context_assembly JSONB DEFAULT '{}',   -- Our context assembly
    phase_3_clinical_intelligence JSONB DEFAULT '{}', -- Our clinical analysis
    phase_4_proposal_generation JSONB DEFAULT '{}', -- Our proposals
    -- Performance tracking (targeting <250ms)
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    total_execution_time_ms INTEGER,
    performance_target_ms INTEGER DEFAULT 250,
    performance_target_met BOOLEAN,
    -- Quality assurance
    quality_score DECIMAL(3,2),
    validation_passed BOOLEAN,
    safety_checks_passed BOOLEAN
);
```

#### 5. **Clinical Calculation Results** (`clinical_calculation_results`)
```sql
-- Stores OUR clinical calculation results (not FHIR data)
CREATE TABLE clinical_calculation_results (
    id UUID PRIMARY KEY,
    calculation_id VARCHAR(255) NOT NULL UNIQUE,
    workflow_execution_id UUID REFERENCES workflow_executions(id),
    patient_id UUID NOT NULL,
    -- FHIR references for input data (NOT the actual data)
    input_fhir_references JSONB NOT NULL DEFAULT '[]',
    -- Calculation specification
    calculation_type VARCHAR(100) NOT NULL,        -- 'dosage_calculation', etc.
    rule_engine_used VARCHAR(100) NOT NULL,        -- 'rust_engine', 'flow2_go'
    algorithm_version VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}',
    -- OUR calculation results (not FHIR duplication)
    results JSONB NOT NULL DEFAULT '{}',           -- Our processed results
    confidence_score DECIMAL(4,3),
    risk_assessment JSONB DEFAULT '{}',            -- Our risk analysis
    safety_flags JSONB DEFAULT '[]',              -- Our safety evaluation
    clinical_recommendations JSONB DEFAULT '[]',   -- Our recommendations
    -- Performance and quality metrics
    processing_time_ms INTEGER,
    quality_score DECIMAL(4,3),
    evidence_strength VARCHAR(20)
);
```

#### 6. **Recipe & Snapshot Architecture**

**Enhanced Recipes** (`recipes`):
```sql
CREATE TABLE recipes (
    id UUID PRIMARY KEY,
    protocol_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    -- Our internal processing rules
    context_requirements JSONB NOT NULL DEFAULT '{}',
    calculation_rules JSONB NOT NULL DEFAULT '[]',
    safety_rules JSONB NOT NULL DEFAULT '[]',
    -- Google FHIR references (NOT data duplication)
    fhir_protocol_reference JSONB,                 -- Reference to PlanDefinition
    fhir_evidence_references JSONB DEFAULT '[]',   -- Evidence references
    fhir_medication_references JSONB DEFAULT '[]', -- Medication references
    -- Performance optimization
    complexity_score DECIMAL(3,2) DEFAULT 0.0,
    average_execution_ms INTEGER DEFAULT 0,
    success_rate DECIMAL(5,4) DEFAULT 0.0000,
    -- Quality assurance
    evidence_quality_score DECIMAL(4,3),
    clinical_validation_status VARCHAR(50) DEFAULT 'pending'
);
```

**Enhanced Clinical Snapshots** (`clinical_snapshots`):
```sql
CREATE TABLE clinical_snapshots (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL,
    recipe_id UUID NOT NULL REFERENCES recipes(id),
    workflow_execution_id UUID REFERENCES workflow_executions(id),
    -- Google FHIR references (NOT data duplication)
    patient_fhir_reference JSONB NOT NULL,
    clinical_context_fhir_references JSONB DEFAULT '[]',
    source_fhir_bundle_reference JSONB,
    -- OUR processed snapshot data
    clinical_data JSONB NOT NULL DEFAULT '{}',      -- Our clinical context
    processing_results JSONB DEFAULT '{}',          -- Our analysis results
    validation_results JSONB NOT NULL DEFAULT '{}', -- Our validation
    -- Cryptographic integrity
    content_hash VARCHAR(64) NOT NULL,              -- SHA-256 of our data
    integrity_signature VARCHAR(512),
    -- Performance tracking
    assembly_duration_ms INTEGER DEFAULT 0,
    completeness_score DECIMAL(4,3) DEFAULT 0.000,
    quality_score DECIMAL(4,3) DEFAULT 0.000,
    -- Evidence envelope for clinical decision support
    evidence_envelope JSONB DEFAULT '{}',
    clinical_reasoning JSONB DEFAULT '{}',          -- Our reasoning
    safety_assessment JSONB DEFAULT '{}',          -- Our safety analysis
    -- Lifecycle management
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

### Audit and Compliance Tables

#### 7. **FHIR Integration Logs** (`fhir_integration_logs`)
```sql
-- Comprehensive logging for Google FHIR operations
CREATE TABLE fhir_integration_logs (
    id UUID PRIMARY KEY,
    log_event_id VARCHAR(255) NOT NULL,
    correlation_id VARCHAR(255),
    -- FHIR operation details
    operation_type VARCHAR(50) NOT NULL,           -- 'read', 'search', 'create'
    fhir_resource_type VARCHAR(100),
    fhir_resource_id VARCHAR(255),
    fhir_endpoint VARCHAR(500) NOT NULL,
    -- HTTP details
    http_method VARCHAR(10) NOT NULL,
    http_status_code INTEGER,
    request_body_size_bytes INTEGER DEFAULT 0,
    response_body_size_bytes INTEGER DEFAULT 0,
    -- Performance metrics
    total_latency_ms INTEGER,
    dns_resolution_ms INTEGER,
    tcp_connection_ms INTEGER,
    tls_handshake_ms INTEGER,
    -- Error tracking
    success BOOLEAN NOT NULL DEFAULT FALSE,
    error_code VARCHAR(100),
    error_message TEXT,
    fhir_operation_outcome JSONB,
    -- Data governance
    patient_id_accessed UUID,
    data_sensitivity_level VARCHAR(20) DEFAULT 'high',
    -- Rate limiting
    rate_limit_remaining INTEGER,
    quota_used INTEGER,
    -- Compliance
    hipaa_compliant BOOLEAN DEFAULT TRUE,
    audit_required BOOLEAN DEFAULT TRUE
);
```

## Integration Patterns

### 1. **Reference Resolution Pattern**
```go
// Instead of storing patient data, we store references
type PatientReference struct {
    FHIRResourceReference `json:"fhir_reference"`
    CacheStatus           string `json:"cache_status"`
    LastSynchronized      time.Time `json:"last_synchronized"`
}

// Usage in workflow
func (w *WorkflowExecution) LoadPatientContext() error {
    // 1. Get patient FHIR reference from our mapping
    patientRef := w.PatientFHIRReference
    
    // 2. Fetch current data from Google FHIR (with caching)
    patientData, err := fhirClient.GetResource(patientRef)
    
    // 3. Process and analyze (store results, not raw data)
    context := w.processPatientData(patientData)
    w.ProcessedContext = context
    
    return nil
}
```

### 2. **Workflow Execution with FHIR Integration**
```
Phase 1: Recipe Resolution
├─ Load recipe from our database
├─ Get FHIR medication references
└─ Resolve protocol parameters

Phase 2: Context Assembly  
├─ Get patient FHIR reference
├─ Fetch current clinical data from Google FHIR
├─ Create clinical snapshot (with FHIR references)
└─ Store OUR processed context

Phase 3: Clinical Intelligence
├─ Analyze assembled context (our processing)
├─ Apply clinical rules (our logic)
├─ Generate safety assessment (our results)
└─ Store calculation results

Phase 4: Proposal Generation
├─ Generate medication proposals (our recommendations)
├─ Create FHIR MedicationRequest references
├─ Store workflow results
└─ Update performance metrics
```

### 3. **Caching and Performance Strategy**

#### Smart Caching Layers
```
Level 1: In-Memory Cache (Redis)
├─ Frequently accessed FHIR references
├─ Recent calculation results
└─ Active workflow states

Level 2: Database Cache (PostgreSQL)
├─ FHIR resource mappings
├─ Processed clinical contexts
└─ Performance metrics

Level 3: Google FHIR (Source of Truth)
├─ Complete clinical data
├─ Patient resources
└─ Medication resources
```

#### Performance Optimization
- **Parallel FHIR Calls**: Batch requests for multiple resources
- **Smart Prefetching**: Pre-load commonly used references
- **Change Detection**: Use content hashes to detect updates
- **Connection Pooling**: Optimize HTTP connections to Google FHIR

## Migration System

### Database Migration Architecture
- **Up/Down Migrations**: Full rollback capability
- **Checksum Validation**: Ensure migration integrity
- **Transaction Support**: Atomic migration execution
- **Audit Trail**: Complete migration history

### Sample Migration Structure
```
migrations/
├─ 001_initial_google_fhir_integration.sql
├─ 002_workflow_execution_tracking.sql
├─ 003_recipe_snapshot_enhanced.sql
├─ 004_medication_proposal_workflows.sql
└─ 005_performance_optimization.sql
```

## Security and Compliance

### HIPAA Compliance
- **Audit Trails**: Complete tracking of all data access
- **Data Minimization**: Store only operational data, reference FHIR
- **Encryption**: At-rest and in-transit encryption
- **Access Controls**: Role-based access with JWT authentication
- **Retention Policies**: Configurable data retention (default: 7 years)

### Data Governance
```sql
-- All tables include data governance fields
data_classification VARCHAR(50) DEFAULT 'clinical',
retention_period_days INTEGER DEFAULT 2555,  -- 7 years
hipaa_tracking JSONB DEFAULT '{}',
compliance_flags JSONB DEFAULT '[]'
```

## Performance Targets

### End-to-End Performance
- **Recipe Resolution**: <10ms
- **Context Assembly**: <100ms  
- **Clinical Intelligence**: <50ms
- **Proposal Generation**: <100ms
- **Total Workflow**: <250ms

### Database Performance
- **Query Response**: <5ms for indexed lookups
- **FHIR Reference Resolution**: <20ms with caching
- **Batch Operations**: 1000+ resources/second
- **Connection Pooling**: 25 max connections, 5 idle

### Google FHIR API Performance
- **Single Resource Fetch**: <50ms
- **Search Operations**: <200ms
- **Rate Limiting**: 1000 requests/minute
- **Quota Management**: Automatic throttling

## Operational Excellence

### Monitoring and Observability
```sql
-- Performance metrics tracking
SELECT 
    workflow_type,
    AVG(total_execution_time_ms) as avg_execution_time,
    COUNT(*) FILTER (WHERE performance_target_met) as target_met_count,
    COUNT(*) as total_workflows
FROM workflow_executions 
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY workflow_type;

-- FHIR integration health
SELECT 
    operation_type,
    COUNT(*) as total_operations,
    COUNT(*) FILTER (WHERE success) as successful_operations,
    AVG(total_latency_ms) as avg_latency
FROM fhir_integration_logs
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY operation_type;
```

### Health Checks
- **Database Health**: Connection pool status, query performance
- **FHIR Integration Health**: API connectivity, quota usage
- **Cache Health**: Hit rates, memory usage
- **Workflow Health**: Success rates, performance targets

### Error Handling and Recovery
- **Circuit Breakers**: Prevent cascade failures
- **Retry Logic**: Exponential backoff with jitter
- **Graceful Degradation**: Continue operations when possible
- **Alerting**: Real-time notifications for critical issues

## Development and Deployment

### Configuration Management
```yaml
# config.yaml
google_fhir:
  project_id: "your-project"
  location: "us-central1" 
  dataset_id: "clinical-data"
  fhir_store_id: "medication-fhir"
  sync_enabled: true
  rate_limit_per_minute: 1000
  enable_caching: true
  cache_ttl_minutes: 30

database:
  url: "postgresql://user:pass@localhost:5434/medication_v2"
  port: 5434  # Avoid conflict with existing service on 5432
```

### Environment Setup
```bash
# Initialize database with migrations
go run cmd/migrate/main.go up

# Start service with Google FHIR integration
export GOOGLE_FHIR_PROJECT_ID="your-project"
export GOOGLE_FHIR_DATASET_ID="clinical-data"
export GOOGLE_FHIR_STORE_ID="medication-fhir"
go run cmd/server/main.go
```

### Testing Strategy
- **Unit Tests**: Individual component testing
- **Integration Tests**: Database and FHIR integration testing
- **Performance Tests**: Load testing with realistic data volumes
- **End-to-End Tests**: Complete workflow validation

## Benefits of This Architecture

### 1. **Operational Excellence**
- Complete workflow state management
- Real-time performance monitoring
- Comprehensive audit trails
- Advanced error handling and recovery

### 2. **Performance Optimization**
- <250ms end-to-end execution
- Smart caching and prefetching
- Parallel FHIR operations
- Database query optimization

### 3. **Clinical Safety**
- Recipe & Snapshot integrity
- Clinical calculation validation
- Safety assessment tracking
- Evidence-based decision support

### 4. **Compliance and Security**
- HIPAA-compliant audit trails
- Data minimization principles
- Role-based access controls
- Comprehensive logging

### 5. **Scalability and Maintainability**
- Microservices architecture
- Database migration system
- Configurable performance targets
- Extensible integration patterns

This architecture provides a comprehensive, performant, and compliant foundation for clinical medication management while properly leveraging Google FHIR Healthcare API as the authoritative source for clinical data.