"""
Clinical Assessment Workflow Template.
Implements BPMN 2.0 compliant clinical assessment workflow with comprehensive evaluation protocols.
"""
from typing import Dict, Any, List
from datetime import datetime, timedelta

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplate, WorkflowTemplateType, WorkflowStep, SafetyCheck, SafetyLevel
)
from app.models.clinical_activity_models import ClinicalActivity, ClinicalActivityType, DataSourceType
from app.clinical.clinical_workflows.workflow_template_builder import ClinicalWorkflowTemplateBuilder


class ClinicalAssessmentWorkflowTemplate:
    """
    Clinical Assessment Workflow Template.
    
    Implements comprehensive clinical assessment protocols with:
    - Structured clinical evaluation
    - Evidence-based assessment tools
    - Risk stratification
    - Clinical decision support integration
    - Quality assurance checkpoints
    
    Assessment Types Supported:
    - Initial patient assessment
    - Focused clinical evaluation
    - Risk assessment
    - Discharge readiness evaluation
    - Specialty consultation assessment
    """
    
    @staticmethod
    def create_template() -> Dict[str, Any]:
        """
        Create the clinical assessment workflow template.
        """
        builder = ClinicalWorkflowTemplateBuilder()
        
        # Create base template
        template = (builder
            .create_template(
                template_id="clinical-assessment-workflow-v1",
                template_name="Clinical Assessment Workflow",
                template_type=WorkflowTemplateType.CLINICAL_ASSESSMENT,
                version="1.0.0",
                description="Comprehensive clinical assessment workflow with structured evaluation protocols"
            )
            .add_start_event("assessment_request_received")
        )
        
        # Step 1: Assessment Preparation and Context Gathering
        template.add_step(WorkflowStep(
            step_id="assessment_preparation",
            step_name="Assessment Preparation and Context Gathering",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="assessment_prep_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=300,  # 5 minutes
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="detailed",
                approved_data_sources=[
                    DataSourceType.FHIR_STORE,
                    DataSourceType.PATIENT_SERVICE,
                    DataSourceType.OBSERVATION_SERVICE,
                    DataSourceType.MEDICATION_SERVICE,
                    DataSourceType.CONDITION_SERVICE
                ]
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="patient_identity_verification",
                    check_name="Patient Identity Verification",
                    safety_level=SafetyLevel.HIGH,
                    timeout_seconds=30,
                    required=True
                )
            ],
            timeout_minutes=10,
            escalation_minutes=5,
            required_roles=["nurse", "clinical_assistant"],
            next_steps=["assessment_type_gateway"]
        ))
        
        # Decision Gateway: Assessment Type Selection
        template.add_gateway("assessment_type_gateway", {
            "type": "exclusive",
            "name": "Assessment Type Selection",
            "conditions": {
                "initial_assessment": "assessment_type == 'initial'",
                "focused_assessment": "assessment_type == 'focused'",
                "risk_assessment": "assessment_type == 'risk'",
                "discharge_assessment": "assessment_type == 'discharge'"
            }
        })
        
        # Step 2A: Initial Comprehensive Assessment
        template.add_step(WorkflowStep(
            step_id="initial_assessment",
            step_name="Initial Comprehensive Assessment",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="initial_assessment_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=3600,  # 1 hour
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="comprehensive_assessment_completeness",
                    check_name="Comprehensive Assessment Completeness",
                    safety_level=SafetyLevel.HIGH,
                    timeout_seconds=60,
                    required=True
                )
            ],
            timeout_minutes=90,
            escalation_minutes=30,
            required_roles=["physician", "nurse_practitioner"],
            condition_expression="assessment_type == 'initial'",
            next_steps=["clinical_reasoning"]
        ))
        
        # Step 2B: Focused Clinical Assessment
        template.add_step(WorkflowStep(
            step_id="focused_assessment",
            step_name="Focused Clinical Assessment",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="focused_assessment_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=1800,  # 30 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="detailed"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="focused_assessment_adequacy",
                    check_name="Focused Assessment Adequacy",
                    safety_level=SafetyLevel.MEDIUM,
                    timeout_seconds=30,
                    required=True
                )
            ],
            timeout_minutes=45,
            escalation_minutes=15,
            required_roles=["physician", "nurse_practitioner", "physician_assistant"],
            condition_expression="assessment_type == 'focused'",
            next_steps=["clinical_reasoning"]
        ))
        
        # Step 2C: Risk Assessment and Stratification
        template.add_step(WorkflowStep(
            step_id="risk_assessment",
            step_name="Risk Assessment and Stratification",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="risk_assessment_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=900,  # 15 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="risk_stratification_validation",
                    check_name="Risk Stratification Validation",
                    safety_level=SafetyLevel.CRITICAL,
                    timeout_seconds=60,
                    required=True
                )
            ],
            timeout_minutes=30,
            escalation_minutes=10,
            required_roles=["physician", "risk_assessment_specialist"],
            condition_expression="assessment_type == 'risk'",
            next_steps=["risk_mitigation_planning"]
        ))
        
        # Step 3: Clinical Reasoning and Decision Support
        template.add_step(WorkflowStep(
            step_id="clinical_reasoning",
            step_name="Clinical Reasoning and Decision Support",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="clinical_reasoning_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=600,  # 10 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="clinical_decision_support_check",
                    check_name="Clinical Decision Support Validation",
                    safety_level=SafetyLevel.HIGH,
                    timeout_seconds=30,
                    required=True
                )
            ],
            timeout_minutes=20,
            escalation_minutes=5,
            required_roles=["physician"],
            next_steps=["assessment_documentation"]
        ))
        
        # Step 4: Risk Mitigation Planning (for risk assessments)
        template.add_step(WorkflowStep(
            step_id="risk_mitigation_planning",
            step_name="Risk Mitigation Planning",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="risk_mitigation_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=1200,  # 20 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="mitigation_plan_adequacy",
                    check_name="Risk Mitigation Plan Adequacy",
                    safety_level=SafetyLevel.HIGH,
                    timeout_seconds=60,
                    required=True
                )
            ],
            timeout_minutes=30,
            escalation_minutes=10,
            required_roles=["physician", "risk_manager"],
            next_steps=["assessment_documentation"]
        ))
        
        # Step 5: Assessment Documentation and Communication
        template.add_step(WorkflowStep(
            step_id="assessment_documentation",
            step_name="Assessment Documentation and Communication",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="assessment_documentation_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=1800,  # 30 minutes
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="documentation_completeness_check",
                    check_name="Documentation Completeness Check",
                    safety_level=SafetyLevel.MEDIUM,
                    timeout_seconds=60,
                    required=True
                )
            ],
            timeout_minutes=45,
            escalation_minutes=15,
            required_roles=["physician", "nurse"],
            next_steps=["quality_review_gateway"]
        ))
        
        # Decision Gateway: Quality Review Required
        template.add_gateway("quality_review_gateway", {
            "type": "exclusive",
            "name": "Quality Review Decision",
            "conditions": {
                "requires_review": "assessment_complexity == 'high' OR risk_level == 'critical'",
                "no_review_needed": "assessment_complexity == 'low' AND risk_level != 'critical'"
            }
        })
        
        # Step 6: Quality Assurance Review (if required)
        template.add_step(WorkflowStep(
            step_id="quality_review",
            step_name="Quality Assurance Review",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="quality_review_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=1200,  # 20 minutes
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive"
            ),
            safety_checks=[
                SafetyCheck(
                    check_id="quality_standards_compliance",
                    check_name="Quality Standards Compliance",
                    safety_level=SafetyLevel.HIGH,
                    timeout_seconds=60,
                    required=True
                )
            ],
            timeout_minutes=30,
            escalation_minutes=10,
            required_roles=["senior_physician", "quality_assurance_specialist"],
            condition_expression="assessment_complexity == 'high' OR risk_level == 'critical'",
            next_steps=["assessment_complete"]
        ))
        
        # Step 7: Assessment Follow-up Planning
        template.add_step(WorkflowStep(
            step_id="followup_planning",
            step_name="Assessment Follow-up Planning",
            step_type="task",
            activity=ClinicalActivity(
                activity_id="followup_planning_activity",
                activity_type=ClinicalActivityType.HUMAN,
                timeout_seconds=600,  # 10 minutes
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="standard"
            ),
            timeout_minutes=15,
            escalation_minutes=5,
            required_roles=["physician", "care_coordinator"],
            next_steps=["assessment_complete"]
        ))
        
        # End Events
        template.add_end_event("assessment_complete", "Clinical Assessment Completed")
        template.add_end_event("assessment_escalated", "Assessment Escalated for Review")
        
        # Global Safety Checks
        template.add_global_safety_check(SafetyCheck(
            check_id="clinical_assessment_standards",
            check_name="Clinical Assessment Standards Compliance",
            safety_level=SafetyLevel.HIGH,
            timeout_seconds=30,
            required=True
        ))
        
        # Emergency Stop Conditions
        template.add_emergency_stop_condition("patient_condition_deteriorated")
        template.add_emergency_stop_condition("assessment_no_longer_relevant")
        
        # Compensation Workflows
        template.add_compensation_workflow("assessment_revision", "revise_incomplete_assessment")
        template.add_compensation_workflow("documentation_correction", "correct_assessment_documentation")
        
        # Performance Requirements
        template.set_performance_requirements({
            "max_initial_assessment_minutes": 90,
            "max_focused_assessment_minutes": 45,
            "max_risk_assessment_minutes": 30,
            "documentation_deadline_hours": 24,
            "quality_review_sla_hours": 48
        })
        
        return template.build()
    
    @staticmethod
    def get_assessment_tools() -> Dict[str, Dict[str, Any]]:
        """
        Get standardized assessment tools and scoring systems.
        """
        return {
            "pain_assessment": {
                "tools": ["numeric_rating_scale", "wong_baker_faces", "behavioral_pain_scale"],
                "frequency": "every_4_hours",
                "documentation_required": True
            },
            "fall_risk_assessment": {
                "tools": ["morse_fall_scale", "hendrich_fall_risk_model"],
                "frequency": "admission_and_daily",
                "high_risk_threshold": 45
            },
            "pressure_ulcer_risk": {
                "tools": ["braden_scale", "norton_scale"],
                "frequency": "admission_and_weekly",
                "high_risk_threshold": 18
            },
            "delirium_screening": {
                "tools": ["cam_icu", "3d_cam", "dos"],
                "frequency": "daily_for_high_risk",
                "positive_threshold": 1
            },
            "nutritional_assessment": {
                "tools": ["mini_nutritional_assessment", "malnutrition_screening_tool"],
                "frequency": "admission_and_weekly",
                "intervention_threshold": "moderate_risk"
            }
        }


# Create singleton instance
clinical_assessment_template = ClinicalAssessmentWorkflowTemplate()
