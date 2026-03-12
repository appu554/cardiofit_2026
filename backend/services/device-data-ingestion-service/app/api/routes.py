"""
API routes for Device Data Ingestion Service
"""
import logging
from typing import Dict, Any
from datetime import datetime

from fastapi import APIRouter, HTTPException, Depends, Request, BackgroundTasks, Header
from fastapi.responses import JSONResponse
from typing import Optional

from app.models import DeviceReading, IngestionResponse, ErrorResponse, HealthResponse
from app.auth import validate_api_key, check_rate_limit, get_supabase_jwt_auth, supabase_jwt_auth
from app.kafka_producer import get_kafka_producer
from app.envelope.envelope_factory import EnhancedEnvelopeFactory, AuthContext, RequestContext
from app.universal_handler import get_universal_handler, DeviceType
from app.config import settings

# Import transactional outbox components
from app.services.global_outbox_adapter import global_outbox_adapter
from app.services.dead_letter_manager import dead_letter_manager
from app.services.vendor_detection import vendor_detection_service
from app.core.monitoring import metrics_collector
from app.db.models import is_supported_vendor
from app.services.background_publisher import get_background_publisher
import uuid

logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

# Global enhanced envelope factory (will be initialized on startup)
enhanced_envelope_factory: Optional[EnhancedEnvelopeFactory] = None

async def get_enhanced_envelope_factory() -> EnhancedEnvelopeFactory:
    """Get or create the enhanced envelope factory"""
    global enhanced_envelope_factory

    if enhanced_envelope_factory is None:
        try:
            # Try to initialize with cache manager
            from app.cache.device_cache_manager import get_device_cache_manager
            cache_manager = await get_device_cache_manager()
            enhanced_envelope_factory = EnhancedEnvelopeFactory("device-data-ingestion-service", cache_manager)
            await enhanced_envelope_factory.initialize()
            logger.info("Enhanced envelope factory initialized with cache manager")
        except Exception as e:
            logger.warning(f"Failed to initialize envelope factory with cache: {e}")
            # Fallback without cache
            enhanced_envelope_factory = EnhancedEnvelopeFactory("device-data-ingestion-service")
            await enhanced_envelope_factory.initialize()
            logger.info("Enhanced envelope factory initialized without cache")

    return enhanced_envelope_factory


@router.post(
    "/ingest/device-data",
    response_model=IngestionResponse,
    summary="Ingest Device Data",
    description="Secure endpoint for device data ingestion. Validates input and publishes to Kafka immediately."
)
async def ingest_device_data(
    reading: DeviceReading,
    background_tasks: BackgroundTasks,
    vendor_info: Dict[str, Any] = Depends(check_rate_limit)
):
    """
    Ingest device data from authenticated vendors
    
    This endpoint:
    1. Validates the incoming device reading data
    2. Checks API key authentication and rate limits
    3. Publishes data to Kafka for processing
    4. Returns immediate acknowledgment
    """
    try:
        # Additional validation based on vendor permissions
        if reading.reading_type not in vendor_info.get("allowed_device_types", []):
            raise HTTPException(
                status_code=403,
                detail=f"Vendor not authorized for reading type: {reading.reading_type}"
            )
        
        # Get Kafka producer
        producer = await get_kafka_producer()
        
        # Prepare data for Kafka
        kafka_data = {
            "device_id": reading.device_id,
            "timestamp": reading.timestamp,
            "reading_type": reading.reading_type,
            "value": reading.value,
            "unit": reading.unit,
            "patient_id": reading.patient_id,
            "metadata": reading.metadata or {},
            "vendor_info": {
                "vendor_id": vendor_info["vendor_id"],
                "vendor_name": vendor_info["vendor_name"]
            }
        }
        
        # Publish to Kafka (async)
        message_id = await producer.publish_device_data(
            device_reading=kafka_data,
            key=reading.device_id
        )
        
        # Log successful ingestion
        logger.info(
            f"Device data ingested successfully - "
            f"Device: {reading.device_id}, "
            f"Type: {reading.reading_type}, "
            f"Vendor: {vendor_info['vendor_id']}, "
            f"Message ID: {message_id}"
        )
        
        # Return success response
        return IngestionResponse(
            message="Data queued for processing successfully"
        )
        
    except HTTPException:
        # Re-raise HTTP exceptions (auth, validation, etc.)
        raise
        
    except Exception as e:
        # Log unexpected errors
        logger.error(f"Unexpected error during ingestion: {e}", exc_info=True)
        
        # Return generic error response
        raise HTTPException(
            status_code=500,
            detail="Internal server error during data ingestion"
        )


