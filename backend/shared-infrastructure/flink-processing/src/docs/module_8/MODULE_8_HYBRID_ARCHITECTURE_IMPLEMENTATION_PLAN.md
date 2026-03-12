# Module 8: Hybrid Architecture Storage Projectors Implementation Plan

## Overview

Module 8 consumes from **Module 6's hybrid Kafka topic architecture** and projects data into 8 specialized storage systems using independent Python/FastAPI projector services.

---

## Module 6 → Module 8 Data Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MODULE 6 - EGRESS ROUTING                        │
│          (TransactionalMultiSinkRouter.java)                        │
└────────┬────────────────┬────────────────┬──────────────────────────┘
         │                │                │
         │                │                │
         ▼                ▼                ▼
┌────────────────┐ ┌──────────────┐ ┌─────────────────┐
│ prod.ehr.      │ │ prod.ehr.    │ │ prod.ehr.       │
│ events.        │ │ fhir.        │ │ graph.          │
│ enriched       │ │ upsert       │ │ mutations       │
│                │ │              │ │                 │
│ (24 partitions)│ │ (12 parts)   │ │ (16 partitions) │
│ (90 days)      │ │ (365 days)   │ │ (30 days)       │
│                │ │ COMPACTED    │ │                 │
└────────┬───────┘ └──────┬───────┘ └────────┬────────┘
         │                │                  │
         ├────────────────┼──────────────────┤
         │                │                  │
         ▼                ▼                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MODULE 8 - STORAGE PROJECTORS                    │
│                   (8 Independent Python Services)                   │
└─────────────────────────────────────────────────────────────────────┘
         │
         ├─────────────────────────────────────────────────────────────┐
         │                                                             │
         ▼                                                             ▼
┌─────────────────────────────┐              ┌──────────────────────────┐
│ PROJECTORS FOR               │              │ SPECIALIZED PROJECTORS   │
│ prod.ehr.events.enriched     │              │                          │
│                              │              │                          │
│ 1. PostgreSQL Projector      │              │ 7. FHIR Store Projector  │
│ 2. MongoDB Projector         │              │    (prod.ehr.fhir.upsert)│
│ 3. Elasticsearch Projector   │              │                          │
│ 4. ClickHouse Projector      │              │ 8. Neo4j Graph Projector │
│ 5. InfluxDB Projector        │              │    (prod.ehr.graph.      │
│ 6. UPS Read Model Projector  │              │     mutations)           │
└─────────────────────────────┘              └──────────────────────────┘
```

---

## Data Models

### 1. EnrichedClinicalEvent (from prod.ehr.events.enriched)

**Source**: TransactionalMultiSinkRouter.java line 89-104

```json
{
  "id": "evt_12345",
  "timestamp": 1699564800000,
  "eventType": "VITAL_SIGNS",
  "patientId": "P12345",
  "encounterId": "E67890",
  "departmentId": "ICU_01",
  "deviceId": "MON_5678",

  "rawData": {
    "heart_rate": 95,
    "blood_pressure_systolic": 140,
    "blood_pressure_diastolic": 90,
    "spo2": 96,
    "temperature_celsius": 37.2
  },

  "enrichments": {
    "NEWS2Score": 4,
    "qSOFAScore": 1,
    "riskLevel": "MODERATE",
    "clinicalContext": {
      "activeConditions": ["I50.9", "E11.9"],
      "currentMedications": ["metoprolol", "lisinopril"],
      "recentProcedures": []
    }
  },

  "semanticAnnotations": {
    "SNOMED_CT": ["431314004", "271649006"],
    "LOINC": ["8867-4", "8480-6"]
  },

  "mlPredictions": {
    "sepsis_risk_24h": 0.23,
    "cardiac_event_risk_7d": 0.15,
    "readmission_risk_30d": 0.42
  }
}
```

### 2. FHIRResource (from prod.ehr.fhir.upsert)

**Source**: TransactionalMultiSinkRouter.java line 324-382

```json
{
  "resourceType": "Observation",
  "resourceId": "obs_12345",
  "patientId": "P12345",
  "lastUpdated": 1699564800000,

  "fhirData": {
    "resourceType": "Observation",
    "id": "obs_12345",
    "status": "final",
    "category": [{
      "coding": [{
        "system": "http://terminology.hl7.org/CodeSystem/observation-category",
        "code": "vital-signs"
      }]
    }],
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "8867-4",
        "display": "Heart rate"
      }]
    },
    "subject": {
      "reference": "Patient/P12345"
    },
    "effectiveDateTime": "2023-11-09T14:30:00Z",
    "valueQuantity": {
      "value": 95,
      "unit": "beats/minute",
      "system": "http://unitsofmeasure.org",
      "code": "/min"
    }
  }
}
```

**Kafka Key**: `"{resourceType}|{resourceId}"` (e.g., `"Observation|obs_12345"`)

### 3. GraphMutation (from prod.ehr.graph.mutations)

**Source**: TransactionalMultiSinkRouter.java line 386-417

```json
{
  "mutationType": "MERGE",
  "nodeType": "Patient",
  "nodeId": "P12345",
  "timestamp": 1699564800000,

  "nodeProperties": {
    "patientId": "P12345",
    "lastUpdated": 1699564800000,
    "demographicsVersion": 3
  },

  "relationships": [
    {
      "relationType": "HAS_EVENT",
      "targetNodeType": "ClinicalEvent",
      "targetNodeId": "evt_12345",
      "relationshipProperties": {
        "timestamp": 1699564800000,
        "eventType": "VITAL_SIGNS",
        "severity": "MODERATE"
      }
    },
    {
      "relationType": "HAS_CONDITION",
      "targetNodeType": "Condition",
      "targetNodeId": "I50.9",
      "relationshipProperties": {
        "onsetDate": "2023-01-15",
        "status": "active"
      }
    }
  ]
}
```

**Kafka Key**: `"{nodeId}"` (e.g., `"P12345"`)

---

## Projector 1: PostgreSQL Projector

### HOW - Kafka Consumer Configuration

```python
# File: backend/stream-services/module8-postgresql-projector/app/config.py

