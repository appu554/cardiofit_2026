"""
GraphQL Federation Schema for Strategic Orchestration

Exposes the Calculate > Validate > Commit pattern via GraphQL
for Apollo Federation integration.
"""

import strawberry
from typing import Dict, Any, Optional, List
import httpx
import logging

from app.orchestration.strategic_orchestrator import (
    strategic_orchestrator,
    CalculateRequest
)

logger = logging.getLogger(__name__)


@strawberry.type
class MedicationRequestInput:
    """GraphQL input type for medication requests"""
    patient_id: str
    encounter_id: Optional[str] = None
    indication: str
    urgency: str = "ROUTINE"
    constraints: Optional[List[str]] = None
    medication: strawberry.scalars.JSON
    provider_id: str
    specialty: Optional[str] = None
    location: Optional[str] = None
    session_token: Optional[str] = None


@strawberry.type
class OrchestrationStepResult:
    """Result from a single orchestration step"""
    step: str
    status: str
    execution_time_ms: Optional[float] = None
    data: Optional[strawberry.scalars.JSON] = None


@strawberry.type
class OrchestrationResult:
    """Complete orchestration result"""
    status: str
    correlation_id: str
    execution_time_ms: Optional[float] = None
    
    # Success path fields
    medication_order_id: Optional[str] = None
    calculation: Optional[strawberry.scalars.JSON] = None
    validation: Optional[strawberry.scalars.JSON] = None
    commitment: Optional[strawberry.scalars.JSON] = None
    performance: Optional[strawberry.scalars.JSON] = None
    
    # Alternative paths
    validation_findings: Optional[List[strawberry.scalars.JSON]] = None
    override_tokens: Optional[List[str]] = None
    proposals: Optional[List[strawberry.scalars.JSON]] = None
    blocking_findings: Optional[List[strawberry.scalars.JSON]] = None
    alternative_approaches: Optional[List[strawberry.scalars.JSON]] = None
    
    # Error handling
    error_code: Optional[str] = None
    error_message: Optional[str] = None


@strawberry.type
class HealthStatus:
    """Health status for orchestration services"""
    status: str
    services: strawberry.scalars.JSON
    orchestration_pattern: str
    performance_targets: strawberry.scalars.JSON


@strawberry.input
class CreateMedicationOrderInput:
    """Input for creating a medication order via strategic orchestration"""
    patient_id: str
    encounter_id: Optional[str] = None
    indication: str
    urgency: str = "ROUTINE"
    constraints: Optional[List[str]] = strawberry.field(default_factory=list)
    medication: strawberry.scalars.JSON
    provider_id: str
    specialty: Optional[str] = None
    location: Optional[str] = None


@strawberry.input
class OverrideDecisionInput:
    """Input for provider override decisions"""
    correlation_id: str
    snapshot_id: str
    selected_proposal_index: int
    override_tokens: List[str]
    provider_justification: str


@strawberry.type
class Query:
    """GraphQL Query type for strategic orchestration"""
    
    @strawberry.field
    async def orchestration_health(self) -> HealthStatus:
        """Get health status of strategic orchestration services"""
        health_data = await strategic_orchestrator.health_check()
        return HealthStatus(
            status=health_data["status"],
            services=health_data["services"],
            orchestration_pattern=health_data["orchestration_pattern"],
            performance_targets=health_data["performance_targets"]
        )
    
    @strawberry.field
    async def orchestration_performance(self) -> strawberry.scalars.JSON:
        """Get performance metrics and targets"""
        return {
            "performance_targets": strategic_orchestrator.performance_targets,
            "architecture_pattern": "Calculate > Validate > Commit",
            "optimization_features": [
                "Recipe Snapshot Architecture",
                "Immutable clinical snapshots", 
                "66% performance improvement",
                "Sub-200ms total latency"
            ],
            "service_endpoints": {
                "flow2_go": strategic_orchestrator.flow2_go_url,
                "flow2_rust": strategic_orchestrator.flow2_rust_url,
                "safety_gateway": strategic_orchestrator.safety_gateway_url,
                "medication_service": strategic_orchestrator.medication_service_url
            }
        }


