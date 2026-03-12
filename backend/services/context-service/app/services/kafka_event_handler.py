"""
Kafka Event Handler for Clinical Context Cache Invalidation
Implements event-driven cache invalidation based on clinical data changes
"""
import logging
import json
import asyncio
from typing import Dict, List, Optional, Any, Callable
from datetime import datetime
from kafka import KafkaConsumer, KafkaProducer
from kafka.errors import KafkaError
import threading

from app.services.cache_service import CacheService

logger = logging.getLogger(__name__)


class ClinicalDataChangeEvent:
    """Represents a clinical data change event"""
    
    def __init__(self, event_data: Dict[str, Any]):
        self.event_type = event_data.get("event_type", "unknown")
        self.patient_id = event_data.get("patient_id")
        self.resource_type = event_data.get("resource_type")
        self.resource_id = event_data.get("resource_id")
        self.change_type = event_data.get("change_type", "update")  # create, update, delete
        self.timestamp = datetime.fromisoformat(event_data.get("timestamp", datetime.utcnow().isoformat()))
        self.source_service = event_data.get("source_service")
        self.affected_fields = event_data.get("affected_fields", [])
        self.metadata = event_data.get("metadata", {})
    
    def __str__(self):
        return f"ClinicalDataChangeEvent({self.event_type}, {self.patient_id}, {self.resource_type})"


