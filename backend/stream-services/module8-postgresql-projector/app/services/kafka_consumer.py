"""
Kafka Consumer Service wrapper
Provides interface for FastAPI to interact with projector
"""
import threading
from typing import Dict, Any, Optional
from datetime import datetime
import structlog

from app.services.projector import PostgreSQLProjector

logger = structlog.get_logger(__name__)


class KafkaConsumerService:
    """
    Service wrapper for PostgreSQL projector
    Manages consumer lifecycle and provides status endpoints
    """

    def __init__(self, kafka_config: Dict[str, Any], postgres_config: Dict[str, Any]):
        self.projector = PostgreSQLProjector(
            kafka_config=kafka_config,
            postgres_config=postgres_config,
        )
        self.consumer_thread: Optional[threading.Thread] = None
        self.running = False
        self.start_time: Optional[datetime] = None

    def start(self) -> None:
        """Start consumer in background thread"""
        if self.running:
            logger.warning("Consumer already running")
            return

        logger.info("Starting consumer service")
        self.running = True
        self.start_time = datetime.utcnow()

        self.consumer_thread = threading.Thread(
            target=self._run_consumer,
            daemon=True,
            name="kafka-consumer-thread",
        )
        self.consumer_thread.start()

        logger.info("Consumer service started")

    def _run_consumer(self) -> None:
        """Run consumer in thread"""
        try:
            self.projector.start()
        except Exception as e:
            logger.error("Consumer thread error", error=str(e), exc_info=True)
            self.running = False

    def shutdown(self) -> None:
        """Gracefully shutdown consumer"""
        if not self.running:
            logger.warning("Consumer not running")
            return

        logger.info("Shutting down consumer service")
        self.running = False
        self.projector.shutdown()

        if self.consumer_thread:
            self.consumer_thread.join(timeout=10)

        logger.info("Consumer service shutdown complete")

    def get_status(self) -> Dict[str, Any]:
        """Get current service status"""
        uptime = 0.0
        if self.start_time:
            uptime = (datetime.utcnow() - self.start_time).total_seconds()

        return {
            "running": self.running,
            "uptime_seconds": uptime,
            "metrics": self.projector.get_metrics(),
            "last_processed": self.projector.last_processed_time,
        }

    def is_healthy(self) -> bool:
        """Check if service is healthy"""
        return self.running and self.projector.is_running()

    def get_metrics(self) -> Dict[str, Any]:
        """Get Prometheus-compatible metrics"""
        return self.projector.get_metrics()
