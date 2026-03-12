"""
Google FHIR Healthcare API Configuration for KB7 Terminology Service

This module provides configuration management for Google Cloud Healthcare API
integration with the KB7 hybrid terminology architecture.
"""

import os
import json
from typing import Optional, Dict, Any
from dataclasses import dataclass
from pathlib import Path


@dataclass
class GoogleFHIRConfig:
    """Configuration for Google FHIR Healthcare API integration."""

    # Google Cloud Project Configuration
    project_id: str
    location: str
    dataset_id: str
    fhir_store_id: str

    # Authentication Configuration
    credentials_path: Optional[str] = None
    credentials_json: Optional[str] = None

    # API Configuration
    base_url: Optional[str] = None
    timeout: int = 30
    max_retries: int = 3
    retry_delay: float = 1.0

    # Resource Configuration
    supported_formats: list = None
    enable_versioning: bool = True
    enable_audit_logging: bool = True

    # Performance Configuration
    max_concurrent_requests: int = 10
    batch_size: int = 100
    cache_ttl: int = 3600

    # Integration Configuration
    sync_enabled: bool = True
    fallback_to_local: bool = True
    sync_interval: int = 300  # seconds

    def __post_init__(self):
        """Initialize computed fields and defaults."""
        if self.base_url is None:
            self.base_url = (
                f"https://healthcare.googleapis.com/v1/projects/{self.project_id}/"
                f"locations/{self.location}/datasets/{self.dataset_id}/"
                f"fhirStores/{self.fhir_store_id}/fhir"
            )

        if self.supported_formats is None:
            self.supported_formats = ["application/fhir+json", "application/json"]

    @property
    def fhir_store_name(self) -> str:
        """Get the fully qualified FHIR store name."""
        return (
            f"projects/{self.project_id}/locations/{self.location}/"
            f"datasets/{self.dataset_id}/fhirStores/{self.fhir_store_id}"
        )

    @property
    def has_credentials(self) -> bool:
        """Check if credentials are configured."""
        return bool(self.credentials_path or self.credentials_json or
                   os.getenv('GOOGLE_APPLICATION_CREDENTIALS'))


def load_google_fhir_config() -> GoogleFHIRConfig:
    """
    Load Google FHIR configuration from environment variables.

    Returns:
        GoogleFHIRConfig: Configured instance

    Raises:
        ValueError: If required configuration is missing
    """
    # Required configuration
    project_id = os.getenv('GOOGLE_CLOUD_PROJECT_ID')
    location = os.getenv('GOOGLE_CLOUD_LOCATION', 'asia-south1')
    dataset_id = os.getenv('GOOGLE_CLOUD_DATASET_ID')
    fhir_store_id = os.getenv('GOOGLE_CLOUD_FHIR_STORE_ID')

    # Validate required fields
    if not project_id:
        raise ValueError("GOOGLE_CLOUD_PROJECT_ID environment variable is required")
    if not dataset_id:
        raise ValueError("GOOGLE_CLOUD_DATASET_ID environment variable is required")
    if not fhir_store_id:
        raise ValueError("GOOGLE_CLOUD_FHIR_STORE_ID environment variable is required")

    # Authentication configuration
    credentials_path = os.getenv('GOOGLE_CLOUD_CREDENTIALS_PATH')
    credentials_json = os.getenv('GOOGLE_APPLICATION_CREDENTIALS_JSON')

    # Optional configuration with defaults
    config = GoogleFHIRConfig(
        project_id=project_id,
        location=location,
        dataset_id=dataset_id,
        fhir_store_id=fhir_store_id,
        credentials_path=credentials_path,
        credentials_json=credentials_json,
        timeout=int(os.getenv('GOOGLE_FHIR_TIMEOUT', '30')),
        max_retries=int(os.getenv('GOOGLE_FHIR_MAX_RETRIES', '3')),
        retry_delay=float(os.getenv('GOOGLE_FHIR_RETRY_DELAY', '1.0')),
        max_concurrent_requests=int(os.getenv('GOOGLE_FHIR_MAX_CONCURRENT', '10')),
        batch_size=int(os.getenv('GOOGLE_FHIR_BATCH_SIZE', '100')),
        cache_ttl=int(os.getenv('GOOGLE_FHIR_CACHE_TTL', '3600')),
        sync_enabled=os.getenv('GOOGLE_FHIR_SYNC_ENABLED', 'true').lower() == 'true',
        fallback_to_local=os.getenv('GOOGLE_FHIR_FALLBACK_ENABLED', 'true').lower() == 'true',
        sync_interval=int(os.getenv('GOOGLE_FHIR_SYNC_INTERVAL', '300')),
        enable_versioning=os.getenv('GOOGLE_FHIR_ENABLE_VERSIONING', 'true').lower() == 'true',
        enable_audit_logging=os.getenv('GOOGLE_FHIR_ENABLE_AUDIT', 'true').lower() == 'true',
    )

    return config


