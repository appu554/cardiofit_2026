"""
Clinical Snapshot REST API Endpoints

This module provides REST API endpoints for the Recipe Snapshot architecture,
enabling creation, retrieval, validation, and management of immutable clinical snapshots.
"""

from fastapi import APIRouter, HTTPException, Query, Path, Body
from typing import Optional, List
import logging
from datetime import datetime

from app.models.snapshot_models import (
    SnapshotRequest,
    ClinicalSnapshot,
    SnapshotValidationResult,
    SnapshotSummary,
    SnapshotMetrics,
    SnapshotStatus,
    SignatureMethod
)
from app.services.snapshot_service import SnapshotService

logger = logging.getLogger(__name__)

router = APIRouter()

# Global snapshot service instance
snapshot_service = SnapshotService()


@router.post("/snapshots", response_model=ClinicalSnapshot)
async def create_snapshot(request: SnapshotRequest):
    """
    Create an immutable clinical snapshot with cryptographic integrity.
    
    This endpoint creates a snapshot of clinical data for a patient using a specific recipe.
    The snapshot includes SHA-256 checksum and digital signature for integrity verification.
    
    Args:
        request: Snapshot creation request with patient ID, recipe ID, and options
        
    Returns:
        Created clinical snapshot with integrity verification
    """
    try:
        logger.info(f"🔍 Creating snapshot for patient {request.patient_id} using recipe {request.recipe_id}")
        
        # Validate request parameters
        if not request.patient_id:
            raise HTTPException(status_code=400, detail="Patient ID is required")
        if not request.recipe_id:
            raise HTTPException(status_code=400, detail="Recipe ID is required")
        
        # Create the snapshot
        snapshot = await snapshot_service.create_snapshot(request)
        
        logger.info(f"✅ Created snapshot {snapshot.id} for patient {request.patient_id}")
        
        return snapshot
        
    except ValueError as e:
        logger.error(f"❌ Validation error creating snapshot: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"❌ Error creating snapshot: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error creating snapshot: {str(e)}")


@router.get("/snapshots/{snapshot_id}", response_model=ClinicalSnapshot)
async def get_snapshot(
    snapshot_id: str = Path(..., description="Snapshot ID to retrieve")
):
    """
    Retrieve a clinical snapshot by ID.
    
    This endpoint retrieves an immutable clinical snapshot and marks it as accessed.
    The snapshot includes all clinical data with integrity verification information.
    
    Args:
        snapshot_id: The unique snapshot ID
        
    Returns:
        Clinical snapshot with complete data and metadata
    """
    try:
        logger.info(f"🔍 Retrieving snapshot {snapshot_id}")
        
        snapshot = await snapshot_service.get_snapshot(snapshot_id)
        
        if not snapshot:
            raise HTTPException(status_code=404, detail=f"Snapshot {snapshot_id} not found")
        
        # Check if snapshot is still valid
        if snapshot.is_expired():
            logger.warning(f"⚠️ Snapshot {snapshot_id} has expired")
            raise HTTPException(
                status_code=410, 
                detail=f"Snapshot {snapshot_id} has expired at {snapshot.expires_at}"
            )
        
        logger.info(f"✅ Retrieved snapshot {snapshot_id} (access count: {snapshot.accessed_count})")
        
        return snapshot
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"❌ Error retrieving snapshot {snapshot_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving snapshot: {str(e)}")


@router.post("/snapshots/{snapshot_id}/validate", response_model=SnapshotValidationResult)
async def validate_snapshot(
    snapshot_id: str = Path(..., description="Snapshot ID to validate")
):
    """
    Validate snapshot integrity and authenticity.
    
    This endpoint performs comprehensive validation of a clinical snapshot including:
    - Data integrity check (SHA-256 checksum)
    - Digital signature verification
    - Expiration status check
    - Clinical safety validation
    
    Args:
        snapshot_id: The unique snapshot ID to validate
        
    Returns:
        Validation result with detailed integrity information
    """
    try:
        logger.info(f"🔍 Validating snapshot {snapshot_id}")
        
        validation_result = await snapshot_service.validate_snapshot(snapshot_id)
        
        if not validation_result.valid:
            logger.warning(f"⚠️ Snapshot {snapshot_id} validation failed: {validation_result.errors}")
        else:
            logger.info(f"✅ Snapshot {snapshot_id} validation passed")
        
        return validation_result
        
    except Exception as e:
        logger.error(f"❌ Error validating snapshot {snapshot_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error validating snapshot: {str(e)}")


