"""
Real Data Validation Framework for Clinical Workflow Engine.
Ensures all clinical data comes from approved real sources only.
"""
import logging
from typing import Any, Dict, List
from datetime import datetime, timedelta
import re
import json

from app.models.clinical_activity_models import (
    DataSourceType, ClinicalDataError, MockDataDetectedError, 
    UnapprovedDataSourceError
)

logger = logging.getLogger(__name__)


class RealDataValidator:
    """
    Validates that all clinical data comes from approved real sources.
    Rejects any mock, synthetic, or fallback data.
    """
    
    # Approved data source endpoints
    APPROVED_SOURCES = {
        DataSourceType.FHIR_STORE: "projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store",
        DataSourceType.GRAPH_DB: "http://localhost:7200",
        DataSourceType.SAFETY_GATEWAY: "localhost:8028",
        DataSourceType.HARMONIZATION_SERVICE: "localhost:8015",
        DataSourceType.CAE_SERVICE: "localhost:8027",
        DataSourceType.CONTEXT_SERVICE: "localhost:8016",
        DataSourceType.MEDICATION_SERVICE: "localhost:8009",
        DataSourceType.PATIENT_SERVICE: "localhost:8003"
    }
    
    # Mock data indicators to detect and reject
    MOCK_INDICATORS = [
        'mock_', 'test_', 'fake_', 'synthetic_', 'dummy_',
        'example_', 'sample_', 'placeholder_', 'demo_',
        'stub_', 'simulated_', 'generated_', 'artificial_'
    ]
    
    # Synthetic ID patterns to detect
    SYNTHETIC_ID_PATTERNS = [
        r'^test-\d+$',
        r'^mock-[a-f0-9-]+$',
        r'^fake-\w+$',
        r'^sample-\d+$',
        r'^demo-\w+$'
    ]
    
    def __init__(self):
        self.validation_cache = {}
        self.cache_ttl_seconds = 300  # 5 minutes cache for validation results
    
    async def validate_data_source(
        self,
        source_type: DataSourceType,
        data: Any,
        metadata: Dict[str, Any]
    ) -> bool:
        """
        Validate that data comes from approved real sources only.
        Reject any mock, synthetic, or fallback data.
        
        Args:
            source_type: Type of data source
            data: The actual data to validate
            metadata: Metadata about the data source and retrieval
            
        Returns:
            bool: True if data is valid and from approved source
            
        Raises:
            ClinicalDataError: If data fails validation
        """
        try:
            # Check if data source is approved
            if not self._is_approved_source(source_type, metadata):
                raise UnapprovedDataSourceError(
                    f"Unapproved data source: {source_type.value}",
                    source_type.value
                )
            
            # Check for mock data indicators
            if self._contains_mock_indicators(data):
                raise MockDataDetectedError(
                    f"Mock data detected from {source_type.value}",
                    source_type.value
                )
            
            # Validate data freshness (no stale cached data)
            if not self._is_data_fresh(metadata):
                raise ClinicalDataError(
                    f"Stale data detected from {source_type.value}",
                    source_type.value
                )
            
            # Validate data structure integrity
            if not self._validate_data_structure(data, source_type):
                raise ClinicalDataError(
                    f"Invalid data structure from {source_type.value}",
                    source_type.value
                )
            
            # Log successful validation
            logger.info(f"Data validation successful for {source_type.value}")
            return True
            
        except ClinicalDataError:
            # Re-raise clinical data errors
            raise
        except Exception as e:
            # Convert unexpected errors to clinical data errors
            logger.error(f"Unexpected error during data validation: {e}")
            raise ClinicalDataError(
                f"Data validation failed for {source_type.value}: {str(e)}"
            )
    
    def _is_approved_source(self, source_type: DataSourceType, metadata: Dict) -> bool:
        """
        Check if data source is in approved whitelist.
        """
        expected_endpoint = self.APPROVED_SOURCES.get(source_type)
        if not expected_endpoint:
            logger.warning(f"No approved endpoint configured for {source_type.value}")
            return False
        
        actual_endpoint = metadata.get('source_endpoint', '')
        
        # For FHIR Store, check if the path contains the expected project/dataset
        if source_type == DataSourceType.FHIR_STORE:
            return expected_endpoint in actual_endpoint
        
        # For other services, check exact match or contains
        return (actual_endpoint == expected_endpoint or 
                expected_endpoint in actual_endpoint)
    
    def _contains_mock_indicators(self, data: Any) -> bool:
        """
        Detect mock data patterns in the data.
        """
        if data is None:
            return False
        
        # Convert data to string for pattern matching
        data_str = str(data).lower()
        
        # Check for mock indicators in the data string
        for indicator in self.MOCK_INDICATORS:
            if indicator in data_str:
                logger.warning(f"Mock indicator '{indicator}' found in data")
                return True
        
        # Check for synthetic ID patterns
        if isinstance(data, dict):
            for key, value in data.items():
                if isinstance(value, str):
                    for pattern in self.SYNTHETIC_ID_PATTERNS:
                        if re.match(pattern, value.lower()):
                            logger.warning(f"Synthetic ID pattern '{pattern}' found in {key}: {value}")
                            return True
        
        return False
    
    def _is_data_fresh(self, metadata: Dict) -> bool:
        """
        Check if data is fresh (not stale cached data).
        """
        retrieved_at = metadata.get('retrieved_at')
        if not retrieved_at:
            # If no timestamp, assume it's fresh (real-time data)
            return True
        
        try:
            if isinstance(retrieved_at, str):
                retrieved_time = datetime.fromisoformat(retrieved_at.replace('Z', '+00:00'))
            else:
                retrieved_time = retrieved_at
            
            # Data is considered stale if older than 1 hour for clinical data
            max_age = timedelta(hours=1)
            age = datetime.utcnow() - retrieved_time.replace(tzinfo=None)
            
            if age > max_age:
                logger.warning(f"Data is stale: {age} > {max_age}")
                return False
            
            return True
            
        except (ValueError, TypeError) as e:
            logger.error(f"Error parsing retrieved_at timestamp: {e}")
            # If we can't parse the timestamp, consider it fresh to avoid false positives
            return True
    
    def _validate_data_structure(self, data: Any, source_type: DataSourceType) -> bool:
        """
        Validate that data has expected structure for the source type.
        """
        if data is None:
            return False
        
        try:
            # Basic structure validation based on source type
            if source_type == DataSourceType.FHIR_STORE:
                return self._validate_fhir_structure(data)
            elif source_type == DataSourceType.SAFETY_GATEWAY:
                return self._validate_safety_gateway_structure(data)
            elif source_type == DataSourceType.CAE_SERVICE:
                return self._validate_cae_structure(data)
            else:
                # For other sources, just check it's not empty
                return bool(data)
                
        except Exception as e:
            logger.error(f"Error validating data structure: {e}")
            return False
    
    def _validate_fhir_structure(self, data: Any) -> bool:
        """
        Validate FHIR resource structure.
        """
        if isinstance(data, dict):
            # FHIR resources should have resourceType
            return 'resourceType' in data
        elif isinstance(data, list):
            # FHIR bundles or arrays of resources
            return len(data) > 0
        return False
    
    def _validate_safety_gateway_structure(self, data: Any) -> bool:
        """
        Validate Safety Gateway response structure.
        """
        if isinstance(data, dict):
            # Safety Gateway responses should have status and results
            return 'status' in data or 'result' in data or 'validation_result' in data
        return False
    
    def _validate_cae_structure(self, data: Any) -> bool:
        """
        Validate CAE service response structure.
        """
        if isinstance(data, dict):
            # CAE responses should have clinical reasoning results
            return ('reasoning_result' in data or 
                    'clinical_decision' in data or 
                    'safety_assessment' in data)
        return False
    
    async def validate_batch_data(
        self,
        data_batch: List[Dict[str, Any]]
    ) -> Dict[str, bool]:
        """
        Validate a batch of data from multiple sources.
        
        Args:
            data_batch: List of data items with source info
            
        Returns:
            Dict mapping data item IDs to validation results
        """
        results = {}
        
        for item in data_batch:
            item_id = item.get('id', 'unknown')
            source_type = item.get('source_type')
            data = item.get('data')
            metadata = item.get('metadata', {})
            
            try:
                if source_type:
                    source_enum = DataSourceType(source_type)
                    results[item_id] = await self.validate_data_source(
                        source_enum, data, metadata
                    )
                else:
                    results[item_id] = False
                    
            except Exception as e:
                logger.error(f"Batch validation failed for item {item_id}: {e}")
                results[item_id] = False
        
        return results


# Global validator instance
real_data_validator = RealDataValidator()
