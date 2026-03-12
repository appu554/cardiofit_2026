"""
Clinical Context Service Main Application
Implements the Clinical Context Recipe System with all three pillars:
1. Federated GraphQL API (The "Unified Data Graph")
2. Clinical Context Recipe System (The "Governance Engine")
3. Multi-Layer Intelligent Cache (The "Performance Accelerator")
"""
import logging
import asyncio
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import strawberry
from strawberry.fastapi import GraphQLRouter

# Import services
from app.services.context_assembly_service import ContextAssemblyService
from app.services.recipe_management_service import RecipeManagementService
from app.services.cache_service import CacheService
from app.services.recipe_governance import RecipeGovernance
from app.services.kafka_event_handler import CacheInvalidationService

# Import GraphQL schema
from app.api.graphql.schema import schema

# Import REST API endpoints
from app.api.endpoints import context, federation, snapshots

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ClinicalContextService:
    """
    Main Clinical Context Service application.
    Orchestrates all three pillars of the ratified architecture.
    """
    
    def __init__(self):
        # Initialize core services
        self.cache_service = CacheService()
        self.recipe_management_service = RecipeManagementService()
        self.context_assembly_service = ContextAssemblyService()
        self.governance = RecipeGovernance()
        self.cache_invalidation_service = CacheInvalidationService(self.cache_service)
        
        # Service status
        self.services_started = False
    
    async def start_services(self):
        """Start all clinical context services"""
        try:
            logger.info("🚀 Starting Clinical Context Service")
            logger.info("   Implementing the Three Pillars of Excellence:")
            logger.info("   1. Federated GraphQL API (The 'Unified Data Graph')")
            logger.info("   2. Clinical Context Recipe System (The 'Governance Engine')")
            logger.info("   3. Multi-Layer Intelligent Cache (The 'Performance Accelerator')")
            
            # Skip Kafka cache invalidation service for now
            logger.info("⚠️ Kafka cache invalidation service disabled for testing")
            logger.info("   Context Service will work without real-time cache invalidation")
            logger.info("   This is acceptable for Apollo Federation testing")
            
            # Load and validate recipes
            await self._validate_loaded_recipes()
            
            self.services_started = True
            logger.info("✅ Clinical Context Service started successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to start Clinical Context Service: {e}")
            raise
    
    async def stop_services(self):
        """Stop all clinical context services"""
        try:
            logger.info("🛑 Stopping Clinical Context Service")
            
            # Skip stopping cache invalidation service (disabled)
            logger.info("⚠️ Kafka cache invalidation service was disabled - nothing to stop")
            
            self.services_started = False
            logger.info("✅ Clinical Context Service stopped")
            
        except Exception as e:
            logger.error(f"❌ Error stopping Clinical Context Service: {e}")
    
    async def _validate_loaded_recipes(self):
        """Validate all loaded recipes meet governance requirements"""
        logger.info("🔍 Validating loaded recipes")
        
        valid_recipes = 0
        invalid_recipes = 0
        
        for recipe_id, recipe in self.recipe_management_service.loaded_recipes.items():
            try:
                validation_result = await self.recipe_management_service.validate_recipe(recipe)
                
                if validation_result["valid"]:
                    valid_recipes += 1
                    logger.debug(f"✅ Recipe valid: {recipe_id}")
                else:
                    invalid_recipes += 1
                    logger.warning(f"⚠️ Recipe invalid: {recipe_id} - {validation_result['errors']}")
                    
            except Exception as e:
                invalid_recipes += 1
                logger.error(f"❌ Recipe validation error: {recipe_id} - {e}")
        
        logger.info(f"📋 Recipe validation complete: {valid_recipes} valid, {invalid_recipes} invalid")

        if valid_recipes == 0:
            logger.warning("⚠️ No valid recipes found - service will start in testing mode")
            logger.warning("   The Context Service will work but recipe-based context assembly may be limited")
            logger.warning("   This is acceptable for testing the Patient Service connection")
    
    async def get_service_status(self):
        """Get comprehensive service status"""
        try:
            # Get cache statistics
            cache_stats = await self.cache_service.get_cache_stats()
            
            # Get invalidation statistics (optional if Kafka is not available)
            try:
                invalidation_stats = await self.cache_invalidation_service.get_invalidation_statistics()
            except Exception as e:
                logger.warning(f"⚠️ Could not get invalidation statistics: {e}")
                invalidation_stats = {
                    "kafka_events": {"events_processed": 0, "cache_invalidations": 0},
                    "manual_invalidations": 0,
                    "status": "kafka_unavailable"
                }
            
            # Get recipe statistics
            recipe_stats = {
                "total_recipes": len(self.recipe_management_service.loaded_recipes),
                "approved_recipes": sum(1 for r in self.recipe_management_service.loaded_recipes.values() if r.validate_governance()),
                "expired_recipes": sum(1 for r in self.recipe_management_service.loaded_recipes.values() if r.is_expired())
            }
            
            return {
                "service_status": "running" if self.services_started else "stopped",
                "timestamp": asyncio.get_event_loop().time(),
                "pillars": {
                    "pillar_1_graphql_api": {
                        "status": "active",
                        "endpoint": "/graphql",
                        "federation_enabled": True
                    },
                    "pillar_2_recipe_system": {
                        "status": "active",
                        "recipes": recipe_stats,
                        "governance_enabled": True
                    },
                    "pillar_3_intelligent_cache": {
                        "status": "active",
                        "performance": cache_stats,
                        "invalidation": invalidation_stats
                    }
                },
                "performance_metrics": {
                    "cache_hit_ratio": cache_stats.get("overall_hit_ratio", 0.0),
                    "l1_response_time_ms": cache_stats.get("performance", {}).get("l1_avg_response_time_ms", 0.0),
                    "l2_response_time_ms": cache_stats.get("performance", {}).get("l2_avg_response_time_ms", 0.0)
                }
            }
            
        except Exception as e:
            logger.error(f"❌ Error getting service status: {e}")
            return {
                "service_status": "error",
                "error": str(e),
                "timestamp": asyncio.get_event_loop().time()
            }


