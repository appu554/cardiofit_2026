# Module 7: System Integration & Interoperability - Implementation Plan

## Executive Summary

Module 7 serves as the **Integration Hub** that connects the Flink analytics pipeline (Modules 1-6) with external healthcare systems, enabling bidirectional data flow and enrichment. This module transforms CardioFit from an isolated analytics system into a comprehensive healthcare interoperability platform.

**Target Performance**: 100K events/sec, <5s latency, 99.9% availability

---

## 1. Architecture Overview

### 1.1 System Context

```
External Systems → Module 7 Integration Hub → Flink Pipeline → Google FHIR Store/Neo4j
     ↓                        ↓                      ↓                    ↓
  HL7 v2              External APIs            Modules 1-6         Apollo Federation
  FHIR APIs           Data Quality            Analytics            Notification Service
  DICOM               Data Lake               Dashboards           EHR UI
  Message Queues      Audit/Lineage
```

### 1.2 Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    MODULE 7 INTEGRATION HUB                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  HL7 v2      │    │  FHIR API    │    │  DICOM       │      │
│  │  Parser      │    │  Connector   │    │  Processor   │      │
│  │  (ADT/ORM/   │    │  (R4)        │    │  (Metadata)  │      │
│  │   ORU/DFT)   │    │              │    │              │      │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘      │
│         │                   │                   │               │
│         └───────────────────┴───────────────────┘               │
│                             ↓                                    │
│                  ┌──────────────────────┐                       │
│                  │  Event Normalizer    │                       │
│                  │  (to common schema)  │                       │
│                  └──────────┬───────────┘                       │
│                             ↓                                    │
│         ┌───────────────────┴───────────────────┐               │
│         ↓                                       ↓               │
│  ┌─────────────────┐                   ┌─────────────────┐     │
│  │  Data Quality   │                   │  External API   │     │
│  │  Validator      │                   │  Enrichment     │     │
│  │  (Completeness, │                   │  (Drug DB,      │     │
│  │   Consistency)  │                   │   Guidelines)   │     │
│  └────────┬────────┘                   └────────┬────────┘     │
│           │                                     │               │
│           └──────────────┬──────────────────────┘               │
│                          ↓                                      │
│              ┌──────────────────────┐                          │
│              │  Kafka Producer      │                          │
│              │  (integration-events)│                          │
│              └──────────┬───────────┘                          │
└─────────────────────────┼───────────────────────────────────────┘
                          │
                          ↓
              ┌───────────────────────┐
              │  Kafka Topics         │
              │  - hl7-events.v1      │
              │  - fhir-events.v1     │
              │  - enriched-events.v1 │
              └───────────┬───────────┘
                          │
                          ↓
              ┌───────────────────────┐
              │  Flink Processing     │
              │  (Modules 1-6)        │
              └───────────┬───────────┘
                          │
          ┌───────────────┼───────────────┐
          ↓               ↓               ↓
┌─────────────────┐ ┌──────────┐ ┌─────────────┐
│ Google FHIR     │ │  Neo4j   │ │  Data Lake  │
│ Store           │ │  Graph   │ │  (S3/Parquet)│
└─────────────────┘ └──────────┘ └─────────────┘
```

### 1.3 Integration Points with Existing System

| Component | Integration Method | Purpose |
|-----------|-------------------|---------|
| **Modules 1-6 (Flink)** | Kafka topics (hl7-events.v1, fhir-events.v1) | Feed normalized events into analytics pipeline |
| **Google FHIR Store** | Direct HTTP API calls | Store FHIR resources, query patient/practitioner data |
| **Neo4j** | Bolt protocol (port 7687) | Query care team relationships, store data lineage |
| **Apollo Federation** | GraphQL subgraph (port 8052) | Expose integration metrics and data quality dashboards |
| **Notification Service** | Kafka topics (enriched-events.v1) | Trigger alerts based on external data |
| **Redis** | Cache layer (port 6379) | Cache external API responses, FHIR queries |
| **PostgreSQL** | Audit database (port 5432) | Store integration logs, data lineage, quality metrics |

---

## 2. Component Specifications

### 2.1 HL7 v2 Parser Service

**Purpose**: Parse HL7 v2 messages from hospital systems and normalize to internal event format

**Technology Stack**:
- **Language**: Java 17 (integrates with Flink ecosystem)
- **Library**: HAPI v2.5.1
- **Framework**: Spring Boot 3.2 for REST API wrapper

**Message Types Supported**:
- **ADT** (Admission/Discharge/Transfer): Patient demographics, bed assignments
- **ORM** (Order): Lab orders, medication orders, imaging orders
- **ORU** (Observation Result): Lab results, vital signs
- **DFT** (Detailed Financial Transaction): Billing and charges
- **MDM** (Medical Document Management): Clinical notes, reports

**Key Capabilities**:
```java
public interface HL7ParserService {
    // Parse raw HL7 v2 message to structured object
    ParsedHL7Message parse(String rawHL7) throws HL7Exception;

    // Validate message against HL7 v2.5 specification
    ValidationResult validate(ParsedHL7Message message);

    // Convert to normalized event schema
    NormalizedEvent normalize(ParsedHL7Message message);

    // Extract FHIR resources from HL7 message
    List<FHIRResource> extractFHIRResources(ParsedHL7Message message);
}
```

**Data Model** (normalized event):
```json
{
  "eventId": "uuid",
  "eventType": "HL7_ADT_A01",
  "sourceSystem": "hospital_ehr_system",
  "timestamp": "2025-11-11T10:30:00Z",
  "patientId": "fhir_patient_id",
  "facilityId": "fhir_organization_id",
  "rawHL7": "MSH|^~\\&|...",
  "parsedSegments": {
    "PID": { "patientName": "John Doe", "mrn": "12345" },
    "PV1": { "encounterType": "INPATIENT", "admitDateTime": "..." }
  },
  "extractedFHIR": [
    { "resourceType": "Patient", "id": "..." },
    { "resourceType": "Encounter", "id": "..." }
  ],
  "metadata": {
    "receivedAt": "2025-11-11T10:30:01Z",
    "processingLatency": 120
  }
}
```

**API Endpoints**:
- `POST /api/v1/hl7/parse` - Parse and normalize HL7 message
- `POST /api/v1/hl7/batch` - Batch processing of multiple messages
- `GET /api/v1/hl7/status/{eventId}` - Check processing status

**Kafka Output**:
- Topic: `hl7-events.v1`
- Partitioning: By `patientId` for ordering guarantees
- Schema Registry: Avro schema for type safety

**Deployment**:
- Docker container
- Port: 8060 (HTTP REST API)
- Replicas: 3 (load balanced)
- Resources: 2 CPU cores, 4GB RAM per instance

---

### 2.2 FHIR API Connector

**Purpose**: Bidirectional FHIR R4 integration with external FHIR servers and Google FHIR Store

**Technology Stack**:
- **Language**: Python 3.11 (reuse existing shared FHIR client)
- **Framework**: FastAPI
- **Library**: HAPI FHIR Python client, fhir.resources

**Key Capabilities**:
```python
class FHIRConnectorService:
    # Inbound: Receive FHIR resources from external systems
    async def receive_fhir_resource(self, resource: dict) -> str:
        """Validate, normalize, and publish to Kafka"""
        pass

    # Outbound: Push analytics results back to external FHIR servers
    async def publish_fhir_resource(self, resource: dict, target_server: str) -> str:
        """Send FHIR resource to external server with retry"""
        pass

    # Mapping: Convert internal events to FHIR resources
    async def map_to_fhir(self, event: dict, resource_type: str) -> dict:
        """Map normalized event to FHIR R4 resource"""
        pass

    # Query: Search external FHIR servers for additional context
    async def search_external(self, resource_type: str, params: dict) -> List[dict]:
        """Search external FHIR server with caching"""
        pass
```

**FHIR Resource Mappings**:

| Internal Event Type | FHIR Resource(s) | Notes |
|---------------------|------------------|-------|
| Patient admission | Encounter + Patient | Create or update encounter |
| Lab order | ServiceRequest | Reference to Patient + Practitioner |
| Lab result | Observation | Bundle with DiagnosticReport |
| Medication order | MedicationRequest | Include dosage and timing |
| Vital signs | Observation (vital-signs profile) | Component for multi-value vitals |
| Clinical note | DocumentReference | Attachment with binary data |

**Configuration** (multi-tenant):
```yaml
fhir_servers:
  - id: "google_fhir_store"
    type: "google_healthcare"
    base_url: "https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/..."
    auth:
      type: "service_account"
      credentials_path: "/app/credentials/google-credentials.json"
    capabilities: ["read", "write", "search"]

  - id: "external_hospital_a"
    type: "hapi_fhir"
    base_url: "https://hospital-a.example.com/fhir"
    auth:
      type: "oauth2"
      client_id: "${HOSPITAL_A_CLIENT_ID}"
      client_secret: "${HOSPITAL_A_CLIENT_SECRET}"
    capabilities: ["read", "search"]
    rate_limit:
      requests_per_second: 10
      burst: 20