def validate_google_credentials(config: GoogleFHIRConfig) -> bool:
    """
    Validate Google Cloud credentials configuration.

    Args:
        config: GoogleFHIRConfig instance

    Returns:
        bool: True if credentials are valid and accessible
    """
    try:
        if config.credentials_path:
            # Check if credentials file exists and is readable
            cred_path = Path(config.credentials_path)
            if not cred_path.exists():
                return False

            # Try to parse as JSON
            with open(cred_path, 'r') as f:
                cred_data = json.load(f)

            # Basic validation of service account key structure
            required_fields = ['type', 'project_id', 'private_key_id', 'private_key', 'client_email']
            if not all(field in cred_data for field in required_fields):
                return False

        elif config.credentials_json:
            # Validate JSON credentials string
            cred_data = json.loads(config.credentials_json)
            required_fields = ['type', 'project_id', 'private_key_id', 'private_key', 'client_email']
            if not all(field in cred_data for field in required_fields):
                return False

        elif os.getenv('GOOGLE_APPLICATION_CREDENTIALS'):
            # Check if default credentials file exists
            default_creds = Path(os.getenv('GOOGLE_APPLICATION_CREDENTIALS'))
            if not default_creds.exists():
                return False
        else:
            # No credentials configured
            return False

        return True

    except (json.JSONDecodeError, FileNotFoundError, PermissionError):
        return False


def get_google_client_options(config: GoogleFHIRConfig) -> Dict[str, Any]:
    """
    Get Google API client options from configuration.

    Args:
        config: GoogleFHIRConfig instance

    Returns:
        Dict: Client options for Google API client initialization
    """
    options = {
        'scopes': ['https://www.googleapis.com/auth/cloud-healthcare'],
        'quota_project_id': config.project_id,
    }

    if config.credentials_path:
        options['credentials_path'] = config.credentials_path
    elif config.credentials_json:
        options['credentials_info'] = json.loads(config.credentials_json)

    return options


def create_fhir_resource_url(config: GoogleFHIRConfig, resource_type: str,
                           resource_id: Optional[str] = None) -> str:
    """
    Create a FHIR resource URL for Google Healthcare API.

    Args:
        config: GoogleFHIRConfig instance
        resource_type: FHIR resource type (e.g., 'CodeSystem', 'ValueSet')
        resource_id: Optional resource ID

    Returns:
        str: Complete FHIR resource URL
    """
    url = f"{config.base_url}/{resource_type}"
    if resource_id:
        url += f"/{resource_id}"
    return url


def create_fhir_operation_url(config: GoogleFHIRConfig, resource_type: str,
                            operation: str, resource_id: Optional[str] = None) -> str:
    """
    Create a FHIR operation URL for Google Healthcare API.

    Args:
        config: GoogleFHIRConfig instance
        resource_type: FHIR resource type (e.g., 'CodeSystem', 'ValueSet')
        operation: FHIR operation name (e.g., '$lookup', '$expand')
        resource_id: Optional resource ID for instance-level operations

    Returns:
        str: Complete FHIR operation URL
    """
    if resource_id:
        # Instance-level operation
        return f"{config.base_url}/{resource_type}/{resource_id}/{operation}"
    else:
        # Type-level operation
        return f"{config.base_url}/{resource_type}/{operation}"


# Configuration validation constants
REQUIRED_ENV_VARS = [
    'GOOGLE_CLOUD_PROJECT_ID',
    'GOOGLE_CLOUD_DATASET_ID',
    'GOOGLE_CLOUD_FHIR_STORE_ID'
]

OPTIONAL_ENV_VARS = [
    'GOOGLE_CLOUD_LOCATION',
    'GOOGLE_CLOUD_CREDENTIALS_PATH',
    'GOOGLE_APPLICATION_CREDENTIALS_JSON',
    'GOOGLE_FHIR_TIMEOUT',
    'GOOGLE_FHIR_MAX_RETRIES',
    'GOOGLE_FHIR_RETRY_DELAY',
    'GOOGLE_FHIR_MAX_CONCURRENT',
    'GOOGLE_FHIR_BATCH_SIZE',
    'GOOGLE_FHIR_CACHE_TTL',
    'GOOGLE_FHIR_SYNC_ENABLED',
    'GOOGLE_FHIR_FALLBACK_ENABLED',
    'GOOGLE_FHIR_SYNC_INTERVAL',
    'GOOGLE_FHIR_ENABLE_VERSIONING',
    'GOOGLE_FHIR_ENABLE_AUDIT'
]

SUPPORTED_FHIR_OPERATIONS = [
    '$lookup',      # CodeSystem lookup
    '$expand',      # ValueSet expansion
    '$translate',   # ConceptMap translation
    '$validate-code'  # Code validation
]

SUPPORTED_RESOURCE_TYPES = [
    'CodeSystem',
    'ValueSet',
    'ConceptMap',
    'NamingSystem',
    'TerminologyCapabilities'
]