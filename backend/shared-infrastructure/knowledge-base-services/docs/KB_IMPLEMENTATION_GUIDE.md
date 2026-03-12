# 📚 Knowledge Base Services Complete Implementation Guide

## 📋 Table of Contents
1. [Executive Summary](#executive-summary)
2. [Current State Assessment](#current-state-assessment)
3. [Phase 0: Evidence Envelope Foundation](#phase-0-evidence-envelope-foundation-weeks-1-2)
4. [Phase 1: Bedrock Knowledge Bases](#phase-1-bedrock-knowledge-bases-weeks-3-6)
5. [Phase 2: Logic & Calculation KBs](#phase-2-logic--calculation-kbs-weeks-7-10)
6. [Phase 3: Operational KB](#phase-3-operational-kb-weeks-11-12)
7. [Phase 4: Integration & Validation](#phase-4-integration--validation-weeks-13-16)
8. [Technical Implementation Details](#technical-implementation-details)
9. [Testing & Validation Strategy](#testing--validation-strategy)
10. [Deployment & Operations](#deployment--operations)
11. [Risk Management](#risk-management)
12. [Success Metrics](#success-metrics)

---

## 🎯 Executive Summary

This guide provides a comprehensive 16-week implementation roadmap for building and integrating 7 Knowledge Base microservices with Evidence Envelope infrastructure. The system will provide clinical intelligence for the Flow2 orchestrator, Clinical Assertion Engine (CAE), and Safety Gateway Platform.

**📋 Related Documentation:**
- [Database Architecture](./KB_DATABASE_ARCHITECTURE.md) - Comprehensive database schemas and technology choices
- [Docker PostgreSQL Setup](./DOCKER_POSTGRES_KB_SETUP.md) - Docker infrastructure configuration

### Key Objectives
- ✅ Implement 7 specialized Knowledge Base services
- ✅ Establish Evidence Envelope for complete audit trails
- ✅ Achieve P95 latency < 10ms across all services
- ✅ Enable clinical governance with digital signatures
- ✅ Support multi-region compliance (FDA/EMA/TGA)
- ✅ Integrate with existing 4-phase workflow

### Timeline Overview
- **Weeks 1-2**: Evidence Envelope Foundation (Critical Path)
- **Weeks 3-6**: Bedrock Knowledge Bases
- **Weeks 7-10**: Logic & Calculation KBs
- **Weeks 11-12**: Operational KB
- **Weeks 13-16**: Integration, Validation & Deployment

---

## 📊 Current State Assessment

### Existing Infrastructure
| Component | Status | Technology | Port | Notes |
|-----------|--------|------------|------|-------|
| KB-Drug-Rules | ✅ Partial | Go/Gin | 8081 | TOML rules implemented |
| KB-Guideline-Evidence | ✅ Partial | Go/Gin | 8083 | Basic structure ready |
| PostgreSQL | ✅ Ready | v15 | 5433 | Docker configured |
| Redis | ✅ Ready | v7 | 6380 | Cache layer active |
| Docker Infrastructure | ✅ Ready | Compose | - | Development ready |

### Services to Implement
| Service | Priority | Complexity | Dependencies |
|---------|----------|------------|--------------|
| Evidence Envelope | 🔴 Critical | High | None - Foundation |
| KB-Terminology | 🔴 Critical | Medium | Evidence Envelope |
| KB-Guidelines (Neo4j) | 🟡 High | High | Terminology |
| KB-Patient-Safety | 🟡 High | High | Terminology |
| KB-Clinical-Context | 🟡 High | Medium | Terminology |
| KB-DDI | 🟡 High | Medium | Terminology |
| KB-Formulary | 🟢 Medium | Medium | All above |

### Database Architecture Overview

**🗃️ Technology Stack by Service** (detailed in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md)):

| KB Service | Primary Database | Cache Layer | Special Features |
|------------|------------------|-------------|------------------|
| Evidence Envelope | PostgreSQL | Redis | Foundation for all services |
| KB-1: Dosing Rules | PostgreSQL + JSONB | Redis + In-memory | TOML rule engine |
| KB-2: Clinical Context | MongoDB | Redis | Document store for complex contexts |
| KB-3: Guidelines | Neo4j | Redis | Graph relationships |
| KB-4: Patient Safety | TimescaleDB | Redis | Time-series analytics |
| KB-5: DDI | PostgreSQL | Redis + In-memory | High-frequency lookups |
| KB-6: Formulary | PostgreSQL + Elasticsearch | Redis | Full-text search |
| KB-7: Terminology | PostgreSQL | Redis + In-memory | Hierarchical data |

---

## 🏗️ Phase 0: Evidence Envelope Foundation (Weeks 1-2)

### ⚠️ CRITICAL: Must Complete Before Any Other Work

### Week 1 Checklist: Database & Core Infrastructure

#### Database Setup
- [ ] Create new PostgreSQL database `clinical_governance`
- [ ] Implement Evidence Envelope schema
- [ ] Set up table partitioning for performance
- [ ] Create indexes for query optimization
- [ ] Configure replication for high availability

**📋 Complete schema specifications in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md)**

```sql
-- Evidence Envelope Foundation Schema
CREATE DATABASE clinical_governance;

\c clinical_governance;

-- Version management table
CREATE TABLE kb_version_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_active_per_env EXCLUDE (environment WITH =) WHERE (active = true)
);

-- Evidence tracking for each transaction
CREATE TABLE evidence_envelopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(100) UNIQUE NOT NULL,
    
    -- Version snapshot at time of transaction
    version_set_id UUID REFERENCES kb_version_sets(id),
    kb_versions JSONB NOT NULL,
    
    -- Decision tracking
    decision_chain JSONB NOT NULL DEFAULT '[]',
    /* Structure:
    [
      {
        "phase": "ORB",
        "timestamp": "2024-01-15T10:30:00Z",
        "kb_calls": ["kb_3_guidelines", "kb_7_terminology"],
        "duration_ms": 45,
        "decisions": [...]
      }
    ]
    */
    
    safety_attestations JSONB NOT NULL DEFAULT '[]',
    performance_metrics JSONB,
    
    -- Clinical context
    patient_id VARCHAR(100),
    encounter_id VARCHAR(100),
    clinical_domain VARCHAR(50),
    request_type VARCHAR(50),
    
    -- Orchestration metadata
    orchestrator_version VARCHAR(50),
    orchestrator_node VARCHAR(100),
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    total_duration_ms INTEGER,
    
    -- Immutability
    checksum VARCHAR(64) NOT NULL,
    signed BOOLEAN DEFAULT FALSE,
    signature TEXT,
    
    -- Partitioning key
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE evidence_envelopes_2025_01 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE evidence_envelopes_2025_02 PARTITION OF evidence_envelopes 
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
-- Continue for 12 months...

-- Indexes for performance
CREATE INDEX idx_kb_version_sets_environment ON kb_version_sets(environment);
CREATE INDEX idx_kb_version_sets_active ON kb_version_sets(active);
CREATE INDEX idx_evidence_envelopes_transaction_id ON evidence_envelopes(transaction_id);
CREATE INDEX idx_evidence_envelopes_patient_id ON evidence_envelopes(patient_id);
CREATE INDEX idx_evidence_envelopes_created_at ON evidence_envelopes(created_at);

-- Audit trail table
CREATE TABLE kb_audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    user_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB
);

CREATE INDEX idx_kb_audit_log_entity ON kb_audit_log(entity_type, entity_id);
CREATE INDEX idx_kb_audit_log_timestamp ON kb_audit_log(timestamp);
```

#### Knowledge Broker Gateway Setup
- [ ] Create new service directory `knowledge-broker-gateway`
- [ ] Implement GraphQL federation with Apollo Gateway
- [ ] Add version-aware data sources
- [ ] Implement Evidence Envelope plugin
- [ ] Add transaction tracking middleware
- [ ] Set up audit logging

### Week 2 Checklist: Integration & Tooling

#### Orchestrator Integration
- [ ] Update Flow2 orchestrator to initialize Evidence Envelopes
- [ ] Add version tracking to all KB API calls
- [ ] Implement decision chain recording
- [ ] Add checksum validation
- [ ] Create rollback mechanisms

#### kbctl CLI Tool Development
- [ ] Initialize Go CLI project structure
- [ ] Implement `validate` command for Universal Framework
- [ ] Add `deploy` command with version management
- [ ] Create `rollback` command for emergencies
- [ ] Add `health` command for service monitoring
- [ ] Implement `audit` command for compliance

#### Testing Infrastructure
- [ ] Set up integration test framework
- [ ] Create Evidence Envelope validation tests
- [ ] Implement version consistency checks
- [ ] Add performance benchmarks
- [ ] Create chaos testing scenarios

---

## 🔧 Phase 1: Bedrock Knowledge Bases (Weeks 3-6)

### Week 3: KB-7 Terminology Service

#### Implementation Checklist
- [ ] **Database Setup**
  - [ ] Create terminology database schema
  - [ ] Set up full-text search indexes
  - [ ] Configure PostgreSQL text search
  - [ ] Implement partitioning for large datasets

- [ ] **Data Ingestion Pipeline**
  - [ ] Download RxNorm dataset
  - [ ] Download LOINC codes
  - [ ] Download SNOMED CT subset
  - [ ] Download ICD-10 codes
  - [ ] Create ETL pipeline for each source
  - [ ] Implement incremental update mechanism

- [ ] **API Development**
  - [ ] Create Go service structure
  - [ ] Implement terminology lookup endpoint
  - [ ] Add code mapping functionality
  - [ ] Create search API with fuzzy matching
  - [ ] Add batch lookup capability

- [ ] **Caching Layer**
  - [ ] Configure Redis for terminology cache
  - [ ] Implement cache warming strategy
  - [ ] Add cache invalidation logic
  - [ ] Set up TTL policies

#### Database Schema

**📋 Complete schema with hierarchical data structures in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-7-terminology-service-database)**

```sql
-- KB-7 Terminology Database
CREATE DATABASE kb_terminology;

\c kb_terminology;

-- Core terminology table
CREATE TABLE terminology_concepts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system VARCHAR(50) NOT NULL,
    code VARCHAR(100) NOT NULL,
    version VARCHAR(20),
    preferred_term TEXT NOT NULL,
    synonyms TEXT[],
    definition TEXT,
    status VARCHAR(20) DEFAULT 'active',
    hierarchy_path TEXT[],
    parent_codes TEXT[],
    child_codes TEXT[],
    search_terms TSVECTOR,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(system, code, version)
);

-- Drug terminology specific
CREATE TABLE drug_terminology (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rxnorm_cui VARCHAR(20) UNIQUE NOT NULL,
    drug_name TEXT NOT NULL,
    generic_name TEXT,
    brand_names TEXT[],
    drug_type VARCHAR(50),
    drug_class TEXT[],
    active_ingredients JSONB,
    atc_codes TEXT[],
    ndc_codes TEXT[],
    search_vector TSVECTOR,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Code mappings between systems
CREATE TABLE code_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(50) NOT NULL,
    source_code VARCHAR(100) NOT NULL,
    source_version VARCHAR(20),
    target_system VARCHAR(50) NOT NULL,
    target_code VARCHAR(100) NOT NULL,
    target_version VARCHAR(20),
    mapping_type VARCHAR(50),
    confidence_score DECIMAL(3,2),
    validated BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    INDEX idx_mapping_source (source_system, source_code),
    INDEX idx_mapping_target (target_system, target_code)
);

-- Lab reference ranges
CREATE TABLE lab_reference_ranges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loinc_code VARCHAR(20) NOT NULL,
    test_name TEXT NOT NULL,
    specimen_type VARCHAR(50),
    unit VARCHAR(20),
    reference_range_low DECIMAL,
    reference_range_high DECIMAL,
    critical_low DECIMAL,
    critical_high DECIMAL,
    age_min INTEGER,
    age_max INTEGER,
    sex VARCHAR(10),
    conditions JSONB,
    source VARCHAR(100),
    version VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    INDEX idx_lab_loinc (loinc_code)
);

-- Full-text search indexes
CREATE INDEX idx_terminology_search ON terminology_concepts USING GIN(search_terms);
CREATE INDEX idx_drug_search ON drug_terminology USING GIN(search_vector);

-- Update search vectors trigger
CREATE OR REPLACE FUNCTION update_terminology_search_vector() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_terms := to_tsvector('english', 
        NEW.preferred_term || ' ' || 
        COALESCE(array_to_string(NEW.synonyms, ' '), '') || ' ' ||
        NEW.code
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_terminology_search 
    BEFORE INSERT OR UPDATE ON terminology_concepts
    FOR EACH ROW EXECUTE FUNCTION update_terminology_search_vector();
```

### Week 4: KB-3 Guidelines Enhancement (Neo4j)

#### Implementation Checklist
- [ ] **Neo4j Setup**
  - [ ] Install Neo4j database
  - [ ] Configure cluster for HA
  - [ ] Set up backup strategy
  - [ ] Create security policies

- [ ] **Data Model Design**
  - [ ] Design guideline graph schema
  - [ ] Create node types (Guideline, Recommendation, Evidence)
  - [ ] Define relationship types
  - [ ] Plan traversal patterns

- [ ] **Guideline Import**
  - [ ] Parse ACC/AHA guidelines
  - [ ] Parse ESC guidelines
  - [ ] Create import scripts
  - [ ] Validate graph integrity

- [ ] **API Enhancement**
  - [ ] Integrate Neo4j driver
  - [ ] Create pathway traversal API
  - [ ] Add recommendation ranking
  - [ ] Implement evidence grading

#### Neo4j Schema

**📋 Complete Neo4j graph schema with relationships in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-3-clinical-guidelines-database)**

```cypher
// Create constraints for data integrity
CREATE CONSTRAINT guideline_id IF NOT EXISTS 
    FOR (g:Guideline) REQUIRE g.id IS UNIQUE;

CREATE CONSTRAINT recommendation_id IF NOT EXISTS 
    FOR (r:Recommendation) REQUIRE r.id IS UNIQUE;

CREATE CONSTRAINT evidence_id IF NOT EXISTS 
    FOR (e:Evidence) REQUIRE e.id IS UNIQUE;

// Create indexes for performance
CREATE INDEX guideline_condition IF NOT EXISTS 
    FOR (g:Guideline) ON (g.condition);

CREATE INDEX recommendation_grade IF NOT EXISTS 
    FOR (r:Recommendation) ON (r.grade);

CREATE INDEX evidence_level IF NOT EXISTS 
    FOR (e:Evidence) ON (e.level);

// Sample guideline import
MERGE (g:Guideline {
    id: 'ACC_AHA_HTN_2017',
    title: '2017 ACC/AHA Guideline for High Blood Pressure',
    publisher: 'ACC/AHA',
    publication_date: date('2017-11-13'),
    condition: 'Hypertension',
    version: '2017.1',
    status: 'active'
})

MERGE (r1:Recommendation {
    id: 'HTN_REC_001',
    text: 'Initiate antihypertensive therapy for Stage 2 HTN',
    grade: 'I',
    level_of_evidence: 'A',
    applies_to: ['stage_2_hypertension']
})

MERGE (r2:Recommendation {
    id: 'HTN_REC_002',
    text: 'Use ACEi/ARB for HTN with CKD',
    grade: 'I',
    level_of_evidence: 'B',
    applies_to: ['hypertension', 'ckd']
})

MERGE (e1:Evidence {
    id: 'EVIDENCE_001',
    study_type: 'RCT',
    pmid: '28146533',
    summary: 'SPRINT trial demonstrates benefit of intensive BP control',
    quality_score: 0.95
})

// Create relationships
MERGE (g)-[:CONTAINS]->(r1)
MERGE (g)-[:CONTAINS]->(r2)
MERGE (r1)-[:SUPPORTED_BY]->(e1)
MERGE (r1)-[:FOLLOWED_BY {condition: 'if_ckd_present'}]->(r2)

// Create decision pathway
MATCH (r:Recommendation)-[:APPLIES_TO]->(:Condition {name: 'hypertension'})
WITH r
ORDER BY r.priority DESC
CREATE (pathway:ClinicalPathway {
    id: 'HTN_PATHWAY_001',
    name: 'Hypertension Management Pathway',
    created_at: datetime()
})
FOREACH (rec IN collect(r) |
    MERGE (pathway)-[:INCLUDES]->(rec)
)
```

### Week 5-6: KB-4 Patient Safety Service

#### Implementation Checklist
- [ ] **TimescaleDB Setup**
  - [ ] Install TimescaleDB extension
  - [ ] Create hypertables for time-series data
  - [ ] Configure continuous aggregates
  - [ ] Set up retention policies

- [ ] **Kafka Integration**
  - [ ] Configure Kafka topics
  - [ ] Create safety event producers
  - [ ] Implement stream processors
  - [ ] Set up consumer groups

- [ ] **Safety Rules Engine**
  - [ ] Define safety rule schema
  - [ ] Implement rule evaluation
  - [ ] Create alert generation
  - [ ] Add risk scoring algorithms

- [ ] **API Development**
  - [ ] Create safety profile endpoint
  - [ ] Add real-time alert API
  - [ ] Implement risk assessment
  - [ ] Add historical query support

#### TimescaleDB Schema

**📋 Complete TimescaleDB setup with continuous aggregates in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-4-patient-safety-database)**

```sql
-- KB-4 Patient Safety Database
CREATE DATABASE kb_patient_safety;

\c kb_patient_safety;

-- Enable TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Safety alerts time-series table
CREATE TABLE safety_alerts (
    time TIMESTAMPTZ NOT NULL,
    alert_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    description TEXT,
    source_system VARCHAR(50),
    triggering_values JSONB,
    recommendations JSONB,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    metadata JSONB
);

-- Convert to hypertable
SELECT create_hypertable('safety_alerts', 'time', 
    chunk_time_interval => INTERVAL '1 day');

-- Create indexes
CREATE INDEX idx_safety_alerts_patient ON safety_alerts(patient_id, time DESC);
CREATE INDEX idx_safety_alerts_type ON safety_alerts(alert_type, time DESC);
CREATE INDEX idx_safety_alerts_severity ON safety_alerts(severity, time DESC);

-- Patient risk profiles
CREATE TABLE patient_risk_profiles (
    patient_id VARCHAR(100) PRIMARY KEY,
    risk_scores JSONB NOT NULL DEFAULT '{}',
    /* Structure:
    {
      "fall_risk": 0.75,
      "readmission_risk": 0.45,
      "adverse_drug_event_risk": 0.30,
      "mortality_risk": 0.15
    }
    */
    risk_factors JSONB,
    contraindications TEXT[],
    safety_flags JSONB,
    last_calculated TIMESTAMPTZ DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Safety rules repository
CREATE TABLE safety_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(200) UNIQUE NOT NULL,
    rule_type VARCHAR(50),
    condition_logic JSONB NOT NULL,
    action_logic JSONB NOT NULL,
    severity VARCHAR(20),
    active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Continuous aggregate for hourly alert summary
CREATE MATERIALIZED VIEW safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients,
    AVG(EXTRACT(EPOCH FROM (acknowledged_at - time))/60)::NUMERIC(10,2) as avg_ack_time_minutes
FROM safety_alerts
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY hour, alert_type, severity
WITH NO DATA;

-- Refresh policy
SELECT add_continuous_aggregate_policy('safety_alerts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Retention policy (keep raw data for 90 days)
SELECT add_retention_policy('safety_alerts', INTERVAL '90 days');
```

---

## 💻 Phase 2: Logic & Calculation KBs (Weeks 7-10)

### Week 7-8: KB-1 Enhanced Drug Rules (Rust Integration)

#### Implementation Checklist
- [ ] **Rust Engine Development**
  - [ ] Set up Rust project structure
  - [ ] Implement dose calculation engine
  - [ ] Add safety bounds validation
  - [ ] Create expression evaluator
  - [ ] Build FFI for Go integration

- [ ] **Enhanced Caching**
  - [ ] Implement in-memory cache (DashMap)
  - [ ] Add Redis integration
  - [ ] Create cache warming
  - [ ] Add invalidation strategy

- [ ] **Performance Optimization**
  - [ ] Profile critical paths
  - [ ] Optimize calculations
  - [ ] Add parallel processing
  - [ ] Implement connection pooling

#### Rust Implementation
```rust
// kb-drug-rules/engine/src/lib.rs
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use redis::AsyncCommands;
use dashmap::DashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseCalculationRequest {
    pub drug_id: String,
    pub patient_weight_kg: f64,
    pub patient_age_years: u8,
    pub renal_function: RenalFunction,
    pub hepatic_function: HepaticFunction,
    pub concurrent_medications: Vec<String>,
    pub indication: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalFunction {
    pub egfr: f64,
    pub creatinine: f64,
    pub ckd_stage: Option<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticFunction {
    pub child_pugh_score: Option<String>,
    pub alt: f64,
    pub ast: f64,
    pub bilirubin: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseRecommendation {
    pub drug_id: String,
    pub calculated_dose: f64,
    pub dose_unit: String,
    pub frequency: String,
    pub route: String,
    pub adjustments_applied: Vec<DoseAdjustment>,
    pub warnings: Vec<String>,
    pub contraindications: Vec<String>,
    pub monitoring_parameters: Vec<String>,
    pub confidence_score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseAdjustment {
    pub adjustment_type: String,
    pub reason: String,
    pub factor: f64,
    pub source: String,
}

pub struct DoseCalculationEngine {
    rules_cache: DashMap<String, DrugRule>,
    redis_client: redis::Client,
    db_pool: sqlx::PgPool,
}

impl DoseCalculationEngine {
    pub async fn new(redis_url: &str, database_url: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let redis_client = redis::Client::open(redis_url)?;
        let db_pool = sqlx::PgPool::connect(database_url).await?;
        
        Ok(Self {
            rules_cache: DashMap::new(),
            redis_client,
            db_pool,
        })
    }
    
    pub async fn calculate_dose(&self, request: DoseCalculationRequest) -> Result<DoseRecommendation, Box<dyn std::error::Error>> {
        // Try cache first
        let cache_key = format!("dose:{}:{}:{}", 
            request.drug_id, 
            request.patient_weight_kg,
            request.renal_function.egfr
        );
        
        // Check Redis cache
        let mut conn = self.redis_client.get_async_connection().await?;
        if let Ok(cached) = conn.get::<_, String>(&cache_key).await {
            if let Ok(recommendation) = serde_json::from_str::<DoseRecommendation>(&cached) {
                return Ok(recommendation);
            }
        }
        
        // Get drug rule
        let rule = self.get_drug_rule(&request.drug_id).await?;
        
        // Calculate base dose
        let mut dose = self.calculate_base_dose(&rule, request.patient_weight_kg);
        let mut adjustments = Vec::new();
        let mut warnings = Vec::new();
        
        // Apply renal adjustment
        if request.renal_function.egfr < 60.0 {
            let renal_factor = self.calculate_renal_adjustment(request.renal_function.egfr);
            dose *= renal_factor;
            adjustments.push(DoseAdjustment {
                adjustment_type: "renal".to_string(),
                reason: format!("eGFR = {:.1} mL/min", request.renal_function.egfr),
                factor: renal_factor,
                source: "Cockcroft-Gault equation".to_string(),
            });
            
            if request.renal_function.egfr < 30.0 {
                warnings.push("Severe renal impairment - use with caution".to_string());
            }
        }
        
        // Apply hepatic adjustment
        if let Some(child_pugh) = &request.hepatic_function.child_pugh_score {
            let hepatic_factor = match child_pugh.as_str() {
                "A" => 1.0,
                "B" => 0.75,
                "C" => 0.50,
                _ => 1.0,
            };
            
            if hepatic_factor < 1.0 {
                dose *= hepatic_factor;
                adjustments.push(DoseAdjustment {
                    adjustment_type: "hepatic".to_string(),
                    reason: format!("Child-Pugh {}", child_pugh),
                    factor: hepatic_factor,
                    source: "Hepatic impairment guidelines".to_string(),
                });
            }
        }
        
        // Apply age adjustment
        if request.patient_age_years > 65 {
            let age_factor = if request.patient_age_years > 75 { 0.75 } else { 0.85 };
            dose *= age_factor;
            adjustments.push(DoseAdjustment {
                adjustment_type: "age".to_string(),
                reason: format!("Age {} years", request.patient_age_years),
                factor: age_factor,
                source: "Geriatric dosing guidelines".to_string(),
            });
        }
        
        // Apply safety bounds
        dose = dose.max(rule.min_dose).min(rule.max_dose);
        
        // Round to practical dose
        dose = self.round_to_practical_dose(dose, &rule.available_strengths);
        
        let recommendation = DoseRecommendation {
            drug_id: request.drug_id.clone(),
            calculated_dose: dose,
            dose_unit: rule.dose_unit.clone(),
            frequency: self.determine_frequency(&rule, &request),
            route: rule.route.clone(),
            adjustments_applied: adjustments,
            warnings,
            contraindications: self.check_contraindications(&rule, &request),
            monitoring_parameters: rule.monitoring_parameters.clone(),
            confidence_score: 0.95,
        };
        
        // Cache result
        let _: () = conn.setex(
            &cache_key,
            3600,
            serde_json::to_string(&recommendation)?
        ).await?;
        
        Ok(recommendation)
    }
    
    fn calculate_base_dose(&self, rule: &DrugRule, weight_kg: f64) -> f64 {
        match rule.dose_calculation_method.as_str() {
            "weight_based" => weight_kg * rule.dose_per_kg,
            "bsa_based" => {
                let bsa = self.calculate_bsa(weight_kg);
                bsa * rule.dose_per_m2
            },
            "fixed" => rule.standard_dose,
            _ => rule.standard_dose,
        }
    }
    
    fn calculate_renal_adjustment(&self, egfr: f64) -> f64 {
        if egfr >= 60.0 {
            1.0
        } else if egfr >= 30.0 {
            0.75
        } else if egfr >= 15.0 {
            0.50
        } else {
            0.25
        }
    }
    
    fn round_to_practical_dose(&self, dose: f64, available_strengths: &[f64]) -> f64 {
        available_strengths
            .iter()
            .min_by_key(|&&strength| ((dose - strength).abs() * 1000.0) as i64)
            .copied()
            .unwrap_or(dose)
    }
}
```

### Week 8-9: KB-2 Clinical Context Service

#### Implementation Checklist
- [ ] **MongoDB Setup**
  - [ ] Install MongoDB cluster
  - [ ] Configure replica sets
  - [ ] Set up sharding
  - [ ] Create backup strategy

- [ ] **Phenotype Engine**
  - [ ] Design phenotype rules
  - [ ] Implement rule evaluation
  - [ ] Create context assembly
  - [ ] Add temporal reasoning

- [ ] **Integration Points**
  - [ ] Connect to patient data
  - [ ] Link to terminology service
  - [ ] Add to orchestrator
  - [ ] Create caching layer

#### MongoDB Schema

**📋 Complete MongoDB collections with validation in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-2-clinical-context-database)**

```javascript
// KB-2 Clinical Context - MongoDB Collections

// Phenotype definitions collection
db.createCollection("phenotype_definitions", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["phenotype_id", "name", "version", "criteria", "status"],
      properties: {
        phenotype_id: {
          bsonType: "string",
          description: "Unique identifier for phenotype"
        },
        name: {
          bsonType: "string",
          description: "Human-readable phenotype name"
        },
        version: {
          bsonType: "string",
          pattern: "^\\d+\\.\\d+\\.\\d+$"
        },
        description: {
          bsonType: "string"
        },
        criteria: {
          bsonType: "object",
          description: "Phenotype detection criteria",
          properties: {
            required_conditions: {
              bsonType: "array",
              items: {
                bsonType: "object",
                properties: {
                  type: { bsonType: "string" },
                  codes: { bsonType: "array" },
                  time_window: { bsonType: "string" },
                  min_occurrences: { bsonType: "int" }
                }
              }
            },
            required_labs: {
              bsonType: "array",
              items: {
                bsonType: "object",
                properties: {
                  loinc_code: { bsonType: "string" },
                  operator: { bsonType: "string" },
                  value: { bsonType: "number" },
                  unit: { bsonType: "string" },
                  time_window: { bsonType: "string" }
                }
              }
            },
            required_medications: {
              bsonType: "array",
              items: {
                bsonType: "object",
                properties: {
                  rxnorm_codes: { bsonType: "array" },
                  duration_days: { bsonType: "int" }
                }
              }
            },
            exclusion_criteria: {
              bsonType: "array"
            }
          }
        },
        clinical_significance: {
          bsonType: "object",
          properties: {
            risk_implications: { bsonType: "array" },
            treatment_modifications: { bsonType: "array" },
            monitoring_requirements: { bsonType: "array" }
          }
        },
        status: {
          enum: ["active", "draft", "deprecated"]
        },
        created_at: {
          bsonType: "date"
        },
        updated_at: {
          bsonType: "date"
        }
      }
    }
  }
});

// Create indexes
db.phenotype_definitions.createIndex({ "phenotype_id": 1, "version": -1 });
db.phenotype_definitions.createIndex({ "status": 1 });
db.phenotype_definitions.createIndex({ "criteria.required_conditions.codes": 1 });

// Patient context collection
db.createCollection("patient_contexts", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["patient_id", "context_id", "timestamp"],
      properties: {
        patient_id: { bsonType: "string" },
        context_id: { bsonType: "string" },
        timestamp: { bsonType: "date" },
        
        demographics: {
          bsonType: "object",
          properties: {
            age_years: { bsonType: "int" },
            sex: { enum: ["M", "F", "Other"] },
            race: { bsonType: "string" },
            ethnicity: { bsonType: "string" }
          }
        },
        
        active_conditions: {
          bsonType: "array",
          items: {
            bsonType: "object",
            properties: {
              code: { bsonType: "string" },
              system: { bsonType: "string" },
              name: { bsonType: "string" },
              onset_date: { bsonType: "date" },
              severity: { bsonType: "string" }
            }
          }
        },
        
        recent_labs: {
          bsonType: "array",
          items: {
            bsonType: "object",
            properties: {
              loinc_code: { bsonType: "string" },
              value: { bsonType: "number" },
              unit: { bsonType: "string" },
              result_date: { bsonType: "date" },
              abnormal_flag: { bsonType: "string" }
            }
          }
        },
        
        current_medications: {
          bsonType: "array",
          items: {
            bsonType: "object",
            properties: {
              rxnorm_code: { bsonType: "string" },
              name: { bsonType: "string" },
              dose: { bsonType: "string" },
              frequency: { bsonType: "string" },
              start_date: { bsonType: "date" }
            }
          }
        },
        
        detected_phenotypes: {
          bsonType: "array",
          items: {
            bsonType: "object",
            properties: {
              phenotype_id: { bsonType: "string" },
              confidence: { bsonType: "double" },
              detected_at: { bsonType: "date" },
              supporting_evidence: { bsonType: "array" }
            }
          }
        },
        
        risk_factors: {
          bsonType: "object"
        },
        
        care_gaps: {
          bsonType: "array"
        },
        
        ttl: {
          bsonType: "date",
          description: "Time to live for context cache"
        }
      }
    }
  }
});

// Create indexes for performance
db.patient_contexts.createIndex({ "patient_id": 1, "timestamp": -1 });
db.patient_contexts.createIndex({ "context_id": 1 }, { unique: true });
db.patient_contexts.createIndex({ "ttl": 1 }, { expireAfterSeconds: 0 });
db.patient_contexts.createIndex({ "detected_phenotypes.phenotype_id": 1 });
```

### Week 9-10: KB-5 DDI Service

#### Implementation Checklist
- [ ] **Database Design**
  - [ ] Create interaction matrix table
  - [ ] Optimize for batch queries
  - [ ] Add severity indexing
  - [ ] Implement caching strategy

- [ ] **Interaction Engine**
  - [ ] Build interaction checker
  - [ ] Add management strategies
  - [ ] Create severity classifier
  - [ ] Implement batch processing

- [ ] **Clinical Integration**
  - [ ] Add to CAE workflow
  - [ ] Connect to Safety Gateway
  - [ ] Create alert generation
  - [ ] Add override capability

#### PostgreSQL Schema

**📋 Optimized DDI schema with lookup cache in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-5-drug-drug-interactions-database)**

```sql
-- KB-5 DDI Database
CREATE DATABASE kb_ddi;

\c kb_ddi;

-- Main interaction table
CREATE TABLE drug_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug1_rxnorm VARCHAR(20) NOT NULL,
    drug1_name TEXT NOT NULL,
    drug2_rxnorm VARCHAR(20) NOT NULL,
    drug2_name TEXT NOT NULL,
    
    -- Interaction details
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('contraindicated', 'major', 'moderate', 'minor')),
    reliability VARCHAR(20) CHECK (reliability IN ('established', 'probable', 'suspected', 'possible')),
    
    -- Clinical information
    mechanism TEXT NOT NULL,
    clinical_effects TEXT NOT NULL,
    
    -- Management
    management_strategy JSONB NOT NULL,
    /* Structure:
    {
      "action": "monitor|adjust_dose|avoid|separate_administration",
      "monitoring_parameters": ["INR", "drug_levels"],
      "dose_adjustment": {
        "drug": "drug1|drug2",
        "factor": 0.5,
        "max_dose": 100
      },
      "timing_separation": {
        "hours": 2,
        "instructions": "Take drug2 2 hours after drug1"
      },
      "alternatives": ["drug_id1", "drug_id2"]
    }
    */
    
    -- Evidence
    evidence_level VARCHAR(20),
    references JSONB DEFAULT '[]',
    
    -- Pharmacokinetics
    onset VARCHAR(50),
    offset VARCHAR(50),
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_by VARCHAR(100),
    review_date DATE,
    
    -- Ensure unique drug pairs
    CONSTRAINT unique_drug_pair UNIQUE (drug1_rxnorm, drug2_rxnorm),
    CONSTRAINT drug_order CHECK (drug1_rxnorm <= drug2_rxnorm)
);

-- Optimized interaction matrix for fast lookups
CREATE TABLE interaction_matrix (
    drug1_code VARCHAR(20) NOT NULL,
    drug2_code VARCHAR(20) NOT NULL,
    severity_score INTEGER NOT NULL, -- 1=minor, 2=moderate, 3=major, 4=contraindicated
    interaction_id UUID REFERENCES drug_interactions(id),
    PRIMARY KEY (drug1_code, drug2_code)
);

-- Drug class interactions
CREATE TABLE drug_class_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class1 VARCHAR(100) NOT NULL,
    class2 VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    mechanism TEXT,
    clinical_significance TEXT,
    applies_to_all BOOLEAN DEFAULT FALSE,
    exceptions JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Interaction check history for analytics
CREATE TABLE interaction_checks (
    id BIGSERIAL PRIMARY KEY,
    check_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100),
    checked_at TIMESTAMPTZ DEFAULT NOW(),
    drug_list JSONB NOT NULL,
    interactions_found JSONB,
    max_severity VARCHAR(20),
    action_taken VARCHAR(50),
    override_reason TEXT,
    user_id VARCHAR(100)
);

-- Create indexes for performance
CREATE INDEX idx_interactions_drug1 ON drug_interactions(drug1_rxnorm);
CREATE INDEX idx_interactions_drug2 ON drug_interactions(drug2_rxnorm);
CREATE INDEX idx_interactions_severity ON drug_interactions(severity);
CREATE INDEX idx_matrix_drugs ON interaction_matrix USING btree(drug1_code, drug2_code);
CREATE INDEX idx_matrix_severity ON interaction_matrix(severity_score);
CREATE INDEX idx_checks_patient ON interaction_checks(patient_id, checked_at DESC);

-- Optimized function for batch interaction checking
CREATE OR REPLACE FUNCTION check_drug_interactions(
    drug_list VARCHAR[],
    OUT interactions JSONB
) RETURNS JSONB AS $$
DECLARE
    interaction_data JSONB := '[]'::JSONB;
    rec RECORD;
BEGIN
    -- Find all interactions in the drug list
    FOR rec IN
        SELECT 
            di.*,
            im.severity_score
        FROM interaction_matrix im
        JOIN drug_interactions di ON di.id = im.interaction_id
        WHERE im.drug1_code = ANY(drug_list)
          AND im.drug2_code = ANY(drug_list)
          AND im.drug1_code < im.drug2_code
        ORDER BY im.severity_score DESC
    LOOP
        interaction_data := interaction_data || jsonb_build_object(
            'drug1', rec.drug1_name,
            'drug2', rec.drug2_name,
            'severity', rec.severity,
            'mechanism', rec.mechanism,
            'clinical_effects', rec.clinical_effects,
            'management', rec.management_strategy
        );
    END LOOP;
    
    interactions := interaction_data;
    RETURN;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

-- Create materialized view for common drug combinations
CREATE MATERIALIZED VIEW common_interactions AS
SELECT 
    drug1_rxnorm,
    drug2_rxnorm,
    severity,
    COUNT(*) as check_frequency
FROM interaction_checks, 
     LATERAL jsonb_array_elements(interactions_found) AS interaction
GROUP BY drug1_rxnorm, drug2_rxnorm, severity
ORDER BY check_frequency DESC;

-- Refresh materialized view daily
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('refresh-common-interactions', '0 2 * * *', 
    'REFRESH MATERIALIZED VIEW CONCURRENTLY common_interactions;');
```

---

## 🏢 Phase 3: Operational KB (Weeks 11-12)

### Week 11-12: KB-6 Formulary & Stock Service

#### Implementation Checklist
- [ ] **External Integrations**
  - [ ] Connect to insurance APIs
  - [ ] Integrate pharmacy systems
  - [ ] Link inventory management
  - [ ] Add pricing feeds

- [ ] **Stock Management**
  - [ ] Real-time inventory tracking
  - [ ] Demand prediction model
  - [ ] Reorder point calculation
  - [ ] Alternative finding logic

- [ ] **Cost Optimization**
  - [ ] Formulary tier mapping
  - [ ] Generic substitution
  - [ ] Therapeutic alternatives
  - [ ] Prior auth requirements

#### Database Schema

**📋 PostgreSQL + Elasticsearch integration in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md#kb-6-formulary-database)**

```sql
-- KB-6 Formulary Database
CREATE DATABASE kb_formulary;

\c kb_formulary;

-- Formulary entries by payer and plan
CREATE TABLE formulary_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payer_id VARCHAR(50) NOT NULL,
    payer_name VARCHAR(200),
    plan_id VARCHAR(50) NOT NULL,
    plan_name VARCHAR(200),
    plan_year INTEGER NOT NULL,
    
    -- Drug identification
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_name TEXT NOT NULL,
    drug_type VARCHAR(50), -- brand, generic, biosimilar
    
    -- Coverage details
    tier VARCHAR(20) NOT NULL,
    /* Tiers: tier1_generic, tier2_preferred_brand, 
              tier3_non_preferred, tier4_specialty, not_covered */
    status VARCHAR(20) DEFAULT 'active',
    
    -- Cost sharing
    copay_amount DECIMAL(10,2),
    coinsurance_percent INTEGER,
    deductible_applies BOOLEAN DEFAULT FALSE,
    
    -- Restrictions
    prior_authorization BOOLEAN DEFAULT FALSE,
    step_therapy BOOLEAN DEFAULT FALSE,
    quantity_limit JSONB,
    /* Structure:
    {
      "max_quantity": 30,
      "per_days": 30,
      "max_fills_per_year": 12
    }
    */
    
    -- Age and gender restrictions
    age_limits JSONB,
    gender_restriction VARCHAR(10),
    
    -- Clinical requirements
    required_diagnosis_codes TEXT[],
    required_lab_values JSONB,
    
    -- Alternatives
    preferred_alternatives JSONB DEFAULT '[]',
    generic_available BOOLEAN DEFAULT FALSE,
    generic_rxnorm VARCHAR(20),
    
    -- Metadata
    effective_date DATE NOT NULL,
    termination_date DATE,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique constraint
    CONSTRAINT unique_formulary_entry 
        UNIQUE (payer_id, plan_id, drug_rxnorm, plan_year)
);

-- Stock inventory tracking
CREATE TABLE drug_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id VARCHAR(100) NOT NULL,
    location_name VARCHAR(200),
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_ndc VARCHAR(20),
    
    -- Stock levels
    quantity_on_hand INTEGER NOT NULL DEFAULT 0,
    quantity_allocated INTEGER NOT NULL DEFAULT 0,
    quantity_available INTEGER GENERATED ALWAYS AS 
        (quantity_on_hand - quantity_allocated) STORED,
    
    -- Reorder parameters
    reorder_point INTEGER,
    reorder_quantity INTEGER,
    max_stock_level INTEGER,
    
    -- Lot tracking
    lot_number VARCHAR(50),
    expiration_date DATE,
    
    -- Cost information
    unit_cost DECIMAL(10,4),
    
    -- Timestamps
    last_counted TIMESTAMPTZ,
    last_ordered TIMESTAMPTZ,
    last_received TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_location_drug_lot 
        UNIQUE (location_id, drug_rxnorm, lot_number)
);

-- Demand prediction data
CREATE TABLE demand_history (
    id BIGSERIAL PRIMARY KEY,
    location_id VARCHAR(100) NOT NULL,
    drug_rxnorm VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    quantity_dispensed INTEGER NOT NULL,
    quantity_ordered INTEGER,
    stockout_occurred BOOLEAN DEFAULT FALSE,
    
    -- Factors affecting demand
    day_of_week INTEGER,
    month INTEGER,
    is_holiday BOOLEAN DEFAULT FALSE,
    weather_impact VARCHAR(20),
    
    UNIQUE(location_id, drug_rxnorm, date)
);

-- Pricing information
CREATE TABLE drug_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_rxnorm VARCHAR(20) NOT NULL,
    drug_ndc VARCHAR(20),
    price_type VARCHAR(50) NOT NULL, -- AWP, WAC, NADAC, MAC
    price DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20),
    effective_date DATE NOT NULL,
    source VARCHAR(100),
    
    UNIQUE(drug_rxnorm, price_type, effective_date)
);

-- Create indexes
CREATE INDEX idx_formulary_payer_plan ON formulary_entries(payer_id, plan_id);
CREATE INDEX idx_formulary_drug ON formulary_entries(drug_rxnorm);
CREATE INDEX idx_formulary_tier ON formulary_entries(tier);
CREATE INDEX idx_inventory_location ON drug_inventory(location_id);
CREATE INDEX idx_inventory_drug ON drug_inventory(drug_rxnorm);
CREATE INDEX idx_inventory_available ON drug_inventory(quantity_available);
CREATE INDEX idx_demand_location_drug ON demand_history(location_id, drug_rxnorm, date DESC);

-- Function to check formulary coverage with cost calculation
CREATE OR REPLACE FUNCTION check_formulary_coverage(
    p_drug_rxnorm VARCHAR,
    p_payer_id VARCHAR,
    p_plan_id VARCHAR,
    p_quantity INTEGER DEFAULT 30
) RETURNS TABLE (
    covered BOOLEAN,
    tier VARCHAR,
    patient_cost DECIMAL,
    requires_prior_auth BOOLEAN,
    alternatives JSONB
) AS $$
DECLARE
    v_formulary RECORD;
    v_drug_price DECIMAL;
    v_patient_cost DECIMAL;
BEGIN
    -- Get formulary entry
    SELECT * INTO v_formulary
    FROM formulary_entries
    WHERE drug_rxnorm = p_drug_rxnorm
      AND payer_id = p_payer_id
      AND plan_id = p_plan_id
      AND plan_year = EXTRACT(YEAR FROM CURRENT_DATE)
      AND status = 'active'
      AND CURRENT_DATE BETWEEN effective_date AND COALESCE(termination_date, '9999-12-31');
    
    IF NOT FOUND THEN
        -- Drug not covered, return alternatives
        RETURN QUERY
        SELECT 
            FALSE AS covered,
            'not_covered'::VARCHAR AS tier,
            NULL::DECIMAL AS patient_cost,
            FALSE AS requires_prior_auth,
            (SELECT jsonb_agg(jsonb_build_object(
                'drug_rxnorm', drug_rxnorm,
                'drug_name', drug_name,
                'tier', tier
            ))
            FROM formulary_entries
            WHERE payer_id = p_payer_id
              AND plan_id = p_plan_id
              AND drug_rxnorm IN (
                  SELECT unnest(preferred_alternatives)::VARCHAR
                  FROM formulary_entries
                  WHERE drug_rxnorm = p_drug_rxnorm
              )) AS alternatives;
        RETURN;
    END IF;
    
    -- Get drug price
    SELECT price INTO v_drug_price
    FROM drug_pricing
    WHERE drug_rxnorm = p_drug_rxnorm
      AND price_type = 'AWP'
    ORDER BY effective_date DESC
    LIMIT 1;
    
    -- Calculate patient cost
    IF v_formulary.copay_amount IS NOT NULL THEN
        v_patient_cost := v_formulary.copay_amount;
    ELSIF v_formulary.coinsurance_percent IS NOT NULL THEN
        v_patient_cost := (v_drug_price * p_quantity) * (v_formulary.coinsurance_percent / 100.0);
    ELSE
        v_patient_cost := 0;
    END IF;
    
    RETURN QUERY
    SELECT 
        TRUE AS covered,
        v_formulary.tier,
        v_patient_cost,
        v_formulary.prior_authorization,
        v_formulary.preferred_alternatives;
END;
$$ LANGUAGE plpgsql;

-- Function for demand prediction
CREATE OR REPLACE FUNCTION predict_demand(
    p_location_id VARCHAR,
    p_drug_rxnorm VARCHAR,
    p_days_ahead INTEGER DEFAULT 7
) RETURNS TABLE (
    predicted_demand INTEGER,
    confidence_interval_low INTEGER,
    confidence_interval_high INTEGER,
    reorder_recommended BOOLEAN
) AS $$
DECLARE
    v_avg_daily_demand DECIMAL;
    v_std_dev DECIMAL;
    v_current_stock INTEGER;
    v_predicted_demand INTEGER;
BEGIN
    -- Calculate average daily demand (last 90 days)
    SELECT 
        AVG(quantity_dispensed),
        STDDEV(quantity_dispensed)
    INTO v_avg_daily_demand, v_std_dev
    FROM demand_history
    WHERE location_id = p_location_id
      AND drug_rxnorm = p_drug_rxnorm
      AND date >= CURRENT_DATE - INTERVAL '90 days';
    
    -- Get current stock
    SELECT quantity_available INTO v_current_stock
    FROM drug_inventory
    WHERE location_id = p_location_id
      AND drug_rxnorm = p_drug_rxnorm
    ORDER BY expiration_date
    LIMIT 1;
    
    -- Calculate prediction
    v_predicted_demand := CEIL(v_avg_daily_demand * p_days_ahead);
    
    RETURN QUERY
    SELECT 
        v_predicted_demand,
        GREATEST(0, v_predicted_demand - (2 * v_std_dev))::INTEGER,
        (v_predicted_demand + (2 * v_std_dev))::INTEGER,
        (v_current_stock < v_predicted_demand * 1.5);
END;
$$ LANGUAGE plpgsql;
```

---

## 🔬 Phase 4: Integration & Validation (Weeks 13-16)

### Week 13-14: Full System Integration

#### Integration Checklist
- [ ] **Service Connectivity**
  - [ ] Test all KB service endpoints
  - [ ] Verify Evidence Envelope flow
  - [ ] Check version consistency
  - [ ] Validate audit trails

- [ ] **Performance Testing**
  - [ ] Load test each service
  - [ ] Measure P95 latency
  - [ ] Check cache hit rates
  - [ ] Optimize slow queries

- [ ] **End-to-End Workflows**
  - [ ] Test hypertension with CKD
  - [ ] Validate diabetes pathway
  - [ ] Check polypharmacy scenarios
  - [ ] Verify safety alerts

#### Integration Test Suite
```python
# tests/integration/test_full_workflow.py
import pytest
import asyncio
import aiohttp
import time
from typing import Dict, List
import json

class TestKnowledgeBaseIntegration:
    """Complete integration test suite for all 7 KB services"""
    
    @pytest.fixture
    async def kb_clients(self):
        """Initialize clients for all KB services"""
        return {
            'drug_rules': KBClient('http://localhost:8081'),
            'context': KBClient('http://localhost:8082'),
            'guidelines': KBClient('http://localhost:8083'),
            'safety': KBClient('http://localhost:8084'),
            'ddi': KBClient('http://localhost:8085'),
            'formulary': KBClient('http://localhost:8086'),
            'terminology': KBClient('http://localhost:8087'),
            'broker': KBClient('http://localhost:4000')  # Knowledge Broker
        }
    
    @pytest.mark.asyncio
    async def test_evidence_envelope_creation(self, kb_clients):
        """Test Evidence Envelope initialization and tracking"""
        
        # Initialize transaction
        envelope = await kb_clients['broker'].init_transaction({
            'patient_id': 'test_patient_001',
            'request_type': 'medication_recommendation',
            'clinical_domain': 'hypertension'
        })
        
        assert envelope['transaction_id'] is not None
        assert envelope['kb_versions'] is not None
        assert len(envelope['kb_versions']) == 7
        
        # Verify version format
        for kb, version in envelope['kb_versions'].items():
            assert '+sha.' in version  # Contains git SHA
            assert version.count('.') >= 2  # Semantic version
    
    @pytest.mark.asyncio
    async def test_complete_hypertension_workflow(self, kb_clients):
        """Test complete HTN workflow with all KB services"""
        
        # Patient context
        patient = {
            'id': 'test_001',
            'age': 68,
            'sex': 'M',
            'weight_kg': 85,
            'conditions': ['hypertension', 'ckd_stage_3a', 'diabetes_type_2'],
            'labs': {
                'egfr': 52,
                'creatinine': 1.4,
                'potassium': 4.2,
                'glucose': 145,
                'hba1c': 7.2,
                'uacr': 45
            },
            'current_medications': [
                {'rxnorm': '316049', 'name': 'Metformin 500mg BID'},
                {'rxnorm': '197361', 'name': 'Aspirin 81mg daily'}
            ],
            'allergies': ['sulfa'],
            'insurance': {
                'payer_id': 'aetna',
                'plan_id': 'standard_2025'
            }
        }
        
        # Initialize Evidence Envelope
        envelope = await kb_clients['broker'].init_transaction({
            'patient_id': patient['id'],
            'request_type': 'medication_recommendation',
            'clinical_domain': 'hypertension'
        })
        transaction_id = envelope['transaction_id']
        
        # Step 1: Get clinical guidelines (KB-3)
        guidelines = await kb_clients['guidelines'].get_guidelines({
            'condition': 'hypertension',
            'comorbidities': patient['conditions'],
            'transaction_id': transaction_id
        })
        
        assert guidelines['recommendation']['drug_class'] in ['ACE_INHIBITOR', 'ARB']
        assert guidelines['evidence_grade'] == 'A'
        
        # Step 2: Build clinical context (KB-2)
        context = await kb_clients['context'].build_context({
            'patient': patient,
            'transaction_id': transaction_id
        })
        
        assert 'ckd_with_albuminuria' in context['phenotypes']
        assert context['risk_scores']['cardiovascular_risk'] > 0.2
        
        # Step 3: Get drug recommendations with dose calculation (KB-1)
        drug_rules = await kb_clients['drug_rules'].get_rules({
            'drug_id': 'lisinopril',
            'region': 'US',
            'transaction_id': transaction_id
        })
        
        dose_recommendation = await kb_clients['drug_rules'].calculate_dose({
            'drug_id': 'lisinopril',
            'patient_weight_kg': patient['weight_kg'],
            'renal_function': {
                'egfr': patient['labs']['egfr'],
                'creatinine': patient['labs']['creatinine']
            },
            'transaction_id': transaction_id
        })
        
        # Verify renal adjustment applied
        assert dose_recommendation['adjustments_applied']
        assert any(adj['type'] == 'renal' for adj in dose_recommendation['adjustments_applied'])
        assert dose_recommendation['calculated_dose'] <= 20  # Max dose for CKD
        
        # Step 4: Check drug interactions (KB-5)
        interactions = await kb_clients['ddi'].check_interactions({
            'active_medications': [med['rxnorm'] for med in patient['current_medications']],
            'candidate_drug': '29046',  # Lisinopril RxNorm
            'transaction_id': transaction_id
        })
        
        assert interactions['max_severity'] in ['none', 'minor']
        assert interactions['safe_to_proceed']
        
        # Step 5: Generate safety profile (KB-4)
        safety_profile = await kb_clients['safety'].generate_profile({
            'patient': patient,
            'proposed_medication': 'lisinopril',
            'transaction_id': transaction_id
        })
        
        assert safety_profile['contraindications'] == []  # No contraindications
        assert 'hyperkalemia' in safety_profile['monitoring_required']
        
        # Step 6: Check formulary coverage (KB-6)
        coverage = await kb_clients['formulary'].check_coverage({
            'drug_rxnorm': '29046',
            'payer_id': patient['insurance']['payer_id'],
            'plan_id': patient['insurance']['plan_id'],
            'transaction_id': transaction_id
        })
        
        assert coverage['covered']
        assert coverage['tier'] in ['tier1_generic', 'tier2_preferred_brand']
        assert coverage['patient_cost'] < 50
        
        # Step 7: Verify Evidence Envelope completeness
        final_envelope = await kb_clients['broker'].get_envelope(transaction_id)
        
        assert len(final_envelope['decision_chain']) >= 6
        assert final_envelope['kb_versions'] == envelope['kb_versions']
        assert final_envelope['checksum'] is not None
        
        # Verify all KB services were called
        kb_calls = set()
        for decision in final_envelope['decision_chain']:
            kb_calls.update(decision['kb_calls'])
        
        assert len(kb_calls) >= 6  # At least 6 different KBs used
    
    @pytest.mark.asyncio
    async def test_performance_requirements(self, kb_clients):
        """Test that all services meet performance requirements"""
        
        latencies = []
        cache_hits = 0
        total_requests = 100
        
        for i in range(total_requests):
            start = time.time()
            
            # Make parallel requests to all services
            tasks = [
                kb_clients['drug_rules'].get_rules({'drug_id': 'metformin'}),
                kb_clients['terminology'].lookup({'term': 'hypertension'}),
                kb_clients['guidelines'].get_guidelines({'condition': 'diabetes'}),
                kb_clients['safety'].check_safety({'drug': 'warfarin'}),
                kb_clients['ddi'].check_interactions({
                    'drugs': ['metformin', 'lisinopril']
                }),
                kb_clients['formulary'].check_coverage({
                    'drug': 'metformin',
                    'payer': 'aetna'
                })
            ]
            
            results = await asyncio.gather(*tasks)
            
            end = time.time()
            latencies.append((end - start) * 1000)  # Convert to ms
            
            # Check cache headers
            for result in results:
                if result.get('cache_status') == 'hit':
                    cache_hits += 1
        
        # Calculate metrics
        latencies.sort()
        p95_latency = latencies[int(len(latencies) * 0.95)]
        p99_latency = latencies[int(len(latencies) * 0.99)]
        cache_hit_rate = cache_hits / (total_requests * 6)  # 6 services
        
        # Verify performance requirements
        assert p95_latency < 10, f"P95 latency {p95_latency}ms exceeds 10ms target"
        assert p99_latency < 25, f"P99 latency {p99_latency}ms exceeds 25ms target"
        assert cache_hit_rate > 0.95, f"Cache hit rate {cache_hit_rate} below 95% target"
    
    @pytest.mark.asyncio
    async def test_clinical_safety_checks(self, kb_clients):
        """Test critical safety scenarios"""
        
        # Test 1: Contraindicated drug interaction
        interaction_result = await kb_clients['ddi'].check_interactions({
            'active_medications': ['857169'],  # Simvastatin
            'candidate_drug': '392151'  # Clarithromycin (contraindicated combo)
        })
        
        assert interaction_result['severity'] == 'contraindicated'
        assert not interaction_result['safe_to_proceed']
        assert 'rhabdomyolysis' in interaction_result['clinical_effects'].lower()
        
        # Test 2: Renal dosing for severe impairment
        dose_calc = await kb_clients['drug_rules'].calculate_dose({
            'drug_id': 'metformin',
            'renal_function': {'egfr': 25}  # Severe impairment
        })
        
        assert 'contraindicated' in dose_calc.get('warnings', []) or \
               dose_calc['calculated_dose'] == 0
        
        # Test 3: Pregnancy contraindication
        safety_profile = await kb_clients['safety'].generate_profile({
            'patient': {
                'sex': 'F',
                'age': 28,
                'pregnancy_status': 'pregnant',
                'trimester': 1
            },
            'proposed_medication': 'warfarin'  # Category X drug
        })
        
        assert 'pregnancy' in safety_profile['contraindications']
        assert safety_profile['risk_category'] == 'X'
    
    @pytest.mark.asyncio
    async def test_version_consistency(self, kb_clients):
        """Test that version sets remain consistent across transactions"""
        
        # Create multiple transactions
        envelopes = []
        for i in range(5):
            envelope = await kb_clients['broker'].init_transaction({
                'patient_id': f'patient_{i}',
                'request_type': 'test'
            })
            envelopes.append(envelope)
            
            # Make some KB calls
            await kb_clients['drug_rules'].get_rules({
                'drug_id': 'metformin',
                'transaction_id': envelope['transaction_id']
            })
        
        # All transactions should use the same version set
        version_sets = [e['kb_versions'] for e in envelopes]
        assert all(vs == version_sets[0] for vs in version_sets)
    
    @pytest.mark.asyncio 
    async def test_rollback_capability(self, kb_clients):
        """Test that version sets can be rolled back"""
        
        # Get current active version set
        current_version = await kb_clients['broker'].get_active_version_set()
        
        # Deploy new version set (simulation)
        new_version = await kb_clients['broker'].deploy_version_set({
            'kb_versions': {
                'kb_1_dosing': '3.3.0+sha.abc123',
                # ... other versions
            }
        })
        
        # Simulate issue detection
        await asyncio.sleep(1)
        
        # Rollback to previous version
        rollback_result = await kb_clients['broker'].rollback_version_set(
            current_version['id']
        )
        
        assert rollback_result['success']
        assert rollback_result['active_version_set'] == current_version['id']

class KBClient:
    """Helper client for KB service communication"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url
        self.session = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.session.close()
    
    async def request(self, method: str, endpoint: str, **kwargs) -> Dict:
        if not self.session:
            self.session = aiohttp.ClientSession()
        
        url = f"{self.base_url}{endpoint}"
        async with self.session.request(method, url, **kwargs) as response:
            return await response.json()
    
    async def get_rules(self, params: Dict) -> Dict:
        return await self.request('GET', '/v1/items/metformin', params=params)
    
    # Add other methods as needed...
```

### Week 15: Clinical Validation

#### Validation Checklist
- [ ] **Shadow Mode Testing**
  - [ ] Run parallel to production
  - [ ] Compare with clinician decisions
  - [ ] Measure agreement rates
  - [ ] Identify discrepancies

- [ ] **Safety Validation**
  - [ ] Test high-risk scenarios
  - [ ] Verify contraindications
  - [ ] Check dose limits
  - [ ] Validate interaction checking

- [ ] **Clinical Accuracy**
  - [ ] Guideline adherence
  - [ ] Appropriate recommendations
  - [ ] Evidence quality
  - [ ] Outcome predictions

### Week 16: Production Deployment

#### Deployment Checklist
- [ ] **Pre-Deployment**
  - [ ] Final security scan
  - [ ] Performance benchmarks
  - [ ] Backup procedures tested
  - [ ] Rollback plan ready

- [ ] **Progressive Rollout**
  - [ ] Deploy to 5% traffic
  - [ ] Monitor for 24 hours
  - [ ] Expand to 25%
  - [ ] Monitor for 48 hours
  - [ ] Full rollout to 100%

- [ ] **Post-Deployment**
  - [ ] Monitor metrics
  - [ ] Check error rates
  - [ ] Verify audit trails
  - [ ] Clinical feedback loop

---

## 🔧 Technical Implementation Details

**📋 Database Implementation:** Complete schemas, performance optimizations, and migration strategies are detailed in [KB_DATABASE_ARCHITECTURE.md](./KB_DATABASE_ARCHITECTURE.md).

### Evidence Envelope Integration

#### Knowledge Broker Gateway Service
```typescript
// knowledge-broker-gateway/src/index.ts
import { ApolloServer } from '@apollo/server';
import { ApolloGateway, RemoteGraphQLDataSource } from '@apollo/gateway';
import { buildSubgraphSchema } from '@apollo/subgraph';

interface EvidenceEnvelope {
  id: string;
  transactionId: string;
  kbVersions: Map<string, string>;
  decisionChain: DecisionNode[];
  startTime: Date;
  checksum?: string;
}

interface DecisionNode {
  phase: string;
  timestamp: Date;
  kbCalls: string[];
  decisions: any[];
  durationMs: number;
}

class VersionAwareDataSource extends RemoteGraphQLDataSource {
  constructor(private config: DataSourceConfig) {
    super();
  }
  
  willSendRequest({ request, context }) {
    // Inject version headers
    const envelope = context.evidenceEnvelope;
    const kbVersion = envelope.kbVersions.get(this.config.name);
    
    request.http.headers.set('x-kb-version', kbVersion);
    request.http.headers.set('x-transaction-id', envelope.transactionId);
    request.http.headers.set('x-envelope-id', envelope.id);
  }
  
  async didReceiveResponse({ response, request, context }) {
    // Record KB call in decision chain
    const envelope = context.evidenceEnvelope;
    const responseTime = Date.now() - parseInt(
      request.http.headers.get('x-request-start')
    );
    
    envelope.decisionChain[envelope.decisionChain.length - 1].kbCalls.push({
      service: this.config.name,
      version: response.http.headers.get('x-kb-actual-version'),
      latencyMs: responseTime,
      cacheHit: response.http.headers.get('x-cache-hit') === 'true'
    });
    
    return response;
  }
}

class KnowledgeBrokerGateway {
  private activeVersionSet: KBVersionSet;
  private gateway: ApolloGateway;
  
  async initialize() {
    // Load active version set from database
    this.activeVersionSet = await this.loadActiveVersionSet();
    
    // Configure gateway with version-aware data sources
    this.gateway = new ApolloGateway({
      supergraphSdl: await this.buildSupergraphSchema(),
      buildService: ({ name, url }) => {
        return new VersionAwareDataSource({
          url,
          name,
          versionSet: this.activeVersionSet
        });
      }
    });
    
    // Create Apollo Server
    const server = new ApolloServer({
      gateway: this.gateway,
      
      plugins: [
        new EvidenceEnvelopePlugin(),
        new VersionManagementPlugin(),
        new AuditLoggingPlugin(),
        new MetricsPlugin()
      ],
      
      context: async ({ req }) => {
        // Initialize Evidence Envelope for this transaction
        const envelope = await this.initializeEnvelope(req);
        
        return {
          evidenceEnvelope: envelope,
          startTime: Date.now()
        };
      }
    });
    
    await server.listen({ port: 4000 });
    console.log('Knowledge Broker Gateway ready at http://localhost:4000');
  }
  
  private async initializeEnvelope(req: any): Promise<EvidenceEnvelope> {
    const envelope: EvidenceEnvelope = {
      id: generateUUID(),
      transactionId: generateTransactionId(),
      kbVersions: new Map(Object.entries(this.activeVersionSet.versions)),
      decisionChain: [],
      startTime: new Date()
    };
    
    // Store in database for audit trail
    await this.persistEnvelope(envelope);
    
    return envelope;
  }
  
  private async loadActiveVersionSet(): Promise<KBVersionSet> {
    const result = await db.query(`
      SELECT kb_versions 
      FROM kb_version_sets 
      WHERE environment = $1 AND active = true
    `, [process.env.ENVIRONMENT || 'development']);
    
    if (!result.rows[0]) {
      throw new Error('No active version set found');
    }
    
    return {
      id: result.rows[0].id,
      versions: result.rows[0].kb_versions
    };
  }
}

// Evidence Envelope Plugin
class EvidenceEnvelopePlugin {
  async requestDidStart() {
    return {
      async willSendResponse(requestContext) {
        const envelope = requestContext.contextValue.evidenceEnvelope;
        
        // Calculate checksum
        envelope.checksum = calculateChecksum(envelope);
        
        // Persist complete envelope
        await persistCompleteEnvelope(envelope);
        
        // Add envelope ID to response headers
        requestContext.response.http.headers.set(
          'x-evidence-envelope-id',
          envelope.id
        );
      }
    };
  }
}
```

### Orchestrator Integration
```go
// orchestrator/evidence_envelope.go
package orchestrator

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"
)

type EvidenceEnvelope struct {
    ID            string                 `json:"id"`
    TransactionID string                 `json:"transaction_id"`
    VersionSetID  string                 `json:"version_set_id"`
    KBVersions    map[string]string      `json:"kb_versions"`
    DecisionChain []DecisionNode         `json:"decision_chain"`
    StartTime     time.Time              `json:"start_time"`
    EndTime       *time.Time             `json:"end_time,omitempty"`
    Checksum      string                 `json:"checksum,omitempty"`
}

type DecisionNode struct {
    Phase         string                 `json:"phase"`
    Timestamp     time.Time              `json:"timestamp"`
    Input         interface{}            `json:"input"`
    Output        interface{}            `json:"output"`
    KBCalls       []KBCall               `json:"kb_calls"`
    DurationMs    int64                  `json:"duration_ms"`
}

type KBCall struct {
    Service       string                 `json:"service"`
    Method        string                 `json:"method"`
    Version       string                 `json:"version"`
    LatencyMs     int64                  `json:"latency_ms"`
    CacheHit      bool                   `json:"cache_hit"`
}

func (o *Orchestrator) InitializeTransaction(
    ctx context.Context, 
    request ClinicalRequest,
) (*EvidenceEnvelope, error) {
    // Get active version set
    versionSet, err := o.db.GetActiveVersionSet(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get version set: %w", err)
    }
    
    // Create Evidence Envelope
    envelope := &EvidenceEnvelope{
        ID:            generateUUID(),
        TransactionID: generateTransactionID(),
        VersionSetID:  versionSet.ID,
        KBVersions:    versionSet.KBVersions,
        StartTime:     time.Now(),
        DecisionChain: []DecisionNode{},
    }
    
    // Store in context
    ctx = context.WithValue(ctx, "evidence_envelope", envelope)
    
    // Persist initial envelope
    if err := o.db.CreateEvidenceEnvelope(ctx, envelope); err != nil {
        return nil, fmt.Errorf("failed to create envelope: %w", err)
    }
    
    // Log transaction start
    o.logger.Info("Transaction initialized",
        "transaction_id", envelope.TransactionID,
        "version_set_id", envelope.VersionSetID,
    )
    
    return envelope, nil
}

func (o *Orchestrator) AddDecisionNode(
    ctx context.Context,
    phase string,
    input interface{},
    output interface{},
) error {
    envelope, ok := ctx.Value("evidence_envelope").(*EvidenceEnvelope)
    if !ok {
        return fmt.Errorf("no evidence envelope in context")
    }
    
    node := DecisionNode{
        Phase:      phase,
        Timestamp:  time.Now(),
        Input:      input,
        Output:     output,
        KBCalls:    []KBCall{},
        DurationMs: 0,
    }
    
    envelope.DecisionChain = append(envelope.DecisionChain, node)
    
    return nil
}

func (o *Orchestrator) FinalizeTransaction(
    ctx context.Context,
) error {
    envelope, ok := ctx.Value("evidence_envelope").(*EvidenceEnvelope)
    if !ok {
        return fmt.Errorf("no evidence envelope in context")
    }
    
    // Set end time
    now := time.Now()
    envelope.EndTime = &now
    
    // Calculate checksum
    envelope.Checksum = o.calculateChecksum(envelope)
    
    // Persist complete envelope
    if err := o.db.UpdateEvidenceEnvelope(ctx, envelope); err != nil {
        return fmt.Errorf("failed to update envelope: %w", err)
    }
    
    // Emit audit event
    o.auditLogger.LogTransaction(envelope)
    
    return nil
}

func (o *Orchestrator) calculateChecksum(envelope *EvidenceEnvelope) string {
    // Create deterministic JSON representation
    data := map[string]interface{}{
        "transaction_id": envelope.TransactionID,
        "kb_versions":    envelope.KBVersions,
        "decision_chain": envelope.DecisionChain,
    }
    
    jsonBytes, _ := json.Marshal(data)
    
    // Calculate SHA256
    hash := sha256.Sum256(jsonBytes)
    return hex.EncodeToString(hash[:])
}

// Updated Phase 1: ORB with Evidence Envelope
func (o *Orchestrator) Phase1_ORB(
    ctx context.Context,
    request ClinicalRequest,
) (*IntentManifest, error) {
    // Initialize Evidence Envelope
    envelope, err := o.InitializeTransaction(ctx, request)
    if err != nil {
        return nil, err
    }
    ctx = context.WithValue(ctx, "evidence_envelope", envelope)
    
    startTime := time.Now()
    
    // Record phase start
    o.AddDecisionNode(ctx, "ORB_START", request, nil)
    
    // Get guidelines with version tracking
    guidelineVersion := envelope.KBVersions["kb_3_guidelines"]
    guidelines, err := o.kb3Client.GetGuidelines(
        ctx,
        request.ChiefComplaint,
        guidelineVersion,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get guidelines: %w", err)
    }
    
    // Record KB call
    envelope.DecisionChain[len(envelope.DecisionChain)-1].KBCalls = append(
        envelope.DecisionChain[len(envelope.DecisionChain)-1].KBCalls,
        KBCall{
            Service:   "kb_3_guidelines",
            Method:    "GetGuidelines",
            Version:   guidelineVersion,
            LatencyMs: time.Since(startTime).Milliseconds(),
            CacheHit:  false, // Get from response header
        },
    )
    
    // Build intent manifest
    manifest := &IntentManifest{
        RequestID:       request.ID,
        ClinicalDomain:  determineDomain(request, guidelines),
        Classifications: classifyCondition(request, guidelines),
        Priority:        calculatePriority(request, guidelines),
    }
    
    // Record phase completion
    o.AddDecisionNode(ctx, "ORB_COMPLETE", request, manifest)
    
    return manifest, nil
}
```

---

## 🧪 Testing & Validation Strategy

### Testing Phases

#### Unit Testing
- Individual service functionality
- Business logic validation
- Error handling
- Edge cases

#### Integration Testing
- Service-to-service communication
- Evidence Envelope flow
- Version consistency
- Transaction integrity

#### Performance Testing
- Load testing (10K RPS target)
- Latency measurements (P95 < 10ms)
- Cache effectiveness (>95% hit rate)
- Resource utilization

#### Clinical Validation
- Shadow mode operation
- Clinician agreement rates
- Safety verification
- Guideline adherence

### Validation Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| API Latency P95 | < 10ms | Prometheus histograms |
| Cache Hit Rate | > 95% | Redis metrics |
| Clinician Agreement | > 95% | Shadow mode comparison |
| Safety Alert Accuracy | > 99% | True/False positive rates |
| System Availability | > 99.9% | Uptime monitoring |
| Audit Completeness | 100% | Evidence Envelope validation |

---

## 🚀 Deployment & Operations

### Deployment Strategy

#### Development Environment
```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  # Knowledge Base Services
  kb-drug-rules:
    build: ./kb-drug-rules
    ports:
      - "8081:8081"
    environment:
      - ENV=development
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/kb_drug_rules
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - postgres
      - redis

  kb-terminology:
    build: ./kb-terminology
    ports:
      - "8087:8087"
    # ... similar configuration

  # Infrastructure
  postgres:
    image: postgres:15
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-scripts:/docker-entrypoint-initdb.d
    environment:
      - POSTGRES_PASSWORD=password

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes

  neo4j:
    image: neo4j:5-enterprise
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      - NEO4J_AUTH=neo4j/password
      - NEO4J_ACCEPT_LICENSE_AGREEMENT=yes

  kafka:
    image: confluentinc/cp-kafka:latest
    # ... Kafka configuration

volumes:
  postgres_data:
  neo4j_data:
  kafka_data:
```

#### Kubernetes Production
```yaml
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kb-drug-rules
  namespace: knowledge-base
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: kb-drug-rules
  template:
    metadata:
      labels:
        app: kb-drug-rules
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8081"
    spec:
      containers:
      - name: kb-drug-rules
        image: kb-services/drug-rules:v1.0.0
        ports:
        - containerPort: 8081
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kb-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8081
          initialDelaySeconds: 5
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kb-drug-rules-hpa
  namespace: knowledge-base
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kb-drug-rules
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Monitoring & Alerting

#### Prometheus Configuration
```yaml
# prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'knowledge-base-services'
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - knowledge-base
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true

rule_files:
  - 'alerts.yml'

alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alertmanager:9093
```

#### Alert Rules
```yaml
# prometheus/alerts.yml
groups:
- name: kb-services
  rules:
  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.01
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High P95 latency detected"
      description: "P95 latency is {{ $value }}s"

  - alert: LowCacheHitRate
    expr: rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m])) < 0.95
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Cache hit rate below target"

  - alert: ServiceDown
    expr: up{job="knowledge-base-services"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "KB service is down"
```

---

## 🛡️ Risk Management

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Data quality issues | High | Medium | Validation at ingestion, dual review process |
| Service latency | High | Low | 3-tier caching, performance monitoring |
| Version inconsistency | High | Low | Evidence Envelope tracking, atomic deployments |
| Integration failures | Medium | Medium | Circuit breakers, fallback mechanisms |
| Cache invalidation | Medium | Low | Event-driven updates, TTL policies |

### Clinical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Incorrect recommendations | Critical | Low | Clinical validation, shadow mode testing |
| Missed interactions | Critical | Low | Comprehensive DDI database, severity classification |
| Dose calculation errors | Critical | Low | Safety bounds, renal/hepatic adjustments |
| Guideline conflicts | Medium | Medium | Priority ranking, clinical review |

### Operational Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Deployment failures | High | Low | Blue-green deployments, automated rollback |
| Data loss | High | Low | Regular backups, replication |
| Security breaches | Critical | Low | Encryption, access controls, audit logging |
| Regulatory non-compliance | High | Low | Evidence Envelope, complete audit trails |

---

## 📊 Success Metrics

### Technical KPIs

| Metric | Target | Measurement | Frequency |
|--------|--------|-------------|-----------|
| API Latency P95 | < 10ms | Prometheus | Real-time |
| API Latency P99 | < 25ms | Prometheus | Real-time |
| Cache Hit Rate | > 95% | Redis/Application | Real-time |
| Service Availability | > 99.9% | Uptime monitoring | Daily |
| Deployment Success Rate | > 95% | CI/CD metrics | Per deployment |
| Mean Time to Recovery | < 15 min | Incident tracking | Per incident |

### Clinical KPIs

| Metric | Target | Measurement | Frequency |
|--------|--------|-------------|-----------|
| Guideline Adherence | > 95% | Audit analysis | Weekly |
| Clinician Agreement | > 95% | Shadow mode | Daily |
| Safety Alert Accuracy | > 99% | True/False positives | Weekly |
| Dose Accuracy | > 99.9% | Clinical review | Monthly |
| Interaction Detection | > 99% | Validation testing | Weekly |

### Business KPIs

| Metric | Target | Measurement | Frequency |
|--------|--------|-------------|-----------|
| System Adoption | > 80% | Usage analytics | Monthly |
| User Satisfaction | > 4.5/5 | Surveys | Quarterly |
| Formulary Savings | > $1M/year | Cost analysis | Quarterly |
| Clinical Outcomes | Improved | Outcome tracking | Quarterly |
| Regulatory Compliance | 100% | Audit reports | Annually |

---

## 📝 Appendix

### Glossary

| Term | Definition |
|------|------------|
| Evidence Envelope | Complete audit trail of a clinical decision transaction |
| KB | Knowledge Base microservice |
| P95/P99 | 95th/99th percentile latency measurements |
| Shadow Mode | Running new system parallel to production for validation |
| Version Set | Coordinated set of KB versions deployed together |
| CAE | Clinical Assertion Engine |
| DDI | Drug-Drug Interaction |
| ORB | Orchestration Request Broker |

### References

1. [FHIR R4 Specification](https://www.hl7.org/fhir/)
2. [ACC/AHA Clinical Guidelines](https://www.acc.org/guidelines)
3. [FDA SaMD Guidance](https://www.fda.gov/medical-devices/digital-health-center-excellence/software-medical-device-samd)
4. [HIPAA Compliance Requirements](https://www.hhs.gov/hipaa/index.html)
5. [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)

### Contact Information

| Role | Name | Email | Responsibilities |
|------|------|-------|-----------------|
| Technical Lead | TBD | - | Overall architecture |
| Clinical Lead | TBD | - | Clinical validation |
| DevOps Lead | TBD | - | Deployment & operations |
| Product Owner | TBD | - | Requirements & priorities |

---

## ✅ Final Checklist

### Pre-Implementation
- [ ] Team assembled and trained
- [ ] Infrastructure provisioned
- [ ] Development environments ready
- [ ] CI/CD pipelines configured
- [ ] Security review completed

### Implementation
- [ ] Evidence Envelope foundation complete
- [ ] All 7 KB services implemented
- [ ] Integration testing passed
- [ ] Performance targets met
- [ ] Clinical validation completed

### Deployment
- [ ] Production environment ready
- [ ] Monitoring & alerting configured
- [ ] Rollback procedures tested
- [ ] Documentation complete
- [ ] Training materials prepared

### Post-Deployment
- [ ] Progressive rollout successful
- [ ] Metrics within targets
- [ ] User feedback positive
- [ ] Regulatory compliance verified
- [ ] Continuous improvement plan active

---

*This implementation guide represents a comprehensive approach to building a production-ready Knowledge Base services platform with Evidence Envelope tracking, ensuring clinical safety, regulatory compliance, and optimal performance.*

**Document Version**: 1.0.0  
**Last Updated**: 2025-01-15  
**Status**: Ready for Implementation