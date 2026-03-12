"""
Clinical Activity Models for Clinical Workflow Engine.
Implements clinical-specific activity types, validation, and error handling.
"""
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, Dict, Any, List
from datetime import datetime
import uuid


class ClinicalActivityType(Enum):
    """
    Clinical activity types with specific timing requirements.
    """
    SYNCHRONOUS = "sync"      # < 1 second (harmonization, validation)
    ASYNCHRONOUS = "async"    # 1-30 seconds (safety checks, context assembly)
    HUMAN = "human"          # minutes-hours (clinical review, signatures)


class DataSourceType(Enum):
    """
    Approved data source types for clinical workflows.
    """
    FHIR_STORE = "fhir_store"
    GRAPH_DB = "graph_db"
    SAFETY_GATEWAY = "safety_gateway"
    HARMONIZATION_SERVICE = "harmonization_service"
    CAE_SERVICE = "cae_service"
    CONTEXT_SERVICE = "context_service"
    MEDICATION_SERVICE = "medication_service"
    PATIENT_SERVICE = "patient_service"


class ClinicalErrorType(Enum):
    """
    Clinical error types with specific handling strategies.
    """
    SAFETY_ERROR = "safety"      # Stop workflow - critical drug interactions
    WARNING_ERROR = "warning"    # Continue with override - non-formulary med
    TECHNICAL_ERROR = "technical" # Retry logic - network timeouts
    DATA_SOURCE_ERROR = "data_source"  # Real data unavailable - fail workflow
    MOCK_DATA_ERROR = "mock_data"      # Mock data detected - immediate failure


class CompensationStrategy(Enum):
    """
    Compensation strategies for clinical workflow failures.
    """
    FULL_COMPENSATION = "full"      # Reverse all activities (safety risk)
    PARTIAL_COMPENSATION = "partial" # Reverse failed branch only
    FORWARD_RECOVERY = "forward"    # Retry with exponential backoff
    IMMEDIATE_FAILURE = "immediate_failure"  # Fail immediately (data integrity)


@dataclass
class ClinicalActivity:
    """
    Clinical activity definition with safety and data requirements.
    """
    activity_id: str
    activity_type: ClinicalActivityType
    timeout_seconds: int
    compensation_handler: Optional[str] = None
    safety_critical: bool = False
    requires_clinical_context: bool = False
    audit_level: str = "standard"  # standard, detailed, comprehensive
    real_data_only: bool = True  # MANDATORY: No mock or fallback data
    fail_on_unavailable: bool = True  # MANDATORY: Fail if real data unavailable
    approved_data_sources: Optional[List[DataSourceType]] = None
    created_at: datetime = None
    
    def __post_init__(self):
        if self.created_at is None:
            self.created_at = datetime.utcnow()
        if self.approved_data_sources is None:
            self.approved_data_sources = []


@dataclass
class ClinicalContext:
    """
    Clinical context for workflow execution.
    """
    patient_id: str
    encounter_id: Optional[str] = None
    provider_id: Optional[str] = None
    clinical_data: Dict[str, Any] = None
    safety_context: Dict[str, Any] = None
    workflow_context: Dict[str, Any] = None
    data_sources: Dict[str, str] = None  # source_type -> endpoint mapping
    created_at: datetime = None
    
    def __post_init__(self):
        if self.clinical_data is None:
            self.clinical_data = {}
        if self.safety_context is None:
            self.safety_context = {}
        if self.workflow_context is None:
            self.workflow_context = {}
        if self.data_sources is None:
            self.data_sources = {}
        if self.created_at is None:
            self.created_at = datetime.utcnow()


@dataclass
class ClinicalError:
    """
    Clinical error with context and recovery information.
    """
    error_id: str
    error_type: ClinicalErrorType
    error_message: str
    activity_id: str
    workflow_instance_id: str
    clinical_context: Optional[ClinicalContext] = None
    error_data: Dict[str, Any] = None
    recovery_strategy: Optional[CompensationStrategy] = None
    created_at: datetime = None
    
    def __post_init__(self):
        if self.error_id is None:
            self.error_id = str(uuid.uuid4())
        if self.error_data is None:
            self.error_data = {}
        if self.created_at is None:
            self.created_at = datetime.utcnow()


class ClinicalDataError(Exception):
    """
    Raised when mock data or unapproved sources detected.
    """
    def __init__(self, message: str, error_type: ClinicalErrorType = ClinicalErrorType.DATA_SOURCE_ERROR):
        super().__init__(message)
        self.error_type = error_type
        self.timestamp = datetime.utcnow()


class MockDataDetectedError(ClinicalDataError):
    """
    Raised when mock data is detected in clinical workflows.
    """
    def __init__(self, message: str, data_source: str):
        super().__init__(message, ClinicalErrorType.MOCK_DATA_ERROR)
        self.data_source = data_source


class UnapprovedDataSourceError(ClinicalDataError):
    """
    Raised when data comes from unapproved sources.
    """
    def __init__(self, message: str, data_source: str):
        super().__init__(message, ClinicalErrorType.DATA_SOURCE_ERROR)
        self.data_source = data_source
