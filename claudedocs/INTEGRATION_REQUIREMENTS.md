# Integration Requirements Specification: FHIR/Neo4j Enrichment Pipeline

**Document Version**: 1.0
**Date**: 2025-10-16
**Project**: CardioFit Clinical Synthesis Hub
**Scope**: Integration of FHIR and Neo4j enrichment capabilities from createEnhancedPipeline into createUnifiedPipeline

---

## Executive Summary

This document specifies the requirements for integrating comprehensive FHIR and Neo4j enrichment capabilities from the legacy `createEnhancedPipeline` into the modern `createUnifiedPipeline` architecture. The integration will combine the temporal aggregation and pattern detection strengths of the unified pipeline with the rich clinical context and population health insights from FHIR/Neo4j data sources.

**Current State**: Two separate architectures exist with complementary capabilities that are not integrated.

**Target State**: A unified pipeline that provides both real-time stateful aggregation AND comprehensive clinical enrichment with external data sources.

---

## 1. Functional Requirements

### 1.1 FHIR Enrichment Capabilities

#### FR-1.1: Patient Demographics Enrichment
**Priority**: CRITICAL
**Description**: System must enrich patient context with complete demographic information from Google Healthcare FHIR API.

**Required Data Elements**:
- Patient full name (given, family)
- Date of birth
- Gender/biological sex
- Age (calculated)
- Medical record number (MRN)
- Contact information (address, phone, email)
- Emergency contacts

**Acceptance Criteria**:
- Demographics retrieved within 10 seconds per patient
- Missing demographic data logged but does not block processing
- Demographics cached in PatientContextState for subsequent events
- Demographics included in EnrichedPatientContext output

#### FR-1.2: Chronic Conditions Enrichment
**Priority**: CRITICAL
**Description**: System must retrieve and maintain patient's chronic condition history from FHIR Condition resources.

**Required Data Elements**:
- Condition SNOMED CT codes
- Condition display names
- Clinical status (active, resolved, inactive)
- Verification status
- Onset date/period
- Severity classification

**Acceptance Criteria**:
- All active conditions retrieved for patient
- Conditions prioritized by clinical relevance (chronic diseases first)
- Maximum 50 conditions per patient to prevent payload bloat
- Conditions available for clinical scoring algorithms

#### FR-1.3: Active Medications Enrichment
**Priority**: HIGH
**Description**: System must retrieve current medication list from FHIR MedicationRequest resources.

**Required Data Elements**:
- Medication name and RxNorm codes
- Dosage and frequency
- Route of administration
- Prescriber information
- Start date and expected duration
- Status (active, on-hold, stopped)

**Acceptance Criteria**:
- Active medications retrieved within 10 seconds
- Integration with aggregated medication data from event stream
- Medication interaction checking enabled
- Medications included in clinical decision support logic

#### FR-1.4: Allergy and Intolerance Enrichment
**Priority**: HIGH
**Description**: System must retrieve patient allergy and intolerance information from FHIR AllergyIntolerance resources.

**Required Data Elements**:
- Allergen identification (substance, medication, food)
- Reaction severity (mild, moderate, severe)
- Reaction manifestations (rash, anaphylaxis, etc.)
- Clinical status (active, inactive, resolved)
- Verification status

**Acceptance Criteria**:
- All active allergies retrieved for patient
- Critical allergies flagged in output
- Allergy data available for medication safety checking
- Allergies included in EnrichedPatientContext

#### FR-1.5: Care Team Enrichment
**Priority**: MEDIUM
**Description**: System must identify current care team members from FHIR CareTeam resources.

**Required Data Elements**:
- Practitioner names and roles
- Specialty information
- Contact information
- Primary care provider identification
- Care team organization

**Acceptance Criteria**:
- Care team retrieved within 10 seconds
- Primary care provider clearly identified
- Care team included in context for routing decisions

#### FR-1.6: Historical Vital Signs Enrichment
**Priority**: MEDIUM
**Description**: System must retrieve historical vital sign observations to supplement real-time aggregation.

**Required Data Elements**:
- Heart rate, blood pressure (systolic/diastolic)
- Respiratory rate, oxygen saturation
- Temperature
- Timestamp and performer information

**Acceptance Criteria**:
- Last 30 days of vital signs retrieved
- Maximum 200 observations per vital type
- Historical data merged with real-time aggregated data
- Trends calculated across historical and real-time data

#### FR-1.7: Historical Laboratory Results Enrichment
**Priority**: MEDIUM
**Description**: System must retrieve historical laboratory results from FHIR Observation resources.

**Required Data Elements**:
- Lab test LOINC codes and names
- Result values with units
- Reference ranges
- Abnormal flags
- Timestamp and ordering provider

**Acceptance Criteria**:
- Last 90 days of lab results retrieved
- Critical lab values (lactate, troponin, creatinine) prioritized
- Lab data available for clinical scoring (sepsis, MODS, ACS)
- Historical labs merged with real-time lab events

### 1.2 Neo4j Graph Data Integration

#### FR-2.1: Similar Patient Cohort Discovery
**Priority**: HIGH
**Description**: System must identify similar patients using Neo4j graph relationships based on clinical characteristics.

**Required Data Elements**:
- Similar patient identifiers
- Similarity score/basis (condition overlap, demographic similarity)
- Outcome data for similar patients
- Treatment patterns in similar cohort

**Acceptance Criteria**:
- Up to 10 similar patients identified per patient
- Similarity algorithm considers age, gender, conditions, medications
- Similar patient data cached for 24 hours
- Privacy-preserving aggregation (no PHI from other patients)

#### FR-2.2: Risk Factor Cohort Identification
**Priority**: HIGH
**Description**: System must identify population-level risk cohorts patient belongs to using Neo4j graph analytics.

**Required Data Elements**:
- Cohort names (e.g., "Urban Metabolic Syndrome Cohort")
- Cohort risk factors
- Population statistics for cohort
- Cohort-specific clinical guidelines

