"""
Resilience monitoring and management endpoints
"""

import logging
from typing import Dict, Any, List
from fastapi import APIRouter, HTTPException, status
from fastapi.responses import JSONResponse

from app.resilience.circuit_breaker_manager import circuit_breaker_manager
from app.kafka_producer import get_kafka_producer
from app.cache.device_cache_manager import get_device_cache_manager

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/v1/resilience", tags=["resilience"])


@router.get("/health")
async def health_check():
    """
    Health check endpoint that includes circuit breaker status
    """
    try:
        # Get circuit breaker states
        cb_states = circuit_breaker_manager.get_circuit_breaker_states()
        open_circuits = circuit_breaker_manager.get_open_circuit_breakers()
        
        # Determine overall health
        is_healthy = len(open_circuits) == 0
        health_status = "healthy" if is_healthy else "degraded"
        
        response = {
            "status": health_status,
            "timestamp": "2025-06-25T16:30:00Z",  # This would be dynamic
            "service": "device-data-ingestion-service",
            "version": "1.0.0",
            "circuit_breakers": {
                "total": len(cb_states),
                "open": len(open_circuits),
                "states": cb_states,
                "open_circuits": open_circuits
            }
        }
        
        status_code = status.HTTP_200_OK if is_healthy else status.HTTP_503_SERVICE_UNAVAILABLE
        return JSONResponse(content=response, status_code=status_code)
        
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return JSONResponse(
            content={
                "status": "unhealthy",
                "error": str(e),
                "service": "device-data-ingestion-service"
            },
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE
        )


@router.get("/circuit-breakers")
async def get_circuit_breaker_status():
    """
    Get detailed status of all circuit breakers
    """
    try:
        metrics = circuit_breaker_manager.get_all_metrics()
        
        return {
            "circuit_breakers": metrics,
            "summary": {
                "total": len(metrics),
                "open": len([m for m in metrics.values() if m["state"] == "open"]),
                "half_open": len([m for m in metrics.values() if m["state"] == "half_open"]),
                "closed": len([m for m in metrics.values() if m["state"] == "closed"])
            }
        }
        
    except Exception as e:
        logger.error(f"Failed to get circuit breaker status: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get circuit breaker status: {e}"
        )


@router.get("/circuit-breakers/{service_name}")
async def get_circuit_breaker_metrics(service_name: str):
    """
    Get detailed metrics for a specific circuit breaker
    """
    try:
        metrics = circuit_breaker_manager.get_service_metrics(service_name)
        
        if not metrics:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Circuit breaker for service '{service_name}' not found"
            )
        
        return {
            "service_name": service_name,
            "metrics": metrics
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to get metrics for {service_name}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get circuit breaker metrics: {e}"
        )


@router.post("/circuit-breakers/{service_name}/reset")
async def reset_circuit_breaker(service_name: str):
    """
    Manually reset a circuit breaker to CLOSED state
    """
    try:
        await circuit_breaker_manager.reset_circuit_breaker(service_name)
        
        return {
            "message": f"Circuit breaker for '{service_name}' has been reset",
            "service_name": service_name,
            "new_state": "closed"
        }
        
    except Exception as e:
        logger.error(f"Failed to reset circuit breaker for {service_name}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to reset circuit breaker: {e}"
        )


@router.post("/circuit-breakers/reset-all")
async def reset_all_circuit_breakers():
    """
    Reset all circuit breakers to CLOSED state
    """
    try:
        await circuit_breaker_manager.reset_all_circuit_breakers()
        
        return {
            "message": "All circuit breakers have been reset",
            "reset_count": len(circuit_breaker_manager.circuit_breakers)
        }
        
    except Exception as e:
        logger.error(f"Failed to reset all circuit breakers: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to reset circuit breakers: {e}"
        )


@router.get("/metrics")
async def get_resilience_metrics():
    """
    Get comprehensive resilience metrics
    """
    try:
        cb_metrics = circuit_breaker_manager.get_all_metrics()
        
        # Calculate aggregate metrics
        total_requests = sum(m["total_requests"] for m in cb_metrics.values())
        total_failures = sum(m["failed_requests"] for m in cb_metrics.values())
        total_fallbacks = sum(m["fallback_executions"] for m in cb_metrics.values())
        
        overall_success_rate = (
            ((total_requests - total_failures) / max(total_requests, 1)) * 100
        )
        
        return {
            "overview": {
                "total_requests": total_requests,
                "total_failures": total_failures,
                "total_fallbacks": total_fallbacks,
                "overall_success_rate": round(overall_success_rate, 2),
                "services_monitored": len(cb_metrics)
            },
            "circuit_breakers": cb_metrics,
            "alerts": {
                "open_circuits": circuit_breaker_manager.get_open_circuit_breakers(),
                "degraded_services": [
                    name for name, metrics in cb_metrics.items()
                    if metrics["success_rate"] < 95.0
                ]
            }
        }
        
    except Exception as e:
        logger.error(f"Failed to get resilience metrics: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get resilience metrics: {e}"
        )


