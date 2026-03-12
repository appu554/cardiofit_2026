# Module 8 Shared Components

Shared module for all Module 8 storage projector services providing:

- **Data Models**: Pydantic models for Kafka topic messages
- **Base Kafka Consumer**: Abstract base class with common functionality
- **Batch Processor**: Generic batch accumulator with size/time-based flushing
- **Metrics**: Standardized Prometheus metrics for all projectors

## Installation

```bash
cd backend/stream-services/module8-shared
pip install -r requirements.txt
```

## Data Models

### EnrichedClinicalEvent

From `prod.ehr.events.enriched` topic:

```python
from app.models import EnrichedClinicalEvent

event = EnrichedClinicalEvent(
    id="evt_12345",
    timestamp=1699564800000,
    eventType="VITAL_SIGNS",
    patientId="P12345",
    encounterId="E67890",
    departmentId="ICU_01",
    deviceId="MON_5678",
    rawData={
        "heart_rate": 95,
        "blood_pressure_systolic": 140,
        "blood_pressure_diastolic": 90,
        "spo2": 96,
        "temperature_celsius": 37.2
    },
    enrichments={
        "NEWS2Score": 4,
        "qSOFAScore": 1,
        "riskLevel": "MODERATE"
    }
)
```

### FHIRResource

From `prod.ehr.fhir.upsert` topic:

```python
from app.models import FHIRResource

resource = FHIRResource(
    resourceType="Observation",
    resourceId="obs_12345",
    patientId="P12345",
    lastUpdated=1699564800000,
    fhirData={
        "resourceType": "Observation",
        "id": "obs_12345",
        "status": "final",
        # ... complete FHIR R4 resource
    }
)

# Get Kafka key
key = resource.get_kafka_key()  # "Observation|obs_12345"
```

### GraphMutation

From `prod.ehr.graph.mutations` topic:

```python
from app.models import GraphMutation, Relationship

mutation = GraphMutation(
    mutationType="MERGE",
    nodeType="Patient",
    nodeId="P12345",
    timestamp=1699564800000,
    nodeProperties={
        "patientId": "P12345",
        "lastUpdated": 1699564800000,
        "demographicsVersion": 3
    },
    relationships=[
        Relationship(
            relationType="HAS_EVENT",
            targetNodeType="ClinicalEvent",
            targetNodeId="evt_12345",
            relationshipProperties={
                "timestamp": 1699564800000,
                "eventType": "VITAL_SIGNS"
            }
        )
    ]
)

# Get Kafka key
key = mutation.get_kafka_key()  # "P12345"
```

## Base Kafka Consumer

Create a new projector by extending `KafkaConsumerBase`:

```python
from app.kafka_consumer_base import KafkaConsumerBase
from app.models import EnrichedClinicalEvent
from typing import List, Any
import structlog

logger = structlog.get_logger(__name__)


class PostgreSQLProjector(KafkaConsumerBase):
    """Projects enriched clinical events to PostgreSQL"""

    def __init__(self, kafka_config: dict, postgres_config: dict):
        super().__init__(
            kafka_config=kafka_config,
            topics=["prod.ehr.events.enriched"],
            batch_size=100,
            batch_timeout_seconds=5.0,
            dlq_topic="prod.ehr.dlq.postgresql",
        )
        self.postgres_config = postgres_config

    def get_projector_name(self) -> str:
        return "postgresql-projector"

    def process_batch(self, messages: List[Any]) -> None:
        """Write batch to PostgreSQL"""
        # Parse messages as EnrichedClinicalEvent
        events = [EnrichedClinicalEvent(**msg) for msg in messages]

        # Write to PostgreSQL
        conn = psycopg2.connect(**self.postgres_config)
        try:
            with conn.cursor() as cur:
                self._insert_events(cur, events)
            conn.commit()

            logger.info(
                "Batch written to PostgreSQL",
                batch_size=len(events)
            )
        except Exception as e:
            conn.rollback()
            logger.error("Batch write failed", error=str(e))
            raise
        finally:
            conn.close()

    def _insert_events(self, cur, events: List[EnrichedClinicalEvent]):
        """Insert events to PostgreSQL tables"""
        # Your PostgreSQL insert logic here
        pass


# Usage
if __name__ == "__main__":
    kafka_config = {
        "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
        "group.id": "module8-postgresql-projector",
        "auto.offset.reset": "earliest",
        "enable.auto.commit": False,
        "security.protocol": "SASL_SSL",
        "sasl.mechanism": "PLAIN",
        "sasl.username": os.getenv("KAFKA_API_KEY"),
        "sasl.password": os.getenv("KAFKA_API_SECRET"),
    }

    postgres_config = {
        "host": "localhost",
        "port": 5432,
        "database": "cardiofit",
        "user": "cardiofit_user",
        "password": os.getenv("POSTGRES_PASSWORD"),
    }

    projector = PostgreSQLProjector(kafka_config, postgres_config)
    projector.start()
```