@router.delete("/snapshots/{snapshot_id}")
async def delete_snapshot(
    snapshot_id: str = Path(..., description="Snapshot ID to delete")
):
    """
    Delete a clinical snapshot.
    
    This endpoint permanently deletes a clinical snapshot from storage.
    Note: Snapshots with TTL will be automatically cleaned up by MongoDB.
    
    Args:
        snapshot_id: The unique snapshot ID to delete
        
    Returns:
        Deletion confirmation
    """
    try:
        logger.info(f"🗑️ Deleting snapshot {snapshot_id}")
        
        deleted = await snapshot_service.delete_snapshot(snapshot_id)
        
        if not deleted:
            raise HTTPException(status_code=404, detail=f"Snapshot {snapshot_id} not found")
        
        logger.info(f"✅ Deleted snapshot {snapshot_id}")
        
        return {
            "status": "success",
            "message": f"Snapshot {snapshot_id} deleted successfully",
            "snapshot_id": snapshot_id,
            "deleted_at": datetime.utcnow().isoformat()
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"❌ Error deleting snapshot {snapshot_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error deleting snapshot: {str(e)}")


@router.get("/snapshots", response_model=List[SnapshotSummary])
async def list_snapshots(
    patient_id: Optional[str] = Query(None, description="Filter by patient ID"),
    provider_id: Optional[str] = Query(None, description="Filter by provider ID"),
    recipe_id: Optional[str] = Query(None, description="Filter by recipe ID"),
    status: Optional[SnapshotStatus] = Query(None, description="Filter by status"),
    limit: int = Query(50, ge=1, le=200, description="Maximum number of snapshots to return")
):
    """
    List clinical snapshots with optional filtering.
    
    This endpoint returns a list of snapshot summaries with optional filtering
    by patient, provider, recipe, or status.
    
    Args:
        patient_id: Optional patient ID filter
        provider_id: Optional provider ID filter  
        recipe_id: Optional recipe ID filter
        status: Optional status filter
        limit: Maximum number of results (1-200)
        
    Returns:
        List of snapshot summaries matching the filters
    """
    try:
        logger.info(f"🔍 Listing snapshots (limit: {limit})")
        
        summaries = await snapshot_service.list_snapshots(
            patient_id=patient_id,
            provider_id=provider_id,
            recipe_id=recipe_id,
            status=status,
            limit=limit
        )
        
        logger.info(f"✅ Listed {len(summaries)} snapshots")
        
        return summaries
        
    except Exception as e:
        logger.error(f"❌ Error listing snapshots: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error listing snapshots: {str(e)}")


@router.get("/snapshots/metrics", response_model=SnapshotMetrics)
async def get_snapshot_metrics():
    """
    Get comprehensive snapshot service metrics.
    
    This endpoint returns detailed metrics about snapshot usage including:
    - Total, active, and expired snapshot counts
    - Average completeness scores and TTL
    - Creation and access rates
    - Top recipes and providers
    
    Returns:
        Comprehensive snapshot metrics
    """
    try:
        logger.info("🔍 Generating snapshot metrics")
        
        metrics = await snapshot_service.get_metrics()
        
        logger.info(f"✅ Generated snapshot metrics: {metrics.total_snapshots} total snapshots")
        
        return metrics
        
    except Exception as e:
        logger.error(f"❌ Error generating snapshot metrics: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error generating metrics: {str(e)}")