**Acceptance Criteria**:
- All applicable cohorts identified per patient
- Cohort membership updated when conditions change
- Cohort-specific protocols applied automatically
- Cohort analytics included in EnrichedPatientContext

#### FR-2.3: Graph-Based Clinical Recommendations
**Priority**: MEDIUM
**Description**: System must generate evidence-based recommendations using Neo4j knowledge graph relationships.

**Required Data Elements**:
- Recommendation text
- Evidence strength/source
- Applicable conditions/contexts
- Priority/urgency level

**Acceptance Criteria**:
- Recommendations relevant to current clinical state
- Evidence-based recommendations prioritized
- Recommendations integrated with protocol logic
- Maximum 5 recommendations per patient to avoid alert fatigue

#### FR-2.4: Cohort Analytics and Insights
**Priority**: MEDIUM
**Description**: System must provide population-level analytics for patient's cohorts from Neo4j.

**Required Data Elements**:
- Cohort size and characteristics
- Average risk scores for cohort
- Common comorbidities in cohort
- Outcome statistics

**Acceptance Criteria**:
- Cohort analytics retrieved within 5 seconds
- Analytics update frequency: hourly
- Analytics available for clinical decision support
- Cohort insights included in output

### 1.3 Clinical Intelligence Features

#### FR-3.1: Advanced Clinical Scoring
**Priority**: HIGH
**Description**: System must calculate comprehensive clinical risk scores using enriched FHIR data.

**Required Scores**:
- **NEWS2 Score**: Already implemented, enhance with FHIR data
- **qSOFA Score**: Already implemented, enhance with FHIR data
- **Framingham Risk Score**: Cardiovascular risk (NEW)
- **CHADS-VASC Score**: Stroke risk in atrial fibrillation (NEW)
- **Metabolic Syndrome Score**: Comprehensive metabolic assessment (NEW)

**Acceptance Criteria**:
- Scores calculated using both real-time and historical FHIR data
- Missing data handled gracefully with partial scores
- Confidence levels included for each score
- Scores recalculated on new data arrival

#### FR-3.2: Evidence-Based Protocol Recommendations
**Priority**: HIGH
**Description**: System must generate structured protocol recommendations with action items.

**Required Protocols**:
- Sepsis management protocol
- Hypertensive crisis protocol
- Acute coronary syndrome protocol
- Diabetic ketoacidosis protocol
- Respiratory distress protocol

**Acceptance Criteria**:
- Protocols triggered by clinical pattern detection
- Specific action items with priority levels
- Time-sensitive actions flagged
- Protocol adherence tracking enabled

#### FR-3.3: Clinical Pattern Detection Enhancement
**Priority**: MEDIUM
**Description**: Enhance existing pattern detection with FHIR/Neo4j context.

**Pattern Types**:
- Sepsis detection (enhance with medication history, comorbidities)
- MODS detection (enhance with historical organ function)
- ACS pattern recognition (enhance with cardiac history, risk factors)
- Deterioration prediction (enhance with baseline function)

**Acceptance Criteria**:
- Pattern detection accuracy improved by 15% with enrichment
- False positive rate reduced by 20%
- Confidence scores provided for all detections
- Historical context used for baseline comparison

#### FR-3.4: Medication Interaction Checking
**Priority**: HIGH
**Description**: System must identify potential medication interactions using FHIR medication data.

**Required Capabilities**:
- Drug-drug interaction detection
- Drug-allergy contraindication checking
- Dose range validation
- Duplicate therapy identification

**Acceptance Criteria**:
- Interactions checked against FHIR medication list
- Critical interactions flagged immediately
- Interaction severity levels provided
- Clinical decision support alerts generated

### 1.4 Data Completeness Requirements

#### FR-4.1: Complete Clinical Context
**Priority**: CRITICAL
**Description**: EnrichedPatientContext must include all available clinical data for comprehensive decision support.

**Required Sections**:
```json
{
  "patientId": "string",
  "demographics": { /* FR-1.1 */ },
  "chronicConditions": [ /* FR-1.2 */ ],
  "activeMedications": [ /* FR-1.3 */ ],
  "allergies": [ /* FR-1.4 */ ],
  "careTeam": [ /* FR-1.5 */ ],
  "vitalSignsHistory": [ /* FR-1.6 */ ],
  "labResultsHistory": [ /* FR-1.7 */ ],
  "cohortMemberships": [ /* FR-2.2 */ ],
  "similarPatients": [ /* FR-2.1 */ ],
  "patientState": { /* Existing aggregated state */ },
  "clinicalScores": { /* FR-3.1 */ },
  "protocolRecommendations": [ /* FR-3.2 */ ],
  "clinicalPatterns": { /* FR-3.3 */ },
  "medicationAlerts": [ /* FR-3.4 */ }
}
```

**Acceptance Criteria**:
- All sections populated when data available
- Graceful degradation with missing data
- Data freshness timestamps included
- Output schema validated before emission

#### FR-4.2: Data Consistency
**Priority**: HIGH
**Description**: Ensure consistency between real-time aggregated data and FHIR historical data.

**Consistency Requirements**:
- Real-time vital signs reconciled with FHIR observations
- Medication events matched with FHIR MedicationRequest
- Lab events validated against FHIR lab orders
- Temporal alignment across all data sources

**Acceptance Criteria**:
- Duplicate detection and deduplication logic
- Conflict resolution rules for discrepancies
- Data lineage tracked (source: real-time vs FHIR)
- Consistency metrics logged

---

## 2. Technical Requirements

### 2.1 Architecture Patterns

#### TR-1.1: Asynchronous I/O Processing
**Priority**: CRITICAL
**Description**: Use Flink AsyncDataStream for non-blocking external service calls.

**Technical Specifications**:
- Async I/O pattern for FHIR API calls
- Async I/O pattern for Neo4j queries
- Unordered wait strategy for parallel requests
- Timeout: 10 seconds per async operation
- Capacity: 500 concurrent async requests

