"""
Data Source Types for Clinical Context Service
Defines all supported data sources and their configurations
"""
from enum import Enum
from dataclasses import dataclass
from typing import Dict, Optional, List


class DataSourceType(Enum):
    """
    Supported data source types for clinical context assembly.
    Each type represents a different service or data store that can provide clinical data.
    """
    # Core Clinical Services
    PATIENT_SERVICE = "patient_service"
    MEDICATION_SERVICE = "medication_service"
    LAB_SERVICE = "lab_service"
    ALLERGY_SERVICE = "allergy_service"
    CONDITION_SERVICE = "condition_service"
    ENCOUNTER_SERVICE = "encounter_service"
    OBSERVATION_SERVICE = "observation_service"
    
    # Data Stores
    FHIR_STORE = "fhir_store"
    GRAPH_DB = "graph_db"
    
    # Clinical Intelligence Services
    CAE_SERVICE = "cae_service"  # Clinical Assertion Engine
    SAFETY_GATEWAY = "safety_gateway"
    CLINICAL_REASONING_SERVICE = "clinical_reasoning_service"
    
    # Context and Orchestration
    CONTEXT_SERVICE_INTERNAL = "context_service_internal"
    WORKFLOW_ENGINE = "workflow_engine"
    
    # External Systems
    EHR_SYSTEM = "ehr_system"
    LABORATORY_SYSTEM = "laboratory_system"
    PHARMACY_SYSTEM = "pharmacy_system"
    DEVICE_DATA_SERVICE = "device_data_service"


@dataclass
class DataSourceConfig:
    """Configuration for a data source"""
    source_type: DataSourceType
    endpoint: str
    protocol: str = "http"  # http, grpc, graphql
    timeout_ms: int = 5000
    retry_count: int = 2
    circuit_breaker_enabled: bool = True
    health_check_endpoint: Optional[str] = None
    authentication_required: bool = True
    rate_limit_per_second: Optional[int] = None
    
    # Service-specific configurations
    headers: Dict[str, str] = None
    query_parameters: Dict[str, str] = None
    
    def __post_init__(self):
        if self.headers is None:
            self.headers = {}
        if self.query_parameters is None:
            self.query_parameters = {}