## Batch Processor

The base consumer uses `BatchProcessor` internally, but you can use it standalone:

```python
from app.batch_processor import BatchProcessor

def flush_callback(batch):
    print(f"Flushing {len(batch)} items")
    # Process batch

processor = BatchProcessor(
    batch_size=100,
    batch_timeout_seconds=5.0,
    flush_callback=flush_callback,
)

# Add items
processor.add(item1)
processor.add(item2)

# Manual flush
processor.flush()

# Get stats
print(f"Current batch size: {processor.get_current_batch_size()}")
print(f"Batch age: {processor.get_batch_age():.2f}s")
```

## Metrics

All projectors automatically track Prometheus metrics:

```python
from app.metrics import ProjectorMetrics

metrics = ProjectorMetrics(projector_name="my-projector")

# Track messages
metrics.messages_consumed.inc()
metrics.messages_processed.inc(batch_size)
metrics.messages_failed.inc()

# Track batch metrics
metrics.batch_size.observe(100)
metrics.batch_flush_duration.observe(1.5)

# Track consumer lag
metrics.consumer_lag.set(150)
```

Available metrics:
- `projector_messages_consumed_total{projector="..."}`
- `projector_messages_processed_total{projector="..."}`
- `projector_messages_failed_total{projector="..."}`
- `projector_batch_size{projector="..."}`
- `projector_batch_flush_duration_seconds{projector="..."}`
- `projector_consumer_lag{projector="..."}`

## Configuration

### Kafka Configuration

```python
kafka_config = {
    # Connection
    "bootstrap.servers": "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092",
    "group.id": "module8-{projector-name}",

    # Consumer behavior
    "auto.offset.reset": "earliest",  # or "latest"
    "enable.auto.commit": False,
    "max.poll.records": 500,
    "max.poll.interval.ms": 300000,

    # Security (Confluent Cloud)
    "security.protocol": "SASL_SSL",
    "sasl.mechanism": "PLAIN",
    "sasl.username": os.getenv("KAFKA_API_KEY"),
    "sasl.password": os.getenv("KAFKA_API_SECRET"),
}
```

### Batch Configuration

```python
# Fast, small batches (for low-latency projectors)
batch_size = 20
batch_timeout_seconds = 2.0

# Large batches (for high-throughput projectors)
batch_size = 500
batch_timeout_seconds = 10.0
```

## Error Handling

The base consumer provides automatic error handling:

1. **Message-level errors**: Single messages are sent to DLQ
2. **Batch-level errors**: Entire batch is sent to DLQ and exception is raised
3. **Graceful shutdown**: SIGINT/SIGTERM signals trigger graceful shutdown

Configure DLQ topic:

```python
projector = MyProjector(
    kafka_config=kafka_config,
    dlq_topic="prod.ehr.dlq.my-projector",
)
```

## Testing

```python
import pytest
from app.models import EnrichedClinicalEvent

def test_enriched_clinical_event():
    event = EnrichedClinicalEvent(
        id="test_123",
        timestamp=1699564800000,
        eventType="VITAL_SIGNS",
        patientId="P123",
        rawData={"heart_rate": 75},
    )

    assert event.id == "test_123"
    assert event.patient_id == "P123"
    assert event.raw_data.heart_rate == 75
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Module 8 Projector Service             │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────────────────────────────────┐  │
│  │       Your Projector Implementation         │  │
│  │  (extends KafkaConsumerBase)                │  │
│  │                                             │  │
│  │  - process_batch()                          │  │
│  │  - get_projector_name()                     │  │
│  └─────────────────┬───────────────────────────┘  │
│                    │                               │
│  ┌─────────────────▼───────────────────────────┐  │
│  │        KafkaConsumerBase (shared)           │  │
│  │                                             │  │
│  │  - Kafka consumer setup                     │  │
│  │  - Batch processing logic                   │  │
│  │  - Offset commit handling                   │  │
│  │  - DLQ support                              │  │
│  │  - Graceful shutdown                        │  │
│  └─────────────────┬───────────────────────────┘  │
│                    │                               │
│  ┌─────────────────▼───────────────────────────┐  │
│  │        BatchProcessor (shared)              │  │
│  │                                             │  │
│  │  - Size-based flushing                      │  │
│  │  - Time-based flushing                      │  │
│  │  - Thread-safe operations                   │  │
│  └─────────────────┬───────────────────────────┘  │
│                    │                               │
│  ┌─────────────────▼───────────────────────────┐  │
│  │        ProjectorMetrics (shared)            │  │
│  │                                             │  │
│  │  - Prometheus metrics                       │  │
│  │  - Consumer lag tracking                    │  │
│  └─────────────────────────────────────────────┘  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## License

Copyright © 2024 CardioFit Platform
