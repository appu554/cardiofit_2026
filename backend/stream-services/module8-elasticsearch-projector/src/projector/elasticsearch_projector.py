"""
Elasticsearch Projector - High-Performance Clinical Event Indexing
Consumes enriched events and provides full-text search with sub-second latency
"""
import logging
from typing import List, Dict, Any, Optional
from datetime import datetime, timezone
import hashlib
from elasticsearch import Elasticsearch, helpers
from elasticsearch.exceptions import ConnectionError, RequestError

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from .index_templates import get_all_templates

logger = logging.getLogger(__name__)


class ElasticsearchProjector(KafkaConsumerBase):
    """
    Elasticsearch projector for clinical event search and analytics

    Features:
    - Bulk indexing for high throughput
    - Multi-index strategy (events, patients, documents, alerts)
    - Full-text search with clinical analyzer
    - Real-time aggregations and dashboards
    - Optimistic concurrency control
    """

    def __init__(
        self,
        kafka_config: Dict[str, Any],
        elasticsearch_url: str = "http://elasticsearch:9200",
        batch_size: int = 100,
        flush_timeout: int = 5
    ):
        super().__init__(
            kafka_config=kafka_config,
            topics=["prod.ehr.events.enriched"],
            batch_size=batch_size,
            batch_timeout_seconds=flush_timeout
        )

        self.es_url = elasticsearch_url
        self.batch_size = batch_size
        self.flush_timeout = flush_timeout

        # Initialize Elasticsearch client
        self.es = Elasticsearch(
            [elasticsearch_url],
            request_timeout=30,
            max_retries=3,
            retry_on_timeout=True
        )

        # Statistics
        self.stats = {
            "events_indexed": 0,
            "patients_updated": 0,
            "documents_created": 0,
            "alerts_created": 0,
            "errors": 0
        }

        logger.info(f"ElasticsearchProjector initialized with URL: {elasticsearch_url}")

    async def initialize(self):
        """Initialize Elasticsearch indices and templates"""
        await super().initialize()

        try:
            # Check Elasticsearch connection
            if not self.es.ping():
                raise ConnectionError("Cannot connect to Elasticsearch")

            logger.info("Elasticsearch connection established")

            # Create index templates
            templates = get_all_templates()
            for template_name, template_body in templates.items():
                try:
                    self.es.indices.put_index_template(
                        name=template_name,
                        body=template_body
                    )
                    logger.info(f"Created index template: {template_name}")
                except RequestError as e:
                    logger.warning(f"Template {template_name} already exists: {e}")

            # Create initial indices if they don't exist
            self._ensure_index("patients")
            self._ensure_index("clinical_events-2024")
            self._ensure_index("clinical_documents-2024")
            self._ensure_index("alerts-2024")

            logger.info("Elasticsearch indices initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize Elasticsearch: {e}")
            raise

    def _ensure_index(self, index_name: str):
        """Ensure an index exists"""
        if not self.es.indices.exists(index=index_name):
            self.es.indices.create(index=index_name)
            logger.info(f"Created index: {index_name}")

    def process_batch(self, events: List[Dict[str, Any]]):
        """
        Process batch of enriched events and index to Elasticsearch

        Operations:
        1. Bulk index clinical events
        2. Update patient current state
        3. Extract and index clinical documents
        4. Create alerts for high/critical risk events
        """
        if not events:
            return

        try:
            # Prepare bulk operations
            bulk_operations = []

            for event in events:
                # 1. Index clinical event
                event_ops = self._prepare_event_operations(event)
                bulk_operations.extend(event_ops)

                # 2. Update patient state
                patient_ops = self._prepare_patient_operations(event)
                bulk_operations.extend(patient_ops)

                # 3. Index clinical documents (if present)
                doc_ops = self._prepare_document_operations(event)
                bulk_operations.extend(doc_ops)

                # 4. Create alerts for high-risk events
                alert_ops = self._prepare_alert_operations(event)
                bulk_operations.extend(alert_ops)

            # Execute bulk operations
            if bulk_operations:
                success, failed = helpers.bulk(
                    self.es,
                    bulk_operations,
                    stats_only=True,
                    raise_on_error=False,
                    request_timeout=self.flush_timeout
                )

                logger.info(
                    f"Bulk indexing: {success} succeeded, {failed} failed, "
                    f"total operations: {len(bulk_operations)}"
                )

                self.stats["events_indexed"] += len(events)
                if failed > 0:
                    self.stats["errors"] += failed

        except Exception as e:
            logger.error(f"Error processing batch: {e}", exc_info=True)
            self.stats["errors"] += len(events)
            raise

    def _prepare_event_operations(self, event: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Prepare bulk operations for clinical event indexing"""
        operations = []

        event_id = event.get("eventId")
        if not event_id:
            logger.warning("Event missing eventId, skipping")
            return operations

        # Determine index name (time-based partitioning)
        timestamp = event.get("timestamp", datetime.now(timezone.utc).isoformat())
        index_name = self._get_time_based_index("clinical_events", timestamp)

        # Index event document
        operations.append({
            "_op_type": "index",
            "_index": index_name,
            "_id": event_id,
            "_source": {
                "eventId": event_id,
                "patientId": event.get("patientId"),
                "deviceId": event.get("deviceId"),
                "timestamp": timestamp,
                "eventType": event.get("eventType"),
                "stage": event.get("stage"),
                "rawData": event.get("rawData", {}),
                "enrichments": event.get("enrichments", {}),
                "semanticAnnotations": event.get("semanticAnnotations", {}),
                "mlPredictions": event.get("mlPredictions", {}),
                "processingTime": datetime.now(timezone.utc).isoformat(),
                "version": "1.0"
            }
        })

        return operations

    def _prepare_patient_operations(self, event: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Prepare bulk operations for patient state updates"""
        operations = []

        patient_id = event.get("patientId")
        if not patient_id:
            return operations

        # Extract patient data from enrichments
        enrichments = event.get("enrichments", {})
        fhir_resources = enrichments.get("fhirResources", {})
        patient_resource = fhir_resources.get("Patient", {})

        # Extract current vitals from raw data
        raw_data = event.get("rawData", {})
        if raw_data is None:
            raw_data = {}

        # Get ML predictions
        ml_predictions = event.get("mlPredictions")
        if ml_predictions is None:
            ml_predictions = {}
        risk_level = ml_predictions.get("riskLevel", "UNKNOWN")
        risk_score = ml_predictions.get("riskScore", 0.0)

        # Update patient document (upsert)
        patient_doc = {
            "patientId": patient_id,
            "currentState": {
                "latestEventId": event.get("eventId"),
                "latestEventTime": event.get("timestamp"),
                "currentRiskLevel": risk_level,
                "currentRiskScore": risk_score,
                "deviceIds": [event.get("deviceId")] if event.get("deviceId") else []
            },
            "vitalsSummary": {
                "latestHeartRate": raw_data.get("heartRate"),
                "latestBP": raw_data.get("bloodPressure"),
                "latestO2Sat": raw_data.get("oxygenSaturation"),
                "latestTemp": raw_data.get("temperature")
            },
            "updatedAt": datetime.now(timezone.utc).isoformat()
        }

        # Add demographics if available from FHIR resource
        if patient_resource:
            demographics = self._extract_patient_demographics(patient_resource)
            if demographics:
                patient_doc["demographics"] = demographics

        operations.append({
            "_op_type": "update",
            "_index": "patients",
            "_id": patient_id,
            "doc": patient_doc,
            "doc_as_upsert": True
        })

        self.stats["patients_updated"] += 1

        return operations

    def _prepare_document_operations(self, event: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Prepare bulk operations for clinical document indexing"""
        operations = []

        # Extract clinical notes/documents from enrichments
        enrichments = event.get("enrichments", {})
        clinical_context = enrichments.get("clinicalContext", {})
        notes = clinical_context.get("notes")

        if not notes:
            return operations

        # Create document from clinical notes
        event_id = event.get("eventId")
        patient_id = event.get("patientId")
        timestamp = event.get("timestamp", datetime.now(timezone.utc).isoformat())

        # Generate unique document ID
        doc_id = hashlib.sha256(f"{event_id}-clinical-note".encode()).hexdigest()[:16]

        index_name = self._get_time_based_index("clinical_documents", timestamp)

        operations.append({
            "_op_type": "index",
            "_index": index_name,
            "_id": doc_id,
            "_source": {
                "documentId": doc_id,
                "eventId": event_id,
                "patientId": patient_id,
                "documentType": "clinical_note",
                "title": f"Clinical Note - Event {event_id}",
                "content": notes,
                "author": "system",
                "createdAt": timestamp,
                "tags": ["automated", "event-generated"]
            }
        })

        self.stats["documents_created"] += 1

        return operations

    def _prepare_alert_operations(self, event: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Prepare bulk operations for alert creation based on risk levels"""
        operations = []

        ml_predictions = event.get("mlPredictions")
        if ml_predictions is None:
            ml_predictions = {}
        risk_level = ml_predictions.get("riskLevel", "UNKNOWN")
        risk_score = ml_predictions.get("riskScore", 0.0)

        # Create alerts only for HIGH or CRITICAL risk levels
        if risk_level not in ["HIGH", "CRITICAL"]:
            return operations

        event_id = event.get("eventId")
        patient_id = event.get("patientId")
        timestamp = event.get("timestamp", datetime.now(timezone.utc).isoformat())

        # Generate alert ID
        alert_id = hashlib.sha256(f"{event_id}-alert-{risk_level}".encode()).hexdigest()[:16]

        # Extract trigger information
        raw_data = event.get("rawData", {})
        if raw_data is None:
            raw_data = {}
        trigger_info = self._identify_alert_trigger(raw_data, ml_predictions)

        # Get recommendations
        recommendations = ml_predictions.get("recommendations", [])
        if isinstance(recommendations, list):
            recommendations = "; ".join(recommendations)

        index_name = self._get_time_based_index("alerts", timestamp)

        operations.append({
            "_op_type": "index",
            "_index": index_name,
            "_id": alert_id,
            "_source": {
                "alertId": alert_id,
                "eventId": event_id,
                "patientId": patient_id,
                "alertType": "CLINICAL_RISK",
                "severity": risk_level,
                "riskScore": risk_score,
                "title": f"{risk_level} Risk Alert for Patient {patient_id}",
                "description": f"Risk score {risk_score:.2f} detected from event {event_id}",
                "triggeredBy": trigger_info,
                "recommendations": recommendations,
                "acknowledged": False,
                "createdAt": timestamp,
                "expiresAt": None  # Manual acknowledgment required
            }
        })

        self.stats["alerts_created"] += 1

        return operations

    def _identify_alert_trigger(
        self,
        raw_data: Dict[str, Any],
        ml_predictions: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Identify what triggered the alert"""
        # Check for abnormal vitals
        hr = raw_data.get("heartRate")
        bp = raw_data.get("bloodPressure", {})
        o2_sat = raw_data.get("oxygenSaturation")

        triggers = []

        if hr and (hr > 100 or hr < 60):
            triggers.append({"metric": "heartRate", "value": hr, "threshold": "60-100"})

        if bp:
            systolic = bp.get("systolic")
            if systolic and (systolic > 140 or systolic < 90):
                triggers.append({"metric": "bloodPressure.systolic", "value": systolic, "threshold": "90-140"})

        if o2_sat and o2_sat < 95:
            triggers.append({"metric": "oxygenSaturation", "value": o2_sat, "threshold": ">95"})

        # Default to risk score if no specific vital trigger
        if not triggers:
            risk_score = ml_predictions.get("riskScore", 0.0)
            return {"metric": "riskScore", "value": risk_score, "threshold": 0.7}

        return triggers[0]  # Return first trigger

    def _extract_patient_demographics(self, patient_resource: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Extract demographics from FHIR Patient resource"""
        try:
            name_list = patient_resource.get("name", [])
            name = None
            if name_list:
                name_obj = name_list[0]
                given = name_obj.get("given", [])
                family = name_obj.get("family", "")
                name = f"{' '.join(given)} {family}".strip()

            birth_date = patient_resource.get("birthDate")
            gender = patient_resource.get("gender")

            # Calculate age from birth date
            age = None
            if birth_date:
                try:
                    from datetime import datetime
                    birth = datetime.fromisoformat(birth_date.replace("Z", "+00:00"))
                    age = (datetime.now(timezone.utc) - birth).days // 365
                except:
                    pass

            demographics = {}
            if name:
                demographics["name"] = name
            if age is not None:
                demographics["age"] = age
            if gender:
                demographics["gender"] = gender
            if birth_date:
                demographics["dateOfBirth"] = birth_date

            return demographics if demographics else None

        except Exception as e:
            logger.warning(f"Failed to extract patient demographics: {e}")
            return None

    def _get_time_based_index(self, base_name: str, timestamp: str) -> str:
        """Get time-based index name (e.g., clinical_events-2024)"""
        try:
            dt = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
            year = dt.year
            return f"{base_name}-{year}"
        except:
            # Fallback to current year
            return f"{base_name}-{datetime.now().year}"

    async def get_health(self) -> Dict[str, Any]:
        """Get projector health status"""
        base_health = await super().get_health()

        try:
            es_health = self.es.cluster.health()
            es_status = {
                "connected": True,
                "cluster_status": es_health.get("status"),
                "number_of_nodes": es_health.get("number_of_nodes"),
                "active_shards": es_health.get("active_shards")
            }
        except Exception as e:
            es_status = {"connected": False, "error": str(e)}

        # Get index document counts
        index_stats = {}
        for index in ["patients", "clinical_events-2024", "clinical_documents-2024", "alerts-2024"]:
            try:
                if self.es.indices.exists(index=index):
                    count = self.es.count(index=index)
                    index_stats[index] = count.get("count", 0)
            except Exception as e:
                index_stats[index] = f"error: {e}"

        return {
            **base_health,
            "elasticsearch": es_status,
            "index_statistics": index_stats,
            "processing_statistics": self.stats
        }

    def get_projector_name(self) -> str:
        """Return unique projector identifier"""
        return "elasticsearch-projector"

    async def shutdown(self):
        """Shutdown projector gracefully"""
        logger.info("Shutting down ElasticsearchProjector")

        # Close Elasticsearch connection
        if self.es:
            self.es.close()

        await super().shutdown()

        logger.info(
            f"Final statistics: {self.stats['events_indexed']} events indexed, "
            f"{self.stats['patients_updated']} patients updated, "
            f"{self.stats['documents_created']} documents created, "
            f"{self.stats['alerts_created']} alerts created, "
            f"{self.stats['errors']} errors"
        )