**Implementation Details**:
```java
DataStream<EnrichedEvent> enriched = AsyncDataStream.unorderedWait(
    inputStream,
    new FHIREnrichmentFunction(),
    10000, // timeout milliseconds
    TimeUnit.MILLISECONDS,
    500    // capacity
).uid("fhir-enrichment-async");
```

**Acceptance Criteria**:
- No blocking calls in async functions
- Graceful timeout handling
- Request failures logged but processing continues
- Async operations monitored (latency, success rate)

#### TR-1.2: Stateful Processing with RocksDB
**Priority**: CRITICAL
**Description**: Maintain patient state across events using RocksDB state backend.

**State Management Requirements**:
- Patient demographics cached in state
- Chronic conditions cached in state
- FHIR data TTL: 24 hours (configurable)
- Neo4j data TTL: 1 hour (configurable)
- State checkpoint interval: 5 minutes

**State Schema**:
```java
public class PatientContextState {
    // Existing fields
    private String patientId;
    private Long firstEventTime;
    private Long lastEventTime;
    private Map<String, VitalTrend> vitalsTrends;
    private Map<String, LabHistory> labsHistory;

    // NEW: FHIR Enrichment Fields
    private PatientDemographics demographics;
    private Long demographicsLastFetched;
    private List<Condition> chronicConditions;
    private Long conditionsLastFetched;
    private List<Medication> activeMedications;
    private Long medicationsLastFetched;
    private List<Allergy> allergies;
    private Long allergiesLastFetched;

    // NEW: Neo4j Enrichment Fields
    private List<String> cohortMemberships;
    private Long cohortsLastFetched;
    private List<SimilarPatient> similarPatients;
    private Long similarPatientsLastFetched;
}
```

**Acceptance Criteria**:
- State serialization/deserialization working
- State recovery on failure
- TTL enforcement for cached data
- State size monitoring and pruning

#### TR-1.3: Data Flow Architecture
**Priority**: HIGH
**Description**: Define clear data flow with enrichment stages integrated into unified pipeline.

**Proposed Flow**:
```
Stage 1: Event Ingestion
CanonicalEvent (from Kafka)
  ↓
Stage 2: FHIR Enrichment (NEW)
AsyncDataStream(FHIREnrichmentFunction)
  ├─ Fetch demographics (if not in state)
  ├─ Fetch conditions (if stale)
  ├─ Fetch medications (if stale)
  └─ Fetch allergies (if stale)
  ↓
EnrichedCanonicalEvent
  ↓
Stage 3: Event Conversion
FlatMap(CanonicalEventToGenericEventConverter)
  ↓
GenericEvent
  ↓
Stage 4: Stateful Aggregation
KeyBy(patientId) → Process(PatientContextAggregator)
  ├─ Aggregate vitals over time
  ├─ Aggregate labs over time
  ├─ Aggregate medications
  ├─ Calculate trends
  └─ Merge FHIR data into state
  ↓
EnrichedPatientContext
  ↓
Stage 5: Neo4j Enrichment (NEW)
AsyncDataStream(Neo4jEnrichmentFunction)
  ├─ Find similar patients
  ├─ Identify cohort memberships
  └─ Retrieve cohort analytics
  ↓
EnrichedPatientContext (with graph data)
  ↓
Stage 6: Clinical Intelligence
Process(ClinicalIntelligenceEvaluator)
  ├─ Calculate advanced scores (Framingham, CHADS-VASC)
  ├─ Detect clinical patterns (enhanced)
  ├─ Generate protocol recommendations
  └─ Check medication interactions
  ↓
EnrichedPatientContext (final)
  ↓
Stage 7: Output
Sink → clinical-patterns.v1 (Kafka)
Optional: ProtocolEventExtractor → protocol-events.v1 (Kafka)
```

**Acceptance Criteria**:
- All stages implemented and tested
- Data lineage trackable through pipeline
- Stage metrics collected (throughput, latency)
- Failure in one stage does not block others

### 2.2 Performance Constraints

#### TR-2.1: Latency Requirements
**Priority**: HIGH
**Description**: Define acceptable latency for end-to-end processing.

**Latency Targets**:
- CanonicalEvent → EnrichedPatientContext: < 15 seconds (p95)
- FHIR enrichment per patient: < 10 seconds
- Neo4j enrichment per patient: < 5 seconds
- State aggregation: < 1 second
- Clinical intelligence evaluation: < 2 seconds

**Acceptance Criteria**:
- Latency metrics collected per stage
- p50, p95, p99 percentiles tracked
- Alerts on latency threshold breaches
- Latency optimization for critical paths

#### TR-2.2: Throughput Requirements
**Priority**: HIGH
**Description**: System must handle expected event volumes without degradation.

**Throughput Targets**:
- Events per second: 1,000 (normal), 5,000 (peak)
- Concurrent patients: 10,000 (normal), 50,000 (peak)
- FHIR API calls: 500 concurrent requests max
- Neo4j queries: 200 concurrent queries max

**Acceptance Criteria**:
- Load testing validates throughput targets
- Backpressure handling implemented
- Auto-scaling triggers configured
- Performance degradation at 80% capacity

#### TR-2.3: Resource Utilization
**Priority**: MEDIUM
**Description**: Optimize resource usage for cost-effectiveness.

**Resource Limits**:
- RocksDB state size: < 50GB per task manager
- Memory per task manager: 8GB (min), 16GB (recommended)
- CPU cores per task manager: 4 (min), 8 (recommended)
- Network bandwidth: 1Gbps per task manager

**Acceptance Criteria**:
- Resource metrics collected (CPU, memory, disk, network)
- Resource limits enforced
- Graceful degradation on resource pressure
- Cost monitoring per patient processed

### 2.3 API Integration Requirements

