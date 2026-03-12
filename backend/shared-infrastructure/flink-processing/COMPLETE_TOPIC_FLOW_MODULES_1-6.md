# Complete Topic Flow: Modules 1-6 Analysis

**Comprehensive mapping of all Kafka topics through the Flink processing pipeline**

---

## 📊 Module-by-Module Topic Flow

### Module 1: Ingestion & Validation

**Purpose**: Ingest raw clinical events and validate data quality

**INPUT Topics** (6 topics - external sources):
```
✅ patient-events-v1 (4 partitions, 3d retention)
✅ medication-events-v1 (4 partitions, 3d retention)
✅ observation-events-v1 (4 partitions, 3d retention)
✅ vital-signs-events-v1 (4 partitions, 3d retention)
✅ lab-result-events-v1 (4 partitions, 7d retention)
✅ validated-device-data-v1 (4 partitions, 3d retention)
```

**OUTPUT Topics** (2 topics):
```
→ enriched-patient-events-v1 (4 partitions, 7d retention)
   Consumer Groups: patient-ingestion-v2, medication-ingestion, observation-ingestion,
                    vital-ingestion, lab-ingestion, device-ingestion

→ dlq.processing-errors.v1 (4 partitions, 30d retention)
   Failed validation events
```

**Code Reference**: Module1_Ingestion.java lines 103-154, 377, 394

---

### Module 2: Context Assembly & Enrichment

**Purpose**: Assemble patient clinical context and enrich events with historical data

**INPUT Topics** (1 topic):
```
← enriched-patient-events-v1 (from Module 1)
   Consumer Group: module2-context-assembly
```

**OUTPUT Topics** (2 topics):
```
→ clinical-patterns.v1 (8 partitions, 30d retention)
   Enriched events with full patient context, active conditions, medications, procedures

→ protocol-triggers.v1 (4 partitions, 30d retention)
   Clinical protocol trigger audit trail
```

**Code Reference**: Module2_Enhanced.java lines 1039, 1062, 1094, 2218

---

### Module 3: Comprehensive Clinical Decision Support (CDS)

**Purpose**: Apply clinical protocols, dosing rules, drug interaction checking

**INPUT Topics** (1 topic):
```
← clinical-patterns.v1 (from Module 2)
   Consumer Group: module3-cds
   Environment variable: MODULE3_INPUT_TOPIC
```

**OUTPUT Topics** (1 topic):
```
→ comprehensive-cds-events.v1 (NEW topic)
   Events with CDS recommendations, protocol adherence, drug safety alerts
   Environment variable: MODULE3_OUTPUT_TOPIC
```

**Code Reference**: Module3_ComprehensiveCDS.java lines 1442, 1473

---

### Module 4: Pattern Detection & Clinical Event Processing (CEP)

**Purpose**: Detect clinical patterns, deterioration, pathway adherence using complex event processing

**INPUT Topics** (3 topics):
```
← semantic-mesh-updates.v1 (from knowledge base)
   Consumer Group: module4-semantic
   Environment variable: MODULE4_SEMANTIC_INPUT_TOPIC

← clinical-patterns.v1 (from Module 2)
   Consumer Group: module4-enriched
   Environment variable: MODULE4_ENRICHED_INPUT_TOPIC

← comprehensive-cds-events.v1 (from Module 3)
   Consumer Group: module4-cds
   Environment variable: MODULE4_CDS_INPUT_TOPIC
```

**OUTPUT Topics** (7 topics):
```
→ pattern-events.v1
   Detected clinical patterns
   Environment variable: MODULE4_PATTERN_EVENTS_TOPIC

→ semantic-mesh-updates.v1 (updated/republished)
   Updated semantic relationships
   Uses: KafkaTopics.SEMANTIC_MESH_UPDATES

→ alert-management.v1
   Deterioration alerts and sepsis warnings
   Environment variable: MODULE4_DETERIORATION_TOPIC

→ pathway-adherence-events.v1 (8 partitions, 30d retention)
   Clinical pathway compliance tracking
   Environment variable: MODULE4_PATHWAY_ADHERENCE_TOPIC

→ safety-events.v1
   Anomaly detection and safety alerts
   Environment variable: MODULE4_ANOMALY_DETECTION_TOPIC

→ clinical-reasoning-events.v1 (8 partitions, 30d retention)
   Trend analysis and clinical reasoning
   Environment variable: MODULE4_TREND_ANALYSIS_TOPIC

→ daily-risk-scores.v1 (NEW)
   Daily patient risk score aggregations
   Environment variable: MODULE4_DAILY_RISK_SCORE_TOPIC
```

