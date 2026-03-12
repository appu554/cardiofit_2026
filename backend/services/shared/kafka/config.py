"""
Kafka configuration for Confluent Cloud integration
"""

import os
from typing import Dict, Any, Optional
from dataclasses import dataclass
import logging

logger = logging.getLogger(__name__)

@dataclass
class KafkaConfig:
    """Kafka configuration for Confluent Cloud"""
    
    # Confluent Cloud credentials
    bootstrap_servers: str = "pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    api_key: str = "LGJ3AQ2L6VRPW4S2"
    api_secret: str = "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
    resource_id: str = "lkc-x86njx"
    
    # Security configuration
    security_protocol: str = "SASL_SSL"
    sasl_mechanism: str = "PLAIN"
    
    # Producer configuration
    producer_config: Optional[Dict[str, Any]] = None
    
    # Consumer configuration  
    consumer_config: Optional[Dict[str, Any]] = None
    
    # Schema Registry (if using Confluent Schema Registry)
    schema_registry_url: Optional[str] = None
    schema_registry_api_key: Optional[str] = None
    schema_registry_api_secret: Optional[str] = None
    
    def __post_init__(self):
        """Initialize default configurations"""
        if self.producer_config is None:
            self.producer_config = self._get_default_producer_config()
            
        if self.consumer_config is None:
            self.consumer_config = self._get_default_consumer_config()
    
    def _get_default_producer_config(self) -> Dict[str, Any]:
        """Get default producer configuration"""
        return {
            'bootstrap.servers': self.bootstrap_servers,
            'security.protocol': self.security_protocol,
            'sasl.mechanism': self.sasl_mechanism,
            'sasl.username': self.api_key,
            'sasl.password': self.api_secret,
            
            # Producer-specific settings
            'acks': 'all',  # Wait for all replicas to acknowledge
            'retries': 10,  # Retry failed sends
            'retry.backoff.ms': 1000,  # Wait between retries
            'delivery.timeout.ms': 300000,  # 5 minutes total timeout
            'request.timeout.ms': 30000,  # 30 seconds per request
            'max.in.flight.requests.per.connection': 5,  # Pipeline requests
            'enable.idempotence': True,  # Prevent duplicate messages
            
            # Compression and batching
            'compression.type': 'snappy',  # Compress messages
            'batch.size': 16384,  # Batch size in bytes
            'linger.ms': 10,  # Wait time to batch messages
            
            # Monitoring
            'statistics.interval.ms': 10000,  # Stats every 10 seconds
        }
    
    def _get_default_consumer_config(self) -> Dict[str, Any]:
        """Get default consumer configuration"""
        return {
            'bootstrap.servers': self.bootstrap_servers,
            'security.protocol': self.security_protocol,
            'sasl.mechanism': self.sasl_mechanism,
            'sasl.username': self.api_key,
            'sasl.password': self.api_secret,
            
            # Consumer-specific settings
            'group.id': 'clinical-synthesis-hub',  # Default group
            'auto.offset.reset': 'earliest',  # Start from beginning if no offset
            'enable.auto.commit': False,  # Manual commit for reliability
            'max.poll.interval.ms': 300000,  # 5 minutes max processing time
            'session.timeout.ms': 30000,  # 30 seconds session timeout
            'heartbeat.interval.ms': 10000,  # 10 seconds heartbeat
            
            # Fetch settings
            'fetch.min.bytes': 1,  # Minimum bytes to fetch
            'fetch.wait.max.ms': 500,  # Max wait for min bytes (correct property name)
            'max.partition.fetch.bytes': 1048576,  # 1MB max per partition
            
            # Monitoring
            'statistics.interval.ms': 10000,  # Stats every 10 seconds
        }
    
    def get_producer_config(self, **overrides) -> Dict[str, Any]:
        """Get producer configuration with optional overrides"""
        config = self.producer_config.copy()
        config.update(overrides)
        return config
    
    def get_consumer_config(self, group_id: str = None, **overrides) -> Dict[str, Any]:
        """Get consumer configuration with optional overrides"""
        config = self.consumer_config.copy()
        if group_id:
            config['group.id'] = group_id
        config.update(overrides)
        return config
    
    @classmethod
    def from_env(cls) -> 'KafkaConfig':
        """Create configuration from environment variables"""
        return cls(
            bootstrap_servers=os.getenv('KAFKA_BOOTSTRAP_SERVERS', cls.bootstrap_servers),
            api_key=os.getenv('KAFKA_API_KEY', cls.api_key),
            api_secret=os.getenv('KAFKA_API_SECRET', cls.api_secret),
            resource_id=os.getenv('KAFKA_RESOURCE_ID', cls.resource_id),
            schema_registry_url=os.getenv('SCHEMA_REGISTRY_URL'),
            schema_registry_api_key=os.getenv('SCHEMA_REGISTRY_API_KEY'),
            schema_registry_api_secret=os.getenv('SCHEMA_REGISTRY_API_SECRET'),
        )

# Global configuration instance
kafka_config = KafkaConfig()

# Topic naming conventions
class TopicNames:
    """Standard topic names for the Clinical Synthesis Hub"""
    
    # Raw data ingestion
    RAW_DEVICE_DATA = "raw-device-data"
    RAW_IMAGING_DATA = "raw-imaging-data"
    RAW_DOCUMENT_DATA = "raw-document-data"
    
    # FHIR events
    FHIR_PATIENT_EVENTS = "fhir-patient-events"
    FHIR_ENCOUNTER_EVENTS = "fhir-encounter-events"
    FHIR_OBSERVATION_EVENTS = "fhir-observation-events"
    FHIR_MEDICATION_EVENTS = "fhir-medication-events"
    FHIR_ORDER_EVENTS = "fhir-order-events"
    FHIR_CONDITION_EVENTS = "fhir-condition-events"
    
    # Processed events
    CLINICAL_ALERTS = "clinical-alerts"
    WORKFLOW_EVENTS = "workflow-events"
    NOTIFICATION_EVENTS = "notification-events"
    
    # Read model updates
    READ_MODEL_UPDATES = "read-model-updates"
    SEARCH_INDEX_UPDATES = "search-index-updates"
    
    # Dead letter queues
    DLQ_PROCESSING_ERRORS = "dlq-processing-errors"
    DLQ_VALIDATION_ERRORS = "dlq-validation-errors"

# Event types
class EventTypes:
    """Standard event types for the Clinical Synthesis Hub"""
    
    # CRUD operations
    CREATED = "created"
    UPDATED = "updated"
    DELETED = "deleted"
    
    # Clinical events
    PATIENT_ADMITTED = "patient.admitted"
    PATIENT_DISCHARGED = "patient.discharged"
    OBSERVATION_RECORDED = "observation.recorded"
    MEDICATION_PRESCRIBED = "medication.prescribed"
    MEDICATION_ADMINISTERED = "medication.administered"
    ORDER_PLACED = "order.placed"
    ORDER_COMPLETED = "order.completed"
    
    # Workflow events
    WORKFLOW_STARTED = "workflow.started"
    WORKFLOW_COMPLETED = "workflow.completed"
    TASK_ASSIGNED = "task.assigned"
    TASK_COMPLETED = "task.completed"
    
    # Alert events
    CRITICAL_VALUE = "alert.critical_value"
    DRUG_INTERACTION = "alert.drug_interaction"
    ALLERGY_ALERT = "alert.allergy"
