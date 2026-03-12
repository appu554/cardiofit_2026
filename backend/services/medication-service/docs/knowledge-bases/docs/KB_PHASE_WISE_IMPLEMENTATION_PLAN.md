# 📋 Knowledge Base Services - Phase-wise Implementation Plan

## Executive Summary
This document provides a detailed phase-wise implementation plan to complete all Knowledge Base (KB) services according to the KB Implementation Guide requirements. Based on the comprehensive gap analysis, we have identified critical missing components and prioritized them for systematic implementation.

---

## 📊 Current Implementation Status Overview

| Phase | Overall Completion | Critical Gaps | Risk Level |
|-------|-------------------|---------------|------------|
| Phase 0 | 95% | Version management, kbctl CLI | 🟡 Medium |
| Phase 1 | 25% | Database technologies, Data ingestion | 🔴 Critical |
| Phase 2 | 45% | Business logic, Integration | 🔴 High |
| Phase 3 | 20% | Search capabilities, Algorithms | 🔴 High |
| Phase 4 | 40% | Testing, Monitoring | 🟡 Medium |

---

## 🎯 Implementation Phases

## **PHASE 0 COMPLETION: Evidence Envelope & Foundation**
**Timeline: 3 Days** | **Priority: CRITICAL** | **Current: 95% Complete**

### Day 1: Fix Apollo Federation Configuration
**Objective:** Correct service mappings and ensure proper federation

#### Tasks:
- [ ] **Fix Service Names in Apollo Federation**
  ```javascript
  // Current (WRONG)
  { name: 'kb3-adverse-events', url: 'http://localhost:8083/api/federation' }
  
  // Should be (CORRECT)
  { name: 'kb3-guidelines', url: 'http://localhost:8083/api/federation' }
  ```
  
- [ ] **Update All Port Mappings**
  - KB-1: Drug Rules → 8081 ✅
  - KB-2: Clinical Context → 8082 (fix from 8082)
  - KB-3: Guidelines → 8083 (fix name)
  - KB-4: Patient Safety → 8084 (fix from analytics)
  - KB-5: DDI → 8085 (fix from outcomes)
  - KB-6: Formulary → 8086 ✅
  - KB-7: Terminology → 8087 ✅
  - Evidence Envelope → 8088 ✅

- [ ] **Test Federation Connectivity**
  ```bash
  # Test each service endpoint
  for port in 8081 8082 8083 8084 8085 8086 8087 8088; do
    echo "Testing KB service on port $port"
    curl -s http://localhost:$port/health | jq .
  done
  ```

### Day 2: Complete Evidence Envelope Versioning
**Objective:** Add missing version management capabilities

#### Database Schema Additions:
```sql
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
```

#### Service Updates:
- [ ] Update Evidence Envelope service to track versions
- [ ] Implement checksum calculation
- [ ] Add digital signature capability
- [ ] Create version validation endpoints

### Day 3: Create kbctl CLI Tool
**Objective:** Build command-line management tool

#### CLI Structure:
```go
// kb-cli/cmd/root.go
package cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "kbctl",
    Short: "KB Services management CLI",
    Long:  "Command-line tool for managing Knowledge Base services",
}

// Commands to implement:
// - kbctl validate [version-set]     # Validate KB version compatibility
// - kbctl deploy [version-set]        # Deploy new version set
// - kbctl rollback [version-id]      # Rollback to previous version
// - kbctl health [service|all]        # Check service health
// - kbctl audit [transaction-id]      # Query audit trail
// - kbctl config [get|set]           # Manage configuration
```

#### Features:
- [ ] Version set validation
- [ ] Deployment orchestration
- [ ] Health monitoring
- [ ] Audit trail queries
- [ ] Configuration management

---

## **PHASE 1: Bedrock Knowledge Bases**
**Timeline: 2 Weeks** | **Priority: CRITICAL** | **Current: 25% Complete**

### Week 1: Database Infrastructure Setup

#### **Day 1-2: Deploy Missing Databases**
**Using docker-compose.databases.yml from KB_MISSING_DATABASES_DOCKER_SETUP.md**