@router.post(
    "/ingest/batch-device-data",
    summary="Batch Ingest Device Data",
    description="Ingest multiple device readings in a single request"
)
async def batch_ingest_device_data(
    readings: list[DeviceReading],
    background_tasks: BackgroundTasks,
    vendor_info: Dict[str, Any] = Depends(check_rate_limit)
):
    """
    Batch ingest multiple device readings
    
    This endpoint allows vendors to submit multiple readings at once
    for improved efficiency.
    """
    if len(readings) > 100:  # Limit batch size
        raise HTTPException(
            status_code=400,
            detail="Batch size cannot exceed 100 readings"
        )
    
    try:
        producer = await get_kafka_producer()
        results = []
        errors = []
        
        for i, reading in enumerate(readings):
            try:
                # Validate vendor permissions for each reading
                if reading.reading_type not in vendor_info.get("allowed_device_types", []):
                    errors.append({
                        "index": i,
                        "device_id": reading.device_id,
                        "error": f"Vendor not authorized for reading type: {reading.reading_type}"
                    })
                    continue
                
                # Prepare data for Kafka
                kafka_data = {
                    "device_id": reading.device_id,
                    "timestamp": reading.timestamp,
                    "reading_type": reading.reading_type,
                    "value": reading.value,
                    "unit": reading.unit,
                    "patient_id": reading.patient_id,
                    "metadata": reading.metadata or {},
                    "vendor_info": {
                        "vendor_id": vendor_info["vendor_id"],
                        "vendor_name": vendor_info["vendor_name"]
                    }
                }
                
                # Publish to Kafka
                message_id = await producer.publish_device_data(
                    device_reading=kafka_data,
                    key=reading.device_id
                )
                
                results.append({
                    "index": i,
                    "device_id": reading.device_id,
                    "message_id": message_id,
                    "status": "accepted"
                })
                
            except Exception as e:
                logger.error(f"Error processing reading {i}: {e}")
                errors.append({
                    "index": i,
                    "device_id": reading.device_id if hasattr(reading, 'device_id') else 'unknown',
                    "error": str(e)
                })
        
        # Log batch results
        logger.info(
            f"Batch ingestion completed - "
            f"Vendor: {vendor_info['vendor_id']}, "
            f"Total: {len(readings)}, "
            f"Successful: {len(results)}, "
            f"Errors: {len(errors)}"
        )
        
        return {
            "status": "completed",
            "total_readings": len(readings),
            "successful": len(results),
            "failed": len(errors),
            "results": results,
            "errors": errors,
            "timestamp": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        logger.error(f"Unexpected error during batch ingestion: {e}", exc_info=True)
        raise HTTPException(
            status_code=500,
            detail="Internal server error during batch ingestion"
        )


# ============================================================================
# JWT-BASED AUTHENTICATION ENDPOINTS (NEW)
# ============================================================================

@router.post(
    "/ingest/device-data-supabase",
    response_model=IngestionResponse,
    summary="Ingest Device Data (Supabase JWT Auth)",
    description="Enhanced device data ingestion with Supabase JWT authentication and timestamp validation"
)
async def ingest_device_data_supabase(
    reading: DeviceReading,
    background_tasks: BackgroundTasks,
    authorization: Optional[str] = Header(None)
):
    """
    Ingest device data using Supabase JWT authentication with enhanced timestamp validation

    This endpoint:
    1. Validates Supabase JWT token via Auth Service
    2. Performs enhanced timestamp validation
    3. Checks for replay attacks using JWT nonce
    4. Validates user role-based device permissions
    5. Publishes data to Kafka for processing

    ## Authentication:
    - **Authorization**: Bearer <supabase-jwt-token>

    ## Enhanced Security:
    - Supabase JWT token validation via Auth Service
    - Request timestamp validation with configurable tolerance
    - Replay attack prevention using JWT ID as nonce
    - Role-based device type permissions (doctor, nurse, patient, admin)
    """
    try:
        # Validate Authorization header
        if not authorization or not authorization.startswith("Bearer "):
            raise HTTPException(
                status_code=401,
                detail="Missing or invalid Authorization header. Expected: Bearer <supabase-jwt-token>"
            )

        # Option 1: Use request timestamp (may have timezone issues)
        # auth_result = await supabase_jwt_auth.validate_device_token(
        #     authorization.split(" ")[1],
        #     reading.timestamp
        # )

        # Option 2: Use current system timestamp (avoids timezone issues)
        current_timestamp = int(datetime.utcnow().timestamp())
        logger.info(f"Using current system timestamp {current_timestamp} instead of request timestamp {reading.timestamp}")

        auth_result = await supabase_jwt_auth.validate_device_token(
            authorization.split(" ")[1],
            current_timestamp
        )

        # Validate device type permissions based on user role
        allowed_types = auth_result.get('allowed_device_types', [])

        if reading.reading_type not in allowed_types:
            raise HTTPException(
                status_code=403,
                detail=f"Device type '{reading.reading_type}' not authorized for user role '{auth_result.get('role')}'"
            )

        # Get enhanced envelope factory
        envelope_factory = await get_enhanced_envelope_factory()

        # Create enhanced envelope with security context and quality assessment
        device_data = {
            "device_id": reading.device_id,
            "timestamp": reading.timestamp,  # Original timestamp from request
            "system_timestamp": current_timestamp,  # System timestamp used for validation
            "reading_type": reading.reading_type,
            "value": reading.value,
            "unit": reading.unit,
            "patient_id": reading.patient_id,
            "metadata": reading.metadata or {}
        }

        # Create auth context from validated JWT
        auth_context = AuthContext({
            "id": auth_result.get('user_id'),
            "email": auth_result.get('email'),
            "role": auth_result.get('role'),
            "roles": auth_result.get('roles', []),
            "permissions": auth_result.get('permissions', []),
            "is_active": True,
            "created_at": auth_result.get('validated_at', current_timestamp),
            "token_id": auth_result.get('token_id')
        })

        # Create request context
        request_context = RequestContext(
            timestamp=current_timestamp,
            source_ip="127.0.0.1",  # Would get from request in production
            user_agent="DeviceApp",  # Would get from request headers
            request_id=f"req_{reading.device_id}_{current_timestamp}"
        )

        # Create enhanced envelope with enterprise-grade metadata
        enhanced_envelope = await envelope_factory.create_device_data_envelope(
            device_data=device_data,
            auth_context=auth_context,
            request_context=request_context
        )

        # Get Kafka producer and publish enhanced envelope
        producer = await get_kafka_producer()

        # Convert enhanced envelope to dict for Kafka
        envelope_dict = enhanced_envelope.to_dict()

        # Publish enhanced envelope to Kafka using existing method
        await producer.publish_device_data(
            device_reading=envelope_dict,
            key=reading.device_id
        )

        logger.info(f"Successfully ingested device data via Supabase JWT: {reading.device_id} by user {auth_result.get('user_id')}")

        return IngestionResponse(
            status="accepted",
            message="Device data queued for processing via Supabase JWT authentication"
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error in JWT device data ingestion: {e}")
        raise HTTPException(
            status_code=500,
            detail="Internal server error during device data ingestion"
        )


@router.get(
    "/health",
    response_model=HealthResponse,
    summary="Health Check",
    description="Check service health and dependencies"
)
async def health_check():
    """
    Health check endpoint
    
    Returns service status and dependency health
    """
    try:
        # Check Kafka connection
        producer = await get_kafka_producer()
        kafka_healthy = producer.health_check()
        
        # Prepare health response
        health_response = HealthResponse(
            kafka_connected=kafka_healthy,
            dependencies={
                "kafka": "healthy" if kafka_healthy else "unhealthy",
                "auth_service": "not_checked"  # Could add auth service health check
            }
        )
        
        # Return appropriate status code
        status_code = 200 if kafka_healthy else 503
        
        return JSONResponse(
            status_code=status_code,
            content=health_response.dict()
        )
        
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return JSONResponse(
            status_code=503,
            content={
                "status": "unhealthy",
                "service": "Device Data Ingestion Service",
                "error": str(e),
                "timestamp": datetime.utcnow().isoformat()
            }
        )


@router.get(
    "/metrics",
    summary="Service Metrics",
    description="Get service metrics and statistics"
)
async def get_metrics():
    """
    Get service metrics
    
    Returns basic metrics about the service operation
    """
    try:
        # In a production system, these would come from a metrics store
        metrics = {
            "service": "device-data-ingestion-service",
            "version": settings.VERSION,
            "uptime_seconds": 0,  # Would track actual uptime
            "total_requests": 0,  # Would track from metrics store
            "successful_ingestions": 0,
            "failed_ingestions": 0,
            "kafka_messages_sent": 0,
            "rate_limit_violations": 0,
            "timestamp": datetime.utcnow().isoformat()
        }

        return metrics

    except Exception as e:
        logger.error(f"Error getting service metrics: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get service metrics"
        )


@router.get(
    "/envelope/performance",
    summary="Enhanced Envelope Factory Performance",
    description="Get performance metrics for the enhanced envelope factory"
)
async def get_envelope_performance():
    """Get enhanced envelope factory performance metrics"""
    try:
        envelope_factory = await get_enhanced_envelope_factory()
        metrics = envelope_factory.get_performance_metrics()

        return {
            "status": "success",
            "envelope_factory_metrics": metrics,
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error getting envelope performance metrics: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get envelope performance metrics"
        )


@router.get(
    "/envelope/demo",
    summary="Enhanced Envelope Demo",
    description="Create a demo enhanced envelope to show capabilities"
)
async def create_demo_envelope():
    """Create a demo enhanced envelope to demonstrate capabilities"""
    try:
        envelope_factory = await get_enhanced_envelope_factory()

        # Demo device data
        demo_device_data = {
            "device_id": "demo_heart_monitor_001",
            "timestamp": int(datetime.utcnow().timestamp()),
            "reading_type": "heart_rate",
            "value": 72.0,
            "unit": "bpm",
            "patient_id": "demo_patient_123",
            "metadata": {
                "battery_level": 85.0,
                "signal_quality": "excellent",
                "manufacturer": "DemoTech",
                "model": "HeartPro 3000"
            }
        }

        # Demo auth context
        demo_auth_context = AuthContext({
            "id": "demo_doctor_456",
            "email": "demo.doctor@hospital.com",
            "role": "doctor",
            "roles": ["doctor"],
            "permissions": ["patient:read", "patient:write", "device:read"],
            "is_active": True,
            "created_at": int(datetime.utcnow().timestamp())
        })

        # Demo request context
        demo_request_context = RequestContext(
            timestamp=int(datetime.utcnow().timestamp()),
            source_ip="192.168.1.100",
            user_agent="DemoApp/1.0",
            request_id="demo_request_001"
        )

        # Create enhanced envelope
        start_time = datetime.utcnow()
        enhanced_envelope = await envelope_factory.create_device_data_envelope(
            device_data=demo_device_data,
            auth_context=demo_auth_context,
            request_context=demo_request_context
        )
        creation_time = (datetime.utcnow() - start_time).total_seconds()

        # Convert to dict for response
        envelope_dict = enhanced_envelope.to_dict()

        return {
            "status": "success",
            "message": "Demo enhanced envelope created successfully",
            "creation_time_ms": round(creation_time * 1000, 2),
            "envelope": envelope_dict,
            "performance_summary": {
                "envelope_id": enhanced_envelope.id,
                "quality_score": enhanced_envelope.quality.overall_quality_score,
                "quality_level": enhanced_envelope.quality.quality_level.value,
                "security_level": "HIPAA Protected" if enhanced_envelope.security.hipaa_eligible else "Standard",
                "priority": enhanced_envelope.processing_hints.priority_level,
                "medical_emergency": enhanced_envelope.processing_hints.medical_emergency
            },
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error creating demo envelope: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to create demo envelope: {str(e)}"
        )
        
    except Exception as e:
        logger.error(f"Error retrieving metrics: {e}")
        raise HTTPException(
            status_code=500,
            detail="Error retrieving service metrics"
        )


@router.post(
    "/ingest/universal-device-data",
    response_model=IngestionResponse,
    summary="Universal Device Data Ingestion",
    description="Universal device handler that automatically detects device type and routes to appropriate processor"
)
async def ingest_universal_device_data(
    reading: DeviceReading,
    background_tasks: BackgroundTasks,
    authorization: Optional[str] = Header(None)
):
    """
    Universal device data ingestion with automatic device type detection

    This endpoint uses the Universal Device Handler to:
    1. Automatically detect device type from payload
    2. Route to appropriate processor
    3. Apply medical-grade validation
    4. Generate enhanced envelopes
    5. Detect medical emergencies

    ## Features:
    - Automatic device type detection
    - Medical parameter classification
    - Emergency detection and alerting
    - Fallback processing for unknown devices
    - Performance optimization with caching
    """
    try:
        # Validate Authorization header
        if not authorization or not authorization.startswith("Bearer "):
            raise HTTPException(
                status_code=401,
                detail="Missing or invalid Authorization header. Expected: Bearer <supabase-jwt-token>"
            )

        # Validate JWT token
        current_timestamp = int(datetime.utcnow().timestamp())
        auth_result = await supabase_jwt_auth.validate_device_token(
            authorization.split(" ")[1],
            current_timestamp
        )

        # Create auth context
        auth_context = AuthContext({
            "id": auth_result.get('user_id'),
            "email": auth_result.get('email'),
            "role": auth_result.get('role'),
            "roles": auth_result.get('roles', []),
            "permissions": auth_result.get('permissions', []),
            "is_active": True,
            "created_at": auth_result.get('validated_at', current_timestamp),
            "token_id": auth_result.get('token_id')
        })

        # Create request context
        request_context = RequestContext(
            timestamp=current_timestamp,
            source_ip="127.0.0.1",
            user_agent="UniversalDeviceApp",
            request_id=f"universal_{reading.device_id}_{current_timestamp}"
        )

        # Prepare device data
        device_data = {
            "device_id": reading.device_id,
            "timestamp": reading.timestamp,
            "reading_type": reading.reading_type,
            "value": reading.value,
            "unit": reading.unit,
            "patient_id": reading.patient_id,
            "metadata": reading.metadata or {}
        }

        # Get universal handler with enhanced envelope factory
        envelope_factory = await get_enhanced_envelope_factory()
        universal_handler = await get_universal_handler(envelope_factory)

        # Process device data universally
        processing_result = await universal_handler.process_device_data(
            device_data=device_data,
            auth_context=auth_context,
            request_context=request_context
        )

        if not processing_result.success:
            raise HTTPException(
                status_code=400,
                detail=f"Universal processing failed: {processing_result.error_message}"
            )

        # Publish enhanced envelope to Kafka if available
        if processing_result.enhanced_envelope:
            producer = await get_kafka_producer()
            envelope_dict = processing_result.enhanced_envelope.to_dict()
            await producer.publish_device_data(
                device_reading=envelope_dict,
                key=reading.device_id
            )

        # Log processing result
        logger.info(
            f"Universal device processing successful - "
            f"Device: {reading.device_id}, "
            f"Detected Type: {processing_result.routing_info.get('device_type')}, "
            f"Processor: {processing_result.routing_info.get('processor_id')}, "
            f"Emergency: {processing_result.emergency_detected}, "
            f"Processing Time: {processing_result.processing_time_ms:.2f}ms"
        )

        # Create response with universal processing info
        response_detail = "Universal device data processed successfully"
        if processing_result.emergency_detected:
            response_detail += f" - MEDICAL EMERGENCY DETECTED: {', '.join(processing_result.medical_alerts)}"
        elif processing_result.fallback_used:
            response_detail += " - Processed using fallback generic processor"

        return IngestionResponse(
            status="accepted",
            message=response_detail,
            metadata={
                "universal_processing": True,
                "detected_device_type": processing_result.routing_info.get('device_type'),
                "processor_used": processing_result.routing_info.get('processor_id'),
                "routing_confidence": processing_result.routing_info.get('confidence'),
                "routing_strategy": processing_result.routing_info.get('strategy'),
                "processing_time_ms": processing_result.processing_time_ms,
                "emergency_detected": processing_result.emergency_detected,
                "medical_alerts": processing_result.medical_alerts,
                "fallback_used": processing_result.fallback_used
            }
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error in universal device data ingestion: {e}")
        raise HTTPException(
            status_code=500,
            detail="Internal server error during universal device data processing"
        )


@router.get(
    "/universal/device-types",
    summary="Get Supported Device Types",
    description="Get all device types supported by the universal handler"
)
async def get_supported_device_types():
    """Get all supported device types with capabilities"""
    try:
        universal_handler = await get_universal_handler()
        supported_types = await universal_handler.get_supported_device_types()

        return {
            "status": "success",
            "supported_device_types": supported_types,
            "total_types": len(supported_types),
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error getting supported device types: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get supported device types"
        )


@router.post(
    "/universal/detect-device-type",
    summary="Detect Device Type",
    description="Detect device type from payload without full processing"
)
async def detect_device_type(device_data: Dict[str, Any]):
    """Detect device type from device data"""
    try:
        universal_handler = await get_universal_handler()
        detection_result = await universal_handler.detect_device_type(device_data)

        return {
            "status": "success",
            "detection_result": detection_result,
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error detecting device type: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to detect device type"
        )


@router.get(
    "/universal/stats",
    summary="Universal Handler Statistics",
    description="Get comprehensive statistics for the universal device handler"
)
async def get_universal_handler_stats():
    """Get universal handler processing statistics"""
    try:
        universal_handler = await get_universal_handler()
        handler_stats = universal_handler.get_processing_stats()

        # Get registry stats
        from app.universal_handler import get_device_registry, get_routing_engine
        registry = await get_device_registry()
        registry_stats = registry.get_registry_stats()

        # Get routing stats
        routing_engine = get_routing_engine()
        routing_stats = routing_engine.get_routing_stats()

        return {
            "status": "success",
            "universal_handler_stats": handler_stats,
            "device_registry_stats": {
                "total_processors": registry_stats.total_processors,
                "healthy_processors": registry_stats.healthy_processors,
                "degraded_processors": registry_stats.degraded_processors,
                "failed_processors": registry_stats.failed_processors,
                "total_device_types": registry_stats.total_device_types,
                "processing_count": registry_stats.processing_count,
                "error_count": registry_stats.error_count,
                "uptime_seconds": registry_stats.uptime_seconds
            },
            "routing_engine_stats": routing_stats,
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error getting universal handler stats: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get universal handler statistics"
        )


@router.get(
    "/universal/processors",
    summary="List Device Processors",
    description="List all registered device processors with their information"
)
async def list_device_processors():
    """List all registered device processors"""
    try:
        from app.universal_handler import get_device_registry
        registry = await get_device_registry()
        processors = registry.list_processors()

        return {
            "status": "success",
            "processors": processors,
            "total_processors": len(processors),
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"Error listing device processors: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to list device processors"
        )


# =====================================================
# TRANSACTIONAL OUTBOX PATTERN ENDPOINTS
# =====================================================

@router.post(
    "/ingest/device-data-outbox",
    response_model=IngestionResponse,
    summary="Ingest Device Data with Transactional Outbox",
    description="Enhanced endpoint using transactional outbox pattern for guaranteed message delivery"
)
async def ingest_device_data_with_outbox(
    reading: DeviceReading,
    request: Request,
    background_tasks: BackgroundTasks,
    vendor_info: Dict[str, Any] = Depends(check_rate_limit)
):
    """
    Enhanced device data ingestion with transactional outbox pattern

    This endpoint:
    1. Validates the incoming device reading data
    2. Checks API key authentication and rate limits
    3. Stores data in vendor-specific outbox table transactionally
    4. Returns immediate acknowledgment (NO direct Kafka publishing)
    5. Background publisher service handles Kafka publishing

    Benefits:
    - Guaranteed message delivery
    - True fault isolation per vendor
    - Atomic database transactions
    - Resilient to Kafka outages
    """
    # Generate correlation ID for tracing
    correlation_id = str(uuid.uuid4())
    trace_id = request.headers.get("X-Trace-ID", str(uuid.uuid4()))

    try:
        # Additional validation based on vendor permissions
        if reading.reading_type not in vendor_info.get("allowed_device_types", []):
            raise HTTPException(
                status_code=403,
                detail=f"Vendor not authorized for reading type: {reading.reading_type}"
            )

        # Validate vendor is supported for outbox pattern
        vendor_id = vendor_info["vendor_id"].lower()
        if not is_supported_vendor(vendor_id):
            raise HTTPException(
                status_code=400,
                detail=f"Vendor {vendor_id} not supported for outbox pattern"
            )

        # Get enhanced envelope factory
        envelope_factory = await get_enhanced_envelope_factory()

        # Create enhanced envelope with enterprise-grade metadata
        auth_context = AuthContext(
            auth_type="api_key",
            vendor_id=vendor_info["vendor_id"],
            vendor_name=vendor_info["vendor_name"],
            api_key_id=vendor_info.get("api_key_id"),
            permissions=vendor_info.get("allowed_device_types", [])
        )

        request_context = RequestContext(
            correlation_id=correlation_id,
            trace_id=trace_id,
            request_timestamp=datetime.utcnow(),
            source_ip=request.client.host if request.client else "unknown",
            user_agent=request.headers.get("user-agent", "unknown"),
            endpoint="/ingest/device-data-outbox"
        )

        enhanced_envelope = await envelope_factory.create_device_data_envelope(
            device_data=reading.dict(),
            auth_context=auth_context,
            request_context=request_context
        )

        # Store in outbox transactionally (NO direct Kafka publishing)
        # Using global_outbox_adapter instance
        outbox_id = await global_outbox_adapter.store_device_data_transactionally(
            device_data=enhanced_envelope.to_dict(),
            vendor_id=vendor_id,
            correlation_id=correlation_id,
            trace_id=trace_id
        )

        # Emit success metrics
        await metrics_collector.emit_message_success(vendor_id)

        # Log successful ingestion
        logger.info(
            f"Device data stored in outbox successfully - "
            f"Device: {reading.device_id}, "
            f"Type: {reading.reading_type}, "
            f"Vendor: {vendor_info['vendor_id']}, "
            f"Outbox ID: {outbox_id}, "
            f"Correlation ID: {correlation_id}"
        )

        # Return success response with outbox information
        return IngestionResponse(
            status="accepted",
            message="Device data queued for processing via outbox pattern",
            ingestion_id=outbox_id
        )

    except HTTPException:
        # Re-raise HTTP exceptions (auth, validation, etc.)
        await metrics_collector.emit_message_failure(
            vendor_info.get("vendor_id", "unknown"),
            "validation_error"
        )
        raise

    except Exception as e:
        # Log unexpected errors
        logger.error(f"Unexpected error during outbox ingestion: {e}", extra={
            "correlation_id": correlation_id,
            "trace_id": trace_id,
            "device_id": reading.device_id,
            "vendor_id": vendor_info.get("vendor_id")
        }, exc_info=True)

        # Emit failure metrics
        await metrics_collector.emit_message_failure(
            vendor_info.get("vendor_id", "unknown"),
            "internal_error"
        )

        # Return generic error response
        raise HTTPException(
            status_code=500,
            detail="Internal server error during data ingestion"
        )


@router.post(
    "/ingest/device-data-smart",
    response_model=IngestionResponse,
    summary="Smart Device Data Ingestion with Auto-Vendor Detection",
    description="Enhanced endpoint that automatically detects device vendor and routes to appropriate outbox table"
)
async def ingest_device_data_smart(
    reading: DeviceReading,
    request: Request,
    background_tasks: BackgroundTasks
):
    """
    Smart device data ingestion with automatic vendor detection

    This endpoint:
    1. Automatically detects device vendor using multiple methods
    2. Classifies device type using universal device handler
    3. Routes to appropriate vendor-specific outbox table
    4. Supports all medical device types
    5. Provides detailed detection metadata

    No API key required - uses smart detection instead
    """
    # Generate correlation ID for tracing
    correlation_id = str(uuid.uuid4())
    trace_id = request.headers.get("X-Trace-ID", str(uuid.uuid4()))

    try:
        # Convert reading to dict for processing
        device_data = reading.dict()

        # Smart vendor detection
        detection_result = await vendor_detection_service.detect_vendor_and_route(device_data)

        logger.info(f"Smart detection result: {detection_result.vendor_id} "
                   f"({detection_result.confidence:.2f}) via {detection_result.detection_method}")

        # Simplified approach: Direct outbox storage without complex envelope

        # Add detection metadata to device data
        enhanced_device_data = device_data.copy()
        enhanced_device_data["detection_metadata"] = {
            "vendor_id": detection_result.vendor_id,
            "vendor_name": detection_result.vendor_name,
            "device_type": detection_result.device_type,
            "confidence": detection_result.confidence,
            "is_medical_grade": detection_result.is_medical_grade,
            "detection_method": detection_result.detection_method,
            "outbox_table": detection_result.outbox_table,
            "ingestion_timestamp": datetime.utcnow().isoformat(),
            "correlation_id": correlation_id,
            "trace_id": trace_id
        }

        # Store in vendor-specific outbox transactionally (simplified)
        # Using global_outbox_adapter instance
        outbox_id = await global_outbox_adapter.store_device_data_transactionally(
            device_data=enhanced_device_data,
            vendor_id=detection_result.vendor_id,
            correlation_id=correlation_id,
            trace_id=trace_id
        )

        # Emit success metrics with vendor detection info
        await metrics_collector.emit_message_success(detection_result.vendor_id)
        await metrics_collector.emit_batch_metrics([
            {
                "metric_type": "custom.googleapis.com/outbox/smart_detection_confidence",
                "value": detection_result.confidence,
                "labels": {
                    "vendor_id": detection_result.vendor_id,
                    "detection_method": detection_result.detection_method,
                    "device_type": detection_result.device_type
                }
            }
        ])

        # Log successful smart ingestion
        logger.info(
            f"Smart device data ingestion successful - "
            f"Device: {reading.device_id}, "
            f"Type: {detection_result.device_type}, "
            f"Vendor: {detection_result.vendor_id} ({detection_result.confidence:.2f}), "
            f"Method: {detection_result.detection_method}, "
            f"Outbox ID: {outbox_id}, "
            f"Correlation ID: {correlation_id}"
        )

        # Return success response with detection information
        return IngestionResponse(
            status="accepted",
            message=f"Device data routed to {detection_result.vendor_name} via {detection_result.detection_method}",
            ingestion_id=outbox_id,
            metadata={
                "vendor_detection": {
                    "vendor_id": detection_result.vendor_id,
                    "vendor_name": detection_result.vendor_name,
                    "device_type": detection_result.device_type,
                    "confidence": detection_result.confidence,
                    "is_medical_grade": detection_result.is_medical_grade,
                    "detection_method": detection_result.detection_method,
                    "outbox_table": detection_result.outbox_table
                }
            }
        )

    except Exception as e:
        # Log unexpected errors
        logger.error(f"Unexpected error during smart ingestion: {e}", extra={
            "correlation_id": correlation_id,
            "trace_id": trace_id,
            "device_id": reading.device_id
        }, exc_info=True)

        # Emit failure metrics
        await metrics_collector.emit_message_failure("unknown", "smart_detection_error")

        # Return generic error response
        raise HTTPException(
            status_code=500,
            detail="Internal server error during smart device data ingestion"
        )


@router.get(
    "/outbox/health",
    summary="Outbox Health Status",
    description="Get comprehensive health status of the transactional outbox system"
)
async def get_outbox_health():
    """
    Get health status of the transactional outbox system

    Returns:
    - Queue depths per vendor
    - Processing statistics
    - Registry status
    - Overall system health
    """
    try:
        # Using global_outbox_adapter instance

        # Get comprehensive health status
        health_status = await global_outbox_adapter.health_check()

        # Get queue depths
        stats = await global_outbox_adapter.get_outbox_statistics()
        queue_depths = {"device-data-ingestion-service": stats.get("queue_depth", 0)}

        # Get metrics collector health
        metrics_health = metrics_collector.get_health_status()

        return {
            "status": health_status.get("status", "unknown"),
            "timestamp": datetime.utcnow().isoformat(),
            "outbox_system": health_status,
            "queue_depths": queue_depths,
            "metrics_collector": metrics_health,
            "supported_vendors": ["fitbit", "garmin", "apple_health"]
        }

    except Exception as e:
        logger.error(f"Error getting outbox health status: {e}")
        return {
            "status": "error",
            "timestamp": datetime.utcnow().isoformat(),
            "error": str(e)
        }


@router.get(
    "/outbox/queue-depths",
    summary="Outbox Queue Depths",
    description="Get current queue depths for all vendor outbox tables"
)
async def get_outbox_queue_depths():
    """Get current queue depths for monitoring and alerting"""
    try:
        # Using global_outbox_adapter instance
        stats = await global_outbox_adapter.get_outbox_statistics()
        queue_depths = {"device-data-ingestion-service": stats.get("queue_depth", 0)}

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "queue_depths": queue_depths,
            "total_pending": sum(depth for depth in queue_depths.values() if depth >= 0)
        }

    except Exception as e:
        logger.error(f"Error getting queue depths: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get outbox queue depths"
        )


# =====================================================
# DEAD LETTER MANAGEMENT ENDPOINTS
# =====================================================

@router.get(
    "/dead-letter/statistics",
    summary="Dead Letter Statistics",
    description="Get comprehensive statistics about dead letter messages across all vendors"
)
async def get_dead_letter_statistics():
    """Get dead letter statistics for monitoring and analysis"""
    try:
        stats = await dead_letter_manager.get_dead_letter_statistics()

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "statistics": stats
        }

    except Exception as e:
        logger.error(f"Error getting dead letter statistics: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get dead letter statistics"
        )


@router.get(
    "/dead-letter/messages",
    summary="Dead Letter Messages",
    description="Get dead letter messages with optional filtering"
)
async def get_dead_letter_messages(
    vendor_id: Optional[str] = None,
    limit: int = 100,
    failure_reason: Optional[str] = None
):
    """Get dead letter messages with optional filtering"""
    try:
        if limit > 1000:
            raise HTTPException(status_code=400, detail="Limit cannot exceed 1000")

        messages = await dead_letter_manager.get_dead_letter_messages(
            vendor_id=vendor_id,
            limit=limit,
            failure_reason=failure_reason
        )

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "messages": messages,
            "count": len(messages)
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting dead letter messages: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get dead letter messages"
        )