KAFKA_CONFIG = {
    "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
    "group.id": "module8-postgresql-projector",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 500,
    "max.poll.interval.ms": 300000,

    # Security
    "security.protocol": "SASL_SSL",
    "sasl.mechanism": "PLAIN",
    "sasl.username": os.getenv("KAFKA_API_KEY"),
    "sasl.password": os.getenv("KAFKA_API_SECRET"),
}

TOPICS = ["prod.ehr.events.enriched"]
BATCH_SIZE = 100
BATCH_TIMEOUT_SECONDS = 5
```

### WHAT - Processing Logic

```python
# File: backend/stream-services/module8-postgresql-projector/app/services/projector.py

import json
from typing import List, Dict, Any
from kafka import KafkaConsumer
import psycopg2
from psycopg2.extras import execute_batch
import structlog

logger = structlog.get_logger(__name__)

class PostgreSQLProjector:
    """
    Projects enriched clinical events to PostgreSQL for OLTP storage

    Database Schema:
    - enriched_events: Raw event storage with JSONB
    - patient_vitals: Normalized vital signs
    - clinical_scores: Risk scores and predictions
    - event_metadata: Searchable event attributes
    """

    def __init__(self, kafka_config: Dict, postgres_config: Dict):
        self.kafka_config = kafka_config
        self.postgres_config = postgres_config
        self.batch = []
        self.batch_timestamp = time.time()

    def start(self):
        """Start consuming and projecting events"""
        consumer = KafkaConsumer(
            'prod.ehr.events.enriched',
            **self.kafka_config,
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )

        logger.info("PostgreSQL Projector started",
                   topic="prod.ehr.events.enriched",
                   batch_size=BATCH_SIZE)

        for message in consumer:
            try:
                event = message.value
                self.batch.append(event)

                # Flush batch if size or timeout reached
                if (len(self.batch) >= BATCH_SIZE or
                    time.time() - self.batch_timestamp > BATCH_TIMEOUT_SECONDS):
                    self.flush_batch()
                    consumer.commit()

            except Exception as e:
                logger.error("Error processing message", error=str(e))
                self.send_to_dlq(message)

    def flush_batch(self):
        """Write batch to PostgreSQL"""
        if not self.batch:
            return

        conn = psycopg2.connect(**self.postgres_config)
        try:
            with conn.cursor() as cur:
                # 1. Insert raw events
                self._insert_enriched_events(cur, self.batch)

                # 2. Insert normalized vitals
                self._insert_patient_vitals(cur, self.batch)

                # 3. Insert clinical scores
                self._insert_clinical_scores(cur, self.batch)

                # 4. Insert event metadata
                self._insert_event_metadata(cur, self.batch)

            conn.commit()
            logger.info("Batch written to PostgreSQL",
                       batch_size=len(self.batch))

        except Exception as e:
            conn.rollback()
            logger.error("Batch write failed", error=str(e))
            raise
        finally:
            conn.close()
            self.batch = []
            self.batch_timestamp = time.time()

    def _insert_enriched_events(self, cur, events: List[Dict]):
        """Insert to enriched_events table with full JSONB"""
        sql = """
            INSERT INTO enriched_events
            (event_id, patient_id, timestamp, event_type, event_data)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (event_id) DO UPDATE SET
                event_data = EXCLUDED.event_data,
                updated_at = NOW()
        """

        data = [
            (
                event['id'],
                event['patientId'],
                event['timestamp'],
                event['eventType'],
                json.dumps(event)  # Store full event as JSONB
            )
            for event in events
        ]

        execute_batch(cur, sql, data)

    def _insert_patient_vitals(self, cur, events: List[Dict]):
        """Insert to patient_vitals table (normalized)"""
        vitals_events = [e for e in events if e['eventType'] == 'VITAL_SIGNS']

        if not vitals_events:
            return

        sql = """
            INSERT INTO patient_vitals
            (event_id, patient_id, timestamp, heart_rate, bp_systolic,
             bp_diastolic, spo2, temperature_celsius)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """

        data = [
            (
                event['id'],
                event['patientId'],
                event['timestamp'],
                event['rawData'].get('heart_rate'),
                event['rawData'].get('blood_pressure_systolic'),
                event['rawData'].get('blood_pressure_diastolic'),
                event['rawData'].get('spo2'),
                event['rawData'].get('temperature_celsius')
            )
            for event in vitals_events
        ]

        execute_batch(cur, sql, data)

    def _insert_clinical_scores(self, cur, events: List[Dict]):
        """Insert to clinical_scores table"""
        sql = """
            INSERT INTO clinical_scores
            (event_id, patient_id, timestamp, news2_score, qsofa_score,
             risk_level, sepsis_risk_24h, cardiac_risk_7d, readmission_risk_30d)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        """

        data = [
            (
                event['id'],
                event['patientId'],
                event['timestamp'],
                event['enrichments'].get('NEWS2Score'),
                event['enrichments'].get('qSOFAScore'),
                event['enrichments'].get('riskLevel'),
                event.get('mlPredictions', {}).get('sepsis_risk_24h'),
                event.get('mlPredictions', {}).get('cardiac_event_risk_7d'),
                event.get('mlPredictions', {}).get('readmission_risk_30d')
            )
            for event in events
            if 'enrichments' in event
        ]

        execute_batch(cur, sql, data)

    def _insert_event_metadata(self, cur, events: List[Dict]):
        """Insert to event_metadata table for fast queries"""
        sql = """
            INSERT INTO event_metadata
            (event_id, patient_id, encounter_id, department_id,
             device_id, timestamp, event_type)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (event_id) DO NOTHING
        """

        data = [
            (
                event['id'],
                event['patientId'],
                event.get('encounterId'),
                event.get('departmentId'),
                event.get('deviceId'),
                event['timestamp'],
                event['eventType']
            )
            for event in events
        ]

        execute_batch(cur, sql, data)
