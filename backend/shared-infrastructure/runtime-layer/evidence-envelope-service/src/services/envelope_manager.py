"""
Evidence Envelope Manager Service
Central service for managing clinical decision evidence envelopes
"""

import asyncio
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional
from uuid import uuid4
import structlog

from ..models.evidence_envelope import (
    EvidenceEnvelope,
    EvidenceEnvelopeRequest,
    EvidenceEnvelopeResponse,
    ClinicalContext,
    InferenceStep,
    ConfidenceMetrics
)
from ..utils.database import mongodb, redis_client
from ..utils.kafka_client import kafka_producer
from ..utils.config import settings

logger = structlog.get_logger()


class EvidenceEnvelopeManager:
    """
    Manages the lifecycle of evidence envelopes for clinical decisions

    Responsibilities:
    - Create and manage evidence envelopes
    - Track inference chains and confidence scores
    - Ensure regulatory compliance and audit trails
    - Cache management for performance optimization
    """

    def __init__(self):
        self.redis_client = redis_client
        self.kafka_producer = kafka_producer
        self._envelope_cache: Dict[str, EvidenceEnvelope] = {}
        self._cache_ttl = timedelta(hours=1)

        logger.info("evidence_envelope_manager_initialized")

    async def create_envelope(
        self,
        request: EvidenceEnvelopeRequest
    ) -> EvidenceEnvelope:
        """
        Create a new evidence envelope for a clinical decision

        Args:
            request: Evidence envelope creation request

        Returns:
            Created evidence envelope
        """
        start_time = datetime.utcnow()

        try:
            # Create new envelope
            envelope = EvidenceEnvelope(
                proposal_id=request.proposal_id,
                snapshot_reference=request.snapshot_id,
                knowledge_versions=request.knowledge_versions,
                clinical_context=request.clinical_context
            )

            # Track creation time
            creation_duration_ms = int((datetime.utcnow() - start_time).total_seconds() * 1000)
            envelope.creation_duration_ms = creation_duration_ms

            # Store in database
            await self._persist_envelope(envelope)

            # Cache for fast access
            await self._cache_envelope(envelope)

            # Publish creation event
            await self._publish_envelope_event(envelope, "created")

            logger.info(
                "evidence_envelope_created",
                envelope_id=envelope.envelope_id,
                proposal_id=envelope.proposal_id,
                duration_ms=creation_duration_ms
            )

            return envelope

        except Exception as e:
            logger.error(
                "envelope_creation_failed",
                proposal_id=request.proposal_id,
                error=str(e)
            )
            raise

    async def add_inference_step(
        self,
        envelope_id: str,
        step_type: str,
        description: str,
        source_data: Dict[str, Any],
        reasoning_logic: str,
        result_data: Dict[str, Any],
        confidence: float,
        execution_time_ms: int,
        knowledge_sources: Optional[List[str]] = None
    ) -> EvidenceEnvelope:
        """
        Add an inference step to an existing envelope

        Args:
            envelope_id: Envelope identifier
            step_type: Type of inference step
            description: Human-readable description
            source_data: Input data for the step
            reasoning_logic: Logic applied
            result_data: Output from the step
            confidence: Confidence score (0-1)
            execution_time_ms: Execution time
            knowledge_sources: Knowledge bases consulted

        Returns:
            Updated evidence envelope
        """
        try:
            # Retrieve envelope
            envelope = await self.get_envelope(envelope_id)
            if not envelope:
                raise ValueError(f"Envelope {envelope_id} not found")

            # Add inference step
            envelope.add_inference_step(
                step_type=step_type,
                description=description,
                source_data=source_data,
                reasoning_logic=reasoning_logic,
                result_data=result_data,
                confidence=confidence,
                execution_time_ms=execution_time_ms,
                knowledge_sources=knowledge_sources or []
            )

            # Update in database
            await self._update_envelope(envelope)

            # Update cache
            await self._cache_envelope(envelope)

            # Publish update event
            await self._publish_envelope_event(envelope, "step_added")

            logger.info(
                "inference_step_added",
                envelope_id=envelope_id,
                step_type=step_type,
                confidence=confidence,
                execution_time_ms=execution_time_ms
            )

            return envelope

        except Exception as e:
            logger.error(
                "add_inference_step_failed",
                envelope_id=envelope_id,
                step_type=step_type,
                error=str(e)
            )
            raise

    async def finalize_envelope(
        self,
        envelope_id: str,
        final_conclusion: Dict[str, Any],
        validation_results: Optional[Dict[str, Any]] = None
    ) -> EvidenceEnvelope:
        """
        Finalize an evidence envelope with conclusions

        Args:
            envelope_id: Envelope identifier
            final_conclusion: Final decision/recommendation
            validation_results: Optional validation results

        Returns:
            Finalized evidence envelope
        """
        try:
            # Retrieve envelope
            envelope = await self.get_envelope(envelope_id)
            if not envelope:
                raise ValueError(f"Envelope {envelope_id} not found")

            if envelope.status == "finalized":
                raise ValueError(f"Envelope {envelope_id} is already finalized")

            # Calculate final confidence if validation provided
            if validation_results:
                self._update_confidence_from_validation(envelope, validation_results)

            # Finalize envelope
            envelope.finalize_envelope(final_conclusion)

            # Persist finalized state
            await self._update_envelope(envelope)

            # Update cache with extended TTL for finalized envelopes
            await self._cache_envelope(envelope, ttl_hours=24)

            # Publish finalization event
            await self._publish_envelope_event(envelope, "finalized")

            # Generate audit record
            audit_record = envelope.to_audit_record()
            await self._store_audit_record(audit_record)

            logger.info(
                "envelope_finalized",
                envelope_id=envelope_id,
                confidence_overall=envelope.confidence_scores.overall,
                processing_time_ms=envelope.total_processing_time_ms,
                integrity_verified=envelope.verify_integrity()
            )

            return envelope

        except Exception as e:
            logger.error(
                "envelope_finalization_failed",
                envelope_id=envelope_id,
                error=str(e)
            )
            raise

    async def get_envelope(
        self,
        envelope_id: str
    ) -> Optional[EvidenceEnvelope]:
        """
        Retrieve an evidence envelope by ID

        Args:
            envelope_id: Envelope identifier

        Returns:
            Evidence envelope or None if not found
        """
        try:
            # Check memory cache first
            if envelope_id in self._envelope_cache:
                envelope = self._envelope_cache[envelope_id]
                logger.debug("envelope_retrieved_from_memory", envelope_id=envelope_id)
                return envelope

            # Check Redis cache
            cached_data = await self.redis_client.get(f"envelope:{envelope_id}")
            if cached_data:
                envelope = EvidenceEnvelope.model_validate_json(cached_data)
                self._envelope_cache[envelope_id] = envelope
                logger.debug("envelope_retrieved_from_redis", envelope_id=envelope_id)
                return envelope

            # Retrieve from database
            envelope = await self._fetch_envelope(envelope_id)

            if envelope:
                # Update caches
                await self._cache_envelope(envelope)
                logger.debug("envelope_retrieved_from_database", envelope_id=envelope_id)

            return envelope

        except Exception as e:
            logger.error(
                "envelope_retrieval_failed",
                envelope_id=envelope_id,
                error=str(e)
            )
            return None

    async def wrap_service_response(
        self,
        service_response: Dict[str, Any],
        envelope_id: str
    ) -> Dict[str, Any]:
        """
        Wrap a service response with evidence envelope metadata

        Args:
            service_response: Original service response
            envelope_id: Evidence envelope identifier

        Returns:
            Response wrapped with evidence envelope
        """
        try:
            envelope = await self.get_envelope(envelope_id)
            if not envelope:
                raise ValueError(f"Envelope {envelope_id} not found")

            wrapped_response = {
                "clinical_data": service_response,
                "evidence_envelope": {
                    "envelope_id": envelope.envelope_id,
                    "proposal_id": envelope.proposal_id,
                    "snapshot_reference": envelope.snapshot_reference,
                    "knowledge_versions": envelope.knowledge_versions,
                    "confidence_scores": {
                        "overall": envelope.confidence_scores.overall,
                        "components": envelope.confidence_scores.components,
                        "methodology": envelope.confidence_scores.methodology
                    },
                    "inference_summary": self._summarize_inference_chain(envelope),
                    "regulatory_compliance": {
                        "standards": envelope.regulatory_compliance.standards_compliance,
                        "audit_complete": envelope.regulatory_compliance.audit_trail_complete,
                        "provenance_verified": envelope.regulatory_compliance.provenance_verified,
                        "integrity_verified": envelope.verify_integrity()
                    },
                    "performance_metrics": {
                        "creation_time_ms": envelope.creation_duration_ms,
                        "total_processing_time_ms": envelope.total_processing_time_ms,
                        "inference_steps": len(envelope.inference_chain.steps)
                    }
                },
                "metadata": {
                    "wrapped_at": datetime.utcnow().isoformat(),
                    "envelope_status": envelope.status
                }
            }

            logger.info(
                "service_response_wrapped",
                envelope_id=envelope_id,
                response_keys=list(service_response.keys())
            )

            return wrapped_response

        except Exception as e:
            logger.error(
                "response_wrapping_failed",
                envelope_id=envelope_id,
                error=str(e)
            )
            # Return unwrapped response on error
            return service_response

    async def query_envelopes(
        self,
        proposal_id: Optional[str] = None,
        patient_id: Optional[str] = None,
        workflow_type: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
        status: Optional[str] = None,
        limit: int = 100
    ) -> List[EvidenceEnvelopeResponse]:
        """
        Query evidence envelopes based on criteria

        Args:
            proposal_id: Filter by proposal ID
            patient_id: Filter by patient ID
            workflow_type: Filter by workflow type
            start_date: Filter by start date
            end_date: Filter by end date
            status: Filter by status
            limit: Maximum results to return

        Returns:
            List of evidence envelope responses
        """
        try:
            envelopes = await self._query_envelopes(
                proposal_id=proposal_id,
                patient_id=patient_id,
                workflow_type=workflow_type,
                start_date=start_date,
                end_date=end_date,
                status=status,
                limit=limit
            )

            responses = [
                EvidenceEnvelopeResponse.from_envelope(env)
                for env in envelopes
            ]

            logger.info(
                "envelopes_queried",
                count=len(responses),
                filters={
                    "proposal_id": proposal_id,
                    "patient_id": patient_id,
                    "workflow_type": workflow_type,
                    "status": status
                }
            )

            return responses

        except Exception as e:
            logger.error(
                "envelope_query_failed",
                error=str(e)
            )
            return []

    async def get_audit_trail(
        self,
        envelope_id: str
    ) -> Dict[str, Any]:
        """
        Get complete audit trail for an envelope

        Args:
            envelope_id: Envelope identifier

        Returns:
            Complete audit trail
        """
        try:
            envelope = await self.get_envelope(envelope_id)
            if not envelope:
                raise ValueError(f"Envelope {envelope_id} not found")

            audit_trail = {
                "envelope_id": envelope_id,
                "audit_records": [],
                "timeline": []
            }

            # Build timeline of events
            timeline_events = []

            # Creation event
            timeline_events.append({
                "timestamp": envelope.created_at.isoformat(),
                "event_type": "envelope_created",
                "details": {
                    "proposal_id": envelope.proposal_id,
                    "snapshot_reference": envelope.snapshot_reference
                }
            })

            # Inference steps
            for step in envelope.inference_chain.steps:
                timeline_events.append({
                    "timestamp": step.timestamp.isoformat(),
                    "event_type": "inference_step",
                    "details": {
                        "step_type": step.step_type,
                        "description": step.description,
                        "confidence": step.confidence,
                        "execution_time_ms": step.execution_time_ms
                    }
                })

            # Finalization event
            if envelope.finalized_at:
                timeline_events.append({
                    "timestamp": envelope.finalized_at.isoformat(),
                    "event_type": "envelope_finalized",
                    "details": {
                        "final_confidence": envelope.confidence_scores.overall,
                        "total_processing_time_ms": envelope.total_processing_time_ms
                    }
                })

            # Sort timeline by timestamp
            timeline_events.sort(key=lambda x: x["timestamp"])
            audit_trail["timeline"] = timeline_events

            # Add audit record
            audit_trail["audit_records"].append(envelope.to_audit_record())

            # Add integrity verification
            audit_trail["integrity"] = {
                "checksum": envelope.checksum,
                "verified": envelope.verify_integrity(),
                "verification_timestamp": datetime.utcnow().isoformat()
            }

            logger.info(
                "audit_trail_generated",
                envelope_id=envelope_id,
                events_count=len(timeline_events)
            )

            return audit_trail

        except Exception as e:
            logger.error(
                "audit_trail_generation_failed",
                envelope_id=envelope_id,
                error=str(e)
            )
            raise

    # Private helper methods

    def _summarize_inference_chain(
        self,
        envelope: EvidenceEnvelope
    ) -> Dict[str, Any]:
        """Summarize the inference chain for response"""

        chain = envelope.inference_chain

        return {
            "total_steps": len(chain.steps),
            "knowledge_sources": chain.get_knowledge_sources(),
            "confidence_range": {
                "min": min((s.confidence for s in chain.steps), default=0.0),
                "max": max((s.confidence for s in chain.steps), default=0.0),
                "average": sum(s.confidence for s in chain.steps) / len(chain.steps) if chain.steps else 0.0
            },
            "total_execution_time_ms": chain.total_execution_time_ms,
            "step_types": list(set(s.step_type for s in chain.steps)),
            "final_conclusion": chain.final_conclusion
        }

    def _update_confidence_from_validation(
        self,
        envelope: EvidenceEnvelope,
        validation_results: Dict[str, Any]
    ):
        """Update confidence scores based on validation results"""

        # Extract validation metrics
        accuracy = validation_results.get("accuracy", 1.0)
        completeness = validation_results.get("completeness", 1.0)
        consistency = validation_results.get("consistency", 1.0)

        # Update component scores
        envelope.confidence_scores.components.update({
            "validation_accuracy": accuracy,
            "validation_completeness": completeness,
            "validation_consistency": consistency
        })

        # Recalculate overall confidence
        base_confidence = envelope.confidence_scores.overall
        validation_factor = (accuracy + completeness + consistency) / 3

        envelope.confidence_scores.overall = base_confidence * validation_factor
        envelope.confidence_scores.methodology = "composite_with_validation"

    async def _cache_envelope(
        self,
        envelope: EvidenceEnvelope,
        ttl_hours: int = 1
    ):
        """Cache envelope in Redis and memory"""

        # Memory cache
        self._envelope_cache[envelope.envelope_id] = envelope

        # Redis cache
        cache_key = f"envelope:{envelope.envelope_id}"
        await self.redis_client.set(
            cache_key,
            envelope.model_dump_json(),
            ttl=ttl_hours * 3600
        )

        # Limit memory cache size
        if len(self._envelope_cache) > 1000:
            # Remove oldest entries
            oldest_ids = sorted(
                self._envelope_cache.keys(),
                key=lambda k: self._envelope_cache[k].created_at
            )[:100]
            for envelope_id in oldest_ids:
                del self._envelope_cache[envelope_id]

    async def _publish_envelope_event(
        self,
        envelope: EvidenceEnvelope,
        event_type: str
    ):
        """Publish envelope event to Kafka"""

        event = {
            "event_type": f"evidence_envelope.{event_type}",
            "envelope_id": envelope.envelope_id,
            "proposal_id": envelope.proposal_id,
            "timestamp": datetime.utcnow().isoformat(),
            "data": envelope.get_summary()
        }

        await self.kafka_producer.publish_envelope_event(
            event_type=event_type,
            envelope_id=envelope.envelope_id,
            envelope_data=envelope.get_summary()
        )

    async def _store_audit_record(
        self,
        audit_record: Dict[str, Any]
    ):
        """Store audit record for compliance"""

        # Store audit record in database
        await self._store_audit_record_db(audit_record)

        # Also send to audit logging service
        await self.kafka_producer.publish_audit_event(
            event_type="audit_record_created",
            envelope_id=audit_record.get("envelope_id"),
            user_id=None,
            details=audit_record
        )

    # Database operations using MongoDB

    async def _persist_envelope(self, envelope: EvidenceEnvelope):
        """Persist envelope to MongoDB"""
        try:
            collection = mongodb.get_collection("envelopes")
            envelope_doc = envelope.model_dump()
            envelope_doc["_id"] = envelope.envelope_id

            await collection.insert_one(envelope_doc)

            logger.debug(
                "envelope_persisted",
                envelope_id=envelope.envelope_id
            )
        except Exception as e:
            logger.error(
                "envelope_persistence_failed",
                envelope_id=envelope.envelope_id,
                error=str(e)
            )
            raise

    async def _update_envelope(self, envelope: EvidenceEnvelope):
        """Update envelope in MongoDB"""
        try:
            collection = mongodb.get_collection("envelopes")
            envelope_doc = envelope.model_dump()
            envelope_doc["_id"] = envelope.envelope_id

            await collection.replace_one(
                {"_id": envelope.envelope_id},
                envelope_doc
            )

            logger.debug(
                "envelope_updated",
                envelope_id=envelope.envelope_id
            )
        except Exception as e:
            logger.error(
                "envelope_update_failed",
                envelope_id=envelope.envelope_id,
                error=str(e)
            )
            raise

    async def _fetch_envelope(self, envelope_id: str) -> Optional[EvidenceEnvelope]:
        """Fetch envelope from MongoDB"""
        try:
            collection = mongodb.get_collection("envelopes")
            envelope_doc = await collection.find_one({"_id": envelope_id})

            if not envelope_doc:
                return None

            # Remove MongoDB _id field for Pydantic parsing
            envelope_doc.pop("_id", None)

            return EvidenceEnvelope.model_validate(envelope_doc)

        except Exception as e:
            logger.error(
                "envelope_fetch_failed",
                envelope_id=envelope_id,
                error=str(e)
            )
            return None

    async def _query_envelopes(self, **filters) -> List[EvidenceEnvelope]:
        """Query envelopes from MongoDB"""
        try:
            collection = mongodb.get_collection("envelopes")

            # Build query filter
            query = {}

            if filters.get("proposal_id"):
                query["proposal_id"] = filters["proposal_id"]

            if filters.get("patient_id"):
                query["clinical_context.patient_id"] = filters["patient_id"]

            if filters.get("workflow_type"):
                query["clinical_context.workflow_type"] = filters["workflow_type"]

            if filters.get("status"):
                query["status"] = filters["status"]

            if filters.get("start_date") or filters.get("end_date"):
                date_filter = {}
                if filters.get("start_date"):
                    date_filter["$gte"] = filters["start_date"]
                if filters.get("end_date"):
                    date_filter["$lte"] = filters["end_date"]
                query["created_at"] = date_filter

            # Execute query with limit
            cursor = collection.find(query).limit(filters.get("limit", 100))
            cursor = cursor.sort("created_at", -1)  # Most recent first

            envelopes = []
            async for doc in cursor:
                doc.pop("_id", None)  # Remove MongoDB _id
                try:
                    envelope = EvidenceEnvelope.model_validate(doc)
                    envelopes.append(envelope)
                except Exception as e:
                    logger.warning(
                        "envelope_deserialization_failed",
                        envelope_id=doc.get("envelope_id"),
                        error=str(e)
                    )
                    continue

            return envelopes

        except Exception as e:
            logger.error(
                "envelope_query_failed",
                error=str(e),
                filters=filters
            )
            return []

    async def _store_audit_record_db(self, audit_record: Dict[str, Any]):
        """Store audit record in MongoDB"""
        try:
            collection = mongodb.get_collection("audit_records")
            audit_record["_id"] = f"audit_{audit_record['envelope_id']}_{int(datetime.utcnow().timestamp() * 1000)}"
            audit_record["stored_at"] = datetime.utcnow()

            await collection.insert_one(audit_record)

            logger.debug(
                "audit_record_stored",
                envelope_id=audit_record.get("envelope_id")
            )
        except Exception as e:
            logger.error(
                "audit_record_storage_failed",
                envelope_id=audit_record.get("envelope_id"),
                error=str(e)
            )
            # Don't raise - audit storage failure shouldn't break main flow


# Singleton instance
envelope_manager = EvidenceEnvelopeManager()