# Global service instance
clinical_context_service = ClinicalContextService()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """FastAPI lifespan context manager"""
    # Startup
    await clinical_context_service.start_services()
    yield
    # Shutdown
    await clinical_context_service.stop_services()


# Create FastAPI application
app = FastAPI(
    title="Clinical Context Service",
    description="Federated Clinical Data Intelligence Hub implementing the Three Pillars of Excellence",
    version="2.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Create GraphQL router
graphql_app = GraphQLRouter(schema)

# Mount GraphQL endpoint (Pillar 1: Federated GraphQL API)
app.include_router(graphql_app, prefix="/graphql")

# Mount REST API endpoints for internal service communication
app.include_router(context.router, prefix="/api/context", tags=["Context API"])

# Mount Clinical Snapshot endpoints (Recipe Snapshot Architecture)
app.include_router(snapshots.router, prefix="/api", tags=["Clinical Snapshots"])

# Mount Apollo Federation endpoint
app.include_router(federation.router, prefix="/api/federation", tags=["Federation"])


@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": "Clinical Context Service",
        "version": "2.0.0",
        "description": "Federated Clinical Data Intelligence Hub",
        "pillars": [
            "Pillar 1: Federated GraphQL API (The 'Unified Data Graph')",
            "Pillar 2: Clinical Context Recipe System (The 'Governance Engine')",
            "Pillar 3: Multi-Layer Intelligent Cache (The 'Performance Accelerator')"
        ],
        "endpoints": {
            "graphql": "/graphql",
            "health": "/health",
            "status": "/status",
            "metrics": "/metrics",
            "rest_api": {
                "patient_context": "/api/context/patient/{patient_id}/context",
                "patient_demographics": "/api/context/patient/{patient_id}/demographics",
                "patient_medications": "/api/context/patient/{patient_id}/medications",
                "context_status": "/api/context/status"
            },
            "snapshot_api": {
                "create_snapshot": "POST /api/snapshots",
                "get_snapshot": "GET /api/snapshots/{snapshot_id}",
                "validate_snapshot": "POST /api/snapshots/{snapshot_id}/validate",
                "delete_snapshot": "DELETE /api/snapshots/{snapshot_id}",
                "list_snapshots": "GET /api/snapshots",
                "snapshot_metrics": "GET /api/snapshots/metrics",
                "patient_summary": "GET /api/snapshots/patient/{patient_id}/summary",
                "batch_create": "POST /api/snapshots/batch-create"
            }
        }
    }


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    try:
        if not clinical_context_service.services_started:
            raise HTTPException(status_code=503, detail="Services not started")
        
        # Basic health checks
        cache_healthy = clinical_context_service.cache_service is not None
        recipes_loaded = len(clinical_context_service.recipe_management_service.loaded_recipes) > 0
        
        if cache_healthy and recipes_loaded:
            return {
                "status": "healthy",
                "timestamp": asyncio.get_event_loop().time(),
                "checks": {
                    "cache_service": "healthy",
                    "recipe_service": "healthy",
                    "governance_service": "healthy"
                }
            }
        else:
            raise HTTPException(status_code=503, detail="Service components unhealthy")
            
    except Exception as e:
        logger.error(f"❌ Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Health check failed: {str(e)}")