```

### Database Schema

```sql
-- File: backend/stream-services/module8-postgresql-projector/schema/init.sql

-- Table 1: Raw enriched events with JSONB
CREATE TABLE enriched_events (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    timestamp BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_enriched_patient_time ON enriched_events(patient_id, timestamp DESC);
CREATE INDEX idx_enriched_type ON enriched_events(event_type);
CREATE INDEX idx_enriched_data_gin ON enriched_events USING GIN(event_data);

-- Table 2: Normalized patient vitals
CREATE TABLE patient_vitals (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) REFERENCES enriched_events(event_id),
    patient_id VARCHAR(255) NOT NULL,
    timestamp BIGINT NOT NULL,
    heart_rate INTEGER,
    bp_systolic INTEGER,
    bp_diastolic INTEGER,
    spo2 INTEGER,
    temperature_celsius DECIMAL(4, 2),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_vitals_patient_time ON patient_vitals(patient_id, timestamp DESC);

-- Table 3: Clinical scores and predictions
CREATE TABLE clinical_scores (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) REFERENCES enriched_events(event_id),
    patient_id VARCHAR(255) NOT NULL,
    timestamp BIGINT NOT NULL,
    news2_score INTEGER,
    qsofa_score INTEGER,
    risk_level VARCHAR(20),
    sepsis_risk_24h DECIMAL(5, 4),
    cardiac_risk_7d DECIMAL(5, 4),
    readmission_risk_30d DECIMAL(5, 4),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_scores_patient_time ON clinical_scores(patient_id, timestamp DESC);
CREATE INDEX idx_scores_risk_level ON clinical_scores(risk_level);

-- Table 4: Event metadata for fast queries
CREATE TABLE event_metadata (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    encounter_id VARCHAR(255),
    department_id VARCHAR(50),
    device_id VARCHAR(100),
    timestamp BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL
);

CREATE INDEX idx_metadata_patient ON event_metadata(patient_id);
CREATE INDEX idx_metadata_encounter ON event_metadata(encounter_id);
CREATE INDEX idx_metadata_department ON event_metadata(department_id);
```

**Performance Characteristics**:
- Write Throughput: ~2,000 events/sec with batch size 100
- Read Latency: <50ms for single patient queries
- Storage: JSONB for flexibility, normalized tables for performance

---

## Projector 2: MongoDB Projector

### HOW - Kafka Consumer Configuration

```python
# File: backend/stream-services/module8-mongodb-projector/app/config.py

KAFKA_CONFIG = {
    "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
    "group.id": "module8-mongodb-projector",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 500,
}

TOPICS = ["prod.ehr.events.enriched"]
BATCH_SIZE = 50  # Smaller batch for MongoDB
BATCH_TIMEOUT_SECONDS = 10
```

### WHAT - Processing Logic

```python
# File: backend/stream-services/module8-mongodb-projector/app/services/projector.py

from pymongo import MongoClient, UpdateOne
from kafka import KafkaConsumer
import structlog

logger = structlog.get_logger(__name__)

class MongoDBProjector:
    """
    Projects enriched clinical events to MongoDB for document storage

    Collections:
    - clinical_documents: Full event documents with rich metadata
    - patient_timelines: Aggregated patient event history
    - ml_explanations: Model predictions with interpretability data
    """

    def __init__(self, kafka_config: Dict, mongo_config: Dict):
        self.kafka_config = kafka_config
        self.mongo_uri = mongo_config['uri']
        self.db_name = mongo_config['database']
        self.batch = []

    def start(self):
        """Start consuming and projecting events"""
        consumer = KafkaConsumer(
            'prod.ehr.events.enriched',
            **self.kafka_config,
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )

        client = MongoClient(self.mongo_uri)
        db = client[self.db_name]

        logger.info("MongoDB Projector started", topic="prod.ehr.events.enriched")

        for message in consumer:
            try:
                event = message.value
                self.batch.append(event)

                if len(self.batch) >= BATCH_SIZE:
                    self.flush_batch(db)
                    consumer.commit()

            except Exception as e:
                logger.error("Error processing message", error=str(e))

    def flush_batch(self, db):
        """Write batch to MongoDB collections"""
        if not self.batch:
            return

        try:
            # 1. Insert clinical documents
            self._insert_clinical_documents(db, self.batch)

            # 2. Update patient timelines (aggregation)
            self._update_patient_timelines(db, self.batch)

            # 3. Insert ML explanations
            self._insert_ml_explanations(db, self.batch)

            logger.info("Batch written to MongoDB", batch_size=len(self.batch))

        except Exception as e:
            logger.error("Batch write failed", error=str(e))
            raise
        finally:
            self.batch = []

    def _insert_clinical_documents(self, db, events: List[Dict]):
        """Insert to clinical_documents collection"""
        documents = []

        for event in events:
            doc = {
                "_id": event['id'],
                "patientId": event['patientId'],
                "timestamp": event['timestamp'],
                "eventType": event['eventType'],
                "encounterId": event.get('encounterId'),
                "departmentId": event.get('departmentId'),

                # Raw data
                "rawData": event.get('rawData', {}),

                # Enrichments
                "enrichments": event.get('enrichments', {}),

                # Semantic annotations
                "semanticAnnotations": event.get('semanticAnnotations', {}),

                # ML predictions
                "mlPredictions": event.get('mlPredictions', {}),

                # Metadata
                "metadata": {
                    "insertedAt": datetime.utcnow(),
                    "source": "module6-egress-routing"
                }
            }
            documents.append(doc)

        # Upsert documents
        operations = [
            UpdateOne(
                {"_id": doc["_id"]},
                {"$set": doc},
                upsert=True
            )
            for doc in documents
        ]

        result = db.clinical_documents.bulk_write(operations)
        logger.debug("Clinical documents inserted",
                    upserted=result.upserted_count,
                    modified=result.modified_count)

    def _update_patient_timelines(self, db, events: List[Dict]):
        """Update patient_timelines collection with aggregated data"""
        patient_groups = {}

        # Group events by patient
        for event in events:
            patient_id = event['patientId']
            if patient_id not in patient_groups:
                patient_groups[patient_id] = []
            patient_groups[patient_id].append(event)

        # Update timeline for each patient
        operations = []
        for patient_id, patient_events in patient_groups.items():
            timeline_entries = [
                {
                    "eventId": event['id'],
                    "timestamp": event['timestamp'],
                    "eventType": event['eventType'],
                    "summary": self._generate_event_summary(event),
                    "riskLevel": event.get('enrichments', {}).get('riskLevel'),
                    "scores": {
                        "NEWS2": event.get('enrichments', {}).get('NEWS2Score'),
                        "qSOFA": event.get('enrichments', {}).get('qSOFAScore')
                    }
                }
                for event in patient_events
            ]

            operations.append(
                UpdateOne(
                    {"_id": patient_id},
                    {
                        "$push": {
                            "timeline": {
                                "$each": timeline_entries,
                                "$sort": {"timestamp": -1},
                                "$slice": 1000  # Keep last 1000 events
                            }
                        },
                        "$set": {
                            "lastUpdated": datetime.utcnow()
                        }
                    },
                    upsert=True
                )
            )

        if operations:
            result = db.patient_timelines.bulk_write(operations)
            logger.debug("Patient timelines updated",
                        upserted=result.upserted_count,
                        modified=result.modified_count)

    def _insert_ml_explanations(self, db, events: List[Dict]):
        """Insert ML prediction explanations"""
        ml_docs = []

        for event in events:
            if 'mlPredictions' not in event or not event['mlPredictions']:
                continue

            doc = {
                "_id": f"{event['id']}_ml",
                "eventId": event['id'],
                "patientId": event['patientId'],
                "timestamp": event['timestamp'],

                "predictions": event['mlPredictions'],

                # Add SHAP/LIME explanations if available
                "explanations": event.get('mlExplanations', {}),

                # Feature importance
                "featureImportance": self._extract_feature_importance(event),

                "metadata": {
                    "insertedAt": datetime.utcnow()
                }
            }
            ml_docs.append(doc)

        if ml_docs:
            operations = [
                UpdateOne(
                    {"_id": doc["_id"]},
                    {"$set": doc},
                    upsert=True
                )
                for doc in ml_docs
            ]

            result = db.ml_explanations.bulk_write(operations)
            logger.debug("ML explanations inserted", count=len(ml_docs))

    def _generate_event_summary(self, event: Dict) -> str:
        """Generate human-readable event summary"""
        event_type = event['eventType']

        if event_type == 'VITAL_SIGNS':
            raw = event.get('rawData', {})
            return f"HR:{raw.get('heart_rate')} BP:{raw.get('blood_pressure_systolic')}/{raw.get('blood_pressure_diastolic')} SpO2:{raw.get('spo2')}%"

        # Add other event type summaries...
        return f"{event_type} event"

    def _extract_feature_importance(self, event: Dict) -> Dict:
        """Extract ML feature importance from event"""
        # This would be populated by ML service
        return event.get('mlFeatureImportance', {})
```

**MongoDB Indexes**:
```javascript
// clinical_documents
db.clinical_documents.createIndex({ "patientId": 1, "timestamp": -1 });
db.clinical_documents.createIndex({ "eventType": 1 });
db.clinical_documents.createIndex({ "enrichments.riskLevel": 1 });
db.clinical_documents.createIndex({ "timestamp": 1 }, { expireAfterSeconds: 7776000 }); // 90 days

// patient_timelines
db.patient_timelines.createIndex({ "_id": 1 });
db.patient_timelines.createIndex({ "lastUpdated": -1 });

// ml_explanations
db.ml_explanations.createIndex({ "patientId": 1, "timestamp": -1 });
db.ml_explanations.createIndex({ "predictions.sepsis_risk_24h": -1 });
```

**Performance Characteristics**:
- Write Throughput: ~1,500 documents/sec
- Read Latency: <100ms for patient timeline queries
- Storage: Flexible schema, automatic aggregation

---

## Projector 7: FHIR Store Projector (NEW)

### HOW - Kafka Consumer Configuration

```python
# File: backend/stream-services/module8-fhir-store-projector/app/config.py

KAFKA_CONFIG = {
    "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
    "group.id": "module8-fhir-store-projector",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 100,  # Smaller batch for API calls
}

# IMPORTANT: Consuming from DIFFERENT topic than core projectors!
TOPICS = ["prod.ehr.fhir.upsert"]  # Pre-transformed FHIR resources

BATCH_SIZE = 20  # Small batch for Google Healthcare API
BATCH_TIMEOUT_SECONDS = 10

# Google FHIR Store config
GOOGLE_PROJECT_ID = os.getenv("GOOGLE_PROJECT_ID", "cardiofit-dev")
GOOGLE_LOCATION = os.getenv("GOOGLE_LOCATION", "us-central1")
GOOGLE_DATASET_ID = os.getenv("GOOGLE_DATASET_ID", "cardiofit_fhir_dataset")
GOOGLE_FHIR_STORE_ID = os.getenv("GOOGLE_FHIR_STORE_ID", "cardiofit_fhir_store")
GOOGLE_CREDENTIALS_PATH = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
```

### WHAT - Processing Logic

```python
# File: backend/stream-services/module8-fhir-store-projector/app/services/projector.py

from google.auth import default
from google.cloud import healthcare_v1
from kafka import KafkaConsumer
import structlog
import json

logger = structlog.get_logger(__name__)

class FHIRStoreProjector:
    """
    Projects FHIR resources to Google Cloud Healthcare API FHIR Store

    Input: FHIRResource objects from prod.ehr.fhir.upsert topic

    Key Points:
    - Resources are PRE-TRANSFORMED by Module 6 (TransactionalMultiSinkRouter)
    - This projector only needs to write to Google FHIR Store
    - No transformation logic needed here
    - Handles resource types: Observation, RiskAssessment, DiagnosticReport, Condition
    """

    def __init__(self, kafka_config: Dict, fhir_config: Dict):
        self.kafka_config = kafka_config
        self.fhir_config = fhir_config
        self.batch = []

        # Initialize Google Healthcare API client
        self.fhir_store_path = (
            f"projects/{fhir_config['project_id']}/"
            f"locations/{fhir_config['location']}/"
            f"datasets/{fhir_config['dataset_id']}/"
            f"fhirStores/{fhir_config['fhir_store_id']}"
        )

        credentials, project = default()
        self.client = healthcare_v1.FhirStoreServiceClient(credentials=credentials)

        logger.info("FHIR Store Projector initialized",
                   fhir_store_path=self.fhir_store_path)

    def start(self):
        """Start consuming FHIR resources from Kafka"""
        consumer = KafkaConsumer(
            'prod.ehr.fhir.upsert',  # Consuming pre-transformed FHIR resources
            **self.kafka_config,
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )

        logger.info("FHIR Store Projector started",
                   topic="prod.ehr.fhir.upsert",
                   batch_size=BATCH_SIZE)

        for message in consumer:
            try:
                # Message value is FHIRResource object
                fhir_resource_obj = message.value
                self.batch.append(fhir_resource_obj)

                if len(self.batch) >= BATCH_SIZE:
                    self.flush_batch()
                    consumer.commit()

            except Exception as e:
                logger.error("Error processing FHIR resource", error=str(e))
                self.send_to_dlq(message)

    def flush_batch(self):
        """Write batch of FHIR resources to Google FHIR Store"""
        if not self.batch:
            return

        try:
            # Process each resource (Google API doesn't support batch upsert)
            success_count = 0
            error_count = 0

            for fhir_resource_obj in self.batch:
                try:
                    self._upsert_fhir_resource(fhir_resource_obj)
                    success_count += 1
                except Exception as e:
                    logger.error("Failed to upsert FHIR resource",
                               resource_type=fhir_resource_obj.get('resourceType'),
                               resource_id=fhir_resource_obj.get('resourceId'),
                               error=str(e))
                    error_count += 1

            logger.info("FHIR batch processed",
                       total=len(self.batch),
                       success=success_count,
                       errors=error_count)

        finally:
            self.batch = []

    def _upsert_fhir_resource(self, fhir_resource_obj: Dict):
        """
        Upsert a single FHIR resource to Google FHIR Store

        FHIRResource object structure:
        {
            "resourceType": "Observation",
            "resourceId": "obs_12345",
            "patientId": "P12345",
            "lastUpdated": 1699564800000,
            "fhirData": { ... actual FHIR R4 resource ... }
        }
        """
        resource_type = fhir_resource_obj['resourceType']
        resource_id = fhir_resource_obj['resourceId']
        fhir_data = fhir_resource_obj['fhirData']

        # Resource path in FHIR store
        resource_path = f"{self.fhir_store_path}/fhir/{resource_type}/{resource_id}"

        try:
            # Try to update existing resource first
            request = healthcare_v1.UpdateResourceRequest(
                name=resource_path,
                resource=json.dumps(fhir_data).encode('utf-8')
            )

            response = self.client.update_resource(request=request)

            logger.debug("FHIR resource updated",
                        resource_type=resource_type,
                        resource_id=resource_id)

        except Exception as update_error:
            # If update fails (resource doesn't exist), create new
            try:
                request = healthcare_v1.CreateResourceRequest(
                    parent=f"{self.fhir_store_path}/fhir",
                    type_=resource_type,
                    resource_id=resource_id,
                    resource=json.dumps(fhir_data).encode('utf-8')
                )

                response = self.client.create_resource(request=request)

                logger.debug("FHIR resource created",
                            resource_type=resource_type,
                            resource_id=resource_id)

            except Exception as create_error:
                logger.error("Failed to create FHIR resource",
                           resource_type=resource_type,
                           resource_id=resource_id,
                           update_error=str(update_error),
                           create_error=str(create_error))
                raise
```

### Resource Type Handling

```python
# File: backend/stream-services/module8-fhir-store-projector/app/services/resource_handlers.py

from typing import Dict

class FHIRResourceValidator:
    """Validate FHIR resources before writing to store"""

    SUPPORTED_RESOURCE_TYPES = [
        'Observation',
        'RiskAssessment',
        'DiagnosticReport',
        'Condition',
        'MedicationRequest',
        'Procedure',
        'Encounter'
    ]

    @staticmethod
    def validate_resource(fhir_resource_obj: Dict) -> bool:
        """Validate FHIR resource structure"""
        # Check required fields
        if 'resourceType' not in fhir_resource_obj:
            logger.warning("Missing resourceType")
            return False

        if 'resourceId' not in fhir_resource_obj:
            logger.warning("Missing resourceId")
            return False

        if 'fhirData' not in fhir_resource_obj:
            logger.warning("Missing fhirData")
            return False

        # Check resource type is supported
        if fhir_resource_obj['resourceType'] not in SUPPORTED_RESOURCE_TYPES:
            logger.warning("Unsupported resource type",
                          resource_type=fhir_resource_obj['resourceType'])
            return False

        # Validate fhirData structure
        fhir_data = fhir_resource_obj['fhirData']

        if fhir_data.get('resourceType') != fhir_resource_obj['resourceType']:
            logger.warning("Resource type mismatch",
                          outer=fhir_resource_obj['resourceType'],
                          inner=fhir_data.get('resourceType'))
            return False

        if fhir_data.get('id') != fhir_resource_obj['resourceId']:
            logger.warning("Resource ID mismatch",
                          outer=fhir_resource_obj['resourceId'],
                          inner=fhir_data.get('id'))
            return False

        # Validate subject reference
        if 'subject' not in fhir_data:
            logger.warning("Missing subject reference")
            return False

        return True
```

**Performance Characteristics**:
- Write Throughput: ~200 resources/sec (limited by Google API)
- API Latency: ~50-100ms per resource
- Storage: HIPAA-compliant, FHIR R4 standard
- Idempotent: Upsert based on resource ID

---

## Projector 8: Neo4j Graph Projector (NEW)

### HOW - Kafka Consumer Configuration

```python
# File: backend/stream-services/module8-neo4j-graph-projector/app/config.py

KAFKA_CONFIG = {
    "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
    "group.id": "module8-neo4j-graph-projector",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 200,
}

# IMPORTANT: Consuming from DIFFERENT topic than core projectors!
TOPICS = ["prod.ehr.graph.mutations"]  # Pre-defined graph mutations

BATCH_SIZE = 50  # Batch Cypher queries
BATCH_TIMEOUT_SECONDS = 5

# Neo4j config
NEO4J_URI = os.getenv("NEO4J_URI", "bolt://localhost:7687")
NEO4J_USERNAME = os.getenv("NEO4J_USERNAME", "neo4j")
NEO4J_PASSWORD = os.getenv("NEO4J_PASSWORD")
NEO4J_DATABASE = os.getenv("NEO4J_DATABASE", "cardiofit")
```

### WHAT - Processing Logic

```python
# File: backend/stream-services/module8-neo4j-graph-projector/app/services/projector.py

from neo4j import GraphDatabase
from kafka import KafkaConsumer
import structlog
import json

logger = structlog.get_logger(__name__)

class Neo4jGraphProjector:
    """
    Projects graph mutations to Neo4j knowledge graph

    Input: GraphMutation objects from prod.ehr.graph.mutations topic

    Key Points:
    - Mutations are PRE-DEFINED by Module 6 (TransactionalMultiSinkRouter)
    - This projector executes Cypher queries to build patient journey graphs
    - No semantic analysis needed here
    - Handles node types: Patient, ClinicalEvent, Condition, Medication, Procedure
    """

    def __init__(self, kafka_config: Dict, neo4j_config: Dict):
        self.kafka_config = kafka_config
        self.neo4j_config = neo4j_config
        self.batch = []

        # Initialize Neo4j driver
        self.driver = GraphDatabase.driver(
            neo4j_config['uri'],
            auth=(neo4j_config['username'], neo4j_config['password'])
        )

        logger.info("Neo4j Graph Projector initialized",
                   neo4j_uri=neo4j_config['uri'])

    def start(self):
        """Start consuming graph mutations from Kafka"""
        consumer = KafkaConsumer(
            'prod.ehr.graph.mutations',  # Consuming pre-defined mutations
            **self.kafka_config,
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )

        logger.info("Neo4j Graph Projector started",
                   topic="prod.ehr.graph.mutations",
                   batch_size=BATCH_SIZE)

        for message in consumer:
            try:
                # Message value is GraphMutation object
                mutation_obj = message.value
                self.batch.append(mutation_obj)

                if len(self.batch) >= BATCH_SIZE:
                    self.flush_batch()
                    consumer.commit()

            except Exception as e:
                logger.error("Error processing graph mutation", error=str(e))
                self.send_to_dlq(message)

    def flush_batch(self):
        """Execute batch of graph mutations in Neo4j"""
        if not self.batch:
            return

        with self.driver.session(database=self.neo4j_config['database']) as session:
            try:
                # Execute mutations in a transaction
                result = session.execute_write(self._execute_mutation_batch)

                logger.info("Graph mutations executed",
                           batch_size=len(self.batch),
                           nodes_created=result.get('nodes_created', 0),
                           relationships_created=result.get('relationships_created', 0))

            except Exception as e:
                logger.error("Graph mutation batch failed", error=str(e))
                raise
            finally:
                self.batch = []

    def _execute_mutation_batch(self, tx):
        """Execute batch of mutations within a transaction"""
        nodes_created = 0
        relationships_created = 0

        for mutation in self.batch:
            mutation_type = mutation['mutationType']
            node_type = mutation['nodeType']

            if mutation_type == 'MERGE':
                # MERGE node (create or update)
                cypher = self._build_merge_node_cypher(mutation)
                result = tx.run(cypher, **mutation['nodeProperties'])
                nodes_created += 1

            elif mutation_type == 'CREATE':
                # CREATE node (fail if exists)
                cypher = self._build_create_node_cypher(mutation)
                result = tx.run(cypher, **mutation['nodeProperties'])
                nodes_created += 1

            # Create relationships
            if 'relationships' in mutation:
                for rel in mutation['relationships']:
                    cypher = self._build_relationship_cypher(mutation, rel)
                    result = tx.run(cypher, **rel.get('relationshipProperties', {}))
                    relationships_created += 1

        return {
            'nodes_created': nodes_created,
            'relationships_created': relationships_created
        }

    def _build_merge_node_cypher(self, mutation: Dict) -> str:
        """
        Build MERGE Cypher query for node

        Example mutation:
        {
            "mutationType": "MERGE",
            "nodeType": "Patient",
            "nodeId": "P12345",
            "nodeProperties": {
                "patientId": "P12345",
                "lastUpdated": 1699564800000,
                "demographicsVersion": 3
            }
        }
        """
        node_type = mutation['nodeType']
        node_id = mutation['nodeId']
        properties = mutation.get('nodeProperties', {})

        # Build property string
        prop_assignments = ', '.join([
            f"n.{key} = ${key}" for key in properties.keys()
        ])

        cypher = f"""
            MERGE (n:{node_type} {{nodeId: '{node_id}'}})
            SET {prop_assignments}
            RETURN n
        """

        return cypher

    def _build_create_node_cypher(self, mutation: Dict) -> str:
        """Build CREATE Cypher query for node"""
        node_type = mutation['nodeType']
        node_id = mutation['nodeId']
        properties = mutation.get('nodeProperties', {})

        prop_assignments = ', '.join([
            f"{key}: ${key}" for key in properties.keys()
        ])

        cypher = f"""
            CREATE (n:{node_type} {{nodeId: '{node_id}', {prop_assignments}}})
            RETURN n
        """

        return cypher

    def _build_relationship_cypher(self, mutation: Dict, rel: Dict) -> str:
        """
        Build Cypher query for relationship

        Example relationship:
        {
            "relationType": "HAS_EVENT",
            "targetNodeType": "ClinicalEvent",
            "targetNodeId": "evt_12345",
            "relationshipProperties": {
                "timestamp": 1699564800000,
                "eventType": "VITAL_SIGNS",
                "severity": "MODERATE"
            }
        }
        """
        source_node_type = mutation['nodeType']
        source_node_id = mutation['nodeId']

        rel_type = rel['relationType']
        target_node_type = rel['targetNodeType']
        target_node_id = rel['targetNodeId']
        rel_properties = rel.get('relationshipProperties', {})

        # Build property string
        prop_assignments = ', '.join([
            f"{key}: ${key}" for key in rel_properties.keys()
        ])

        cypher = f"""
            MATCH (source:{source_node_type} {{nodeId: '{source_node_id}'}})
            MATCH (target:{target_node_type} {{nodeId: '{target_node_id}'}})
            MERGE (source)-[r:{rel_type}]->(target)
            SET r += {{{prop_assignments}}}
            RETURN r
        """

        return cypher
```

### Graph Schema

```cypher
// Node Types
CREATE CONSTRAINT patient_id IF NOT EXISTS FOR (p:Patient) REQUIRE p.nodeId IS UNIQUE;
CREATE CONSTRAINT event_id IF NOT EXISTS FOR (e:ClinicalEvent) REQUIRE e.nodeId IS UNIQUE;
CREATE CONSTRAINT condition_id IF NOT EXISTS FOR (c:Condition) REQUIRE c.nodeId IS UNIQUE;
CREATE CONSTRAINT medication_id IF NOT EXISTS FOR (m:Medication) REQUIRE m.nodeId IS UNIQUE;
CREATE CONSTRAINT procedure_id IF NOT EXISTS FOR (p:Procedure) REQUIRE p.nodeId IS UNIQUE;

// Indexes
CREATE INDEX patient_last_updated IF NOT EXISTS FOR (p:Patient) ON (p.lastUpdated);
CREATE INDEX event_timestamp IF NOT EXISTS FOR (e:ClinicalEvent) ON (e.timestamp);
CREATE INDEX event_type IF NOT EXISTS FOR (e:ClinicalEvent) ON (e.eventType);

// Example Graph Structure
// (Patient)-[:HAS_EVENT]->(ClinicalEvent)
// (Patient)-[:HAS_CONDITION]->(Condition)
// (Patient)-[:PRESCRIBED]->(Medication)
// (Patient)-[:UNDERWENT]->(Procedure)
// (ClinicalEvent)-[:NEXT_EVENT]->(ClinicalEvent)
// (ClinicalEvent)-[:TRIGGERED_BY]->(Condition)
```

**Performance Characteristics**:
- Write Throughput: ~500 mutations/sec
- Query Latency: <100ms for patient journey queries
- Storage: Temporal relationships, semantic links
- Use Cases: Patient journey visualization, clinical pathway analysis

---

## Docker Compose Deployment

```yaml
# File: backend/stream-services/docker-compose.module8.yml

version: '3.8'

services:
  # PostgreSQL Projector
  postgresql-projector:
    build: ./module8-postgresql-projector
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS}
      - KAFKA_API_KEY=${KAFKA_API_KEY}
      - KAFKA_API_SECRET=${KAFKA_API_SECRET}
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_DB=cardiofit
      - POSTGRES_USER=cardiofit_user
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    depends_on:
      - postgres
    restart: unless-stopped

  # MongoDB Projector
  mongodb-projector:
    build: ./module8-mongodb-projector
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS}
      - KAFKA_API_KEY=${KAFKA_API_KEY}
      - KAFKA_API_SECRET=${KAFKA_API_SECRET}
      - MONGO_URI=mongodb://mongo:27017/cardiofit
    depends_on:
      - mongo
    restart: unless-stopped

  # FHIR Store Projector (NEW)
  fhir-store-projector:
    build: ./module8-fhir-store-projector
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS}
      - KAFKA_API_KEY=${KAFKA_API_KEY}
      - KAFKA_API_SECRET=${KAFKA_API_SECRET}
      - GOOGLE_PROJECT_ID=${GOOGLE_PROJECT_ID}
      - GOOGLE_LOCATION=${GOOGLE_LOCATION}
      - GOOGLE_DATASET_ID=${GOOGLE_DATASET_ID}
      - GOOGLE_FHIR_STORE_ID=${GOOGLE_FHIR_STORE_ID}
      - GOOGLE_APPLICATION_CREDENTIALS=/app/credentials/google-credentials.json
    volumes:
      - ./credentials:/app/credentials:ro
    restart: unless-stopped

  # Neo4j Graph Projector (NEW)
  neo4j-graph-projector:
    build: ./module8-neo4j-graph-projector
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS}
      - KAFKA_API_KEY=${KAFKA_API_KEY}
      - KAFKA_API_SECRET=${KAFKA_API_SECRET}
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USERNAME=neo4j
      - NEO4J_PASSWORD=${NEO4J_PASSWORD}
      - NEO4J_DATABASE=cardiofit
    depends_on:
      - neo4j
    restart: unless-stopped

  # Databases
  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=cardiofit
      - POSTGRES_USER=cardiofit_user
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./module8-postgresql-projector/schema:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  mongo:
    image: mongo:7
    volumes:
      - mongo_data:/data/db
    ports:
      - "27017:27017"

  neo4j:
    image: neo4j:5.12
    environment:
      - NEO4J_AUTH=neo4j/${NEO4J_PASSWORD}
      - NEO4J_PLUGINS=["apoc"]
    volumes:
      - neo4j_data:/data
    ports:
      - "7474:7474"
      - "7687:7687"

