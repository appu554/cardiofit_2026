"""
Health Controller for Stage 2: Storage Fan-Out Service

Provides comprehensive health checks for all components including
sinks, DLQ service, and Kafka consumer.
"""

import time
from typing import Dict, Any

import structlog
from fastapi import APIRouter, HTTPException

from app.config import settings

logger = structlog.get_logger(__name__)

router = APIRouter()

# Global references to services (set by main.py)
kafka_consumer = None
fhir_transformer = None
multi_sink_writer = None


def set_service_references(consumer, transformer, writer):
    """Set global service references for health checks"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    kafka_consumer = consumer
    fhir_transformer = transformer
    multi_sink_writer = writer


@router.get("/")
async def overall_health():
    """Overall health check for Stage 2 service"""
    try:
        health_status = {
            "service": "stage2-storage-fanout",
            "port": settings.PORT,
            "status": "UP",
            "timestamp": int(time.time()),
            "components": {}
        }
        
        # Check each component
        components_healthy = True
        
        # Kafka Consumer health
        if kafka_consumer:
            consumer_healthy = kafka_consumer.is_healthy()
            health_status["components"]["kafka_consumer"] = {
                "status": "UP" if consumer_healthy else "DOWN",
                "details": kafka_consumer.get_metrics()
            }
            components_healthy = components_healthy and consumer_healthy
        else:
            health_status["components"]["kafka_consumer"] = {"status": "DOWN", "error": "Not initialized"}
            components_healthy = False
        
        # FHIR Transformer health
        if fhir_transformer:
            transformer_healthy = fhir_transformer.is_healthy()
            health_status["components"]["fhir_transformer"] = {
                "status": "UP" if transformer_healthy else "DOWN"
            }
            components_healthy = components_healthy and transformer_healthy
        else:
            health_status["components"]["fhir_transformer"] = {"status": "DOWN", "error": "Not initialized"}
            components_healthy = False
        
        # Multi-Sink Writer health
        if multi_sink_writer:
            writer_healthy = multi_sink_writer.is_healthy()
            health_status["components"]["multi_sink_writer"] = {
                "status": "UP" if writer_healthy else "DOWN",
                "details": multi_sink_writer.get_metrics()
            }
            components_healthy = components_healthy and writer_healthy
        else:
            health_status["components"]["multi_sink_writer"] = {"status": "DOWN", "error": "Not initialized"}
            components_healthy = False
        
        # Overall status
        health_status["status"] = "UP" if components_healthy else "DOWN"
        
        if components_healthy:
            return health_status
        else:
            raise HTTPException(status_code=503, detail=health_status)
            
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "service": "stage2-storage-fanout",
            "status": "DOWN",
            "error": str(e)
        })


@router.get("/kafka")
async def kafka_health():
    """Kafka consumer health check"""
    try:
        if not kafka_consumer:
            raise HTTPException(status_code=503, detail={
                "component": "kafka_consumer",
                "status": "DOWN",
                "error": "Kafka consumer not initialized"
            })
        
        is_healthy = kafka_consumer.is_healthy()
        metrics = kafka_consumer.get_metrics()
        lag_info = kafka_consumer.get_consumer_lag()
        
        health_data = {
            "component": "kafka_consumer",
            "status": "UP" if is_healthy else "DOWN",
            "metrics": metrics,
            "consumer_lag": lag_info,
            "input_topic": settings.KAFKA_INPUT_TOPIC,
            "consumer_group": settings.KAFKA_CONSUMER_GROUP
        }
        
        if is_healthy:
            return health_data
        else:
            raise HTTPException(status_code=503, detail=health_data)
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Kafka health check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "component": "kafka_consumer",
            "status": "DOWN",
            "error": str(e)
        })


@router.get("/sinks")
async def sinks_health():
    """Multi-sink writer health check"""
    try:
        if not multi_sink_writer:
            raise HTTPException(status_code=503, detail={
                "component": "multi_sink_writer",
                "status": "DOWN",
                "error": "Multi-sink writer not initialized"
            })
        
        is_healthy = multi_sink_writer.is_healthy()
        metrics = multi_sink_writer.get_metrics()
        
        # Get individual sink health
        sink_health = {}
        for sink_name, sink in multi_sink_writer.sinks.items():
            sink_health[sink_name] = {
                "status": "UP" if sink.is_healthy() else "DOWN",
                "metrics": sink.get_metrics()
            }
        
        health_data = {
            "component": "multi_sink_writer",
            "status": "UP" if is_healthy else "DOWN",
            "overall_metrics": metrics,
            "sink_health": sink_health,
            "enabled_sinks": {
                "fhir_store": settings.FHIR_STORE_ENABLED,
                "elasticsearch": settings.ELASTICSEARCH_ENABLED,
                "mongodb": settings.MONGODB_ENABLED
            }
        }
        
        if is_healthy:
            return health_data
        else:
            raise HTTPException(status_code=503, detail=health_data)
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Sinks health check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "component": "multi_sink_writer",
            "status": "DOWN",
            "error": str(e)
        })


@router.get("/dlq")
async def dlq_health():
    """Dead Letter Queue service health check"""
    try:
        if not multi_sink_writer or not multi_sink_writer.dlq_service:
            raise HTTPException(status_code=503, detail={
                "component": "dlq_service",
                "status": "DOWN",
                "error": "DLQ service not initialized"
            })
        
        dlq_service = multi_sink_writer.dlq_service
        is_healthy = dlq_service.is_healthy()
        metrics = dlq_service.get_dlq_metrics()
        
        health_data = {
            "component": "dlq_service",
            "status": "UP" if is_healthy else "DOWN",
            "metrics": metrics,
            "dlq_topics": dlq_service.dlq_topics
        }
        
        if is_healthy:
            return health_data
        else:
            raise HTTPException(status_code=503, detail=health_data)
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error("DLQ health check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "component": "dlq_service",
            "status": "DOWN",
            "error": str(e)
        })


@router.get("/fhir-transformer")
async def fhir_transformer_health():
    """FHIR transformer health check"""
    try:
        if not fhir_transformer:
            raise HTTPException(status_code=503, detail={
                "component": "fhir_transformer",
                "status": "DOWN",
                "error": "FHIR transformer not initialized"
            })
        
        is_healthy = fhir_transformer.is_healthy()
        
        health_data = {
            "component": "fhir_transformer",
            "status": "UP" if is_healthy else "DOWN",
            "service_name": fhir_transformer.service_name
        }
        
        if is_healthy:
            return health_data
        else:
            raise HTTPException(status_code=503, detail=health_data)
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error("FHIR transformer health check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "component": "fhir_transformer",
            "status": "DOWN",
            "error": str(e)
        })


@router.get("/readiness")
async def readiness_check():
    """Readiness check for Kubernetes"""
    try:
        # Check if all critical components are ready
        ready = True
        components = {}
        
        if kafka_consumer:
            consumer_ready = kafka_consumer.is_healthy()
            components["kafka_consumer"] = consumer_ready
            ready = ready and consumer_ready
        else:
            components["kafka_consumer"] = False
            ready = False
        
        if multi_sink_writer:
            writer_ready = multi_sink_writer.is_healthy()
            components["multi_sink_writer"] = writer_ready
            ready = ready and writer_ready
        else:
            components["multi_sink_writer"] = False
            ready = False
        
        readiness_data = {
            "service": "stage2-storage-fanout",
            "ready": ready,
            "components": components
        }
        
        if ready:
            return readiness_data
        else:
            raise HTTPException(status_code=503, detail=readiness_data)
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Readiness check failed", error=str(e))
        raise HTTPException(status_code=503, detail={
            "service": "stage2-storage-fanout",
            "ready": False,
            "error": str(e)
        })


@router.get("/liveness")
async def liveness_check():
    """Liveness check for Kubernetes"""
    # Simple liveness check - just verify the service is running
    return {
        "service": "stage2-storage-fanout",
        "alive": True,
        "port": settings.PORT
    }
