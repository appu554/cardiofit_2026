"""
Metrics Controller for Stage 2: Storage Fan-Out Service

Provides comprehensive metrics for monitoring and observability including
Prometheus metrics, DLQ statistics, and performance metrics.
"""

import time
from typing import Dict, Any

import structlog
from fastapi import APIRouter, Response
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

from app.config import settings

logger = structlog.get_logger(__name__)

router = APIRouter()

# Global references to services (set by main.py)
kafka_consumer = None
fhir_transformer = None
multi_sink_writer = None


def set_service_references(consumer, transformer, writer):
    """Set global service references for metrics collection"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    kafka_consumer = consumer
    fhir_transformer = transformer
    multi_sink_writer = writer


@router.get("/")
async def overall_metrics():
    """Get comprehensive metrics for all components"""
    try:
        metrics = {
            "service": "stage2-storage-fanout",
            "port": settings.PORT,
            "timestamp": int(time.time()),
            "uptime_seconds": time.time() - start_time if 'start_time' in globals() else 0,
            "components": {}
        }
        
        # Kafka Consumer metrics
        if kafka_consumer:
            metrics["components"]["kafka_consumer"] = kafka_consumer.get_metrics()
        
        # Multi-Sink Writer metrics
        if multi_sink_writer:
            metrics["components"]["multi_sink_writer"] = multi_sink_writer.get_metrics()
        
        # FHIR Transformer metrics (basic)
        if fhir_transformer:
            metrics["components"]["fhir_transformer"] = {
                "service_name": fhir_transformer.service_name,
                "is_healthy": fhir_transformer.is_healthy()
            }
        
        return metrics
        
    except Exception as e:
        logger.error("Failed to collect overall metrics", error=str(e))
        return {
            "service": "stage2-storage-fanout",
            "error": str(e),
            "timestamp": int(time.time())
        }


@router.get("/kafka")
async def kafka_metrics():
    """Get Kafka consumer metrics"""
    try:
        if not kafka_consumer:
            return {"error": "Kafka consumer not initialized"}
        
        metrics = kafka_consumer.get_metrics()
        lag_info = kafka_consumer.get_consumer_lag()
        
        return {
            "component": "kafka_consumer",
            "metrics": metrics,
            "consumer_lag": lag_info,
            "configuration": {
                "input_topic": settings.KAFKA_INPUT_TOPIC,
                "consumer_group": settings.KAFKA_CONSUMER_GROUP,
                "max_poll_records": settings.KAFKA_MAX_POLL_RECORDS,
                "auto_commit": settings.KAFKA_ENABLE_AUTO_COMMIT
            }
        }
        
    except Exception as e:
        logger.error("Failed to collect Kafka metrics", error=str(e))
        return {"error": str(e)}


@router.get("/sinks")
async def sink_metrics():
    """Get multi-sink writer metrics"""
    try:
        if not multi_sink_writer:
            return {"error": "Multi-sink writer not initialized"}
        
        overall_metrics = multi_sink_writer.get_metrics()
        
        # Get detailed sink metrics
        sink_details = {}
        for sink_name, sink in multi_sink_writer.sinks.items():
            sink_details[sink_name] = {
                "metrics": sink.get_metrics(),
                "is_healthy": sink.is_healthy(),
                "enabled": multi_sink_writer._is_sink_enabled(sink_name)
            }
        
        return {
            "component": "multi_sink_writer",
            "overall_metrics": overall_metrics,
            "sink_details": sink_details,
            "configuration": {
                "parallel_writes": settings.PARALLEL_WRITES,
                "thread_pool_size": settings.THREAD_POOL_SIZE,
                "sink_timeout_seconds": settings.SINK_TIMEOUT_SECONDS,
                "batch_size": settings.BATCH_SIZE
            }
        }
        
    except Exception as e:
        logger.error("Failed to collect sink metrics", error=str(e))
        return {"error": str(e)}


@router.get("/dlq")
async def dlq_metrics():
    """Get Dead Letter Queue metrics"""
    try:
        if not multi_sink_writer or not multi_sink_writer.dlq_service:
            return {"error": "DLQ service not initialized"}
        
        dlq_service = multi_sink_writer.dlq_service
        dlq_metrics = dlq_service.get_dlq_metrics()
        
        return {
            "component": "dlq_service",
            "metrics": dlq_metrics,
            "topics": dlq_service.dlq_topics,
            "configuration": {
                "dlq_topic": settings.KAFKA_DLQ_TOPIC,
                "retry_enabled": settings.RETRY_ENABLED,
                "max_retry_attempts": settings.RETRY_MAX_ATTEMPTS
            }
        }
        
    except Exception as e:
        logger.error("Failed to collect DLQ metrics", error=str(e))
        return {"error": str(e)}


@router.get("/performance")
async def performance_metrics():
    """Get performance-related metrics"""
    try:
        performance_data = {
            "component": "performance",
            "timestamp": int(time.time())
        }
        
        # Kafka consumer performance
        if kafka_consumer:
            consumer_metrics = kafka_consumer.get_metrics()
            performance_data["kafka_consumer"] = {
                "throughput_msgs_per_sec": consumer_metrics.get("processed_messages", 0) / max(time.time() - start_time if 'start_time' in globals() else 1, 1),
                "success_rate": consumer_metrics.get("success_rate", 0),
                "total_processed": consumer_metrics.get("processed_messages", 0),
                "total_failed": consumer_metrics.get("failed_messages", 0)
            }
        
        # Sink writer performance
        if multi_sink_writer:
            writer_metrics = multi_sink_writer.get_metrics()
            performance_data["multi_sink_writer"] = {
                "overall_success_rate": writer_metrics.get("success_rate", 0),
                "total_writes": writer_metrics.get("total_writes", 0),
                "successful_writes": writer_metrics.get("successful_writes", 0),
                "failed_writes": writer_metrics.get("failed_writes", 0)
            }
            
            # Individual sink performance
            sink_performance = {}
            for sink_name, metrics in writer_metrics.get("sink_metrics", {}).items():
                sink_performance[sink_name] = {
                    "success_rate": metrics.get("successful_writes", 0) / max(metrics.get("total_writes", 1), 1),
                    "avg_write_time_ms": metrics.get("avg_write_time", 0) * 1000,
                    "total_writes": metrics.get("total_writes", 0)
                }
            performance_data["sink_performance"] = sink_performance
        
        return performance_data
        
    except Exception as e:
        logger.error("Failed to collect performance metrics", error=str(e))
        return {"error": str(e)}


@router.get("/circuit-breakers")
async def circuit_breaker_metrics():
    """Get circuit breaker status and metrics"""
    try:
        if not multi_sink_writer:
            return {"error": "Multi-sink writer not initialized"}
        
        cb_data = {
            "component": "circuit_breakers",
            "enabled": settings.CIRCUIT_BREAKER_ENABLED,
            "configuration": {
                "failure_threshold": settings.CIRCUIT_BREAKER_FAILURE_THRESHOLD,
                "recovery_timeout": settings.CIRCUIT_BREAKER_RECOVERY_TIMEOUT
            }
        }
        
        if settings.CIRCUIT_BREAKER_ENABLED and multi_sink_writer.circuit_breakers:
            cb_states = {}
            for sink_name, cb in multi_sink_writer.circuit_breakers.items():
                cb_states[sink_name] = {
                    "state": cb.current_state,
                    "failure_count": cb.fail_counter,
                    "last_failure_time": getattr(cb, 'last_failure_time', None),
                    "success_count": getattr(cb, 'success_counter', 0)
                }
            cb_data["circuit_breaker_states"] = cb_states
        else:
            cb_data["circuit_breaker_states"] = "Circuit breakers disabled"
        
        return cb_data
        
    except Exception as e:
        logger.error("Failed to collect circuit breaker metrics", error=str(e))
        return {"error": str(e)}


@router.get("/prometheus")
async def prometheus_metrics():
    """Get Prometheus-formatted metrics"""
    try:
        if not settings.PROMETHEUS_ENABLED:
            return Response(
                content="Prometheus metrics disabled",
                status_code=503,
                media_type="text/plain"
            )
        
        # Generate Prometheus metrics
        metrics_output = generate_latest()
        
        return Response(
            content=metrics_output,
            media_type=CONTENT_TYPE_LATEST
        )
        
    except Exception as e:
        logger.error("Failed to generate Prometheus metrics", error=str(e))
        return Response(
            content=f"Error generating metrics: {str(e)}",
            status_code=500,
            media_type="text/plain"
        )


@router.get("/summary")
async def metrics_summary():
    """Get a summary of key metrics for dashboards"""
    try:
        summary = {
            "service": "stage2-storage-fanout",
            "timestamp": int(time.time()),
            "status": "operational"
        }
        
        # Key performance indicators
        if kafka_consumer and multi_sink_writer:
            consumer_metrics = kafka_consumer.get_metrics()
            writer_metrics = multi_sink_writer.get_metrics()
            dlq_metrics = multi_sink_writer.dlq_service.get_dlq_metrics() if multi_sink_writer.dlq_service else {}
            
            summary.update({
                "messages_processed_total": consumer_metrics.get("processed_messages", 0),
                "messages_failed_total": consumer_metrics.get("failed_messages", 0),
                "consumer_success_rate": consumer_metrics.get("success_rate", 0),
                "sink_writes_total": writer_metrics.get("total_writes", 0),
                "sink_success_rate": writer_metrics.get("success_rate", 0),
                "dlq_messages_total": dlq_metrics.get("total_dlq_messages", 0),
                "critical_failures": dlq_metrics.get("poison_messages", 0),
                "healthy_sinks": sum(1 for sink in multi_sink_writer.sinks.values() if sink.is_healthy()),
                "total_sinks": len(multi_sink_writer.sinks)
            })
        
        return summary
        
    except Exception as e:
        logger.error("Failed to generate metrics summary", error=str(e))
        return {
            "service": "stage2-storage-fanout",
            "status": "error",
            "error": str(e),
            "timestamp": int(time.time())
        }


# Initialize start time for uptime calculation
start_time = time.time()