@router.post("/snapshots/cleanup")
async def cleanup_expired_snapshots():
    """
    Manually cleanup expired snapshots.
    
    This endpoint triggers manual cleanup of expired snapshots.
    Note: TTL indexes should handle automatic cleanup, but this provides manual override.
    
    Returns:
        Cleanup results
    """
    try:
        logger.info("🧹 Starting manual snapshot cleanup")
        
        deleted_count = await snapshot_service.cleanup_expired_snapshots()
        
        logger.info(f"✅ Cleanup completed: {deleted_count} snapshots removed")
        
        return {
            "status": "success",
            "message": f"Cleaned up {deleted_count} expired snapshots",
            "deleted_count": deleted_count,
            "cleaned_at": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        logger.error(f"❌ Error during snapshot cleanup: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error during cleanup: {str(e)}")


@router.get("/snapshots/patient/{patient_id}/summary")
async def get_patient_snapshot_summary(
    patient_id: str = Path(..., description="Patient ID"),
    limit: int = Query(10, ge=1, le=50, description="Maximum number of recent snapshots")
):
    """
    Get snapshot summary for a specific patient.
    
    This endpoint returns a summary of recent snapshots for a patient including
    usage patterns and data completeness trends.
    
    Args:
        patient_id: The patient ID
        limit: Maximum number of recent snapshots to include
        
    Returns:
        Patient-specific snapshot summary
    """
    try:
        logger.info(f"🔍 Getting snapshot summary for patient {patient_id}")
        
        # Get recent snapshots for the patient
        recent_snapshots = await snapshot_service.list_snapshots(
            patient_id=patient_id,
            limit=limit
        )
        
        # Calculate summary statistics
        if not recent_snapshots:
            return {
                "patient_id": patient_id,
                "total_snapshots": 0,
                "active_snapshots": 0,
                "average_completeness": 0.0,
                "recent_snapshots": [],
                "most_used_recipes": [],
                "generated_at": datetime.utcnow().isoformat()
            }
        
        total_snapshots = len(recent_snapshots)
        active_snapshots = len([s for s in recent_snapshots if s.status == SnapshotStatus.ACTIVE])
        average_completeness = sum(s.completeness_score for s in recent_snapshots) / total_snapshots
        
        # Count recipe usage
        recipe_usage = {}
        for snapshot in recent_snapshots:
            recipe_usage[snapshot.recipe_id] = recipe_usage.get(snapshot.recipe_id, 0) + 1
        
        most_used_recipes = [
            {"recipe_id": recipe_id, "usage_count": count}
            for recipe_id, count in sorted(recipe_usage.items(), key=lambda x: x[1], reverse=True)
        ]
        
        summary = {
            "patient_id": patient_id,
            "total_snapshots": total_snapshots,
            "active_snapshots": active_snapshots,
            "average_completeness": round(average_completeness, 4),
            "recent_snapshots": [
                {
                    "snapshot_id": s.id,
                    "recipe_id": s.recipe_id,
                    "created_at": s.created_at.isoformat(),
                    "expires_at": s.expires_at.isoformat(),
                    "completeness_score": s.completeness_score,
                    "accessed_count": s.accessed_count,
                    "status": s.status.value
                }
                for s in recent_snapshots[:5]  # Show top 5
            ],
            "most_used_recipes": most_used_recipes[:3],  # Show top 3
            "generated_at": datetime.utcnow().isoformat()
        }
        
        logger.info(f"✅ Generated snapshot summary for patient {patient_id}")
        
        return summary
        
    except Exception as e:
        logger.error(f"❌ Error getting patient snapshot summary: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error generating patient summary: {str(e)}")


@router.post("/snapshots/batch-create")
async def batch_create_snapshots(
    requests: List[SnapshotRequest] = Body(..., description="List of snapshot creation requests")
):
    """
    Create multiple clinical snapshots in batch.
    
    This endpoint allows creation of multiple snapshots simultaneously for efficiency.
    Each snapshot is created independently - partial failures are reported.
    
    Args:
        requests: List of snapshot creation requests
        
    Returns:
        Batch creation results with successes and failures
    """
    try:
        if len(requests) > 20:
            raise HTTPException(status_code=400, detail="Maximum 20 snapshots per batch request")
        
        logger.info(f"🔍 Batch creating {len(requests)} snapshots")
        
        results = {
            "total_requested": len(requests),
            "successful": [],
            "failed": [],
            "created_at": datetime.utcnow().isoformat()
        }
        
        for i, request in enumerate(requests):
            try:
                snapshot = await snapshot_service.create_snapshot(request)
                results["successful"].append({
                    "index": i,
                    "snapshot_id": snapshot.id,
                    "patient_id": snapshot.patient_id,
                    "recipe_id": snapshot.recipe_id
                })
            except Exception as e:
                results["failed"].append({
                    "index": i,
                    "patient_id": request.patient_id,
                    "recipe_id": request.recipe_id,
                    "error": str(e)
                })
        
        success_count = len(results["successful"])
        failure_count = len(results["failed"])
        
        logger.info(f"✅ Batch creation completed: {success_count} successful, {failure_count} failed")
        
        # Return appropriate status code
        if failure_count == 0:
            return results
        elif success_count == 0:
            raise HTTPException(status_code=400, detail="All snapshot creations failed")
        else:
            # Partial success - return 207 Multi-Status would be ideal, but use 200 with detailed response
            return results
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"❌ Error in batch snapshot creation: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error in batch creation: {str(e)}")


@router.get("/snapshots/status")
async def get_snapshot_service_status():
    """
    Get snapshot service status and health information.
    
    Returns:
        Service status, configuration, and operational metrics
    """
    try:
        metrics = await snapshot_service.get_metrics()
        
        return {
            "service": "clinical-snapshot-service",
            "status": "healthy",
            "version": "1.0.0",
            "features": [
                "immutable-clinical-snapshots",
                "cryptographic-integrity-verification",
                "ttl-based-lifecycle-management",
                "recipe-based-data-assembly",
                "digital-signatures",
                "sha256-checksums"
            ],
            "endpoints": [
                "POST /api/snapshots - Create snapshot",
                "GET /api/snapshots/{id} - Retrieve snapshot",
                "POST /api/snapshots/{id}/validate - Validate snapshot",
                "DELETE /api/snapshots/{id} - Delete snapshot",
                "GET /api/snapshots - List snapshots",
                "GET /api/snapshots/metrics - Service metrics",
                "POST /api/snapshots/cleanup - Manual cleanup",
                "GET /api/snapshots/patient/{id}/summary - Patient summary",
                "POST /api/snapshots/batch-create - Batch creation"
            ],
            "current_metrics": {
                "total_snapshots": metrics.total_snapshots,
                "active_snapshots": metrics.active_snapshots,
                "expired_snapshots": metrics.expired_snapshots,
                "average_completeness": round(metrics.average_completeness, 4),
                "creation_rate_per_hour": metrics.creation_rate_per_hour,
                "access_rate_per_hour": metrics.access_rate_per_hour
            },
            "signature_methods_supported": [method.value for method in SignatureMethod],
            "ttl_range_hours": "1-24",
            "timestamp": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting snapshot service status: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error getting service status: {str(e)}")