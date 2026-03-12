"""
Dead Letter Queue Service for Stage 2: Storage Fan-Out

Handles sink write failures, FHIR transformation errors, and other processing
failures with comprehensive error categorization and routing.
"""

import json
import time
from datetime import datetime
from typing import Dict, Any, Optional, List
from enum import Enum

import structlog
from kafka import KafkaProducer

from app.config import settings, get_kafka_config

logger = structlog.get_logger(__name__)


class DLQErrorType(Enum):
    """DLQ Error Types"""
    FHIR_TRANSFORMATION_FAILURE = "FHIR_TRANSFORMATION_FAILURE"
    SINK_WRITE_FAILURE = "SINK_WRITE_FAILURE"
    FHIR_STORE_FAILURE = "FHIR_STORE_FAILURE"
    ELASTICSEARCH_FAILURE = "ELASTICSEARCH_FAILURE"
    MONGODB_FAILURE = "MONGODB_FAILURE"
    CIRCUIT_BREAKER_OPEN = "CIRCUIT_BREAKER_OPEN"
    TIMEOUT_FAILURE = "TIMEOUT_FAILURE"
    POISON_MESSAGE = "POISON_MESSAGE"
    UNKNOWN_ERROR = "UNKNOWN_ERROR"


class DLQService:
    """
    Dead Letter Queue Service for Stage 2
    
    Handles all types of failures in the storage fan-out process with
    proper categorization, routing, and retry logic.
    """
    
    def __init__(self):
        self.service_name = "stage2-storage-fanout"
        self.producer = None
        self.dlq_topics = {
            "sink_failures": settings.KAFKA_DLQ_TOPIC,
            "critical_failures": "critical-sink-failures.v1",
            "poison_messages": "poison-messages-stage2.v1"
        }
        
        # Metrics
        self.total_dlq_messages = 0
        self.sink_failures = 0
        self.fhir_failures = 0
        self.elasticsearch_failures = 0
        self.mongodb_failures = 0
        self.poison_messages = 0
        
        logger.info("DLQ Service initialized", topics=self.dlq_topics)
    
    async def initialize(self):
        """Initialize Kafka producer for DLQ"""
        try:
            kafka_config = get_kafka_config()
            # Remove consumer-specific configs
            producer_config = {
                'bootstrap_servers': kafka_config['bootstrap_servers'],
                'security_protocol': kafka_config['security_protocol'],
                'sasl_mechanism': kafka_config['sasl_mechanism'],
                'sasl_plain_username': kafka_config['sasl_plain_username'],
                'sasl_plain_password': kafka_config['sasl_plain_password'],
                'value_serializer': lambda x: json.dumps(x).encode('utf-8'),
                'key_serializer': lambda x: x.encode('utf-8') if x else None,
                'acks': 'all',
                'retries': 3,
                'retry_backoff_ms': 1000
            }
            
            self.producer = KafkaProducer(**producer_config)
            logger.info("DLQ Kafka producer initialized")
            
        except Exception as e:
            logger.error("Failed to initialize DLQ Kafka producer", error=str(e))
            raise
    
    async def send_fhir_transformation_failure(self, original_data: Dict[str, Any], 
                                             error: Exception, device_id: str = None):
        """Send FHIR transformation failure to DLQ"""
        dlq_record = self._create_dlq_record(
            original_data=original_data,
            error_type=DLQErrorType.FHIR_TRANSFORMATION_FAILURE,
            error_message=f"FHIR transformation failed: {str(error)}",
            device_id=device_id,
            error_details={
                "exception_type": type(error).__name__,
                "exception_message": str(error),
                "transformation_stage": "fhir_observation_creation"
            }
        )
        
        await self._send_to_dlq(dlq_record, device_id)
        self.fhir_failures += 1
        
        logger.error("FHIR transformation failure sent to DLQ", 
                    device_id=device_id, error=str(error))
    
    async def send_sink_write_failure(self, original_data: Dict[str, Any], 
                                    sink_name: str, error: Exception, 
                                    device_id: str = None, is_critical: bool = False):
        """Send sink write failure to DLQ"""
        error_type = self._get_sink_error_type(sink_name)
        
        dlq_record = self._create_dlq_record(
            original_data=original_data,
            error_type=error_type,
            error_message=f"{sink_name} write failed: {str(error)}",
            device_id=device_id,
            is_critical=is_critical,
            retryable=True,
            max_retries=3,
            error_details={
                "sink_name": sink_name,
                "exception_type": type(error).__name__,
                "exception_message": str(error),
                "is_critical_data": is_critical
            }
        )
        
        # Route critical failures to special topic
        topic = self.dlq_topics["critical_failures"] if is_critical else self.dlq_topics["sink_failures"]
        await self._send_to_dlq(dlq_record, device_id, topic)
        
        self.sink_failures += 1
        self._update_sink_failure_metrics(sink_name)
        
        logger.error("Sink write failure sent to DLQ", 
                    sink_name=sink_name, device_id=device_id, 
                    is_critical=is_critical, error=str(error))
    
    async def send_circuit_breaker_failure(self, original_data: Dict[str, Any], 
                                         sink_name: str, device_id: str = None):
        """Send circuit breaker failure to DLQ"""
        dlq_record = self._create_dlq_record(
            original_data=original_data,
            error_type=DLQErrorType.CIRCUIT_BREAKER_OPEN,
            error_message=f"Circuit breaker open for {sink_name}",
            device_id=device_id,
            retryable=True,
            max_retries=5,
            error_details={
                "sink_name": sink_name,
                "circuit_breaker_state": "OPEN",
                "retry_after_seconds": 60
            }
        )
        
        await self._send_to_dlq(dlq_record, device_id)
        
        logger.warning("Circuit breaker failure sent to DLQ", 
                      sink_name=sink_name, device_id=device_id)
    
    async def send_timeout_failure(self, original_data: Dict[str, Any], 
                                 sink_name: str, timeout_seconds: int, 
                                 device_id: str = None):
        """Send timeout failure to DLQ"""
        dlq_record = self._create_dlq_record(
            original_data=original_data,
            error_type=DLQErrorType.TIMEOUT_FAILURE,
            error_message=f"{sink_name} write timed out after {timeout_seconds}s",
            device_id=device_id,
            retryable=True,
            max_retries=2,
            error_details={
                "sink_name": sink_name,
                "timeout_seconds": timeout_seconds,
                "suggested_timeout": timeout_seconds * 2
            }
        )
        
        await self._send_to_dlq(dlq_record, device_id)
        
        logger.warning("Timeout failure sent to DLQ", 
                      sink_name=sink_name, timeout_seconds=timeout_seconds, 
                      device_id=device_id)
    
    async def send_poison_message(self, original_data: Dict[str, Any], 
                                reason: str, retry_count: int, 
                                device_id: str = None):
        """Send poison message to special DLQ"""
        dlq_record = self._create_dlq_record(
            original_data=original_data,
            error_type=DLQErrorType.POISON_MESSAGE,
            error_message=f"Poison message after {retry_count} retries: {reason}",
            device_id=device_id,
            retryable=False,
            error_details={
                "retry_count": retry_count,
                "reason": reason,
                "requires_manual_review": True,
                "poison_message_timestamp": time.time()
            }
        )
        
        await self._send_to_dlq(dlq_record, device_id, self.dlq_topics["poison_messages"])
        self.poison_messages += 1
        
        logger.error("Poison message sent to DLQ", 
                    retry_count=retry_count, reason=reason, device_id=device_id)
    
    def _create_dlq_record(self, original_data: Dict[str, Any], 
                          error_type: DLQErrorType, error_message: str,
                          device_id: str = None, is_critical: bool = False,
                          retryable: bool = False, max_retries: int = 0,
                          error_details: Dict[str, Any] = None) -> Dict[str, Any]:
        """Create standardized DLQ record"""
        return {
            "original_data": original_data,
            "error_type": error_type.value,
            "error_message": error_message,
            "device_id": device_id,
            "patient_id": original_data.get("patient_id") if original_data else None,
            "failure_timestamp": time.time(),
            "failure_datetime": datetime.utcnow().isoformat() + "Z",
            "processing_stage": self.service_name,
            "is_critical_data": is_critical,
            "retryable": retryable,
            "max_retries": max_retries,
            "retry_count": 0,
            "error_details": error_details or {},
            "dlq_version": "1.0"
        }
    
    async def _send_to_dlq(self, dlq_record: Dict[str, Any], key: str = None, 
                          topic: str = None):
        """Send record to appropriate DLQ topic"""
        try:
            if not self.producer:
                logger.error("DLQ producer not initialized")
                return
            
            topic = topic or self.dlq_topics["sink_failures"]
            key = key or dlq_record.get("device_id", "unknown")
            
            # Send to Kafka
            future = self.producer.send(topic, key=key, value=dlq_record)
            
            # Wait for send to complete (with timeout)
            future.get(timeout=10)
            
            self.total_dlq_messages += 1
            
            logger.debug("DLQ record sent successfully", 
                        topic=topic, key=key, error_type=dlq_record["error_type"])
            
        except Exception as e:
            logger.error("Failed to send DLQ record", error=str(e), 
                        topic=topic, key=key)
    
    def _get_sink_error_type(self, sink_name: str) -> DLQErrorType:
        """Get specific error type based on sink name"""
        sink_error_mapping = {
            "fhir_store": DLQErrorType.FHIR_STORE_FAILURE,
            "elasticsearch": DLQErrorType.ELASTICSEARCH_FAILURE,
            "mongodb": DLQErrorType.MONGODB_FAILURE
        }
        return sink_error_mapping.get(sink_name.lower(), DLQErrorType.SINK_WRITE_FAILURE)
    
    def _update_sink_failure_metrics(self, sink_name: str):
        """Update sink-specific failure metrics"""
        if "fhir" in sink_name.lower():
            self.fhir_failures += 1
        elif "elasticsearch" in sink_name.lower():
            self.elasticsearch_failures += 1
        elif "mongodb" in sink_name.lower():
            self.mongodb_failures += 1
    
    def get_dlq_metrics(self) -> Dict[str, int]:
        """Get DLQ metrics"""
        return {
            "total_dlq_messages": self.total_dlq_messages,
            "sink_failures": self.sink_failures,
            "fhir_failures": self.fhir_failures,
            "elasticsearch_failures": self.elasticsearch_failures,
            "mongodb_failures": self.mongodb_failures,
            "poison_messages": self.poison_messages
        }
    
    def is_healthy(self) -> bool:
        """Check if DLQ service is healthy"""
        return self.producer is not None
    
    async def close(self):
        """Close DLQ service and cleanup resources"""
        if self.producer:
            self.producer.close()
            logger.info("DLQ service closed")
