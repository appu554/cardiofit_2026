"""
Evidence Envelope Service
FastAPI application for clinical decision evidence management
"""

from contextlib import asynccontextmanager
from typing import Dict, Any, Optional, List
from datetime import datetime

from fastapi import FastAPI, HTTPException, status, Query, Path
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app, Counter, Histogram, Gauge
import structlog
import uvicorn

from models.evidence_envelope import (
    EvidenceEnvelopeRequest,
    EvidenceEnvelopeResponse,
    ClinicalContext
)
from services.envelope_manager import envelope_manager
from middleware.auth_middleware import AuthMiddleware
from utils.config import settings
from utils.database import init_databases, close_databases

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
envelope_created_counter = Counter(
    'evidence_envelope_created_total',
    'Total number of evidence envelopes created'
)

envelope_finalized_counter = Counter(
    'evidence_envelope_finalized_total',
    'Total number of evidence envelopes finalized'
)

inference_steps_counter = Counter(
    'evidence_envelope_inference_steps_total',
    'Total number of inference steps added'
)

envelope_creation_duration = Histogram(
    'evidence_envelope_creation_duration_seconds',
    'Time taken to create evidence envelope'
)

envelope_confidence_gauge = Gauge(
    'evidence_envelope_confidence_score',
    'Current confidence score of evidence envelope',
    ['envelope_id']
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Application lifecycle management
    """
    logger.info("evidence_envelope_service_starting", version=settings.VERSION)

    # Initialize databases
    await init_databases()

    # Initialize services
    await envelope_manager.kafka_producer.start()

    yield

    # Cleanup
    await envelope_manager.kafka_producer.stop()
    await close_databases()

    logger.info("evidence_envelope_service_stopped")


# Create FastAPI application
app = FastAPI(
    title="Evidence Envelope Service",
    description="Clinical decision evidence management and audit trail service",
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
        "service": "evidence-envelope-service",
        "version": settings.VERSION,
        "timestamp": datetime.utcnow().isoformat()
    }


@app.get("/health/ready", tags=["health"])
async def readiness_check() -> Dict[str, Any]:
    """
    Readiness check endpoint
    """
    # Check dependent services
    redis_ready = await envelope_manager.redis_client.ping()
    kafka_ready = envelope_manager.kafka_producer.is_connected()

    if not (redis_ready and kafka_ready):
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Service dependencies not ready"
        )

    return {
        "status": "ready",
        "dependencies": {
            "redis": redis_ready,
            "kafka": kafka_ready
        }
    }


# Evidence Envelope endpoints

@app.post(
    "/envelopes",
    response_model=EvidenceEnvelopeResponse,
    status_code=status.HTTP_201_CREATED,
    tags=["envelopes"]
)
async def create_evidence_envelope(
    request: EvidenceEnvelopeRequest
) -> EvidenceEnvelopeResponse:
    """
    Create a new evidence envelope for clinical decision tracking

    This endpoint initializes a new evidence envelope that will track
    all inference steps, confidence scores, and regulatory compliance
    information for a clinical decision.
    """
    try:
        with envelope_creation_duration.time():
            envelope = await envelope_manager.create_envelope(request)

        envelope_created_counter.inc()
        envelope_confidence_gauge.labels(envelope_id=envelope.envelope_id).set(
            envelope.confidence_scores.overall
        )

        logger.info(
            "envelope_created",
            envelope_id=envelope.envelope_id,
            proposal_id=request.proposal_id
        )

        return EvidenceEnvelopeResponse.from_envelope(envelope)

    except Exception as e:
        logger.error("envelope_creation_failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to create evidence envelope: {str(e)}"
        )


@app.get(
    "/envelopes/{envelope_id}",
    response_model=EvidenceEnvelopeResponse,
    tags=["envelopes"]
)
async def get_evidence_envelope(
    envelope_id: str = Path(..., description="Evidence envelope identifier")
) -> EvidenceEnvelopeResponse:
    """
    Retrieve an evidence envelope by ID

    Returns the complete evidence envelope including all inference steps,
    confidence scores, and regulatory compliance information.
    """
    envelope = await envelope_manager.get_envelope(envelope_id)

    if not envelope:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Evidence envelope {envelope_id} not found"
        )

    return EvidenceEnvelopeResponse.from_envelope(envelope)


@app.post(
    "/envelopes/{envelope_id}/inference-steps",
    response_model=EvidenceEnvelopeResponse,
    tags=["envelopes"]
)
async def add_inference_step(
    envelope_id: str,
    step_type: str,
    description: str,
    source_data: Dict[str, Any],
    reasoning_logic: str,
    result_data: Dict[str, Any],
    confidence: float = Query(..., ge=0.0, le=1.0),
    execution_time_ms: int = Query(..., ge=0),
    knowledge_sources: Optional[List[str]] = None
) -> EvidenceEnvelopeResponse:
    """
    Add an inference step to an existing envelope

    Records a single step in the clinical reasoning chain, including
    the logic applied, data used, and confidence in the result.
    """
    try:
        envelope = await envelope_manager.add_inference_step(
            envelope_id=envelope_id,
            step_type=step_type,
            description=description,
            source_data=source_data,
            reasoning_logic=reasoning_logic,
            result_data=result_data,
            confidence=confidence,
            execution_time_ms=execution_time_ms,
            knowledge_sources=knowledge_sources
        )

        inference_steps_counter.inc()
        envelope_confidence_gauge.labels(envelope_id=envelope_id).set(
            envelope.confidence_scores.overall
        )

        return EvidenceEnvelopeResponse.from_envelope(envelope)

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=str(e)
        )
    except Exception as e:
        logger.error(
            "add_inference_step_failed",
            envelope_id=envelope_id,
            error=str(e)
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to add inference step: {str(e)}"
        )


@app.post(
    "/envelopes/{envelope_id}/finalize",
    response_model=EvidenceEnvelopeResponse,
    tags=["envelopes"]
)
async def finalize_envelope(
    envelope_id: str,
    final_conclusion: Dict[str, Any],
    validation_results: Optional[Dict[str, Any]] = None
) -> EvidenceEnvelopeResponse:
    """
    Finalize an evidence envelope

    Completes the evidence envelope with final conclusions and generates
    integrity checksums for audit compliance.
    """
    try:
        envelope = await envelope_manager.finalize_envelope(
            envelope_id=envelope_id,
            final_conclusion=final_conclusion,
            validation_results=validation_results
        )

        envelope_finalized_counter.inc()
        envelope_confidence_gauge.labels(envelope_id=envelope_id).set(
            envelope.confidence_scores.overall
        )

        logger.info(
            "envelope_finalized",
            envelope_id=envelope_id,
            confidence=envelope.confidence_scores.overall
        )

        return EvidenceEnvelopeResponse.from_envelope(envelope)

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        logger.error(
            "envelope_finalization_failed",
            envelope_id=envelope_id,
            error=str(e)
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to finalize envelope: {str(e)}"
        )


@app.get(
    "/envelopes",
    response_model=List[EvidenceEnvelopeResponse],
    tags=["envelopes"]
)
async def query_envelopes(
    proposal_id: Optional[str] = Query(None, description="Filter by proposal ID"),
    patient_id: Optional[str] = Query(None, description="Filter by patient ID"),
    workflow_type: Optional[str] = Query(None, description="Filter by workflow type"),
    status: Optional[str] = Query(None, description="Filter by status"),
    start_date: Optional[datetime] = Query(None, description="Filter by start date"),
    end_date: Optional[datetime] = Query(None, description="Filter by end date"),
    limit: int = Query(100, ge=1, le=1000, description="Maximum results to return")
) -> List[EvidenceEnvelopeResponse]:
    """
    Query evidence envelopes based on criteria

    Search for evidence envelopes using various filters. Useful for
    audit reporting and compliance tracking.
    """
    envelopes = await envelope_manager.query_envelopes(
        proposal_id=proposal_id,
        patient_id=patient_id,
        workflow_type=workflow_type,
        start_date=start_date,
        end_date=end_date,
        status=status,
        limit=limit
    )

    return envelopes


@app.get(
    "/envelopes/{envelope_id}/audit-trail",
    response_model=Dict[str, Any],
    tags=["audit"]
)
async def get_audit_trail(
    envelope_id: str = Path(..., description="Evidence envelope identifier")
) -> Dict[str, Any]:
    """
    Get complete audit trail for an envelope

    Returns the full audit trail including all events, timestamps,
    and integrity verification information required for regulatory compliance.
    """
    try:
        audit_trail = await envelope_manager.get_audit_trail(envelope_id)
        return audit_trail

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=str(e)
        )
    except Exception as e:
        logger.error(
            "audit_trail_generation_failed",
            envelope_id=envelope_id,
            error=str(e)
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to generate audit trail: {str(e)}"
        )


@app.post(
    "/envelopes/{envelope_id}/wrap-response",
    response_model=Dict[str, Any],
    tags=["integration"]
)
async def wrap_service_response(
    envelope_id: str,
    service_response: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Wrap a service response with evidence envelope metadata

    Enhances a clinical service response with complete evidence trail,
    confidence scores, and regulatory compliance information.
    """
    wrapped_response = await envelope_manager.wrap_service_response(
        service_response=service_response,
        envelope_id=envelope_id
    )

    return wrapped_response


@app.get(
    "/envelopes/{envelope_id}/integrity",
    response_model=Dict[str, Any],
    tags=["audit"]
)
async def verify_envelope_integrity(
    envelope_id: str = Path(..., description="Evidence envelope identifier")
) -> Dict[str, Any]:
    """
    Verify the integrity of an evidence envelope

    Validates the cryptographic checksum to ensure the envelope
    has not been tampered with since finalization.
    """
    envelope = await envelope_manager.get_envelope(envelope_id)

    if not envelope:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Evidence envelope {envelope_id} not found"
        )

    return {
        "envelope_id": envelope_id,
        "integrity_verified": envelope.verify_integrity(),
        "checksum": envelope.checksum,
        "status": envelope.status,
        "verification_timestamp": datetime.utcnow().isoformat()
    }


if __name__ == "__main__":
    """
    Run the Evidence Envelope service
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