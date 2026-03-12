"""
Clinical Workflow Templates for Clinical Workflow Engine.
Implements BPMN 2.0 compliant clinical workflow templates with safety mechanisms.
"""
from enum import Enum
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
import uuid

from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, DataSourceType
)


class WorkflowTemplateType(Enum):
    """
    Types of clinical workflow templates.
    """
    MEDICATION_ORDERING = "medication_ordering"
    PATIENT_ADMISSION = "patient_admission"
    PATIENT_DISCHARGE = "patient_discharge"
    CLINICAL_ASSESSMENT = "clinical_assessment"
    EMERGENCY_RESPONSE = "emergency_response"


class SafetyCheckType(Enum):
    """
    Types of safety checks in clinical workflows.
    """
    DRUG_INTERACTION = "drug_interaction"
    ALLERGY_CHECK = "allergy_check"
    DOSAGE_VALIDATION = "dosage_validation"
    CONTRAINDICATION = "contraindication"
    DUPLICATE_THERAPY = "duplicate_therapy"
    CLINICAL_CONTEXT = "clinical_context"
    PROVIDER_AUTHORIZATION = "provider_authorization"


class CompensationPattern(Enum):
    """
    Clinical compensation patterns for workflow failures.
    """
    MEDICATION_REVERSAL = "medication_reversal"
    ORDER_CANCELLATION = "order_cancellation"
    CLINICAL_NOTIFICATION = "clinical_notification"
    ESCALATION_ALERT = "escalation_alert"
    AUDIT_LOG_ENTRY = "audit_log_entry"
    PATIENT_SAFETY_ALERT = "patient_safety_alert"


@dataclass
class SafetyCheck:
    """
    Safety check definition for clinical workflows.
    """
    check_id: str
    check_type: SafetyCheckType
    description: str
    required_data_sources: List[DataSourceType]
    timeout_seconds: int = 30
    critical: bool = True
    compensation_on_failure: List[CompensationPattern] = field(default_factory=list)
    retry_count: int = 3
    escalation_rules: Dict[str, Any] = field(default_factory=dict)


@dataclass
class WorkflowStep:
    """
    Individual step in a clinical workflow template.
    """
    step_id: str
    step_name: str
    step_type: str  # task, gateway, event, timer
    activity: Optional[ClinicalActivity] = None
    safety_checks: List[SafetyCheck] = field(default_factory=list)
    parallel_execution: bool = False
    timeout_minutes: int = 60
    escalation_minutes: int = 30
    required_roles: List[str] = field(default_factory=list)
    input_mapping: Dict[str, str] = field(default_factory=dict)
    output_mapping: Dict[str, str] = field(default_factory=dict)
    compensation_handler: Optional[str] = None
    next_steps: List[str] = field(default_factory=list)
    condition_expression: Optional[str] = None


@dataclass
class ClinicalWorkflowTemplate:
    """
    Clinical workflow template with BPMN 2.0 compliance and safety mechanisms.
    """
    template_id: str
    template_name: str
    template_type: WorkflowTemplateType
    version: str
    description: str
    
    # BPMN 2.0 Elements
    start_event: str
    end_events: List[str]
    steps: List[WorkflowStep]
    gateways: Dict[str, Dict[str, Any]] = field(default_factory=dict)
    timers: Dict[str, Dict[str, Any]] = field(default_factory=dict)
    
    # Clinical Safety Features
    global_safety_checks: List[SafetyCheck] = field(default_factory=list)
    emergency_stop_conditions: List[str] = field(default_factory=list)
    compensation_workflows: Dict[str, str] = field(default_factory=dict)
    
    # Compliance and Audit
    required_approvals: List[str] = field(default_factory=list)
    audit_level: str = "comprehensive"
    retention_years: int = 7
    phi_handling: bool = True
    
    # Performance Requirements
    max_execution_time_hours: int = 24
    sla_targets: Dict[str, int] = field(default_factory=dict)
    
    # Metadata
    created_by: str = "system"
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
    status: str = "active"


