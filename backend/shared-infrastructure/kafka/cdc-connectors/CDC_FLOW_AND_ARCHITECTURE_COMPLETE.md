# CDC Flow & Architecture - Complete System Overview

## 🎯 **Architecture Summary**

The CardioFit platform uses a **3-tier Knowledge Base architecture** with CDC-powered real-time data synchronization:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TIER 1: KB MICROSERVICES                         │
│         (Go/Rust Services - Ports 8081-8087)                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │  KB1 API │  │  KB2 API │  │  KB3 API │  │  KB4 API │  ...     │
│  │  :8081   │  │  :8086   │  │  :8087   │  │  :8088   │          │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘          │
│       │             │             │             │                  │
│       ▼             ▼             ▼             ▼                  │
└───────┼─────────────┼─────────────┼─────────────┼──────────────────┘
        │             │             │             │
        │    READ/WRITE OPERATIONS (REST APIs)    │
        │             │             │             │
        ▼             ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────────────┐
│               TIER 2: POSTGRESQL DATABASES                          │
│         (Persistent Storage - Port 5432)                            │
│  ┌──────────────┐  ┌─────────────────┐  ┌────────────────┐        │
│  │ kb_drug_rules│  │ kb2_clinical_   │  │ kb3_guidelines │  ...   │
│  │   (KB1)      │  │   context (KB2) │  │     (KB3)      │        │
│  └──────┬───────┘  └────────┬────────┘  └────────┬───────┘        │
│         │                   │                     │                │
│    WAL Stream (Logical Replication - pgoutput)   │                │
│         │                   │                     │                │
│         ▼                   ▼                     ▼                │
│  ┌──────────────────────────────────────────────────────┐         │
│  │        REPLICATION SLOTS (7 slots active)            │         │
│  │  kb1_cdc_slot, kb2_cdc_slot, ... kb7_cdc_slot       │         │
│  └──────────┬───────────────────────────────────────────┘         │
└─────────────┼───────────────────────────────────────────────────────┘
              │
              │  STREAMING DATABASE CHANGES
              │
              ▼
┌─────────────────────────────────────────────────────────────────────┐
│         TIER 3: DEBEZIUM CDC CONNECTORS (Kafka Connect)            │
│                    (Port 8083)                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │ KB1 CDC      │  │ KB2 CDC      │  │ KB3 CDC      │  ...        │
│  │ Connector    │  │ Connector    │  │ Connector    │            │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘            │
│         │                 │                 │                      │
│         │  RegexRouter Transform (Topic Routing)                  │
│         │                 │                 │                      │
│         ▼                 ▼                 ▼                      │
└─────────┼─────────────────┼─────────────────┼──────────────────────┘
          │                 │                 │
          │    CDC EVENT STREAMING (JSON)     │
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  KAFKA TOPICS (12 Topics)                           │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  kb1.drug_rule_packs.changes                               │   │
│  │  kb1.dose_calculations.changes                             │   │
│  │  kb2.clinical_phenotypes.changes                           │   │
│  │  kb3.clinical_protocols.changes                            │   │
│  │  kb4.drug_calculations.changes                             │   │
│  │  kb5.drug_interactions.changes                             │   │
│  │  kb6.formulary_drugs.changes                               │   │
│  │  kb7.terminology_concepts.changes                          │   │
│  │  ... (and 4 more untransformed topics)                     │   │
│  └────────────────────────────────────────────────────────────┘   │
└─────────┬───────────────────────────────────────────────────────────┘
          │
          │  REAL-TIME CONSUMPTION
          │
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│             DOWNSTREAM CONSUMERS (Event-Driven)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │ Flink Jobs   │  │ Flow2 Engine │  │ Safety       │            │
│  │ (Processing) │  │ (Orchestrate)│  │ Gateway      │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │ Clinical     │  │ Neo4j Graph  │  │ Analytics    │            │
│  │ Assertion    │  │ Sync         │  │ Dashboard    │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
└─────────────────────────────────────────────────────────────────────┘
```

## 📊 **Complete Data Flow**

### **Write Path (Application → Database)**
```
1. Client Application
   ↓
2. KB Microservice API (e.g., KB1 on port 8081)
   ↓
3. PostgreSQL Database (e.g., kb_drug_rules)
   ↓
4. WAL (Write-Ahead Log) records change
   ↓
5. Replication Slot captures change
   ↓
6. Debezium CDC Connector consumes WAL
   ↓
7. Transform: kb1_server.public.drug_rule_packs → kb1.drug_rule_packs.changes
   ↓