**Code Reference**: Module4_PatternDetection.java lines 586, 607, 628, 1497-1580

---

### Module 5: ML Inference & Predictive Analytics

**Purpose**: Real-time ML model inference for sepsis, readmission, deterioration, fall risk, mortality

**INPUT Topics** (2 topics):
```
← semantic-mesh-updates.v1 (from Module 4)
   Semantic context for ML features

← pattern-events.v1 (from Module 4)
   Detected patterns for ML features
   Note: Also reads from clinical-patterns.v1 for backward compatibility
```

**OUTPUT Topics** (6+ topics):
```
→ ml-risk-alerts.v1 (PRIMARY OUTPUT - USER CONFIRMED)
   *** Main ML risk predictions topic ***
   Consolidates all ML-based risk scores (sepsis, readmission, deterioration, fall, mortality)

→ inference-results.v1 (8 partitions, 30d retention)
   General ML inference results
   Uses: KafkaTopics.INFERENCE_RESULTS

→ clinical-reasoning-events.v1
   Readmission risk predictions and mortality predictions
   Uses: KafkaTopics.CLINICAL_REASONING_EVENTS

→ alert-management.v1
   Sepsis predictions and deterioration risk alerts
   Uses: KafkaTopics.ALERT_MANAGEMENT (HIGH/CRITICAL severity)

→ safety-events.v1
   Fall risk predictions
   Uses: KafkaTopics.SAFETY_EVENTS

→ clinical-patterns.v1 (enriched with ML)
   Re-published enriched events with ML predictions added
   Consumer Group: module5-clinical-patterns
```

**Code Reference**: Module5_MLInference.java lines 213, 235, 907, 927, 939-1013
**User Correction**: Module 5 primary output is ml-risk-alerts.v1

---

### Module 6A: Alert Composition

**Purpose**: Compose and prioritize alerts from multiple sources

**INPUT Topics** (2+ topics):
```
← {configured via environment} - Various alert sources
   Consumer Group: module6-alert-composition

← clinical-patterns.v1
   Base clinical events for alert correlation
```

**OUTPUT Topics** (3 topics):
```
→ simple-alerts.v1 (4 partitions, 7d retention)
   Module 2 threshold-based alerts
   Uses: KafkaTopics.SIMPLE_ALERTS

→ composed-alerts.v1 (4 partitions, 7d retention)
   Module 6 composed alerts (all severities)
   Uses: KafkaTopics.COMPOSED_ALERTS
   *** CONSUMED BY MODULE 6C Analytics Engine ***

→ urgent-alerts.v1 (4 partitions, 7d retention)
   Module 6 urgent alerts (HIGH + CRITICAL only)
   Uses: KafkaTopics.URGENT_ALERTS
```

**Code Reference**: Module6_AlertComposition.java lines 713, 739, 765, 800, 832

---

### Module 6B: Egress Routing & Multi-Sink Distribution

**Purpose**: Route processed events to hybrid Kafka architecture and downstream systems

**INPUT Topics** (4 topics from previous modules):
```
← semantic-mesh-updates.v1 (from Module 4)
   Consumer Group: egress-semantic
   Uses: KafkaTopics.SEMANTIC_MESH_UPDATES

← clinical-patterns.v1 (from Module 2)
   Consumer Group: egress-patterns
   Uses: KafkaTopics.CLINICAL_PATTERNS

← inference-results.v1 (from Module 5)
   Consumer Group: egress-ml
   Uses: KafkaTopics.INFERENCE_RESULTS

← patient-context-snapshots.v1 (12 partitions, 7d retention)
   Consumer Group: egress-context
   Uses: KafkaTopics.PATIENT_CONTEXT_SNAPSHOTS
```

