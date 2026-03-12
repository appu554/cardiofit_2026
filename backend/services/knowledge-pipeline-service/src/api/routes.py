"""
API Routes for Knowledge Pipeline Service
"""

import asyncio
from typing import Dict, Any, Optional
from fastapi import APIRouter, HTTPException, BackgroundTasks, Query
from pydantic import BaseModel
import structlog

from core.pipeline_orchestrator import PipelineOrchestrator


logger = structlog.get_logger(__name__)

# Router instance
pipeline_router = APIRouter(prefix="/pipeline", tags=["pipeline"])

# Global orchestrator reference (set by main.py)
orchestrator: Optional[PipelineOrchestrator] = None


def set_orchestrator(pipeline_orchestrator: PipelineOrchestrator):
    """Set the global orchestrator reference"""
    global orchestrator
    orchestrator = pipeline_orchestrator


class PipelineRunRequest(BaseModel):
    """Request model for pipeline execution"""
    force_download: bool = False
    sources: Optional[list] = None  # If None, run all sources


class IngesterRunRequest(BaseModel):
    """Request model for single ingester execution"""
    force_download: bool = False


@pipeline_router.get("/status")
async def get_pipeline_status():
    """Get current pipeline status"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        status = await orchestrator.get_status()
        return status
    
    except Exception as e:
        logger.error("Error getting pipeline status", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.post("/run")
async def run_pipeline(
    request: PipelineRunRequest,
    background_tasks: BackgroundTasks
):
    """Start full pipeline execution"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        # Check if pipeline is already running
        if orchestrator.current_execution:
            raise HTTPException(
                status_code=409, 
                detail="Pipeline is already running"
            )
        
        # Start pipeline in background
        background_tasks.add_task(
            orchestrator.run_full_pipeline,
            force_download=request.force_download
        )
        
        return {
            "message": "Pipeline execution started",
            "force_download": request.force_download
        }
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Error starting pipeline", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.post("/run/{source_name}")
async def run_single_ingester(
    source_name: str,
    request: IngesterRunRequest,
    background_tasks: BackgroundTasks
):
    """Run a single ingester"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        # Validate source name
        if source_name not in orchestrator.ingesters:
            available_sources = list(orchestrator.ingesters.keys())
            raise HTTPException(
                status_code=404,
                detail=f"Unknown source '{source_name}'. Available sources: {available_sources}"
            )
        
        # Run ingester
        result = await orchestrator.run_single_ingester(
            source_name=source_name,
            force_download=request.force_download
        )
        
        return {
            "message": f"Ingester '{source_name}' completed",
            "result": result
        }
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Error running single ingester", 
                    source=source_name, 
                    error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.post("/cancel")
async def cancel_pipeline():
    """Cancel current pipeline execution"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        if not orchestrator.current_execution:
            raise HTTPException(status_code=404, detail="No pipeline execution is currently running")
        
        await orchestrator.cancel_current_execution()
        
        return {"message": "Pipeline execution cancelled"}
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Error cancelling pipeline", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.get("/history")
async def get_execution_history(limit: int = Query(default=10, ge=1, le=50)):
    """Get pipeline execution history"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        history = [exec.to_dict() for exec in orchestrator.execution_history[-limit:]]
        
        return {
            "executions": history,
            "total_count": len(orchestrator.execution_history)
        }
    
    except Exception as e:
        logger.error("Error getting execution history", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.get("/sources")
async def get_available_sources():
    """Get list of available data sources"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        sources = {}
        for name, ingester in orchestrator.ingesters.items():
            status = await ingester.get_status()
            sources[name] = {
                "name": name,
                "status": status,
                "description": getattr(ingester, '__doc__', 'No description available')
            }
        
        return {
            "sources": sources,
            "ingestion_order": orchestrator.ingestion_order
        }
    
    except Exception as e:
        logger.error("Error getting available sources", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.get("/harmonization/stats")
async def get_harmonization_stats():
    """Get harmonization statistics"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        stats = await orchestrator.harmonization_engine.get_harmonization_stats()
        
        return {
            "harmonization_stats": stats
        }
    
    except Exception as e:
        logger.error("Error getting harmonization stats", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.get("/validate")
async def validate_pipeline():
    """Validate pipeline configuration and dependencies"""
    try:
        if not orchestrator:
            raise HTTPException(status_code=503, detail="Pipeline orchestrator not initialized")
        
        validation_results = {
            "graphdb_connection": False,
            "ingesters_initialized": False,
            "harmonization_engine": False,
            "data_directories": False,
            "errors": []
        }
        
        # Test GraphDB connection
        try:
            validation_results["graphdb_connection"] = await orchestrator.graphdb_client.test_connection()
        except Exception as e:
            validation_results["errors"].append(f"GraphDB connection failed: {str(e)}")
        
        # Check ingesters
        try:
            validation_results["ingesters_initialized"] = len(orchestrator.ingesters) > 0
        except Exception as e:
            validation_results["errors"].append(f"Ingesters check failed: {str(e)}")
        
        # Check harmonization engine
        try:
            harmony_stats = await orchestrator.harmonization_engine.get_harmonization_stats()
            validation_results["harmonization_engine"] = True
        except Exception as e:
            validation_results["errors"].append(f"Harmonization engine check failed: {str(e)}")
        
        # Check data directories
        try:
            from pathlib import Path
            from core.config import settings
            
            dirs_exist = all([
                Path(settings.DATA_DIR).exists(),
                Path(settings.TEMP_DIR).exists(),
                Path(settings.CACHE_DIR).exists()
            ])
            validation_results["data_directories"] = dirs_exist
            
            if not dirs_exist:
                validation_results["errors"].append("Some data directories do not exist")
        
        except Exception as e:
            validation_results["errors"].append(f"Data directories check failed: {str(e)}")
        
        # Overall validation status
        validation_results["overall_status"] = (
            validation_results["graphdb_connection"] and
            validation_results["ingesters_initialized"] and
            validation_results["harmonization_engine"] and
            validation_results["data_directories"]
        )
        
        return validation_results
    
    except Exception as e:
        logger.error("Error validating pipeline", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@pipeline_router.get("/health")
async def health_check():
    """Health check endpoint for the pipeline service"""
    try:
        if not orchestrator:
            return {
                "status": "unhealthy",
                "reason": "Pipeline orchestrator not initialized"
            }
        
        # Quick health checks
        graphdb_healthy = await orchestrator.graphdb_client.test_connection()
        
        return {
            "status": "healthy" if graphdb_healthy else "degraded",
            "graphdb_connection": graphdb_healthy,
            "ingesters_count": len(orchestrator.ingesters),
            "current_execution": orchestrator.current_execution is not None
        }
    
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        return {
            "status": "unhealthy",
            "error": str(e)
        }