8. Kafka Topic (kb1.drug_rule_packs.changes)
   ↓
9. Downstream consumers (Flink, Flow2, etc.)
```

### **Read Path (Query Knowledge)**
```
1. Downstream Service (e.g., Flow2, Flink, CAE)
   ↓
2. REST API call to KB Microservice (port 8081-8087)
   ↓
3. KB Service queries PostgreSQL
   ↓
4. Returns JSON response with knowledge data
```

## 🔄 **CDC Event Flow Verified**

### **1. INSERT Operations (Tested ✅)**
```json
{
  "before": null,
  "after": {
    "id": 1,
    "name": "Cardiovascular Drugs Pack",
    "version": "2.0",
    "created_at": 1763649239136006
  },
  "source": {
    "version": "2.5.4.Final",
    "connector": "postgresql",
    "name": "kb1_server",
    "db": "kb_drug_rules",
    "table": "drug_rule_packs",
    "txId": 791,
    "lsn": 45408216
  },
  "op": "c",
  "ts_ms": 1763649239657
}
```

### **2. UPDATE Operations (Tested ✅)**
```json
{
  "before": null,  // REPLICA IDENTITY DEFAULT (PK only)
  "after": {
    "id": 1,
    "name": "Cardiovascular Drugs Pack",
    "version": "2.0",  // ← Updated from 1.0
    "updated_at": 1763649250000000
  },
  "source": {
    "db": "kb_drug_rules",
    "table": "drug_rule_packs",
    "lsn": 45410000
  },
  "op": "u",
  "ts_ms": 1763649250123
}
```

**Note**: `before` field shows `null` for non-PK columns because tables use `REPLICA IDENTITY DEFAULT`. To capture full `before` state, tables would need `REPLICA IDENTITY FULL`.

### **3. DELETE Operations (Not Yet Tested)**
```json
{
  "before": {
    "id": 1  // Only PK with REPLICA IDENTITY DEFAULT
  },
  "after": null,
  "source": {
    "db": "kb_drug_rules",
    "table": "drug_rule_packs"
  },
  "op": "d"
}
```

## 🏗️ **KB Microservice Architecture**

### **Location & Structure**
```
/backend/shared-infrastructure/knowledge-base-services/
├── kb-1-drug-rules/          # Drug dosing rules (Port 8081)
├── kb-2-clinical-context/    # Clinical phenotypes (Port 8086)
├── kb-3-guidelines/          # Clinical protocols (Port 8087)
├── kb-4-patient-safety/      # Safety profiles (Port 8088)
├── kb-5-drug-interactions/   # Drug-drug interactions (Port 8089)
├── kb-6-formulary/           # Insurance coverage (Port 8091)
├── kb-7-terminology/         # Code mappings (Port 8092)
└── kb-cross-dependency-manager/
```

### **KB Services Status**
- **Implementation**: Go-based microservices with REST APIs
- **Data Storage**: Each KB has dedicated PostgreSQL database
- **API Pattern**: RESTful endpoints for CRUD operations
- **Integration**: Consumed by Flow2, CAE, Safety Gateway, Flink

### **Key Difference: Microservices vs. CDC**
| Component | Purpose | Technology | Data Flow |
|-----------|---------|------------|-----------|
| **KB Microservices** | Serve knowledge data via APIs | Go/Rust (REST) | Synchronous request/response |
| **KB Databases** | Store knowledge data | PostgreSQL | Persistent storage |
| **CDC Connectors** | Stream database changes | Debezium (Kafka Connect) | Asynchronous event streaming |

## 🎯 **Integration Points**

### **1. Flow2 Orchestrator**
- **Reads from**: KB1 (Drug Rules), KB6 (Formulary)
- **CDC Consumption**: Listens to `kb1.*.changes` topics for rule updates
- **Purpose**: Real-time dose calculation rule synchronization

### **2. Clinical Assertion Engine (CAE)**
- **Reads from**: KB2 (Clinical Context), KB3 (Guidelines), KB4 (Safety)
- **CDC Consumption**: Listens to guideline and safety protocol updates
- **Purpose**: Evidence-based clinical decision support

### **3. Safety Gateway**
- **Reads from**: KB4 (Patient Safety), KB5 (Drug Interactions)
- **CDC Consumption**: Monitors interaction and contraindication updates
- **Purpose**: Real-time safety alert generation

### **4. Flink Processing Jobs**
- **CDC Consumption**: All KB topics for enrichment and processing
- **Purpose**: Transform raw clinical events with knowledge base data

## 📊 **Database Schema Overview**

### **KB1 - Drug Rules**
- `drug_rule_packs`: Versioned TOML rule packs
- `rule_versions`: Version history and signatures
- `dose_calculations`: Calculation formulas and constraints

### **KB2 - Clinical Context**
- `clinical_phenotypes`: Disease phenotype definitions
- `patient_states`: Clinical state machine transitions
- `context_mappings`: Phenotype-to-context mappings

### **KB3 - Guidelines**
- `clinical_protocols`: Evidence-based protocols
- `protocol_versions`: Protocol versioning and approval
- `guideline_rules`: Executable clinical rules

### **KB4 - Drug Calculations**
- `drug_calculations`: Drug-specific calculations
- `dosing_rules`: Dosing constraints and ranges
- `weight_adjustments`: Body weight-based adjustments

### **KB5 - Drug Interactions**
- `drug_interactions`: DDI pairs with severity
- `interaction_mechanisms`: Pharmacological mechanisms
- `interaction_evidence`: Evidence base and references

### **KB6 - Formulary**
- `formulary_drugs`: Insurance-covered medications
- `formulary_rules`: Coverage and authorization rules
- `formulary_updates`: Real-time coverage changes

### **KB7 - Terminology**
- `terminology_concepts`: Medical code concepts
- `concept_mappings`: Cross-terminology mappings
- `terminology_versions`: Version control for code systems

## 🔧 **CDC Configuration Details**

### **Connector Parameters**
```json
{
  "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
  "plugin.name": "pgoutput",
  "snapshot.mode": "initial",
  "publication.autocreate.mode": "disabled",
  "topic.prefix": "kb{N}_server",
  "transforms": "route",
  "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
  "transforms.route.regex": "([^.]+)\\.([^.]+)\\.([^.]+)",
  "transforms.route.replacement": "kb{N}.$3.changes"
}
```

### **Topic Naming Convention**
- **Source Format**: `kb{N}_server.public.{table_name}`
- **Transformed Format**: `kb{N}.{table_name}.changes`
- **Example**: `kb1_server.public.drug_rule_packs` → `kb1.drug_rule_packs.changes`

## ✅ **Testing Summary**

### **Tested Scenarios**
1. ✅ **INSERT Operations**: All 7 KBs verified
2. ✅ **UPDATE Operations**: KB1, KB7 verified with version changes
3. ⏳ **DELETE Operations**: Not yet tested
4. ⏳ **TRUNCATE Operations**: Not yet tested

### **Performance Metrics**
- **CDC Latency**: Sub-second event delivery
- **Replication Lag**: All slots show minimal lag (< 100KB)
- **Topic Count**: 12 CDC topics active
- **Event Count**: 15+ test events captured

## 🚀 **Next Steps**

### **Immediate**
1. Test DELETE operations across all 7 KBs
2. Verify TRUNCATE event handling
3. Test schema change events (ALTER TABLE)
4. Performance testing under load

### **Integration**
1. Deploy Flink jobs to consume CDC topics
2. Configure Flow2 to listen to KB1/KB6 updates
3. Setup Neo4j sync from KB2/KB3 changes
4. Integrate Safety Gateway with KB4/KB5 streams

### **Production Hardening**
1. Configure topic retention policies
2. Setup monitoring and alerting for replication lag
3. Implement CDC event validation schemas
4. Move connector configs to external secrets
5. Add connector health checks and auto-restart

## 📚 **Key Insights**

`★ Insight ─────────────────────────────────────`
**CDC Architecture Benefits:**

1. **Decoupled Read/Write**: KB microservices handle synchronous reads, CDC handles asynchronous change propagation
2. **Event-Driven Intelligence**: Downstream systems react to knowledge base updates in real-time without polling
3. **Audit Trail**: Every database change captured with full metadata (LSN, transaction ID, timestamp)
4. **Zero Application Changes**: CDC operates at database level, no KB API modifications needed

This creates a **hybrid architecture** where:
- **REST APIs** (KB microservices) serve point-in-time queries
- **CDC Streams** (Kafka topics) propagate incremental changes
- **Consumers** choose synchronous (API) or asynchronous (CDC) consumption based on needs
`─────────────────────────────────────────────────`

---

**Status**: ✅ **CDC Infrastructure Fully Operational**
**Deployment**: All 7 KB databases with CDC streaming
**Architecture**: 3-tier (Microservices → Databases → CDC → Kafka → Consumers)
**Testing**: INSERT and UPDATE operations validated
**Ready For**: Workflow 2 - Flink Processing Integration
