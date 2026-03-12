"""
PostgreSQL Projector Service
Consumes enriched clinical events from Kafka and writes to PostgreSQL
"""
import sys
from pathlib import Path
from typing import List, Any, Dict, Optional
from datetime import datetime
import json

import psycopg2
from psycopg2.extras import execute_batch
import structlog

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from module8_shared.models import EnrichedClinicalEvent

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
        self.schema = "module8_projections"
        self.last_processed_time: Optional[datetime] = None

        # Test PostgreSQL connection
        self._test_connection()

        logger.info(
            "PostgreSQL projector initialized",
            postgres_host=postgres_config["host"],
            postgres_db=postgres_config["database"],
            schema=self.schema,
        )

    def _test_connection(self) -> None:
        """Test PostgreSQL connection and ensure schema exists"""
        try:
            conn = psycopg2.connect(**self.postgres_config)
            try:
                with conn.cursor() as cur:
                    # Check if schema exists
                    cur.execute(
                        "SELECT schema_name FROM information_schema.schemata WHERE schema_name = %s",
                        (self.schema,)
                    )
                    result = cur.fetchone()

                    if not result:
                        logger.warning(
                            "Schema does not exist, creating",
                            schema=self.schema
                        )
                        # Read and execute init.sql
                        init_sql_path = Path(__file__).parent.parent.parent / "schema" / "init.sql"
                        if init_sql_path.exists():
                            with open(init_sql_path, 'r') as f:
                                cur.execute(f.read())
                            conn.commit()
                            logger.info("Schema created successfully", schema=self.schema)
                        else:
                            logger.error("init.sql not found", path=str(init_sql_path))
                            raise FileNotFoundError(f"Schema initialization file not found: {init_sql_path}")

                    logger.info("PostgreSQL connection successful")
            finally:
                conn.close()
        except Exception as e:
            logger.error("PostgreSQL connection failed", error=str(e))
            raise

    def get_projector_name(self) -> str:
        return "postgresql-projector"

    def process_batch(self, messages: List[Any]) -> None:
        """Write batch to PostgreSQL with transaction support"""
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
            logger.warning("No valid events to process")
            return

        # Filter out events with null patient_id (e.g., PATTERN_EVENT types)
        events_before_filter = len(events)
        events = [e for e in events if e.patient_id is not None]
        filtered_count = events_before_filter - len(events)

        if filtered_count > 0:
            logger.info(
                "Filtered events with null patient_id",
                filtered_count=filtered_count,
                remaining_events=len(events)
            )

        if not events:
            logger.warning("No valid events to process after filtering null patient_id")
            return

        # Write to PostgreSQL with transaction
        conn = psycopg2.connect(**self.postgres_config)
        try:
            with conn.cursor() as cur:
                # Set schema
                cur.execute(f"SET search_path TO {self.schema}, public")

                # Insert to all tables
                self._insert_enriched_events(cur, events)
                self._insert_patient_vitals(cur, events)
                self._insert_clinical_scores(cur, events)
                self._insert_event_metadata(cur, events)

            conn.commit()
            self.last_processed_time = datetime.utcnow()

            logger.info(
                "Batch written to PostgreSQL",
                batch_size=len(events),
                total_messages=len(messages),
                vitals_count=len([e for e in events if e.event_type == 'VITAL_SIGNS']),
                scored_count=len([e for e in events if e.enrichments]),
            )

        except Exception as e:
            conn.rollback()
            logger.error("PostgreSQL batch write failed", error=str(e), exc_info=True)
            raise
        finally:
            conn.close()

    def _insert_enriched_events(self, cur, events: List[EnrichedClinicalEvent]) -> None:
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
                datetime.fromtimestamp(event.timestamp / 1000.0) if isinstance(event.timestamp, int) else event.timestamp,  # Convert Unix ms to datetime
                event.event_type,
                json.dumps(event.dict())  # Store full event as JSON
            )
            for event in events
        ]

        execute_batch(cur, sql, data)
        logger.debug("Inserted enriched_events", count=len(data))

    def _insert_patient_vitals(self, cur, events: List[EnrichedClinicalEvent]) -> None:
        """Insert to patient_vitals table (normalized)"""
        vitals_events = [e for e in events if e.event_type == 'VITAL_SIGNS' and e.raw_data]

        if not vitals_events:
            return

        sql = """
            INSERT INTO patient_vitals
            (event_id, patient_id, timestamp, heart_rate, bp_systolic,
             bp_diastolic, spo2, temperature_celsius)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (event_id) DO NOTHING
        """

        data = []
        for event in vitals_events:
            try:
                data.append((
                    event.id,
                    event.patient_id,
                    datetime.fromtimestamp(event.timestamp / 1000.0) if isinstance(event.timestamp, int) else event.timestamp,
                    event.raw_data.heart_rate if event.raw_data else None,
                    event.raw_data.blood_pressure_systolic if event.raw_data else None,
                    event.raw_data.blood_pressure_diastolic if event.raw_data else None,
                    event.raw_data.spo2 if event.raw_data else None,
                    event.raw_data.temperature_celsius if event.raw_data else None
                ))
            except AttributeError as e:
                logger.warning("Missing vital data", event_id=event.id, error=str(e))
                continue

        if data:
            execute_batch(cur, sql, data)
            logger.debug("Inserted patient_vitals", count=len(data))

    def _insert_clinical_scores(self, cur, events: List[EnrichedClinicalEvent]) -> None:
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

        data = []
        for event in scored_events:
            try:
                data.append((
                    event.id,
                    event.patient_id,
                    datetime.fromtimestamp(event.timestamp / 1000.0) if isinstance(event.timestamp, int) else event.timestamp,
                    event.enrichments.NEWS2Score if event.enrichments else None,
                    event.enrichments.qSOFAScore if event.enrichments else None,
                    event.enrichments.riskLevel if event.enrichments else None,
                    event.ml_predictions.sepsis_risk_24h if event.ml_predictions else None,
                    event.ml_predictions.cardiac_event_risk_7d if event.ml_predictions else None,
                    event.ml_predictions.readmission_risk_30d if event.ml_predictions else None,
                ))
            except AttributeError as e:
                logger.warning("Missing enrichment data", event_id=event.id, error=str(e))
                continue

        if data:
            execute_batch(cur, sql, data)
            logger.debug("Inserted clinical_scores", count=len(data))

    def _insert_event_metadata(self, cur, events: List[EnrichedClinicalEvent]) -> None:
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
                datetime.fromtimestamp(event.timestamp / 1000.0) if isinstance(event.timestamp, int) else event.timestamp,
                event.event_type
            )
            for event in events
        ]

        execute_batch(cur, sql, data)
        logger.debug("Inserted event_metadata", count=len(data))

    def is_running(self) -> bool:
        """Check if consumer is running"""
        return hasattr(self, 'consumer') and self.consumer is not None

    def get_metrics(self) -> Dict[str, Any]:
        """Get consumer metrics"""
        return {
            "messages_consumed": getattr(self, 'message_count', 0),
            "messages_processed": getattr(self, 'batch_count', 0) * getattr(self, 'batch_size', 100),
            "messages_failed": 0,  # Track failures if needed
            "batches_processed": getattr(self, 'batch_count', 0),
            "consumer_lag": 0,  # Can be enhanced with actual lag calculation
        }