# Default configurations for each data source type
DEFAULT_DATA_SOURCE_CONFIGS = {
    DataSourceType.PATIENT_SERVICE: DataSourceConfig(
        source_type=DataSourceType.PATIENT_SERVICE,
        endpoint="http://localhost:8003",
        protocol="http",
        timeout_ms=3000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.MEDICATION_SERVICE: DataSourceConfig(
        source_type=DataSourceType.MEDICATION_SERVICE,
        endpoint="http://localhost:8009",
        protocol="http",
        timeout_ms=5000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.LAB_SERVICE: DataSourceConfig(
        source_type=DataSourceType.LAB_SERVICE,
        endpoint="http://localhost:8000",
        protocol="http",
        timeout_ms=4000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.ALLERGY_SERVICE: DataSourceConfig(
        source_type=DataSourceType.ALLERGY_SERVICE,
        endpoint="http://localhost:8003/api/allergies",
        protocol="http",
        timeout_ms=3000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.CONDITION_SERVICE: DataSourceConfig(
        source_type=DataSourceType.CONDITION_SERVICE,
        endpoint="http://localhost:8010",
        protocol="http",
        timeout_ms=4000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.ENCOUNTER_SERVICE: DataSourceConfig(
        source_type=DataSourceType.ENCOUNTER_SERVICE,
        endpoint="http://localhost:8020",
        protocol="http",
        timeout_ms=3000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.OBSERVATION_SERVICE: DataSourceConfig(
        source_type=DataSourceType.OBSERVATION_SERVICE,
        endpoint="http://localhost:8007",
        protocol="http",
        timeout_ms=4000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.FHIR_STORE: DataSourceConfig(
        source_type=DataSourceType.FHIR_STORE,
        endpoint="projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store",
        protocol="google_healthcare_api",
        timeout_ms=8000,
        authentication_required=True,
        headers={"Content-Type": "application/fhir+json", "Accept": "application/fhir+json"}
    ),
    
    DataSourceType.GRAPH_DB: DataSourceConfig(
        source_type=DataSourceType.GRAPH_DB,
        endpoint="http://localhost:7200",
        protocol="http",
        timeout_ms=6000,
        health_check_endpoint="/rest/repositories",
        headers={"Content-Type": "application/sparql-query", "Accept": "application/sparql-results+json"}
    ),
    
    DataSourceType.CAE_SERVICE: DataSourceConfig(
        source_type=DataSourceType.CAE_SERVICE,
        endpoint="localhost:8027",
        protocol="grpc",
        timeout_ms=10000,
        health_check_endpoint="/health",
        headers={}
    ),
    
    DataSourceType.SAFETY_GATEWAY: DataSourceConfig(
        source_type=DataSourceType.SAFETY_GATEWAY,
        endpoint="http://localhost:8028",
        protocol="http",
        timeout_ms=5000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.CLINICAL_REASONING_SERVICE: DataSourceConfig(
        source_type=DataSourceType.CLINICAL_REASONING_SERVICE,
        endpoint="http://localhost:8025",
        protocol="http",
        timeout_ms=7000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.CONTEXT_SERVICE_INTERNAL: DataSourceConfig(
        source_type=DataSourceType.CONTEXT_SERVICE_INTERNAL,
        endpoint="http://localhost:8016",
        protocol="http",
        timeout_ms=2000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.WORKFLOW_ENGINE: DataSourceConfig(
        source_type=DataSourceType.WORKFLOW_ENGINE,
        endpoint="http://localhost:8015",
        protocol="http",
        timeout_ms=3000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    ),
    
    DataSourceType.DEVICE_DATA_SERVICE: DataSourceConfig(
        source_type=DataSourceType.DEVICE_DATA_SERVICE,
        endpoint="http://localhost:8013",
        protocol="http",
        timeout_ms=4000,
        health_check_endpoint="/health",
        headers={"Content-Type": "application/json", "Accept": "application/json"}
    )
}


def get_data_source_config(source_type: DataSourceType) -> DataSourceConfig:
    """Get configuration for a data source type"""
    return DEFAULT_DATA_SOURCE_CONFIGS.get(source_type)


def get_all_data_source_types() -> List[DataSourceType]:
    """Get all supported data source types"""
    return list(DataSourceType)


def get_service_data_sources() -> List[DataSourceType]:
    """Get data sources that are microservices"""
    return [
        DataSourceType.PATIENT_SERVICE,
        DataSourceType.MEDICATION_SERVICE,
        DataSourceType.LAB_SERVICE,
        DataSourceType.ALLERGY_SERVICE,
        DataSourceType.CONDITION_SERVICE,
        DataSourceType.ENCOUNTER_SERVICE,
        DataSourceType.OBSERVATION_SERVICE,
        DataSourceType.CAE_SERVICE,
        DataSourceType.SAFETY_GATEWAY,
        DataSourceType.CLINICAL_REASONING_SERVICE,
        DataSourceType.WORKFLOW_ENGINE,
        DataSourceType.DEVICE_DATA_SERVICE
    ]


def get_data_store_sources() -> List[DataSourceType]:
    """Get data sources that are data stores"""
    return [
        DataSourceType.FHIR_STORE,
        DataSourceType.GRAPH_DB
    ]


def get_external_system_sources() -> List[DataSourceType]:
    """Get data sources that are external systems"""
    return [
        DataSourceType.EHR_SYSTEM,
        DataSourceType.LABORATORY_SYSTEM,
        DataSourceType.PHARMACY_SYSTEM
    ]


def is_grpc_source(source_type: DataSourceType) -> bool:
    """Check if data source uses gRPC protocol"""
    config = get_data_source_config(source_type)
    return config and config.protocol == "grpc"


def is_http_source(source_type: DataSourceType) -> bool:
    """Check if data source uses HTTP protocol"""
    config = get_data_source_config(source_type)
    return config and config.protocol == "http"


def get_timeout_ms(source_type: DataSourceType) -> int:
    """Get timeout in milliseconds for a data source"""
    config = get_data_source_config(source_type)
    return config.timeout_ms if config else 5000


def requires_authentication(source_type: DataSourceType) -> bool:
    """Check if data source requires authentication"""
    config = get_data_source_config(source_type)
    return config.authentication_required if config else True
