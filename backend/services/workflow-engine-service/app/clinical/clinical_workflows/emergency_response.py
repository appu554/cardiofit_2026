"""
Emergency Response Workflow Template.
Implements BPMN 2.0 compliant emergency response workflow with digital reflex arc capabilities.
"""
from typing import Dict, Any, List
from datetime import datetime, timedelta

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplate, WorkflowTemplateType, WorkflowStep, SafetyCheck, SafetyLevel
)
from app.models.clinical_activity_models import ClinicalActivity, ClinicalActivityType, DataSourceType
from app.clinical.clinical_workflows.workflow_template_builder import ClinicalWorkflowTemplateBuilder


class EmergencyResponseWorkflowTemplate:
    """
    Emergency Response Workflow Template.
    
    Implements rapid emergency response protocols with:
    - Digital reflex arc execution (sub-100ms)
    - Autonomous intervention capabilities
    - Real-time safety monitoring
    - Parallel emergency team activation
    - Comprehensive audit trail for emergency situations
    
    Emergency Types Supported:
    - Cardiac arrest
    - Anaphylaxis
    - Sepsis
    - Clinical deterioration
    - Respiratory failure
    - Stroke
    """
    
    @staticmethod
    def create_template() -> Dict[str, Any]:
        """
        Create the emergency response workflow template.
        """
        builder = ClinicalWorkflowTemplateBuilder()
        
        # Create base template
        template = (builder
            .create_template(
                template_id="emergency-response-workflow-v1",
                template_name="Emergency Response Workflow",
                template_type=WorkflowTemplateType.EMERGENCY_RESPONSE,
                version="1.0.0",
                description="Digital reflex arc emergency response workflow with autonomous interventions"
            )
            .add_start_event("emergency_alert_received")
        )
        
        # Step 1: Emergency Triage and Classification (Autonomous - 10ms)
        template.add_step(WorkflowStep(
            step_id="emergency_triage",
            step_name="Emergency Triage and Classification",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="emergency_triage_activity",
                activity_type=ClinicalActivityType.SYNCHRONOUS,
                timeout_seconds=1,  # 1 second max
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive",
                approved_data_sources=[
                    DataSourceType.FHIR_STORE,
                    DataSourceType.PATIENT_SERVICE,
                    DataSourceType.OBSERVATION_SERVICE
                ]
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="emergency_severity_validation",
                    check_name="Emergency Severity Validation",
                    safety_level=SafetyLevel.CRITICAL,
                    timeout_seconds=0.5,
                    required=True
                )
            ],
            timeout_minutes=1,
            escalation_minutes=0,  # Immediate escalation
            required_roles=["emergency_system"],
            next_steps=["parallel_emergency_gateway"]
        ))
        
        # Parallel Gateway: Simultaneous Emergency Actions
        template.add_gateway("parallel_emergency_gateway", {
            "type": "parallel",
            "name": "Parallel Emergency Actions",
            "outgoing_flows": [
                "immediate_interventions",
                "team_activation", 
                "resource_mobilization",
                "family_notification"
            ]
        })
        
        # Step 2A: Immediate Autonomous Interventions (Digital Reflex Arc - 20ms)
        template.add_step(WorkflowStep(
            step_id="immediate_interventions",
            step_name="Immediate Autonomous Interventions",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="autonomous_interventions_activity",
                activity_type=ClinicalActivityType.SYNCHRONOUS,
                timeout_seconds=2,
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="intervention_safety_check",
                    check_name="Autonomous Intervention Safety",
                    safety_level=SafetyLevel.CRITICAL,
                    timeout_seconds=0.5,
                    required=True
                )
            ],
            parallel_execution=True,
            timeout_minutes=2,
            escalation_minutes=0,
            required_roles=["emergency_system", "autonomous_agent"],
            next_steps=["intervention_monitoring"]
        ))
        
        # Step 2B: Emergency Team Activation (Parallel - 30ms)
        template.add_step(WorkflowStep(
            step_id="team_activation",
            step_name="Emergency Team Activation",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="team_activation_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=30,
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            parallel_execution=True,
            timeout_minutes=5,
            escalation_minutes=1,
            required_roles=["emergency_coordinator"],
            next_steps=["team_coordination"]
        ))
        
        # Step 2C: Resource Mobilization (Parallel - 60ms)
        template.add_step(WorkflowStep(
            step_id="resource_mobilization",
            step_name="Emergency Resource Mobilization",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="resource_mobilization_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=60,
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="detailed"
            ),
            parallel_execution=True,
            timeout_minutes=10,
            escalation_minutes=2,
            required_roles=["resource_coordinator"],
            next_steps=["resource_confirmation"]
        ))
        
        # Step 2D: Family Notification (Parallel - 120ms)
        template.add_step(WorkflowStep(
            step_id="family_notification",
            step_name="Family Emergency Notification",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="family_notification_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=120,
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="standard"
            ),
            parallel_execution=True,
            timeout_minutes=15,
            escalation_minutes=5,
            required_roles=["social_worker", "nurse"],
            next_steps=["notification_confirmation"]
        ))
        
        # Step 3: Continuous Monitoring and Adjustment (Real-time)
        template.add_step(WorkflowStep(
            step_id="intervention_monitoring",
            step_name="Continuous Intervention Monitoring",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="continuous_monitoring_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=300,  # 5 minutes continuous
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="continuous_safety_monitoring",
                    check_name="Continuous Safety Monitoring",
                    safety_level=SafetyLevel.CRITICAL,
                    timeout_seconds=10,
                    required=True
                )
            ],
            timeout_minutes=30,
            escalation_minutes=2,
            required_roles=["emergency_physician", "critical_care_nurse"],
            next_steps=["response_evaluation_gateway"]
        ))
        
        # Decision Gateway: Response Evaluation
        template.add_gateway("response_evaluation_gateway", {
            "type": "exclusive",
            "name": "Emergency Response Evaluation",
            "conditions": {
                "patient_stabilized": "patient_status == 'stable'",
                "escalate_care": "patient_status == 'critical'",
                "continue_monitoring": "patient_status == 'improving'"
            }
        })
        
        # Step 4A: Patient Stabilized - Transition to Standard Care
        template.add_step(WorkflowStep(
            step_id="transition_standard_care",
            step_name="Transition to Standard Care",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="care_transition_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=1800,  # 30 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            timeout_minutes=60,
            escalation_minutes=15,
            required_roles=["attending_physician", "charge_nurse"],
            condition_expression="patient_status == 'stable'",
            next_steps=["emergency_documentation"]
        ))
        
        # Step 4B: Escalate Care - Transfer to ICU/Specialist
        template.add_step(WorkflowStep(
            step_id="escalate_care",
            step_name="Escalate to Higher Level Care",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="care_escalation_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=900,  # 15 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            timeout_minutes=30,
            escalation_minutes=5,
            required_roles=["intensivist", "emergency_physician"],
            condition_expression="patient_status == 'critical'",
            next_steps=["transfer_coordination"]
        ))
        
        # Step 5: Emergency Documentation and Reporting
        template.add_step(WorkflowStep(
            step_id="emergency_documentation",
            step_name="Emergency Documentation and Reporting",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="emergency_documentation_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=3600,  # 1 hour
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            timeout_minutes=120,
            escalation_minutes=30,
            required_roles=["emergency_physician", "documentation_specialist"],
            next_steps=["emergency_complete"]
        ))
        
        # End Events
        template.add_end_event("emergency_complete", "Emergency Response Completed")
        template.add_end_event("patient_transferred", "Patient Transferred to Higher Care")
        
        # Global Safety Checks
        template.add_global_safety_check(SafetyCheck(
            check_id="emergency_protocol_compliance",
            check_name="Emergency Protocol Compliance",
            safety_level=SafetyLevel.CRITICAL,
            timeout_seconds=5,
            required=True
        ))
        
        # Emergency Stop Conditions
        template.add_emergency_stop_condition("patient_deceased")
        template.add_emergency_stop_condition("family_requests_comfort_care")
        template.add_emergency_stop_condition("medical_futility_declared")
        
        # Compensation Workflows
        template.add_compensation_workflow("intervention_reversal", "reverse_autonomous_interventions")
        template.add_compensation_workflow("resource_cleanup", "release_emergency_resources")
        
        # Performance Requirements
        template.set_performance_requirements({
            "max_total_time_minutes": 240,  # 4 hours max
            "critical_path_time_seconds": 300,  # 5 minutes for critical path
            "autonomous_response_time_ms": 100,  # Sub-100ms for digital reflex arc
            "team_activation_time_seconds": 180,  # 3 minutes for team activation
            "documentation_deadline_hours": 24  # 24 hours for complete documentation
        })
        
        return template.build()
    
    @staticmethod
    def get_emergency_protocols() -> Dict[str, Dict[str, Any]]:
        """
        Get specific emergency protocols for different emergency types.
        """
        return {
            "cardiac_arrest": {
                "interventions": ["cpr", "defibrillation", "epinephrine", "airway_management"],
                "team_roles": ["code_leader", "compressor", "airway_manager", "medication_nurse"],
                "time_budget_ms": 60,
                "autonomous_actions": ["chest_compressions", "aed_analysis"]
            },
            "anaphylaxis": {
                "interventions": ["epinephrine_injection", "iv_access", "steroid_administration", "airway_support"],
                "team_roles": ["emergency_physician", "allergy_nurse", "respiratory_therapist"],
                "time_budget_ms": 90,
                "autonomous_actions": ["epinephrine_autoinjector", "oxygen_delivery"]
            },
            "sepsis": {
                "interventions": ["blood_cultures", "antibiotic_administration", "fluid_resuscitation", "lactate_monitoring"],
                "team_roles": ["emergency_physician", "infectious_disease_specialist", "critical_care_nurse"],
                "time_budget_ms": 120,
                "autonomous_actions": ["sepsis_alert", "protocol_activation"]
            },
            "stroke": {
                "interventions": ["ct_scan", "thrombolytic_assessment", "neurological_monitoring", "blood_pressure_management"],
                "team_roles": ["neurologist", "emergency_physician", "stroke_nurse", "radiologist"],
                "time_budget_ms": 150,
                "autonomous_actions": ["stroke_alert", "imaging_order"]
            }
        }


# Create singleton instance
emergency_response_template = EmergencyResponseWorkflowTemplate()
