"""
FHIR Store Handler for Google Cloud Healthcare API
Handles FHIR resource upserts with retry logic and validation
"""

import json
import time
from typing import Dict, Any, Optional
import structlog

# Try to import real Google Cloud Healthcare API, fall back to mock
try:
    from google.cloud import healthcare_v1
    from google.api_core import exceptions as google_exceptions
    from google.oauth2 import service_account
    USE_MOCK = False
except ImportError:
    print("⚠️  Google Cloud Healthcare API not available - using MOCK")
    from app import mock_healthcare_v1 as healthcare_v1
    from app import mock_exceptions as google_exceptions
    USE_MOCK = True
    # Mock service account module
    class service_account:
        class Credentials:
            @staticmethod
            def from_service_account_file(path, scopes=None):
                return None

logger = structlog.get_logger(__name__)


class FHIRStoreHandler:
    """
    Handler for Google Cloud Healthcare FHIR Store operations

    Supports upsert operations (CREATE or UPDATE) for FHIR R4 resources
    with retry logic, validation, and comprehensive error handling.
    """

    # Supported FHIR resource types (from implementation plan)
    SUPPORTED_RESOURCE_TYPES = {
        'Observation',
        'RiskAssessment',
        'DiagnosticReport',
        'Condition',
        'MedicationRequest',
        'Procedure',
        'Encounter',
        'Patient',  # Core resource
    }

    def __init__(
        self,
        project_id: str,
        location: str,
        dataset_id: str,
        store_id: str,
        credentials_path: str,
        max_retries: int = 3,
        retry_backoff_factor: float = 2.0,
    ):
        """
        Initialize FHIR Store handler

        Args:
            project_id: GCP project ID
            location: GCP location (e.g., us-central1)
            dataset_id: Healthcare dataset ID
            store_id: FHIR store ID
            credentials_path: Path to service account credentials JSON
            max_retries: Maximum retry attempts for failed requests
            retry_backoff_factor: Exponential backoff multiplier
        """
        self.project_id = project_id
        self.location = location
        self.dataset_id = dataset_id
        self.store_id = store_id
        self.max_retries = max_retries
        self.retry_backoff_factor = retry_backoff_factor

        # Build FHIR store path
        self.fhir_store_path = (
            f"projects/{project_id}/locations/{location}/"
            f"datasets/{dataset_id}/fhirStores/{store_id}"
        )

        # Initialize Google Healthcare API client
        credentials = service_account.Credentials.from_service_account_file(
            credentials_path,
            scopes=['https://www.googleapis.com/auth/cloud-healthcare']
        )

        self.client = healthcare_v1.FhirServiceClient(credentials=credentials)

        # Track statistics
        self.stats = {
            'total_upserts': 0,
            'successful_creates': 0,
            'successful_updates': 0,
            'failed_upserts': 0,
            'validation_errors': 0,
            'api_errors': 0,
            'resource_type_counts': {},
        }

        logger.info(
            "FHIR Store handler initialized",
            fhir_store_path=self.fhir_store_path,
            supported_types=list(self.SUPPORTED_RESOURCE_TYPES),
        )

    def upsert_resource(self, fhir_resource_obj: Dict[str, Any]) -> Dict[str, Any]:
        """
        Upsert FHIR resource to Google Cloud Healthcare API

        Strategy:
        1. Validate resource structure
        2. Try UPDATE first (resource may already exist)
        3. If 404 (not found), CREATE new resource
        4. Retry on transient errors with exponential backoff

        Args:
            fhir_resource_obj: Dict with keys:
                - resourceType: FHIR resource type
                - resourceId: Resource identifier
                - patientId: Patient reference
                - fhirData: Complete FHIR R4 resource dict

        Returns:
            Dict with operation result:
                - success: bool
                - operation: 'CREATE' or 'UPDATE'
                - resource_type: FHIR resource type
                - resource_id: Resource identifier
                - error: Optional error message

        Raises:
            ValueError: If resource validation fails
        """
        self.stats['total_upserts'] += 1

        # Extract fields
        resource_type = fhir_resource_obj.get('resourceType')
        resource_id = fhir_resource_obj.get('resourceId')
        fhir_data = fhir_resource_obj.get('fhirData', {})

        # Validate resource
        self._validate_resource(resource_type, resource_id, fhir_data)

        # Build resource path
        resource_path = f"{self.fhir_store_path}/fhir/{resource_type}/{resource_id}"

        # Track resource type
        self.stats['resource_type_counts'][resource_type] = \
            self.stats['resource_type_counts'].get(resource_type, 0) + 1

        # Attempt upsert with retry logic
        for attempt in range(self.max_retries):
            try:
                # Try UPDATE first (most common case)
                result = self._update_resource(resource_path, fhir_data)

                if result['success']:
                    self.stats['successful_updates'] += 1
                    logger.debug(
                        "FHIR resource updated",
                        resource_type=resource_type,
                        resource_id=resource_id,
                        attempt=attempt + 1,
                    )
                    return result

            except google_exceptions.NotFound:
                # Resource doesn't exist, CREATE it
                try:
                    result = self._create_resource(
                        resource_type,
                        resource_id,
                        fhir_data
                    )

                    if result['success']:
                        self.stats['successful_creates'] += 1
                        logger.debug(
                            "FHIR resource created",
                            resource_type=resource_type,
                            resource_id=resource_id,
                            attempt=attempt + 1,
                        )
                        return result

                except Exception as create_error:
                    logger.error(
                        "CREATE failed after UPDATE 404",
                        resource_type=resource_type,
                        resource_id=resource_id,
                        error=str(create_error),
                        attempt=attempt + 1,
                    )

                    # Retry on transient errors
                    if self._is_retryable_error(create_error) and attempt < self.max_retries - 1:
                        self._wait_with_backoff(attempt)
                        continue
                    else:
                        self.stats['failed_upserts'] += 1
                        self.stats['api_errors'] += 1
                        return {
                            'success': False,
                            'operation': 'CREATE',
                            'resource_type': resource_type,
                            'resource_id': resource_id,
                            'error': str(create_error),
                        }

            except Exception as update_error:
                logger.error(
                    "UPDATE failed",
                    resource_type=resource_type,
                    resource_id=resource_id,
                    error=str(update_error),
                    attempt=attempt + 1,
                )

                # Retry on transient errors
                if self._is_retryable_error(update_error) and attempt < self.max_retries - 1:
                    self._wait_with_backoff(attempt)
                    continue
                else:
                    self.stats['failed_upserts'] += 1
                    self.stats['api_errors'] += 1
                    return {
                        'success': False,
                        'operation': 'UPDATE',
                        'resource_type': resource_type,
                        'resource_id': resource_id,
                        'error': str(update_error),
                    }

        # Max retries exceeded
        self.stats['failed_upserts'] += 1
        return {
            'success': False,
            'operation': 'RETRY_EXHAUSTED',
            'resource_type': resource_type,
            'resource_id': resource_id,
            'error': f'Max retries ({self.max_retries}) exceeded',
        }

    def _validate_resource(
        self,
        resource_type: str,
        resource_id: str,
        fhir_data: Dict[str, Any]
    ) -> None:
        """
        Validate FHIR resource structure

        Args:
            resource_type: FHIR resource type
            resource_id: Resource identifier
            fhir_data: Complete FHIR resource

        Raises:
            ValueError: If validation fails
        """
        # Check resource type support
        if resource_type not in self.SUPPORTED_RESOURCE_TYPES:
            self.stats['validation_errors'] += 1
            raise ValueError(
                f"Unsupported resource type: {resource_type}. "
                f"Supported: {self.SUPPORTED_RESOURCE_TYPES}"
            )

        # Check resource ID
        if not resource_id or not isinstance(resource_id, str):
            self.stats['validation_errors'] += 1
            raise ValueError(f"Invalid resource_id: {resource_id}")

        # Check FHIR data exists
        if not fhir_data or not isinstance(fhir_data, dict):
            self.stats['validation_errors'] += 1
            raise ValueError("fhir_data must be non-empty dict")

        # Validate FHIR data has resourceType
        if fhir_data.get('resourceType') != resource_type:
            self.stats['validation_errors'] += 1
            raise ValueError(
                f"resourceType mismatch: wrapper={resource_type}, "
                f"fhirData={fhir_data.get('resourceType')}"
            )

        # Validate FHIR data has id
        if fhir_data.get('id') != resource_id:
            self.stats['validation_errors'] += 1
            raise ValueError(
                f"id mismatch: wrapper={resource_id}, "
                f"fhirData={fhir_data.get('id')}"
            )

    def _update_resource(
        self,
        resource_path: str,
        fhir_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Update existing FHIR resource

        Args:
            resource_path: Full resource path
            fhir_data: Complete FHIR resource

        Returns:
            Dict with success=True and operation='UPDATE'

        Raises:
            google_exceptions.NotFound: If resource doesn't exist
            Exception: On API errors
        """
        request = healthcare_v1.UpdateResourceRequest(
            name=resource_path,
            body=json.dumps(fhir_data).encode('utf-8'),
        )

        response = self.client.update_resource(request=request)

        return {
            'success': True,
            'operation': 'UPDATE',
            'resource_type': fhir_data.get('resourceType'),
            'resource_id': fhir_data.get('id'),
            'response_data': response.data.decode('utf-8'),
        }

    def _create_resource(
        self,
        resource_type: str,
        resource_id: str,
        fhir_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Create new FHIR resource

        Args:
            resource_type: FHIR resource type
            resource_id: Resource identifier
            fhir_data: Complete FHIR resource

        Returns:
            Dict with success=True and operation='CREATE'

        Raises:
            Exception: On API errors
        """
        # Build parent path (FHIR store path + resource type)
        parent = f"{self.fhir_store_path}/fhir/{resource_type}"

        request = healthcare_v1.CreateResourceRequest(
            parent=parent,
            type_=resource_type,
            body=json.dumps(fhir_data).encode('utf-8'),
        )

        response = self.client.create_resource(request=request)

        return {
            'success': True,
            'operation': 'CREATE',
            'resource_type': resource_type,
            'resource_id': resource_id,
            'response_data': response.data.decode('utf-8'),
        }

    def _is_retryable_error(self, error: Exception) -> bool:
        """Check if error is retryable (transient)"""
        retryable_exceptions = (
            google_exceptions.ServiceUnavailable,
            google_exceptions.DeadlineExceeded,
            google_exceptions.ResourceExhausted,
            google_exceptions.InternalServerError,
        )
        return isinstance(error, retryable_exceptions)

    def _wait_with_backoff(self, attempt: int) -> None:
        """Wait with exponential backoff"""
        wait_time = self.retry_backoff_factor ** attempt
        logger.debug(f"Retrying after {wait_time}s", attempt=attempt + 1)
        time.sleep(wait_time)

    def get_stats(self) -> Dict[str, Any]:
        """Get handler statistics"""
        return {
            **self.stats,
            'success_rate': (
                (self.stats['successful_creates'] + self.stats['successful_updates']) /
                max(self.stats['total_upserts'], 1)
            ),
        }

    def reset_stats(self) -> None:
        """Reset statistics counters"""
        for key in self.stats:
            if isinstance(self.stats[key], dict):
                self.stats[key] = {}
            else:
                self.stats[key] = 0