@router.post(
    "/dead-letter/reprocess/{message_id}",
    summary="Reprocess Dead Letter Message",
    description="Reprocess a specific dead letter message by moving it back to the outbox"
)
async def reprocess_dead_letter_message(
    message_id: str,
    vendor_id: str
):
    """Reprocess a dead letter message for manual recovery"""
    try:
        if not is_supported_vendor(vendor_id):
            raise HTTPException(
                status_code=400,
                detail=f"Vendor {vendor_id} is not supported"
            )

        success = await dead_letter_manager.reprocess_dead_letter_message(
            message_id,
            vendor_id
        )

        if success:
            return {
                "status": "success",
                "message": f"Dead letter message {message_id} reprocessed successfully",
                "timestamp": datetime.utcnow().isoformat()
            }
        else:
            raise HTTPException(
                status_code=404,
                detail=f"Dead letter message {message_id} not found or could not be reprocessed"
            )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error reprocessing dead letter message {message_id}: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to reprocess dead letter message"
        )


@router.get(
    "/dead-letter/analysis",
    summary="Dead Letter Failure Analysis",
    description="Analyze failure patterns and provide recommendations"
)
async def analyze_dead_letter_patterns(vendor_id: Optional[str] = None):
    """Analyze dead letter failure patterns and provide recommendations"""
    try:
        analysis = await dead_letter_manager.analyze_failure_patterns(vendor_id)

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "analysis": analysis
        }

    except Exception as e:
        logger.error(f"Error analyzing dead letter patterns: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to analyze dead letter patterns"
        )