```

**API Endpoints**:
- `POST /api/v1/fhir/inbound/{resourceType}` - Receive FHIR resource from external system
- `POST /api/v1/fhir/outbound/{serverId}/{resourceType}` - Send resource to external server
- `POST /api/v1/fhir/batch` - Batch operations (FHIR Bundle)
- `GET /api/v1/fhir/search/{serverId}/{resourceType}` - Search external FHIR server

**Kafka Output**:
- Topic: `fhir-events.v1`
- Partitioning: By `subject.reference` (patient ID)
- Schema: FHIR JSON format (validated against R4 StructureDefinitions)

**Deployment**:
- Docker container
- Port: 8061 (HTTP REST API)
- Replicas: 3
- Resources: 1 CPU core, 2GB RAM per instance

---

### 2.3 External API Enrichment Service

**Purpose**: Enrich clinical events with external knowledge sources (drug databases, clinical guidelines, risk calculators)

**Technology Stack**:
- **Language**: Python 3.11 (async/await for concurrent API calls)
- **Framework**: FastAPI
- **Libraries**: aiohttp, Resilience4j pattern (circuit breaker, retry, rate limiter)

**External APIs Integrated**:

| API | Purpose | Rate Limit | Cache TTL |
|-----|---------|------------|-----------|
| **RxNorm/RxNav** | Drug information, interactions | 20 req/sec | 7 days |
| **OpenFDA** | Drug safety alerts, recalls | 240 req/min | 1 day |
| **ClinicalTrials.gov** | Relevant clinical trials | 10 req/sec | 30 days |
| **PubMed/LitCovid** | Latest clinical evidence | 3 req/sec | 7 days |
| **SNOMED CT** | Terminology mapping | N/A (local) | N/A |
| **LOINC** | Lab test codes | N/A (local) | N/A |

**Key Capabilities**:
```python
class EnrichmentService:
    async def enrich_medication_event(self, event: dict) -> dict:
        """Add drug interactions, contraindications, dosing guidelines"""
        # Concurrent API calls with circuit breaker
        drug_info = await self.rxnorm_client.get_drug_info(rxcui)
        interactions = await self.rxnorm_client.get_interactions(rxcui_list)
        safety_alerts = await self.openfda_client.get_alerts(drug_name)

        # Merge into enriched event
        return {**event, "enrichment": {...}}

    async def enrich_lab_result(self, event: dict) -> dict:
        """Add reference ranges, clinical significance, guidelines"""
        loinc_code = event["labTest"]["code"]
        reference_range = await self.loinc_client.get_reference_range(loinc_code)
        guidelines = await self.pubmed_client.search_guidelines(loinc_code)

        return {**event, "enrichment": {...}}

    async def enrich_diagnosis(self, event: dict) -> dict:
        """Add ICD-10 mapping, clinical trials, treatment guidelines"""
        snomed_code = event["diagnosis"]["code"]
        icd10_codes = await self.snomed_client.map_to_icd10(snomed_code)
        trials = await self.clinicaltrials_client.search(snomed_code)

        return {**event, "enrichment": {...}}
```

**Resilience Patterns**:

```python
# Circuit Breaker Configuration
circuit_breaker_config = {
    "failure_rate_threshold": 50,  # Open circuit at 50% failure rate
    "slow_call_rate_threshold": 50,  # Open at 50% slow calls
    "slow_call_duration_threshold": 3.0,  # 3 seconds
    "wait_duration_in_open_state": 60,  # 60 seconds before half-open
    "permitted_calls_in_half_open": 3,
    "sliding_window_size": 100,
    "minimum_calls": 10
}

# Retry Configuration
retry_config = {
    "max_attempts": 3,
    "wait_strategy": "exponential",  # 1s, 2s, 4s
    "retry_on_exceptions": [ConnectionError, TimeoutError],
    "retry_on_status_codes": [500, 502, 503, 504]
}

# Rate Limiter Configuration
rate_limiter_config = {
    "rxnorm": {"limit": 20, "period": 1},  # 20 req/sec
    "openfda": {"limit": 4, "period": 1},  # 240 req/min = 4 req/sec
    "clinicaltrials": {"limit": 10, "period": 1},
    "pubmed": {"limit": 3, "period": 1}
}
```

**Caching Strategy**:
- **L1 Cache**: Redis (hot data, <1 hour TTL)
- **L2 Cache**: PostgreSQL (warm data, 1-30 days TTL)
- **Cache Key Pattern**: `enrichment:{api}:{resource_type}:{identifier}:{version}`

**API Endpoints**:
- `POST /api/v1/enrich/medication` - Enrich medication event
- `POST /api/v1/enrich/lab` - Enrich lab result
- `POST /api/v1/enrich/diagnosis` - Enrich diagnosis
- `POST /api/v1/enrich/batch` - Batch enrichment

**Kafka Integration**:
- **Input Topic**: `fhir-events.v1`, `hl7-events.v1`
- **Output Topic**: `enriched-events.v1`
- **Processing**: Async consumer with parallel enrichment (10 concurrent tasks per instance)

**Deployment**:
- Docker container
- Port: 8062 (HTTP REST API)
- Replicas: 5 (high throughput)
- Resources: 2 CPU cores, 4GB RAM per instance

---

### 2.4 Data Quality Validator

**Purpose**: Ensure data completeness, consistency, and validity before entering analytics pipeline

**Technology Stack**:
- **Language**: Python 3.11
- **Framework**: Pydantic for schema validation
- **Libraries**: Great Expectations for data quality rules

**Validation Dimensions**:

| Dimension | Checks | Action on Failure |
|-----------|--------|-------------------|
| **Completeness** | Required fields present, non-null values | Reject + DLQ |
| **Consistency** | Cross-field validation, referential integrity | Reject + DLQ |
| **Conformity** | Data types, formats, code systems | Reject + DLQ |
| **Accuracy** | Range checks, pattern matching | Flag + warning |
| **Uniqueness** | Duplicate detection (event ID, message ID) | Deduplicate |
| **Timeliness** | Event timestamp within acceptable window | Flag + warning |

**Validation Rules Examples**:

```python
class EventValidationRules:
    @rule(name="patient_id_required")
    def validate_patient_id(self, event: dict) -> ValidationResult:
        if not event.get("patientId"):
            return ValidationResult(
                passed=False,
                severity="CRITICAL",
                message="Missing required field: patientId",
                remediation="Reject event, send to DLQ"
            )
        return ValidationResult(passed=True)

    @rule(name="timestamp_format")
    def validate_timestamp(self, event: dict) -> ValidationResult:
        timestamp = event.get("timestamp")
        if not self._is_iso8601(timestamp):
            return ValidationResult(
                passed=False,
                severity="CRITICAL",
                message=f"Invalid timestamp format: {timestamp}",
                remediation="Reject event, send to DLQ"
            )
        return ValidationResult(passed=True)

    @rule(name="timestamp_timeliness")
    def validate_timestamp_timeliness(self, event: dict) -> ValidationResult:
        event_time = parse_iso8601(event["timestamp"])
        now = datetime.now(timezone.utc)
        age = (now - event_time).total_seconds()

        if age > 86400:  # 24 hours
            return ValidationResult(
                passed=True,  # Don't reject, just flag
                severity="WARNING",
                message=f"Event is {age/3600:.1f} hours old",
                remediation="Flag for review, allow processing"
            )
        return ValidationResult(passed=True)

    @rule(name="fhir_reference_integrity")
    async def validate_fhir_references(self, event: dict) -> ValidationResult:
        # Check that referenced FHIR resources exist
        for resource in event.get("extractedFHIR", []):
            for ref in self._extract_references(resource):
                exists = await self.fhir_client.resource_exists(ref)
                if not exists:
                    return ValidationResult(
                        passed=False,
                        severity="ERROR",
                        message=f"Referenced resource not found: {ref}",
                        remediation="Create placeholder or reject"
                    )
        return ValidationResult(passed=True)
```

**Data Quality Metrics**:
- **Completeness Score**: % of required fields present
- **Validity Score**: % of fields passing format/range checks
- **Consistency Score**: % passing cross-field validation
- **Overall Quality Score**: Weighted average of above

**DLQ (Dead Letter Queue) Handling**:
```yaml
dlq_config:
  kafka_topic: "integration-dlq.v1"
  retention: 7 days
  reprocessing:
    manual_review_required: true
    auto_retry_after_fix: true
    max_retry_attempts: 3