@router.get("/status")
async def get_resilience_status():
    """
    Get high-level resilience status
    """
    try:
        open_circuits = circuit_breaker_manager.get_open_circuit_breakers()
        cb_states = circuit_breaker_manager.get_circuit_breaker_states()
        
        # Determine resilience level
        if len(open_circuits) == 0:
            resilience_level = "optimal"
            status_message = "All services operating normally"
        elif len(open_circuits) <= len(cb_states) * 0.3:  # Less than 30% open
            resilience_level = "degraded"
            status_message = f"{len(open_circuits)} service(s) experiencing issues"
        else:
            resilience_level = "critical"
            status_message = f"Multiple services failing ({len(open_circuits)} circuits open)"
        
        return {
            "resilience_level": resilience_level,
            "status_message": status_message,
            "open_circuits": len(open_circuits),
            "total_circuits": len(cb_states),
            "affected_services": open_circuits,
            "recommendations": _get_resilience_recommendations(open_circuits, cb_states)
        }
        
    except Exception as e:
        logger.error(f"Failed to get resilience status: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get resilience status: {e}"
        )


@router.get("/performance/batching")
async def get_batching_performance():
    """
    Get adaptive batching performance metrics
    """
    try:
        producer = await get_kafka_producer()

        if not producer.batch_manager:
            return {
                "status": "disabled",
                "message": "Adaptive batching is not enabled"
            }

        metrics = await producer.batch_manager.get_performance_metrics()
        device_patterns = await producer.batch_manager.get_device_patterns()

        return {
            "batching_metrics": metrics,
            "device_patterns": device_patterns,
            "batching_enabled": producer.batching_enabled
        }

    except Exception as e:
        logger.error(f"Failed to get batching performance: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get batching performance: {e}"
        )


@router.get("/performance/cache")
async def get_cache_performance():
    """
    Get cache performance metrics
    """
    try:
        cache_manager = await get_device_cache_manager()

        cache_stats = await cache_manager.get_cache_statistics()
        cache_health = await cache_manager.get_cache_health()

        return {
            "cache_statistics": cache_stats,
            "cache_health": cache_health
        }

    except Exception as e:
        logger.error(f"Failed to get cache performance: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get cache performance: {e}"
        )


@router.get("/performance/overview")
async def get_performance_overview():
    """
    Get comprehensive performance overview
    """
    try:
        # Get circuit breaker metrics
        cb_metrics = circuit_breaker_manager.get_all_metrics()

        # Get batching metrics
        producer = await get_kafka_producer()
        batching_metrics = None
        if producer.batch_manager:
            batching_metrics = await producer.batch_manager.get_performance_metrics()

        # Get cache metrics
        cache_manager = await get_device_cache_manager()
        cache_stats = await cache_manager.get_cache_statistics()

        return {
            "timestamp": "2025-06-25T20:30:00Z",  # Would be dynamic
            "circuit_breakers": {
                "total_services": len(cb_metrics),
                "healthy_services": len([m for m in cb_metrics.values() if m["state"] == "closed"]),
                "overall_success_rate": sum(m["success_rate"] for m in cb_metrics.values()) / max(len(cb_metrics), 1)
            },
            "batching": {
                "enabled": producer.batching_enabled,
                "metrics": batching_metrics
            },
            "caching": {
                "enabled": cache_stats.get("redis_metrics", {}).get("is_healthy", False),
                "hit_rate": cache_stats.get("redis_metrics", {}).get("cache_hit_rate", 0)
            }
        }

    except Exception as e:
        logger.error(f"Failed to get performance overview: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get performance overview: {e}"
        )


def _get_resilience_recommendations(open_circuits: List[str],
                                  all_states: Dict[str, str]) -> List[str]:
    """
    Generate recommendations based on circuit breaker states
    """
    recommendations = []

    if "auth_service" in open_circuits:
        recommendations.append("Check authentication service health and connectivity")

    if "kafka_producer" in open_circuits:
        recommendations.append("Verify Kafka cluster connectivity and broker health")

    if "google_healthcare_api" in open_circuits:
        recommendations.append("Check Google Healthcare API service status and quotas")

    if len(open_circuits) > len(all_states) * 0.5:
        recommendations.append("Consider enabling degraded mode operation")
        recommendations.append("Review network connectivity and infrastructure health")

    if not recommendations:
        recommendations.append("System resilience is optimal - no action required")

    return recommendations
