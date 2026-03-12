"""
Safety Gateway HTTP Client for workflow engine integration.

This client interfaces with the Safety Gateway HTTP REST API to perform
comprehensive validation as part of the Calculate > Validate > Commit workflow.
"""
import logging
import asyncio
from datetime import datetime, timezone
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from enum import Enum
import httpx
import json

logger = logging.getLogger(__name__)


class ValidationVerdict(Enum):
    """Validation verdict enumeration"""
    SAFE = "SAFE"
    WARNING = "WARNING"
    UNSAFE = "UNSAFE"
    MANUAL_REVIEW = "MANUAL_REVIEW"
    ERROR = "ERROR"


@dataclass
class SafetyValidationRequest:
    """Safety validation request structure"""
    proposal_set_id: str
    snapshot_id: str
    proposals: List[Dict[str, Any]]
    patient_context: Dict[str, Any]
    validation_requirements: Dict[str, Any]
    correlation_id: str
    request_id: Optional[str] = None
    patient_id: Optional[str] = None
    clinician_id: Optional[str] = None
    priority: Optional[str] = "ROUTINE"
    source: Optional[str] = "workflow_engine"


@dataclass
class ValidationFinding:
    """Individual validation finding"""
    finding_id: str
    severity: str
    category: str
    description: str
    clinical_significance: str
    recommendation: str
    confidence_score: float
    engine_source: str


@dataclass
class EngineResult:
    """Individual engine execution result"""
    engine_id: str
    engine_name: str
    status: str
    risk_score: float
    violations: List[str]
    warnings: List[str]
    confidence: float
    duration_ms: int
    tier: int
    error: Optional[str] = None


@dataclass
class SafetyValidationResponse:
    """Safety validation response structure"""
    validation_id: str
    verdict: ValidationVerdict
    findings: List[ValidationFinding]
    override_tokens: Optional[List[str]]
    override_requirements: Optional[Dict[str, Any]]
    processing_time_ms: int
    engine_results: List[EngineResult]
    risk_score: float
    timestamp: datetime
    metadata: Optional[Dict[str, Any]] = None