#### TR-3.1: Google Healthcare FHIR API Integration
**Priority**: CRITICAL
**Description**: Integrate with Google Cloud Healthcare API for FHIR resource access.

**API Specifications**:
- API Endpoint: `https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{fhirStore}/fhir`
- Authentication: Service account with Healthcare FHIR Reader role
- API Version: FHIR R4
- Rate Limits: 1,000 requests per minute per project

**Required API Operations**:
- `GET /Patient/{id}` - Fetch patient demographics
- `GET /Condition?patient={id}&clinical-status=active` - Fetch conditions
- `GET /MedicationRequest?patient={id}&status=active` - Fetch medications
- `GET /AllergyIntolerance?patient={id}&clinical-status=active` - Fetch allergies
- `GET /CareTeam?patient={id}&status=active` - Fetch care team
- `GET /Observation?patient={id}&category=vital-signs` - Fetch vitals
- `GET /Observation?patient={id}&category=laboratory` - Fetch labs

**Acceptance Criteria**:
- API client properly configured
- Authentication working with service account
- Rate limiting handled (exponential backoff)
- Error responses handled gracefully
- API latency monitored

#### TR-3.2: Neo4j Graph Database Integration
**Priority**: HIGH
**Description**: Integrate with Neo4j graph database for population health analytics.