@app.get("/status")
async def service_status():
    """Detailed service status endpoint"""
    return await clinical_context_service.get_service_status()


@app.get("/metrics")
async def service_metrics():
    """Service metrics endpoint for monitoring"""
    try:
        status = await clinical_context_service.get_service_status()
        
        # Extract key metrics for monitoring systems
        metrics = {
            "cache_hit_ratio": status.get("performance_metrics", {}).get("cache_hit_ratio", 0.0),
            "l1_response_time_ms": status.get("performance_metrics", {}).get("l1_response_time_ms", 0.0),
            "l2_response_time_ms": status.get("performance_metrics", {}).get("l2_response_time_ms", 0.0),
            "total_recipes": status.get("pillars", {}).get("pillar_2_recipe_system", {}).get("recipes", {}).get("total_recipes", 0),
            "approved_recipes": status.get("pillars", {}).get("pillar_2_recipe_system", {}).get("recipes", {}).get("approved_recipes", 0),
            "service_uptime": status.get("timestamp", 0),
            "kafka_events_processed": status.get("pillars", {}).get("pillar_3_intelligent_cache", {}).get("invalidation", {}).get("kafka_events", {}).get("events_processed", 0),
            "cache_invalidations": status.get("pillars", {}).get("pillar_3_intelligent_cache", {}).get("invalidation", {}).get("kafka_events", {}).get("cache_invalidations", 0)
        }
        
        return {
            "service": "clinical-context-service",
            "timestamp": asyncio.get_event_loop().time(),
            "metrics": metrics
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting metrics: {e}")
        raise HTTPException(status_code=500, detail=f"Metrics error: {str(e)}")


@app.post("/admin/invalidate-cache")
async def admin_invalidate_cache(patient_id: str, reason: str = "admin_request"):
    """Admin endpoint for manual cache invalidation"""
    try:
        await clinical_context_service.cache_invalidation_service.manual_invalidate_patient(
            patient_id, reason
        )
        
        return {
            "status": "success",
            "message": f"Cache invalidated for patient {patient_id}",
            "reason": reason,
            "timestamp": asyncio.get_event_loop().time()
        }
        
    except Exception as e:
        logger.error(f"❌ Manual cache invalidation failed: {e}")
        raise HTTPException(status_code=500, detail=f"Cache invalidation failed: {str(e)}")


@app.get("/admin/recipes")
async def admin_list_recipes():
    """Admin endpoint for listing all recipes"""
    try:
        recipes = []
        
        for recipe_id, recipe in clinical_context_service.recipe_management_service.loaded_recipes.items():
            recipe_info = {
                "recipe_id": recipe.recipe_id,
                "recipe_name": recipe.recipe_name,
                "version": recipe.version,
                "clinical_scenario": recipe.clinical_scenario,
                "workflow_category": recipe.workflow_category,
                "execution_pattern": recipe.execution_pattern,
                "sla_ms": recipe.sla_ms,
                "governance_approved": recipe.validate_governance(),
                "expired": recipe.is_expired(),
                "data_points_count": len(recipe.required_data_points),
                "conditional_rules_count": len(recipe.conditional_rules)
            }
            recipes.append(recipe_info)
        
        return {
            "total_recipes": len(recipes),
            "recipes": recipes,
            "timestamp": asyncio.get_event_loop().time()
        }
        
    except Exception as e:
        logger.error(f"❌ Error listing recipes: {e}")
        raise HTTPException(status_code=500, detail=f"Recipe listing failed: {str(e)}")


if __name__ == "__main__":
    import uvicorn
    
    logger.info("🚀 Starting Clinical Context Service")
    logger.info("   Port: 8016")
    logger.info("   GraphQL Endpoint: http://localhost:8016/graphql")
    logger.info("   Health Check: http://localhost:8016/health")
    
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8016,
        reload=True,
        log_level="info"
    )