- [ ] **MongoDB for KB-2**
  ```bash
  docker-compose -f docker-compose.databases.yml up -d mongodb mongo-express
  docker exec -it kb-mongodb mongosh /docker-entrypoint-initdb.d/01-init-kb-clinical-context.js
  ```

- [ ] **Neo4j for KB-3**
  ```bash
  docker-compose -f docker-compose.databases.yml up -d neo4j
  docker exec -it kb-neo4j cypher-shell < init-scripts/neo4j/01-init-guidelines.cypher
  ```

- [ ] **TimescaleDB for KB-4**
  ```bash
  docker-compose -f docker-compose.databases.yml up -d timescaledb
  # Auto-initialized via docker-entrypoint-initdb.d
  ```

- [ ] **Elasticsearch for KB-6**
  ```bash
  docker-compose -f docker-compose.databases.yml up -d elasticsearch kibana
  docker exec -it kb-elasticsearch bash /usr/share/elasticsearch/init/01-init-formulary.sh
  ```

#### **Day 3-4: KB-7 Terminology Data Ingestion**

**Data Sources to Download:**
- [ ] RxNorm (https://www.nlm.nih.gov/research/umls/rxnorm/docs/rxnormfiles.html)
- [ ] LOINC (https://loinc.org/downloads/)
- [ ] SNOMED CT subset (https://www.nlm.nih.gov/healthit/snomedct/)
- [ ] ICD-10 (https://www.cms.gov/medicare/icd-10/icd-10-cm-official-guidelines-coding-reporting)

**ETL Pipeline Implementation:**
```go
// kb-7-terminology/internal/etl/rxnorm_loader.go
package etl

type RxNormLoader struct {
    db     *sql.DB
    logger *zap.Logger
}

func (r *RxNormLoader) LoadRxNormData(filePath string) error {
    // 1. Parse RRF files
    // 2. Transform to internal format
    // 3. Bulk insert into PostgreSQL
    // 4. Update search vectors
    // 5. Build code mappings
}
```

**Search Implementation:**
```sql
-- Full-text search setup
ALTER TABLE drug_terminology ADD COLUMN search_vector tsvector;

CREATE INDEX idx_drug_search ON drug_terminology USING GIN(search_vector);

-- Update search vectors
UPDATE drug_terminology 
SET search_vector = to_tsvector('english', 
    drug_name || ' ' || 
    COALESCE(generic_name, '') || ' ' || 
    COALESCE(array_to_string(brand_names, ' '), '')
);
```

#### **Day 5: KB-3 Guidelines Neo4j Integration**

**Neo4j Connection Setup:**
```go
// kb-guideline-evidence/internal/database/neo4j_connection.go
package database

import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jConnection struct {
    driver neo4j.Driver
}

func NewNeo4jConnection(uri, username, password string) (*Neo4jConnection, error) {
    driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        return nil, err
    }
    return &Neo4jConnection{driver: driver}, nil
}
```

**Graph Query Implementation:**
```go
// Recommendation traversal
func (n *Neo4jConnection) GetRecommendations(condition string) ([]Recommendation, error) {
    session := n.driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    
    result, err := session.Run(`
        MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)-[:APPLIES_TO]->(c:Condition {name: $condition})
        RETURN r.id, r.text, r.grade, r.level_of_evidence
        ORDER BY r.priority
    `, map[string]interface{}{"condition": condition})
    
    // Process results...
}
```

### Week 2: Core Functionality Implementation

#### **Day 1-2: KB-4 Patient Safety TimescaleDB**

**Time-Series Implementation:**
```go
// kb-4-patient-safety/internal/services/safety_service.go
package services

type SafetyService struct {
    db    *sql.DB
    kafka *kafka.Producer
}

func (s *SafetyService) CreateSafetyAlert(alert SafetyAlert) error {
    query := `
        INSERT INTO patient_safety.safety_alerts 
        (time, patient_id, alert_type, severity, description, triggering_values)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
    // Execute insert...
    
    // Publish to Kafka for real-time streaming
    s.kafka.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
        Value:          alertJSON,
    }, nil)
}
```

**Continuous Aggregate Setup:**
```sql
-- Real-time materialized views
CREATE MATERIALIZED VIEW safety_metrics_realtime
WITH (timescaledb.continuous, timescaledb.materialized_only=false) AS
SELECT 
    time_bucket('5 minutes', time) AS bucket,
    patient_id,
    COUNT(*) FILTER (WHERE severity = 'critical') as critical_count,
    COUNT(*) FILTER (WHERE severity = 'high') as high_count,
    AVG(EXTRACT(EPOCH FROM (acknowledged_at - time))) as avg_response_time