```

**API Endpoints**:
- `POST /api/v1/validate` - Validate single event
- `POST /api/v1/validate/batch` - Batch validation
- `GET /api/v1/quality/metrics` - Get quality metrics
- `GET /api/v1/quality/dlq` - Query DLQ events

**Kafka Integration**:
- **Input Topics**: `hl7-events.v1`, `fhir-events.v1`, `enriched-events.v1`
- **Output Topics**:
  - Valid events → Original topic (passthrough)
  - Invalid events → `integration-dlq.v1`
  - Quality metrics → `data-quality-metrics.v1`

**Deployment**:
- Docker container
- Port: 8063 (HTTP REST API)
- Replicas: 3
- Resources: 1 CPU core, 2GB RAM per instance

---

### 2.5 Data Lake Writer

**Purpose**: Archive all integration events to S3-compatible storage in Parquet format for long-term analytics and compliance

**Technology Stack**:
- **Language**: Python 3.11 (Apache Arrow/PyArrow for Parquet)
- **Storage**: MinIO (S3-compatible) or AWS S3
- **Format**: Apache Parquet with Snappy compression
- **Catalog**: Delta Lake for ACID transactions and time travel

**Data Lake Structure**:

```
s3://cardiofit-data-lake/
├── raw/                          # Raw events (7 years retention)
│   ├── hl7/
│   │   └── year=2025/month=11/day=11/hour=10/
│   │       └── hl7-events-{partition}.parquet
│   ├── fhir/
│   │   └── year=2025/month=11/day=11/hour=10/
│   │       └── fhir-events-{partition}.parquet
│   └── enriched/
│       └── year=2025/month=11/day=11/hour=10/
│           └── enriched-events-{partition}.parquet
│
├── processed/                    # Analytics-ready data (3 years retention)
│   ├── patient_timelines/
│   ├── clinical_patterns/
│   └── alert_history/
│
└── audit/                        # Audit logs (10 years retention)
    └── integration_logs/
        └── year=2025/month=11/
            └── audit-{date}.parquet
```

**Parquet Schema** (for hl7-events):
```python
schema = pa.schema([
    pa.field('event_id', pa.string()),
    pa.field('event_type', pa.string()),
    pa.field('source_system', pa.string()),
    pa.field('timestamp', pa.timestamp('us', tz='UTC')),
    pa.field('patient_id', pa.string()),
    pa.field('facility_id', pa.string()),
    pa.field('raw_hl7', pa.string()),
    pa.field('parsed_segments', pa.struct([
        pa.field('PID', pa.string()),  # JSON string
        pa.field('PV1', pa.string()),
    ])),
    pa.field('extracted_fhir', pa.list_(pa.string())),  # List of JSON strings
    pa.field('metadata', pa.struct([
        pa.field('received_at', pa.timestamp('us', tz='UTC')),
        pa.field('processing_latency_ms', pa.int64()),
    ])),
    # Partition columns
    pa.field('year', pa.int32()),
    pa.field('month', pa.int32()),
    pa.field('day', pa.int32()),
    pa.field('hour', pa.int32()),
])
```

**Write Strategy**:
- **Batching**: 1000 events or 60 seconds (whichever comes first)
- **Compression**: Snappy (good balance of speed and ratio)
- **Partitioning**: By year/month/day/hour for efficient queries
- **File Size**: Target 128MB per file (optimal for analytics)

**Delta Lake Configuration**:
```python
delta_config = {
    "delta.logRetentionDuration": "interval 30 days",
    "delta.deletedFileRetentionDuration": "interval 7 days",
    "delta.enableChangeDataFeed": "true",  # For CDC
    "delta.autoOptimize.optimizeWrite": "true",
    "delta.autoOptimize.autoCompact": "true",
}
```

**Key Capabilities**:
```python
class DataLakeWriter:
    async def write_batch(self, events: List[dict], event_type: str):
        """Write batch of events to data lake with Delta Lake ACID guarantees"""

        # Convert to Arrow table
        table = self._to_arrow_table(events)

        # Add partition columns
        table = self._add_partitions(table)

        # Write to Delta Lake
        delta_path = f"s3://cardiofit-data-lake/raw/{event_type}/"
        DeltaTable.write(table, delta_path, mode="append",
                        partition_by=["year", "month", "day", "hour"])

    async def compact_partitions(self, date: str):
        """Compact small files in a partition for query performance"""
        pass

    async def query_time_range(self, start: datetime, end: datetime, event_type: str):
        """Query events from data lake with predicate pushdown"""
        pass
```

**Kafka Integration**:
- **Input Topics**: `hl7-events.v1`, `fhir-events.v1`, `enriched-events.v1`
- **Consumer Group**: `data-lake-writer-group`
- **Processing**: Micro-batching with 60-second window

**Deployment**:
- Docker container
- No HTTP API (pure Kafka consumer)
- Replicas: 2 (for high availability)
- Resources: 2 CPU cores, 4GB RAM per instance

---

### 2.6 DICOM Metadata Processor

**Purpose**: Extract clinical metadata from DICOM medical imaging files

**Technology Stack**:
- **Language**: Python 3.11
- **Library**: pydicom 2.4+
- **Framework**: FastAPI for REST API

**Key Capabilities**:
```python
class DICOMProcessor:
    async def process_dicom_file(self, file_path: str) -> dict:
        """Extract metadata from DICOM file"""
        ds = pydicom.dcmread(file_path)

        return {
            "studyInstanceUID": str(ds.StudyInstanceUID),
            "seriesInstanceUID": str(ds.SeriesInstanceUID),
            "sopInstanceUID": str(ds.SOPInstanceUID),
            "patientID": str(ds.PatientID),
            "studyDate": str(ds.StudyDate),
            "modality": str(ds.Modality),  # CT, MRI, XR, etc.
            "bodyPartExamined": str(ds.BodyPartExamined),
            "studyDescription": str(ds.StudyDescription),
            # Extract relevant clinical fields
            "acquisition": self._extract_acquisition_params(ds),
            "protocol": self._extract_protocol_info(ds),
        }

    def map_to_fhir_imaging_study(self, dicom_metadata: dict) -> dict:
        """Convert DICOM metadata to FHIR ImagingStudy resource"""
        return {
            "resourceType": "ImagingStudy",
            "id": dicom_metadata["studyInstanceUID"],
            "identifier": [...],
            "status": "available",
            "subject": {"reference": f"Patient/{dicom_metadata['patientID']}"},
            "started": dicom_metadata["studyDate"],
            "modality": [{"system": "http://dicom.nema.org/resources/ontology/DCM",
                         "code": dicom_metadata["modality"]}],
            "description": dicom_metadata["studyDescription"],
        }
```

**API Endpoints**:
- `POST /api/v1/dicom/process` - Process DICOM file (multipart upload)
- `POST /api/v1/dicom/batch` - Batch process multiple files
- `GET /api/v1/dicom/study/{studyUID}` - Retrieve processed metadata

**Kafka Output**:
- Topic: `dicom-events.v1`
- Schema: DICOM metadata + FHIR ImagingStudy resource

**Deployment**:
- Docker container
- Port: 8064 (HTTP REST API)
- Replicas: 2
- Resources: 2 CPU cores, 4GB RAM per instance
- Storage: Ephemeral (files deleted after processing)

---

### 2.7 Message Queue Handler (RabbitMQ/SQS)

**Purpose**: Integrate with external message queues for asynchronous event ingestion

**Technology Stack**:
- **Language**: Python 3.11
- **Libraries**: aio-pika (RabbitMQ), aioboto3 (AWS SQS)

**Key Capabilities**:
```python
class MessageQueueHandler:
    async def consume_rabbitmq(self, queue_name: str):
        """Consume messages from RabbitMQ and publish to Kafka"""
        connection = await aio_pika.connect_robust(self.rabbitmq_url)
        channel = await connection.channel()
        queue = await channel.declare_queue(queue_name, durable=True)

        async for message in queue:
            async with message.process():
                # Convert to normalized event
                event = self._normalize_message(message.body)
                # Publish to Kafka
                await self.kafka_producer.send("external-events.v1", event)

    async def consume_sqs(self, queue_url: str):
        """Consume messages from AWS SQS and publish to Kafka"""
        session = aioboto3.Session()
        async with session.client('sqs') as sqs:
            while True:
                response = await sqs.receive_message(
                    QueueUrl=queue_url,
                    MaxNumberOfMessages=10,
                    WaitTimeSeconds=20  # Long polling
                )
                for message in response.get('Messages', []):
                    event = self._normalize_message(message['Body'])
                    await self.kafka_producer.send("external-events.v1", event)
                    await sqs.delete_message(
                        QueueUrl=queue_url,
                        ReceiptHandle=message['ReceiptHandle']
                    )
