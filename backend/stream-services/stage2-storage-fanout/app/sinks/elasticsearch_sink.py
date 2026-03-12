"""
Elasticsearch Sink Implementation

Handles writing UI-optimized documents to Elasticsearch with comprehensive
error handling and DLQ integration.
"""

import json
from datetime import datetime
from typing import Dict, Any, Optional

import structlog
from elasticsearch import Elasticsearch, exceptions as es_exceptions

from app.config import settings, get_elasticsearch_config

logger = structlog.get_logger(__name__)


class ElasticsearchSink:
    """
    Elasticsearch Sink for UI-optimized documents
    
    Writes device reading documents to Elasticsearch for fast search and
    real-time dashboard queries.
    """
    
    def __init__(self):
        self.sink_name = "elasticsearch"
        self.client = None
        self.index_prefix = settings.ELASTICSEARCH_INDEX_PREFIX
        self.total_writes = 0
        self.successful_writes = 0
        self.failed_writes = 0
        
        logger.info("Elasticsearch Sink initialized", 
                   index_prefix=self.index_prefix,
                   url=settings.ELASTICSEARCH_URL)
    
    async def initialize(self):
        """Initialize Elasticsearch client"""
        try:
            es_config = get_elasticsearch_config()
            self.client = Elasticsearch(**es_config)
            
            # Test connection
            if not self.client.ping():
                raise ConnectionError("Failed to connect to Elasticsearch")
            
            # Create index templates if needed
            await self._ensure_index_templates()
            
            logger.info("Elasticsearch client initialized successfully",
                       cluster_info=self.client.info())
            
        except Exception as e:
            logger.error("Failed to initialize Elasticsearch client", error=str(e))
            raise
    
    def write_ui_document(self, ui_data: str, device_id: str) -> bool:
        """
        Write UI document to Elasticsearch
        
        Args:
            ui_data: UI-optimized document JSON string
            device_id: Device ID for logging
            
        Returns:
            True if successful, False otherwise
        """
        try:
            self.total_writes += 1
            
            # Parse UI document
            ui_document = json.loads(ui_data)
            
            # Generate document ID and index name
            doc_id = self._generate_document_id(ui_document, device_id)
            index_name = self._get_index_name(ui_document)
            
            # Add indexing metadata
            ui_document.update({
                "indexed_at": datetime.utcnow().isoformat() + "Z",
                "sink_name": self.sink_name,
                "document_id": doc_id
            })
            
            # Write to Elasticsearch
            response = self.client.index(
                index=index_name,
                id=doc_id,
                body=ui_document,
                timeout=f"{settings.ELASTICSEARCH_TIMEOUT}s"
            )
            
            self.successful_writes += 1
            
            logger.debug("UI document written to Elasticsearch",
                        device_id=device_id, doc_id=doc_id,
                        index=index_name, result=response.get("result"))
            
            return True
            
        except json.JSONDecodeError as e:
            self.failed_writes += 1
            logger.error("Invalid UI document JSON", device_id=device_id, error=str(e))
            raise ValueError(f"Invalid UI document JSON: {str(e)}")
            
        except es_exceptions.ConnectionError as e:
            self.failed_writes += 1
            logger.error("Elasticsearch connection error", device_id=device_id, error=str(e))
            raise ConnectionError(f"Elasticsearch connection failed: {str(e)}")
            
        except es_exceptions.RequestError as e:
            self.failed_writes += 1
            logger.error("Elasticsearch request error", device_id=device_id, error=str(e))
            raise ValueError(f"Elasticsearch request error: {str(e)}")
            
        except es_exceptions.ElasticsearchException as e:
            self.failed_writes += 1
            logger.error("Elasticsearch error", device_id=device_id, error=str(e))
            raise
            
        except Exception as e:
            self.failed_writes += 1
            logger.error("Elasticsearch write failed", device_id=device_id, error=str(e))
            raise
    
    def write_fhir_observation(self, fhir_data: str, device_id: str) -> bool:
        """
        Elasticsearch doesn't handle FHIR observations - this is a no-op
        
        Args:
            fhir_data: FHIR Observation JSON string
            device_id: Device ID for logging
            
        Returns:
            True (no-op)
        """
        logger.debug("FHIR observation write skipped for Elasticsearch", device_id=device_id)
        return True
    
    def write_raw_data(self, raw_data: Dict[str, Any], device_id: str) -> bool:
        """
        Write raw device data to Elasticsearch (optional backup)
        
        Args:
            raw_data: Raw device reading data
            device_id: Device ID for logging
            
        Returns:
            True if successful, False otherwise
        """
        try:
            self.total_writes += 1
            
            # Generate document ID and index name for raw data
            doc_id = f"{device_id}_{raw_data.get('timestamp', int(datetime.utcnow().timestamp()))}_raw"
            index_name = f"{self.index_prefix}-raw-{datetime.utcnow().strftime('%Y-%m')}"
            
            # Add indexing metadata
            raw_document = raw_data.copy()
            raw_document.update({
                "indexed_at": datetime.utcnow().isoformat() + "Z",
                "sink_name": self.sink_name,
                "document_type": "raw_device_reading",
                "document_id": doc_id
            })
            
            # Write to Elasticsearch
            response = self.client.index(
                index=index_name,
                id=doc_id,
                body=raw_document,
                timeout=f"{settings.ELASTICSEARCH_TIMEOUT}s"
            )
            
            self.successful_writes += 1
            
            logger.debug("Raw data written to Elasticsearch",
                        device_id=device_id, doc_id=doc_id,
                        index=index_name, result=response.get("result"))
            
            return True
            
        except Exception as e:
            self.failed_writes += 1
            logger.error("Elasticsearch raw data write failed", device_id=device_id, error=str(e))
            raise
    
    def _generate_document_id(self, ui_document: Dict[str, Any], device_id: str) -> str:
        """Generate unique document ID"""
        timestamp = ui_document.get("reading_timestamp", int(datetime.utcnow().timestamp()))
        reading_type = ui_document.get("reading_type", "unknown")
        return f"{device_id}_{timestamp}_{reading_type}"
    
    def _get_index_name(self, ui_document: Dict[str, Any]) -> str:
        """Get index name based on document data"""
        # Use monthly indices for better management
        timestamp = ui_document.get("reading_timestamp")
        if timestamp:
            try:
                dt = datetime.fromtimestamp(timestamp)
                month_suffix = dt.strftime("%Y-%m")
            except (ValueError, OSError):
                month_suffix = datetime.utcnow().strftime("%Y-%m")
        else:
            month_suffix = datetime.utcnow().strftime("%Y-%m")
        
        return f"{self.index_prefix}-{month_suffix}"
    
    async def _ensure_index_templates(self):
        """Ensure index templates exist for proper mapping"""
        try:
            template_name = f"{self.index_prefix}-template"
            
            # Define index template for device readings
            template_body = {
                "index_patterns": [f"{self.index_prefix}-*"],
                "template": {
                    "settings": {
                        # Removed shard/replica settings for serverless mode compatibility
                        "refresh_interval": "5s"
                    },
                    "mappings": {
                        "properties": {
                            "device_id": {"type": "keyword"},
                            "patient_id": {"type": "keyword"},
                            "reading_timestamp": {"type": "date", "format": "epoch_second"},
                            "reading_type": {"type": "keyword"},
                            "reading_value": {"type": "double"},
                            "reading_unit": {"type": "keyword"},
                            "alert_level": {"type": "keyword"},
                            "reading_category": {"type": "keyword"},
                            "is_critical": {"type": "boolean"},
                            "indexed_at": {"type": "date"},
                            "battery_level": {"type": "integer"},
                            "signal_quality": {"type": "keyword"},
                            "vendor_id": {"type": "keyword"},
                            "vendor_name": {"type": "keyword"}
                        }
                    }
                }
            }
            
            # Create or update template (serverless mode compatible)
            try:
                self.client.indices.put_index_template(
                    name=template_name,
                    body=template_body
                )
                logger.info("Elasticsearch index template created/updated (serverless compatible)",
                           template_name=template_name)
            except Exception as template_error:
                # Serverless mode might not support templates or shard settings - that's OK
                logger.warning("Could not create index template (serverless mode)",
                             error=str(template_error))

        except Exception as e:
            logger.warning("Failed to create Elasticsearch index template", error=str(e))
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get Elasticsearch sink metrics"""
        return {
            "sink_name": self.sink_name,
            "total_writes": self.total_writes,
            "successful_writes": self.successful_writes,
            "failed_writes": self.failed_writes,
            "success_rate": self.successful_writes / max(self.total_writes, 1),
            "index_prefix": self.index_prefix
        }
    
    def is_healthy(self) -> bool:
        """Check if Elasticsearch sink is healthy"""
        try:
            if not self.client:
                return False
            
            # Simple health check - ping Elasticsearch
            return self.client.ping()
            
        except Exception as e:
            logger.warning("Elasticsearch health check failed", error=str(e))
            return False
    
    async def close(self):
        """Close Elasticsearch sink and cleanup resources"""
        if self.client:
            self.client.close()
            self.client = None
        
        logger.info("Elasticsearch sink closed",
                   total_writes=self.total_writes,
                   successful_writes=self.successful_writes,
                   failed_writes=self.failed_writes)
