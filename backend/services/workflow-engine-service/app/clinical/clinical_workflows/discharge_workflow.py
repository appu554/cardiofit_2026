"""
Patient Discharge Workflow Template.
Implements BPMN 2.0 compliant patient discharge with medication reconciliation and safety checks.
"""
from typing import Dict, Any
from datetime import datetime

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplateBuilder, WorkflowTemplateType, CommonSafetyChecks, SafetyCheck, SafetyCheckType
)
from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, DataSourceType
)


class PatientDischargeWorkflowTemplate:
    """
    Patient discharge workflow with medication reconciliation and comprehensive safety checks.
    
    Workflow Steps:
    1. Start Event (Discharge Order)
    2. Discharge Readiness Assessment (Human)
    3. Parallel Gateway for Discharge Preparation:
       - Medication Reconciliation (Sync + Human)
       - Discharge Instructions Preparation (Human)
       - Follow-up Appointments Scheduling (Async)
       - Insurance/Billing Finalization (Async)
    4. Discharge Safety Gateway:
       - Final Medication Review
       - Discharge Criteria Validation
       - Patient Education Verification
    5. Discharge Authorization (Human)
    6. Patient Education & Handoff (Human)
    7. Discharge Confirmation (Sync)
    8. Post-Discharge Follow-up Setup (Async)
    9. End Event
    
    Safety Features:
    - Comprehensive medication reconciliation
    - Discharge criteria validation
    - Patient education verification
    - Follow-up care coordination
    - Comprehensive audit trail
    """
    
    @staticmethod
    def create_template() -> Dict[str, Any]:
        """
        Create the patient discharge workflow template.
        """
        builder = ClinicalWorkflowTemplateBuilder()
        
        # Create base template
        template = (builder
            .create_template(
                template_id="patient-discharge-workflow-v1",
                template_name="Patient Discharge Workflow",
                template_type=WorkflowTemplateType.PATIENT_DISCHARGE,
                version="1.0.0",
                description="Clinical workflow for patient discharge with medication reconciliation and safety checks"
            )
            .add_start_event("start_discharge_order")
        )
        
        # Step 1: Discharge Readiness Assessment
        template.add_human_task(
            step_id="discharge_readiness_assessment",
            step_name="Discharge Readiness Assessment",
            required_roles=["attending_physician", "hospitalist", "discharge_planner"],
            timeout_minutes=60,
            escalation_minutes=30
        )
        
        # Step 2: Parallel Gateway for Discharge Preparation
        template.add_safety_gateway(
            gateway_id="discharge_preparation_gateway",
            gateway_name="Parallel Discharge Preparation",
            safety_checks=[],  # No safety checks at gateway level
            parallel_execution=True
        )
        
        # Parallel Branch 1: Medication Reconciliation
        medication_reconciliation_activity = ClinicalActivity(
            activity_id="medication_reconciliation",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=60,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.MEDICATION_SERVICE, DataSourceType.FHIR_STORE, DataSourceType.CAE_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="reconcile_medications",
            step_name="Medication Reconciliation",
            activity=medication_reconciliation_activity,
            timeout_minutes=30,
            required_roles=["clinical_pharmacist", "nurse"]
        )
        
        # Medication Reconciliation Review (Human Task)
        template.add_human_task(
            step_id="review_medication_reconciliation",
            step_name="Review Medication Reconciliation",
            required_roles=["clinical_pharmacist", "attending_physician"],
            timeout_minutes=30,
            escalation_minutes=15
        )
        
        # Parallel Branch 2: Discharge Instructions Preparation
        template.add_human_task(
            step_id="prepare_discharge_instructions",
            step_name="Prepare Discharge Instructions",
            required_roles=["nurse", "discharge_planner"],
            timeout_minutes=45,
            escalation_minutes=20
        )
        
        # Parallel Branch 3: Follow-up Appointments Scheduling
        followup_scheduling_activity = ClinicalActivity(
            activity_id="followup_scheduling",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=180,  # 3 minutes
            safety_critical=False,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=False,  # Can proceed without follow-up scheduling
            audit_level="standard"
        )
        
        template.add_clinical_task(
            step_id="schedule_followup",
            step_name="Schedule Follow-up Appointments",
            activity=followup_scheduling_activity,
            timeout_minutes=20,
            required_roles=["scheduler", "case_manager"]
        )
        
        # Parallel Branch 4: Insurance/Billing Finalization
        billing_finalization_activity = ClinicalActivity(
            activity_id="billing_finalization",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=120,  # 2 minutes
            safety_critical=False,
            requires_clinical_context=False,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=False,
            audit_level="standard"
        )
        
        template.add_clinical_task(
            step_id="finalize_billing",
            step_name="Finalize Insurance and Billing",
            activity=billing_finalization_activity,
            timeout_minutes=15,
            required_roles=["billing_specialist", "case_manager"]
        )
        
        # Step 3: Discharge Safety Gateway
        discharge_safety_checks = [
            PatientDischargeWorkflowTemplate._create_final_medication_review_check(),
            PatientDischargeWorkflowTemplate._create_discharge_criteria_check(),
            PatientDischargeWorkflowTemplate._create_patient_education_check()
        ]
        
        template.add_safety_gateway(
            gateway_id="discharge_safety_gateway",
            gateway_name="Discharge Safety Validation",
            safety_checks=discharge_safety_checks,
            parallel_execution=True
        )
        
        # Step 4: Discharge Authorization
        template.add_human_task(
            step_id="discharge_authorization",
            step_name="Final Discharge Authorization",
            required_roles=["attending_physician", "hospitalist"],
            timeout_minutes=30,
            escalation_minutes=15
        )
        
        # Step 5: Patient Education & Handoff
        template.add_human_task(
            step_id="patient_education_handoff",
            step_name="Patient Education and Handoff",
            required_roles=["nurse", "discharge_planner"],
            timeout_minutes=60,
            escalation_minutes=30
        )
        
        # Step 6: Discharge Confirmation
        discharge_confirmation_activity = ClinicalActivity(
            activity_id="discharge_confirmation",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=15,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.PATIENT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="confirm_discharge",
            step_name="Confirm Patient Discharge",
            activity=discharge_confirmation_activity,
            timeout_minutes=10,
            required_roles=["system"]
        )
        
        # Step 7: Post-Discharge Follow-up Setup
        post_discharge_setup_activity = ClinicalActivity(
            activity_id="post_discharge_setup",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=90,
            safety_critical=False,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=False,
            audit_level="standard"
        )
        
        template.add_clinical_task(
            step_id="setup_post_discharge_followup",
            step_name="Setup Post-Discharge Follow-up",
            activity=post_discharge_setup_activity,
            timeout_minutes=15,
            required_roles=["case_manager", "system"]
        )
        
        # Add sequence flows
        (template
            .add_sequence_flow("start_discharge_order", "discharge_readiness_assessment")
            .add_sequence_flow("discharge_readiness_assessment", "discharge_preparation_gateway", "ready_for_discharge")
            .add_sequence_flow("discharge_preparation_gateway", "reconcile_medications")
            .add_sequence_flow("discharge_preparation_gateway", "prepare_discharge_instructions")
            .add_sequence_flow("discharge_preparation_gateway", "schedule_followup")
            .add_sequence_flow("discharge_preparation_gateway", "finalize_billing")
            .add_sequence_flow("reconcile_medications", "review_medication_reconciliation")
            .add_sequence_flow("review_medication_reconciliation", "discharge_safety_gateway")
            .add_sequence_flow("prepare_discharge_instructions", "discharge_safety_gateway")
            .add_sequence_flow("schedule_followup", "discharge_safety_gateway")
            .add_sequence_flow("finalize_billing", "discharge_safety_gateway")
            .add_sequence_flow("discharge_safety_gateway", "discharge_authorization", "safety_checks_passed")
            .add_sequence_flow("discharge_authorization", "patient_education_handoff", "authorized")
            .add_sequence_flow("patient_education_handoff", "confirm_discharge", "education_completed")
            .add_sequence_flow("confirm_discharge", "setup_post_discharge_followup")
            .add_sequence_flow("setup_post_discharge_followup", "patient-discharge-workflow-v1_end")
        )
        
        # Add compensation handlers
        (template
            .add_compensation_handler("medication_reconciliation_failure", "handle_medication_discrepancy")
            .add_compensation_handler("discharge_safety_failure", "cancel_discharge_process")
            .add_compensation_handler("patient_education_failure", "reschedule_discharge")
            .add_compensation_handler("discharge_cancellation", "full_discharge_rollback")
        )
        
        # Add global safety checks
        global_medication_safety = SafetyCheck(
            check_id="global_medication_safety",
            check_type=SafetyCheckType.DRUG_INTERACTION,
            description="Continuous medication safety monitoring during discharge",
            required_data_sources=[DataSourceType.CAE_SERVICE, DataSourceType.MEDICATION_SERVICE],
            timeout_seconds=10,
            critical=True
        )
        template.add_global_safety_check(global_medication_safety)
        
        # Add emergency stop conditions
        (template
            .add_emergency_stop_condition("critical_medication_interaction_detected")
            .add_emergency_stop_condition("patient_condition_deteriorated")
            .add_emergency_stop_condition("discharge_criteria_not_met")
            .add_emergency_stop_condition("patient_education_incomplete")
        )
        
        # Set SLA targets
        template.set_sla_targets({
            "total_workflow_minutes": 240,  # 4 hours max
            "readiness_assessment_minutes": 60,
            "medication_reconciliation_minutes": 60,
            "discharge_instructions_minutes": 45,
            "safety_checks_seconds": 60,
            "discharge_authorization_minutes": 30,
            "patient_education_minutes": 60
        })
        
        return template.build()
    
    @staticmethod
    def _create_final_medication_review_check() -> SafetyCheck:
        """
        Create final medication review safety check.
        """
        return SafetyCheck(
            check_id="final_medication_review",
            check_type=SafetyCheckType.DRUG_INTERACTION,
            description="Final comprehensive medication review before discharge",
            required_data_sources=[DataSourceType.CAE_SERVICE, DataSourceType.MEDICATION_SERVICE, DataSourceType.FHIR_STORE],
            timeout_seconds=30,
            critical=True,
            retry_count=2
        )
    
    @staticmethod
    def _create_discharge_criteria_check() -> SafetyCheck:
        """
        Create discharge criteria validation safety check.
        """
        return SafetyCheck(
            check_id="discharge_criteria_validation",
            check_type=SafetyCheckType.CLINICAL_CONTEXT,
            description="Validate all discharge criteria are met",
            required_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.PATIENT_SERVICE],
            timeout_seconds=20,
            critical=True,
            retry_count=2
        )
    
    @staticmethod
    def _create_patient_education_check() -> SafetyCheck:
        """
        Create patient education verification safety check.
        """
        return SafetyCheck(
            check_id="patient_education_verification",
            check_type=SafetyCheckType.CLINICAL_CONTEXT,
            description="Verify patient education and understanding",
            required_data_sources=[DataSourceType.CONTEXT_SERVICE],
            timeout_seconds=10,
            critical=True,
            retry_count=1
        )
    
    @staticmethod
    def get_bpmn_xml() -> str:
        """
        Generate BPMN 2.0 XML for the patient discharge workflow.
        """
        return """<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="PatientDischargeWorkflow"
                  targetNamespace="http://clinical-synthesis-hub.com/workflows">
  
  <bpmn:process id="patient-discharge-workflow-v1" name="Patient Discharge Workflow" isExecutable="true">
    
    <!-- Start Event -->
    <bpmn:startEvent id="start_discharge_order" name="Discharge Order">
      <bpmn:outgoing>flow_to_readiness_assessment</bpmn:outgoing>
    </bpmn:startEvent>
    
    <!-- Discharge Readiness Assessment -->
    <bpmn:userTask id="discharge_readiness_assessment" name="Discharge Readiness Assessment">
      <bpmn:incoming>flow_to_readiness_assessment</bpmn:incoming>
      <bpmn:outgoing>flow_to_preparation_gateway</bpmn:outgoing>
    </bpmn:userTask>
    
    <!-- Parallel Gateway for Discharge Preparation -->
    <bpmn:parallelGateway id="discharge_preparation_gateway" name="Parallel Discharge Preparation">
      <bpmn:incoming>flow_to_preparation_gateway</bpmn:incoming>
      <bpmn:outgoing>flow_to_medication_reconciliation</bpmn:outgoing>
      <bpmn:outgoing>flow_to_discharge_instructions</bpmn:outgoing>
      <bpmn:outgoing>flow_to_followup_scheduling</bpmn:outgoing>
      <bpmn:outgoing>flow_to_billing</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Parallel Processing Tasks -->
    <bpmn:serviceTask id="reconcile_medications" name="Medication Reconciliation">
      <bpmn:incoming>flow_to_medication_reconciliation</bpmn:incoming>
      <bpmn:outgoing>flow_to_medication_review</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:userTask id="review_medication_reconciliation" name="Review Medication Reconciliation">
      <bpmn:incoming>flow_to_medication_review</bpmn:incoming>
      <bpmn:outgoing>flow_from_medication_reconciliation</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:userTask id="prepare_discharge_instructions" name="Prepare Discharge Instructions">
      <bpmn:incoming>flow_to_discharge_instructions</bpmn:incoming>
      <bpmn:outgoing>flow_from_discharge_instructions</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:serviceTask id="schedule_followup" name="Schedule Follow-up Appointments">
      <bpmn:incoming>flow_to_followup_scheduling</bpmn:incoming>
      <bpmn:outgoing>flow_from_followup_scheduling</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="finalize_billing" name="Finalize Insurance and Billing">
      <bpmn:incoming>flow_to_billing</bpmn:incoming>
      <bpmn:outgoing>flow_from_billing</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Convergence Gateway -->
    <bpmn:parallelGateway id="preparation_convergence_gateway" name="Preparation Convergence">
      <bpmn:incoming>flow_from_medication_reconciliation</bpmn:incoming>
      <bpmn:incoming>flow_from_discharge_instructions</bpmn:incoming>
      <bpmn:incoming>flow_from_followup_scheduling</bpmn:incoming>
      <bpmn:incoming>flow_from_billing</bpmn:incoming>
      <bpmn:outgoing>flow_to_safety_gateway</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Discharge Safety Gateway -->
    <bpmn:parallelGateway id="discharge_safety_gateway" name="Discharge Safety Validation">
      <bpmn:incoming>flow_to_safety_gateway</bpmn:incoming>
      <bpmn:outgoing>flow_to_authorization</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Discharge Authorization -->
    <bpmn:userTask id="discharge_authorization" name="Final Discharge Authorization">
      <bpmn:incoming>flow_to_authorization</bpmn:incoming>
      <bpmn:outgoing>flow_to_patient_education</bpmn:outgoing>
    </bpmn:userTask>
    
    <!-- Patient Education & Handoff -->
    <bpmn:userTask id="patient_education_handoff" name="Patient Education and Handoff">
      <bpmn:incoming>flow_to_patient_education</bpmn:incoming>
      <bpmn:outgoing>flow_to_confirmation</bpmn:outgoing>
    </bpmn:userTask>
    
    <!-- Discharge Confirmation -->
    <bpmn:serviceTask id="confirm_discharge" name="Confirm Patient Discharge">
      <bpmn:incoming>flow_to_confirmation</bpmn:incoming>
      <bpmn:outgoing>flow_to_post_discharge</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Post-Discharge Follow-up Setup -->
    <bpmn:serviceTask id="setup_post_discharge_followup" name="Setup Post-Discharge Follow-up">
      <bpmn:incoming>flow_to_post_discharge</bpmn:incoming>
      <bpmn:outgoing>flow_to_end</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- End Event -->
    <bpmn:endEvent id="patient-discharge-workflow-v1_end" name="Discharge Complete">
      <bpmn:incoming>flow_to_end</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow_to_readiness_assessment" sourceRef="start_discharge_order" targetRef="discharge_readiness_assessment"/>
    <bpmn:sequenceFlow id="flow_to_preparation_gateway" sourceRef="discharge_readiness_assessment" targetRef="discharge_preparation_gateway"/>
    <bpmn:sequenceFlow id="flow_to_medication_reconciliation" sourceRef="discharge_preparation_gateway" targetRef="reconcile_medications"/>
    <bpmn:sequenceFlow id="flow_to_discharge_instructions" sourceRef="discharge_preparation_gateway" targetRef="prepare_discharge_instructions"/>
    <bpmn:sequenceFlow id="flow_to_followup_scheduling" sourceRef="discharge_preparation_gateway" targetRef="schedule_followup"/>
    <bpmn:sequenceFlow id="flow_to_billing" sourceRef="discharge_preparation_gateway" targetRef="finalize_billing"/>
    <bpmn:sequenceFlow id="flow_to_medication_review" sourceRef="reconcile_medications" targetRef="review_medication_reconciliation"/>
    <bpmn:sequenceFlow id="flow_from_medication_reconciliation" sourceRef="review_medication_reconciliation" targetRef="preparation_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_discharge_instructions" sourceRef="prepare_discharge_instructions" targetRef="preparation_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_followup_scheduling" sourceRef="schedule_followup" targetRef="preparation_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_billing" sourceRef="finalize_billing" targetRef="preparation_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_to_safety_gateway" sourceRef="preparation_convergence_gateway" targetRef="discharge_safety_gateway"/>
    <bpmn:sequenceFlow id="flow_to_authorization" sourceRef="discharge_safety_gateway" targetRef="discharge_authorization"/>
    <bpmn:sequenceFlow id="flow_to_patient_education" sourceRef="discharge_authorization" targetRef="patient_education_handoff"/>
    <bpmn:sequenceFlow id="flow_to_confirmation" sourceRef="patient_education_handoff" targetRef="confirm_discharge"/>
    <bpmn:sequenceFlow id="flow_to_post_discharge" sourceRef="confirm_discharge" targetRef="setup_post_discharge_followup"/>
    <bpmn:sequenceFlow id="flow_to_end" sourceRef="setup_post_discharge_followup" targetRef="patient-discharge-workflow-v1_end"/>
    
  </bpmn:process>
  
</bpmn:definitions>"""
    
    @staticmethod
    def get_compensation_workflows() -> Dict[str, str]:
        """
        Get compensation workflow definitions.
        """
        return {
            "handle_medication_discrepancy": """
                1. Log medication reconciliation discrepancy
                2. Alert clinical pharmacist and attending physician
                3. Hold discharge until discrepancy resolved
                4. Re-run medication reconciliation process
                5. Create medication safety incident report
            """,
            "cancel_discharge_process": """
                1. Stop all discharge preparation activities
                2. Cancel scheduled follow-up appointments
                3. Hold billing finalization
                4. Notify clinical team of discharge cancellation
                5. Update patient status to continued care
                6. Create discharge cancellation audit trail
            """,
            "reschedule_discharge": """
                1. Pause discharge process
                2. Schedule additional patient education session
                3. Notify family/caregivers of delay
                4. Update discharge timeline
                5. Reassess discharge readiness
            """,
            "full_discharge_rollback": """
                1. Cancel discharge confirmation
                2. Reverse all discharge preparations
                3. Cancel follow-up appointments
                4. Reverse billing finalizations
                5. Restore patient to active inpatient status
                6. Notify all involved parties
                7. Create comprehensive rollback audit trail
            """
        }
