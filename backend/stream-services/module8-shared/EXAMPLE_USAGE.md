# Module 8 Shared Components - Usage Example

## Complete Example: PostgreSQL Projector

This example shows how to create a complete projector service using the shared components.

### 1. Project Structure

```
module8-postgresql-projector/
├── app/
│   ├── __init__.py
│   ├── config.py
│   ├── projector.py
│   └── main.py
├── Dockerfile
├── requirements.txt
└── README.md
```

### 2. Configuration (`app/config.py`)

```python
import os
from typing import Dict, Any

# Kafka Configuration
KAFKA_CONFIG: Dict[str, Any] = {
    "bootstrap.servers": os.getenv(
        "KAFKA_BOOTSTRAP_SERVERS",
        "pkc-xxxxx.us-east-1.aws.confluent.cloud:9092"
    ),
    "group.id": "module8-postgresql-projector",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 500,
    "max.poll.interval.ms": 300000,

    # Security (Confluent Cloud)
    "security.protocol": "SASL_SSL",
    "sasl.mechanism": "PLAIN",
    "sasl.username": os.getenv("KAFKA_API_KEY"),
    "sasl.password": os.getenv("KAFKA_API_SECRET"),
}

# Topics
TOPICS = ["prod.ehr.events.enriched"]

# Batch Configuration
BATCH_SIZE = 100
BATCH_TIMEOUT_SECONDS = 5.0

# DLQ Configuration
DLQ_TOPIC = "prod.ehr.dlq.postgresql"

# PostgreSQL Configuration
POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "localhost"),
    "port": int(os.getenv("POSTGRES_PORT", "5432")),
    "database": os.getenv("POSTGRES_DB", "cardiofit"),
    "user": os.getenv("POSTGRES_USER", "cardiofit_user"),
    "password": os.getenv("POSTGRES_PASSWORD"),
}
```

### 3. Projector Implementation (`app/projector.py`)

```python
import sys
from pathlib import Path

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from typing import List, Any
import psycopg2
from psycopg2.extras import execute_batch
import structlog

from app.kafka_consumer_base import KafkaConsumerBase
from app.models import EnrichedClinicalEvent

logger = structlog.get_logger(__name__)


class PostgreSQLProjector(KafkaConsumerBase):
    """
    Projects enriched clinical events to PostgreSQL

    Database Schema:
    - enriched_events: Raw event storage with JSONB
    - patient_vitals: Normalized vital signs
    - clinical_scores: Risk scores and predictions
    - event_metadata: Searchable event attributes
    """

    def __init__(self, kafka_config: dict, postgres_config: dict):
        super().__init__(
            kafka_config=kafka_config,
            topics=["prod.ehr.events.enriched"],
            batch_size=100,
            batch_timeout_seconds=5.0,
            dlq_topic="prod.ehr.dlq.postgresql",
        )
        self.postgres_config = postgres_config

        logger.info(
            "PostgreSQL projector initialized",
            postgres_host=postgres_config["host"],
            postgres_db=postgres_config["database"],
        )

    def get_projector_name(self) -> str:
        return "postgresql-projector"

    def process_batch(self, messages: List[Any]) -> None:
        """Write batch to PostgreSQL"""
        if not messages:
            return

        # Parse messages as EnrichedClinicalEvent
        events = []
        for msg in messages:
            try:
                event = EnrichedClinicalEvent(**msg)
                events.append(event)
            except Exception as e:
                logger.error("Failed to parse event", error=str(e), message=msg)
                continue

        if not events:
            return

        # Write to PostgreSQL
        conn = psycopg2.connect(**self.postgres_config)
        try:
            with conn.cursor() as cur:
                # Insert to all tables
                self._insert_enriched_events(cur, events)
                self._insert_patient_vitals(cur, events)
                self._insert_clinical_scores(cur, events)
                self._insert_event_metadata(cur, events)

            conn.commit()

            logger.info(
                "Batch written to PostgreSQL",
                batch_size=len(events),
                total_messages=len(messages),
            )

        except Exception as e:
            conn.rollback()
            logger.error("PostgreSQL batch write failed", error=str(e))
            raise
        finally:
            conn.close()

    def _insert_enriched_events(self, cur, events: List[EnrichedClinicalEvent]):
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
                event.id,
                event.patient_id,
                event.timestamp,
                event.event_type,
                event.json()  # Store full event as JSON
            )
            for event in events
        ]

        execute_batch(cur, sql, data)

    def _insert_patient_vitals(self, cur, events: List[EnrichedClinicalEvent]):
        """Insert to patient_vitals table (normalized)"""
        vitals_events = [e for e in events if e.event_type == 'VITAL_SIGNS']

        if not vitals_events:
            return

        sql = """
            INSERT INTO patient_vitals
            (event_id, patient_id, timestamp, heart_rate, bp_systolic,
             bp_diastolic, spo2, temperature_celsius)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (event_id) DO NOTHING
        """

        data = [
            (
                event.id,
                event.patient_id,
                event.timestamp,
                event.raw_data.heart_rate,
                event.raw_data.blood_pressure_systolic,
                event.raw_data.blood_pressure_diastolic,
                event.raw_data.spo2,
                event.raw_data.temperature_celsius
            )
            for event in vitals_events
        ]

        execute_batch(cur, sql, data)

    def _insert_clinical_scores(self, cur, events: List[EnrichedClinicalEvent]):
        """Insert to clinical_scores table"""
        scored_events = [e for e in events if e.enrichments]

        if not scored_events:
            return

        sql = """
            INSERT INTO clinical_scores
            (event_id, patient_id, timestamp, news2_score, qsofa_score,
             risk_level, sepsis_risk_24h, cardiac_risk_7d, readmission_risk_30d)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (event_id) DO NOTHING
        """

        data = [
            (
                event.id,
                event.patient_id,
                event.timestamp,
                event.enrichments.NEWS2Score,
                event.enrichments.qSOFAScore,
                event.enrichments.riskLevel,
                event.ml_predictions.sepsis_risk_24h if event.ml_predictions else None,
                event.ml_predictions.cardiac_event_risk_7d if event.ml_predictions else None,
                event.ml_predictions.readmission_risk_30d if event.ml_predictions else None,
            )
            for event in scored_events
        ]

        execute_batch(cur, sql, data)

    def _insert_event_metadata(self, cur, events: List[EnrichedClinicalEvent]):
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
                event.id,
                event.patient_id,
                event.encounter_id,
                event.department_id,
                event.device_id,
                event.timestamp,
                event.event_type
            )
            for event in events
        ]

        execute_batch(cur, sql, data)
```

