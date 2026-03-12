"""MongoDB Projector service - consumes enriched events and projects to MongoDB."""

import json
import logging
from typing import List, Dict, Any, Optional, Union
from datetime import datetime
from pymongo import MongoClient, UpdateOne, ASCENDING, DESCENDING
from pymongo.collection import Collection
from pymongo.errors import BulkWriteError, PyMongoError

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from module8_shared.models import EnrichedClinicalEvent
from module8_shared.data_adapter import Module6DataAdapter
from ..config import get_settings
from ..models.schemas import (
    ClinicalDocument,
    TimelineEvent,
    MLExplanation,
    PredictionDetail,
)

logger = logging.getLogger(__name__)


class MongoDBProjector(KafkaConsumerBase):
    """MongoDB projector that consumes enriched events and writes to MongoDB."""

    def __init__(self):
        """Initialize MongoDB projector."""
        settings = get_settings()

        # Build Kafka configuration
        kafka_config = {
            "bootstrap.servers": settings.kafka_bootstrap_servers,
            "group.id": settings.kafka_group_id,
            "auto.offset.reset": settings.kafka_auto_offset_reset,
            "enable.auto.commit": settings.kafka_enable_auto_commit,
            "max.poll.records": settings.kafka_max_poll_records,
            "session.timeout.ms": settings.kafka_session_timeout_ms,
        }

        super().__init__(
            kafka_config=kafka_config,
            topics=[settings.kafka_topic],
            batch_size=settings.batch_size,
            batch_timeout_seconds=settings.batch_timeout_seconds,
            message_deserializer=self._deserialize_message,
        )

        self.settings = settings
        self.client: Optional[MongoClient] = None
        self.db = None
        self.clinical_docs: Optional[Collection] = None
        self.patient_timelines: Optional[Collection] = None
        self.ml_explanations: Optional[Collection] = None
        self.indexes_created = False
        self.adapter = Module6DataAdapter()

        # Statistics
        self.stats = {
            "documents_written": 0,
            "timelines_updated": 0,
            "explanations_written": 0,
            "errors": 0,
        }

    def get_projector_name(self) -> str:
        """Return unique projector identifier."""
        return "mongodb-projector"

    def _deserialize_message(self, message: Union[bytes, Dict[str, Any]]) -> EnrichedClinicalEvent:
        """
        Deserialize Kafka message to EnrichedClinicalEvent.

        Args:
            message: Raw message bytes or dict from Kafka

        Returns:
            EnrichedClinicalEvent object
        """
        try:
            # If message is bytes, decode to JSON first
            if isinstance(message, bytes):
                message = json.loads(message.decode('utf-8'))

            # Adapt Module 6 format to Module 8 format
            adapted_message = self.adapter.adapt_event(message)
            return EnrichedClinicalEvent(**adapted_message)
        except Exception as e:
            logger.error(f"Failed to deserialize message: {e}")
            raise

    def connect_mongodb(self):
        """Connect to MongoDB and initialize collections."""
        try:
            logger.info(f"Connecting to MongoDB at {self.settings.mongodb_uri}")
            self.client = MongoClient(
                self.settings.mongodb_uri,
                maxPoolSize=self.settings.mongodb_max_pool_size,
                minPoolSize=self.settings.mongodb_min_pool_size,
                connectTimeoutMS=self.settings.mongodb_connect_timeout_ms,
                serverSelectionTimeoutMS=self.settings.mongodb_server_selection_timeout_ms,
            )

            # Verify connection
            self.client.admin.command("ping")
            logger.info("MongoDB connection established")

            # Get database and collections
            self.db = self.client[self.settings.mongodb_database]
            self.clinical_docs = self.db.clinical_documents
            self.patient_timelines = self.db.patient_timelines
            self.ml_explanations = self.db.ml_explanations

            # Create indexes on first run
            if not self.indexes_created:
                self._create_indexes()
                self.indexes_created = True

        except PyMongoError as e:
            logger.error(f"Failed to connect to MongoDB: {e}")
            raise

    def _create_indexes(self):
        """Create MongoDB indexes for optimal query performance."""
        logger.info("Creating MongoDB indexes...")

        try:
            # Clinical documents indexes
            self.clinical_docs.create_index(
                [("patientId", ASCENDING), ("timestamp", DESCENDING)],
                name="patient_timestamp_idx",
            )
            self.clinical_docs.create_index(
                [("eventType", ASCENDING)], name="event_type_idx"
            )
            self.clinical_docs.create_index(
                [("enrichments.riskLevel", ASCENDING)], name="risk_level_idx"
            )
            self.clinical_docs.create_index(
                [("timestamp", DESCENDING)], name="timestamp_idx"
            )

            # Patient timelines indexes
            self.patient_timelines.create_index([("_id", ASCENDING)], name="patient_id_idx")
            self.patient_timelines.create_index(
                [("lastUpdated", DESCENDING)], name="last_updated_idx"
            )

            # ML explanations indexes
            self.ml_explanations.create_index(
                [("patientId", ASCENDING), ("timestamp", DESCENDING)],
                name="patient_timestamp_idx",
            )
            self.ml_explanations.create_index(
                [("predictions.sepsis_risk_24h.prediction", DESCENDING)],
                name="sepsis_risk_idx",
            )

            logger.info("MongoDB indexes created successfully")

        except PyMongoError as e:
            logger.error(f"Failed to create indexes: {e}")
            raise

    def process_batch(self, events: List[EnrichedClinicalEvent]) -> None:
        """
        Process a batch of enriched events and write to MongoDB.

        Args:
            events: List of enriched events
        """
        if not events:
            return

        try:
            # Prepare bulk operations
            clinical_doc_ops = []
            timeline_ops = []
            ml_explanation_ops = []

            for event in events:
                # 1. Clinical document upsert
                clinical_doc = self._create_clinical_document(event)
                clinical_doc_ops.append(
                    UpdateOne(
                        {"_id": clinical_doc["_id"]},
                        {"$set": clinical_doc},
                        upsert=True,
                    )
                )

                # 2. Patient timeline update
                timeline_event = self._create_timeline_event(event)
                timeline_ops.append(
                    UpdateOne(
                        {"_id": event.patient_id},
                        {
                            "$push": {
                                "events": {
                                    "$each": [timeline_event],
                                    "$sort": {"timestamp": -1},
                                    "$slice": self.settings.max_events_per_patient,
                                }
                            },
                            "$set": {
                                "lastUpdated": datetime.utcnow(),
                                "latestEventTime": event.timestamp,
                            },
                            "$inc": {"eventCount": 1},
                            "$setOnInsert": {
                                "firstEventTime": event.timestamp,
                            },
                        },
                        upsert=True,
                    )
                )

                # 3. ML explanation (if predictions exist)
                if event.ml_predictions:
                    ml_explanation = self._create_ml_explanation(event)
                    if ml_explanation:
                        ml_explanation_ops.append(ml_explanation)

            # Execute bulk writes
            results = {
                "clinical_docs": 0,
                "timelines": 0,
                "explanations": 0,
            }

            if clinical_doc_ops:
                result = self.clinical_docs.bulk_write(clinical_doc_ops, ordered=False)
                results["clinical_docs"] = (
                    result.upserted_count + result.modified_count
                )
                self.stats["documents_written"] += results["clinical_docs"]

            if timeline_ops:
                result = self.patient_timelines.bulk_write(timeline_ops, ordered=False)
                results["timelines"] = result.upserted_count + result.modified_count
                self.stats["timelines_updated"] += results["timelines"]

            if ml_explanation_ops:
                result = self.ml_explanations.insert_many(ml_explanation_ops)
                results["explanations"] = len(result.inserted_ids)
                self.stats["explanations_written"] += results["explanations"]

            logger.info(
                f"Batch processed: {results['clinical_docs']} docs, "
                f"{results['timelines']} timelines, {results['explanations']} explanations"
            )

        except BulkWriteError as e:
            logger.error(f"Bulk write error: {e.details}")
            self.stats["errors"] += 1
            raise

        except PyMongoError as e:
            logger.error(f"MongoDB error during batch processing: {e}")
            self.stats["errors"] += 1
            raise

        except Exception as e:
            logger.error(f"Unexpected error during batch processing: {e}")
            self.stats["errors"] += 1
            raise

    def _create_clinical_document(self, event: EnrichedClinicalEvent) -> Dict[str, Any]:
        """Create clinical document from enriched event."""
        summary = self._generate_event_summary(event)

        doc = {
            "_id": event.id,
            "patientId": event.patient_id,
            "timestamp": event.timestamp,
            "eventType": event.event_type,
            "deviceId": event.device_id,
            "rawData": event.raw_data.model_dump() if event.raw_data else None,
            "enrichments": event.enrichments.model_dump() if event.enrichments else None,
            "mlPredictions": (
                event.ml_predictions.model_dump() if event.ml_predictions else None
            ),
            "createdAt": datetime.utcnow(),
            "summary": summary,
        }

        # Remove None values
        return {k: v for k, v in doc.items() if v is not None}

    def _create_timeline_event(self, event: EnrichedClinicalEvent) -> Dict[str, Any]:
        """Create timeline event entry from enriched event."""
        timeline_event = {
            "eventId": event.id,
            "timestamp": event.timestamp,
            "eventType": event.event_type,
            "summary": self._generate_event_summary(event),
        }

        # Add risk level if available
        if event.enrichments and event.enrichments.riskLevel:
            timeline_event["riskLevel"] = event.enrichments.riskLevel

        # Add vital signs summary from raw_data
        if event.raw_data:
            timeline_event["vitalSigns"] = {
                "heartRate": event.raw_data.heart_rate,
                "bloodPressureSystolic": event.raw_data.blood_pressure_systolic,
                "bloodPressureDiastolic": event.raw_data.blood_pressure_diastolic,
                "temperature": event.raw_data.temperature_celsius,
                "oxygenSaturation": event.raw_data.spo2,
            }

        # Add prediction scores
        if event.ml_predictions:
            timeline_event["predictions"] = {
                "sepsis_risk_24h": event.ml_predictions.sepsis_risk_24h,
                "cardiac_event_risk_7d": event.ml_predictions.cardiac_event_risk_7d,
                "readmission_risk_30d": event.ml_predictions.readmission_risk_30d,
            }

        return timeline_event

    def _create_ml_explanation(self, event: EnrichedClinicalEvent) -> Optional[Dict[str, Any]]:
        """Create ML explanation document from enriched event."""
        if not event.ml_predictions:
            return None

        explanation = {
            "patientId": event.patient_id,
            "eventId": event.id,
            "timestamp": event.timestamp,
            "predictions": event.ml_predictions.model_dump(),
            "created_at": datetime.utcnow(),
        }

        return explanation

    def _generate_event_summary(self, event: EnrichedClinicalEvent) -> str:
        """Generate human-readable summary of the event."""
        parts = [f"{event.event_type} event"]

        # Add vital signs if available
        if event.raw_data:
            rd = event.raw_data
            vitals_parts = []
            if rd.heart_rate:
                vitals_parts.append(f"HR {rd.heart_rate}")
            if rd.blood_pressure_systolic and rd.blood_pressure_diastolic:
                vitals_parts.append(f"BP {rd.blood_pressure_systolic}/{rd.blood_pressure_diastolic}")
            if rd.temperature_celsius:
                vitals_parts.append(f"Temp {rd.temperature_celsius}°C")
            if rd.spo2:
                vitals_parts.append(f"SpO2 {rd.spo2}%")

            if vitals_parts:
                parts.append(f"Vitals: {', '.join(vitals_parts)}")

        # Add risk level if available
        if event.enrichments and event.enrichments.riskLevel:
            parts.append(f"Risk: {event.enrichments.riskLevel}")

        # Add alert information
        if event.ml_predictions:
            alerts = []
            if event.ml_predictions.sepsis_risk_24h and event.ml_predictions.sepsis_risk_24h > 0.7:
                alerts.append("sepsis_risk_24h")
            if event.ml_predictions.cardiac_event_risk_7d and event.ml_predictions.cardiac_event_risk_7d > 0.7:
                alerts.append("cardiac_event_risk_7d")
            if event.ml_predictions.readmission_risk_30d and event.ml_predictions.readmission_risk_30d > 0.7:
                alerts.append("readmission_risk_30d")
            if alerts:
                parts.append(f"Alerts: {', '.join(alerts)}")

        return " | ".join(parts)

    def get_statistics(self) -> Dict[str, Any]:
        """Get projector statistics."""
        stats = self.stats.copy()
        stats.update(self.get_consumer_stats())

        # Add MongoDB collection counts
        if self.clinical_docs:
            try:
                stats["total_clinical_docs"] = self.clinical_docs.count_documents({})
                stats["total_patient_timelines"] = self.patient_timelines.count_documents({})
                stats["total_ml_explanations"] = self.ml_explanations.count_documents({})
            except PyMongoError:
                pass

        return stats

    def cleanup(self):
        """Cleanup resources."""
        # Call parent cleanup if it exists
        if hasattr(super(), 'cleanup'):
            super().cleanup()

        if self.client:
            try:
                self.client.close()
                logger.info("MongoDB connection closed")
            except Exception as e:
                logger.error(f"Error closing MongoDB connection: {e}")
