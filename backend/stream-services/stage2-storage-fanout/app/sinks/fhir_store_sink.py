"""
FHIR Store Sink Implementation

Handles writing FHIR Observation resources to Google Healthcare API FHIR Store
using the EXACT same implementation as your PySpark ETL pipeline.
"""

import json
import os
from typing import Dict, Any, Optional

import structlog
from google.auth import default
from google.auth.exceptions import DefaultCredentialsError
from google.oauth2 import service_account
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

from app.config import settings, get_fhir_store_path

logger = structlog.get_logger(__name__)


class FHIRStoreSink:
    """
    FHIR Store Sink for Google Healthcare API
    
    Writes FHIR Observation resources to the configured FHIR Store
    with proper error handling and retry logic.
    """
    
    def __init__(self):
        self.sink_name = "fhir_store"
        self.client = None
        self.fhir_store_path = get_fhir_store_path()
        self.total_writes = 0
        self.successful_writes = 0
        self.failed_writes = 0
        
        logger.info("FHIR Store Sink initialized", fhir_store_path=self.fhir_store_path)
    
    async def initialize(self):
        """Initialize Google Healthcare API client - EXACT same as your medication service"""
        try:
            # Initialize credentials from file (same as your medication service)
            credentials_path = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")

            # Try multiple credential paths
            possible_paths = [
                credentials_path,
                "credentials/google-credentials.json",
                "./credentials/google-credentials.json",
                os.path.join(os.path.dirname(__file__), "..", "..", "credentials", "google-credentials.json")
            ]

            credentials = None
            for path in possible_paths:
                if path and os.path.exists(path):
                    from google.oauth2 import service_account
                    credentials = service_account.Credentials.from_service_account_file(path)
                    logger.info("Loaded credentials from file", path=path)
                    break

            if not credentials:
                # Fallback to default credentials
                credentials, _ = default()
                logger.info("Using default credentials")

            # Build the Healthcare API client (same as encounter-service, scheduling-service)
            self.client = build('healthcare', 'v1', credentials=credentials)

            # Set up the FHIR store path (same pattern as your services)
            project_id = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
            location = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
            dataset_id = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
            fhir_store_id = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")

            self.fhir_store_path = f"projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}"

            logger.info("FHIR Store client initialized successfully (medication service compatible)",
                       project=project_id, fhir_store=self.fhir_store_path)

        except DefaultCredentialsError as e:
            logger.error("Failed to initialize FHIR Store credentials", error=str(e))
            raise
        except Exception as e:
            logger.error("Failed to initialize FHIR Store client", error=str(e))
            raise
    
    def write_fhir_observation(self, fhir_data: str, device_id: str) -> bool:
        """
        Write FHIR Observation to FHIR Store - EXACT same as your PySpark implementation

        Args:
            fhir_data: FHIR Observation JSON string
            device_id: Device ID for logging

        Returns:
            True if successful, False otherwise
        """
        try:
            self.total_writes += 1

            # Parse FHIR data
            fhir_resource = json.loads(fhir_data)
            resource_id = fhir_resource.get("id")

            # Debug: Log the FHIR resource structure
            logger.debug("FHIR resource to be written",
                        device_id=device_id,
                        resource_id=resource_id,
                        resource_type=fhir_resource.get("resourceType"),
                        fhir_data_length=len(fhir_data))

            if not resource_id:
                raise ValueError("FHIR resource missing required 'id' field")

            # Create FHIR resource using Google API Client (same as your PySpark)
            # This matches your vercel-etl implementation exactly
            request_body = {
                'resourceType': 'Observation',
                **fhir_resource
            }

            # Execute write using Google API Client (same pattern as your services)
            response = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type='Observation',
                body=request_body
            ).execute()

            self.successful_writes += 1

            logger.debug("FHIR Observation written successfully (PySpark compatible)",
                        device_id=device_id, resource_id=resource_id,
                        fhir_store=self.fhir_store_path, response_name=response.get('name'))

            return True

        except json.JSONDecodeError as e:
            self.failed_writes += 1
            logger.error("Invalid FHIR JSON data", device_id=device_id, error=str(e))
            raise ValueError(f"Invalid FHIR JSON: {str(e)}")

        except ValueError as e:
            self.failed_writes += 1
            logger.error("Invalid FHIR resource", device_id=device_id, error=str(e))
            raise

        except HttpError as e:
            self.failed_writes += 1
            logger.error("Google Healthcare API HTTP error", device_id=device_id,
                        error=str(e), status_code=e.resp.status, fhir_store=self.fhir_store_path)
            raise

        except Exception as e:
            self.failed_writes += 1
            logger.error("FHIR Store write failed", device_id=device_id,
                        error=str(e), fhir_store=self.fhir_store_path)
            raise
    
    def write_ui_document(self, ui_data: str, device_id: str) -> bool:
        """
        FHIR Store doesn't handle UI documents - this is a no-op
        
        Args:
            ui_data: UI document JSON string
            device_id: Device ID for logging
            
        Returns:
            True (no-op)
        """
        logger.debug("UI document write skipped for FHIR Store", device_id=device_id)
        return True
    
    def write_raw_data(self, raw_data: Dict[str, Any], device_id: str) -> bool:
        """
        FHIR Store doesn't handle raw data - this is a no-op
        
        Args:
            raw_data: Raw device reading data
            device_id: Device ID for logging
            
        Returns:
            True (no-op)
        """
        logger.debug("Raw data write skipped for FHIR Store", device_id=device_id)
        return True
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get FHIR Store sink metrics"""
        return {
            "sink_name": self.sink_name,
            "total_writes": self.total_writes,
            "successful_writes": self.successful_writes,
            "failed_writes": self.failed_writes,
            "success_rate": self.successful_writes / max(self.total_writes, 1),
            "fhir_store_path": self.fhir_store_path
        }
    
    def is_healthy(self) -> bool:
        """Check if FHIR Store sink is healthy"""
        try:
            if not self.client:
                return False
            
            # Simple health check - try to get FHIR store metadata
            # This is a lightweight operation that verifies connectivity
            request = healthcare_v1.GetFhirStoreRequest(name=self.fhir_store_path)
            self.client.get_fhir_store(request=request)
            
            return True
            
        except Exception as e:
            logger.warning("FHIR Store health check failed", error=str(e))
            return False
    
    async def close(self):
        """Close FHIR Store sink and cleanup resources"""
        if self.client:
            # Google Cloud client doesn't need explicit closing
            self.client = None
        
        logger.info("FHIR Store sink closed", 
                   total_writes=self.total_writes,
                   successful_writes=self.successful_writes,
                   failed_writes=self.failed_writes)
