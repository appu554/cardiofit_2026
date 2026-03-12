"""
L1 Cache and Prefetcher Service
Ultra-fast caching with ML-based intelligent prefetching
"""

from contextlib import asynccontextmanager
from typing import Dict, Any, Optional, List
from datetime import datetime

from fastapi import FastAPI, HTTPException, status, Query, Path, Depends
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app, Counter, Histogram, Gauge
import structlog
import uvicorn

from models.cache_models import (
    CacheRequest,
    CacheResponse,
    PrefetchRequest,
    PrefetchResponse,
    CacheMetrics,
    SessionContext,
    CacheKeyType
)
from services.l1_cache_manager import L1CacheManager
from services.prefetch_manager import PrefetchManager, DataSource
from ml.prefetch_predictor import PrefetchPredictor
from middleware.auth_middleware import AuthMiddleware, get_current_user
from utils.config import settings

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()

# Prometheus metrics
cache_requests_counter = Counter(
    'l1_cache_requests_total',
    'Total number of cache requests',
    ['operation', 'result']
)

cache_response_time_histogram = Histogram(
    'l1_cache_response_time_seconds',
    'Cache response time in seconds',
    ['operation']
)

prefetch_requests_counter = Counter(
    'prefetch_requests_total',
    'Total number of prefetch requests'
)

prefetch_accuracy_gauge = Gauge(
    'prefetch_accuracy_ratio',
    'Prefetch prediction accuracy ratio'
)

cache_hit_rate_gauge = Gauge(
    'l1_cache_hit_rate',
    'L1 cache hit rate'
)

cache_memory_usage_gauge = Gauge(
    'l1_cache_memory_bytes',
    'L1 cache memory usage in bytes'
)

# Global service instances
l1_cache_manager: Optional[L1CacheManager] = None
prefetch_predictor: Optional[PrefetchPredictor] = None
prefetch_manager: Optional[PrefetchManager] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Application lifecycle management
    """
    global l1_cache_manager, prefetch_predictor, prefetch_manager

    logger.info("l1_cache_service_starting", version=settings.VERSION)

    try:
        # Initialize L1 Cache Manager
        l1_cache_manager = L1CacheManager(
            max_size_mb=settings.L1_CACHE_SIZE_MB,
            default_ttl_seconds=settings.L1_CACHE_DEFAULT_TTL,
            max_entries=settings.L1_CACHE_MAX_ENTRIES,
            memory_pressure_threshold=settings.MEMORY_PRESSURE_THRESHOLD
        )

        # Initialize ML Prefetch Predictor
        prefetch_predictor = PrefetchPredictor(
            model_update_interval_hours=settings.ML_MODEL_UPDATE_INTERVAL_HOURS,
            min_training_samples=settings.ML_MIN_TRAINING_SAMPLES,
            prediction_horizon_hours=settings.ML_PREDICTION_HORIZON_HOURS
        )

        # Initialize Prefetch Manager
        prefetch_manager = PrefetchManager(
            l1_cache=l1_cache_manager,
            predictor=prefetch_predictor,
            max_concurrent_fetches=settings.MAX_CONCURRENT_FETCHES,
            prefetch_budget_mb=settings.PREFETCH_BUDGET_MB,
            min_confidence_threshold=settings.MIN_CONFIDENCE_THRESHOLD
        )

        # Configure data sources
        await _configure_data_sources()

        # Start background tasks
        await l1_cache_manager.start_background_tasks()
        await prefetch_predictor.start_training_loop()
        await prefetch_manager.start_background_tasks()

        logger.info("l1_cache_service_initialized")

        yield

        # Cleanup
        if l1_cache_manager:
            await l1_cache_manager.stop_background_tasks()
        if prefetch_predictor:
            await prefetch_predictor.stop_training_loop()
        if prefetch_manager:
            await prefetch_manager.stop_background_tasks()

        logger.info("l1_cache_service_stopped")

    except Exception as e:
        logger.error("service_initialization_failed", error=str(e))
        raise


async def _configure_data_sources():
    """Configure external data sources for prefetching"""
    if not prefetch_manager:
        return

    # Add data sources from configuration
    data_sources = [
        DataSource("patient_service", settings.PATIENT_SERVICE_URL, 500),
        DataSource("clinical_service", settings.CLINICAL_SERVICE_URL, 300),
        DataSource("medication_service", settings.MEDICATION_SERVICE_URL, 400),
        DataSource("guideline_service", settings.GUIDELINE_SERVICE_URL, 1000),
        DataSource("semantic_service", settings.SEMANTIC_SERVICE_URL, 800),
    ]

    for source in data_sources:
        prefetch_manager.add_data_source(source)


# Create FastAPI application
app = FastAPI(
    title="L1 Cache and Prefetcher Service",
    description="Ultra-fast clinical data caching with ML-based intelligent prefetching",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add authentication middleware
app.add_middleware(AuthMiddleware)

# Mount Prometheus metrics endpoint
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)


# Health check endpoints

@app.get("/health", tags=["health"])
async def health_check() -> Dict[str, Any]:
    """
    Health check endpoint
    """
    return {
        "status": "healthy",
        "service": "l1-cache-prefetcher-service",
        "version": settings.VERSION,
        "timestamp": datetime.utcnow().isoformat()
    }


@app.get("/health/ready", tags=["health"])
async def readiness_check() -> Dict[str, Any]:
    """
    Readiness check endpoint
    """
    if not all([l1_cache_manager, prefetch_predictor, prefetch_manager]):
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Service not ready - components not initialized"
        )

    return {
        "status": "ready",
        "components": {
            "l1_cache": l1_cache_manager is not None,
            "predictor": prefetch_predictor is not None,
            "prefetch_manager": prefetch_manager is not None
        }
    }


# L1 Cache endpoints

@app.get(
    "/cache/{key}",
    response_model=CacheResponse,
    tags=["cache"]
)
async def get_cached_data(
    key: str = Path(..., description="Cache key to retrieve"),
    session_id: Optional[str] = Query(None, description="Session ID for cache locality"),
    user: Dict[str, Any] = Depends(get_current_user)
) -> CacheResponse:
    """
    Retrieve data from L1 cache with <10ms response time
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    with cache_response_time_histogram.labels(operation="get").time():
        response = await l1_cache_manager.get(
            key=key,
            session_id=session_id,
            user_id=user.get("user_id")
        )

    # Update metrics
    cache_requests_counter.labels(
        operation="get",
        result="hit" if response.hit else "miss"
    ).inc()

    return response