```

**Configuration**:
```yaml
message_queues:
  rabbitmq:
    url: "amqp://user:pass@rabbitmq-host:5672/"
    queues:
      - name: "hospital_a_hl7"
        routing_key: "hl7.#"
        durable: true
        auto_ack: false

  aws_sqs:
    region: "us-east-1"
    queues:
      - url: "https://sqs.us-east-1.amazonaws.com/123456789/hospital-b-events"
        visibility_timeout: 30
        wait_time_seconds: 20
```

**Deployment**:
- Docker container
- No HTTP API (pure message queue consumer)
- Replicas: 2
- Resources: 1 CPU core, 1GB RAM per instance

---

## 3. Integration with Existing Modules

### 3.1 Flink Pipeline Integration (Modules 1-6)

**Module 7 → Flink Data Flow**:

```
Module 7 Services → Kafka Topics → Flink Sources (Module 1) → Processing → Sinks
     ↓                  ↓
hl7-events.v1      Patient events
fhir-events.v1     Observation events
enriched-events.v1 Medication events
dicom-events.v1    Imaging study events
```

**New Kafka Topics**:

| Topic Name | Schema | Producers | Consumers |
|------------|--------|-----------|-----------|
| `hl7-events.v1` | Normalized HL7 event | HL7 Parser | Module 1 (Ingestion) |
| `fhir-events.v1` | FHIR R4 resource | FHIR Connector | Module 1 (Ingestion) |
| `enriched-events.v1` | Enriched event | Enrichment Service | Module 2 (Context Assembly) |
| `dicom-events.v1` | DICOM metadata | DICOM Processor | Module 1 (Ingestion) |
| `integration-dlq.v1` | Failed events | Data Quality Validator | Manual review service |
| `data-quality-metrics.v1` | Quality metrics | Data Quality Validator | Module 6 (Dashboards) |

**Flink Source Configuration** (add to Module 1):

```java
// HL7 Events Source
DataStream<NormalizedEvent> hl7Stream = env
    .addSource(new FlinkKafkaConsumer<>(
        "hl7-events.v1",
        new NormalizedEventDeserializationSchema(),
        kafkaProps
    ))
    .name("HL7 Events Source")
    .uid("hl7-events-source");

// FHIR Events Source
DataStream<FHIREvent> fhirStream = env
    .addSource(new FlinkKafkaConsumer<>(
        "fhir-events.v1",
        new FHIREventDeserializationSchema(),
        kafkaProps
    ))
    .name("FHIR Events Source")
    .uid("fhir-events-source");

// Union with existing patient-events-v1 stream
DataStream<CanonicalEvent> unifiedStream = hl7Stream
    .map(new HL7ToCanonicalMapper())
    .union(fhirStream.map(new FHIRToCanonicalMapper()))
    .union(existingPatientEventsStream);
```

**Module 6 Dashboard Integration**:
- Display data quality metrics from `data-quality-metrics.v1` topic
- Show integration throughput (events/sec by source system)
- Alert on DLQ event accumulation
- Display enrichment coverage (% events enriched)

---

### 3.2 Google FHIR Store Integration

**Bidirectional Data Flow**:

```
External Systems → Module 7 → Kafka → Flink → Google FHIR Store
                                              ↓
                                     Module 7 queries for context
```

**Write Path** (Module 7 → FHIR Store):
1. Flink processes events and creates FHIR resources
2. Flink writes to Kafka topic `fhir-write-requests.v1`
3. Module 7 FHIR Connector consumes and writes to Google FHIR Store

**Read Path** (FHIR Store → Module 7):
1. Module 7 queries FHIR Store for patient/practitioner context during enrichment
2. Results cached in Redis (5-minute TTL)

**FHIR Store Operations**:
```python
# Write FHIR resources from Flink
async def write_fhir_from_flink(self, resource: dict):
    resource_type = resource["resourceType"]
    resource_id = resource.get("id")

    if resource_id:
        # Update existing resource
        await self.fhir_client.update_resource(resource_type, resource_id, resource)
    else:
        # Create new resource
        result = await self.fhir_client.create_resource(resource_type, resource)
        return result["id"]

# Read context for enrichment
async def get_patient_context(self, patient_id: str) -> dict:
    # Check cache first
    cached = await self.redis.get(f"patient:{patient_id}")
    if cached:
        return json.loads(cached)

    # Query FHIR Store
    patient = await self.fhir_client.get_resource("Patient", patient_id)
    conditions = await self.fhir_client.search_resources(
        "Condition", {"subject": f"Patient/{patient_id}"}
    )
    medications = await self.fhir_client.search_resources(
        "MedicationRequest", {"subject": f"Patient/{patient_id}", "status": "active"}
    )

    context = {
        "patient": patient,
        "conditions": conditions,
        "medications": medications
    }

    # Cache for 5 minutes
    await self.redis.setex(f"patient:{patient_id}", 300, json.dumps(context))
    return context
```

---

### 3.3 Neo4j Integration

**Data Lineage Tracking**:

```cypher
// Track integration event lineage
CREATE (source:ExternalSystem {
  id: "hospital_a_ehr",
  name: "Hospital A EHR System",
  type: "HL7_v2"
})

CREATE (event:IntegrationEvent {
  eventId: "evt_12345",
  eventType: "HL7_ADT_A01",
  timestamp: datetime("2025-11-11T10:30:00Z"),
  sourceSystem: "hospital_a_ehr"
})

CREATE (fhirResource:FHIRResource {
  resourceType: "Encounter",
  resourceId: "enc_67890",
  created: datetime()
})

CREATE (source)-[:GENERATED]->(event)
CREATE (event)-[:TRANSFORMED_TO]->(fhirResource)
CREATE (fhirResource)-[:STORED_IN]->(:DataStore {name: "Google FHIR Store"})
```

**Query Care Team for Alert Routing** (existing pattern):
```cypher
// Find care team for patient
MATCH (p:Patient {id: $patientId})-[:HAS_PROVIDER]->(provider:Provider)
RETURN provider.id, provider.name, provider.role
```

**Module 7 Neo4j Operations**:
```python
async def record_integration_lineage(self, event: dict, fhir_resources: List[dict]):
    """Record data lineage in Neo4j"""
    async with self.neo4j_driver.session() as session:
        await session.run("""
            MERGE (source:ExternalSystem {id: $sourceSystem})
            CREATE (event:IntegrationEvent {
                eventId: $eventId,
                eventType: $eventType,
                timestamp: datetime($timestamp)
            })
            CREATE (source)-[:GENERATED]->(event)

            UNWIND $fhirResources AS resource
            CREATE (fhirRes:FHIRResource {
                resourceType: resource.resourceType,
                resourceId: resource.id,
                created: datetime()
            })
            CREATE (event)-[:TRANSFORMED_TO]->(fhirRes)
        """, {
            "sourceSystem": event["sourceSystem"],
            "eventId": event["eventId"],
            "eventType": event["eventType"],
            "timestamp": event["timestamp"],
            "fhirResources": fhir_resources
        })
```

---

### 3.4 Apollo Federation Integration

**New GraphQL Subgraph**: Integration Metrics Service

**Schema** (`schemas/integration-service.graphql`):
```graphql
extend schema
  @link(url: "https://specs.apollo.dev/federation/v2.0",
        import: ["@key", "@shareable"])

type Query {
  integrationMetrics(timeRange: TimeRange!): IntegrationMetrics!
  dataQualityReport(sourceSystem: String, timeRange: TimeRange!): DataQualityReport!
  integrationHealth: [SystemHealth!]!
  dlqEvents(limit: Int = 100, offset: Int = 0): DLQEventsConnection!
}

type IntegrationMetrics {
  totalEventsProcessed: Int!
  eventsPerSecond: Float!
  averageLatency: Float!
  errorRate: Float!
  bySourceSystem: [SourceSystemMetrics!]!
  byEventType: [EventTypeMetrics!]!
}

type SourceSystemMetrics {
  systemId: String!
  systemName: String!
  eventsProcessed: Int!
  successRate: Float!
  averageLatency: Float!
  lastEventTimestamp: String!
}

type DataQualityReport {
  overallScore: Float!
  completenessScore: Float!
  validityScore: Float!
  consistencyScore: Float!
  failedValidations: [ValidationFailure!]!
}

type SystemHealth {
  serviceId: String!
  serviceName: String!
  status: HealthStatus!
  uptime: Float!
  lastHealthCheck: String!
  errorCount: Int!
}

enum HealthStatus {
  HEALTHY
  DEGRADED
  UNHEALTHY
}

type DLQEventsConnection {
  totalCount: Int!
  events: [DLQEvent!]!
  pageInfo: PageInfo!
}