FROM patient_safety.safety_alerts
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY bucket, patient_id;
```

#### **Day 3-4: Clinical Guidelines Data Import**

**ACC/AHA Guidelines Import:**
```cypher
// Import ACC/AHA Hypertension Guidelines
LOAD CSV WITH HEADERS FROM 'file:///acc_aha_guidelines.csv' AS row
MERGE (g:Guideline {id: row.guideline_id})
SET g.title = row.title,
    g.publisher = row.publisher,
    g.publication_date = date(row.pub_date),
    g.version = row.version

MERGE (r:Recommendation {id: row.rec_id})
SET r.text = row.recommendation,
    r.grade = row.grade,
    r.level_of_evidence = row.evidence_level

MERGE (g)-[:CONTAINS]->(r);
```

#### **Day 5: Integration Testing**
- [ ] Test all database connections
- [ ] Verify data ingestion pipelines
- [ ] Validate search functionality
- [ ] Check graph traversals
- [ ] Confirm time-series operations

---

## **PHASE 2: Logic & Calculation KBs**
**Timeline: 2 Weeks** | **Priority: HIGH** | **Current: 45% Complete**

### Week 1: Core Business Logic

#### **Day 1-2: KB-2 Clinical Context MongoDB**

**MongoDB Connection Implementation:**
```go
// kb-2-clinical-context/internal/database/mongodb_connection.go
package database