**Database Specifications**:
- Neo4j Version: 4.4+ or 5.x
- Connection: Bolt protocol (bolt://neo4j:7687)
- Authentication: Username/password or OAuth
- Database: `cardiofit` (configurable)

**Required Cypher Queries**:
```cypher
// Find similar patients
MATCH (p:Patient {id: $patientId})-[:HAS_CONDITION]->(c:Condition)
MATCH (similar:Patient)-[:HAS_CONDITION]->(c)
WHERE similar.id <> $patientId
WITH similar, count(c) as sharedConditions
ORDER BY sharedConditions DESC
LIMIT 10
RETURN similar.id, sharedConditions

// Find cohort memberships
MATCH (p:Patient {id: $patientId})-[:BELONGS_TO]->(cohort:Cohort)
RETURN cohort.name, cohort.riskFactors, cohort.populationSize

// Get cohort analytics
MATCH (cohort:Cohort {name: $cohortName})-[:HAS_STATISTICS]->(stats:CohortStats)
RETURN stats.avgAge, stats.avgRiskScore, stats.commonComorbidities
```

**Acceptance Criteria**:
- Neo4j driver properly configured
- Connection pooling implemented
- Query timeouts configured (5 seconds)
- Query results cached appropriately
- Neo4j metrics monitored

#### TR-3.3: Error Handling and Retry Logic
**Priority**: HIGH
**Description**: Implement robust error handling for external service failures.

**Error Scenarios**:
- FHIR API timeout (10 seconds)
- FHIR API 429 rate limit
- FHIR API 404 patient not found
- Neo4j connection failure
- Neo4j query timeout (5 seconds)

**Retry Strategy**:
- Transient errors: Exponential backoff (1s, 2s, 4s)
- Rate limit errors: Wait for retry-after header
- Connection errors: Circuit breaker pattern (fail after 3 attempts)
- Data not found: Log but continue processing (no retry)

**Acceptance Criteria**:
- All error types handled explicitly
- Retry logic tested with fault injection
- Circuit breaker prevents cascade failures
- Error metrics collected per error type
- Alerts configured for error thresholds

### 2.4 Data Quality and Validation

#### TR-4.1: Input Validation
**Priority**: HIGH
**Description**: Validate all external data before integration into patient context.

**Validation Rules**:
- FHIR resources: Schema validation against R4 specification
- Patient demographics: Required fields present (id, name, DOB)
- Clinical codes: SNOMED CT, LOINC, RxNorm code validation
- Temporal data: Timestamps within valid ranges
- Neo4j results: Patient ID matching, data type validation

**Acceptance Criteria**:
- Validation library integrated
- Invalid data rejected with clear error messages
- Validation failures logged for monitoring
- Partial data accepted with warnings when appropriate

#### TR-4.2: Data Transformation
**Priority**: MEDIUM
**Description**: Transform external data formats into internal representations.

**Transformation Requirements**:
- FHIR JSON → Java POJOs (using HAPI FHIR library)
- SNOMED CT codes → human-readable display names
- LOINC codes → standardized lab test names
- Neo4j query results → Java objects
- Timestamp normalization (ISO 8601)

**Acceptance Criteria**:
- Transformation logic unit tested
- Edge cases handled (null values, missing fields)
- Transformation errors logged
- Performance impact minimal (< 100ms per record)

---

## 3. Integration Strategy

### 3.1 Phase 1: FHIR Enrichment Integration (Essential)

**Timeline**: 4-6 weeks
**Priority**: CRITICAL
**Dependencies**: Google Healthcare API access configured

#### Phase 1 Deliverables:

**D1.1: FHIREnrichmentFunction Implementation**
- Create async function for FHIR API calls
- Implement parallel fetching (demographics, conditions, medications, allergies)
- Add result caching in PatientContextState
- Unit tests for function logic

**D1.2: Integration with Unified Pipeline**
- Insert FHIREnrichmentFunction BEFORE CanonicalEventToGenericEventConverter
- Update CanonicalEvent schema to carry FHIR data
- Modify PatientContextAggregator to merge FHIR data into state
- Integration tests with mock FHIR API

**D1.3: State Schema Updates**
- Add FHIR fields to PatientContextState
- Implement TTL logic for cached data
- Add state serialization tests
- Performance testing with state backend

**D1.4: Output Schema Enhancement**
- Update EnrichedPatientContext to include FHIR data
- Validate output against clinical-patterns.v1 schema
- Update downstream consumers documentation
- End-to-end testing with real FHIR data

**Success Criteria**:
- EnrichedPatientContext includes demographics, conditions, medications, allergies
- FHIR API calls complete within 10 seconds (p95)
- State caching reduces API calls by 80%
- No regression in existing clinical pattern detection

### 3.2 Phase 2: Neo4j Enrichment Integration (High Value)

**Timeline**: 3-4 weeks
**Priority**: HIGH
**Dependencies**: Phase 1 complete, Neo4j database configured

#### Phase 2 Deliverables:

**D2.1: Neo4jEnrichmentFunction Implementation**
- Create async function for Neo4j queries
- Implement similar patient discovery query
- Implement cohort membership query
- Add Neo4j connection pool configuration

**D2.2: Integration Point Selection**
- Insert Neo4jEnrichmentFunction AFTER PatientContextAggregator
- Enrich aggregated context with graph data
- Implement caching strategy for graph queries
- Integration tests with mock Neo4j

**D2.3: State and Output Updates**
- Add Neo4j fields to PatientContextState
- Update EnrichedPatientContext with cohort data
- Implement graph data TTL (1 hour)
- Performance testing with real graph queries

**D2.4: Clinical Decision Support Enhancement**
- Integrate cohort-specific guidelines into protocol recommendations
- Use similar patient outcomes for confidence scoring
- Add graph-based recommendations to output
- Validation with clinical team

**Success Criteria**:
- Similar patients identified for 90% of patients
- Cohort memberships identified for 80% of patients
- Neo4j queries complete within 5 seconds (p95)
- Clinical recommendations improved with graph context

### 3.3 Phase 3: Advanced Clinical Scoring (Medium Priority)

**Timeline**: 2-3 weeks
**Priority**: MEDIUM
**Dependencies**: Phase 1 complete

#### Phase 3 Deliverables:

**D3.1: Framingham Risk Score Implementation**
- Add cardiovascular risk calculation to ClinicalIntelligenceEvaluator
- Use age, gender, cholesterol, HDL, blood pressure from FHIR/aggregated data
- Handle missing data with partial scoring
- Validate against published Framingham algorithm

**D3.2: CHADS-VASC Score Implementation**
- Add stroke risk calculation for atrial fibrillation patients
- Use age, gender, conditions (CHF, hypertension, diabetes, stroke history)
- Detect atrial fibrillation from vital signs or conditions
- Clinical validation with cardiologist

**D3.3: Enhanced Metabolic Syndrome Scoring**
- Comprehensive metabolic assessment using FHIR data
- Include waist circumference, triglycerides, HDL, glucose, blood pressure
- Integrate with existing metabolic syndrome logic
- Provide component-level scores

**D3.4: Protocol Recommendation Enhancement**
- Use advanced scores for protocol triggering
- Add confidence levels based on data completeness
- Generate specific action items per protocol
- Priority-based recommendation ordering

**Success Criteria**:
- Framingham score calculated for 70% of adult patients
- CHADS-VASC score calculated for 100% of AFib patients
- Protocol recommendations include specific action items
- Clinical validation shows improved decision support quality

### 3.4 Phase 4: Protocol Events and Audit Trail (Optional)

**Timeline**: 1-2 weeks
**Priority**: LOW
**Dependencies**: Phase 3 complete

#### Phase 4 Deliverables:

**D4.1: ProtocolEventExtractor Implementation**
- Create side output for protocol-specific events
- Extract protocol recommendations from EnrichedPatientContext
- Generate structured protocol events
- Add event metadata (timestamp, patient ID, protocol type)

**D4.2: Protocol Events Kafka Topic**
- Create `protocol-events.v1` Kafka topic
- Define protocol event schema
- Implement Kafka sink for protocol events
- Configure retention and partitioning

**D4.3: Audit Trail and Compliance**
- Log all protocol recommendations
- Track protocol adherence
- Generate compliance reports
- Integration with audit system

**D4.4: Downstream Integration**
- Document protocol events schema
- Update downstream consumers
- Create sample protocol event processors
- Alerting and notification integration

**Success Criteria**:
- Protocol events extracted for all recommendations
- Events persisted to Kafka with < 1 second latency
- Audit trail queryable by patient, protocol, time range
- Downstream systems successfully consume protocol events

---

## 4. Success Criteria

### 4.1 Data Completeness Metrics

#### SC-1.1: FHIR Data Completeness
**Measurement**: Percentage of EnrichedPatientContext records with complete FHIR data.

**Targets**:
- Demographics present: > 95% of patients
- Chronic conditions present: > 90% of patients (exclude new patients)
- Active medications present: > 85% of patients
- Allergies present: > 80% of patients (many patients have no documented allergies)
- Care team present: > 70% of patients

**Monitoring**:
- Data completeness dashboard in Grafana
- Daily completeness reports
- Alerts on completeness drops below thresholds

#### SC-1.2: Neo4j Data Completeness
**Measurement**: Percentage of patients with graph enrichment data.

**Targets**:
- Similar patients identified: > 80% of patients
- Cohort memberships identified: > 70% of patients
- Cohort analytics available: > 90% of cohorts

**Monitoring**:
- Graph data availability metrics
- Neo4j query success rate
- Cache hit rate for graph queries

#### SC-1.3: Clinical Score Calculation Rate
**Measurement**: Percentage of patients with calculated clinical scores.

**Targets**:
- NEWS2 score: > 98% (requires vitals only)
- qSOFA score: > 95% (requires vitals only)
- Framingham score: > 70% (requires lab data)
- CHADS-VASC score: > 95% of AFib patients
- Metabolic syndrome score: > 60% (requires comprehensive data)

**Monitoring**:
- Score calculation success rate per score type
- Missing data reasons tracked
- Data quality improvement over time

### 4.2 Performance Benchmarks

#### SC-2.1: Latency Performance
**Measurement**: Processing latency from CanonicalEvent ingestion to EnrichedPatientContext output.

**Targets**:
| Stage | p50 | p95 | p99 |
|-------|-----|-----|-----|
| FHIR Enrichment | 2s | 8s | 12s |
| Event Conversion | 50ms | 100ms | 200ms |
| State Aggregation | 200ms | 500ms | 1s |
| Neo4j Enrichment | 1s | 4s | 6s |
| Clinical Intelligence | 500ms | 1.5s | 2.5s |
| **End-to-End** | **5s** | **12s** | **18s** |

**Monitoring**:
- Per-stage latency metrics in Prometheus
- Latency percentile dashboards
- Alerts on p95 latency > 15 seconds

#### SC-2.2: Throughput Performance
**Measurement**: Events processed per second by the integrated pipeline.

**Targets**:
- Normal load: 1,000 events/second with < 5s latency
- Peak load: 5,000 events/second with < 15s latency
- Sustained load: 2,000 events/second for 24 hours

**Monitoring**:
- Throughput dashboard (events/sec, patients/sec)
- Backpressure indicators
- Kafka lag monitoring

#### SC-2.3: API Call Efficiency
**Measurement**: Effectiveness of caching and API call reduction.

**Targets**:
- FHIR API cache hit rate: > 80%
- Neo4j query cache hit rate: > 70%
- Average FHIR API calls per patient: < 5 per hour
- Average Neo4j queries per patient: < 2 per hour

**Monitoring**:
- Cache hit/miss rates per data type
- API call volume and rate limiting incidents
- Cache effectiveness dashboard

### 4.3 Clinical Intelligence Validation

#### SC-3.1: Clinical Scoring Accuracy
**Measurement**: Validation of clinical scores against gold standard calculations.

**Validation Method**:
- Retrospective analysis with 1,000 patient records
- Manual calculation by clinical team
- Comparison with established calculators

**Targets**:
- NEWS2 score accuracy: > 98% exact match
- qSOFA score accuracy: > 98% exact match
- Framingham score accuracy: > 95% within 1% error
- CHADS-VASC score accuracy: > 98% exact match

**Validation Plan**:
- Phase 1: Validate with test dataset before production
- Phase 2: Ongoing monitoring with random sampling
- Phase 3: Clinical review of discrepancies

#### SC-3.2: Pattern Detection Improvement
**Measurement**: Improvement in clinical pattern detection with FHIR/Neo4j enrichment.

**Metrics**:
- Sepsis detection sensitivity: Increase by 15%
- Sepsis detection specificity: Maintain > 90%
- ACS detection accuracy: Increase by 20%
- False positive rate: Reduce by 20% overall

**Validation Method**:
- A/B testing: With enrichment vs without
- Clinical outcome correlation
- Receiver Operating Characteristic (ROC) analysis

**Targets**:
- Sepsis detection AUC: > 0.85
- ACS detection AUC: > 0.80
- MODS detection AUC: > 0.82

#### SC-3.3: Protocol Recommendation Quality
**Measurement**: Clinical relevance and actionability of protocol recommendations.

**Assessment Method**:
- Clinical team review of 200 recommendations
- Scoring: Relevant (yes/no), Actionable (yes/no), Appropriate timing (yes/no)

**Targets**:
- Relevance rate: > 85%
- Actionability rate: > 80%
- Appropriate timing: > 90%
- Clinician satisfaction score: > 4.0/5.0

**Monitoring**:
- Clinician feedback collection system
- Recommendation acceptance rate tracking
- Protocol adherence improvement measurement

### 4.4 System Reliability Metrics

#### SC-4.1: Data Availability
**Measurement**: Uptime and availability of enrichment data sources.

**Targets**:
- FHIR API availability: > 99.5%
- Neo4j availability: > 99.9%
- RocksDB state recovery time: < 5 minutes
- Pipeline availability: > 99.9%

**Monitoring**:
- Health check dashboards for all dependencies
- Failure detection and alerting
- Automatic failover testing

#### SC-4.2: Error Handling Effectiveness
**Measurement**: System resilience to external service failures.

**Targets**:
- Graceful degradation: 100% of external service failures handled
- Data loss on failure: 0%
- Recovery time from transient failures: < 30 seconds
- Events processed despite missing enrichment: > 95%

**Testing**:
- Chaos engineering tests (random API failures)
- Network partition simulation
- Rate limiting scenarios
- Data unavailability scenarios

#### SC-4.3: Data Quality Monitoring
**Measurement**: Quality of enriched data over time.

**Metrics**:
- Data freshness: % of records with stale data (> 24h old)
- Data consistency: % of records with conflicting data
- Data validation failures: Count per day
- Missing critical data: % of records

**Targets**:
- Stale data rate: < 5%
- Data consistency: > 99%
- Validation failures: < 1% of records
- Missing critical data: < 10%

**Monitoring**:
- Data quality dashboard
- Automated data quality checks
- Data quality score per patient

---

## 5. Risk Assessment

### 5.1 State Management Complexity

#### Risk 5.1.1: RocksDB State Size Explosion
**Severity**: HIGH
**Probability**: MEDIUM
**Description**: Adding FHIR/Neo4j data to PatientContextState may cause state size to exceed RocksDB capacity.

**Impact**:
- State checkpoint failures
- Out of memory errors
- Slow state recovery on restart
- Increased storage costs

**Mitigation Strategies**:
1. Implement aggressive TTL for cached FHIR data (24 hours)
2. Store only essential FHIR fields, not full resources
3. Implement state pruning for inactive patients (no events in 7 days)
4. Monitor state size per key and alert on anomalies
5. Use incremental checkpoints instead of full checkpoints
6. Implement state compression

**Contingency Plan**:
- Reduce TTL to 12 hours if state size issues occur
- Move infrequently accessed data to external cache (Redis)
- Implement lazy loading for FHIR data (fetch on-demand)

#### Risk 5.1.2: State Serialization Performance
**Severity**: MEDIUM
**Probability**: MEDIUM
**Description**: Complex FHIR objects may slow down state serialization/deserialization.

**Impact**:
- Increased checkpoint time
- Higher CPU usage
- Reduced throughput
- State recovery delays

**Mitigation Strategies**:
1. Use efficient serialization (Kryo instead of Java serialization)
2. Profile serialization performance in testing
3. Simplify FHIR objects before storing in state
4. Implement custom serializers for FHIR types
5. Benchmark state size and serialization time

**Contingency Plan**:
- Implement custom lightweight DTOs for state storage
- Move large objects to external storage with references in state
- Use async checkpoint mode to reduce blocking

#### Risk 5.1.3: State Consistency Across Restarts
**Severity**: MEDIUM
**Probability**: LOW
**Description**: State recovery may result in inconsistent FHIR data after failures.

**Impact**:
- Stale FHIR data served after restart
- Inconsistent clinical scores
- Duplicate API calls
- Clinical decision errors

**Mitigation Strategies**:
1. Store data fetch timestamps in state
2. Re-validate stale data on recovery
3. Implement state versioning
4. Test recovery scenarios extensively
5. Add state consistency checks on recovery

**Contingency Plan**:
- Purge FHIR cache on recovery and re-fetch
- Implement state repair job for inconsistencies
- Gradual rollout with state validation

### 5.2 External Service Dependencies

#### Risk 5.2.1: FHIR API Rate Limiting
**Severity**: HIGH
**Probability**: MEDIUM
**Description**: Google Healthcare API may rate limit requests during high load periods.

**Impact**:
- Failed FHIR enrichment requests
- Incomplete patient context
- Backlog of unenriched events
- Degraded clinical decision support

**Mitigation Strategies**:
1. Implement exponential backoff with retry
2. Request rate limit increase from Google
3. Implement request batching where possible
4. Add caching layer (Redis) for frequently accessed patients
5. Monitor API quota usage in real-time
6. Implement circuit breaker to prevent cascade failures

**Contingency Plan**:
- Graceful degradation: Process events without FHIR enrichment
- Priority queue: Enrich critical patients first
- Fallback to minimal enrichment (demographics only)

#### Risk 5.2.2: Neo4j Performance Degradation
**Severity**: MEDIUM
**Probability**: MEDIUM
**Description**: Complex graph queries may become slow as graph size grows.

**Impact**:
- Neo4j query timeouts
- Increased latency in pipeline
- Reduced throughput
- Missing graph enrichment data

**Mitigation Strategies**:
1. Implement graph query optimization (indexes, query tuning)
2. Set aggressive query timeouts (5 seconds)
3. Cache graph query results (1 hour TTL)
4. Monitor query performance and optimize slow queries
5. Implement materialized views for common queries
6. Scale Neo4j cluster horizontally

**Contingency Plan**:
- Simplify graph queries to reduce complexity
- Implement query result pre-computation
- Graceful degradation without graph enrichment

#### Risk 5.2.3: Network Failures and Timeouts
**Severity**: MEDIUM
**Probability**: LOW
**Description**: Network issues may cause failures in FHIR or Neo4j communication.

**Impact**:
- Async operation timeouts
- Incomplete enrichment
- Error rate increase
- User experience degradation

**Mitigation Strategies**:
1. Implement connection pooling with health checks
2. Use async I/O with appropriate timeouts
3. Implement retry with exponential backoff
4. Add circuit breaker pattern
5. Monitor network latency and errors
6. Test with network fault injection

**Contingency Plan**:
- Continue processing without enrichment
- Queue failed enrichment for retry
- Alert operations team on sustained failures

### 5.3 Data Consistency Concerns

#### Risk 5.3.1: Temporal Data Misalignment
**Severity**: MEDIUM
**Probability**: MEDIUM
**Description**: Real-time events and FHIR historical data may have temporal inconsistencies.

**Impact**:
- Duplicate data in output
- Conflicting vital signs or lab results
- Incorrect trend calculations
- Clinical decision errors

**Mitigation Strategies**:
1. Implement timestamp-based deduplication
2. Define clear data source priority (real-time > FHIR historical)
3. Add data lineage tracking (source identifier)
4. Validate temporal consistency in tests
5. Implement conflict resolution rules
6. Monitor for duplicate detection

**Contingency Plan**:
- Prefer real-time data over historical FHIR data
- Flag conflicting data for clinical review
- Implement data reconciliation batch job

#### Risk 5.3.2: FHIR Data Staleness
**Severity**: MEDIUM
**Probability**: HIGH
**Description**: Cached FHIR data may become stale if patient data changes externally.

**Impact**:
- Outdated medications in medication interaction checks
- Stale allergy information
- Incorrect clinical scoring
- Patient safety risks

**Mitigation Strategies**:
1. Implement reasonable TTL (24 hours for demographics, 1 hour for medications)
2. Force refresh on critical events (new medication event)
3. Add data freshness indicators in output
4. Monitor staleness metrics
5. Implement webhook-based cache invalidation if available
6. Add manual refresh capability

**Contingency Plan**:
- Reduce TTL to 6 hours if staleness issues occur
- Implement FHIR subscription for real-time updates
- Flag stale data in clinical interface

#### Risk 5.3.3: Data Privacy and Compliance
**Severity**: CRITICAL
**Probability**: LOW
**Description**: Improper handling of FHIR PHI in state or logs may violate HIPAA.

**Impact**:
- HIPAA violations
- Patient privacy breaches
- Legal and financial penalties
- Reputational damage

**Mitigation Strategies**:
1. Encrypt state backend at rest and in transit
2. Sanitize logs to remove PHI
3. Implement access controls on state storage
4. Audit all FHIR data access
5. Conduct privacy impact assessment
6. Regular security audits
7. Data minimization: Store only necessary FHIR fields

**Contingency Plan**:
- Immediate data breach response protocol
- Notify affected patients and authorities
- Forensic analysis and remediation

### 5.4 Implementation Risks

#### Risk 5.4.1: Integration Complexity
**Severity**: MEDIUM
**Probability**: HIGH
**Description**: Integrating two different architectures (stateless enrichment + stateful aggregation) is complex.

**Impact**:
- Extended development timeline
- Integration bugs
- Difficult debugging
- Technical debt accumulation

**Mitigation Strategies**:
1. Phased implementation approach (Phase 1-4)
2. Comprehensive integration testing
3. Code review requirements
4. Pair programming for complex integrations
5. Architectural decision documentation
6. Regular technical debt review

**Contingency Plan**:
- Extend timelines if integration issues arise
- Incremental rollout with feature flags
- Rollback capability at each phase

#### Risk 5.4.2: Performance Regression
**Severity**: HIGH
**Probability**: MEDIUM
**Description**: Adding enrichment stages may degrade pipeline performance.

**Impact**:
- Increased end-to-end latency
- Reduced throughput
- Backpressure and event lag
- System instability

**Mitigation Strategies**:
1. Performance benchmarking before and after integration
2. Load testing at each phase
3. Profiling and optimization
4. Parallel execution where possible
5. Resource scaling plan
6. Performance regression testing in CI/CD

**Contingency Plan**:
- Performance optimization sprint if regression detected
- Horizontal scaling of Flink cluster
- Disable enrichment stages if critical performance issues

#### Risk 5.4.3: Clinical Validation Delays
**Severity**: MEDIUM
**Probability**: MEDIUM
**Description**: Clinical team validation of new features may take longer than expected.

**Impact**:
- Delayed production rollout
- Reduced clinical adoption
- Missed business objectives
- Stakeholder dissatisfaction

**Mitigation Strategies**:
1. Early clinical team engagement
2. Regular validation checkpoints
3. Clinical advisory board involvement
4. User acceptance testing in staging
5. Clinical workflow integration planning
6. Training and documentation

**Contingency Plan**:
- Phased clinical rollout (pilot units first)
- Extended staging period for validation
- Clinical champion program for early adopters

---

## 6. Acceptance Criteria Summary

### 6.1 Must-Have Criteria (Go/No-Go)

1. **FHIR Enrichment Operational**
   - Demographics retrieved for > 95% of patients
   - Chronic conditions retrieved for > 90% of patients
   - Active medications retrieved for > 85% of patients
   - FHIR API calls complete within 10 seconds (p95)

2. **Neo4j Enrichment Operational**
   - Similar patients identified for > 80% of patients
   - Cohort memberships identified for > 70% of patients
   - Neo4j queries complete within 5 seconds (p95)

3. **Performance Requirements Met**
   - End-to-end latency < 15 seconds (p95)
   - Throughput: 1,000 events/second sustained
   - No performance regression in existing features

4. **Data Quality Standards**
   - Data completeness > 90% for critical fields
   - Data validation failures < 1%
   - State size within acceptable limits (< 50GB per task manager)

5. **System Reliability**
   - Pipeline availability > 99.9%
   - Graceful degradation on external service failures
   - Zero data loss on failures

6. **Clinical Validation**
   - Clinical scores validated by clinical team
   - Pattern detection accuracy approved
   - Protocol recommendations clinically appropriate

### 6.2 Should-Have Criteria (Desirable)

1. **Advanced Clinical Scoring**
   - Framingham score calculated for > 70% of adult patients
   - CHADS-VASC score calculated for > 95% of AFib patients
   - Metabolic syndrome score available

2. **Clinical Intelligence Enhancement**
   - Pattern detection sensitivity improved by 15%
   - False positive rate reduced by 20%
   - Protocol recommendations include specific action items

3. **Operational Excellence**
   - Monitoring dashboards comprehensive
   - Alerting configured for all critical metrics
   - Runbooks documented for common issues

4. **Documentation Complete**
   - Architecture documentation updated
   - API documentation for output schema
   - Operational runbooks created

### 6.3 Nice-to-Have Criteria (Optional)

1. **Protocol Events Pipeline**
   - Protocol events extracted to separate topic
   - Audit trail queryable
   - Downstream systems integrated

2. **Advanced Features**
   - Medication interaction checking operational
   - Care team notifications integrated
   - Clinical guideline compliance tracking

3. **Optimization**
   - Cache hit rate > 80% for FHIR data
   - Cache hit rate > 70% for Neo4j data
   - Resource utilization optimized

---

## 7. Appendices

### Appendix A: Glossary

- **ACS**: Acute Coronary Syndrome
- **AFib**: Atrial Fibrillation
- **CHADS-VASC**: Stroke risk score for atrial fibrillation patients
- **FHIR**: Fast Healthcare Interoperability Resources (HL7 standard)
- **Framingham**: Cardiovascular risk assessment tool
- **LOINC**: Logical Observation Identifiers Names and Codes (lab test codes)
- **MODS**: Multiple Organ Dysfunction Syndrome
- **NEWS2**: National Early Warning Score 2 (UK clinical deterioration score)
- **PHI**: Protected Health Information
- **qSOFA**: Quick Sequential Organ Failure Assessment (sepsis screening)
- **RxNorm**: Medication naming standard
- **SNOMED CT**: Systematized Nomenclature of Medicine Clinical Terms
- **TTL**: Time To Live (cache expiration)

### Appendix B: Reference Documents

- **INTEGRATION_ANALYSIS_OLD_TO_NEW.md**: Source analysis document
- **Module2_Enhanced.java**: Flink pipeline implementation
- **Google Healthcare API Documentation**: FHIR API reference
- **Neo4j Documentation**: Cypher query language reference
- **FHIR R4 Specification**: HL7 FHIR standard documentation
- **Clinical Scoring Guidelines**: NEWS2, qSOFA, Framingham, CHADS-VASC calculation methods

### Appendix C: Contact Information

- **Technical Lead**: [To be assigned]
- **Clinical Advisor**: [To be assigned]
- **Project Manager**: [To be assigned]
- **Security/Compliance**: [To be assigned]

---

**Document Status**: DRAFT
**Requires Approval From**: Technical Lead, Clinical Advisor, Project Manager
**Next Review Date**: [To be determined after initial review]
