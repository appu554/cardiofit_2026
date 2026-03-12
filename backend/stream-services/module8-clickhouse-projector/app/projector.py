"""
ClickHouse Projector for OLAP Analytics
Consumes from prod.ehr.events.enriched and writes to ClickHouse columnar storage
"""

import json
import logging
from datetime import datetime
from typing import List, Dict, Any
from clickhouse_driver import Client
from module8_shared.kafka_consumer_base import KafkaConsumerBase

logger = logging.getLogger(__name__)


class ClickHouseProjector(KafkaConsumerBase):
    """
    ClickHouse projector for OLAP analytics on enriched clinical events.

    Features:
    - Columnar storage for fast aggregations
    - Partitioned by month for efficient time-range queries
    - TTL for automatic data retention
    - Sub-second queries on billions of rows
    """

    def get_projector_name(self) -> str:
        """Return the projector name for logging and identification."""
        return "clickhouse-projector"

    def __init__(self, config: Dict[str, Any]):
        """Initialize ClickHouse projector with connection and settings."""
        # Extract Kafka configuration
        kafka_config_raw = config.get('kafka', {})
        topics = [kafka_config_raw.get('topic', 'prod.ehr.events.enriched')]
        batch_config = config.get('batch', {})

        # Filter out 'topic' from kafka_config as it's passed separately to KafkaConsumer
        kafka_config = {k: v for k, v in kafka_config_raw.items() if k != 'topic'}

        # Initialize base class with Kafka configuration
        super().__init__(
            kafka_config=kafka_config,
            topics=topics,
            batch_size=batch_config.get('size', 500),
            batch_timeout_seconds=batch_config.get('timeout', 30)
        )

        # ClickHouse connection settings
        ch_config = config.get('clickhouse', {})
        self.ch_host = ch_config.get('host', 'clickhouse')
        self.ch_port = ch_config.get('port', 9000)  # Native protocol port
        self.ch_database = ch_config.get('database', 'module8_analytics')
        self.ch_user = ch_config.get('user', 'module8_user')
        self.ch_password = ch_config.get('password', 'module8_password')

        # Initialize ClickHouse client
        self.client = None
        self._connect_clickhouse()

        # Batch buffers for efficient inserts
        self.clinical_events_buffer: List[tuple] = []
        self.ml_predictions_buffer: List[tuple] = []
        self.alerts_buffer: List[tuple] = []

        logger.info(
            f"ClickHouseProjector initialized: {self.ch_host}:{self.ch_port}/{self.ch_database}"
        )

    def _connect_clickhouse(self):
        """Establish connection to ClickHouse."""
        try:
            self.client = Client(
                host=self.ch_host,
                port=self.ch_port,
                database=self.ch_database,
                user=self.ch_user,
                password=self.ch_password,
                settings={
                    'use_numpy': False,
                    'max_block_size': 10000,
                }
            )

            # Test connection
            result = self.client.execute('SELECT 1')
            logger.info(f"ClickHouse connection established: {result}")

            # Ensure database exists
            self.client.execute(f'CREATE DATABASE IF NOT EXISTS {self.ch_database}')

        except Exception as e:
            logger.error(f"Failed to connect to ClickHouse: {e}")
            raise

    def process_batch(self, messages: List[Dict[str, Any]]) -> None:
        """
        Process batch of enriched events and insert into ClickHouse tables.

        Strategy:
        1. Parse and categorize events
        2. Extract data for each fact table
        3. Batch insert to ClickHouse (500 rows at a time for analytics)
        """
        for message in messages:
            try:
                self._process_event(message)
            except Exception as e:
                logger.error(f"Error processing event {message.get('eventId')}: {e}")
                self.metrics.messages_failed.inc()

        # Flush buffers to ClickHouse
        self._flush_buffers()

        # Note: messages_processed metric is updated by base class after successful batch flush

    def _process_event(self, event: Dict[str, Any]):
        """Process single enriched event and add to appropriate buffers."""
        # Debug: Log first event structure
        if not hasattr(self, '_logged_sample'):
            logger.info(f"Sample event structure: {list(event.keys())[:10]}")
            self._logged_sample = True

        event_id = event.get('eventId')
        if not event_id:
            logger.warning(f"Event missing eventId. Keys present: {list(event.keys())[:5]}")
            return

        patient_id = event.get('patientId') or 'UNKNOWN'
        timestamp = self._parse_timestamp(event.get('timestamp'))
        event_type = event.get('eventType') or 'UNKNOWN'
        department_id = event.get('departmentId') or 'UNKNOWN'

        # Extract vitals with null safety
        vitals = event.get('vitalSigns') or {}
        if vitals is None:
            vitals = {}

        # Extract clinical scores with null safety
        enrichment = event.get('enrichment') or {}
        if enrichment is None:
            enrichment = {}
        clinical_scores = enrichment.get('clinicalScores') or {}
        if clinical_scores is None:
            clinical_scores = {}
        risk_level = enrichment.get('riskLevel') or 'UNKNOWN'

        # 1. Clinical Events Fact
        clinical_row = (
            event_id,
            patient_id,
            timestamp,
            event_type,
            department_id,
            vitals.get('heartRate'),
            vitals.get('bloodPressure', {}).get('systolic'),
            vitals.get('bloodPressure', {}).get('diastolic'),
            vitals.get('spO2'),
            vitals.get('temperature'),
            clinical_scores.get('news2'),
            clinical_scores.get('qsofa'),
            risk_level,
            json.dumps(event)  # Store full event as JSON string
        )
        self.clinical_events_buffer.append(clinical_row)

        # 2. ML Predictions Fact (if predictions present)
        ml_predictions = enrichment.get('mlPredictions') or {}
        if ml_predictions is None:
            ml_predictions = {}
        if ml_predictions:
            ml_row = (
                event_id,
                patient_id,
                timestamp,
                ml_predictions.get('sepsisRisk24h'),
                ml_predictions.get('cardiacRisk7d'),
                ml_predictions.get('readmissionRisk30d'),
                json.dumps(ml_predictions)
            )
            self.ml_predictions_buffer.append(ml_row)

        # 3. Alerts Fact (if high risk)
        if risk_level in ['HIGH', 'CRITICAL']:
            alert_row = (
                event_id,
                patient_id,
                timestamp,
                f"RISK_{risk_level}",
                risk_level,
                department_id,
                None  # response_time_seconds - would be populated by alert response system
            )
            self.alerts_buffer.append(alert_row)

    def _flush_buffers(self):
        """Flush all buffers to ClickHouse with batch inserts."""
        try:
            # Insert clinical events
            if self.clinical_events_buffer:
                self.client.execute(
                    """
                    INSERT INTO clinical_events_fact
                    (event_id, patient_id, timestamp, event_type, department_id,
                     heart_rate, bp_systolic, bp_diastolic, spo2, temperature,
                     news2_score, qsofa_score, risk_level, event_data)
                    VALUES
                    """,
                    self.clinical_events_buffer
                )
                logger.info(f"Inserted {len(self.clinical_events_buffer)} clinical events")
                self.clinical_events_buffer.clear()

            # Insert ML predictions
            if self.ml_predictions_buffer:
                self.client.execute(
                    """
                    INSERT INTO ml_predictions_fact
                    (event_id, patient_id, timestamp, sepsis_risk_24h,
                     cardiac_risk_7d, readmission_risk_30d, prediction_data)
                    VALUES
                    """,
                    self.ml_predictions_buffer
                )
                logger.info(f"Inserted {len(self.ml_predictions_buffer)} ML predictions")
                self.ml_predictions_buffer.clear()

            # Insert alerts
            if self.alerts_buffer:
                self.client.execute(
                    """
                    INSERT INTO alerts_fact
                    (event_id, patient_id, timestamp, alert_type,
                     severity, department_id, response_time_seconds)
                    VALUES
                    """,
                    self.alerts_buffer
                )
                logger.info(f"Inserted {len(self.alerts_buffer)} alerts")
                self.alerts_buffer.clear()

        except Exception as e:
            logger.error(f"Error flushing buffers to ClickHouse: {e}")
            raise

    def _parse_timestamp(self, ts_str: str) -> datetime:
        """Parse ISO timestamp string to datetime (second precision for ClickHouse)."""
        try:
            dt = datetime.fromisoformat(ts_str.replace('Z', '+00:00'))
            # Convert to UTC and remove microseconds for ClickHouse DateTime compatibility
            return dt.replace(microsecond=0, tzinfo=None)
        except Exception as e:
            logger.warning(f"Failed to parse timestamp {ts_str}: {e}")
            return datetime.utcnow().replace(microsecond=0)

    def get_analytics_summary(self) -> Dict[str, Any]:
        """Get analytics summary from ClickHouse for monitoring."""
        try:
            # Total events count
            total_events = self.client.execute(
                'SELECT count() FROM clinical_events_fact'
            )[0][0]

            # High risk events count
            high_risk_count = self.client.execute(
                "SELECT count() FROM clinical_events_fact WHERE risk_level IN ('HIGH', 'CRITICAL')"
            )[0][0]

            # Predictions count
            predictions_count = self.client.execute(
                'SELECT count() FROM ml_predictions_fact'
            )[0][0]

            # Alerts count
            alerts_count = self.client.execute(
                'SELECT count() FROM alerts_fact'
            )[0][0]

            return {
                'total_events': total_events,
                'high_risk_events': high_risk_count,
                'ml_predictions': predictions_count,
                'alerts': alerts_count,
                'storage_info': self._get_storage_info()
            }
        except Exception as e:
            logger.error(f"Error getting analytics summary: {e}")
            return {}

    def _get_storage_info(self) -> Dict[str, Any]:
        """Get storage information for all tables."""
        try:
            result = self.client.execute(
                """
                SELECT
                    table,
                    formatReadableSize(sum(bytes)) as size,
                    sum(rows) as rows
                FROM system.parts
                WHERE database = %(database)s
                GROUP BY table
                """,
                {'database': self.ch_database}
            )

            return {row[0]: {'size': row[1], 'rows': row[2]} for row in result}
        except Exception as e:
            logger.error(f"Error getting storage info: {e}")
            return {}

    def close(self):
        """Flush remaining buffers and close ClickHouse connection."""
        try:
            self._flush_buffers()
            if self.client:
                self.client.disconnect()
            logger.info("ClickHouse connection closed")
        except Exception as e:
            logger.error(f"Error closing ClickHouse connection: {e}")

        super().close()