**OUTPUT Topics - HYBRID ARCHITECTURE** (7 topics):
```
🔵 PRIMARY OUTPUTS (via TransactionalMultiSinkRouter):

→ prod.ehr.events.enriched (24 partitions, 90d retention)
   Central system of record - ALL enriched clinical events
   Uses: KafkaTopics.EHR_EVENTS_ENRICHED
   *** CONSUMED BY MODULE 8 (6 core projectors) ***

→ prod.ehr.fhir.upsert (12 partitions, 365d retention, COMPACTED)
   FHIR resource upserts for Google Healthcare API
   Uses: KafkaTopics.EHR_FHIR_UPSERT
   *** CONSUMED BY MODULE 8 (FHIR Store Projector) ***

→ prod.ehr.graph.mutations (16 partitions, 30d retention)
   Neo4j graph database mutations for patient journeys
   Uses: KafkaTopics.EHR_GRAPH_MUTATIONS
   *** CONSUMED BY MODULE 8 (Neo4j Graph Projector) ***

→ prod.ehr.alerts.critical (16 partitions, 7d retention)
   Critical clinical alerts requiring immediate action
   Uses: KafkaTopics.EHR_ALERTS_CRITICAL_ACTION

→ prod.ehr.analytics.events (32 partitions, 180d retention)
   High-throughput analytics events
   Uses: KafkaTopics.EHR_ANALYTICS_EVENTS

→ prod.ehr.semantic.mesh (4 partitions, 365d retention, COMPACTED)
   Semantic mesh updates - knowledge graph changes
   Uses: KafkaTopics.EHR_SEMANTIC_MESH

→ prod.ehr.audit.logs (8 partitions, 2555d/7y retention)
   Compliance audit logs
   Uses: KafkaTopics.EHR_AUDIT_LOGS
```

**OUTPUT Topics - LEGACY SYSTEMS** (6 topics for migration):
```
→ workflow-events.v1 (8 partitions, 30d retention)
   Clinical workflow routing
   Uses: KafkaTopics.WORKFLOW_EVENTS

→ alert-management.v1 (8 partitions, 30d retention)
   Critical alerts (legacy)
   Uses: KafkaTopics.ALERT_MANAGEMENT

→ performance-metrics.v1 (12 partitions, 7d retention)
   Analytics metrics (legacy)
   Uses: KafkaTopics.PERFORMANCE_METRICS

→ hl7-outbound.v1 (8 partitions, 90d retention)
   External system integration
   Uses: KafkaTopics.HL7_OUTBOUND

→ audit-events.v1 (6 partitions, 365d retention)
   Audit and compliance (legacy)
   Uses: KafkaTopics.AUDIT_EVENTS

→ notification-events.v1 (8 partitions, 7d retention)
   Real-time notifications
   Uses: KafkaTopics.NOTIFICATION_EVENTS

→ precomputed-views.v1 (8 partitions, 7d retention)
   Batch analytics materialization
   Uses: KafkaTopics.PRECOMPUTED_VIEWS

→ dlq.processing-errors.v1 (4 partitions, 30d retention)
   Failed routing events
   Uses: KafkaTopics.DLQ_PROCESSING_ERRORS
```

**Code Reference**: Module6_EgressRouting.java lines 140-145, 201-240, 803-853

---

### Module 6C: Analytics Engine (SQL + DataStream)

**Purpose**: Real-time analytics and materialized views for operational dashboards

**INPUT Topics** (3 topics):
```
← comprehensive-cds-events.v1 (from Module 3)
   Consumer Group: module6-analytics-proctime
   Enriched patient context events for analytics

← composed-alerts.v1 (from Module 6A)
   Consumer Group: module6-analytics-alerts-proctime
   Alert metrics aggregation

← inference-results.v1 (from Module 5)
   Consumer Group: module6-analytics-ml-proctime, module6-population-health
   ML predictions for performance monitoring
```

