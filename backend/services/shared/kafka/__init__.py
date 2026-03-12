"""
Kafka integration module for Clinical Synthesis Hub
Provides event-driven architecture capabilities using Confluent Cloud
"""

from .config import KafkaConfig
from .producer import EventProducer
from .consumer import EventConsumer
from .schemas import SchemaRegistry
from .monitoring import KafkaMonitor

__all__ = [
    'KafkaConfig',
    'EventProducer', 
    'EventConsumer',
    'SchemaRegistry',
    'KafkaMonitor'
]