@app.post(
    "/cache",
    tags=["cache"]
)
async def store_cached_data(
    request: CacheRequest,
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Store data in L1 cache
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    # Set user context
    if not request.user_id:
        request.user_id = user.get("user_id")

    with cache_response_time_histogram.labels(operation="put").time():
        success = await l1_cache_manager.put(request)

    cache_requests_counter.labels(
        operation="put",
        result="success" if success else "failure"
    ).inc()

    if not success:
        raise HTTPException(
            status_code=status.HTTP_507_INSUFFICIENT_STORAGE,
            detail="Failed to store data in cache"
        )

    return {"success": True, "key": request.key}


@app.delete("/cache/{key}", tags=["cache"])
async def invalidate_cached_data(
    key: str = Path(..., description="Cache key to invalidate"),
    session_id: Optional[str] = Query(None, description="Session ID for targeted invalidation"),
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Invalidate cached data
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    success = await l1_cache_manager.invalidate(key, session_id)

    cache_requests_counter.labels(
        operation="invalidate",
        result="success" if success else "failure"
    ).inc()

    return {"success": success, "key": key}


@app.delete("/cache/sessions/{session_id}", tags=["cache"])
async def invalidate_session_cache(
    session_id: str = Path(..., description="Session ID to invalidate"),
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Invalidate all cached data for a session
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    count = await l1_cache_manager.invalidate_session(session_id)

    return {"success": True, "session_id": session_id, "invalidated_keys": count}


# Prefetch endpoints

@app.post(
    "/prefetch/predict",
    response_model=PrefetchResponse,
    tags=["prefetch"]
)
async def prefetch_predicted_data(
    session_id: Optional[str] = Query(None, description="Session ID for context"),
    max_items: int = Query(50, ge=1, le=200, description="Maximum items to prefetch"),
    confidence_threshold: float = Query(0.6, ge=0.0, le=1.0, description="Minimum confidence threshold"),
    user: Dict[str, Any] = Depends(get_current_user)
) -> PrefetchResponse:
    """
    Execute ML-based predictive prefetching
    """
    if not prefetch_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Prefetch manager not available")

    response = await prefetch_manager.prefetch_predictions(
        session_id=session_id,
        user_id=user.get("user_id"),
        max_items=max_items,
        confidence_threshold=confidence_threshold
    )

    prefetch_requests_counter.inc()

    return response


@app.post(
    "/prefetch/explicit",
    response_model=PrefetchResponse,
    tags=["prefetch"]
)
async def prefetch_explicit_data(
    request: PrefetchRequest,
    user: Dict[str, Any] = Depends(get_current_user)
) -> PrefetchResponse:
    """
    Execute explicit prefetch for specific keys
    """
    if not prefetch_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Prefetch manager not available")

    response = await prefetch_manager.prefetch_explicit(request)

    prefetch_requests_counter.inc()

    return response


# Metrics and monitoring endpoints

@app.get("/metrics/cache", tags=["metrics"])
async def get_cache_metrics(
    user: Dict[str, Any] = Depends(get_current_user)
) -> CacheMetrics:
    """
    Get detailed cache performance metrics
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    metrics = await l1_cache_manager.get_metrics()

    # Update Prometheus gauges
    cache_hit_rate_gauge.set(metrics.hit_rate)
    cache_memory_usage_gauge.set(metrics.total_size_bytes)

    return metrics


@app.get("/metrics/prefetch", tags=["metrics"])
async def get_prefetch_metrics(
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Get prefetch performance metrics
    """
    if not prefetch_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Prefetch manager not available")

    metrics = prefetch_manager.get_metrics()

    # Update Prometheus gauges
    if metrics.get('successful_prefetches', 0) > 0:
        accuracy = metrics['successful_prefetches'] / (
            metrics['successful_prefetches'] + metrics['failed_prefetches']
        )
        prefetch_accuracy_gauge.set(accuracy)

    return metrics


@app.get("/analytics/access-patterns", tags=["analytics"])
async def get_access_patterns(
    limit: int = Query(100, ge=1, le=1000, description="Maximum patterns to return"),
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Get access pattern analytics for ML insights
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    access_patterns = l1_cache_manager.get_access_patterns()

    # Convert to serializable format and limit results
    patterns_data = []
    for key, pattern in list(access_patterns.items())[:limit]:
        patterns_data.append({
            "key": pattern.key,
            "key_type": pattern.key_type.value,
            "access_count": pattern.access_count,
            "access_frequency": pattern.access_frequency,
            "last_accessed": pattern.last_accessed.isoformat(),
            "session_count": len(pattern.session_correlation),
            "user_count": len(pattern.user_correlation)
        })

    return {
        "total_patterns": len(access_patterns),
        "returned_patterns": len(patterns_data),
        "patterns": patterns_data,
        "timestamp": datetime.utcnow().isoformat()
    }


@app.get("/analytics/sessions", tags=["analytics"])
async def get_session_analytics(
    user: Dict[str, Any] = Depends(get_current_user)
) -> Dict[str, Any]:
    """
    Get session-based analytics
    """
    if not l1_cache_manager:
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Cache not available")

    session_contexts = l1_cache_manager.get_session_contexts()

    sessions_data = []
    for session_id, context in session_contexts.items():
        sessions_data.append({
            "session_id": context.session_id,
            "user_id": context.user_id,
            "workflow_type": context.workflow_type,
            "started_at": context.started_at.isoformat(),
            "last_activity": context.last_activity.isoformat(),
            "access_count": len(context.access_pattern),
            "is_active": context.is_active(),
            "cache_budget_mb": context.cache_budget_mb
        })

    return {
        "total_sessions": len(session_contexts),
        "active_sessions": sum(1 for s in sessions_data if s["is_active"]),
        "sessions": sessions_data,
        "timestamp": datetime.utcnow().isoformat()
    }


if __name__ == "__main__":
    """
    Run the L1 Cache and Prefetcher service
    """

    # Configure uvicorn for production
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=settings.PORT,
        reload=settings.DEBUG,
        log_config={
            "version": 1,
            "disable_existing_loggers": False,
            "formatters": {
                "default": {
                    "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
                },
                "json": {
                    "()": "pythonjsonlogger.jsonlogger.JsonFormatter",
                    "format": "%(asctime)s %(name)s %(levelname)s %(message)s"
                }
            },
            "handlers": {
                "default": {
                    "formatter": "json",
                    "class": "logging.StreamHandler",
                    "stream": "ext://sys.stdout"
                }
            },
            "root": {
                "level": "INFO",
                "handlers": ["default"]
            }
        }
    )