**OUTPUT Topics - SQL Analytics** (5 topics):
```
→ analytics-patient-census (USER CONFIRMED)
   1-minute tumbling window
   Real-time patient census by event type and acuity level
   Uses: Kafka connector with transactional-id-prefix

→ analytics-alert-metrics (USER CONFIRMED)
   1-minute tumbling window
   Alert aggregation by severity, pattern type, department
   Uses: Upsert-Kafka connector with PRIMARY KEY

→ analytics-ml-performance (USER CONFIRMED)
   5-minute tumbling window
   ML model performance, latency, prediction distribution
   Uses: Kafka connector with transactional-id-prefix

→ analytics-department-workload (USER CONFIRMED)
   1-hour sliding window, 5-minute slide
   Department workload trending and high-acuity tracking
   Uses: Kafka connector with transactional-id-prefix

→ analytics-sepsis-surveillance
   Real-time streaming (no windowing)
   Immediate sepsis risk identification (qSOFA >= 2 OR NEWS2 >= 5)
   Uses: Kafka connector with transactional-id-prefix
```

**OUTPUT Topics - DataStream Analytics** (2 topics):
```
→ analytics-vital-timeseries (USER CONFIRMED)
   1-minute vital sign rollups
   Time-series aggregation from comprehensive-cds-events.v1
   Uses: AT_LEAST_ONCE delivery guarantee

→ analytics-population-health
   Department-level population metrics
   Keyed by department from inference-results.v1
   Uses: AT_LEAST_ONCE delivery guarantee
```

**Code Reference**: Module6_AnalyticsEngine.java lines 85-640
- SQL Views: lines 331-640 (5 materialized views)
- DataStream Analytics: lines 133-208 (2 components)

---

## 🔄 Complete Data Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│  EXTERNAL SOURCES (HL7, FHIR APIs, Device Streams, Lab Systems)      │
└────────┬─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 1: Ingestion & Validation                                    │
│                                                                     │
│ IN:  patient-events-v1, medication-events-v1,                      │
│      observation-events-v1, vital-signs-events-v1,                 │
│      lab-result-events-v1, validated-device-data-v1                │
│                                                                     │
│ OUT: enriched-patient-events-v1 ✅                                  │
│      dlq.processing-errors.v1                                      │
└────────┬────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 2: Context Assembly & Enrichment                            │
│                                                                     │
│ IN:  enriched-patient-events-v1 ✅                                  │
│                                                                     │
│ OUT: clinical-patterns.v1 ✅                                        │
│      protocol-triggers.v1                                          │
└────────┬────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 3: Clinical Decision Support (CDS)                          │
│                                                                     │
│ IN:  clinical-patterns.v1 ✅                                        │
│                                                                     │
│ OUT: comprehensive-cds-events.v1                                   │
└────────┬────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 4: Pattern Detection (CEP)                                  │
│                                                                     │
│ IN:  semantic-mesh-updates.v1 (from KB)                            │
│      clinical-patterns.v1 ✅                                        │
│      comprehensive-cds-events.v1                                   │
│                                                                     │
│ OUT: pattern-events.v1                                             │
│      semantic-mesh-updates.v1 (updated)                            │
│      alert-management.v1                                           │
│      pathway-adherence-events.v1 ✅                                 │
│      safety-events.v1                                              │
│      clinical-reasoning-events.v1                                  │
│      daily-risk-scores.v1                                          │
└────────┬────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 5: ML Inference & Predictions                               │
│                                                                     │
│ IN:  semantic-mesh-updates.v1                                      │
│      pattern-events.v1                                             │
│      clinical-patterns.v1 ✅                                        │
│                                                                     │
│ OUT: ml-risk-alerts.v1 ✅ (PRIMARY OUTPUT - USER CONFIRMED)         │
│      inference-results.v1 ✅                                        │
│      clinical-reasoning-events.v1 (readmission, mortality)         │
│      alert-management.v1 (sepsis, deterioration)                   │
│      safety-events.v1 (fall risk)                                  │
│      clinical-patterns.v1 (enriched with ML)                       │
└────────┬────────────────────────────────────────────────────────────┘
         │
         ├─────────────────────────┬─────────────────────────────────┐
         ▼                         ▼                                 ▼
