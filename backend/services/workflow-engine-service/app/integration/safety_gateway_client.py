"""
Safety Gateway Client for Strategic Orchestration

This module provides integration with the Safety Gateway Platform
for the VALIDATE step of the Calculate > Validate > Commit pattern.
"""

import logging
import asyncio
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from enum import Enum
import httpx
import json
from datetime import datetime

logger = logging.getLogger(__name__)


class ValidationVerdict(Enum):
    """Safety validation verdict enumeration"""
    SAFE = "SAFE"
    WARNING = "WARNING"  
    UNSAFE = "UNSAFE"
    ERROR = "ERROR"


class FindingSeverity(Enum):
    """Clinical finding severity levels"""
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"
    INFO = "INFO"


@dataclass
class ValidationFinding:
    """Individual validation finding"""
    finding_id: str
    severity: FindingSeverity
    category: str  # DDI, ALLERGY, DOSE, CONTRAINDICATION, etc.
    description: str
    clinical_significance: str
    recommendation: str
    source_rule: str
    confidence_score: float


@dataclass
class SafetyValidationRequest:
    """Request structure for Safety Gateway validation"""
    proposal_set_id: str
    snapshot_id: str  # Same snapshot used in Calculate step
    proposals: List[Dict[str, Any]]
    patient_context: Dict[str, Any]
    validation_requirements: Dict[str, Any]
    correlation_id: str
    urgency: str = "ROUTINE"


@dataclass 
class SafetyValidationResponse:
    """Response structure from Safety Gateway validation"""
    validation_id: str
    verdict: ValidationVerdict
    overall_risk_score: float
    findings: List[ValidationFinding]
    validation_summary: Dict[str, Any]
    
    # Override capabilities
    override_tokens: Optional[List[str]] = None
    override_requirements: Optional[Dict[str, Any]] = None
    
    # Timing and metadata
    validation_time_ms: float = 0.0
    engines_used: List[str] = None
    kb_versions: Dict[str, str] = None


