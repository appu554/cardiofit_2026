"""
Kafka Consumer Service for Neo4j Graph Projector
Wraps the projector with service management
"""
import threading
import time
from datetime import datetime
from typing import Optional

import structlog

from .projector import Neo4jGraphProjector

logger = structlog.get_logger(__name__)


class KafkaConsumerService:
    """
    Service wrapper for Neo4jGraphProjector
    Manages lifecycle and provides service status
    """

    def __init__(self, kafka_config: dict, neo4j_config: dict):
        self.kafka_config = kafka_config
        self.neo4j_config = neo4j_config
        self.projector: Optional[Neo4jGraphProjector] = None
        self.consumer_thread: Optional[threading.Thread] = None
        self.running = False
        self.start_time: Optional[datetime] = None

    def start(self) -> None:
        """Start the consumer service in a background thread"""
        if self.running:
            logger.warning("Service already running")
            return

        # Initialize projector
        self.projector = Neo4jGraphProjector(
            kafka_config=self.kafka_config,
            neo4j_config=self.neo4j_config,
        )

        # Start consumer in background thread
        self.consumer_thread = threading.Thread(
            target=self._run_consumer,
            daemon=True,
            name="neo4j-graph-projector"
        )
        self.consumer_thread.start()
        self.running = True
        self.start_time = datetime.utcnow()

        logger.info("Neo4j graph projector service started")

    def _run_consumer(self) -> None:
        """Run the consumer (called in background thread)"""
        try:
            if self.projector:
                self.projector.start()
        except Exception as e:
            logger.error("Consumer thread error", error=str(e), exc_info=True)
            self.running = False

    def shutdown(self) -> None:
        """Shutdown the service"""
        if not self.running:
            return

        logger.info("Shutting down service")
        self.running = False

        if self.projector:
            self.projector.shutdown()

        if self.consumer_thread and self.consumer_thread.is_alive():
            self.consumer_thread.join(timeout=10.0)

        logger.info("Service shutdown complete")

    def is_healthy(self) -> bool:
        """Check if service is healthy"""
        return self.running and self.consumer_thread and self.consumer_thread.is_alive()

    def get_metrics(self) -> dict:
        """Get service metrics"""
        if not self.projector:
            return {
                "messages_consumed": 0,
                "messages_processed": 0,
                "messages_failed": 0,
                "batches_processed": 0,
                "consumer_lag": 0,
            }

        metrics = self.projector.metrics

        return {
            "messages_consumed": metrics.messages_consumed._value.get(),
            "messages_processed": metrics.messages_processed._value.get(),
            "messages_failed": metrics.messages_failed._value.get(),
            "batches_processed": int(metrics.messages_processed._value.get() / self.projector.batch_size),
            "consumer_lag": metrics.consumer_lag._value.get(),
        }

    def get_status(self) -> dict:
        """Get detailed service status"""
        uptime_seconds = 0.0
        if self.start_time:
            uptime_seconds = (datetime.utcnow() - self.start_time).total_seconds()

        return {
            "running": self.running,
            "uptime_seconds": uptime_seconds,
            "metrics": self.get_metrics(),
            "last_processed": self.projector.last_processed_time if self.projector else None,
        }

    def get_graph_stats(self) -> dict:
        """Get Neo4j graph statistics"""
        if not self.projector:
            return {
                "node_counts": {},
                "relationship_count": 0,
                "total_nodes": 0,
            }

        return self.projector.get_graph_stats()

    def query_patient_journey(self, patient_id: str) -> list:
        """Query patient journey from Neo4j"""
        if not self.projector:
            return []

        return self.projector.query_patient_journey(patient_id)
