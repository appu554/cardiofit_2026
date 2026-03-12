"""
UPS (Unified Patient Summary) Read Model Projector

Consumes from prod.ehr.events.enriched and maintains denormalized patient summaries
in PostgreSQL for sub-10ms query performance.

Performance Target:
- Single patient lookup: <10ms
- Throughput: 500 updates/sec
- Batch processing with UPSERT optimization
"""

import json
import time
from typing import Dict, List, Any, Optional
from datetime import datetime
import logging

import psycopg2
from psycopg2.extras import execute_batch
from psycopg2.pool import ThreadedConnectionPool

from module8_shared.kafka_base import KafkaConsumerBase
from module8_shared.config import StreamConfig

logger = logging.getLogger(__name__)


class UPSProjector(KafkaConsumerBase):
    """
    Unified Patient Summary Projector

    Projects enriched events into denormalized patient summary table.
    Optimized for real-time dashboard queries.
    """

    def __init__(self, config: StreamConfig):
        super().__init__(
            consumer_group="module8-ups-projector",
            topics=["prod.ehr.events.enriched"],
            config=config
        )

        # PostgreSQL connection pool (same container as other projectors)
        self.db_pool = ThreadedConnectionPool(
            minconn=2,
            maxconn=10,
            host=config.postgres_host,
            port=config.postgres_port,
            database=config.postgres_db,
            user=config.postgres_user,
            password=config.postgres_password
        )

        self.batch_size = 100
        self.batch_timeout = 5.0  # seconds

        # Performance metrics
        self.metrics = {
            "events_processed": 0,
            "patients_updated": 0,
            "batches_processed": 0,
            "total_processing_time_ms": 0,
            "upsert_time_ms": 0
        }

        logger.info("UPS Projector initialized")

    def process_batch(self, events: List[Dict[str, Any]]) -> None:
        """
        Process batch of enriched events into UPS read model.

        Strategy:
        1. Group events by patient_id
        2. For each patient, merge latest data
        3. Bulk UPSERT into PostgreSQL
        4. Track state changes for audit
        """
        start_time = time.time()

        if not events:
            return

        try:
            # Group events by patient_id
            patient_updates = self._group_events_by_patient(events)

            # Prepare UPSERT statements
            upsert_data = self._prepare_upserts(patient_updates)

            # Execute batch UPSERT
            upsert_start = time.time()
            self._execute_batch_upsert(upsert_data)
            upsert_time = (time.time() - upsert_start) * 1000

            # Track state changes for significant events
            self._track_state_changes(patient_updates)

            # Update metrics
            processing_time = (time.time() - start_time) * 1000
            self.metrics["events_processed"] += len(events)
            self.metrics["patients_updated"] += len(patient_updates)
            self.metrics["batches_processed"] += 1
            self.metrics["total_processing_time_ms"] += processing_time
            self.metrics["upsert_time_ms"] += upsert_time

            logger.info(
                f"Processed batch: {len(events)} events, {len(patient_updates)} patients, "
                f"{processing_time:.2f}ms total, {upsert_time:.2f}ms UPSERT"
            )

        except Exception as e:
            logger.error(f"Error processing batch: {e}", exc_info=True)
            raise

    def _group_events_by_patient(self, events: List[Dict[str, Any]]) -> Dict[str, List[Dict[str, Any]]]:
        """Group events by patient_id for efficient processing."""
        patient_events = {}

        for event in events:
            try:
                payload = event.get("payload", {})
                patient_id = payload.get("patient_id")

                if not patient_id:
                    logger.warning(f"Event missing patient_id: {event.get('event_id')}")
                    continue

                if patient_id not in patient_events:
                    patient_events[patient_id] = []

                patient_events[patient_id].append(event)

            except Exception as e:
                logger.error(f"Error grouping event: {e}", exc_info=True)
                continue

        return patient_events

    def _prepare_upserts(self, patient_updates: Dict[str, List[Dict[str, Any]]]) -> List[tuple]:
        """
        Prepare UPSERT data for batch execution.

        For each patient, merge all events and extract latest state.
        """
        upsert_data = []

        for patient_id, events in patient_updates.items():
            try:
                # Sort events by timestamp (most recent last)
                sorted_events = sorted(
                    events,
                    key=lambda e: e.get("metadata", {}).get("timestamp", 0)
                )

                # Extract merged state from events
                merged_state = self._merge_patient_state(patient_id, sorted_events)

                if merged_state:
                    upsert_data.append(merged_state)

            except Exception as e:
                logger.error(f"Error preparing UPSERT for patient {patient_id}: {e}", exc_info=True)
                continue

        return upsert_data

    def _merge_patient_state(self, patient_id: str, events: List[Dict[str, Any]]) -> Optional[tuple]:
        """
        Merge events into single patient state for UPSERT.

        Returns tuple for UPSERT query.
        """
        try:
            # Most recent event (for metadata)
            latest_event = events[-1]
            metadata = latest_event.get("metadata", {})
            payload = latest_event.get("payload", {})
            enrichments = latest_event.get("enrichments", {})

            # Extract latest vitals (if present)
            latest_vitals = None
            latest_vitals_timestamp = None

            if payload.get("event_type") == "VITAL_SIGNS":
                vital_data = payload.get("vital_signs", {})
                if vital_data:
                    latest_vitals = json.dumps(vital_data)
                    latest_vitals_timestamp = metadata.get("timestamp")

            # Extract clinical scores from enrichments
            clinical_scores = enrichments.get("clinical_scores", {})
            news2_score = clinical_scores.get("news2", {}).get("total_score")
            news2_category = clinical_scores.get("news2", {}).get("category")
            qsofa_score = clinical_scores.get("qSOFA", {}).get("score")
            sofa_score = clinical_scores.get("SOFA", {}).get("score")

            # Extract risk level
            risk_assessment = enrichments.get("risk_assessment", {})
            risk_level = risk_assessment.get("level", "UNKNOWN")

            # Extract ML predictions
            ml_predictions = enrichments.get("ml_predictions")
            ml_predictions_json = json.dumps(ml_predictions) if ml_predictions else None
            ml_predictions_timestamp = metadata.get("timestamp") if ml_predictions else None

            # Build active alerts array
            active_alerts = []
            alerts = enrichments.get("alerts", [])
            for alert in alerts:
                if alert.get("priority") in ["HIGH", "CRITICAL"]:
                    active_alerts.append({
                        "alert_id": alert.get("alert_id"),
                        "type": alert.get("type"),
                        "priority": alert.get("priority"),
                        "message": alert.get("message"),
                        "timestamp": metadata.get("timestamp")
                    })

            active_alerts_json = json.dumps(active_alerts)
            active_alerts_count = len(active_alerts)

            # Extract location (if present in payload)
            current_department = payload.get("department")
            current_location = payload.get("location")

            # Protocol compliance
            protocol_compliance = enrichments.get("protocol_compliance")
            protocol_compliance_json = json.dumps(protocol_compliance) if protocol_compliance else None
            protocol_status = None
            if protocol_compliance:
                protocol_status = protocol_compliance.get("status", "UNKNOWN")

            # Demographics (placeholder - would be enriched from patient service)
            demographics = payload.get("demographics")
            demographics_json = json.dumps(demographics) if demographics else None

            # Event metadata
            last_event_id = latest_event.get("event_id")
            last_event_type = payload.get("event_type")
            last_updated = metadata.get("timestamp")
            event_count = len(events)

            # Return tuple for UPSERT
            return (
                patient_id,
                demographics_json,
                current_department,
                current_location,
                None,  # admission_timestamp (placeholder)
                latest_vitals,
                latest_vitals_timestamp,
                news2_score,
                news2_category,
                qsofa_score,
                sofa_score,
                risk_level,
                ml_predictions_json,
                ml_predictions_timestamp,
                active_alerts_json,
                active_alerts_count,
                protocol_compliance_json,
                protocol_status,
                None,  # vitals_trend (placeholder)
                None,  # trend_confidence (placeholder)
                last_event_id,
                last_event_type,
                last_updated,
                event_count
            )

        except Exception as e:
            logger.error(f"Error merging state for patient {patient_id}: {e}", exc_info=True)
            return None

    def _execute_batch_upsert(self, upsert_data: List[tuple]) -> None:
        """
        Execute batch UPSERT into PostgreSQL.

        Uses INSERT ... ON CONFLICT UPDATE for efficiency.
        """
        if not upsert_data:
            return

        conn = self.db_pool.getconn()
        try:
            with conn.cursor() as cursor:
                upsert_query = """
                    INSERT INTO module8_projections.ups_read_model (
                        patient_id,
                        demographics,
                        current_department,
                        current_location,
                        admission_timestamp,
                        latest_vitals,
                        latest_vitals_timestamp,
                        news2_score,
                        news2_category,
                        qsofa_score,
                        sofa_score,
                        risk_level,
                        ml_predictions,
                        ml_predictions_timestamp,
                        active_alerts,
                        active_alerts_count,
                        protocol_compliance,
                        protocol_status,
                        vitals_trend,
                        trend_confidence,
                        last_event_id,
                        last_event_type,
                        last_updated,
                        event_count
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    ON CONFLICT (patient_id) DO UPDATE SET
                        demographics = COALESCE(EXCLUDED.demographics, ups_read_model.demographics),
                        current_department = COALESCE(EXCLUDED.current_department, ups_read_model.current_department),
                        current_location = COALESCE(EXCLUDED.current_location, ups_read_model.current_location),
                        latest_vitals = CASE
                            WHEN EXCLUDED.latest_vitals IS NOT NULL AND
                                 EXCLUDED.latest_vitals_timestamp > COALESCE(ups_read_model.latest_vitals_timestamp, 0)
                            THEN EXCLUDED.latest_vitals
                            ELSE ups_read_model.latest_vitals
                        END,
                        latest_vitals_timestamp = CASE
                            WHEN EXCLUDED.latest_vitals_timestamp > COALESCE(ups_read_model.latest_vitals_timestamp, 0)
                            THEN EXCLUDED.latest_vitals_timestamp
                            ELSE ups_read_model.latest_vitals_timestamp
                        END,
                        news2_score = COALESCE(EXCLUDED.news2_score, ups_read_model.news2_score),
                        news2_category = COALESCE(EXCLUDED.news2_category, ups_read_model.news2_category),
                        qsofa_score = COALESCE(EXCLUDED.qsofa_score, ups_read_model.qsofa_score),
                        sofa_score = COALESCE(EXCLUDED.sofa_score, ups_read_model.sofa_score),
                        risk_level = COALESCE(EXCLUDED.risk_level, ups_read_model.risk_level),
                        ml_predictions = CASE
                            WHEN EXCLUDED.ml_predictions IS NOT NULL AND
                                 EXCLUDED.ml_predictions_timestamp > COALESCE(ups_read_model.ml_predictions_timestamp, 0)
                            THEN EXCLUDED.ml_predictions
                            ELSE ups_read_model.ml_predictions
                        END,
                        ml_predictions_timestamp = CASE
                            WHEN EXCLUDED.ml_predictions_timestamp > COALESCE(ups_read_model.ml_predictions_timestamp, 0)
                            THEN EXCLUDED.ml_predictions_timestamp
                            ELSE ups_read_model.ml_predictions_timestamp
                        END,
                        active_alerts = EXCLUDED.active_alerts,
                        active_alerts_count = EXCLUDED.active_alerts_count,
                        protocol_compliance = COALESCE(EXCLUDED.protocol_compliance, ups_read_model.protocol_compliance),
                        protocol_status = COALESCE(EXCLUDED.protocol_status, ups_read_model.protocol_status),
                        last_event_id = EXCLUDED.last_event_id,
                        last_event_type = EXCLUDED.last_event_type,
                        last_updated = EXCLUDED.last_updated,
                        event_count = ups_read_model.event_count + EXCLUDED.event_count,
                        updated_at = NOW();
                """

                execute_batch(cursor, upsert_query, upsert_data, page_size=100)
                conn.commit()

                logger.debug(f"UPSERT completed: {len(upsert_data)} patients")

        except Exception as e:
            conn.rollback()
            logger.error(f"Error executing batch UPSERT: {e}", exc_info=True)
            raise
        finally:
            self.db_pool.putconn(conn)

    def _track_state_changes(self, patient_updates: Dict[str, List[Dict[str, Any]]]) -> None:
        """
        Track significant state changes in audit log.

        Examples: RISK_ESCALATION, ALERT_TRIGGERED, PROTOCOL_VIOLATION
        """
        # TODO: Implement state change tracking
        # Would require comparing old vs new state from database
        # For now, skip to avoid additional DB reads
        pass

    def get_health(self) -> Dict[str, Any]:
        """Health check with projector-specific metrics."""
        health = super().get_health()

        # Add projector-specific metrics
        avg_processing_time = 0
        if self.metrics["batches_processed"] > 0:
            avg_processing_time = (
                self.metrics["total_processing_time_ms"] / self.metrics["batches_processed"]
            )

        avg_upsert_time = 0
        if self.metrics["batches_processed"] > 0:
            avg_upsert_time = (
                self.metrics["upsert_time_ms"] / self.metrics["batches_processed"]
            )

        health["projector"] = {
            "events_processed": self.metrics["events_processed"],
            "patients_updated": self.metrics["patients_updated"],
            "batches_processed": self.metrics["batches_processed"],
            "avg_batch_processing_ms": round(avg_processing_time, 2),
            "avg_upsert_ms": round(avg_upsert_time, 2)
        }

        # Check database connection
        try:
            conn = self.db_pool.getconn()
            with conn.cursor() as cursor:
                cursor.execute("SELECT COUNT(*) FROM module8_projections.ups_read_model")
                patient_count = cursor.fetchone()[0]
                health["database"] = {
                    "status": "healthy",
                    "total_patients": patient_count
                }
            self.db_pool.putconn(conn)
        except Exception as e:
            health["database"] = {
                "status": "unhealthy",
                "error": str(e)
            }

        return health

    def shutdown(self):
        """Graceful shutdown with connection pool cleanup."""
        super().shutdown()
        if self.db_pool:
            self.db_pool.closeall()
            logger.info("Database connection pool closed")