class SafetyGatewayClient:
    """
    Client for Safety Gateway Platform integration
    
    Coordinates comprehensive safety validation using:
    - CAE (Clinical Alert Engine) 
    - Protocol Engine
    - Drug-drug interaction screening
    - Allergy checking
    - Dose validation
    - Contraindication screening
    """
    
    def __init__(self, gateway_url: str = "http://localhost:8018"):
        self.gateway_url = gateway_url
        self.http_client = httpx.AsyncClient(timeout=30.0)
        
        # Validation configuration
        self.validation_config = {
            "enable_cae_engine": True,
            "enable_protocol_engine": True,
            "enable_ddi_screening": True,
            "enable_allergy_checking": True,
            "enable_dose_validation": True,
            "enable_contraindication_screening": True,
            "parallel_validation": True,  # Run engines in parallel
            "fail_fast": False,  # Complete all validations even if one fails
            "include_warnings": True,
            "generate_override_tokens": True
        }
    
    async def comprehensive_validation(
        self, 
        request: SafetyValidationRequest
    ) -> SafetyValidationResponse:
        """
        Perform comprehensive safety validation via Safety Gateway
        
        This is the VALIDATE step in Calculate > Validate > Commit pattern.
        Uses the same snapshot ID to ensure data consistency with Calculate step.
        """
        validation_start = datetime.utcnow()
        
        logger.info(f"Starting comprehensive validation {request.correlation_id} for snapshot {request.snapshot_id}")
        
        try:
            # Prepare validation request for Safety Gateway
            gateway_request = {
                "validation_type": "COMPREHENSIVE_MEDICATION_SAFETY",
                "request_id": request.correlation_id,
                "snapshot_id": request.snapshot_id,  # Critical for consistency
                "proposal_set_id": request.proposal_set_id,
                "proposals": request.proposals,
                "patient_context": request.patient_context,
                "validation_config": {
                    **self.validation_config,
                    **request.validation_requirements
                },
                "urgency": request.urgency,
                "timestamp": validation_start.isoformat()
            }
            
            # Call Safety Gateway comprehensive validation endpoint
            response = await self.http_client.post(
                f"{self.gateway_url}/api/v1/safety/comprehensive-validation",
                json=gateway_request,
                headers={
                    "Content-Type": "application/json",
                    "X-Correlation-ID": request.correlation_id,
                    "X-Validation-Source": "workflow-platform"
                }
            )
            response.raise_for_status()
            
            gateway_result = response.json()
            validation_time = (datetime.utcnow() - validation_start).total_seconds() * 1000
            
            # Parse findings from Safety Gateway response
            findings = []
            for finding_data in gateway_result.get("findings", []):
                finding = ValidationFinding(
                    finding_id=finding_data["finding_id"],
                    severity=FindingSeverity(finding_data["severity"]),
                    category=finding_data["category"],
                    description=finding_data["description"],
                    clinical_significance=finding_data["clinical_significance"],
                    recommendation=finding_data["recommendation"],
                    source_rule=finding_data["source_rule"],
                    confidence_score=finding_data["confidence_score"]
                )
                findings.append(finding)
            
            # Determine overall verdict based on findings
            verdict = self._determine_verdict(findings, gateway_result)
            
            return SafetyValidationResponse(
                validation_id=gateway_result["validation_id"],
                verdict=verdict,
                overall_risk_score=gateway_result["overall_risk_score"],
                findings=findings,
                validation_summary=gateway_result["validation_summary"],
                override_tokens=gateway_result.get("override_tokens"),
                override_requirements=gateway_result.get("override_requirements"),
                validation_time_ms=validation_time,
                engines_used=gateway_result.get("engines_used", []),
                kb_versions=gateway_result.get("kb_versions", {})
            )
            
        except httpx.HTTPStatusError as e:
            logger.error(f"Safety Gateway HTTP error {e.response.status_code}: {e.response.text}")
            return self._create_error_response(request.correlation_id, f"HTTP {e.response.status_code}: {e.response.text}")
            
        except Exception as e:
            logger.error(f"Safety Gateway validation failed for {request.correlation_id}: {str(e)}")
            return self._create_error_response(request.correlation_id, str(e))
    
    async def validate_override_request(
        self,
        validation_id: str,
        override_tokens: List[str],
        provider_justification: str,
        correlation_id: str
    ) -> Dict[str, Any]:
        """
        Validate provider override request
        
        Ensures override tokens are valid and provider has appropriate privileges.
        """
        try:
            override_request = {
                "validation_id": validation_id,
                "override_tokens": override_tokens,
                "provider_justification": provider_justification,
                "correlation_id": correlation_id,
                "timestamp": datetime.utcnow().isoformat()
            }
            
            response = await self.http_client.post(
                f"{self.gateway_url}/api/v1/safety/validate-override",
                json=override_request,
                headers={
                    "Content-Type": "application/json",
                    "X-Correlation-ID": correlation_id
                }
            )
            response.raise_for_status()
            
            return response.json()
            
        except Exception as e:
            logger.error(f"Override validation failed for {correlation_id}: {str(e)}")
            return {
                "status": "OVERRIDE_VALIDATION_FAILED",
                "error": str(e)
            }
    
    async def get_validation_status(self, validation_id: str) -> Dict[str, Any]:
        """Get status of a validation request"""
        try:
            response = await self.http_client.get(
                f"{self.gateway_url}/api/v1/safety/validation/{validation_id}/status"
            )
            response.raise_for_status()
            
            return response.json()
            
        except Exception as e:
            logger.error(f"Failed to get validation status for {validation_id}: {str(e)}")
            return {
                "status": "STATUS_UNAVAILABLE",
                "error": str(e)
            }
    
    async def health_check(self) -> Dict[str, Any]:
        """Check Safety Gateway health"""
        try:
            response = await self.http_client.get(f"{self.gateway_url}/health")
            
            if response.status_code == 200:
                health_data = response.json()
                return {
                    "status": "healthy",
                    "safety_gateway": health_data,
                    "validation_capabilities": [
                        "CAE Engine",
                        "Protocol Engine", 
                        "DDI Screening",
                        "Allergy Checking",
                        "Dose Validation",
                        "Contraindication Screening"
                    ]
                }
            else:
                return {
                    "status": "unhealthy",
                    "error": f"HTTP {response.status_code}"
                }
                
        except Exception as e:
            return {
                "status": "unavailable",
                "error": str(e)
            }
    
    def _determine_verdict(
        self, 
        findings: List[ValidationFinding], 
        gateway_result: Dict[str, Any]
    ) -> ValidationVerdict:
        """Determine overall validation verdict based on findings"""
        
        # Check for critical or high severity findings
        critical_findings = [f for f in findings if f.severity == FindingSeverity.CRITICAL]
        high_findings = [f for f in findings if f.severity == FindingSeverity.HIGH]
        
        if critical_findings:
            return ValidationVerdict.UNSAFE
        elif high_findings:
            return ValidationVerdict.WARNING
        elif gateway_result.get("overall_risk_score", 0) > 0.7:
            return ValidationVerdict.WARNING
        else:
            return ValidationVerdict.SAFE
    
    def _create_error_response(
        self, 
        correlation_id: str, 
        error_message: str
    ) -> SafetyValidationResponse:
        """Create error response for validation failures"""
        return SafetyValidationResponse(
            validation_id="",
            verdict=ValidationVerdict.ERROR,
            overall_risk_score=1.0,  # Max risk for errors
            findings=[
                ValidationFinding(
                    finding_id="VALIDATION_ERROR",
                    severity=FindingSeverity.CRITICAL,
                    category="SYSTEM_ERROR",
                    description=f"Safety validation system error: {error_message}",
                    clinical_significance="Cannot determine medication safety",
                    recommendation="Manual clinical review required",
                    source_rule="SYSTEM_ERROR_HANDLER",
                    confidence_score=1.0
                )
            ],
            validation_summary={
                "error": error_message,
                "status": "VALIDATION_FAILED"
            },
            validation_time_ms=0.0,
            engines_used=[],
            kb_versions={}
        )
    
    async def close(self):
        """Close HTTP client"""
        await self.http_client.aclose()


# Global client instance
safety_gateway_client = SafetyGatewayClient()