class SafetyGatewayHTTPClient:
    """
    HTTP client for Safety Gateway REST API integration.
    
    This client provides comprehensive validation capabilities for the
    Calculate > Validate > Commit workflow pattern.
    """
    
    def __init__(self, base_url: str = "http://localhost:8018", timeout: float = 30.0):
        """
        Initialize Safety Gateway HTTP client.
        
        Args:
            base_url: Safety Gateway HTTP service base URL
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.client = httpx.AsyncClient(timeout=timeout)
        self._initialized = False
        
        logger.info(f"Safety Gateway HTTP client initialized with base URL: {base_url}")
    
    async def initialize(self) -> bool:
        """Initialize the client and verify connectivity."""
        try:
            # Health check to verify connectivity
            response = await self.client.get(f"{self.base_url}/api/v1/health")
            if response.status_code == 200:
                health_data = response.json()
                logger.info(f"Safety Gateway health check passed: {health_data.get('status')}")
                self._initialized = True
                return True
            else:
                logger.error(f"Safety Gateway health check failed: {response.status_code}")
                return False
        except Exception as e:
            logger.error(f"Failed to initialize Safety Gateway client: {e}")
            return False
    
    @property
    def initialized(self) -> bool:
        """Check if client is initialized."""
        return self._initialized
    
    async def comprehensive_validation(
        self, 
        request: SafetyValidationRequest
    ) -> SafetyValidationResponse:
        """
        Perform comprehensive safety validation.
        
        This is the main method called by the strategic orchestrator during
        the Validate phase of the Calculate > Validate > Commit workflow.
        
        Args:
            request: Safety validation request
            
        Returns:
            SafetyValidationResponse with validation results
            
        Raises:
            ValidationError: If validation fails or encounters errors
        """
        start_time = datetime.now(timezone.utc)
        request_logger = logger.bind(
            correlation_id=request.correlation_id,
            proposal_set_id=request.proposal_set_id,
            snapshot_id=request.snapshot_id
        )
        
        request_logger.info("Starting comprehensive safety validation")
        
        try:
            # Prepare HTTP request payload
            payload = self._prepare_validation_payload(request)
            
            # Make HTTP request to Safety Gateway
            response = await self.client.post(
                f"{self.base_url}/api/v1/validate/comprehensive",
                json=payload,
                headers={"Content-Type": "application/json"}
            )
            
            # Handle response
            if response.status_code != 200:
                error_msg = f"Safety Gateway validation failed: {response.status_code}"
                if response.status_code >= 400:
                    try:
                        error_data = response.json()
                        error_msg += f" - {error_data.get('error', {}).get('message', 'Unknown error')}"
                    except:
                        error_msg += f" - {response.text}"
                
                request_logger.error(error_msg)
                return SafetyValidationResponse(
                    validation_id=request.request_id or request.correlation_id,
                    verdict=ValidationVerdict.ERROR,
                    findings=[],
                    override_tokens=None,
                    override_requirements=None,
                    processing_time_ms=int((datetime.now(timezone.utc) - start_time).total_seconds() * 1000),
                    engine_results=[],
                    risk_score=1.0,  # Maximum risk for errors
                    timestamp=datetime.now(timezone.utc),
                    metadata={"error": error_msg}
                )
            
            # Parse successful response
            validation_data = response.json()
            validation_response = self._parse_validation_response(validation_data, start_time)
            
            request_logger.info(
                "Safety validation completed successfully",
                verdict=validation_response.verdict.value,
                risk_score=validation_response.risk_score,
                processing_time_ms=validation_response.processing_time_ms,
                findings_count=len(validation_response.findings)
            )
            
            return validation_response
            
        except httpx.TimeoutException:
            error_msg = f"Safety Gateway validation timeout after {self.timeout}s"
            request_logger.error(error_msg)
            return SafetyValidationResponse(
                validation_id=request.request_id or request.correlation_id,
                verdict=ValidationVerdict.ERROR,
                findings=[],
                override_tokens=None,
                override_requirements=None,
                processing_time_ms=int(self.timeout * 1000),
                engine_results=[],
                risk_score=1.0,
                timestamp=datetime.now(timezone.utc),
                metadata={"error": error_msg, "error_type": "timeout"}
            )
            
        except Exception as e:
            error_msg = f"Safety Gateway validation error: {str(e)}"
            request_logger.error(error_msg, exc_info=True)
            return SafetyValidationResponse(
                validation_id=request.request_id or request.correlation_id,
                verdict=ValidationVerdict.ERROR,
                findings=[],
                override_tokens=None,
                override_requirements=None,
                processing_time_ms=int((datetime.now(timezone.utc) - start_time).total_seconds() * 1000),
                engine_results=[],
                risk_score=1.0,
                timestamp=datetime.now(timezone.utc),
                metadata={"error": error_msg, "error_type": "exception"}
            )
    
    async def validate_override(
        self, 
        token_id: str, 
        clinician_id: str, 
        reason: str
    ) -> Dict[str, Any]:
        """
        Validate an override token.
        
        Args:
            token_id: Override token ID
            clinician_id: Clinician requesting override
            reason: Override reason
            
        Returns:
            Override validation result
        """
        try:
            payload = {
                "token_id": token_id,
                "clinician_id": clinician_id,
                "reason": reason
            }
            
            response = await self.client.post(
                f"{self.base_url}/api/v1/override/validate",
                json=payload,
                headers={"Content-Type": "application/json"}
            )
            
            if response.status_code == 200:
                return response.json()
            else:
                logger.error(f"Override validation failed: {response.status_code}")
                return {"valid": False, "reason": f"Validation failed: {response.status_code}"}
                
        except Exception as e:
            logger.error(f"Override validation error: {e}")
            return {"valid": False, "reason": f"Error: {str(e)}"}
    
    async def get_engine_status(self) -> Dict[str, Any]:
        """Get safety engine status information."""
        try:
            response = await self.client.get(f"{self.base_url}/api/v1/engines/status")
            if response.status_code == 200:
                return response.json()
            else:
                logger.error(f"Engine status request failed: {response.status_code}")
                return {"engines": [], "metadata": {"error": f"Request failed: {response.status_code}"}}
        except Exception as e:
            logger.error(f"Engine status error: {e}")
            return {"engines": [], "metadata": {"error": str(e)}}
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check on Safety Gateway."""
        try:
            response = await self.client.get(f"{self.base_url}/api/v1/health")
            if response.status_code == 200:
                return response.json()
            else:
                return {"status": "unhealthy", "error": f"HTTP {response.status_code}"}
        except Exception as e:
            return {"status": "unhealthy", "error": str(e)}
    
    def _prepare_validation_payload(self, request: SafetyValidationRequest) -> Dict[str, Any]:
        """Prepare HTTP request payload from validation request."""
        return {
            "proposal_set_id": request.proposal_set_id,
            "snapshot_id": request.snapshot_id,
            "proposals": request.proposals,
            "patient_context": request.patient_context,
            "validation_requirements": request.validation_requirements,
            "correlation_id": request.correlation_id,
            "request_id": request.request_id or request.correlation_id,
            "patient_id": request.patient_id,
            "clinician_id": request.clinician_id,
            "priority": request.priority,
            "source": request.source
        }
    
    def _parse_validation_response(
        self, 
        data: Dict[str, Any], 
        start_time: datetime
    ) -> SafetyValidationResponse:
        """Parse HTTP response data into SafetyValidationResponse."""
        
        # Parse findings
        findings = []
        for finding_data in data.get("findings", []):
            findings.append(ValidationFinding(
                finding_id=finding_data["finding_id"],
                severity=finding_data["severity"],
                category=finding_data["category"],
                description=finding_data["description"],
                clinical_significance=finding_data["clinical_significance"],
                recommendation=finding_data["recommendation"],
                confidence_score=finding_data["confidence_score"],
                engine_source=finding_data["engine_source"]
            ))
        
        # Parse engine results
        engine_results = []
        for result_data in data.get("engine_results", []):
            engine_results.append(EngineResult(
                engine_id=result_data["engine_id"],
                engine_name=result_data["engine_name"],
                status=result_data["status"],
                risk_score=result_data["risk_score"],
                violations=result_data["violations"],
                warnings=result_data["warnings"],
                confidence=result_data["confidence"],
                duration_ms=result_data["duration_ms"],
                tier=result_data["tier"],
                error=result_data.get("error")
            ))
        
        # Parse timestamp
        timestamp_str = data.get("timestamp")
        if timestamp_str:
            timestamp = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
        else:
            timestamp = datetime.now(timezone.utc)
        
        # Determine verdict
        verdict_str = data.get("verdict", "ERROR")
        try:
            verdict = ValidationVerdict(verdict_str)
        except ValueError:
            logger.warning(f"Unknown verdict: {verdict_str}, defaulting to ERROR")
            verdict = ValidationVerdict.ERROR
        
        return SafetyValidationResponse(
            validation_id=data.get("validation_id", "unknown"),
            verdict=verdict,
            findings=findings,
            override_tokens=data.get("override_tokens"),
            override_requirements=data.get("override_requirements"),
            processing_time_ms=data.get("processing_time_ms", 0),
            engine_results=engine_results,
            risk_score=data.get("risk_score", 0.0),
            timestamp=timestamp,
            metadata=data.get("metadata")
        )
    
    async def close(self):
        """Close the HTTP client."""
        await self.client.aclose()
        logger.info("Safety Gateway HTTP client closed")


# Global client instance for workflow engine
safety_gateway_client = SafetyGatewayHTTPClient()


class ValidationError(Exception):
    """Custom exception for validation errors."""
    pass