@strawberry.type
class Mutation:
    """GraphQL Mutation type for strategic orchestration"""
    
    @strawberry.mutation
    async def create_medication_order(
        self, 
        input: CreateMedicationOrderInput
    ) -> OrchestrationResult:
        """
        Create a medication order using Calculate > Validate > Commit orchestration
        
        This is the main entry point from the UI via Apollo Federation.
        Routes through the strategic orchestrator instead of directly to Flow2 engines.
        """
        import uuid
        
        correlation_id = str(uuid.uuid4())
        
        logger.info(f"GraphQL mutation: create_medication_order {correlation_id} for patient {input.patient_id}")
        
        try:
            # Convert GraphQL input to internal request format
            calculate_request = CalculateRequest(
                patient_id=input.patient_id,
                medication_request=input.medication,
                clinical_intent={
                    "indication": input.indication,
                    "urgency": input.urgency,
                    "constraints": input.constraints or []
                },
                provider_context={
                    "provider_id": input.provider_id,
                    "specialty": input.specialty,
                    "location": input.location,
                    "encounter_id": input.encounter_id
                },
                correlation_id=correlation_id,
                urgency=input.urgency
            )
            
            # Execute strategic orchestration
            result = await strategic_orchestrator.orchestrate_medication_request(calculate_request)
            
            # Convert result to GraphQL response
            return OrchestrationResult(
                status=result["status"],
                correlation_id=result["correlation_id"],
                execution_time_ms=result.get("execution_time_ms"),
                medication_order_id=result.get("medication_order_id"),
                calculation=result.get("calculation"),
                validation=result.get("validation"),
                commitment=result.get("commitment"),
                performance=result.get("performance"),
                validation_findings=result.get("validation_findings"),
                override_tokens=result.get("override_tokens"),
                proposals=result.get("proposals"),
                blocking_findings=result.get("blocking_findings"),
                alternative_approaches=result.get("alternative_approaches"),
                error_code=result.get("error_code"),
                error_message=result.get("error_message")
            )
            
        except Exception as e:
            logger.error(f"GraphQL mutation failed for {correlation_id}: {str(e)}")
            return OrchestrationResult(
                status="ERROR",
                correlation_id=correlation_id,
                error_code="GRAPHQL_ORCHESTRATION_FAILED",
                error_message=str(e)
            )
    
    @strawberry.mutation
    async def handle_provider_override(
        self, 
        input: OverrideDecisionInput
    ) -> OrchestrationResult:
        """Handle provider override decisions for WARNING validation results"""
        
        logger.info(f"GraphQL mutation: handle_provider_override for {input.correlation_id}")
        
        try:
            # Create override request for REST API
            override_data = {
                "correlation_id": input.correlation_id,
                "snapshot_id": input.snapshot_id,
                "selected_proposal_index": input.selected_proposal_index,
                "override_tokens": input.override_tokens,
                "provider_justification": input.provider_justification
            }
            
            # Call the REST API endpoint (internal service communication)
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    "http://localhost:8015/api/v1/orchestrate/medication/override",
                    json=override_data,
                    headers={"Content-Type": "application/json"}
                )
                response.raise_for_status()
                
                result = response.json()
                
                return OrchestrationResult(
                    status=result["status"],
                    correlation_id=result["correlation_id"],
                    execution_time_ms=result.get("execution_time_ms"),
                    medication_order_id=result.get("medication_order_id"),
                    commitment=result.get("commitment"),
                    error_code=result.get("error_code"),
                    error_message=result.get("error_message")
                )
                
        except Exception as e:
            logger.error(f"GraphQL override mutation failed for {input.correlation_id}: {str(e)}")
            return OrchestrationResult(
                status="ERROR",
                correlation_id=input.correlation_id,
                error_code="GRAPHQL_OVERRIDE_FAILED",
                error_message=str(e)
            )


# Create the federated schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    enable_federation_2=True
)