┌──────────────────────┐  ┌────────────────────┐  ┌─────────────────────────┐
│ MODULE 6A:           │  │ MODULE 6B:         │  │ MODULE 6C:              │
│ Alert Composition    │  │ Egress Routing     │  │ Analytics Engine        │
│                      │  │                    │  │                         │
│ IN:  clinical-       │  │ IN:  semantic-mesh │  │ IN:  comprehensive-cds  │
│      patterns ✅     │  │      clinical-     │  │      composed-alerts    │
│      (alerts)        │  │      inference-    │  │      inference-results  │
│                      │  │      patient-ctx   │  │                         │
│ OUT: simple-alerts   │  │                    │  │ OUT (SQL Analytics):    │
│      composed-alerts │──┤ OUT HYBRID: ──┐    │  │  analytics-patient-     │
│      urgent-alerts   │  │  prod.ehr.*   │    │  │    census ✅            │
└──────────────────────┘  │  (7 topics)   │    │  │  analytics-alert-       │
                          │               │    │  │    metrics ✅           │
                          │ OUT LEGACY:   │    │  │  analytics-ml-          │
                          │  workflow-    │    │  │    performance ✅       │
                          │  alert-mgmt   │    │  │  analytics-department-  │
                          │  perf-metrics │    │  │    workload ✅          │
                          │  hl7-outbound │    │  │  analytics-sepsis-      │
                          │  notification │    │  │    surveillance         │
                          │  precomputed  │    │  │                         │
                          │  (6 topics)   │    │  │ OUT (DataStream):       │
                          └───────┬───────┘    │  │  analytics-vital-       │
                                  │            │  │    timeseries ✅        │
                                  │            │  │  analytics-population-  │
                                  │            │  │    health               │
                                  ▼            │  └─────────────────────────┘
                         ┌───────────────┐     │
                         │ HYBRID TOPICS │     │
                         │ (Module 8)    │◄────┘
                         └───────┬───────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│ MODULE 8: Storage Projectors (Python FastAPI Services)             │
│                                                                     │
│ 6 CORE PROJECTORS (consume prod.ehr.events.enriched):              │
│   1. PostgreSQL Projector → PostgreSQL (OLTP)                      │
│   2. MongoDB Projector → MongoDB (Documents)                       │
│   3. Elasticsearch Projector → Elasticsearch (Search)              │
│   4. ClickHouse Projector → ClickHouse (OLAP)                      │
│   5. InfluxDB Projector → InfluxDB (Time-series)                   │
│   6. UPS Read Model Projector → PostgreSQL (Hot queries)           │
│                                                                     │
│ 2 SPECIALIZED PROJECTORS:                                          │
│   7. FHIR Store Projector → Google Healthcare API                  │
│      (consumes prod.ehr.fhir.upsert)                               │
│   8. Neo4j Graph Projector → Neo4j (Patient journeys)              │
│      (consumes prod.ehr.graph.mutations)                           │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 🔑 Key Topics with Data Presence

**Currently Have Data** (✅ verified with offsets > 0):
```
✅ enriched-patient-events-v1: 15,305 messages
✅ clinical-patterns.v1: Has data (from Module 2)
```

**Newly Created** (ready for Module 6):
```
🆕 prod.ehr.events.enriched (24 partitions, 90d) - READY
🆕 prod.ehr.fhir.upsert (12 partitions, 365d, compacted) - READY
🆕 prod.ehr.graph.mutations (16 partitions, 30d) - READY
```

**Missing/Need Verification**:
```
❓ comprehensive-cds-events.v1 (Module 3 → 4)
❓ pattern-events.v1 (Module 4 → 5)
❓ inference-results.v1 (Module 5 → 6)
❓ patient-context-snapshots.v1 (Module 2 → 6)
❓ semantic-mesh-updates.v1 (Module 4 output/input)
```

---

