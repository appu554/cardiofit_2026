"""
Direct Elasticsearch Data Source for Clinical Context Service
Bypasses microservices and connects directly to Elasticsearch for better performance
"""
import logging
import json
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
from elasticsearch import AsyncElasticsearch
import asyncio

from app.models.context_models import DataPoint, SourceMetadata, DataSourceType

logger = logging.getLogger(__name__)


class ElasticsearchDataSource:
    """
    Direct Elasticsearch data source for clinical context assembly.
    Provides high-performance access to clinical data without microservice overhead.
    """
    
    def __init__(self):
        # Your Elastic Cloud configuration
        self.elasticsearch_config = {
            "hosts": ["https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443"],
            "api_key": "d0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ==",
            "verify_certs": True,
            "ssl_show_warn": False,
            "timeout": 30,
            "max_retries": 3,
            "retry_on_timeout": True
        }
        
        # Index mappings for different data types
        self.index_mappings = {
            "patient_demographics": "patient-readings*",
            "patient_medications": "patient-readings*",
            "patient_conditions": "patient-readings*",
            "patient_allergies": "patient-readings*",
            "lab_results": "patient-readings*",
            "vital_signs": "patient-readings*",
            "fhir_observations": "fhir-observations*"
        }
        
        self.client = None
        self.connection_healthy = False
    
    async def initialize(self):
        """Initialize Elasticsearch client and test connection"""
        try:
            self.client = AsyncElasticsearch(**self.elasticsearch_config)
            
            # Test connection
            cluster_info = await self.client.info()
            self.connection_healthy = True
            
            logger.info("✅ Elasticsearch direct connection established")
            logger.info(f"   Cluster: {cluster_info.get('cluster_name', 'Unknown')}")
            logger.info(f"   Version: {cluster_info.get('version', {}).get('number', 'Unknown')}")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to connect to Elasticsearch: {e}")
            self.connection_healthy = False
            return False
    
    async def fetch_patient_demographics(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch patient demographics directly from Elasticsearch"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            # Build query for patient demographics
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {"exists": {"field": "patient_name"}}
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 1,
                "_source": ["patient_id", "patient_name", "reading_timestamp", "metadata"]
            }
            
            response = await self.client.search(
                index=self.index_mappings["patient_demographics"],
                body=query
            )
            
            if response["hits"]["total"]["value"] > 0:
                hit = response["hits"]["hits"][0]
                source = hit["_source"]
                
                # Extract demographics from metadata or construct from available data
                demographics = {
                    "patient_id": source.get("patient_id"),
                    "patient_name": source.get("patient_name"),
                    "age": self._extract_age_from_metadata(source.get("metadata", {})),
                    "gender": self._extract_gender_from_metadata(source.get("metadata", {})),
                    "weight": self._extract_weight_from_readings(patient_id),
                    "last_updated": source.get("reading_timestamp")
                }
                
                # Create source metadata
                source_metadata = SourceMetadata(
                    source_type=DataSourceType.CONTEXT_SERVICE_INTERNAL,
                    source_endpoint="elasticsearch_direct",
                    retrieved_at=datetime.utcnow(),
                    data_version="1.0",
                    completeness=self._calculate_completeness(demographics, data_point.fields),
                    response_time_ms=0.0,  # Would measure actual response time
                    cache_hit=False
                )
                
                return {
                    "data": demographics,
                    "metadata": source_metadata,
                    "success": True
                }
            else:
                return {
                    "data": {},
                    "metadata": None,
                    "success": False,
                    "error": f"No demographics found for patient {patient_id}"
                }
                
        except Exception as e:
            logger.error(f"❌ Error fetching patient demographics: {e}")
            return {
                "data": {},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def fetch_patient_medications(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch patient medications directly from Elasticsearch"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            # Build query for medication readings
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {"terms": {"reading_category": ["medication", "prescription", "drug"]}}
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 50,  # Get recent medications
                "_source": ["patient_id", "reading_type", "reading_value", "reading_unit", 
                           "reading_timestamp", "metadata", "reading_category"]
            }
            
            response = await self.client.search(
                index=self.index_mappings["patient_medications"],
                body=query
            )
            
            medications = []
            if response["hits"]["total"]["value"] > 0:
                for hit in response["hits"]["hits"]:
                    source = hit["_source"]
                    
                    medication = {
                        "medication_name": source.get("reading_type", "Unknown"),
                        "dosage": source.get("reading_value"),
                        "unit": source.get("reading_unit"),
                        "prescribed_date": source.get("reading_timestamp"),
                        "status": "active",  # Assume active if recent
                        "metadata": source.get("metadata", {})
                    }
                    medications.append(medication)
            
            # Create source metadata
            source_metadata = SourceMetadata(
                source_type=DataSourceType.CONTEXT_SERVICE_INTERNAL,
                source_endpoint="elasticsearch_direct",
                retrieved_at=datetime.utcnow(),
                data_version="1.0",
                completeness=1.0 if medications else 0.0,
                response_time_ms=0.0,
                cache_hit=False
            )
            
            return {
                "data": {
                    "medications": medications,
                    "total_count": len(medications)
                },
                "metadata": source_metadata,
                "success": True
            }
            
        except Exception as e:
            logger.error(f"❌ Error fetching patient medications: {e}")
            return {
                "data": {"medications": [], "total_count": 0},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def fetch_patient_vitals(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch patient vital signs directly from Elasticsearch"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            # Build query for vital signs
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {"terms": {"reading_category": ["vital", "vitals", "physiological"]}}
                        ],
                        "filter": [
                            {"range": {"reading_timestamp": {"gte": "now-24h"}}}  # Last 24 hours
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 100,
                "_source": ["reading_type", "reading_value", "reading_unit", 
                           "reading_timestamp", "alert_level", "metadata"]
            }
            
            response = await self.client.search(
                index=self.index_mappings["vital_signs"],
                body=query
            )
            
            vitals = {}
            if response["hits"]["total"]["value"] > 0:
                for hit in response["hits"]["hits"]:
                    source = hit["_source"]
                    reading_type = source.get("reading_type", "unknown")
                    
                    # Group by reading type (heart_rate, blood_pressure, etc.)
                    if reading_type not in vitals:
                        vitals[reading_type] = []
                    
                    vital_reading = {
                        "value": source.get("reading_value"),
                        "unit": source.get("reading_unit"),
                        "timestamp": source.get("reading_timestamp"),
                        "alert_level": source.get("alert_level"),
                        "metadata": source.get("metadata", {})
                    }
                    vitals[reading_type].append(vital_reading)
            
            # Create source metadata
            source_metadata = SourceMetadata(
                source_type=DataSourceType.CONTEXT_SERVICE_INTERNAL,
                source_endpoint="elasticsearch_direct",
                retrieved_at=datetime.utcnow(),
                data_version="1.0",
                completeness=1.0 if vitals else 0.0,
                response_time_ms=0.0,
                cache_hit=False
            )
            
            return {
                "data": {
                    "vital_signs": vitals,
                    "reading_count": sum(len(readings) for readings in vitals.values())
                },
                "metadata": source_metadata,
                "success": True
            }
            
        except Exception as e:
            logger.error(f"❌ Error fetching patient vitals: {e}")
            return {
                "data": {"vital_signs": {}, "reading_count": 0},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def fetch_lab_results(self, patient_id: str, data_point: DataPoint) -> Dict[str, Any]:
        """Fetch lab results directly from Elasticsearch"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            # Build query for lab results
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {"terms": {"reading_category": ["lab", "laboratory", "test"]}}
                        ],
                        "filter": [
                            {"range": {"reading_timestamp": {"gte": "now-7d"}}}  # Last 7 days
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 50,
                "_source": ["reading_type", "reading_value", "reading_unit", 
                           "reading_timestamp", "alert_level", "metadata"]
            }
            
            response = await self.client.search(
                index=self.index_mappings["lab_results"],
                body=query
            )
            
            lab_results = []
            if response["hits"]["total"]["value"] > 0:
                for hit in response["hits"]["hits"]:
                    source = hit["_source"]
                    
                    lab_result = {
                        "test_name": source.get("reading_type"),
                        "value": source.get("reading_value"),
                        "unit": source.get("reading_unit"),
                        "collected_date": source.get("reading_timestamp"),
                        "alert_level": source.get("alert_level"),
                        "reference_range": self._extract_reference_range(source.get("metadata", {})),
                        "metadata": source.get("metadata", {})
                    }
                    lab_results.append(lab_result)
            
            # Create source metadata
            source_metadata = SourceMetadata(
                source_type=DataSourceType.CONTEXT_SERVICE_INTERNAL,
                source_endpoint="elasticsearch_direct",
                retrieved_at=datetime.utcnow(),
                data_version="1.0",
                completeness=1.0 if lab_results else 0.0,
                response_time_ms=0.0,
                cache_hit=False
            )
            
            return {
                "data": {
                    "lab_results": lab_results,
                    "total_count": len(lab_results)
                },
                "metadata": source_metadata,
                "success": True
            }
            
        except Exception as e:
            logger.error(f"❌ Error fetching lab results: {e}")
            return {
                "data": {"lab_results": [], "total_count": 0},
                "metadata": None,
                "success": False,
                "error": str(e)
            }
    
    async def search_patient_data(self, patient_id: str, search_terms: List[str]) -> Dict[str, Any]:
        """Generic search across all patient data"""
        try:
            if not self.connection_healthy:
                await self.initialize()
            
            # Build multi-match query
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {
                                "multi_match": {
                                    "query": " ".join(search_terms),
                                    "fields": ["reading_type^2", "reading_category", "metadata.*"],
                                    "type": "best_fields"
                                }
                            }
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 20
            }
            
            response = await self.client.search(
                index="patient-readings*",
                body=query
            )
            
            results = []
            if response["hits"]["total"]["value"] > 0:
                for hit in response["hits"]["hits"]:
                    results.append({
                        "score": hit["_score"],
                        "data": hit["_source"],
                        "index": hit["_index"]
                    })
            
            return {
                "data": {
                    "search_results": results,
                    "total_hits": response["hits"]["total"]["value"]
                },
                "success": True
            }
            
        except Exception as e:
            logger.error(f"❌ Error searching patient data: {e}")
            return {
                "data": {"search_results": [], "total_hits": 0},
                "success": False,
                "error": str(e)
            }
    
    async def get_connection_health(self) -> Dict[str, Any]:
        """Check Elasticsearch connection health"""
        try:
            if not self.client:
                return {"healthy": False, "error": "Client not initialized"}
            
            cluster_health = await self.client.cluster.health()
            
            return {
                "healthy": True,
                "cluster_name": cluster_health.get("cluster_name"),
                "status": cluster_health.get("status"),
                "number_of_nodes": cluster_health.get("number_of_nodes"),
                "active_primary_shards": cluster_health.get("active_primary_shards"),
                "active_shards": cluster_health.get("active_shards")
            }
            
        except Exception as e:
            return {
                "healthy": False,
                "error": str(e)
            }
    
    def _extract_age_from_metadata(self, metadata: Dict[str, Any]) -> Optional[int]:
        """Extract age from metadata if available"""
        # Look for age in various metadata fields
        age_fields = ["age", "patient_age", "demographics.age"]
        for field in age_fields:
            if field in metadata:
                try:
                    return int(metadata[field])
                except (ValueError, TypeError):
                    continue
        return None
    
    def _extract_gender_from_metadata(self, metadata: Dict[str, Any]) -> Optional[str]:
        """Extract gender from metadata if available"""
        gender_fields = ["gender", "sex", "patient_gender", "demographics.gender"]
        for field in gender_fields:
            if field in metadata:
                return str(metadata[field])
        return None
    
    async def _extract_weight_from_readings(self, patient_id: str) -> Optional[float]:
        """Extract most recent weight reading for patient"""
        try:
            query = {
                "query": {
                    "bool": {
                        "must": [
                            {"term": {"patient_id": patient_id}},
                            {"terms": {"reading_type": ["weight", "body_weight", "mass"]}}
                        ]
                    }
                },
                "sort": [{"reading_timestamp": {"order": "desc"}}],
                "size": 1
            }
            
            response = await self.client.search(
                index="patient-readings*",
                body=query
            )
            
            if response["hits"]["total"]["value"] > 0:
                return float(response["hits"]["hits"][0]["_source"]["reading_value"])
            
        except Exception:
            pass
        
        return None
    
    def _extract_reference_range(self, metadata: Dict[str, Any]) -> Optional[str]:
        """Extract reference range from lab metadata"""
        range_fields = ["reference_range", "normal_range", "ref_range"]
        for field in range_fields:
            if field in metadata:
                return str(metadata[field])
        return None
    
    def _calculate_completeness(self, data: Dict[str, Any], required_fields: List[str]) -> float:
        """Calculate data completeness score"""
        if not required_fields:
            return 1.0
        
        present_fields = sum(1 for field in required_fields if data.get(field) is not None)
        return present_fields / len(required_fields)
    
    async def close(self):
        """Close Elasticsearch connection"""
        if self.client:
            await self.client.close()
            logger.info("Elasticsearch connection closed")