class KafkaEventHandler:
    """
    Handles Kafka events for clinical data changes and cache invalidation.
    Implements event-driven cache invalidation patterns.
    """
    
    def __init__(self, cache_service: CacheService):
        self.cache_service = cache_service
        
        # Kafka configuration
        self.kafka_config = {
            "bootstrap_servers": ["pkc-619z3.us-east1.gcp.confluent.cloud:9092"],
            "security_protocol": "SASL_SSL",
            "sasl_mechanism": "PLAIN",
            "sasl_plain_username": "LGJ3AQ2L6VRPW4S2",
            "sasl_plain_password": "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
        }
        
        # Topics to subscribe to
        self.subscribed_topics = [
            "clinical-data-changes",
            "patient-data-updates",
            "medication-changes",
            "allergy-updates",
            "condition-updates",
            "lab-results-updates",
            "encounter-updates"
        ]
        
        # Event handlers mapping
        self.event_handlers: Dict[str, Callable] = {
            "patient.demographics.updated": self._handle_patient_demographics_update,
            "patient.medication.added": self._handle_medication_change,
            "patient.medication.updated": self._handle_medication_change,
            "patient.medication.removed": self._handle_medication_change,
            "patient.allergy.added": self._handle_allergy_change,
            "patient.allergy.updated": self._handle_allergy_change,
            "patient.allergy.removed": self._handle_allergy_change,
            "patient.condition.added": self._handle_condition_change,
            "patient.condition.updated": self._handle_condition_change,
            "patient.lab.updated": self._handle_lab_results_update,
            "patient.encounter.started": self._handle_encounter_change,
            "patient.encounter.updated": self._handle_encounter_change,
            "patient.encounter.ended": self._handle_encounter_change,
            "patient.vitals.updated": self._handle_vitals_update
        }
        
        # Consumer and producer
        self.consumer = None
        self.producer = None
        self.consumer_thread = None
        self.running = False
        
        # Event processing statistics
        self.event_stats = {
            "events_processed": 0,
            "cache_invalidations": 0,
            "processing_errors": 0,
            "last_event_time": None,
            "events_by_type": {}
        }
    
    async def start(self):
        """Start the Kafka event handler"""
        try:
            logger.info("🚀 Starting Kafka event handler for cache invalidation")
            
            # Initialize Kafka consumer
            self.consumer = KafkaConsumer(
                *self.subscribed_topics,
                **self.kafka_config,
                group_id="context-service-cache-invalidation",
                auto_offset_reset="latest",
                enable_auto_commit=True,
                value_deserializer=lambda x: json.loads(x.decode('utf-8')) if x else None
            )
            
            # Initialize Kafka producer for audit events
            self.producer = KafkaProducer(
                **self.kafka_config,
                value_serializer=lambda x: json.dumps(x).encode('utf-8')
            )
            
            # Start consumer in background thread
            self.running = True
            self.consumer_thread = threading.Thread(target=self._consume_events, daemon=True)
            self.consumer_thread.start()
            
            logger.info("✅ Kafka event handler started successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to start Kafka event handler: {e}")
            raise
    
    async def stop(self):
        """Stop the Kafka event handler"""
        try:
            logger.info("🛑 Stopping Kafka event handler")
            
            self.running = False
            
            if self.consumer:
                self.consumer.close()
            
            if self.producer:
                self.producer.close()
            
            if self.consumer_thread and self.consumer_thread.is_alive():
                self.consumer_thread.join(timeout=5)
            
            logger.info("✅ Kafka event handler stopped")
            
        except Exception as e:
            logger.error(f"❌ Error stopping Kafka event handler: {e}")
    
    def _consume_events(self):
        """Background thread for consuming Kafka events"""
        logger.info("📡 Starting Kafka event consumption")
        
        try:
            for message in self.consumer:
                if not self.running:
                    break
                
                try:
                    # Process the event
                    asyncio.run(self._process_event(message))
                    
                except Exception as e:
                    logger.error(f"❌ Error processing Kafka event: {e}")
                    self.event_stats["processing_errors"] += 1
                    
        except Exception as e:
            logger.error(f"❌ Kafka consumer error: {e}")
        
        logger.info("📡 Kafka event consumption stopped")
    
    async def _process_event(self, message):
        """Process a single Kafka event"""
        try:
            # Parse event data
            event_data = message.value
            if not event_data:
                return
            
            # Create event object
            event = ClinicalDataChangeEvent(event_data)
            
            logger.debug(f"📨 Processing event: {event}")
            
            # Update statistics
            self.event_stats["events_processed"] += 1
            self.event_stats["last_event_time"] = datetime.utcnow()
            
            event_type = event.event_type
            if event_type not in self.event_stats["events_by_type"]:
                self.event_stats["events_by_type"][event_type] = 0
            self.event_stats["events_by_type"][event_type] += 1
            
            # Route to appropriate handler
            if event_type in self.event_handlers:
                await self.event_handlers[event_type](event)
            else:
                # Generic handler for unknown event types
                await self._handle_generic_data_change(event)
            
            # Publish cache invalidation audit event
            await self._publish_cache_invalidation_audit(event)
            
        except Exception as e:
            logger.error(f"❌ Error processing event: {e}")
            self.event_stats["processing_errors"] += 1
    
    async def _handle_patient_demographics_update(self, event: ClinicalDataChangeEvent):
        """Handle patient demographics updates"""
        logger.info(f"👤 Handling patient demographics update: {event.patient_id}")
        
        # Invalidate all contexts for this patient
        await self.cache_service.invalidate_patient_contexts(event.patient_id)
        
        # Specific invalidation for demographics-dependent recipes
        demographics_dependent_recipes = [
            "medication_prescribing_v2",
            "routine_medication_refill_v1",
            "base_clinical_context_v1"
        ]
        
        for recipe_id in demographics_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Demographics cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_medication_change(self, event: ClinicalDataChangeEvent):
        """Handle medication-related changes"""
        logger.info(f"💊 Handling medication change: {event.patient_id} - {event.change_type}")
        
        # Invalidate medication-related contexts
        medication_dependent_recipes = [
            "medication_prescribing_v2",
            "routine_medication_refill_v1",
            "clinical_deterioration_response_v1"
        ]
        
        for recipe_id in medication_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        # Also invalidate provider-specific contexts if provider info available
        if "provider_id" in event.metadata:
            provider_id = event.metadata["provider_id"]
            for recipe_id in medication_dependent_recipes:
                cache_key = f"context:{event.patient_id}:{recipe_id}:{provider_id}"
                await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Medication cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_allergy_change(self, event: ClinicalDataChangeEvent):
        """Handle allergy-related changes"""
        logger.info(f"🚨 Handling allergy change: {event.patient_id} - {event.change_type}")
        
        # Allergy changes are critical for medication prescribing
        allergy_dependent_recipes = [
            "medication_prescribing_v2",
            "base_clinical_context_v1"
        ]
        
        for recipe_id in allergy_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Allergy cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_condition_change(self, event: ClinicalDataChangeEvent):
        """Handle condition/diagnosis changes"""
        logger.info(f"🏥 Handling condition change: {event.patient_id} - {event.change_type}")
        
        # Conditions affect multiple contexts
        condition_dependent_recipes = [
            "medication_prescribing_v2",
            "clinical_deterioration_response_v1",
            "base_clinical_context_v1"
        ]
        
        for recipe_id in condition_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Condition cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_lab_results_update(self, event: ClinicalDataChangeEvent):
        """Handle lab results updates"""
        logger.info(f"🧪 Handling lab results update: {event.patient_id}")
        
        # Lab results affect medication dosing and clinical deterioration
        lab_dependent_recipes = [
            "medication_prescribing_v2",
            "clinical_deterioration_response_v1"
        ]
        
        for recipe_id in lab_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Lab results cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_encounter_change(self, event: ClinicalDataChangeEvent):
        """Handle encounter-related changes"""
        logger.info(f"🏥 Handling encounter change: {event.patient_id} - {event.change_type}")
        
        # Encounter changes affect context assembly
        encounter_dependent_recipes = [
            "clinical_deterioration_response_v1",
            "base_clinical_context_v1"
        ]
        
        for recipe_id in encounter_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Encounter cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_vitals_update(self, event: ClinicalDataChangeEvent):
        """Handle vital signs updates"""
        logger.info(f"📊 Handling vitals update: {event.patient_id}")
        
        # Vitals are critical for deterioration detection
        vitals_dependent_recipes = [
            "clinical_deterioration_response_v1"
        ]
        
        for recipe_id in vitals_dependent_recipes:
            cache_key = f"context:{event.patient_id}:{recipe_id}"
            await self.cache_service.invalidate(cache_key)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Vitals cache invalidation complete for patient {event.patient_id}")
    
    async def _handle_generic_data_change(self, event: ClinicalDataChangeEvent):
        """Handle generic data changes for unknown event types"""
        logger.info(f"🔄 Handling generic data change: {event.patient_id} - {event.event_type}")
        
        # For unknown events, invalidate all contexts for the patient
        await self.cache_service.invalidate_patient_contexts(event.patient_id)
        
        self.event_stats["cache_invalidations"] += 1
        logger.debug(f"✅ Generic cache invalidation complete for patient {event.patient_id}")
    
    async def _publish_cache_invalidation_audit(self, event: ClinicalDataChangeEvent):
        """Publish cache invalidation audit event"""
        try:
            audit_event = {
                "event_type": "cache.invalidation.completed",
                "patient_id": event.patient_id,
                "original_event_type": event.event_type,
                "timestamp": datetime.utcnow().isoformat(),
                "source_service": "context-service",
                "invalidation_scope": "patient_contexts",
                "metadata": {
                    "original_source": event.source_service,
                    "processing_time_ms": 0,  # Would calculate actual processing time
                    "affected_recipes": []  # Would list affected recipes
                }
            }
            
            # Publish to audit topic
            if self.producer:
                self.producer.send("cache-invalidation-audit", audit_event)
            
        except Exception as e:
            logger.warning(f"⚠️ Failed to publish cache invalidation audit: {e}")
    
    async def get_event_statistics(self) -> Dict[str, Any]:
        """Get event processing statistics"""
        return {
            "events_processed": self.event_stats["events_processed"],
            "cache_invalidations": self.event_stats["cache_invalidations"],
            "processing_errors": self.event_stats["processing_errors"],
            "last_event_time": self.event_stats["last_event_time"].isoformat() if self.event_stats["last_event_time"] else None,
            "events_by_type": self.event_stats["events_by_type"],
            "subscribed_topics": self.subscribed_topics,
            "running": self.running
        }
    
    async def publish_context_assembled_event(self, context_id: str, patient_id: str, recipe_id: str):
        """Publish event when context is assembled"""
        try:
            event = {
                "event_type": "context.assembled",
                "context_id": context_id,
                "patient_id": patient_id,
                "recipe_id": recipe_id,
                "timestamp": datetime.utcnow().isoformat(),
                "source_service": "context-service",
                "metadata": {
                    "cache_populated": True,
                    "assembly_successful": True
                }
            }
            
            if self.producer:
                self.producer.send("context-assembly-events", event)
                logger.debug(f"📤 Published context assembled event: {context_id}")
            
        except Exception as e:
            logger.warning(f"⚠️ Failed to publish context assembled event: {e}")