type DLQEvent {
  eventId: String!
  eventType: String!
  sourceSystem: String!
  failureReason: String!
  failureTimestamp: String!
  retryCount: Int!
  rawPayload: String!
}

input TimeRange {
  startTime: String!
  endTime: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
}
```

**Resolvers** (`resolvers/integration-service-resolvers.js`):
```javascript
const resolvers = {
  Query: {
    integrationMetrics: async (_, { timeRange }, { dataSources }) => {
      return await dataSources.integrationMetricsAPI.getMetrics(timeRange);
    },

    dataQualityReport: async (_, { sourceSystem, timeRange }, { dataSources }) => {
      return await dataSources.dataQualityAPI.getReport(sourceSystem, timeRange);
    },

    integrationHealth: async (_, __, { dataSources }) => {
      return await dataSources.healthCheckAPI.getSystemHealth();
    },

    dlqEvents: async (_, { limit, offset }, { dataSources }) => {
      return await dataSources.dlqAPI.getEvents(limit, offset);
    }
  }
};

module.exports = resolvers;
```

**Apollo Federation Registration** (add to `index.js`):
```javascript
const federationServices = [
  { name: 'analytics', url: 'http://localhost:8050/graphql' },
  { name: 'integration', url: 'http://localhost:8065/graphql' }, // NEW
];
```

**Integration Service Deployment**:
- Docker container
- Port: 8065 (GraphQL API)
- Data sources: PostgreSQL (audit DB), Redis (metrics cache), Kafka (real-time metrics)
- Replicas: 2

---

### 3.5 Notification Service Integration

**Alert Routing with External Data**:

```python
# In Notification Service: Enrich alerts with external context
async def route_alert_with_external_context(self, alert: dict):
    patient_id = alert["patientId"]

    # Query Google FHIR Store for patient context
    patient_context = await self.fhir_client.get_patient_context(patient_id)

    # Query Neo4j for care team
    care_team = await self.neo4j_client.get_care_team(patient_id)

    # Check for relevant clinical trials (from enrichment)
    if alert.get("enrichment", {}).get("clinicalTrials"):
        alert["actionable_items"].append({
            "type": "CLINICAL_TRIAL",
            "trials": alert["enrichment"]["clinicalTrials"]
        })

    # Route to care team with full context
    for provider in care_team:
        await self.send_notification(provider["id"], alert)
```

**Kafka Topic Subscriptions** (Notification Service):
- Subscribe to `enriched-events.v1` for events with external API enrichment
- Use enrichment data to add clinical trial information, drug safety alerts to notifications

---

## 4. Implementation Phases

### Phase 1: Foundation (Weeks 1-2)

**Goal**: Set up core infrastructure and HL7 parsing capability

**Deliverables**:
1. ✅ Docker Compose setup for all Module 7 services
2. ✅ Kafka topics created (`hl7-events.v1`, `fhir-events.v1`, etc.)
3. ✅ HL7 Parser Service (Java + HAPI v2.5)
   - REST API for parsing HL7 messages
   - Kafka producer for normalized events
   - Support for ADT, ORM, ORU message types
4. ✅ Data Quality Validator (Python + Pydantic)
   - Basic validation rules (completeness, consistency)
   - DLQ implementation
5. ✅ PostgreSQL audit database schema
6. ✅ Redis cache setup

**Acceptance Criteria**:
- HL7 Parser can process 1000 ADT messages/sec
- Data Quality Validator rejects invalid events to DLQ
- Kafka topics operational with at least 3 partitions each

**Dependencies**:
- Kafka cluster operational (existing)
- Docker environment configured

---

### Phase 2: FHIR & External APIs (Weeks 3-4)

**Goal**: Add FHIR connectivity and external API enrichment

**Deliverables**:
1. ✅ FHIR API Connector (Python + FastAPI)
   - Inbound: receive FHIR resources from external systems
   - Outbound: push FHIR resources to Google FHIR Store
   - Multi-tenant configuration for multiple FHIR servers
2. ✅ External API Enrichment Service (Python + aiohttp)
   - RxNorm/RxNav integration for drug information
   - OpenFDA integration for safety alerts
   - Circuit breaker pattern with Resilience4j
   - Redis caching layer (L1 cache)
3. ✅ Update Flink Module 1 to consume `fhir-events.v1` topic
4. ✅ Integration testing: HL7 → FHIR → Enrichment → Flink

**Acceptance Criteria**:
- FHIR Connector handles 500 resources/sec
- Enrichment Service achieves 50 enrichments/sec with <2s latency
- Circuit breaker triggers correctly on API failures
- 80% cache hit rate for common drug queries

**Dependencies**:
- Phase 1 complete
- Google FHIR Store credentials configured
- External API keys obtained (RxNorm, OpenFDA)

---

### Phase 3: Data Lake & Quality (Weeks 5-6)

**Goal**: Implement data lake archival and comprehensive data quality

**Deliverables**:
1. ✅ Data Lake Writer (Python + PyArrow)
   - Write events to Parquet format
   - Delta Lake ACID guarantees
   - Partitioning by year/month/day/hour
2. ✅ MinIO setup (S3-compatible storage)
3. ✅ Enhanced Data Quality Validator
   - Advanced validation rules (timeliness, uniqueness)
   - Quality scoring and metrics
   - Integration with Great Expectations
4. ✅ Data Quality Dashboard (GraphQL endpoint)
5. ✅ DLQ reprocessing workflow

**Acceptance Criteria**:
- 100% of events archived to data lake within 5 minutes
- Data lake queries return results in <10 seconds for 1-day range
- Data Quality Dashboard shows real-time metrics
- DLQ events can be manually reviewed and reprocessed

**Dependencies**:
- Phase 2 complete
- MinIO storage configured (1TB minimum)

---

### Phase 4: DICOM & Message Queues (Weeks 7-8)

**Goal**: Add medical imaging and external queue integrations

**Deliverables**:
1. ✅ DICOM Metadata Processor (Python + pydicom)
   - Extract metadata from DICOM files
   - Map to FHIR ImagingStudy resources
   - Kafka producer for `dicom-events.v1`
2. ✅ Message Queue Handler (Python + aio-pika + aioboto3)
   - RabbitMQ consumer
   - AWS SQS consumer
   - Configurable queue mappings
3. ✅ Flink Module 1 update to consume `dicom-events.v1`
4. ✅ End-to-end testing: RabbitMQ → Module 7 → Flink → FHIR Store

**Acceptance Criteria**:
- DICOM Processor handles 100 files/sec
- Message Queue Handler maintains <1s latency from queue to Kafka
- DICOM metadata available in Module 6 dashboard

**Dependencies**:
- Phase 3 complete
- RabbitMQ instance configured
- AWS SQS queue access (if applicable)
- Sample DICOM files for testing

---

### Phase 5: Apollo Federation & Neo4j (Weeks 9-10)

**Goal**: Integrate with Apollo Federation and Neo4j for full-stack visibility

**Deliverables**:
1. ✅ Integration Metrics GraphQL Service (Node.js)
   - GraphQL schema and resolvers
   - Register with Apollo Federation
   - Real-time metrics from Kafka + PostgreSQL
2. ✅ Neo4j data lineage implementation
   - Record integration event lineage
   - Query data provenance
   - Cypher queries for care team routing
3. ✅ Update Notification Service to consume enriched events
4. ✅ EHR UI dashboard for integration monitoring

**Acceptance Criteria**:
- GraphQL queries return integration metrics in <500ms
- Neo4j lineage graph shows complete event-to-FHIR-resource paths
- Notification Service routes alerts with enriched clinical trial data
- EHR UI displays integration health dashboard

**Dependencies**:
- Phase 4 complete
- Apollo Federation gateway operational (existing)
- Neo4j instance configured (existing)

---

### Phase 6: Performance & Compliance (Weeks 11-12)

**Goal**: Optimize for 100K events/sec and ensure HIPAA compliance

**Deliverables**:
1. ✅ Performance testing and optimization
   - Load testing with K6 or JMeter
   - Kafka partition tuning
   - Redis cache optimization
   - Flink parallelism tuning
2. ✅ Security hardening
   - Encrypt Kafka topics (TLS)
   - Encrypt data lake (S3 server-side encryption)
   - Audit logging for all FHIR operations
   - PHI de-identification for non-production environments
3. ✅ Monitoring and alerting
   - Prometheus metrics for all services
   - Grafana dashboards
   - PagerDuty integration for critical alerts
4. ✅ Documentation
   - Architecture diagrams
   - API documentation (OpenAPI/GraphQL)
   - Runbooks for operations
   - Disaster recovery procedures

**Acceptance Criteria**:
- System sustains 100K events/sec with <5s end-to-end latency
- 99.9% availability over 1-week test period
- All PHI encrypted at rest and in transit
- Complete audit trail for compliance
- Zero critical security vulnerabilities (OWASP scan)

**Dependencies**:
- Phase 5 complete
- Production-like environment for load testing
- Security scanning tools configured

---

## 5. Technology Stack Summary

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **HL7 Parser** | Java + Spring Boot | 17 + 3.2 | Parse HL7 v2 messages |
| | HAPI v2.5 | 2.5.1 | HL7 parsing library |
| **FHIR Connector** | Python + FastAPI | 3.11 + 0.104 | FHIR API integration |
| | fhir.resources | 7.1.0 | FHIR R4 models |
| **Enrichment Service** | Python + aiohttp | 3.11 + 3.9 | External API calls |
| | Resilience4j patterns | N/A | Circuit breaker, retry |
| **Data Quality** | Python + Pydantic | 3.11 + 2.5 | Schema validation |
| | Great Expectations | 0.18 | Data quality rules |
| **Data Lake** | Python + PyArrow | 3.11 + 14.0 | Parquet writing |
| | Delta Lake | 3.0 | ACID transactions |
| **DICOM** | Python + pydicom | 3.11 + 2.4 | DICOM parsing |
| **Message Queues** | Python + aio-pika | 3.11 + 9.3 | RabbitMQ client |
| | aioboto3 | 12.3 | AWS SQS client |
| **GraphQL Service** | Node.js + Apollo | 20 + 4.0 | Federation subgraph |
| **Storage** | MinIO | Latest | S3-compatible storage |
| | PostgreSQL | 15 | Audit logs, metrics |
| | Redis | 7.2 | Cache layer |
| **Messaging** | Apache Kafka | 3.6 | Event streaming |
| **Monitoring** | Prometheus + Grafana | Latest | Metrics & dashboards |

---

## 6. Configuration & Environment Variables

### 6.1 Global Configuration

**`docker-compose.yml`** (add to existing compose file):

```yaml
version: '3.8'