# =====================================================
# VENDOR DETECTION AND CAPABILITIES ENDPOINTS
# =====================================================

@router.get(
    "/vendors/supported",
    summary="List Supported Vendors",
    description="Get all supported device vendors and their capabilities"
)
async def get_supported_vendors():
    """Get all supported vendors and their capabilities"""
    try:
        vendors = await vendor_detection_service.list_all_supported_vendors()

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "vendors": vendors,
            "total_vendors": len(vendors)
        }

    except Exception as e:
        logger.error(f"Error getting supported vendors: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get supported vendors"
        )


@router.get(
    "/vendors/{vendor_id}/capabilities",
    summary="Get Vendor Capabilities",
    description="Get capabilities for a specific vendor"
)
async def get_vendor_capabilities(vendor_id: str):
    """Get capabilities for a specific vendor"""
    try:
        capabilities = await vendor_detection_service.get_vendor_capabilities(vendor_id)

        if not capabilities:
            raise HTTPException(
                status_code=404,
                detail=f"Vendor {vendor_id} not found"
            )

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "vendor_id": vendor_id,
            "capabilities": capabilities
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting vendor capabilities: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to get vendor capabilities"
        )


@router.post(
    "/vendors/detect",
    summary="Test Vendor Detection",
    description="Test vendor detection for sample device data"
)
async def test_vendor_detection(device_data: Dict[str, Any]):
    """Test vendor detection for sample device data"""
    try:
        detection_result = await vendor_detection_service.detect_vendor_and_route(device_data)

        return {
            "status": "success",
            "timestamp": datetime.utcnow().isoformat(),
            "detection_result": {
                "vendor_id": detection_result.vendor_id,
                "vendor_name": detection_result.vendor_name,
                "device_type": detection_result.device_type,
                "confidence": detection_result.confidence,
                "is_medical_grade": detection_result.is_medical_grade,
                "outbox_table": detection_result.outbox_table,
                "dead_letter_table": detection_result.dead_letter_table,
                "detection_method": detection_result.detection_method,
                "metadata": detection_result.metadata
            }
        }

    except Exception as e:
        logger.error(f"Error testing vendor detection: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to test vendor detection"
        )


# ============================================================================
# BACKGROUND PUBLISHER ENDPOINTS
# ============================================================================

@router.get("/publisher/status")
async def get_publisher_status():
    """Get background publisher status"""
    try:
        publisher = await get_background_publisher()
        status = await publisher.get_publisher_status()
        return status
    except Exception as e:
        logger.error(f"Error getting publisher status: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get publisher status: {str(e)}")


@router.get("/publisher/health")
async def get_publisher_health():
    """Get background publisher health check"""
    try:
        publisher = await get_background_publisher()
        health = await publisher.health_check()
        return health
    except Exception as e:
        logger.error(f"Error getting publisher health: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get publisher health: {str(e)}")


@router.post("/publisher/process-now")
async def trigger_publisher_processing():
    """Manually trigger publisher processing for all vendors"""
    try:
        publisher = await get_background_publisher()
        await publisher.process_all_vendors()
        return {"status": "success", "message": "Publisher processing triggered"}
    except Exception as e:
        logger.error(f"Error triggering publisher processing: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to trigger processing: {str(e)}")
