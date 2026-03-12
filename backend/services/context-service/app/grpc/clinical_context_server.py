"""
gRPC Server Implementation for Clinical Context Service.
High-performance gRPC interface for clinical context assembly.
"""
import logging
import asyncio
from typing import AsyncIterator
import grpc
from grpc import aio
from google.protobuf.timestamp_pb2 import Timestamp
from google.protobuf.struct_pb2 import Struct
from datetime import datetime

# Import generated protobuf classes
import clinical_context_pb2
import clinical_context_pb2_grpc

from app.services.context_assembly_service import ContextAssemblyService
from app.services.recipe_management_service import RecipeManagementService
from app.services.cache_service import CacheService
from app.models.context_models import ClinicalContext, ContextRecipe

logger = logging.getLogger(__name__)


class ClinicalContextServicer(clinical_context_pb2_grpc.ClinicalContextServiceServicer):
    """
    gRPC servicer for Clinical Context Service.
    Provides high-performance context assembly for workflow engine.
    """
    
    def __init__(self):
        self.context_assembly_service = ContextAssemblyService()
        self.recipe_service = RecipeManagementService()
        self.cache_service = CacheService()
        
        logger.info("🚀 Clinical Context gRPC Servicer initialized")
    
    async def GetContextByRecipe(
        self,
        request: clinical_context_pb2.GetContextByRecipeRequest,
        context: grpc.aio.ServicerContext
    ) -> clinical_context_pb2.ClinicalContextResponse:
        """
        Get clinical context using recipe-based approach via gRPC.
        """
        try:
            logger.info(f"🔍 gRPC GetContextByRecipe request:")
            logger.info(f"   Patient ID: {request.patient_id}")
            logger.info(f"   Recipe ID: {request.recipe_id}")
            logger.info(f"   Provider ID: {request.provider_id}")
            logger.info(f"   Force Refresh: {request.force_refresh}")
            
            # Load recipe
            recipe = await self.recipe_service.load_recipe(
                recipe_id=request.recipe_id
            )
            
            if not recipe:
                await context.abort(
                    grpc.StatusCode.NOT_FOUND,
                    f"Recipe not found: {request.recipe_id}"
                )
            
            # Assemble clinical context
            clinical_context = await self.context_assembly_service.assemble_context(
                patient_id=request.patient_id,
                recipe=recipe,
                provider_id=request.provider_id if request.provider_id else None,
                encounter_id=request.encounter_id if request.encounter_id else None,
                force_refresh=request.force_refresh
            )
            
            # Convert to protobuf response
            response = await self._convert_to_protobuf_response(clinical_context)
            
            logger.info(f"✅ gRPC context assembled successfully")
            logger.info(f"   Context ID: {response.context_id}")
            logger.info(f"   Completeness: {response.completeness_score:.2%}")
            logger.info(f"   Safety Flags: {len(response.safety_flags)}")
            
            return response
            
        except Exception as e:
            logger.error(f"❌ gRPC GetContextByRecipe failed: {e}")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                f"Context assembly failed: {str(e)}"
            )
    
    async def GetContextFields(
        self,
        request: clinical_context_pb2.GetContextFieldsRequest,
        context: grpc.aio.ServicerContext
    ) -> clinical_context_pb2.ContextFieldsResponse:
        """
        Get specific context fields for domain services via gRPC.
        """
        try:
            logger.info(f"🔍 gRPC GetContextFields request:")
            logger.info(f"   Patient ID: {request.patient_id}")
            logger.info(f"   Fields: {list(request.fields)}")
            
            # Create dynamic recipe for requested fields
            dynamic_recipe = await self.recipe_service.create_dynamic_recipe(
                fields=list(request.fields)
            )
            
            # Assemble context
            clinical_context = await self.context_assembly_service.assemble_context(
                patient_id=request.patient_id,
                recipe=dynamic_recipe,
                provider_id=request.provider_id if request.provider_id else None
            )
            
            # Convert assembled data to protobuf Struct
            data_struct = Struct()
            data_struct.update(clinical_context.assembled_data)
            
            metadata_struct = Struct()
            metadata_struct.update(clinical_context.source_metadata)
            
            response = clinical_context_pb2.ContextFieldsResponse(
                data=data_struct,
                completeness=clinical_context.completeness_score,
                metadata=metadata_struct,
                status=clinical_context_pb2.CONTEXT_STATUS_SUCCESS
            )
            
            logger.info(f"✅ gRPC context fields retrieved successfully")
            logger.info(f"   Fields Retrieved: {len(clinical_context.assembled_data)}")
            logger.info(f"   Completeness: {clinical_context.completeness_score:.2%}")
            
            return response
            
        except Exception as e:
            logger.error(f"❌ gRPC GetContextFields failed: {e}")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                f"Context fields retrieval failed: {str(e)}"
            )
    
    async def ValidateContextAvailability(
        self,
        request: clinical_context_pb2.ValidateContextAvailabilityRequest,
        context: grpc.aio.ServicerContext
    ) -> clinical_context_pb2.ContextAvailabilityResponse:
        """
        Validate context availability before workflow execution via gRPC.
        """
        try:
            logger.info(f"🔍 gRPC ValidateContextAvailability request:")
            logger.info(f"   Patient ID: {request.patient_id}")
            logger.info(f"   Recipe ID: {request.recipe_id}")
            
            # Load recipe
            recipe = await self.recipe_service.load_recipe(request.recipe_id)
            
            if not recipe:
                response = clinical_context_pb2.ContextAvailabilityResponse(
                    available=False,
                    patient_id=request.patient_id,
                    recipe_id=request.recipe_id,
                    error=f"Recipe not found: {request.recipe_id}",
                    checked_at=self._datetime_to_timestamp(datetime.utcnow())
                )
                return response
            
            # Check data source availability
            availability_results = await self._check_data_source_availability(
                recipe, request.patient_id
            )
            
            # Determine overall availability
            all_available = all(
                result.available for result in availability_results.values()
            )
            
            # Convert to protobuf response
            data_sources_pb = {}
            for source_name, availability in availability_results.items():
                data_sources_pb[source_name] = clinical_context_pb2.DataSourceAvailability(
                    available=availability.available,
                    endpoint=availability.endpoint,
                    error=availability.error if availability.error else "",
                    response_time_ms=availability.response_time_ms
                )
            
            response = clinical_context_pb2.ContextAvailabilityResponse(
                available=all_available,
                patient_id=request.patient_id,
                recipe_id=request.recipe_id,
                data_sources=data_sources_pb,
                required_data=[dp.name for dp in recipe.required_data_points],
                checked_at=self._datetime_to_timestamp(datetime.utcnow())
            )
            
            logger.info(f"✅ gRPC availability check complete: {'Available' if all_available else 'Unavailable'}")
            
            return response
            
        except Exception as e:
            logger.error(f"❌ gRPC ValidateContextAvailability failed: {e}")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                f"Availability validation failed: {str(e)}"
            )
    
    async def InvalidateContextCache(
        self,
        request: clinical_context_pb2.InvalidateContextCacheRequest,
        context: grpc.aio.ServicerContext
    ) -> clinical_context_pb2.InvalidateContextCacheResponse:
        """
        Invalidate context cache for real-time updates via gRPC.
        """
        try:
            logger.info(f"🔄 gRPC InvalidateContextCache request:")
            logger.info(f"   Patient ID: {request.patient_id}")
            logger.info(f"   Recipe ID: {request.recipe_id}")
            
            # Invalidate cache
            invalidated_count = await self.cache_service.invalidate_pattern(
                pattern=f"context:{request.patient_id}:*" if not request.recipe_id 
                       else f"context:{request.patient_id}:{request.recipe_id}"
            )
            
            response = clinical_context_pb2.InvalidateContextCacheResponse(
                success=True,
                invalidated_entries=invalidated_count
            )
            
            logger.info(f"✅ gRPC cache invalidated: {invalidated_count} entries")
            
            return response
            
        except Exception as e:
            logger.error(f"❌ gRPC InvalidateContextCache failed: {e}")
            response = clinical_context_pb2.InvalidateContextCacheResponse(
                success=False,
                invalidated_entries=0,
                error=str(e)
            )
            return response
    
    async def GetServiceHealth(
        self,
        request: clinical_context_pb2.GetServiceHealthRequest,
        context: grpc.aio.ServicerContext
    ) -> clinical_context_pb2.ServiceHealthResponse:
        """
        Get context service health and data source connectivity via gRPC.
        """
        try:
            logger.info(f"🏥 gRPC GetServiceHealth request")
            
            # Check service health
            service_status = clinical_context_pb2.SERVICE_STATUS_HEALTHY
            
            # Get cache stats
            cache_stats = await self.cache_service.get_stats()
            cache_stats_pb = clinical_context_pb2.CacheStats(
                total_entries=cache_stats.get('total_entries', 0),
                hit_ratio=cache_stats.get('hit_ratio', 0.0),
                l1_entries=cache_stats.get('l1_entries', 0),
                l2_entries=cache_stats.get('l2_entries', 0),
                last_updated=self._datetime_to_timestamp(datetime.utcnow())
            )
            
            # Check data sources if requested
            data_sources_health = {}
            if request.include_data_sources:
                data_sources_health = await self._check_all_data_sources_health()
            
            response = clinical_context_pb2.ServiceHealthResponse(
                status=service_status,
                version="1.0.0",
                timestamp=self._datetime_to_timestamp(datetime.utcnow()),
                data_sources=data_sources_health,
                cache_stats=cache_stats_pb
            )
            
            logger.info(f"✅ gRPC service health check complete")
            
            return response
            
        except Exception as e:
            logger.error(f"❌ gRPC GetServiceHealth failed: {e}")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                f"Health check failed: {str(e)}"
            )
    
    async def StreamContextUpdates(
        self,
        request: clinical_context_pb2.StreamContextUpdatesRequest,
        context: grpc.aio.ServicerContext
    ) -> AsyncIterator[clinical_context_pb2.ContextUpdateEvent]:
        """
        Stream context updates for real-time workflows via gRPC.
        """
        try:
            logger.info(f"📡 gRPC StreamContextUpdates request:")
            logger.info(f"   Patient ID: {request.patient_id}")
            logger.info(f"   Recipe IDs: {list(request.recipe_ids)}")
            
            # Set up streaming (simplified implementation)
            while not context.cancelled():
                # In a real implementation, this would listen to actual data changes
                # For now, simulate periodic updates
                await asyncio.sleep(30)  # Check every 30 seconds
                
                # Create update event
                update_event = clinical_context_pb2.ContextUpdateEvent(
                    patient_id=request.patient_id,
                    recipe_id=request.recipe_ids[0] if request.recipe_ids else "",
                    update_type=clinical_context_pb2.CONTEXT_UPDATE_TYPE_DATA_CHANGED,
                    timestamp=self._datetime_to_timestamp(datetime.utcnow())
                )
                
                yield update_event
                
        except Exception as e:
            logger.error(f"❌ gRPC StreamContextUpdates failed: {e}")
            await context.abort(
                grpc.StatusCode.INTERNAL,
                f"Context streaming failed: {str(e)}"
            )
    
    # Helper methods
    async def _convert_to_protobuf_response(
        self,
        clinical_context: ClinicalContext
    ) -> clinical_context_pb2.ClinicalContextResponse:
        """
        Convert ClinicalContext to protobuf response.
        """
        # Convert assembled data to protobuf Struct
        assembled_data_struct = Struct()
        assembled_data_struct.update(clinical_context.assembled_data)
        
        # Convert data freshness to protobuf Struct
        data_freshness_struct = Struct()
        freshness_data = {}
        for key, timestamp in clinical_context.data_freshness.items():
            freshness_data[key] = timestamp.isoformat() if hasattr(timestamp, 'isoformat') else str(timestamp)
        data_freshness_struct.update(freshness_data)
        
        # Convert source metadata to protobuf Struct
        source_metadata_struct = Struct()
        source_metadata_struct.update(clinical_context.source_metadata)
        
        # Convert safety flags
        safety_flags_pb = []
        for flag in clinical_context.safety_flags:
            safety_flag_pb = clinical_context_pb2.SafetyFlag(
                flag_type=self._convert_safety_flag_type(flag.flag_type),
                severity=self._convert_safety_severity(flag.severity),
                message=flag.message
            )
            safety_flags_pb.append(safety_flag_pb)
        
        # Convert connection errors
        connection_errors_pb = []
        for error in clinical_context.connection_errors:
            error_pb = clinical_context_pb2.ConnectionError(
                data_point=error.get('data_point', ''),
                source=error.get('source', ''),
                error=error.get('error', ''),
                timestamp=self._datetime_to_timestamp(datetime.utcnow())
            )
            connection_errors_pb.append(error_pb)
        
        response = clinical_context_pb2.ClinicalContextResponse(
            context_id=clinical_context.context_id,
            patient_id=clinical_context.patient_id,
            provider_id=clinical_context.provider_id or "",
            encounter_id=clinical_context.encounter_id or "",
            recipe_used=clinical_context.recipe_used,
            assembled_data=assembled_data_struct,
            completeness_score=clinical_context.completeness_score,
            data_freshness=data_freshness_struct,
            source_metadata=source_metadata_struct,
            safety_flags=safety_flags_pb,
            governance_tags=clinical_context.governance_tags,
            connection_errors=connection_errors_pb,
            assembled_at=self._datetime_to_timestamp(clinical_context.assembled_at),
            status=clinical_context_pb2.CONTEXT_STATUS_SUCCESS
        )
        
        return response
    
    def _datetime_to_timestamp(self, dt: datetime) -> Timestamp:
        """Convert datetime to protobuf Timestamp."""
        timestamp = Timestamp()
        timestamp.FromDatetime(dt)
        return timestamp
    
    def _convert_safety_flag_type(self, flag_type: str) -> int:
        """Convert safety flag type to protobuf enum."""
        mapping = {
            "drug_interaction": clinical_context_pb2.SAFETY_FLAG_TYPE_DRUG_INTERACTION,
            "allergy_alert": clinical_context_pb2.SAFETY_FLAG_TYPE_ALLERGY_ALERT,
            "dosage_warning": clinical_context_pb2.SAFETY_FLAG_TYPE_DOSAGE_WARNING,
            "contraindication": clinical_context_pb2.SAFETY_FLAG_TYPE_CONTRAINDICATION,
            "data_quality": clinical_context_pb2.SAFETY_FLAG_TYPE_DATA_QUALITY,
            "stale_data": clinical_context_pb2.SAFETY_FLAG_TYPE_STALE_DATA
        }
        return mapping.get(flag_type, clinical_context_pb2.SAFETY_FLAG_TYPE_UNKNOWN)
    
    def _convert_safety_severity(self, severity: str) -> int:
        """Convert safety severity to protobuf enum."""
        mapping = {
            "info": clinical_context_pb2.SAFETY_SEVERITY_INFO,
            "warning": clinical_context_pb2.SAFETY_SEVERITY_WARNING,
            "critical": clinical_context_pb2.SAFETY_SEVERITY_CRITICAL,
            "fatal": clinical_context_pb2.SAFETY_SEVERITY_FATAL
        }
        return mapping.get(severity, clinical_context_pb2.SAFETY_SEVERITY_UNKNOWN)
    
    async def _check_data_source_availability(self, recipe: ContextRecipe, patient_id: str):
        """Check availability of data sources for recipe."""
        # Simplified implementation - would check actual data sources
        availability_results = {}
        
        for data_point in recipe.required_data_points:
            availability_results[data_point.source_type.value] = type('Availability', (), {
                'available': True,
                'endpoint': f"http://localhost:800{hash(data_point.source_type.value) % 10}",
                'error': None,
                'response_time_ms': 50
            })()
        
        return availability_results
    
    async def _check_all_data_sources_health(self):
        """Check health of all data sources."""
        # Simplified implementation
        data_sources_health = {}
        
        sources = [
            ("patient_service", "http://localhost:8003"),
            ("medication_service", "http://localhost:8009"),
            ("lab_service", "http://localhost:8000"),
            ("cae_service", "http://localhost:8027")
        ]
        
        for source_name, endpoint in sources:
            data_sources_health[source_name] = clinical_context_pb2.DataSourceHealth(
                status=clinical_context_pb2.SERVICE_STATUS_HEALTHY,
                endpoint=endpoint,
                response_time_ms=50,
                last_check=self._datetime_to_timestamp(datetime.utcnow())
            )
        
        return data_sources_health


async def serve():
    """
    Start the gRPC server for Clinical Context Service.
    """
    server = aio.server()
    
    # Add the servicer
    clinical_context_pb2_grpc.add_ClinicalContextServiceServicer_to_server(
        ClinicalContextServicer(), server
    )
    
    # Configure server
    listen_addr = '[::]:50051'  # gRPC port
    server.add_insecure_port(listen_addr)
    
    logger.info(f"🚀 Starting Clinical Context gRPC Server on {listen_addr}")
    
    await server.start()
    
    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("🛑 Shutting down Clinical Context gRPC Server")
        await server.stop(5)


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    asyncio.run(serve())
