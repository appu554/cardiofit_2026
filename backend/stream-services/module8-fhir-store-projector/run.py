#!/usr/bin/env python3
"""
Run FHIR Store Projector

Starts Kafka consumer and FastAPI health server in separate threads
"""

import os
import sys
import threading
import logging
import structlog
import uvicorn

# Add module8-shared to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'module8-shared'))

from app.config import Config
from app.services.projector import FHIRStoreProjector
from app.main import app, set_projector_instance

# Configure structured logging
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(
        getattr(logging, Config.LOG_LEVEL.upper(), logging.INFO)
    ),
    context_class=dict,
    logger_factory=structlog.PrintLoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


def run_kafka_consumer(projector: FHIRStoreProjector):
    """Run Kafka consumer in background thread"""
    try:
        logger.info("Starting Kafka consumer")
        projector.start()
    except Exception as e:
        logger.error("Kafka consumer error", error=str(e), exc_info=True)
        sys.exit(1)


def run_health_server():
    """Run FastAPI health server"""
    try:
        logger.info(
            "Starting health server",
            port=Config.SERVICE_PORT,
        )
        uvicorn.run(
            app,
            host="0.0.0.0",
            port=Config.SERVICE_PORT,
            log_level=Config.LOG_LEVEL.lower(),
        )
    except Exception as e:
        logger.error("Health server error", error=str(e), exc_info=True)
        sys.exit(1)


def main():
    """Main entry point"""
    logger.info(
        "Starting FHIR Store Projector",
        config={
            "kafka_topic": Config.KAFKA_TOPIC_FHIR_UPSERT,
            "kafka_group_id": Config.KAFKA_GROUP_ID,
            "batch_size": Config.BATCH_SIZE,
            "batch_timeout": Config.BATCH_TIMEOUT_SECONDS,
            "fhir_store": Config.get_fhir_store_path(),
            "service_port": Config.SERVICE_PORT,
        },
    )

    # Build configuration
    config = {
        'kafka': Config.get_kafka_config(),
        'topics': {
            'fhir_upsert': Config.KAFKA_TOPIC_FHIR_UPSERT,
            'dlq': Config.KAFKA_TOPIC_DLQ,
        },
        'batch_size': Config.BATCH_SIZE,
        'batch_timeout_seconds': Config.BATCH_TIMEOUT_SECONDS,
        'fhir_store': {
            'project_id': Config.GOOGLE_CLOUD_PROJECT_ID,
            'location': Config.GOOGLE_CLOUD_LOCATION,
            'dataset_id': Config.GOOGLE_CLOUD_DATASET_ID,
            'store_id': Config.GOOGLE_CLOUD_FHIR_STORE_ID,
            'credentials_path': Config.GOOGLE_APPLICATION_CREDENTIALS,
            'max_retries': Config.RETRY_MAX_ATTEMPTS,
            'retry_backoff_factor': Config.RETRY_BACKOFF_FACTOR,
        },
    }

    # Initialize projector
    projector = FHIRStoreProjector(config)

    # Set global instance for health endpoints
    set_projector_instance(projector)

    # Start Kafka consumer in background thread
    consumer_thread = threading.Thread(
        target=run_kafka_consumer,
        args=(projector,),
        daemon=True,
    )
    consumer_thread.start()

    # Run health server in main thread (blocking)
    run_health_server()


if __name__ == "__main__":
    main()