class ClinicalWorkflowTemplateBuilder:
    """
    Builder for creating clinical workflow templates with safety mechanisms.
    """
    
    def __init__(self):
        self.template = None
        self.current_step_id = None
    
    def create_template(
        self,
        template_id: str,
        template_name: str,
        template_type: WorkflowTemplateType,
        version: str = "1.0.0",
        description: str = ""
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Create a new clinical workflow template.
        """
        self.template = ClinicalWorkflowTemplate(
            template_id=template_id,
            template_name=template_name,
            template_type=template_type,
            version=version,
            description=description,
            start_event=f"{template_id}_start",
            end_events=[f"{template_id}_end"],
            steps=[]
        )
        return self
    
    def add_start_event(self, event_id: str = None) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add start event to the workflow.
        """
        if not event_id:
            event_id = f"{self.template.template_id}_start"
        
        self.template.start_event = event_id
        return self
    
    def add_clinical_task(
        self,
        step_id: str,
        step_name: str,
        activity: ClinicalActivity,
        safety_checks: List[SafetyCheck] = None,
        timeout_minutes: int = 60,
        required_roles: List[str] = None
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add a clinical task step to the workflow.
        """
        if safety_checks is None:
            safety_checks = []
        if required_roles is None:
            required_roles = []
        
        step = WorkflowStep(
            step_id=step_id,
            step_name=step_name,
            step_type="task",
            activity=activity,
            safety_checks=safety_checks,
            timeout_minutes=timeout_minutes,
            required_roles=required_roles
        )
        
        self.template.steps.append(step)
        self.current_step_id = step_id
        return self
    
    def add_safety_gateway(
        self,
        gateway_id: str,
        gateway_name: str,
        safety_checks: List[SafetyCheck],
        parallel_execution: bool = False
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add a safety gateway for clinical validation.
        """
        step = WorkflowStep(
            step_id=gateway_id,
            step_name=gateway_name,
            step_type="gateway",
            safety_checks=safety_checks,
            parallel_execution=parallel_execution
        )
        
        self.template.steps.append(step)
        self.template.gateways[gateway_id] = {
            "type": "exclusive" if not parallel_execution else "parallel",
            "safety_checks": [check.check_id for check in safety_checks]
        }
        
        self.current_step_id = gateway_id
        return self
    
    def add_human_task(
        self,
        step_id: str,
        step_name: str,
        required_roles: List[str],
        timeout_minutes: int = 240,  # 4 hours default
        escalation_minutes: int = 60
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add a human task step (clinical review, approval, etc.).
        """
        activity = ClinicalActivity(
            activity_id=step_id,
            activity_type=ClinicalActivityType.HUMAN,
            timeout_seconds=timeout_minutes * 60,
            requires_clinical_context=True,
            audit_level="comprehensive"
        )
        
        step = WorkflowStep(
            step_id=step_id,
            step_name=step_name,
            step_type="task",
            activity=activity,
            timeout_minutes=timeout_minutes,
            escalation_minutes=escalation_minutes,
            required_roles=required_roles
        )
        
        self.template.steps.append(step)
        self.current_step_id = step_id
        return self
    
    def add_timer_event(
        self,
        timer_id: str,
        timer_name: str,
        duration_minutes: int,
        timer_type: str = "duration"
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add a timer event to the workflow.
        """
        step = WorkflowStep(
            step_id=timer_id,
            step_name=timer_name,
            step_type="timer"
        )
        
        self.template.steps.append(step)
        self.template.timers[timer_id] = {
            "type": timer_type,
            "duration_minutes": duration_minutes,
            "escalation_enabled": True
        }
        
        self.current_step_id = timer_id
        return self
    
    def add_compensation_handler(
        self,
        compensation_id: str,
        compensation_workflow: str
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add compensation handler for the current step.
        """
        if self.current_step_id:
            for step in self.template.steps:
                if step.step_id == self.current_step_id:
                    step.compensation_handler = compensation_id
                    break
            
            self.template.compensation_workflows[compensation_id] = compensation_workflow
        
        return self
    
    def add_sequence_flow(
        self,
        from_step: str,
        to_step: str,
        condition: str = None
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add sequence flow between steps.
        """
        for step in self.template.steps:
            if step.step_id == from_step:
                step.next_steps.append(to_step)
                if condition:
                    step.condition_expression = condition
                break
        
        return self
    
    def add_global_safety_check(
        self,
        safety_check: SafetyCheck
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add global safety check that applies to entire workflow.
        """
        self.template.global_safety_checks.append(safety_check)
        return self
    
    def add_emergency_stop_condition(
        self,
        condition: str
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Add emergency stop condition.
        """
        self.template.emergency_stop_conditions.append(condition)
        return self
    
    def set_sla_targets(
        self,
        targets: Dict[str, int]
    ) -> 'ClinicalWorkflowTemplateBuilder':
        """
        Set SLA targets for the workflow.
        """
        self.template.sla_targets = targets
        return self
    
    def build(self) -> ClinicalWorkflowTemplate:
        """
        Build and return the clinical workflow template.
        """
        if not self.template:
            raise ValueError("No template created. Call create_template() first.")
        
        # Validate template
        self._validate_template()
        
        # Update timestamp
        self.template.updated_at = datetime.utcnow()
        
        return self.template
    
    def _validate_template(self):
        """
        Validate the clinical workflow template.
        """
        if not self.template.steps:
            raise ValueError("Template must have at least one step")
        
        if not self.template.start_event:
            raise ValueError("Template must have a start event")
        
        if not self.template.end_events:
            raise ValueError("Template must have at least one end event")
        
        # Validate step references
        step_ids = {step.step_id for step in self.template.steps}
        for step in self.template.steps:
            for next_step in step.next_steps:
                if next_step not in step_ids and next_step not in self.template.end_events:
                    raise ValueError(f"Invalid next step reference: {next_step}")
        
        # Validate safety checks have required data sources
        for step in self.template.steps:
            for safety_check in step.safety_checks:
                if not safety_check.required_data_sources:
                    raise ValueError(f"Safety check {safety_check.check_id} must specify required data sources")


# Pre-built safety checks for common clinical scenarios
class CommonSafetyChecks:
    """
    Pre-built safety checks for common clinical scenarios.
    """
    
    @staticmethod
    def drug_interaction_check() -> SafetyCheck:
        return SafetyCheck(
            check_id="drug_interaction_check",
            check_type=SafetyCheckType.DRUG_INTERACTION,
            description="Check for drug-drug interactions using CAE service",
            required_data_sources=[DataSourceType.CAE_SERVICE, DataSourceType.MEDICATION_SERVICE],
            timeout_seconds=10,
            critical=True,
            compensation_on_failure=[CompensationPattern.MEDICATION_REVERSAL, CompensationPattern.CLINICAL_NOTIFICATION],
            retry_count=2
        )
    
    @staticmethod
    def allergy_check() -> SafetyCheck:
        return SafetyCheck(
            check_id="allergy_check",
            check_type=SafetyCheckType.ALLERGY_CHECK,
            description="Check patient allergies against prescribed medications",
            required_data_sources=[DataSourceType.PATIENT_SERVICE, DataSourceType.FHIR_STORE],
            timeout_seconds=5,
            critical=True,
            compensation_on_failure=[CompensationPattern.ORDER_CANCELLATION, CompensationPattern.PATIENT_SAFETY_ALERT],
            retry_count=3
        )
    
    @staticmethod
    def dosage_validation() -> SafetyCheck:
        return SafetyCheck(
            check_id="dosage_validation",
            check_type=SafetyCheckType.DOSAGE_VALIDATION,
            description="Validate medication dosage against clinical guidelines",
            required_data_sources=[DataSourceType.CAE_SERVICE, DataSourceType.HARMONIZATION_SERVICE],
            timeout_seconds=15,
            critical=True,
            compensation_on_failure=[CompensationPattern.CLINICAL_NOTIFICATION, CompensationPattern.ESCALATION_ALERT],
            retry_count=2
        )
    
    @staticmethod
    def provider_authorization() -> SafetyCheck:
        return SafetyCheck(
            check_id="provider_authorization",
            check_type=SafetyCheckType.PROVIDER_AUTHORIZATION,
            description="Verify provider has authorization for this action",
            required_data_sources=[DataSourceType.CONTEXT_SERVICE],
            timeout_seconds=5,
            critical=True,
            compensation_on_failure=[CompensationPattern.AUDIT_LOG_ENTRY, CompensationPattern.ESCALATION_ALERT],
            retry_count=1
        )