class CacheInvalidationService:
    """
    Service for managing cache invalidation based on clinical data changes.
    Integrates with Kafka event handler for event-driven invalidation.
    """
    
    def __init__(self, cache_service: CacheService):
        self.cache_service = cache_service
        self.kafka_handler = KafkaEventHandler(cache_service)
        self.running = False
    
    async def start(self):
        """Start the cache invalidation service"""
        try:
            logger.info("🚀 Starting cache invalidation service")
            
            await self.kafka_handler.start()
            self.running = True
            
            logger.info("✅ Cache invalidation service started")
            
        except Exception as e:
            logger.error(f"❌ Failed to start cache invalidation service: {e}")
            raise
    
    async def stop(self):
        """Stop the cache invalidation service"""
        try:
            logger.info("🛑 Stopping cache invalidation service")
            
            await self.kafka_handler.stop()
            self.running = False
            
            logger.info("✅ Cache invalidation service stopped")
            
        except Exception as e:
            logger.error(f"❌ Error stopping cache invalidation service: {e}")
    
    async def manual_invalidate_patient(self, patient_id: str, reason: str = "manual"):
        """Manually invalidate all contexts for a patient"""
        logger.info(f"🔧 Manual cache invalidation for patient {patient_id}: {reason}")
        
        await self.cache_service.invalidate_patient_contexts(patient_id)
        
        # Publish manual invalidation event
        await self.kafka_handler.publish_context_assembled_event(
            context_id=f"manual_invalidation_{patient_id}",
            patient_id=patient_id,
            recipe_id="all"
        )
    
    async def get_invalidation_statistics(self) -> Dict[str, Any]:
        """Get cache invalidation statistics"""
        kafka_stats = await self.kafka_handler.get_event_statistics()
        cache_stats = await self.cache_service.get_cache_stats()
        
        return {
            "service_running": self.running,
            "kafka_events": kafka_stats,
            "cache_performance": cache_stats,
            "invalidation_summary": {
                "total_events_processed": kafka_stats["events_processed"],
                "total_invalidations": kafka_stats["cache_invalidations"],
                "error_rate": kafka_stats["processing_errors"] / max(kafka_stats["events_processed"], 1),
                "cache_hit_ratio": cache_stats["overall_hit_ratio"]
            }
        }
