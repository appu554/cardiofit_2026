# CDC BroadcastStream & Neo4j Database Aliasing Implementation Guide

## Executive Summary

This document details the implementation of **zero-downtime terminology versioning** for the CardioFit Clinical Synthesis Hub using:
- **Flink BroadcastStream** for real-time terminology hot-swapping across streaming pipelines
- **Neo4j Database Aliasing** for atomic database switching without client reconfiguration
- **CDC (Change Data Capture)** for event-driven propagation of terminology updates

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [CDC Pipeline Flow](#2-cdc-pipeline-flow)
3. [Flink BroadcastStream Pattern](#3-flink-broadcaststream-pattern)
4. [Neo4j Database Aliasing](#4-neo4j-database-aliasing)
5. [Component Implementation](#5-component-implementation)
6. [Integration Points](#6-integration-points)
7. [Deployment Guide](#7-deployment-guide)
8. [Monitoring & Operations](#8-monitoring--operations)
9. [Troubleshooting](#9-troubleshooting)

---

## 1. Architecture Overview

### 1.1 High-Level Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    KB-7 TERMINOLOGY VERSIONING ARCHITECTURE                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │  Knowledge      │     │   PostgreSQL    │     │    Debezium     │       │
│  │  Factory        │────▶│   kb_releases   │────▶│    CDC          │       │
│  │  Pipeline       │     │   (Outbox)      │     │    Connector    │       │
│  └─────────────────┘     └─────────────────┘     └────────┬────────┘       │
│         │                                                  │                │
│         │ GCS Artifacts                                    │ CDC Events    │
│         ▼                                                  ▼                │
│  ┌─────────────────┐     ┌─────────────────────────────────────────┐       │
│  │    GraphDB      │     │              KAFKA                       │       │
│  │  (kb7_v2_repo)  │     │  ┌─────────────────────────────────┐    │       │
│  └────────┬────────┘     │  │  kb7.terminology.releases       │    │       │
│           │              │  └─────────────────────────────────┘    │       │
│           │              └──────────────┬──────────────────────────┘       │
│           │                             │                                   │
│           │              ┌──────────────┼──────────────┐                   │
│           │              ▼              ▼              ▼                   │
│           │     ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│           │     │   Flink    │  │Notification│  │  Neo4j     │            │
│           │     │ Broadcast  │  │  Service   │  │   Sync     │            │
│           │     │  Stream    │  │            │  │  Service   │            │
│           │     └─────┬──────┘  └─────┬──────┘  └─────┬──────┘            │
│           │           │               │               │                    │
│           │           ▼               ▼               ▼                    │
│           │     ┌──────────┐   ┌──────────┐   ┌──────────────┐            │
│           │     │ Patient  │   │  Redis   │   │    Neo4j     │            │
│           └────▶│ Events   │   │  Cache   │   │ kb7_production│           │
│                 │ Enriched │   │ Invalidate│  │   (alias)    │            │
│                 └──────────┘   └──────────┘   └──────────────┘            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Summary

| Component | Technology | Purpose | Location |
|-----------|------------|---------|----------|
| Knowledge Factory | GCP Cloud Run | Build terminology artifacts | `kb-7-terminology/knowledge-factory/` |
| CDC Connector | Debezium | Capture PostgreSQL changes | `kafka/cdc-connectors/configs/kb7-terminology-releases-cdc.json` |
| Flink BroadcastStream | Apache Flink | Hot-swap terminology in streaming | `flink-processing/src/main/java/.../Module_KB7_TerminologyBroadcast.java` |
| Notification Service | Python/Kafka | Notify downstream consumers | `kafka/cdc-connectors/services/terminology_notification_service.py` |
| **N10s RDF Importer** | Neo4j/n10s | Native RDF→LPG import from GCS | `kb-7-terminology/runtime-layer-MOVED-TO-SHARED/adapters/n10s_rdf_importer.py` |
| Neo4j Sync Service | Python | Database aliasing + n10s import | `kb-7-terminology/runtime-layer-MOVED-TO-SHARED/services/neo4j_sync_service.py` |
| Neo4j Projector | Python | Graph mutations to Neo4j | `stream-services/module8-neo4j-graph-projector/` |
| CDC Event Model | Java | Debezium event parsing | `flink-processing/src/main/java/.../cdc/TerminologyReleaseCDCEvent.java` |
| SQL Schema | PostgreSQL | Outbox table definition | `kafka/cdc-connectors/sql/kb7-releases-schema.sql` |

### 1.3 Version Isolation Strategy

```
┌────────────────────────────────────────────────────────────────┐
│                   VERSION ISOLATION LAYERS                      │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  STORAGE LAYER                                                  │
│  ─────────────                                                  │
│  GCS:     gs://kb7-artifacts/v1/  │  gs://kb7-artifacts/v2/    │
│           └── snomed.ttl          │  └── snomed.ttl            │
│           └── rxnorm.ttl          │  └── rxnorm.ttl            │
│           └── loinc.ttl           │  └── loinc.ttl             │
│                                                                 │
│  GRAPHDB LAYER                                                  │
│  ─────────────                                                  │
│  Repositories: kb7_v1_repository  │  kb7_v2_repository         │
│                (archived)         │  (current)                  │
│                                                                 │
│  NEO4J LAYER                                                    │
│  ───────────                                                    │
│  Databases:   kb7_v1 (old)        │  kb7_v2 (new)              │
│                     ▲                    ▲                      │
│                     └────────────────────┘                      │
│                              │                                  │
│  Alias:              kb7_production ────────▶ kb7_v2           │
│                      (clients use this)                         │
│                                                                 │
│  REGISTRY LAYER                                                 │
│  ──────────────                                                 │
│  PostgreSQL kb_releases table:                                  │
│  ┌──────────┬────────┬─────────────┬───────────────────────┐   │
│  │version_id│ status │triple_count │ graphdb_repository    │   │
│  ├──────────┼────────┼─────────────┼───────────────────────┤   │
│  │ v1       │ARCHIVED│  2,500,000  │ kb7_v1_repository     │   │
│  │ v2       │ ACTIVE │  2,750,000  │ kb7_v2_repository     │   │
│  └──────────┴────────┴─────────────┴───────────────────────┘   │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## 2. CDC Pipeline Flow

### 2.1 Commit-Last Strategy

The **Commit-Last Strategy** ensures CDC events are only published after all data loading is complete and validated:

```
┌─────────────────────────────────────────────────────────────────┐
│                    COMMIT-LAST STRATEGY                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Step 1: Load Data                                               │
│  ─────────────────                                               │
│  Knowledge Factory Pipeline:                                     │
│    1. Download SNOMED/RxNorm/LOINC sources                      │
│    2. Convert to RDF/OWL format                                 │
│    3. Load into GraphDB repository                              │
│    4. Run OWL reasoning                                         │
│    5. Validate triple counts                                    │
│                                                                  │
│  Step 2: Health Check                                            │
│  ────────────────────                                            │
│  Before committing to registry:                                  │
│    - SPARQL query returns expected results                      │
│    - Triple count matches expected                              │
│    - No OWL inconsistencies                                     │
│                                                                  │
│  Step 3: Commit to Registry (LAST!)                              │
│  ──────────────────────────────────                              │
│  Only after validation passes:                                   │
│    INSERT INTO kb_releases (                                     │
│      version_id, status, triple_count,                          │
│      graphdb_endpoint, graphdb_repository                       │
│    ) VALUES ('v2', 'ACTIVE', 2750000, ...);                     │
│                                                                  │
│  Step 4: CDC Captures Change                                     │
│  ───────────────────────────                                     │
│  Debezium detects INSERT → publishes to Kafka                   │
│                                                                  │
│  ✓ Downstream consumers NEVER see incomplete data               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 CDC Event Schema

**Kafka Topic**: `kb7.terminology.releases`

```json
{
  "schema": {
    "type": "struct",
    "name": "kb7.public.kb_releases.Envelope"
  },
  "payload": {
    "op": "c",
    "ts_ms": 1701619200000,
    "before": null,
    "after": {
      "version_id": "2024.12.01",
      "status": "ACTIVE",
      "snomed_version": "2024-09-01",
      "rxnorm_version": "2024-11-04",
      "loinc_version": "2.77",
      "triple_count": 2750000,
      "graphdb_endpoint": "http://graphdb:7200",
      "graphdb_repository": "kb7_v2_repository",
      "gcs_uri": "gs://kb7-artifacts/2024.12.01/",
      "created_at": "2024-12-03T12:00:00Z"
    },
    "source": {
      "version": "2.4.0.Final",
      "connector": "postgresql",
      "name": "kb7",
      "ts_ms": 1701619200000,
      "db": "kb7_registry",
      "schema": "public",
      "table": "kb_releases"
    }
  }
}
```

### 2.3 PostgreSQL Table Schema

```sql
-- File: kafka/cdc-connectors/sql/kb7-releases-schema.sql
-- This table acts as the CDC event source for terminology updates
-- Debezium monitors this table and publishes changes to Kafka
--
-- IMPORTANT: Only INSERT into this table AFTER GraphDB is fully loaded and verified
-- This implements the "Commit-Last" strategy to prevent race conditions in EDA

CREATE TABLE IF NOT EXISTS kb_releases (
    id SERIAL PRIMARY KEY,

    -- Version identification
    version_id VARCHAR(50) UNIQUE NOT NULL,  -- e.g., "20251203" or "latest"

    -- Timestamps
    release_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    graphdb_load_started_at TIMESTAMP WITH TIME ZONE,
    graphdb_load_completed_at TIMESTAMP WITH TIME ZONE,

    -- Source terminology versions
    snomed_version VARCHAR(50),      -- e.g., "2024-09"
    rxnorm_version VARCHAR(50),      -- e.g., "12012025"
    loinc_version VARCHAR(50),       -- e.g., "2.77"

    -- Content metrics
    triple_count BIGINT,             -- Total triples in GraphDB
    concept_count INTEGER,           -- Total concepts loaded
    snomed_concept_count INTEGER,
    rxnorm_concept_count INTEGER,
    loinc_concept_count INTEGER,

    -- File information
    kernel_file_size_bytes BIGINT,
    kernel_checksum VARCHAR(64),     -- SHA-256 hash
    gcs_uri VARCHAR(500),            -- gs://bucket/version/kb7-kernel.ttl

    -- GraphDB information
    graphdb_repository VARCHAR(100) DEFAULT 'kb7-terminology',
    graphdb_endpoint VARCHAR(500),

    -- Status tracking
    status VARCHAR(20) DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'LOADING', 'ACTIVE', 'ARCHIVED', 'FAILED')),
    error_message TEXT,

    -- Metadata
    created_by VARCHAR(100) DEFAULT 'kb-factory-pipeline',
    notes TEXT
);

-- Enable CDC on this table (required for Debezium)
ALTER TABLE kb_releases REPLICA IDENTITY FULL;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_kb_releases_version ON kb_releases(version_id);
CREATE INDEX IF NOT EXISTS idx_kb_releases_status ON kb_releases(status);
CREATE INDEX IF NOT EXISTS idx_kb_releases_date ON kb_releases(release_date DESC);

-- Create a view for the current active release
CREATE OR REPLACE VIEW current_kb_release AS
SELECT * FROM kb_releases
WHERE status = 'ACTIVE'
ORDER BY release_date DESC
LIMIT 1;

-- Function to archive previous active releases when a new one becomes active
CREATE OR REPLACE FUNCTION archive_previous_releases()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'ACTIVE' THEN
        UPDATE kb_releases
        SET status = 'ARCHIVED'
        WHERE id != NEW.id AND status = 'ACTIVE';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-archive previous releases
DROP TRIGGER IF EXISTS trigger_archive_previous_releases ON kb_releases;
CREATE TRIGGER trigger_archive_previous_releases
    AFTER UPDATE ON kb_releases
    FOR EACH ROW
    WHEN (NEW.status = 'ACTIVE')
    EXECUTE FUNCTION archive_previous_releases();
```

---

## 3. Flink BroadcastStream Pattern

### 3.1 Dual Stream Concept

The BroadcastStream pattern solves the **reference data join problem** in streaming:

```
┌─────────────────────────────────────────────────────────────────┐
│                  DUAL STREAM ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  STREAM 1: Patient Events (High Volume, Keyed)                   │
│  ══════════════════════════════════════════════                  │
│                                                                  │
│  Characteristics:                                                │
│    • Volume: Millions of events per second                      │
│    • Partitioning: Keyed by patient_id                          │
│    • Processing: Each event goes to ONE parallel task           │
│                                                                  │
│  Source: enriched-patient-events-v1 Kafka topic                 │
│                                                                  │
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐              │
│  │P1 │ │P2 │ │P1 │ │P3 │ │P2 │ │P4 │ │P1 │ │P5 │ ...          │
│  └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘              │
│    │     │     │     │     │     │     │     │                  │
│    ▼     ▼     ▼     ▼     ▼     ▼     ▼     ▼                  │
│  ┌─────────────────────────────────────────────┐                │
│  │           KEYED BY PATIENT_ID               │                │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐   │                │
│  │  │Task 0│  │Task 1│  │Task 2│  │Task 3│   │                │
│  │  │P1,P5 │  │P2    │  │P3    │  │P4    │   │                │
│  │  └──────┘  └──────┘  └──────┘  └──────┘   │                │
│  └─────────────────────────────────────────────┘                │
│                                                                  │
│                                                                  │
│  STREAM 2: Terminology CDC (Low Volume, Broadcast)               │
│  ══════════════════════════════════════════════════              │
│                                                                  │
│  Characteristics:                                                │
│    • Volume: Few events per day (terminology releases)          │
│    • Partitioning: NONE - broadcast to ALL tasks                │
│    • Processing: Every task receives EVERY event                │
│                                                                  │
│  Source: kb7.terminology.releases Kafka topic                   │
│                                                                  │
│  ┌─────────────────────────────────────────────┐                │
│  │  CDC Event: "KB7 v2.1 ACTIVE"               │                │
│  └──────────────────────┬──────────────────────┘                │
│                         │                                        │
│           ┌─────────────┼─────────────┐                         │
│           ▼             ▼             ▼                         │
│  ┌──────────────────────────────────────────────┐               │
│  │          BROADCAST TO ALL TASKS              │               │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐    │               │
│  │  │Task 0│  │Task 1│  │Task 2│  │Task 3│    │               │
│  │  │ v2.1 │  │ v2.1 │  │ v2.1 │  │ v2.1 │    │               │
│  │  └──────┘  └──────┘  └──────┘  └──────┘    │               │
│  └──────────────────────────────────────────────┘               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Implementation Code

**File**: `flink-processing/src/main/java/com/cardiofit/flink/operators/Module_KB7_TerminologyBroadcast.java`

```java
/**
 * Module KB7: Terminology BroadcastStream for CDC-driven hot-swap
 *
 * Purpose: Enriches patient events with terminology context that can be
 * updated at runtime without pipeline restart.
 *
 * Pattern: KeyedBroadcastProcessFunction
 *   - Keyed Stream: Patient events (partitioned by patient_id)
 *   - Broadcast Stream: Terminology CDC events (replicated to all tasks)
 */
public class Module_KB7_TerminologyBroadcast {

    // Broadcast state descriptor - shared across all parallel instances
    private static final MapStateDescriptor<String, TerminologyReleaseCDCEvent>
        TERMINOLOGY_STATE_DESCRIPTOR = new MapStateDescriptor<>(
            "terminology-broadcast-state",
            Types.STRING,
            Types.POJO(TerminologyReleaseCDCEvent.class)
        );

    public static DataStream<EnrichedPatientContext> enrichWithTerminology(
            DataStream<CanonicalEvent> patientEvents,
            DataStream<TerminologyReleaseCDCEvent> terminologyStream,
            StreamExecutionEnvironment env) {

        // Step 1: Key patient events by patient ID
        KeyedStream<CanonicalEvent, String> keyedPatients =
            patientEvents.keyBy(CanonicalEvent::getPatientId);

        // Step 2: Broadcast terminology updates to all parallel instances
        BroadcastStream<TerminologyReleaseCDCEvent> terminologyBroadcast =
            terminologyStream.broadcast(TERMINOLOGY_STATE_DESCRIPTOR);

        // Step 3: Connect and process both streams
        return keyedPatients
            .connect(terminologyBroadcast)
            .process(new TerminologyEnrichmentProcessor())
            .name("KB7-Terminology-Enrichment")
            .uid("kb7-terminology-enrichment");
    }

    /**
     * Processor that joins patient events with broadcast terminology state
     */
    private static class TerminologyEnrichmentProcessor
            extends KeyedBroadcastProcessFunction<
                String,                          // Key type (patient_id)
                CanonicalEvent,                  // Input 1: Patient events
                TerminologyReleaseCDCEvent,      // Input 2: Terminology CDC
                EnrichedPatientContext> {        // Output: Enriched events

        private static final String ACTIVE_VERSION_KEY = "ACTIVE";

        /**
         * Called for each patient event - enriches with current terminology
         */
        @Override
        public void processElement(
                CanonicalEvent event,
                ReadOnlyContext ctx,
                Collector<EnrichedPatientContext> out) throws Exception {

            // Read broadcast state (read-only in processElement)
            ReadOnlyBroadcastState<String, TerminologyReleaseCDCEvent> state =
                ctx.getBroadcastState(TERMINOLOGY_STATE_DESCRIPTOR);

            // Get current active terminology version
            TerminologyReleaseCDCEvent activeTerminology =
                state.get(ACTIVE_VERSION_KEY);

            // Create enriched context with terminology
            EnrichedPatientContext enriched = new EnrichedPatientContext();
            enriched.setPatientId(event.getPatientId());
            enriched.setEventType(event.getEventType());

            // Add terminology context if available
            if (activeTerminology != null) {
                Map<String, Object> terminologyContext = new HashMap<>();
                terminologyContext.put("terminology_version",
                    activeTerminology.getVersionId());
                terminologyContext.put("snomed_version",
                    activeTerminology.getSnomedVersion());
                terminologyContext.put("rxnorm_version",
                    activeTerminology.getRxnormVersion());
                terminologyContext.put("loinc_version",
                    activeTerminology.getLoincVersion());
                terminologyContext.put("graphdb_endpoint",
                    activeTerminology.getGraphdbEndpoint());
                terminologyContext.put("graphdb_repository",
                    activeTerminology.getGraphdbRepository());

                enriched.setTerminologyContext(terminologyContext);
            }

            out.collect(enriched);
        }

        /**
         * Called for each terminology CDC event - updates broadcast state
         * This runs on ALL parallel instances simultaneously
         */
        @Override
        public void processBroadcastElement(
                TerminologyReleaseCDCEvent cdcEvent,
                Context ctx,
                Collector<EnrichedPatientContext> out) throws Exception {

            // Only process ACTIVE status transitions
            if (!"ACTIVE".equals(cdcEvent.getStatus())) {
                return;
            }

            // Get mutable broadcast state
            BroadcastState<String, TerminologyReleaseCDCEvent> state =
                ctx.getBroadcastState(TERMINOLOGY_STATE_DESCRIPTOR);

            // Log the hot-swap
            LOG.info("🔄 Terminology hot-swap: {} → {} (SNOMED: {}, RxNorm: {})",
                state.get(ACTIVE_VERSION_KEY) != null ?
                    state.get(ACTIVE_VERSION_KEY).getVersionId() : "none",
                cdcEvent.getVersionId(),
                cdcEvent.getSnomedVersion(),
                cdcEvent.getRxnormVersion());

            // Update broadcast state - immediately visible to all tasks
            state.put(ACTIVE_VERSION_KEY, cdcEvent);
        }
    }
}
```

### 3.3 Side Outputs and Notification Sink

The Flink implementation uses **side outputs** to emit terminology update notifications independently from the main enriched event stream:

```
┌─────────────────────────────────────────────────────────────────┐
│              FLINK SIDE OUTPUT ARCHITECTURE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  TerminologyEnrichmentProcessor                          │    │
│  │  (KeyedBroadcastProcessFunction)                        │    │
│  │                                                          │    │
│  │  processBroadcastElement() ──┬─── BroadcastState Update │    │
│  │                              └─── Side Output Notification│    │
│  │                                                          │    │
│  │  processElement() ────────────── Main Output (Enriched) │    │
│  └───────────────────────────────────────────────────────────┘    │
│                    │                        │                     │
│                    ▼                        ▼                     │
│         ┌─────────────────┐      ┌──────────────────────┐       │
│         │ MAIN OUTPUT     │      │ SIDE OUTPUT          │       │
│         │                 │      │                      │       │
│         │ Kafka Topic:    │      │ Kafka Topic:         │       │
│         │ terminology-    │      │ terminology-version- │       │
│         │ enriched-       │      │ updates.v1           │       │
│         │ events.v1       │      │                      │       │
│         └─────────────────┘      └──────────────────────┘       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Implementation Code**:

```java
// File: Module_KB7_TerminologyBroadcast.java

// Define side output tag for notifications
private static final OutputTag<TerminologyUpdateNotification> NOTIFICATION_OUTPUT =
        new OutputTag<TerminologyUpdateNotification>("terminology-notifications") {};

// In pipeline creation:
SingleOutputStreamOperator<EnrichedPatientContext> enrichedWithTerminology = clinicalEvents
        .keyBy(EnrichedPatientContext::getPatientId)
        .connect(releaseBroadcastStream)
        .process(new TerminologyEnrichmentProcessor())
        .uid("terminology-enrichment-processor")
        .name("Terminology Enrichment with CDC Hot-Swap");

// Main output: enriched events
enrichedWithTerminology.sinkTo(createEnrichedEventsSink())
        .uid("terminology-enriched-events-sink")
        .name("Terminology-Enriched Events Sink");

// Side output: notifications about terminology updates
DataStream<TerminologyUpdateNotification> notifications =
        enrichedWithTerminology.getSideOutput(NOTIFICATION_OUTPUT);

notifications.sinkTo(createNotificationsSink())
        .uid("terminology-notifications-sink")
        .name("Terminology Update Notifications Sink");
```

**Notification Payload**:

```java
public class TerminologyUpdateNotification {
    private String versionId;
    private String snomedVersion;
    private String rxnormVersion;
    private String loincVersion;
    private Long tripleCount;
    private String graphdbEndpoint;
    private long notificationTime;
}
```

**Output Topics**:

| Topic | Purpose | Consumer |
|-------|---------|----------|
| `terminology-enriched-events.v1` | Patient events enriched with terminology context | Downstream clinical modules |
| `terminology-version-updates.v1` | Notifications when terminology version changes | Notification Service, Cache Invalidation |

### 3.4 Race Condition Prevention

The BroadcastStream pattern prevents race conditions that would occur with other approaches:

```
┌─────────────────────────────────────────────────────────────────┐
│              RACE CONDITION ANALYSIS                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ❌ APPROACH 1: Database Lookup Per Event                        │
│  ───────────────────────────────────────                         │
│                                                                  │
│  Time →                                                          │
│  Task 0: [Event] → DB Query → Uses v1                           │
│  Task 1: [Event] → DB Query → [DB Updated to v2] → Uses v2      │
│  Task 2: [Event] → DB Query → Uses v2                           │
│                                                                  │
│  Problem: Inconsistent versions within same time window          │
│  Also: High latency, DB overload                                │
│                                                                  │
│                                                                  │
│  ❌ APPROACH 2: Merge Streams (Keyed CDC)                        │
│  ─────────────────────────────────────────                       │
│                                                                  │
│  Time →                                                          │
│  Task 0: [P1 Event] ───────────────────── v1 (no CDC received)  │
│  Task 1: [CDC v2] ──────────────────────── v2 (only this task)  │
│  Task 2: [P2 Event] ───────────────────── v1 (no CDC received)  │
│                                                                  │
│  Problem: CDC keyed → only one task receives update             │
│                                                                  │
│                                                                  │
│  ✅ APPROACH 3: BroadcastStream (Implemented)                    │
│  ─────────────────────────────────────────────                   │
│                                                                  │
│  Time →                                                          │
│  Task 0: ─────────── [CDC v2] ─── [P1 Event] → v2 ✓             │
│  Task 1: ─────────── [CDC v2] ─── [P2 Event] → v2 ✓             │
│  Task 2: ─────────── [CDC v2] ─── [P3 Event] → v2 ✓             │
│                       ↑                                          │
│                       │                                          │
│          All tasks receive CDC at same logical time              │
│                                                                  │
│  Result: Perfect consistency across all parallel instances       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Neo4j Database Aliasing

### 4.1 Alias Swap Pattern

Neo4j Database Aliasing enables **atomic database switching** without client reconfiguration:

```
┌─────────────────────────────────────────────────────────────────┐
│                 NEO4J DATABASE ALIASING PATTERN                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  BEFORE SWITCH                          AFTER SWITCH             │
│  ─────────────                          ────────────             │
│                                                                  │
│  ┌─────────────┐                        ┌─────────────┐         │
│  │ kb7_v1      │ ◀── kb7_production     │ kb7_v1      │         │
│  │ (active)    │     (alias)            │ (archived)  │         │
│  └─────────────┘                        └─────────────┘         │
│                                                                  │
│  ┌─────────────┐                        ┌─────────────┐         │
│  │ kb7_v2      │                        │ kb7_v2      │ ◀── kb7_production
│  │ (loading)   │                        │ (active)    │     (alias)
│  └─────────────┘                        └─────────────┘         │
│                                                                  │
│                    ATOMIC SWITCH                                 │
│                    ════════════                                  │
│                                                                  │
│  Cypher Command:                                                 │
│  ALTER ALIAS kb7_production SET DATABASE = kb7_v2               │
│                                                                  │
│  Properties:                                                     │
│  • Zero downtime - no connection drops                          │
│  • Atomic - all queries switch instantly                        │
│  • No client changes - always use "kb7_production"              │
│  • Rollback capable - just re-point alias                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Five-Phase Implementation

```
┌─────────────────────────────────────────────────────────────────┐
│                   FIVE-PHASE DATABASE SWITCH                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  PHASE 1: CREATE DATABASE                                        │
│  ─────────────────────────                                       │
│  CREATE DATABASE kb7_v2                                         │
│                                                                  │
│  Triggered by: CDC event with status='LOADING'                  │
│  Duration: ~1 second                                            │
│                                                                  │
│  ┌─────────────┐                                                │
│  │ kb7_v2      │ ← Empty database created                       │
│  │ (empty)     │                                                │
│  └─────────────┘                                                │
│                                                                  │
│                                                                  │
│  PHASE 2: IMPORT DATA                                            │
│  ────────────────────                                            │
│  GraphDB SPARQL → Transform → Neo4j Cypher                      │
│                                                                  │
│  Triggered by: Phase 1 completion                               │
│  Duration: 10-30 minutes (depending on data size)               │
│                                                                  │
│  ┌─────────────┐     ┌─────────────┐                           │
│  │ GraphDB     │ ──▶ │ kb7_v2      │                           │
│  │ kb7_v2_repo │     │ (loading)   │                           │
│  └─────────────┘     └─────────────┘                           │
│                                                                  │
│                                                                  │
│  PHASE 3: VALIDATE                                               │
│  ─────────────────                                               │
│  Health checks:                                                  │
│  • Node counts match expected                                   │
│  • Relationship counts match expected                           │
│  • Sample queries return correct results                        │
│  • Index creation verified                                      │
│                                                                  │
│  Triggered by: Phase 2 completion                               │
│  Duration: ~2 minutes                                           │
│                                                                  │
│  Validation Queries:                                            │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ MATCH (d:Drug) RETURN count(d) as drug_count           │    │
│  │ MATCH (c:Concept) RETURN count(c) as concept_count     │    │
│  │ MATCH ()-[r:INTERACTS_WITH]->() RETURN count(r)        │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│                                                                  │
│  PHASE 4: SWITCH ALIAS (ATOMIC!)                                 │
│  ───────────────────────────────                                 │
│  ALTER ALIAS kb7_production SET DATABASE = kb7_v2               │
│                                                                  │
│  Triggered by: CDC event with status='ACTIVE' + Phase 3 pass   │
│  Duration: <100ms                                               │
│                                                                  │
│  ┌─────────────┐                                                │
│  │ kb7_production │ ──────────────▶ kb7_v2                      │
│  │ (alias)     │                                                │
│  └─────────────┘                                                │
│                                                                  │
│                                                                  │
│  PHASE 5: CLEANUP                                                │
│  ────────────────                                                │
│  DROP DATABASE kb7_v1 (after grace period)                      │
│                                                                  │
│  Triggered by: Scheduled job (24 hours after switch)            │
│  Duration: ~5 seconds                                           │
│                                                                  │
│  Cleanup Policy:                                                 │
│  • Keep old database for 24 hours (rollback window)             │
│  • Verify no active connections                                 │
│  • Drop database and reclaim storage                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Implementation Code

**File**: `kb-7-terminology/runtime-layer/services/neo4j_sync_service.py`

```python
"""
Neo4j Terminology Sync Service
Implements 5-phase database aliasing for zero-downtime terminology switching
"""

import asyncio
from neo4j import AsyncGraphDatabase
from typing import Dict, Any, Optional
from datetime import datetime, timedelta
import redis.asyncio as redis
import structlog

logger = structlog.get_logger(__name__)


class Neo4jTerminologySyncService:
    """
    Orchestrates Neo4j database creation, import, and alias switching
    for terminology version management.
    """

    PRODUCTION_ALIAS = "kb7_production"
    CLEANUP_GRACE_HOURS = 24

    def __init__(self, config: Dict[str, Any]):
        self.driver = AsyncGraphDatabase.driver(
            config['neo4j_uri'],
            auth=(config['neo4j_user'], config['neo4j_password'])
        )
        self.graphdb_adapter = GraphDBToNeo4jAdapter(
            config['graphdb_url'],
            self
        )
        self.redis = redis.from_url(config.get('redis_url', 'redis://localhost:6379'))
        self.config = config

    async def handle_cdc_event(self, event: Dict[str, Any]) -> bool:
        """
        Handle terminology CDC event and trigger appropriate phase

        Args:
            event: Debezium CDC event payload

        Returns:
            Success status
        """
        status = event.get('status')
        version_id = event.get('version_id')

        logger.info(
            "Processing terminology CDC event",
            version_id=version_id,
            status=status
        )

        if status == 'LOADING':
            # Phase 1 & 2: Create database and start import
            return await self._phase_1_2_create_and_import(event)

        elif status == 'ACTIVE':
            # Phase 3 & 4: Validate and switch alias
            return await self._phase_3_4_validate_and_switch(event)

        elif status == 'ARCHIVED':
            # Phase 5: Schedule cleanup
            return await self._phase_5_schedule_cleanup(event)

        return True

    # ══════════════════════════════════════════════════════════════
    # PHASE 1 & 2: CREATE DATABASE AND IMPORT
    # ══════════════════════════════════════════════════════════════

    async def _phase_1_2_create_and_import(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """
        Phase 1: Create new database
        Phase 2: Import data from GraphDB
        """
        version_id = event['version_id']
        db_name = f"kb7_{version_id.replace('.', '_').replace('-', '_')}"

        try:
            # Phase 1: Create database
            logger.info(f"Phase 1: Creating database {db_name}")
            await self._create_database(db_name)

            # Phase 2: Import from GraphDB
            logger.info(f"Phase 2: Importing data to {db_name}")
            graphdb_repo = event.get('graphdb_repository')

            stats = await self.graphdb_adapter.sync_to_database(
                graphdb_repo,
                db_name
            )

            logger.info(
                "Phase 2 complete",
                db_name=db_name,
                stats=stats
            )

            # Store import stats for validation
            await self.redis.hset(
                f"neo4j:import:{version_id}",
                mapping={
                    'db_name': db_name,
                    'drug_count': str(stats.get('drug_concepts', 0)),
                    'interaction_count': str(stats.get('interactions', 0)),
                    'imported_at': datetime.utcnow().isoformat()
                }
            )

            return True

        except Exception as e:
            logger.error(
                "Phase 1/2 failed",
                error=str(e),
                version_id=version_id
            )
            return False

    async def _create_database(self, db_name: str) -> None:
        """Create a new Neo4j database"""
        async with self.driver.session(database="system") as session:
            # Check if database already exists
            result = await session.run(
                "SHOW DATABASES WHERE name = $name",
                name=db_name
            )
            existing = await result.single()

            if existing:
                logger.warning(f"Database {db_name} already exists")
                return

            # Create new database
            await session.run(f"CREATE DATABASE {db_name}")

            # Wait for database to be online
            for _ in range(30):
                result = await session.run(
                    "SHOW DATABASE $name",
                    name=db_name
                )
                record = await result.single()
                if record and record['currentStatus'] == 'online':
                    break
                await asyncio.sleep(1)

            logger.info(f"Database {db_name} created and online")

    # ══════════════════════════════════════════════════════════════
    # PHASE 3 & 4: VALIDATE AND SWITCH ALIAS
    # ══════════════════════════════════════════════════════════════

    async def _phase_3_4_validate_and_switch(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """
        Phase 3: Validate imported data
        Phase 4: Switch alias to new database
        """
        version_id = event['version_id']

        # Get database name from import metadata
        import_meta = await self.redis.hgetall(f"neo4j:import:{version_id}")
        if not import_meta:
            logger.error(f"No import metadata found for {version_id}")
            return False

        db_name = import_meta.get(b'db_name', b'').decode()
        expected_drugs = int(import_meta.get(b'drug_count', b'0'))

        try:
            # Phase 3: Validate
            logger.info(f"Phase 3: Validating database {db_name}")
            validation = await self._validate_database(
                db_name,
                expected_drugs
            )

            if not validation['passed']:
                logger.error(
                    "Phase 3 validation failed",
                    validation=validation
                )
                return False

            logger.info("Phase 3 validation passed", validation=validation)

            # Get current active database for rollback tracking
            old_db = await self._get_current_alias_target()

            # Phase 4: Switch alias
            logger.info(f"Phase 4: Switching alias to {db_name}")
            await self._switch_alias(db_name)

            # Store old database for cleanup scheduling
            if old_db:
                await self.redis.setex(
                    f"neo4j:pending_cleanup:{old_db}",
                    self.CLEANUP_GRACE_HOURS * 3600,
                    datetime.utcnow().isoformat()
                )

            # Post-switch health check
            if not await self._post_switch_health_check():
                # Rollback!
                logger.error("Post-switch health check failed, rolling back")
                await self._switch_alias(old_db)
                return False

            logger.info(
                "Phase 4 complete - alias switched",
                old_db=old_db,
                new_db=db_name
            )

            return True

        except Exception as e:
            logger.error(
                "Phase 3/4 failed",
                error=str(e),
                version_id=version_id
            )
            return False

    async def _validate_database(
        self,
        db_name: str,
        expected_drugs: int
    ) -> Dict[str, Any]:
        """Validate database contents"""
        validation = {
            'passed': True,
            'checks': {}
        }

        async with self.driver.session(database=db_name) as session:
            # Check drug count
            result = await session.run(
                "MATCH (d:Drug) RETURN count(d) as count"
            )
            record = await result.single()
            actual_drugs = record['count']

            validation['checks']['drug_count'] = {
                'expected': expected_drugs,
                'actual': actual_drugs,
                'passed': actual_drugs >= expected_drugs * 0.95  # 5% tolerance
            }

            # Check interaction count
            result = await session.run(
                "MATCH ()-[r:INTERACTS_WITH]->() RETURN count(r) as count"
            )
            record = await result.single()
            validation['checks']['interaction_count'] = {
                'actual': record['count'],
                'passed': record['count'] > 0
            }

            # Check indexes exist
            result = await session.run("SHOW INDEXES")
            indexes = [r async for r in result]
            validation['checks']['indexes'] = {
                'count': len(indexes),
                'passed': len(indexes) >= 5
            }

            # Overall pass/fail
            validation['passed'] = all(
                check['passed']
                for check in validation['checks'].values()
            )

        return validation

    async def _switch_alias(self, target_db: str) -> None:
        """Atomically switch the production alias to target database"""
        async with self.driver.session(database="system") as session:
            # Check if alias exists
            result = await session.run(
                "SHOW ALIASES WHERE name = $alias",
                alias=self.PRODUCTION_ALIAS
            )
            existing = await result.single()

            if existing:
                # Update existing alias
                await session.run(
                    f"ALTER ALIAS {self.PRODUCTION_ALIAS} "
                    f"SET DATABASE = {target_db}"
                )
            else:
                # Create new alias
                await session.run(
                    f"CREATE ALIAS {self.PRODUCTION_ALIAS} "
                    f"FOR DATABASE {target_db}"
                )

            logger.info(
                "Alias switched",
                alias=self.PRODUCTION_ALIAS,
                target=target_db
            )

    async def _get_current_alias_target(self) -> Optional[str]:
        """Get the database currently pointed to by production alias"""
        async with self.driver.session(database="system") as session:
            result = await session.run(
                "SHOW ALIASES WHERE name = $alias",
                alias=self.PRODUCTION_ALIAS
            )
            record = await result.single()
            return record['database'] if record else None

    async def _post_switch_health_check(self) -> bool:
        """Verify production alias is working after switch"""
        try:
            async with self.driver.session(
                database=self.PRODUCTION_ALIAS
            ) as session:
                result = await session.run("RETURN 1 as test")
                await result.single()

                # Quick query test
                result = await session.run(
                    "MATCH (d:Drug) RETURN count(d) as count LIMIT 1"
                )
                await result.single()

            return True
        except Exception as e:
            logger.error(f"Post-switch health check failed: {e}")
            return False

    # ══════════════════════════════════════════════════════════════
    # PHASE 5: CLEANUP
    # ══════════════════════════════════════════════════════════════

    async def _phase_5_schedule_cleanup(
        self,
        event: Dict[str, Any]
    ) -> bool:
        """Schedule old database for cleanup after grace period"""
        version_id = event['version_id']

        import_meta = await self.redis.hgetall(f"neo4j:import:{version_id}")
        if not import_meta:
            return True

        db_name = import_meta.get(b'db_name', b'').decode()

        # Schedule cleanup
        cleanup_time = datetime.utcnow() + timedelta(
            hours=self.CLEANUP_GRACE_HOURS
        )

        await self.redis.zadd(
            "neo4j:scheduled_cleanups",
            {db_name: cleanup_time.timestamp()}
        )

        logger.info(
            "Scheduled database cleanup",
            db_name=db_name,
            cleanup_time=cleanup_time.isoformat()
        )

        return True

    async def run_cleanup_job(self) -> int:
        """
        Run scheduled cleanup job (call from cron/scheduler)
        Returns number of databases cleaned up
        """
        now = datetime.utcnow().timestamp()

        # Get databases due for cleanup
        due = await self.redis.zrangebyscore(
            "neo4j:scheduled_cleanups",
            0,
            now
        )

        cleaned = 0
        for db_name_bytes in due:
            db_name = db_name_bytes.decode()

            try:
                # Verify not currently in use
                current = await self._get_current_alias_target()
                if current == db_name:
                    logger.warning(
                        f"Skipping cleanup of {db_name} - still in use"
                    )
                    continue

                # Drop database
                async with self.driver.session(database="system") as session:
                    await session.run(f"DROP DATABASE {db_name}")

                # Remove from scheduled cleanups
                await self.redis.zrem("neo4j:scheduled_cleanups", db_name)

                logger.info(f"Cleaned up database {db_name}")
                cleaned += 1

            except Exception as e:
                logger.error(f"Cleanup failed for {db_name}: {e}")

        return cleaned

    async def rollback(self, to_version: str) -> bool:
        """
        Emergency rollback to previous version

        Args:
            to_version: Version ID to rollback to

        Returns:
            Success status
        """
        import_meta = await self.redis.hgetall(f"neo4j:import:{to_version}")
        if not import_meta:
            logger.error(f"No import metadata for rollback version {to_version}")
            return False

        db_name = import_meta.get(b'db_name', b'').decode()

        try:
            # Verify database still exists
            async with self.driver.session(database="system") as session:
                result = await session.run(
                    "SHOW DATABASES WHERE name = $name",
                    name=db_name
                )
                if not await result.single():
                    logger.error(f"Rollback database {db_name} not found")
                    return False

            # Switch alias back
            await self._switch_alias(db_name)

            logger.info(f"Rolled back to {to_version} ({db_name})")
            return True

        except Exception as e:
            logger.error(f"Rollback failed: {e}")
            return False

    async def close(self):
        """Close connections"""
        await self.driver.close()
        await self.redis.close()
```

### 4.4 Client Configuration

All clients should use the **alias name**, never versioned database names:

```python
# ✅ CORRECT - Use alias
driver = GraphDatabase.driver(uri, auth=auth)
with driver.session(database="kb7_production") as session:
    result = session.run("MATCH (d:Drug) RETURN d")

# ❌ WRONG - Never use versioned names
with driver.session(database="kb7_v2") as session:  # Don't do this!
    result = session.run("MATCH (d:Drug) RETURN d")
```

---

## 5. Component Implementation

### 5.1 Terminology Notification Service

**File**: `kafka/cdc-connectors/services/terminology_notification_service.py`

This service consumes CDC events and notifies downstream consumers via three channels:

```python
"""
Notification Channels:
1. HTTP Webhooks - Direct notification to KB services
2. Redis Pub/Sub - Real-time cache invalidation
3. Kafka Topic - Async notification propagation
"""

class TerminologyNotificationService:
    KB_SERVICE_WEBHOOKS = {
        'KB1': 'http://kb1:8081/webhooks/terminology-update',
        'KB2': 'http://kb2:8086/webhooks/terminology-update',
        'KB3': 'http://kb3:8087/webhooks/terminology-update',
        'KB4': 'http://kb4:8088/webhooks/terminology-update',
        'KB5': 'http://kb5:8089/webhooks/terminology-update',
        'KB6': 'http://kb6:8091/webhooks/terminology-update',
        'KB7': 'http://kb7:8092/webhooks/terminology-update',
    }

    async def _notify_all(self, event):
        """Send notifications via all channels concurrently"""
        await asyncio.gather(
            self._notify_via_webhooks(notification),
            self._notify_via_redis(notification),
            self._notify_via_kafka(notification),
            return_exceptions=True
        )
```

### 5.2 Neosemantics (n10s) RDF Import

**File**: `kb-7-terminology/runtime-layer-MOVED-TO-SHARED/adapters/n10s_rdf_importer.py`

> ⚠️ **Architecture Update (v2.0)**: The previous Python-based SPARQL→Cypher adapter has been replaced
> with the native Neo4j neosemantics (n10s) plugin for superior performance and reliability.

This is a **Polyglot Persistence** architecture:
- **GraphDB (RDF)**: Semantic "Source of Truth" - Reasoning, Ontologies
- **Neo4j (LPG)**: "Application Cache" - High-speed traversals, Graph Algorithms

**Import Flow**:

```
┌─────────────────────────────────────────────────────────────────┐
│           N10S NATIVE RDF IMPORT ARCHITECTURE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐     ┌─────────────────┐                    │
│  │   Knowledge     │     │      GCS        │                    │
│  │    Factory      │────▶│   kb7-kernel.ttl│                    │
│  │    Pipeline     │     │   (TTL files)   │                    │
│  └─────────────────┘     └────────┬────────┘                    │
│                                   │                              │
│                                   │ Signed URL                   │
│                                   ▼                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                Neo4j Sync Service                        │    │
│  │                                                          │    │
│  │  1. Generate GCS Signed URL                             │    │
│  │  2. CREATE DATABASE kb7_{version}                       │    │
│  │  3. CALL n10s.graphconfig.init()                        │    │
│  │  4. CALL n10s.rdf.import.fetch(signed_url, 'Turtle')    │    │
│  │  5. Neo4j pulls TTL directly from GCS                   │    │
│  │  6. n10s converts RDF → Property Graph                  │    │
│  │                                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                   │                              │
│                                   ▼                              │
│                          ┌─────────────────┐                    │
│                          │     Neo4j       │                    │
│                          │   :Resource     │                    │
│                          │   :Class        │                    │
│                          │   :Drug         │                    │
│                          └─────────────────┘                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Why n10s over Python Adapter?**

| Aspect | Python Adapter (Deprecated) | n10s (Current) |
|--------|---------------------------|----------------|
| Performance | Slow (SPARQL→Python→Cypher) | Fast (native bulk import) |
| Complexity | High (custom transformation) | Low (declarative config) |
| Reliability | Manual error handling | Built-in robustness |
| Semantics | Lost during transformation | Preserved with `handleVocabUris` |

**Prerequisites** (Neo4j Configuration):

```cypher
-- 1. Create uniqueness constraint (essential for performance)
CREATE CONSTRAINT n10s_unique_uri IF NOT EXISTS
FOR (r:Resource) REQUIRE r.uri IS UNIQUE;

-- 2. Initialize n10s graph config
CALL n10s.graphconfig.init({
  handleVocabUris: 'SHORTEN',
  applyNeo4jNaming: true,
  multivalPropList: ['http://www.w3.org/2000/01/rdf-schema#label']
});
```

**Import Implementation**:

```python
class Neo4jTerminologySyncService:
    """
    Uses n10s.rdf.import.fetch() to let Neo4j pull TTL files
    directly from GCS via signed URLs.
    """

    async def _import_rdf_via_n10s(self, db_name: str, signed_url: str):
        async with self.driver.session(database=db_name) as session:
            result = await session.run("""
                CALL n10s.rdf.import.fetch($url, 'Turtle', { verifyUriSyntax: false })
            """, url=signed_url)

            record = await result.single()
            return {
                'triples_parsed': record.get('triplesLoaded', 0),
                'termination_status': record.get('terminationStatus', 'OK')
            }

    def _generate_signed_url(self, version_id: str) -> str:
        """Generate GCS signed URL for private bucket access"""
        blob_name = f"{version_id}/kb7-kernel.ttl"
        blob = self.gcs_client.bucket(self.gcs_bucket).blob(blob_name)
        return blob.generate_signed_url(version="v4", expiration=3600, method="GET")
```

**RDF → Neo4j Transformation Example**:

Input (RDF Turtle):
```turtle
rxnorm:123 a owl:Class ;
    rdfs:label "Aspirin 81mg" ;
    owl:sameAs snomed:387345001 .
```

Output (Neo4j Property Graph):
- **Node**: Labels `:Resource`, `:Class`
- **Properties**: `uri: "http://.../123"`, `label: "Aspirin 81mg"`
- **Relationship**: `(rxnorm:123)-[:owl__sameAs]->(snomed:387345001)`

**Deprecated Adapter**:

The old Python adapter (`graphdb_neo4j_adapter_deprecated.py`) is kept for backward compatibility
but should NOT be used for new implementations. It will emit a deprecation warning when imported.

**Neo4j Output** (Cypher):

```cypher
MERGE (d1:Drug:SemanticStream {rxnorm: $drug1_rxnorm})
MERGE (d2:Drug:SemanticStream {rxnorm: $drug2_rxnorm})
MERGE (d1)-[i:INTERACTS_WITH]-(d2)
SET i.severity = $severity,
    i.mechanism = $mechanism,
    i.source = 'GraphDB',
    i.updated = datetime()
```

### 5.3 Files Modified for CDC Integration

| File | Changes |
|------|---------|
| `setup-postgresql-cdc.sh` | Added KB4/KB5 to setup/cleanup loops |
| `EnrichedPatientContext.java` | Added `terminologyContext` field |
| `Module_KB7_TerminologyBroadcast.java` | BroadcastStream implementation |
| `projector.py` | Neo4j Projector (use alias) |
| `dual_stream_manager.py` | Label-based partitioning |
| `n10s_rdf_importer.py` | n10s RDF→Neo4j import |
| `graphdb_neo4j_adapter.py` | Deprecated shim (backward compat) |

---

## 6. Integration Points

### 6.1 CDC → Flink Integration

```
┌────────────────────────────────────────────────────────────────┐
│                  CDC → FLINK INTEGRATION                        │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Kafka Topic: kb7.terminology.releases                         │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  FlinkKafkaConsumer<TerminologyReleaseCDCEvent>     │       │
│  │                                                      │       │
│  │  Properties:                                         │       │
│  │    bootstrap.servers: kafka:9092                    │       │
│  │    group.id: flink-terminology-consumer             │       │
│  │    auto.offset.reset: earliest                      │       │
│  └───────────────────────────┬─────────────────────────┘       │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  .broadcast(terminologyStateDescriptor)             │       │
│  │                                                      │       │
│  │  Creates BroadcastStream that replicates            │       │
│  │  CDC events to all parallel instances               │       │
│  └───────────────────────────┬─────────────────────────┘       │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  KeyedBroadcastProcessFunction                       │       │
│  │                                                      │       │
│  │  processBroadcastElement():                         │       │
│  │    Updates shared state with new terminology        │       │
│  │                                                      │       │
│  │  processElement():                                  │       │
│  │    Enriches patient events with terminology context │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### 6.2 CDC → Neo4j Integration

```
┌────────────────────────────────────────────────────────────────┐
│                  CDC → NEO4J INTEGRATION                        │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Kafka Topic: kb7.terminology.releases                         │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  TerminologyNotificationService                      │       │
│  │                                                      │       │
│  │  Consumes CDC events                                │       │
│  │  Triggers Neo4j Sync Service                        │       │
│  └───────────────────────────┬─────────────────────────┘       │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  Neo4jTerminologySyncService                         │       │
│  │                                                      │       │
│  │  handle_cdc_event():                                │       │
│  │    status='LOADING' → Phase 1 & 2                   │       │
│  │    status='ACTIVE'  → Phase 3 & 4                   │       │
│  │    status='ARCHIVED'→ Phase 5                       │       │
│  └───────────────────────────┬─────────────────────────┘       │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  Neo4j Enterprise                                    │       │
│  │                                                      │       │
│  │  Databases: kb7_v1, kb7_v2, ...                     │       │
│  │  Alias: kb7_production → (current active)           │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## 7. Deployment Guide

### 7.1 Prerequisites

```bash
# Required infrastructure
- Apache Kafka 3.5+
- Apache Flink 1.17+
- Neo4j Enterprise 5.x (for database aliasing)
- PostgreSQL 14+ (with logical replication)
- Redis 7+
- Debezium 2.4+
```

### 7.2 Deployment Steps

```bash
# Step 1: Deploy CDC Connectors
cd backend/shared-infrastructure/kafka/cdc-connectors
./scripts/deploy-all-cdc-connectors.sh

# Step 2: Create Kafka Topics
./scripts/create-kafka-topics.sh

# Step 3: Setup PostgreSQL CDC
./scripts/setup-postgresql-cdc.sh setup

# Step 4: Deploy Notification Service
cd services
./run_notification_service.sh

# Step 5: Deploy Neo4j Sync Service
cd ../../knowledge-base-services/kb-7-terminology/runtime-layer
python -m services.neo4j_sync_service

# Step 6: Deploy Flink Job
cd ../../../../flink-processing
./deploy-modules-1-2.sh  # Includes KB7 BroadcastStream
```

### 7.3 Environment Variables

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=kafka:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT

# Neo4j
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# PostgreSQL (CDC source)
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=cdc_user
POSTGRES_PASSWORD=password
POSTGRES_DB=kb7_registry

# Redis
REDIS_URL=redis://redis:6379

# GraphDB
GRAPHDB_URL=http://graphdb:7200
```

---

## 8. Monitoring & Operations

### 8.1 Key Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `cdc.events.processed` | CDC events processed | N/A |
| `cdc.events.failed` | Failed CDC events | > 0 |
| `flink.broadcast.updates` | Terminology hot-swaps | N/A |
| `neo4j.alias.switches` | Database alias switches | N/A |
| `neo4j.validation.failures` | Failed validations | > 0 |
| `terminology.version.active` | Current active version | N/A |

### 8.2 Health Checks

```bash
# Check CDC connector status
curl http://kafka-connect:8083/connectors/kb7-terminology-cdc/status

# Check Flink job status
curl http://flink-jobmanager:8081/jobs

# Check Neo4j alias
cypher-shell -d system "SHOW ALIASES"

# Check current active version
redis-cli GET terminology:current_version
```

### 8.3 Operational Commands

```bash
# Manual terminology version switch (emergency)
python -c "
from services.neo4j_sync_service import Neo4jTerminologySyncService
import asyncio
svc = Neo4jTerminologySyncService(config)
asyncio.run(svc._switch_alias('kb7_v1'))
"

# Rollback to previous version
python -c "
from services.neo4j_sync_service import Neo4jTerminologySyncService
import asyncio
svc = Neo4jTerminologySyncService(config)
asyncio.run(svc.rollback('2024.11.01'))
"

# Force cleanup of old database
python -c "
from services.neo4j_sync_service import Neo4jTerminologySyncService
import asyncio
svc = Neo4jTerminologySyncService(config)
asyncio.run(svc.run_cleanup_job())
"
```

---

## 9. Troubleshooting

### 9.1 Common Issues

#### CDC Events Not Arriving

```bash
# Check Debezium connector
curl http://kafka-connect:8083/connectors/kb7-terminology-cdc/status

# Check PostgreSQL replication slot
psql -c "SELECT * FROM pg_replication_slots WHERE slot_name LIKE 'debezium%';"

# Check Kafka topic
kafka-console-consumer --bootstrap-server kafka:9092 \
  --topic kb7.terminology.releases --from-beginning --max-messages 5
```

#### Flink Not Receiving Broadcast Updates

```bash
# Check Flink consumer group lag
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --describe --group flink-terminology-consumer

# Check Flink task manager logs
kubectl logs -l component=taskmanager -c flink-main-container
```

#### Neo4j Alias Switch Failed

```bash
# Check alias status
cypher-shell -d system "SHOW ALIASES"

# Check database status
cypher-shell -d system "SHOW DATABASES"

# Manual alias fix
cypher-shell -d system "ALTER ALIAS kb7_production SET DATABASE = kb7_v2"
```

### 9.2 Recovery Procedures

#### Full CDC Pipeline Reset

```bash
# 1. Stop all consumers
kubectl scale deployment terminology-notification-service --replicas=0
kubectl scale deployment neo4j-sync-service --replicas=0

# 2. Reset Kafka consumer offsets
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group terminology-notification-service \
  --reset-offsets --to-earliest --topic kb7.terminology.releases --execute

# 3. Restart consumers
kubectl scale deployment terminology-notification-service --replicas=1
kubectl scale deployment neo4j-sync-service --replicas=1
```

#### Emergency Rollback

```bash
# 1. Identify previous working version
redis-cli KEYS "neo4j:import:*"

# 2. Execute rollback
python -c "
from services.neo4j_sync_service import Neo4jTerminologySyncService
import asyncio
svc = Neo4jTerminologySyncService(config)
asyncio.run(svc.rollback('2024.11.01'))
"

# 3. Verify
cypher-shell -d kb7_production "MATCH (d:Drug) RETURN count(d)"
```

---

## Appendix A: File Locations

```
backend/
├── shared-infrastructure/
│   ├── kafka/
│   │   └── cdc-connectors/
│   │       ├── configs/
│   │       │   └── kb7-terminology-releases-cdc.json  # Debezium connector config
│   │       ├── sql/
│   │       │   └── kb7-releases-schema.sql           # PostgreSQL outbox table
│   │       ├── scripts/
│   │       │   ├── setup-postgresql-cdc.sh
│   │       │   └── deploy-all-cdc-connectors.sh
│   │       └── services/
│   │           ├── terminology_notification_service.py
│   │           ├── requirements.txt
│   │           └── run_notification_service.sh
│   │
│   ├── flink-processing/
│   │   └── src/main/java/com/cardiofit/flink/
│   │       ├── cdc/
│   │       │   ├── TerminologyReleaseCDCEvent.java   # CDC event model
│   │       │   └── DebeziumJSONDeserializer.java     # Debezium JSON parsing
│   │       ├── operators/
│   │       │   └── Module_KB7_TerminologyBroadcast.java
│   │       └── models/
│   │           └── EnrichedPatientContext.java
│   │
│   └── knowledge-base-services/
│       └── kb-7-terminology/
│           └── runtime-layer-MOVED-TO-SHARED/
│               ├── neo4j-setup/
│               │   └── dual_stream_manager.py        # Neo4j dual-stream (Patient + Semantic)
│               ├── adapters/
│               │   ├── n10s_rdf_importer.py          # n10s RDF→Neo4j (CURRENT)
│               │   └── graphdb_neo4j_adapter.py      # Deprecated shim
│               ├── cache-warming/
│               │   └── cdc_subscriber.py             # CDC-driven cache warming
│               └── services/
│                   └── neo4j_sync_service.py         # 5-phase database aliasing
│
└── stream-services/
    └── module8-neo4j-graph-projector/
        └── app/services/
            └── projector.py
```

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **BroadcastStream** | Flink pattern that replicates data to all parallel tasks |
| **CDC** | Change Data Capture - tracking database changes |
| **Commit-Last** | Strategy where CDC event is published after data is fully loaded |
| **Database Alias** | Neo4j pointer that redirects queries to a physical database |
| **Debezium** | Open-source CDC platform for Kafka |
| **Hot-Swap** | Updating configuration without restart |
| **Keyed Stream** | Flink stream partitioned by a key |
| **Outbox Pattern** | Using a database table as an event source |

---

*Document Version: 1.1*
*Last Updated: 2025-12-04*
*Author: CDC Integration Team*

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2024-12-03 | Initial document creation |
| 1.1 | 2025-12-04 | Gap analysis fixes: corrected file paths, added complete SQL schema, added Flink side outputs documentation, updated file locations appendix |
