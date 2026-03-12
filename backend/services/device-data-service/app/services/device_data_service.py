"""
Device Data Service

Service for querying processed device data from Elasticsearch and FHIR Store.
This service provides the read-side of the event-driven architecture.
"""
import logging
from typing import List, Optional, Dict, Any
from datetime import datetime
import json

from ..config import settings

logger = logging.getLogger(__name__)


class DeviceDataService:
    """
    Service for querying device data from Elasticsearch and FHIR Store.
    
    This service provides the read-side queries for device data that has been
    processed by the ETL pipeline and stored in optimized formats.
    """
    
    def __init__(self):
        """Initialize the device data service"""
        self.elasticsearch_client = None
        self.fhir_client = None
        self._initialize_clients()
    
    def _initialize_clients(self):
        """Initialize Elasticsearch and FHIR clients"""
        try:
            # Initialize Elasticsearch client
            from elasticsearch import AsyncElasticsearch
            self.elasticsearch_client = AsyncElasticsearch(
                [settings.ELASTICSEARCH_URL],
                verify_certs=False,  # For development
                ssl_show_warn=False
            )
            logger.info("Elasticsearch client initialized")
            
        except ImportError:
            logger.warning("Elasticsearch not available, using mock data")
        except Exception as e:
            logger.error(f"Failed to initialize Elasticsearch client: {e}")
        
        try:
            # Initialize FHIR client (Google Healthcare API)
            # TODO: Implement Google Healthcare API client
            logger.info("FHIR client would be initialized here")
            
        except Exception as e:
            logger.error(f"Failed to initialize FHIR client: {e}")
    
    async def get_reading_by_id(self, reading_id: str) -> Optional[Dict[str, Any]]:
        """Get a specific reading by ID"""
        try:
            if self.elasticsearch_client:
                response = await self.elasticsearch_client.get(
                    index=settings.ELASTICSEARCH_INDEX,
                    id=reading_id
                )
                return self._transform_es_document(response["_source"])
            else:
                # Mock data for testing
                return self._get_mock_reading(reading_id)
                
        except Exception as e:
            logger.error(f"Error getting reading {reading_id}: {e}")
            return None
    
    async def get_patient_readings(
        self,
        patient_id: str,
        reading_type: Optional[str] = None,
        alert_level: Optional[str] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> Dict[str, Any]:
        """Get readings for a specific patient"""
        try:
            if self.elasticsearch_client:
                query = self._build_patient_query(
                    patient_id, reading_type, alert_level, start_date, end_date
                )
                
                response = await self.elasticsearch_client.search(
                    index=settings.ELASTICSEARCH_INDEX,
                    body=query,
                    from_=(page - 1) * limit,
                    size=limit
                )
                
                return self._transform_search_response(response, page, limit)
            else:
                # Mock data for testing
                return self._get_mock_patient_readings(patient_id, page, limit)
                
        except Exception as e:
            logger.error(f"Error getting patient readings for {patient_id}: {e}")
            return {"items": [], "total": 0, "page": page, "limit": limit, "has_next_page": False, "has_previous_page": False}
    
    async def get_device_readings(
        self,
        device_id: str,
        reading_type: Optional[str] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> Dict[str, Any]:
        """Get readings for a specific device"""
        try:
            if self.elasticsearch_client:
                query = self._build_device_query(
                    device_id, reading_type, start_date, end_date
                )
                
                response = await self.elasticsearch_client.search(
                    index=settings.ELASTICSEARCH_INDEX,
                    body=query,
                    from_=(page - 1) * limit,
                    size=limit
                )
                
                return self._transform_search_response(response, page, limit)
            else:
                # Mock data for testing
                return self._get_mock_device_readings(device_id, page, limit)
                
        except Exception as e:
            logger.error(f"Error getting device readings for {device_id}: {e}")
            return {"items": [], "total": 0, "page": page, "limit": limit, "has_next_page": False, "has_previous_page": False}
    
    async def search_readings(
        self,
        reading_type: Optional[str] = None,
        alert_level: Optional[str] = None,
        reading_category: Optional[str] = None,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        page: int = 1,
        limit: int = 50
    ) -> Dict[str, Any]:
        """Search readings with filters"""
        try:
            if self.elasticsearch_client:
                query = self._build_search_query(
                    reading_type, alert_level, reading_category, start_date, end_date
                )
                
                response = await self.elasticsearch_client.search(
                    index=settings.ELASTICSEARCH_INDEX,
                    body=query,
                    from_=(page - 1) * limit,
                    size=limit
                )
                
                return self._transform_search_response(response, page, limit)
            else:
                # Mock data for testing
                return self._get_mock_search_readings(page, limit)
                
        except Exception as e:
            logger.error(f"Error searching readings: {e}")
            return {"items": [], "total": 0, "page": page, "limit": limit, "has_next_page": False, "has_previous_page": False}
    
    async def get_patient_reading_stats(
        self,
        patient_id: str,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get reading statistics for a patient"""
        try:
            if self.elasticsearch_client:
                query = self._build_stats_query(patient_id, start_date, end_date)
                
                response = await self.elasticsearch_client.search(
                    index=settings.ELASTICSEARCH_INDEX,
                    body=query
                )
                
                return self._transform_stats_response(response)
            else:
                # Mock data for testing
                return self._get_mock_stats()
                
        except Exception as e:
            logger.error(f"Error getting patient stats for {patient_id}: {e}")
            return self._get_mock_stats()
    
    async def get_device_reading_stats(
        self,
        device_id: str,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get reading statistics for a device"""
        # Similar implementation to patient stats but filtered by device_id
        return await self.get_patient_reading_stats("device-" + device_id, start_date, end_date)
    
    async def get_global_reading_stats(
        self,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get global reading statistics"""
        # Similar implementation but without patient/device filter
        return await self.get_patient_reading_stats("global", start_date, end_date)
    
    async def get_patient_devices(self, patient_id: str) -> List[str]:
        """Get all device IDs associated with a patient"""
        try:
            if self.elasticsearch_client:
                query = {
                    "query": {
                        "term": {"patient_id": patient_id}
                    },
                    "aggs": {
                        "devices": {
                            "terms": {
                                "field": "device_id",
                                "size": 100
                            }
                        }
                    },
                    "size": 0
                }
                
                response = await self.elasticsearch_client.search(
                    index=settings.ELASTICSEARCH_INDEX,
                    body=query
                )
                
                buckets = response["aggregations"]["devices"]["buckets"]
                return [bucket["key"] for bucket in buckets]
            else:
                # Mock data
                return [f"device-{i}" for i in range(1, 4)]
                
        except Exception as e:
            logger.error(f"Error getting patient devices for {patient_id}: {e}")
            return []
    
    def _build_patient_query(self, patient_id: str, reading_type: Optional[str], 
                           alert_level: Optional[str], start_date: Optional[str], 
                           end_date: Optional[str]) -> Dict[str, Any]:
        """Build Elasticsearch query for patient readings"""
        must_clauses = [{"term": {"patient_id": patient_id}}]
        
        if reading_type:
            must_clauses.append({"term": {"reading_type": reading_type}})
        
        if alert_level:
            must_clauses.append({"term": {"alert_level": alert_level}})
        
        if start_date or end_date:
            range_clause = {"range": {"reading_datetime": {}}}
            if start_date:
                range_clause["range"]["reading_datetime"]["gte"] = start_date
            if end_date:
                range_clause["range"]["reading_datetime"]["lte"] = end_date
            must_clauses.append(range_clause)
        
        return {
            "query": {
                "bool": {
                    "must": must_clauses
                }
            },
            "sort": [{"reading_timestamp": {"order": "desc"}}]
        }
    
    def _build_device_query(self, device_id: str, reading_type: Optional[str],
                          start_date: Optional[str], end_date: Optional[str]) -> Dict[str, Any]:
        """Build Elasticsearch query for device readings"""
        must_clauses = [{"term": {"device_id": device_id}}]
        
        if reading_type:
            must_clauses.append({"term": {"reading_type": reading_type}})
        
        if start_date or end_date:
            range_clause = {"range": {"reading_datetime": {}}}
            if start_date:
                range_clause["range"]["reading_datetime"]["gte"] = start_date
            if end_date:
                range_clause["range"]["reading_datetime"]["lte"] = end_date
            must_clauses.append(range_clause)
        
        return {
            "query": {
                "bool": {
                    "must": must_clauses
                }
            },
            "sort": [{"reading_timestamp": {"order": "desc"}}]
        }
    
    def _build_search_query(self, reading_type: Optional[str], alert_level: Optional[str],
                          reading_category: Optional[str], start_date: Optional[str], 
                          end_date: Optional[str]) -> Dict[str, Any]:
        """Build Elasticsearch query for general search"""
        must_clauses = []
        
        if reading_type:
            must_clauses.append({"term": {"reading_type": reading_type}})
        
        if alert_level:
            must_clauses.append({"term": {"alert_level": alert_level}})
        
        if reading_category:
            must_clauses.append({"term": {"reading_category": reading_category}})
        
        if start_date or end_date:
            range_clause = {"range": {"reading_datetime": {}}}
            if start_date:
                range_clause["range"]["reading_datetime"]["gte"] = start_date
            if end_date:
                range_clause["range"]["reading_datetime"]["lte"] = end_date
            must_clauses.append(range_clause)
        
        query = {
            "sort": [{"reading_timestamp": {"order": "desc"}}]
        }
        
        if must_clauses:
            query["query"] = {
                "bool": {
                    "must": must_clauses
                }
            }
        else:
            query["query"] = {"match_all": {}}
        
        return query

    def _build_stats_query(self, patient_id: str, start_date: Optional[str],
                          end_date: Optional[str]) -> Dict[str, Any]:
        """Build Elasticsearch query for statistics"""
        must_clauses = []

        if patient_id != "global":
            must_clauses.append({"term": {"patient_id": patient_id}})

        if start_date or end_date:
            range_clause = {"range": {"reading_datetime": {}}}
            if start_date:
                range_clause["range"]["reading_datetime"]["gte"] = start_date
            if end_date:
                range_clause["range"]["reading_datetime"]["lte"] = end_date
            must_clauses.append(range_clause)

        query = {
            "size": 0,
            "aggs": {
                "by_type": {
                    "terms": {"field": "reading_type"}
                },
                "by_alert_level": {
                    "terms": {"field": "alert_level"}
                },
                "latest": {
                    "top_hits": {
                        "sort": [{"reading_timestamp": {"order": "desc"}}],
                        "size": 1
                    }
                },
                "date_range": {
                    "stats": {"field": "reading_timestamp"}
                }
            }
        }

        if must_clauses:
            query["query"] = {
                "bool": {
                    "must": must_clauses
                }
            }
        else:
            query["query"] = {"match_all": {}}

        return query

    def _transform_search_response(self, response: Dict[str, Any], page: int, limit: int) -> Dict[str, Any]:
        """Transform Elasticsearch search response"""
        hits = response["hits"]["hits"]
        total = response["hits"]["total"]["value"] if isinstance(response["hits"]["total"], dict) else response["hits"]["total"]

        items = [self._transform_es_document(hit["_source"]) for hit in hits]

        return {
            "items": items,
            "total": total,
            "page": page,
            "limit": limit,
            "has_next_page": (page * limit) < total,
            "has_previous_page": page > 1
        }

    def _transform_stats_response(self, response: Dict[str, Any]) -> Dict[str, Any]:
        """Transform Elasticsearch stats response"""
        aggs = response["aggregations"]

        readings_by_type = [
            {"reading_type": bucket["key"], "count": bucket["doc_count"]}
            for bucket in aggs["by_type"]["buckets"]
        ]

        readings_by_alert_level = [
            {"alert_level": bucket["key"], "count": bucket["doc_count"]}
            for bucket in aggs["by_alert_level"]["buckets"]
        ]

        latest_reading = None
        if aggs["latest"]["hits"]["hits"]:
            latest_reading = self._transform_es_document(
                aggs["latest"]["hits"]["hits"][0]["_source"]
            )

        date_stats = aggs["date_range"]
        date_range = None
        if date_stats["min"] and date_stats["max"]:
            date_range = {
                "start_date": datetime.fromtimestamp(date_stats["min"] / 1000).isoformat(),
                "end_date": datetime.fromtimestamp(date_stats["max"] / 1000).isoformat()
            }

        return {
            "total_readings": response["hits"]["total"]["value"] if isinstance(response["hits"]["total"], dict) else response["hits"]["total"],
            "readings_by_type": readings_by_type,
            "readings_by_alert_level": readings_by_alert_level,
            "latest_reading": latest_reading,
            "date_range": date_range
        }

    def _transform_es_document(self, doc: Dict[str, Any]) -> Dict[str, Any]:
        """Transform Elasticsearch document to GraphQL format"""
        return {
            "id": doc.get("id", "unknown"),
            "device_id": doc.get("device_id", "unknown"),
            "patient_id": doc.get("patient_id"),
            "reading_timestamp": doc.get("reading_timestamp", 0),
            "reading_datetime": doc.get("reading_datetime", ""),
            "reading_date": doc.get("reading_date", ""),
            "reading_time": doc.get("reading_time", ""),
            "reading_type": doc.get("reading_type", "unknown"),
            "reading_type_display": doc.get("reading_type_display", ""),
            "reading_value": doc.get("reading_value", 0.0),
            "reading_unit": doc.get("reading_unit", ""),
            "reading_category": doc.get("reading_category", "other"),
            "alert_level": doc.get("alert_level", "normal"),
            "is_critical": doc.get("is_critical", False),
            "is_abnormal": doc.get("is_abnormal", False),
            "vendor_info": doc.get("vendor_info"),
            "device_metadata": doc.get("device_metadata"),
            "indexed_at": doc.get("indexed_at", ""),
            "year": doc.get("year", 0),
            "month": doc.get("month", 0),
            "day": doc.get("day", 0),
            "hour": doc.get("hour", 0)
        }

    # Mock data methods for testing without Elasticsearch
    def _get_mock_reading(self, reading_id: str) -> Dict[str, Any]:
        """Get mock reading data"""
        return {
            "id": reading_id,
            "device_id": "mock-device-001",
            "patient_id": "patient-12345",
            "reading_timestamp": int(datetime.now().timestamp()),
            "reading_datetime": datetime.now().isoformat(),
            "reading_date": datetime.now().strftime("%Y-%m-%d"),
            "reading_time": datetime.now().strftime("%H:%M:%S"),
            "reading_type": "heart_rate",
            "reading_type_display": "Heart Rate",
            "reading_value": 75.0,
            "reading_unit": "bpm",
            "reading_category": "cardiovascular",
            "alert_level": "normal",
            "is_critical": False,
            "is_abnormal": False,
            "vendor_info": {"vendor_id": "test-vendor", "vendor_name": "Test Vendor"},
            "device_metadata": {"battery_level": 85, "signal_quality": "good"},
            "indexed_at": datetime.now().isoformat(),
            "year": datetime.now().year,
            "month": datetime.now().month,
            "day": datetime.now().day,
            "hour": datetime.now().hour
        }

    def _get_mock_patient_readings(self, patient_id: str, page: int, limit: int) -> Dict[str, Any]:
        """Get mock patient readings"""
        mock_readings = [self._get_mock_reading(f"reading-{i}") for i in range(1, 11)]
        for i, reading in enumerate(mock_readings):
            reading["patient_id"] = patient_id
            reading["reading_value"] = 70.0 + i * 2

        start_idx = (page - 1) * limit
        end_idx = start_idx + limit
        items = mock_readings[start_idx:end_idx]

        return {
            "items": items,
            "total": len(mock_readings),
            "page": page,
            "limit": limit,
            "has_next_page": end_idx < len(mock_readings),
            "has_previous_page": page > 1
        }

    def _get_mock_device_readings(self, device_id: str, page: int, limit: int) -> Dict[str, Any]:
        """Get mock device readings"""
        return self._get_mock_patient_readings(f"device-{device_id}", page, limit)

    def _get_mock_search_readings(self, page: int, limit: int) -> Dict[str, Any]:
        """Get mock search readings"""
        return self._get_mock_patient_readings("global", page, limit)

    def _get_mock_stats(self) -> Dict[str, Any]:
        """Get mock statistics"""
        return {
            "total_readings": 100,
            "readings_by_type": [
                {"reading_type": "heart_rate", "count": 40},
                {"reading_type": "blood_pressure_systolic", "count": 30},
                {"reading_type": "blood_glucose", "count": 30}
            ],
            "readings_by_alert_level": [
                {"alert_level": "normal", "count": 80},
                {"alert_level": "high", "count": 15},
                {"alert_level": "critical", "count": 5}
            ],
            "latest_reading": self._get_mock_reading("latest"),
            "date_range": {
                "start_date": "2023-12-01T00:00:00",
                "end_date": "2023-12-21T23:59:59"
            }
        }


# Global service instance
_device_data_service: Optional[DeviceDataService] = None


def get_device_data_service() -> DeviceDataService:
    """Get or create global device data service instance"""
    global _device_data_service

    if _device_data_service is None:
        _device_data_service = DeviceDataService()

    return _device_data_service