## 🎯 Critical Finding: Module 6 TransactionalMultiSinkRouter

**Status**: ✅ **IS IN THE PIPELINE** (Line 137 of Module6_EgressRouting.java)

```java
// Line 134-137 in Module6_EgressRouting.java
// **CORE: Transactional Multi-Sink Router**
// This implements the recommended hybrid architecture with EXACTLY_ONCE semantics
enrichedEvents
    .process(new TransactionalMultiSinkRouter());
```

**But**:
- ❌ Module 6 is **NOT RUNNING** (`curl http://localhost:8081/jobs` returns empty array)
- ❌ No Flink jobs deployed at all
- ✅ Topics NOW CREATED (we just created them)

---

## 🚀 Next Steps to Activate Module 6 → Module 8 Flow

### Step 1: Verify Current Flink Status
```bash
# Check if Flink is running
curl http://localhost:8081/jobs | python3 -m json.tool

# Expected: Empty jobs array (needs deployment)
```

### Step 2: Deploy Module 6
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Build and deploy Module 6
./deploy-module6.sh
```

### Step 3: Verify Module 6 is Writing
```bash
# Check hybrid topic offsets (should increase)
docker exec 3c7ffa06d20d kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic prod.ehr.events.enriched \
  --time -1

# Consume a test message
docker exec 3c7ffa06d20d kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --from-beginning \
  --max-messages 1
```

### Step 4: Start Module 8 Projectors
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services

# Start all 8 projectors
./start-module8-projectors.sh

# Verify consumption
./health-check-module8.sh
```

---

## 📊 Topic Statistics Summary

| Module | Input Topics | Output Topics | Total Unique Topics |
|--------|--------------|---------------|---------------------|
| Module 1 | 6 | 2 | 8 |
| Module 2 | 1 | 2 | 3 |
| Module 3 | 1 | 1 | 2 |
| Module 4 | 3 | 7 | 10 |
| Module 5 | 2 | 6 | 8 |
| Module 6A | 2 | 3 | 5 |
| Module 6B | 4 | 13 (7 hybrid + 6 legacy) | 17 |
| Module 6C | 3 | 7 (5 SQL + 2 DataStream) | 10 |
| **TOTAL** | **22 unique inputs** | **41 unique outputs** | **~50 topics** |

### Module 6 Complete Architecture

Module 6 consists of **3 independent components**:
- **Module 6A (Alert Composition)**: Composes and prioritizes alerts → 3 output topics
- **Module 6B (Egress Routing)**: Routes to hybrid architecture → 13 output topics
- **Module 6C (Analytics Engine)**: Real-time analytics and dashboards → 7 output topics

**Total Module 6 Outputs**: 23 topics (3 alert + 13 hybrid/legacy + 7 analytics)

---

## 🔍 Environment Variable Configuration

Some topics use environment variables for configuration:

```bash
# Module 3
MODULE3_INPUT_TOPIC=clinical-patterns.v1
MODULE3_OUTPUT_TOPIC=comprehensive-cds-events.v1

# Module 4
MODULE4_SEMANTIC_INPUT_TOPIC=semantic-mesh-updates.v1
MODULE4_ENRICHED_INPUT_TOPIC=clinical-patterns.v1
MODULE4_CDS_INPUT_TOPIC=comprehensive-cds-events.v1
MODULE4_PATTERN_EVENTS_TOPIC=pattern-events.v1
MODULE4_DETERIORATION_TOPIC=alert-management.v1
MODULE4_PATHWAY_ADHERENCE_TOPIC=pathway-adherence-events.v1
MODULE4_ANOMALY_DETECTION_TOPIC=safety-events.v1
MODULE4_TREND_ANALYSIS_TOPIC=clinical-reasoning-events.v1
MODULE4_DAILY_RISK_SCORE_TOPIC=daily-risk-scores.v1
```

---

**Document Generated**: 2025-11-16
**Kafka Container**: 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754
**Flink Web UI**: http://localhost:8081 (currently no jobs running)
**Status**: Topics created ✅ | Module 6 needs deployment ❌