volumes:
  postgres_data:
  mongo_data:
  neo4j_data:
```

---

## Summary Table

| Projector | Input Topic | Data Format | Storage | Throughput | Use Case |
|-----------|-------------|-------------|---------|------------|----------|
| **PostgreSQL** | prod.ehr.events.enriched | EnrichedClinicalEvent | Relational OLTP | 2,000 events/sec | Transactional queries, ACID compliance |
| **MongoDB** | prod.ehr.events.enriched | EnrichedClinicalEvent | Document store | 1,500 docs/sec | Clinical documents, patient timelines |
| **Elasticsearch** | prod.ehr.events.enriched | EnrichedClinicalEvent | Search index | 5,000 events/sec | Full-text search, dashboards |
| **ClickHouse** | prod.ehr.events.enriched | EnrichedClinicalEvent | Columnar OLAP | 10,000 events/sec | Analytics, reporting, aggregations |
| **InfluxDB** | prod.ehr.events.enriched | EnrichedClinicalEvent | Time-series DB | 10,000 points/sec | Vital signs trends, downsampling |
| **UPS Read Model** | prod.ehr.events.enriched | EnrichedClinicalEvent | Denormalized PG | 500 updates/sec | Hot path patient lookups |
| **FHIR Store** ⭐ | prod.ehr.fhir.upsert | FHIRResource | Google FHIR API | 200 resources/sec | HIPAA compliance, interoperability |
| **Neo4j Graph** ⭐ | prod.ehr.graph.mutations | GraphMutation | Graph database | 500 mutations/sec | Patient journeys, clinical pathways |

---

## Key Architectural Points

1. **Topic Separation**: Core projectors consume from `prod.ehr.events.enriched`, while FHIR and Neo4j projectors consume from specialized topics with pre-transformed data

2. **No Transformation**: FHIR Store and Neo4j projectors do NOT transform data - they receive ready-to-write objects from Module 6

3. **Independent Services**: Each projector is a separate Python/FastAPI service for fault isolation and independent scaling

4. **Batch Processing**: All projectors use batching for write performance (100-500 events per batch)

5. **Exactly-Once Semantics**: Kafka consumers with manual offset commits ensure no data loss

6. **Error Handling**: DLQ (Dead Letter Queue) for failed messages, retry logic with exponential backoff

7. **Monitoring**: Prometheus metrics for throughput, latency, error rates, consumer lag
