"""
Context Service Client - Shows EXACTLY where context data is connected.
This client connects the Workflow Engine to the real Context Service and data sources.
"""
import logging
import aiohttp
import asyncio
from typing import Dict, List, Optional, Any
from datetime import datetime
import json

from app.models.clinical_activity_models import ClinicalContext, ClinicalDataError

logger = logging.getLogger(__name__)


class ContextServiceClient:
    """
    Client that connects Workflow Engine to real Context Service.
    This shows EXACTLY where the context data connections happen.
    """
    
    def __init__(self):
        # REAL SERVICE ENDPOINTS - These are the actual connections
        self.context_service_url = "http://localhost:8016"  # Context Service
        self.graphql_endpoint = f"{self.context_service_url}/graphql"
        self.rest_endpoint = f"{self.context_service_url}/api"
        
        # Connection settings
        self.timeout = aiohttp.ClientTimeout(connect=10, total=30)
        
        # These are the REAL data source endpoints that Context Service connects to
        self.real_data_sources = {
            "patient_service": "http://localhost:8003",
            "medication_service": "http://localhost:8009", 
            "fhir_store": "projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store",
            "lab_service": "http://localhost:8000",
            "cae_service": "http://localhost:8027",
            "context_service": "http://localhost:8016"
        }
        
        logger.info("🔗 Context Service Client initialized with REAL endpoints:")
        for name, endpoint in self.real_data_sources.items():
            logger.info(f"   {name}: {endpoint}")
    
    async def get_clinical_context_by_recipe(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get clinical context using recipe - REAL CONNECTION to Context Service.
        This shows exactly how the Workflow Engine connects to get real clinical data.
        """
        try:
            logger.info(f"🌐 REAL CONNECTION: Getting clinical context via recipe")
            logger.info(f"   Patient ID: {patient_id}")
            logger.info(f"   Recipe ID: {recipe_id}")
            logger.info(f"   Context Service: {self.graphql_endpoint}")
            
            # GraphQL query to Context Service
            query = """
            query GetContextByRecipe($patientId: ID!, $recipeId: String!, $providerId: ID, $encounterId: ID) {
                getContextByRecipe(
                    patientId: $patientId,
                    recipeId: $recipeId,
                    providerId: $providerId,
                    encounterId: $encounterId
                ) {
                    contextId
                    patientId
                    recipeUsed
                    assembledData
                    completenessScore
                    dataFreshness
                    sourceMetadata
                    safetyFlags {
                        flagType
                        severity
                        message
                    }
                    governanceTags
                    connectionErrors
                    assembledAt
                }
            }
            """
            
            variables = {
                "patientId": patient_id,
                "recipeId": recipe_id,
                "providerId": provider_id,
                "encounterId": encounter_id
            }
            
            # REAL HTTP POST to Context Service GraphQL endpoint
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                payload = {
                    "query": query,
                    "variables": variables
                }
                
                headers = {
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                }
                
                logger.info(f"🌐 REAL HTTP POST: {self.graphql_endpoint}")
                logger.debug(f"   Payload: {json.dumps(payload, indent=2)}")
                
                async with session.post(
                    self.graphql_endpoint,
                    json=payload,
                    headers=headers
                ) as response:
                    
                    if response.status == 200:
                        result = await response.json()
                        
                        if "errors" in result:
                            error_msg = "; ".join([err["message"] for err in result["errors"]])
                            logger.error(f"❌ GraphQL errors: {error_msg}")
                            raise ClinicalDataError(f"Context Service GraphQL error: {error_msg}")
                        
                        context_data = result["data"]["getContextByRecipe"]
                        
                        # Convert to ClinicalContext object
                        clinical_context = ClinicalContext(
                            patient_id=context_data["patientId"],
                            encounter_id=context_data.get("encounterId"),
                            provider_id=context_data.get("providerId"),
                            clinical_data=context_data["assembledData"],
                            data_sources=self.real_data_sources,  # Show real endpoints
                            workflow_context={
                                "context_id": context_data["contextId"],
                                "recipe_used": context_data["recipeUsed"],
                                "completeness_score": context_data["completenessScore"],
                                "data_freshness": context_data["dataFreshness"],
                                "source_metadata": context_data["sourceMetadata"],
                                "safety_flags": context_data["safetyFlags"],
                                "governance_tags": context_data["governanceTags"],
                                "connection_errors": context_data.get("connectionErrors", []),
                                "assembled_at": context_data["assembledAt"]
                            }
                        )
                        
                        logger.info(f"✅ REAL CONTEXT RETRIEVED:")
                        logger.info(f"   Context ID: {context_data['contextId']}")
                        logger.info(f"   Completeness: {context_data['completenessScore']:.2%}")
                        logger.info(f"   Data Sources: {len(context_data['sourceMetadata'])}")
                        logger.info(f"   Safety Flags: {len(context_data['safetyFlags'])}")
                        
                        # Log which real services were contacted
                        source_metadata = context_data["sourceMetadata"]
                        logger.info(f"   REAL SERVICES CONTACTED:")
                        for source_name, metadata in source_metadata.items():
                            endpoint = metadata.get("source_endpoint", "unknown")
                            logger.info(f"     {source_name}: {endpoint}")
                        
                        return clinical_context
                        
                    else:
                        error_text = await response.text()
                        logger.error(f"❌ Context Service HTTP {response.status}: {error_text}")
                        raise ClinicalDataError(f"Context Service returned HTTP {response.status}")
                        
        except asyncio.TimeoutError:
            logger.error(f"❌ TIMEOUT connecting to Context Service: {self.graphql_endpoint}")
            raise ClinicalDataError("Context Service timeout")
        except aiohttp.ClientError as e:
            logger.error(f"❌ CONNECTION ERROR to Context Service: {e}")
            raise ClinicalDataError(f"Context Service connection failed: {str(e)}")
        except Exception as e:
            logger.error(f"❌ CONTEXT RETRIEVAL FAILED: {e}")
            raise ClinicalDataError(f"Failed to get clinical context: {str(e)}")
    
    async def get_context_fields(
        self,
        patient_id: str,
        fields: List[str],
        provider_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Get specific context fields - REAL CONNECTION for domain services.
        """
        try:
            logger.info(f"🌐 REAL CONNECTION: Getting specific context fields")
            logger.info(f"   Patient ID: {patient_id}")
            logger.info(f"   Fields: {fields}")
            
            # GraphQL query for specific fields
            query = """
            query GetContextFields($patientId: ID!, $fields: [String!]!, $providerId: ID) {
                getContextFields(
                    patientId: $patientId,
                    fields: $fields,
                    providerId: $providerId
                ) {
                    data
                    completeness
                    metadata
                }
            }
            """
            
            variables = {
                "patientId": patient_id,
                "fields": fields,
                "providerId": provider_id
            }
            
            # REAL HTTP POST to Context Service
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                payload = {
                    "query": query,
                    "variables": variables
                }
                
                headers = {
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                }
                
                logger.info(f"🌐 REAL HTTP POST: {self.graphql_endpoint}")
                
                async with session.post(
                    self.graphql_endpoint,
                    json=payload,
                    headers=headers
                ) as response:
                    
                    if response.status == 200:
                        result = await response.json()
                        
                        if "errors" in result:
                            error_msg = "; ".join([err["message"] for err in result["errors"]])
                            raise ClinicalDataError(f"Context Service GraphQL error: {error_msg}")
                        
                        field_data = result["data"]["getContextFields"]
                        
                        logger.info(f"✅ CONTEXT FIELDS RETRIEVED:")
                        logger.info(f"   Completeness: {field_data['completeness']:.2%}")
                        logger.info(f"   Fields Retrieved: {len(field_data['data'])}")
                        
                        return field_data
                        
                    else:
                        raise ClinicalDataError(f"Context Service returned HTTP {response.status}")
                        
        except Exception as e:
            logger.error(f"❌ FIELD RETRIEVAL FAILED: {e}")
            raise ClinicalDataError(f"Failed to get context fields: {str(e)}")
    
    async def validate_context_availability(
        self,
        patient_id: str,
        recipe_id: str
    ) -> Dict[str, Any]:
        """
        Validate context availability - REAL CONNECTION check.
        """
        try:
            logger.info(f"🔍 REAL CONNECTION: Validating context availability")
            logger.info(f"   Patient ID: {patient_id}")
            logger.info(f"   Recipe ID: {recipe_id}")
            
            # REST API call to Context Service
            url = f"{self.rest_endpoint}/context/availability"
            params = {
                "patient_id": patient_id,
                "recipe_id": recipe_id
            }
            
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                logger.info(f"🌐 REAL HTTP GET: {url}")
                
                async with session.get(url, params=params) as response:
                    if response.status == 200:
                        availability = await response.json()
                        
                        logger.info(f"✅ AVAILABILITY CHECK COMPLETE:")
                        logger.info(f"   Available: {availability['available']}")
                        
                        if availability['available']:
                            data_sources = availability.get('data_sources', {})
                            available_sources = sum(1 for source in data_sources.values() if source.get('available'))
                            total_sources = len(data_sources)
                            logger.info(f"   Data Sources: {available_sources}/{total_sources} available")
                            
                            # Log which real services are available
                            logger.info(f"   REAL SERVICE STATUS:")
                            for source_name, source_info in data_sources.items():
                                status = "✅ AVAILABLE" if source_info.get('available') else "❌ UNAVAILABLE"
                                endpoint = source_info.get('endpoint', 'unknown')
                                logger.info(f"     {source_name}: {status} ({endpoint})")
                        else:
                            logger.warning(f"   Error: {availability.get('error', 'Unknown')}")
                        
                        return availability
                        
                    else:
                        raise ClinicalDataError(f"Context Service returned HTTP {response.status}")
                        
        except Exception as e:
            logger.error(f"❌ AVAILABILITY CHECK FAILED: {e}")
            return {
                "available": False,
                "error": str(e),
                "checked_at": datetime.utcnow().isoformat()
            }
    
    async def invalidate_context_cache(
        self,
        patient_id: str,
        recipe_id: Optional[str] = None
    ) -> bool:
        """
        Invalidate context cache - REAL CONNECTION to Context Service.
        """
        try:
            logger.info(f"🔄 REAL CONNECTION: Invalidating context cache")
            logger.info(f"   Patient ID: {patient_id}")
            logger.info(f"   Recipe ID: {recipe_id}")
            
            # REST API call to Context Service
            url = f"{self.rest_endpoint}/context/cache/invalidate"
            payload = {
                "patient_id": patient_id,
                "recipe_id": recipe_id
            }
            
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                logger.info(f"🌐 REAL HTTP POST: {url}")
                
                async with session.post(url, json=payload) as response:
                    if response.status == 200:
                        result = await response.json()
                        invalidated_count = result.get('invalidated_entries', 0)
                        
                        logger.info(f"✅ CACHE INVALIDATED: {invalidated_count} entries")
                        return True
                        
                    else:
                        logger.error(f"❌ Cache invalidation failed: HTTP {response.status}")
                        return False
                        
        except Exception as e:
            logger.error(f"❌ CACHE INVALIDATION FAILED: {e}")
            return False
    
    async def get_context_service_health(self) -> Dict[str, Any]:
        """
        Check Context Service health and data source connectivity.
        """
        try:
            logger.info(f"🏥 REAL CONNECTION: Checking Context Service health")
            
            # Health check endpoint
            url = f"{self.rest_endpoint}/health"
            
            async with aiohttp.ClientSession(timeout=self.timeout) as session:
                logger.info(f"🌐 REAL HTTP GET: {url}")
                
                async with session.get(url) as response:
                    if response.status == 200:
                        health = await response.json()
                        
                        logger.info(f"✅ CONTEXT SERVICE HEALTH:")
                        logger.info(f"   Status: {health.get('status', 'unknown')}")
                        logger.info(f"   Version: {health.get('version', 'unknown')}")
                        
                        # Log data source connectivity
                        data_sources = health.get('data_sources', {})
                        logger.info(f"   REAL DATA SOURCE CONNECTIVITY:")
                        for source_name, source_health in data_sources.items():
                            status = source_health.get('status', 'unknown')
                            endpoint = source_health.get('endpoint', 'unknown')
                            response_time = source_health.get('response_time_ms', 'unknown')
                            logger.info(f"     {source_name}: {status} ({endpoint}) - {response_time}ms")
                        
                        return health
                        
                    else:
                        logger.error(f"❌ Health check failed: HTTP {response.status}")
                        return {
                            "status": "unhealthy",
                            "error": f"HTTP {response.status}"
                        }
                        
        except Exception as e:
            logger.error(f"❌ HEALTH CHECK FAILED: {e}")
            return {
                "status": "unhealthy",
                "error": str(e)
            }


# Global context service client instance
context_service_client = ContextServiceClient()