services:
  # HL7 Parser Service
  hl7-parser:
    build: ./backend/services/module7-integration/hl7-parser
    ports:
      - "8060:8060"
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_HL7: "hl7-events.v1"
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}
      LOG_LEVEL: ${LOG_LEVEL:-INFO}
    depends_on:
      - kafka
      - postgres
      - redis
    networks:
      - cardiofit-network

  # FHIR Connector Service
  fhir-connector:
    build: ./backend/services/module7-integration/fhir-connector
    ports:
      - "8061:8061"
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_FHIR: "fhir-events.v1"
      GOOGLE_APPLICATION_CREDENTIALS: /app/credentials/google-credentials.json
      FHIR_STORE_PROJECT: ${FHIR_STORE_PROJECT:-cardiofit-905a8}
      FHIR_STORE_LOCATION: ${FHIR_STORE_LOCATION:-us-central1}
      FHIR_STORE_DATASET: ${FHIR_STORE_DATASET:-clinical_synthesis_hub}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
    volumes:
      - ./backend/shared/credentials:/app/credentials:ro
    depends_on:
      - kafka
      - postgres
      - redis
    networks:
      - cardiofit-network

  # External API Enrichment Service
  enrichment-service:
    build: ./backend/services/module7-integration/enrichment-service
    ports:
      - "8062:8062"
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_INPUT: "fhir-events.v1"
      KAFKA_TOPIC_OUTPUT: "enriched-events.v1"
      RXNORM_API_KEY: ${RXNORM_API_KEY}
      OPENFDA_API_KEY: ${OPENFDA_API_KEY}
      CLINICALTRIALS_API_KEY: ${CLINICALTRIALS_API_KEY}
      PUBMED_API_KEY: ${PUBMED_API_KEY}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
      CIRCUIT_BREAKER_THRESHOLD: ${CIRCUIT_BREAKER_THRESHOLD:-50}
      CACHE_TTL_SECONDS: ${CACHE_TTL_SECONDS:-604800}  # 7 days
    depends_on:
      - kafka
      - postgres
      - redis
    networks:
      - cardiofit-network

  # Data Quality Validator
  data-quality-validator:
    build: ./backend/services/module7-integration/data-quality-validator
    ports:
      - "8063:8063"
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPICS_INPUT: "hl7-events.v1,fhir-events.v1,enriched-events.v1"
      KAFKA_TOPIC_DLQ: "integration-dlq.v1"
      KAFKA_TOPIC_METRICS: "data-quality-metrics.v1"
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
      VALIDATION_SEVERITY_REJECT: "CRITICAL,ERROR"
      VALIDATION_SEVERITY_WARN: "WARNING"
    depends_on:
      - kafka
      - postgres
    networks:
      - cardiofit-network

  # Data Lake Writer
  data-lake-writer:
    build: ./backend/services/module7-integration/data-lake-writer
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPICS_INPUT: "hl7-events.v1,fhir-events.v1,enriched-events.v1"
      S3_ENDPOINT: ${S3_ENDPOINT:-http://minio:9000}
      S3_ACCESS_KEY: ${S3_ACCESS_KEY:-minioadmin}
      S3_SECRET_KEY: ${S3_SECRET_KEY:-minioadmin}
      S3_BUCKET: ${S3_BUCKET:-cardiofit-data-lake}
      BATCH_SIZE: ${BATCH_SIZE:-1000}
      BATCH_TIMEOUT_SECONDS: ${BATCH_TIMEOUT_SECONDS:-60}
    depends_on:
      - kafka
      - minio
    networks:
      - cardiofit-network

  # DICOM Processor
  dicom-processor:
    build: ./backend/services/module7-integration/dicom-processor
    ports:
      - "8064:8064"
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_DICOM: "dicom-events.v1"
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
    depends_on:
      - kafka
      - postgres
    networks:
      - cardiofit-network

  # Message Queue Handler
  message-queue-handler:
    build: ./backend/services/module7-integration/message-queue-handler
    environment:
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_OUTPUT: "external-events.v1"
      RABBITMQ_URL: ${RABBITMQ_URL}
      RABBITMQ_QUEUES: ${RABBITMQ_QUEUES}
      AWS_SQS_REGION: ${AWS_SQS_REGION}
      AWS_SQS_QUEUES: ${AWS_SQS_QUEUES}
    depends_on:
      - kafka
    networks:
      - cardiofit-network

  # Integration Metrics GraphQL Service
  integration-metrics-service:
    build: ./backend/services/module7-integration/integration-metrics-service
    ports:
      - "8065:8065"
    environment:
      PORT: 8065
      DATABASE_URL: ${DATABASE_URL:-postgresql://user:pass@postgres:5432/integration_db}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}
      KAFKA_BROKERS: ${KAFKA_BROKERS:-kafka:9092}
      KAFKA_TOPIC_METRICS: "data-quality-metrics.v1"
    depends_on:
      - postgres
      - redis
      - kafka
    networks:
      - cardiofit-network

  # MinIO (S3-compatible storage)
  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"  # API
      - "9001:9001"  # Console
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
    command: server /data --console-address ":9001"
    volumes:
      - minio-data:/data
    networks:
      - cardiofit-network

volumes:
  minio-data:

networks:
  cardiofit-network:
    external: true
```

### 6.2 Environment Variables (`.env` file)

```bash
# Kafka Configuration
KAFKA_BROKERS=localhost:9092,localhost:9093,localhost:9094

# Database Configuration
DATABASE_URL=postgresql://cardiofit_user:cardiofit_pass@localhost:5432/integration_db

# Redis Configuration
REDIS_URL=redis://localhost:6379

# Google FHIR Store
FHIR_STORE_PROJECT=cardiofit-905a8
FHIR_STORE_LOCATION=us-central1
FHIR_STORE_DATASET=clinical_synthesis_hub

# External API Keys
RXNORM_API_KEY=your_rxnorm_api_key
OPENFDA_API_KEY=your_openfda_api_key
CLINICALTRIALS_API_KEY=your_clinicaltrials_api_key
PUBMED_API_KEY=your_pubmed_api_key

# MinIO (S3-compatible storage)
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=cardiofit-data-lake

# RabbitMQ (if applicable)
RABBITMQ_URL=amqp://user:pass@rabbitmq-host:5672/
RABBITMQ_QUEUES=hospital_a_hl7,hospital_b_orders

# AWS SQS (if applicable)
AWS_SQS_REGION=us-east-1
AWS_SQS_QUEUES=https://sqs.us-east-1.amazonaws.com/123456789/hospital-b-events

# Monitoring
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000

# Logging
LOG_LEVEL=INFO
```

---

## 7. Testing Strategy

### 7.1 Unit Testing

**Per Service**:
- HL7 Parser: Test parsing of all message types (ADT, ORM, ORU, DFT, MDM)
- FHIR Connector: Test FHIR resource validation and mapping
- Enrichment Service: Test external API calls with mocking
- Data Quality Validator: Test all validation rules
- Data Lake Writer: Test Parquet writing and Delta Lake operations

**Coverage Target**: >80% code coverage

### 7.2 Integration Testing

**End-to-End Flows**:
1. **HL7 Flow**: HL7 message → Parser → Kafka → Flink → FHIR Store
2. **FHIR Flow**: External FHIR resource → Connector → Kafka → Flink → FHIR Store
3. **Enrichment Flow**: Event → Enrichment → Kafka → Flink
4. **Data Lake Flow**: Event → Data Lake → Query results
5. **DLQ Flow**: Invalid event → DLQ → Manual review → Reprocess

**Tools**: Pytest, TestContainers (Docker), Kafka test harness

### 7.3 Performance Testing

**Load Testing**:
- **Tool**: K6 or Apache JMeter
- **Scenarios**:
  - 10K events/sec sustained for 1 hour
  - 50K events/sec sustained for 10 minutes
  - 100K events/sec burst for 1 minute
- **Metrics**:
  - End-to-end latency (p50, p95, p99)
  - Kafka lag
  - Error rate
  - Resource utilization (CPU, memory, disk I/O)

**Benchmark Targets**:
| Metric | Target | Critical Threshold |
|--------|--------|--------------------|
| Throughput | 100K events/sec | 50K events/sec |
| Latency (p95) | <5 seconds | <10 seconds |
| Error Rate | <0.1% | <1% |
| Availability | 99.9% | 99% |

### 7.4 Security Testing

**OWASP Top 10**:
- SQL Injection (database queries)
- XSS (GraphQL API)
- Authentication/Authorization (FHIR Store access)
- Sensitive Data Exposure (PHI encryption)
- XML External Entities (HL7 parsing)

**Tools**: OWASP ZAP, Burp Suite, Snyk

### 7.5 Compliance Testing

**HIPAA Audit**:
- ✅ PHI encrypted at rest (data lake, databases)
- ✅ PHI encrypted in transit (TLS for all services)
- ✅ Audit logging for all PHI access (FHIR Store operations)
- ✅ Access controls (RBAC for services)
- ✅ Data retention policies (7 years for raw events)

---

## 8. Deployment & Operations

### 8.1 Deployment Strategy

**Kubernetes Deployment** (production-ready):

```yaml
# deployment.yaml for HL7 Parser Service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hl7-parser
  namespace: cardiofit-module7
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hl7-parser
  template:
    metadata:
      labels:
        app: hl7-parser
    spec:
      containers:
      - name: hl7-parser
        image: cardiofit/hl7-parser:1.0.0
        ports:
        - containerPort: 8060
        env:
        - name: KAFKA_BROKERS
          valueFrom:
            configMapKeyRef:
              name: kafka-config
              key: brokers
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8060
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8060
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: hl7-parser-service
  namespace: cardiofit-module7
spec:
  selector:
    app: hl7-parser
  ports:
  - port: 8060
    targetPort: 8060
  type: ClusterIP
```

**Horizontal Pod Autoscaling**:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hl7-parser-hpa
  namespace: cardiofit-module7
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hl7-parser
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 8.2 Monitoring & Alerting

**Prometheus Metrics** (each service exposes):

```python
# Example metrics for Enrichment Service
from prometheus_client import Counter, Histogram, Gauge

enrichment_requests = Counter(
    'enrichment_requests_total',
    'Total enrichment requests',
    ['api', 'status']
)

enrichment_latency = Histogram(
    'enrichment_latency_seconds',
    'Enrichment request latency',
    ['api']
)

circuit_breaker_state = Gauge(
    'circuit_breaker_state',
    'Circuit breaker state (0=closed, 1=open, 2=half-open)',
    ['api']
)

cache_hit_rate = Gauge(
    'cache_hit_rate',
    'Cache hit rate percentage',
    ['cache_layer']
)
```

**Grafana Dashboards**:
1. **Integration Overview**: Total events processed, error rate, throughput
2. **Service Health**: Uptime, latency, error count per service
3. **Data Quality**: Quality scores, DLQ count, validation failures
4. **External APIs**: API latency, circuit breaker states, rate limit usage
5. **Data Lake**: Storage usage, query performance, partition health

**Alerting Rules** (PagerDuty integration):

```yaml
# prometheus-alerts.yaml
groups:
  - name: module7_alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: rate(enrichment_requests_total{status="error"}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in enrichment service"
          description: "Error rate is {{ $value }}% over the last 5 minutes"

      - alert: CircuitBreakerOpen
        expr: circuit_breaker_state > 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Circuit breaker open for {{ $labels.api }}"
          description: "Circuit breaker has been open for 2 minutes"

      - alert: DLQAccumulation
        expr: kafka_consumer_lag{topic="integration-dlq.v1"} > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "DLQ accumulating events"
          description: "{{ $value }} events in DLQ, review required"

      - alert: DataLakeWriteFailure
        expr: rate(data_lake_writes_failed_total[5m]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Data lake writes failing"
          description: "Data lake writes have been failing for 5 minutes"
```

### 8.3 Disaster Recovery

**Backup Strategy**:
- **Kafka**: Retain events for 7 days (configurable retention)
- **PostgreSQL**: Daily backups to S3, 30-day retention
- **Data Lake**: S3 versioning enabled, lifecycle policy to Glacier after 90 days
- **Neo4j**: Daily graph database backups

**Recovery Procedures**:
1. **Service Failure**: Kubernetes automatically restarts failed pods
2. **Kafka Broker Failure**: Replication factor 3, automatic leader election
3. **Database Failure**: Restore from latest backup, replay Kafka events since backup timestamp
4. **Data Lake Corruption**: Delta Lake time travel to previous version

**RTO/RPO Targets**:
- **RTO (Recovery Time Objective)**: <1 hour
- **RPO (Recovery Point Objective)**: <5 minutes (Kafka retention)

---

## 9. Security & Compliance

### 9.1 Encryption

**At Rest**:
- Kafka topics: LUKS encryption on broker volumes
- PostgreSQL: Transparent Data Encryption (TDE)
- Data Lake (S3): Server-side encryption (SSE-S3 or SSE-KMS)
- Redis: RDB persistence encrypted

**In Transit**:
- All service-to-service communication: TLS 1.3
- Kafka: SSL/TLS encryption
- External API calls: HTTPS only

### 9.2 Authentication & Authorization

**Service-to-Service**:
- mTLS (mutual TLS) for service mesh (Istio or Linkerd)
- Service accounts with least-privilege access

**External Systems**:
- OAuth 2.0 for FHIR servers
- API keys with rate limiting for external APIs
- Service account authentication for Google FHIR Store

### 9.3 Audit Logging

**Logged Events**:
- All FHIR Store read/write operations (patient ID, resource type, timestamp, user/service)
- External API calls (API endpoint, response status, latency)
- Data quality validation failures (event ID, reason, severity)
- DLQ events (event ID, failure reason, timestamp)

**Audit Log Storage**:
- PostgreSQL audit database
- Retention: 10 years (HIPAA requirement)
- Immutable logs (append-only table)

**Audit Query API**:
```graphql
type Query {
  auditLogs(
    resourceType: String,
    patientId: String,
    startTime: String!,
    endTime: String!,
    limit: Int = 100
  ): [AuditLogEntry!]!
}

type AuditLogEntry {
  timestamp: String!
  eventType: String!
  resourceType: String
  resourceId: String
  patientId: String
  userId: String
  serviceId: String
  action: String!
  result: String!
  ipAddress: String
}
```

---

## 10. Cost Estimation

### 10.1 Infrastructure Costs (Monthly, AWS-based)

| Component | Service | Configuration | Cost |
|-----------|---------|---------------|------|
| **Compute** | EKS + EC2 | 10 x m6i.2xlarge (8 vCPU, 32GB) | $1,400 |
| **Kafka** | MSK | 3 brokers (kafka.m5.large) | $600 |
| **PostgreSQL** | RDS PostgreSQL | db.m6g.xlarge (4 vCPU, 16GB) | $300 |
| **Redis** | ElastiCache | cache.m6g.large (2 vCPU, 6.38GB) | $150 |
| **Neo4j** | EC2 | m6i.xlarge (4 vCPU, 16GB) | $200 |
| **Data Lake** | S3 | 10TB Standard + 100TB Glacier | $350 |
| **Egress** | Data Transfer | 5TB/month | $450 |
| **Monitoring** | CloudWatch + Prometheus | Metrics, logs | $200 |
| **External APIs** | RxNorm, OpenFDA, etc. | API call volume | $500 |
| **Total** | | | **$4,150/month** |

### 10.2 Optimization Opportunities

- **Spot Instances**: Save 50-70% on compute costs for non-critical services
- **Reserved Instances**: 30-50% savings on RDS, ElastiCache with 1-year commitment
- **S3 Intelligent-Tiering**: Automatic cost optimization for data lake
- **Kafka Self-Managed**: Run Kafka on EC2 instead of MSK (save $400/month)

---

## 11. Success Metrics

### 11.1 Technical KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Throughput** | 100K events/sec | Kafka metrics |
| **Latency (p95)** | <5 seconds | End-to-end tracing |
| **Availability** | 99.9% | Uptime monitoring |
| **Error Rate** | <0.1% | Service logs |
| **Data Quality Score** | >95% | Validation metrics |
| **Cache Hit Rate** | >80% | Redis metrics |
| **API Enrichment Coverage** | >70% | Enrichment metrics |

### 11.2 Business KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Integration Partners** | 5 hospitals in 6 months | Partner onboarding tracker |
| **Event Volume** | 1M events/day after 3 months | Kafka throughput |
| **Data Lake Usage** | 50 queries/day by analytics team | Query logs |
| **Alert Enrichment Impact** | 30% reduction in alert fatigue | Notification service metrics |
| **Compliance Readiness** | 100% audit trail coverage | Audit log validation |

---

## 12. Next Steps

### Immediate Actions (Week 1)

1. **Review & Approve Plan**: Stakeholder alignment on scope and timeline
2. **Provision Infrastructure**:
   - Set up Kafka topics
   - Configure PostgreSQL audit database
   - Deploy MinIO for data lake
3. **Obtain API Keys**:
   - RxNorm/RxNav
   - OpenFDA
   - ClinicalTrials.gov
   - PubMed/NCBI
4. **Create GitHub Repository Structure**:
   ```
   backend/services/module7-integration/
   ├── hl7-parser/
   ├── fhir-connector/
   ├── enrichment-service/
   ├── data-quality-validator/
   ├── data-lake-writer/
   ├── dicom-processor/
   ├── message-queue-handler/
   ├── integration-metrics-service/
   ├── docker-compose.yml
   └── README.md
   ```
5. **Set Up CI/CD**:
   - GitHub Actions for automated testing
   - Docker image building and registry push
   - Kubernetes deployment automation

### Phase 1 Kickoff (Week 1-2)

- Sprint planning for HL7 Parser and Data Quality Validator
- Set up development environment for Java (HL7 Parser)
- Begin implementation of core HL7 parsing logic

---

## 13. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **External API rate limits** | High | Medium | Aggressive caching, graceful degradation |
| **Kafka scaling bottlenecks** | Medium | High | Partition tuning, consumer group optimization |
| **FHIR Store quota limits** | Medium | High | Request batching, quotas monitoring |
| **Data lake query performance** | Medium | Medium | Proper partitioning, Delta Lake optimization |
| **PHI data breach** | Low | Critical | Encryption, access controls, audit logging |
| **Integration partner delays** | High | Low | Mock external systems for development |

---

## Appendix A: HL7 v2 Message Examples

### ADT^A01 (Patient Admission)

```hl7
MSH|^~\&|SENDING_APP|SENDING_FACILITY|RECEIVING_APP|RECEIVING_FACILITY|20251111103000||ADT^A01|MSG00001|P|2.5|||AL||
EVN|A01|20251111103000|||
PID|1||MRN12345^^^HOSPITAL^MR||DOE^JOHN^A||19800115|M|||123 MAIN ST^^ANYTOWN^CA^12345^USA|||||||123-45-6789|||
PV1|1|I|WARD^ROOM101^BED1^HOSPITAL||||ATTEND123^SMITH^JANE^MD|||MED||||ADM|A0||||ADMIT123|||||||||||||||||HOSPITAL|||||20251111100000|||
```

### ORM^O01 (Lab Order)

```hl7
MSH|^~\&|LAB_SYS|HOSPITAL|LAB|HOSPITAL|20251111104500||ORM^O01|MSG00002|P|2.5|||AL||
PID|1||MRN12345^^^HOSPITAL^MR||DOE^JOHN^A||19800115|M|||
ORC|NW|ORDER001|||||^^^20251111104500||20251111104500|||ORDPHY123^JONES^ROBERT^MD|||
OBR|1|ORDER001||CBC^COMPLETE BLOOD COUNT^L|||20251111104500|||||||||||ORDPHY123^JONES^ROBERT^MD||||||20251111104500|||F||
```

---

## Appendix B: FHIR Resource Mapping Examples

### HL7 ADT^A01 → FHIR Encounter + Patient

**Input HL7**:
```hl7
MSH|^~\&|SENDING_APP|SENDING_FACILITY|RECEIVING_APP|RECEIVING_FACILITY|20251111103000||ADT^A01|MSG00001|P|2.5|||AL||
PID|1||MRN12345^^^HOSPITAL^MR||DOE^JOHN^A||19800115|M|||123 MAIN ST^^ANYTOWN^CA^12345^USA|||||||123-45-6789|||
PV1|1|I|WARD^ROOM101^BED1^HOSPITAL||||ATTEND123^SMITH^JANE^MD|||MED||||ADM|A0||||ADMIT123|||||||||||||||||HOSPITAL|||||20251111100000|||
```

**Output FHIR**:
```json
{
  "resourceType": "Bundle",
  "type": "transaction",
  "entry": [
    {
      "resource": {
        "resourceType": "Patient",
        "id": "MRN12345",
        "identifier": [
          {
            "system": "http://hospital.example.com/mrn",
            "value": "MRN12345"
          },
          {
            "system": "http://hl7.org/fhir/sid/us-ssn",
            "value": "123-45-6789"
          }
        ],
        "name": [
          {
            "use": "official",
            "family": "DOE",
            "given": ["JOHN", "A"]
          }
        ],
        "gender": "male",
        "birthDate": "1980-01-15",
        "address": [
          {
            "line": ["123 MAIN ST"],
            "city": "ANYTOWN",
            "state": "CA",
            "postalCode": "12345",
            "country": "USA"
          }
        ]
      },
      "request": {
        "method": "PUT",
        "url": "Patient/MRN12345"
      }
    },
    {
      "resource": {
        "resourceType": "Encounter",
        "id": "ADMIT123",
        "identifier": [
          {
            "system": "http://hospital.example.com/encounter",
            "value": "ADMIT123"
          }
        ],
        "status": "in-progress",
        "class": {
          "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
          "code": "IMP",
          "display": "inpatient encounter"
        },
        "subject": {
          "reference": "Patient/MRN12345"
        },
        "participant": [
          {
            "individual": {
              "reference": "Practitioner/ATTEND123",
              "display": "Dr. Jane Smith"
            }
          }
        ],
        "period": {
          "start": "2025-11-11T10:00:00Z"
        },
        "location": [
          {
            "location": {
              "display": "WARD ROOM101 BED1"
            }
          }
        ],
        "serviceProvider": {
          "reference": "Organization/HOSPITAL"
        }
      },
      "request": {
        "method": "PUT",
        "url": "Encounter/ADMIT123"
      }
    }
  ]
}
```

---

## Appendix C: Data Quality Validation Rules

### Completeness Rules

```python
completeness_rules = {
    "patient_id_required": {
        "field": "patientId",
        "condition": "not_null",
        "severity": "CRITICAL",
        "action": "reject"
    },
    "timestamp_required": {
        "field": "timestamp",
        "condition": "not_null",
        "severity": "CRITICAL",
        "action": "reject"
    },
    "source_system_required": {
        "field": "sourceSystem",
        "condition": "not_null",
        "severity": "ERROR",
        "action": "reject"
    }
}
```

### Consistency Rules

```python
consistency_rules = {
    "timestamp_order": {
        "condition": "event.timestamp <= current_time",
        "severity": "ERROR",
        "message": "Event timestamp is in the future",
        "action": "reject"
    },
    "patient_reference_exists": {
        "condition": "fhir_store.resource_exists('Patient', event.patientId)",
        "severity": "ERROR",
        "message": "Referenced patient does not exist",
        "action": "create_placeholder_or_reject"
    }
}
```

### Conformity Rules

```python
conformity_rules = {
    "timestamp_format": {
        "field": "timestamp",
        "regex": r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$",
        "severity": "CRITICAL",
        "action": "reject"
    },
    "event_type_valid": {
        "field": "eventType",
        "allowed_values": ["HL7_ADT_A01", "HL7_ORM_O01", "HL7_ORU_R01", "FHIR_OBSERVATION", ...],
        "severity": "ERROR",
        "action": "reject"
    }
}
```

---

**End of Module 7 Implementation Plan**

**Document Version**: 1.0
**Last Updated**: 2025-11-11
**Author**: Claude Code (CardioFit Platform)
**Status**: Ready for Review & Approval