### 4. Main Entry Point (`app/main.py`)

```python
import sys
from pathlib import Path
import os

# Configure logging
import structlog
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from app.projector import PostgreSQLProjector
from app.config import KAFKA_CONFIG, POSTGRES_CONFIG

logger = structlog.get_logger(__name__)


def main():
    """Start PostgreSQL projector"""
    logger.info("Starting PostgreSQL projector service")

    # Validate configuration
    if not os.getenv("KAFKA_API_KEY"):
        logger.error("KAFKA_API_KEY environment variable not set")
        sys.exit(1)

    if not os.getenv("POSTGRES_PASSWORD"):
        logger.error("POSTGRES_PASSWORD environment variable not set")
        sys.exit(1)

    # Create and start projector
    projector = PostgreSQLProjector(
        kafka_config=KAFKA_CONFIG,
        postgres_config=POSTGRES_CONFIG,
    )

    try:
        projector.start()
    except KeyboardInterrupt:
        logger.info("Received interrupt signal, shutting down")
        projector.shutdown()
    except Exception as e:
        logger.error("Fatal error", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
```

### 5. Requirements (`requirements.txt`)

```
# Parent shared module
-e ../module8-shared

# PostgreSQL
psycopg2-binary==2.9.9

# Logging
structlog==24.1.0
```

### 6. Dockerfile

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY app/ ./app/

# Copy shared module (if not using pip install -e)
COPY ../module8-shared /app/module8-shared

# Expose Prometheus metrics port
EXPOSE 9090

# Run projector
CMD ["python", "-m", "app.main"]
```

### 7. Environment Variables (.env)

```bash
# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
KAFKA_API_KEY=your-api-key
KAFKA_API_SECRET=your-api-secret

# PostgreSQL Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=cardiofit
POSTGRES_USER=cardiofit_user
POSTGRES_PASSWORD=your-password
```

### 8. Running the Projector

```bash
# Install dependencies
pip install -r requirements.txt

# Run projector
python -m app.main

# Or with Docker
docker build -t postgresql-projector .
docker run --env-file .env postgresql-projector
```

### 9. Monitoring

Access Prometheus metrics at `http://localhost:9090/metrics`:

```
# Metrics available
projector_messages_consumed_total{projector="postgresql-projector"}
projector_messages_processed_total{projector="postgresql-projector"}
projector_messages_failed_total{projector="postgresql-projector"}
projector_batch_size{projector="postgresql-projector"}
projector_batch_flush_duration_seconds{projector="postgresql-projector"}
projector_consumer_lag{projector="postgresql-projector"}
```

## Key Benefits

1. **Minimal Boilerplate**: Base class handles all Kafka consumer logic
2. **Type Safety**: Pydantic models ensure data validation
3. **Automatic Metrics**: Prometheus metrics work out of the box
4. **Error Handling**: DLQ support and graceful shutdown included
5. **Batch Processing**: Efficient batching with configurable size/timeout
6. **Production Ready**: Security, monitoring, and error handling built-in

## Creating Other Projectors

To create MongoDB, Elasticsearch, or other projectors:

1. Copy the structure above
2. Change the `process_batch()` method to write to your target storage
3. Update configuration for your storage system
4. Adjust batch sizes based on storage performance

Example for MongoDB:

```python
def process_batch(self, messages: List[Any]) -> None:
    """Write batch to MongoDB"""
    events = [EnrichedClinicalEvent(**msg) for msg in messages]

    client = MongoClient(self.mongo_uri)
    db = client[self.db_name]

    # Insert clinical documents
    operations = [
        UpdateOne(
            {"_id": event.id},
            {"$set": event.dict()},
            upsert=True
        )
        for event in events
    ]

    result = db.clinical_documents.bulk_write(operations)

    logger.info(
        "Batch written to MongoDB",
        upserted=result.upserted_count,
        modified=result.modified_count
    )
```