import (
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoConnection(uri string) (*mongo.Client, error) {
    client, err := mongo.Connect(context.Background(), 
        options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }
    return client, nil
}
```

**Phenotype Detection Engine:**
```go
// kb-2-clinical-context/internal/services/phenotype_service.go
type PhenotypeEngine struct {
    db     *mongo.Database
    cache  *redis.Client
}

func (p *PhenotypeEngine) DetectPhenotypes(patient PatientData) ([]Phenotype, error) {
    // 1. Load phenotype definitions
    // 2. Apply criteria to patient data
    // 3. Calculate confidence scores
    // 4. Return detected phenotypes
    
    pipeline := mongo.Pipeline{
        {{"$match", bson.D{{"status", "active"}}}},
        {{"$project", bson.D{
            {"phenotype_id", 1},
            {"criteria", 1},
            {"confidence", bson.D{
                {"$cond", bson.A{
                    matchesCriteria(patient),
                    calculateConfidence(patient),
                    0,
                }},
            }},
        }}},
        {{"$match", bson.D{{"confidence", bson.D{{"$gt", 0.7}}}}}},
    }
    
    cursor, err := p.db.Collection("phenotype_definitions").Aggregate(
        context.Background(), pipeline)
    // Process results...
}
```

#### **Day 3-4: KB-5 DDI Enhancement**

**Interaction Matrix Implementation:**
```sql
-- Create optimized interaction matrix
CREATE TABLE drug_interaction_matrix (
    drug1_code VARCHAR(20) NOT NULL,
    drug2_code VARCHAR(20) NOT NULL,
    severity_score INTEGER NOT NULL,
    interaction_id UUID,
    PRIMARY KEY (drug1_code, drug2_code)
) PARTITION BY HASH (drug1_code);

-- Create partitions for scalability
CREATE TABLE drug_interaction_matrix_p0 PARTITION OF drug_interaction_matrix
    FOR VALUES WITH (modulus 4, remainder 0);
CREATE TABLE drug_interaction_matrix_p1 PARTITION OF drug_interaction_matrix
    FOR VALUES WITH (modulus 4, remainder 1);
-- etc...

-- Batch checking function
CREATE OR REPLACE FUNCTION check_drug_interactions_batch(
    drug_list VARCHAR[],
    OUT interactions JSONB[]
) RETURNS JSONB[] AS $$
DECLARE
    interaction_data JSONB[] := '{}';
BEGIN
    WITH drug_pairs AS (
        SELECT DISTINCT 
            LEAST(d1.drug, d2.drug) as drug1,
            GREATEST(d1.drug, d2.drug) as drug2
        FROM unnest(drug_list) d1(drug)
        CROSS JOIN unnest(drug_list) d2(drug)
        WHERE d1.drug < d2.drug
    )
    SELECT array_agg(
        jsonb_build_object(
            'drug1', di.drug1_name,
            'drug2', di.drug2_name,
            'severity', di.severity,
            'management', di.management_strategy
        )
    ) INTO interaction_data
    FROM drug_pairs dp
    JOIN drug_interactions di ON 
        (di.drug1_rxnorm = dp.drug1 AND di.drug2_rxnorm = dp.drug2);
    
    RETURN interaction_data;
END;
$$ LANGUAGE plpgsql PARALLEL SAFE;
```

#### **Day 5: KB-1 Rust Engine Integration**

**Rust Calculation Engine:**
```rust
// kb-drug-rules/engine/src/lib.rs
use std::ffi::{CStr, CString};
use std::os::raw::c_char;

#[no_mangle]
pub extern "C" fn calculate_dose_ffi(
    drug_id: *const c_char,
    weight_kg: f64,
    egfr: f64,
    age: i32,
) -> *mut c_char {
    let drug_id_str = unsafe {
        CStr::from_ptr(drug_id).to_string_lossy().into_owned()
    };
    
    let result = calculate_dose_internal(drug_id_str, weight_kg, egfr, age);
    let json_result = serde_json::to_string(&result).unwrap();
    
    CString::new(json_result).unwrap().into_raw()
}

fn calculate_dose_internal(
    drug_id: String,
    weight_kg: f64,
    egfr: f64,
    age: i32,
) -> DoseRecommendation {
    // High-performance dose calculation
    // Renal adjustments
    // Hepatic adjustments
    // Safety bounds checking
}
```

**Go FFI Integration:**
```go
// kb-drug-rules/internal/services/rust_engine.go
// #cgo LDFLAGS: -L./engine/target/release -ldose_engine
// #include <stdlib.h>
// char* calculate_dose_ffi(const char* drug_id, double weight, double egfr, int age);
import "C"
import "unsafe"

func CalculateDoseWithRust(drugID string, weight, egfr float64, age int) (*DoseRecommendation, error) {
    cDrugID := C.CString(drugID)
    defer C.free(unsafe.Pointer(cDrugID))
    
    resultPtr := C.calculate_dose_ffi(cDrugID, C.double(weight), C.double(egfr), C.int(age))
    defer C.free(unsafe.Pointer(resultPtr))
    
    resultStr := C.GoString(resultPtr)
    
    var recommendation DoseRecommendation
    err := json.Unmarshal([]byte(resultStr), &recommendation)
    return &recommendation, err
}
```

### Week 2: Advanced Features

#### **Day 1-2: Safety Rules Engine**

```go
// kb-4-patient-safety/internal/rules/safety_engine.go
type SafetyRule struct {
    ID        string
    Name      string
    Condition json.RawMessage
    Action    json.RawMessage
    Severity  string
}

type SafetyEngine struct {
    rules  []SafetyRule
    engine *govaluate.EvaluableExpression
}

func (e *SafetyEngine) EvaluatePatient(patient PatientData) ([]SafetyAlert, error) {
    alerts := []SafetyAlert{}
    
    for _, rule := range e.rules {
        // Parse condition
        expr, _ := govaluate.NewEvaluableExpression(string(rule.Condition))
        
        // Evaluate against patient data
        parameters := map[string]interface{}{
            "age":        patient.Age,
            "medications": patient.Medications,
            "labs":       patient.Labs,
            "conditions": patient.Conditions,
        }
        
        result, _ := expr.Evaluate(parameters)
        if result.(bool) {
            // Generate alert
            alert := SafetyAlert{
                RuleName:    rule.Name,
                Severity:    rule.Severity,
                PatientID:   patient.ID,
                Description: e.generateDescription(rule, patient),
                Timestamp:   time.Now(),
            }
            alerts = append(alerts, alert)
        }
    }
    
    return alerts, nil
}
```

#### **Day 3-4: Clinical Context Assembly**

```go
// kb-2-clinical-context/internal/services/context_assembly.go
type ContextAssembler struct {
    phenotypeEngine *PhenotypeEngine
    terminology     *TerminologyService
    guidelines      *GuidelineService
}

func (c *ContextAssembler) AssembleContext(patientID string) (*ClinicalContext, error) {
    // 1. Gather patient data
    patient := c.fetchPatientData(patientID)
    
    // 2. Detect phenotypes
    phenotypes := c.phenotypeEngine.DetectPhenotypes(patient)
    
    // 3. Identify risk factors
    risks := c.calculateRiskFactors(patient, phenotypes)
    
    // 4. Find care gaps
    gaps := c.identifyCareGaps(patient, phenotypes)
    
    // 5. Get relevant guidelines
    guidelines := c.guidelines.GetRelevantGuidelines(phenotypes)
    
    // 6. Assemble complete context
    context := &ClinicalContext{
        PatientID:          patientID,
        Timestamp:          time.Now(),
        Demographics:       patient.Demographics,
        ActiveConditions:   patient.Conditions,
        CurrentMedications: patient.Medications,
        DetectedPhenotypes: phenotypes,
        RiskFactors:        risks,
        CareGaps:          gaps,
        Guidelines:         guidelines,
        TTL:               time.Now().Add(24 * time.Hour),
    }
    
    // 7. Cache context
    c.cacheContext(context)
    
    return context, nil
}
```

---

## **PHASE 3: Operational KB**
**Timeline: 1 Week** | **Priority: MEDIUM** | **Current: 20% Complete**

### KB-6 Formulary Enhancement

#### **Day 1-2: Elasticsearch Integration**

```go
// kb-6-formulary/internal/search/elasticsearch.go
package search

import (
    "github.com/elastic/go-elasticsearch/v8"
)

type FormularySearch struct {
    client *elasticsearch.Client
}

func (f *FormularySearch) SearchDrugs(query string, filters FormularyFilters) ([]Drug, error) {
    // Build search query
    searchQuery := map[string]interface{}{
        "query": map[string]interface{}{
            "bool": map[string]interface{}{
                "must": []interface{}{
                    map[string]interface{}{
                        "multi_match": map[string]interface{}{
                            "query":  query,
                            "fields": []string{"drug_name^3", "generic_name^2", "brand_names"},
                            "type":   "best_fields",
                            "fuzziness": "AUTO",
                        },
                    },
                },
                "filter": buildFilters(filters),
            },
        },
        "highlight": map[string]interface{}{
            "fields": map[string]interface{}{
                "drug_name": map[string]interface{}{},
                "generic_name": map[string]interface{}{},
            },
        },
        "size": 20,
    }
    
    // Execute search
    res, err := f.client.Search(
        f.client.Search.WithIndex("formulary"),
        f.client.Search.WithBody(searchQuery),
    )
    // Process results...
}
```

#### **Day 3-4: Cost Optimization Logic**

```go
// kb-6-formulary/internal/services/cost_optimizer.go
type CostOptimizer struct {
    formulary *FormularyService
    pricing   *PricingService
}

func (c *CostOptimizer) OptimizeMedication(
    drug string, 
    insurance InsuranceInfo,
) (*OptimizedSelection, error) {
    // 1. Check current drug coverage
    coverage := c.formulary.CheckCoverage(drug, insurance)
    
    // 2. Find therapeutic alternatives
    alternatives := c.findTherapeuticAlternatives(drug)
    
    // 3. Calculate costs for each option
    options := []CostOption{}
    for _, alt := range alternatives {
        cost := c.calculatePatientCost(alt, insurance)
        options = append(options, CostOption{
            Drug:     alt,
            Cost:     cost,
            Tier:     alt.Tier,
            Savings:  coverage.Cost - cost,
        })
    }
    
    // 4. Sort by cost-effectiveness
    sort.Slice(options, func(i, j int) bool {
        return options[i].Cost < options[j].Cost
    })
    
    return &OptimizedSelection{
        CurrentDrug: drug,
        CurrentCost: coverage.Cost,
        Options:     options,
        Recommended: options[0],
    }, nil
}
```

#### **Day 5: Demand Prediction**

```python
# kb-6-formulary/ml/demand_prediction.py
import pandas as pd
import numpy as np
from prophet import Prophet
from sklearn.ensemble import RandomForestRegressor

class DemandPredictor:
    def __init__(self):
        self.prophet_model = Prophet(
            seasonality_mode='multiplicative',
            weekly_seasonality=True,
            yearly_seasonality=True
        )
        self.rf_model = RandomForestRegressor(n_estimators=100)
    
    def predict_demand(self, drug_id, location_id, days_ahead=30):
        # 1. Load historical data
        history = self.load_history(drug_id, location_id)
        
        # 2. Prepare features
        history['ds'] = pd.to_datetime(history['date'])
        history['y'] = history['quantity_dispensed']
        
        # 3. Add regressors
        history['day_of_week'] = history['ds'].dt.dayofweek
        history['month'] = history['ds'].dt.month
        history['is_holiday'] = history['is_holiday'].astype(int)
        
        # 4. Train Prophet model
        self.prophet_model.fit(history[['ds', 'y']])
        
        # 5. Make predictions
        future = self.prophet_model.make_future_dataframe(periods=days_ahead)
        forecast = self.prophet_model.predict(future)
        
        # 6. Calculate confidence intervals
        predictions = {
            'predicted_demand': forecast['yhat'].iloc[-days_ahead:].sum(),
            'lower_bound': forecast['yhat_lower'].iloc[-days_ahead:].sum(),
            'upper_bound': forecast['yhat_upper'].iloc[-days_ahead:].sum(),
            'daily_forecast': forecast[['ds', 'yhat']].iloc[-days_ahead:].to_dict('records')
        }
        
        return predictions
```

---

## **PHASE 4: Integration & Validation**
**Timeline: 1 Week** | **Priority: HIGH** | **Current: 40% Complete**

### Day 1-2: Integration Testing Suite

```python
# tests/integration/test_kb_integration.py
import pytest
import asyncio
from typing import Dict, List
import aiohttp

class TestKBIntegration:
    """Comprehensive integration test suite"""
    
    @pytest.fixture
    async def kb_services(self):
        """Initialize connections to all KB services"""
        return {
            'evidence_envelope': 'http://localhost:8088',
            'kb1_drug_rules': 'http://localhost:8081',
            'kb2_context': 'http://localhost:8082',
            'kb3_guidelines': 'http://localhost:8083',
            'kb4_safety': 'http://localhost:8084',
            'kb5_ddi': 'http://localhost:8085',
            'kb6_formulary': 'http://localhost:8086',
            'kb7_terminology': 'http://localhost:8087',
        }
    
    @pytest.mark.asyncio
    async def test_complete_workflow(self, kb_services):
        """Test complete clinical workflow"""
        
        # 1. Initialize Evidence Envelope
        envelope = await self.create_evidence_envelope()
        transaction_id = envelope['transaction_id']
        
        # 2. Patient context
        patient = {
            'id': 'TEST_001',
            'age': 65,
            'conditions': ['hypertension', 'diabetes', 'ckd_stage_3'],
            'medications': ['metformin', 'aspirin'],
            'labs': {
                'egfr': 45,
                'hba1c': 7.5,
                'potassium': 4.5
            }
        }
        
        # 3. Get clinical context (KB-2)
        context = await self.get_clinical_context(
            kb_services['kb2_context'], 
            patient, 
            transaction_id
        )
        assert 'ckd_with_diabetes' in context['phenotypes']
        
        # 4. Get guidelines (KB-3)
        guidelines = await self.get_guidelines(
            kb_services['kb3_guidelines'],
            context['phenotypes'],
            transaction_id
        )
        assert len(guidelines['recommendations']) > 0
        
        # 5. Calculate drug dose (KB-1)
        dose = await self.calculate_dose(
            kb_services['kb1_drug_rules'],
            'lisinopril',
            patient,
            transaction_id
        )
        assert dose['adjustments']['renal'] == True
        assert dose['calculated_dose'] <= 20  # Max for CKD
        
        # 6. Check interactions (KB-5)
        interactions = await self.check_interactions(
            kb_services['kb5_ddi'],
            patient['medications'] + ['lisinopril'],
            transaction_id
        )
        assert interactions['max_severity'] != 'contraindicated'
        
        # 7. Safety check (KB-4)
        safety = await self.safety_check(
            kb_services['kb4_safety'],
            patient,
            'lisinopril',
            transaction_id
        )
        assert 'hyperkalemia_monitoring' in safety['required_monitoring']
        
        # 8. Check formulary (KB-6)
        coverage = await self.check_formulary(
            kb_services['kb6_formulary'],
            'lisinopril',
            'aetna_standard',
            transaction_id
        )
        assert coverage['covered'] == True
        
        # 9. Verify Evidence Envelope
        final_envelope = await self.get_evidence_envelope(
            kb_services['evidence_envelope'],
            transaction_id
        )
        assert len(final_envelope['kb_calls']) == 6
        assert final_envelope['checksum'] is not None
```

### Day 3-4: Performance Testing

```python
# tests/performance/test_kb_performance.py
import locust
from locust import HttpUser, task, between
import random
import time

class KBPerformanceTest(HttpUser):
    wait_time = between(0.1, 0.5)
    
    def on_start(self):
        """Initialize test data"""
        self.drug_list = ['metformin', 'lisinopril', 'atorvastatin', 'metoprolol']
        self.patient_ids = [f'PATIENT_{i:04d}' for i in range(100)]
    
    @task(10)
    def test_terminology_lookup(self):
        """High-frequency terminology lookups"""
        drug = random.choice(self.drug_list)
        with self.client.get(
            f"/api/terminology/lookup?term={drug}",
            name="terminology_lookup"
        ) as response:
            assert response.status_code == 200
            assert response.elapsed.total_seconds() < 0.01  # 10ms target
    
    @task(5)
    def test_ddi_check(self):
        """Drug interaction checking"""
        drugs = random.sample(self.drug_list, 3)
        with self.client.post(
            "/api/ddi/check",
            json={"medications": drugs},
            name="ddi_check"
        ) as response:
            assert response.status_code == 200
            assert response.elapsed.total_seconds() < 0.025  # 25ms target
    
    @task(3)
    def test_safety_alert(self):
        """Safety alert generation"""
        patient_id = random.choice(self.patient_ids)
        with self.client.post(
            "/api/safety/evaluate",
            json={"patient_id": patient_id},
            name="safety_evaluation"
        ) as response:
            assert response.status_code == 200
            assert response.elapsed.total_seconds() < 0.05  # 50ms target

# Run with: locust -f test_kb_performance.py --host=http://localhost:4000
```

### Day 5: Monitoring Setup

```yaml
# monitoring/prometheus-rules.yml
groups:
  - name: kb_services
    interval: 30s
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.01
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "KB Service {{ $labels.service }} P95 latency > 10ms"
          description: "P95 latency is {{ $value }}s for {{ $labels.service }}"
      
      - alert: LowCacheHitRate
        expr: |
          rate(cache_hits_total[5m]) / 
          (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m])) < 0.95
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cache hit rate below 95% for {{ $labels.service }}"
      
      - alert: DatabaseConnectionFailure
        expr: up{job="database_exporters"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database {{ $labels.database }} is down"
      
      - alert: HighMemoryUsage
        expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Container {{ $labels.container }} memory usage > 90%"
```

```yaml
# monitoring/grafana-dashboard.json
{
  "dashboard": {
    "title": "KB Services Overview",
    "panels": [
      {
        "title": "API Latency P95",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{ service }}"
          }
        ]
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m])) * 100",
            "legendFormat": "{{ service }}"
          }
        ]
      },
      {
        "title": "Evidence Envelope Transactions/sec",
        "targets": [
          {
            "expr": "rate(evidence_envelope_transactions_total[1m])",
            "legendFormat": "Transactions"
          }
        ]
      },
      {
        "title": "Database Connections",
        "targets": [
          {
            "expr": "pg_stat_database_numbackends",
            "legendFormat": "{{ datname }}"
          }
        ]
      }
    ]
  }
}
```

---

## 📊 Success Metrics & KPIs

### Technical Metrics
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| P95 Latency | < 10ms | Prometheus histogram |
| P99 Latency | < 25ms | Prometheus histogram |
| Cache Hit Rate | > 95% | Redis/App metrics |
| Service Availability | > 99.9% | Uptime monitoring |
| Database Query Time | < 5ms | pg_stat_statements |
| Memory Usage | < 80% | Container metrics |

### Business Metrics
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Clinical Accuracy | > 95% | Shadow mode validation |
| Guideline Adherence | > 90% | Audit analysis |
| Safety Alert Precision | > 98% | False positive rate |
| Formulary Savings | > $1M/year | Cost analysis |
| User Satisfaction | > 4.5/5 | Surveys |

### Operational Metrics
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Deployment Success Rate | > 95% | CI/CD metrics |
| MTTR (Mean Time to Recovery) | < 15 min | Incident tracking |
| Audit Completeness | 100% | Evidence Envelope |
| Backup Success Rate | 100% | Backup monitoring |
| Security Scan Pass Rate | 100% | Security tools |

---

## 🚦 Go/No-Go Criteria for Production

### Phase 0 Completion ✅
- [ ] Evidence Envelope versioning complete
- [ ] kbctl CLI operational
- [ ] Apollo Federation corrected
- [ ] All services health checks passing

### Phase 1 Completion ✅
- [ ] All databases deployed and initialized
- [ ] Terminology data loaded (>1M terms)
- [ ] Neo4j guidelines imported
- [ ] TimescaleDB streaming operational

### Phase 2 Completion ✅
- [ ] MongoDB phenotype detection working
- [ ] DDI matrix populated
- [ ] Rust engine integrated
- [ ] Safety rules engine active

### Phase 3 Completion ✅
- [ ] Elasticsearch search functional
- [ ] Formulary data loaded
- [ ] Cost optimization working
- [ ] Demand prediction accurate

### Phase 4 Completion ✅
- [ ] Integration tests passing (100%)
- [ ] Performance targets met
- [ ] Monitoring dashboards active
- [ ] Documentation complete

---

## 📅 Timeline Summary

| Week | Phase | Deliverables | Completion Target |
|------|-------|--------------|-------------------|
| 1 | Phase 0 | Federation fix, Versioning, CLI | 95% → 100% |
| 2-3 | Phase 1 | Database setup, Data ingestion | 25% → 100% |
| 4-5 | Phase 2 | Business logic, Integrations | 45% → 100% |
| 6 | Phase 3 | Search, Algorithms | 20% → 100% |
| 7 | Phase 4 | Testing, Monitoring | 40% → 100% |

**Total Timeline: 7 Weeks**

---

## 🎯 Next Steps

1. **Week 1**: Complete Phase 0 (Evidence Envelope, Federation, CLI)
2. **Week 2-3**: Deploy missing databases and load data
3. **Week 4-5**: Implement core business logic
4. **Week 6**: Complete operational features
5. **Week 7**: Full integration testing and monitoring setup

**Critical Success Factors:**
- Database infrastructure must be fully operational before Phase 2
- Data ingestion pipelines must be tested and validated
- Integration testing must cover all service interactions
- Performance targets must be met before production deployment

---

*This implementation plan provides a systematic approach to completing all Knowledge Base services with clear deliverables, timelines, and success criteria.*