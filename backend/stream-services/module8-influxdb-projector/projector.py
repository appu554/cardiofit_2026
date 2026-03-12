"""InfluxDB Projector - Consumes enriched events and writes to InfluxDB."""
import logging
from typing import List, Dict, Any
from datetime import datetime
from influxdb_client import Point

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from module8_shared.models import EnrichedClinicalEvent
from config import config
from influxdb_manager import influxdb_manager

logger = logging.getLogger(__name__)


class InfluxDBProjector(KafkaConsumerBase):
    """Projects enriched events to InfluxDB time-series database."""

    def __init__(self):
        """Initialize InfluxDB projector."""
        # Determine security protocol based on bootstrap servers
        is_local_kafka = 'localhost' in config.KAFKA_BOOTSTRAP_SERVERS or '127.0.0.1' in config.KAFKA_BOOTSTRAP_SERVERS

        kafka_config = {
            'bootstrap.servers': config.KAFKA_BOOTSTRAP_SERVERS,
            'group.id': config.KAFKA_CONSUMER_GROUP,
            'auto.offset.reset': config.KAFKA_AUTO_OFFSET_RESET,
            'enable.auto.commit': True,
        }

        # Add SASL config only for cloud Kafka
        if not is_local_kafka:
            kafka_config.update({
                'security.protocol': 'SASL_SSL',
                'sasl.mechanism': 'PLAIN',
                'sasl.username': config.KAFKA_SASL_USERNAME,
                'sasl.password': config.KAFKA_SASL_PASSWORD,
            })
        else:
            kafka_config['security.protocol'] = 'PLAINTEXT'

        super().__init__(
            kafka_config=kafka_config,
            topics=[config.KAFKA_TOPIC],
            batch_size=config.INFLUXDB_BATCH_SIZE,
            batch_timeout_seconds=config.INFLUXDB_FLUSH_INTERVAL / 1000.0  # Convert ms to seconds
        )

        self.stats = {
            "total_events_processed": 0,
            "vitals_written": 0,
            "heart_rate_count": 0,
            "blood_pressure_count": 0,
            "spo2_count": 0,
            "temperature_count": 0,
            "non_vital_skipped": 0,
            "errors": 0
        }

    def get_projector_name(self) -> str:
        """Return unique projector identifier."""
        return "influxdb-projector"

    def process_batch(self, events: List[EnrichedClinicalEvent]) -> None:
        """Process batch of enriched events and write to InfluxDB.

        Args:
            events: List of enriched events to process
        """
        vital_points = []

        for event in events:
            try:
                # Only process vital signs events
                if event.event_type.upper() not in ["VITAL_SIGNS", "VITALS"]:
                    self.stats["non_vital_skipped"] += 1
                    continue

                # Extract vital signs from raw data
                points = self._extract_vital_points(event)
                vital_points.extend(points)

                self.stats["total_events_processed"] += 1

            except Exception as e:
                logger.error(f"Error processing event {event.id}: {e}")
                self.stats["errors"] += 1

        # Batch write all points to InfluxDB
        if vital_points:
            try:
                influxdb_manager.write_vital_signs(vital_points)
                self.stats["vitals_written"] += len(vital_points)
                logger.info(f"Wrote {len(vital_points)} vital points to InfluxDB")
            except Exception as e:
                logger.error(f"Failed to write batch to InfluxDB: {e}")
                self.stats["errors"] += 1

    def _extract_vital_points(self, event: EnrichedClinicalEvent) -> List[Point]:
        """Extract vital sign data points from enriched event.

        Args:
            event: Enriched event containing vital signs

        Returns:
            List of InfluxDB Point objects
        """
        points = []

        # Get raw data (might be None)
        if not event.raw_data:
            return points

        raw_data = event.raw_data

        # Extract metadata
        patient_id = event.patient_id or "UNKNOWN"
        device_id = event.device_id or "UNKNOWN"
        department_id = event.department_id or "UNKNOWN"
        timestamp = datetime.fromtimestamp(event.timestamp / 1000.0)  # Convert milliseconds to datetime

        # Extract heart rate
        if raw_data.heart_rate and raw_data.heart_rate > 0:
            point = influxdb_manager.create_vital_point(
                measurement="heart_rate",
                patient_id=patient_id,
                device_id=device_id,
                department_id=department_id,
                fields={"value": float(raw_data.heart_rate)},
                timestamp=timestamp
            )
            points.append(point)
            self.stats["heart_rate_count"] += 1

        # Extract blood pressure
        if raw_data.blood_pressure_systolic and raw_data.blood_pressure_diastolic:
            point = influxdb_manager.create_vital_point(
                measurement="blood_pressure",
                patient_id=patient_id,
                device_id=device_id,
                department_id=department_id,
                fields={
                    "systolic": float(raw_data.blood_pressure_systolic),
                    "diastolic": float(raw_data.blood_pressure_diastolic)
                },
                timestamp=timestamp
            )
            points.append(point)
            self.stats["blood_pressure_count"] += 1

        # Extract SpO2
        if raw_data.spo2 and 0 <= raw_data.spo2 <= 100:
            point = influxdb_manager.create_vital_point(
                measurement="spo2",
                patient_id=patient_id,
                device_id=device_id,
                department_id=department_id,
                fields={"value": float(raw_data.spo2)},
                timestamp=timestamp
            )
            points.append(point)
            self.stats["spo2_count"] += 1

        # Extract temperature
        if raw_data.temperature_celsius and raw_data.temperature_celsius > 0:
            point = influxdb_manager.create_vital_point(
                measurement="temperature",
                patient_id=patient_id,
                device_id=device_id,
                department_id=department_id,
                fields={"value": float(raw_data.temperature_celsius)},
                timestamp=timestamp
            )
            points.append(point)
            self.stats["temperature_count"] += 1

        return points

    def get_stats(self) -> Dict[str, Any]:
        """Get projector statistics.

        Returns:
            Dictionary of statistics
        """
        return self.stats.copy()


# Global instance
projector = InfluxDBProjector()
