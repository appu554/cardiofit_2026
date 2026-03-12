"""
Patient Admission Workflow Template.
Implements BPMN 2.0 compliant patient admission with parallel processing and safety checks.
"""
from typing import Dict, Any
from datetime import datetime

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplateBuilder, WorkflowTemplateType, CommonSafetyChecks, SafetyCheck, SafetyCheckType
)
from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, DataSourceType
)


class PatientAdmissionWorkflowTemplate:
    """
    Patient admission workflow with parallel processing and comprehensive safety checks.
    
    Workflow Steps:
    1. Start Event (Admission Request)
    2. Patient Identity Verification (Sync)
    3. Parallel Gateway for Admission Processing:
       - Clinical Assessment (Human)
       - Insurance Verification (Async)
       - Bed Assignment (Async)
       - Medical History Review (Sync)
    4. Admission Safety Gateway:
       - Allergy Check
       - Current Medications Review
       - Clinical Contraindications
    5. Admission Authorization (Human)
    6. Admission Confirmation (Sync)
    7. Care Team Assignment (Async)
    8. End Event
    
    Safety Features:
    - Patient identity verification
    - Parallel processing for efficiency
    - Clinical safety validations
    - Real-time bed management
    - Comprehensive audit trail
    """
    
    @staticmethod
    def create_template() -> Dict[str, Any]:
        """
        Create the patient admission workflow template.
        """
        builder = ClinicalWorkflowTemplateBuilder()
        
        # Create base template
        template = (builder
            .create_template(
                template_id="patient-admission-workflow-v1",
                template_name="Patient Admission Workflow",
                template_type=WorkflowTemplateType.PATIENT_ADMISSION,
                version="1.0.0",
                description="Clinical workflow for patient admission with parallel processing and safety checks"
            )
            .add_start_event("start_admission_request")
        )
        
        # Step 1: Patient Identity Verification
        identity_verification_activity = ClinicalActivity(
            activity_id="patient_identity_verification",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=10,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.PATIENT_SERVICE, DataSourceType.FHIR_STORE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="verify_patient_identity",
            step_name="Verify Patient Identity",
            activity=identity_verification_activity,
            timeout_minutes=10,
            required_roles=["registration_clerk", "nurse"]
        )
        
        # Step 2: Parallel Gateway for Admission Processing
        template.add_safety_gateway(
            gateway_id="admission_processing_gateway",
            gateway_name="Parallel Admission Processing",
            safety_checks=[],  # No safety checks at gateway level
            parallel_execution=True
        )
        
        # Parallel Branch 1: Clinical Assessment (Human Task)
        template.add_human_task(
            step_id="clinical_assessment",
            step_name="Initial Clinical Assessment",
            required_roles=["doctor", "nurse_practitioner"],
            timeout_minutes=60,
            escalation_minutes=30
        )
        
        # Parallel Branch 2: Insurance Verification
        insurance_verification_activity = ClinicalActivity(
            activity_id="insurance_verification",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=120,  # 2 minutes
            safety_critical=False,
            requires_clinical_context=False,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=False,  # Can proceed without insurance verification
            audit_level="standard"
        )
        
        template.add_clinical_task(
            step_id="verify_insurance",
            step_name="Verify Insurance Coverage",
            activity=insurance_verification_activity,
            timeout_minutes=15,
            required_roles=["system", "registration_clerk"]
        )
        
        # Parallel Branch 3: Bed Assignment
        bed_assignment_activity = ClinicalActivity(
            activity_id="bed_assignment",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=60,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="detailed"
        )
        
        template.add_clinical_task(
            step_id="assign_bed",
            step_name="Assign Patient Bed",
            activity=bed_assignment_activity,
            timeout_minutes=20,
            required_roles=["bed_manager", "charge_nurse"]
        )
        
        # Parallel Branch 4: Medical History Review
        medical_history_activity = ClinicalActivity(
            activity_id="medical_history_review",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=30,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.PATIENT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="review_medical_history",
            step_name="Review Medical History",
            activity=medical_history_activity,
            timeout_minutes=15,
            required_roles=["system"]
        )
        
        # Step 3: Admission Safety Gateway
        admission_safety_checks = [
            CommonSafetyChecks.allergy_check(),
            PatientAdmissionWorkflowTemplate._create_medication_review_check(),
            PatientAdmissionWorkflowTemplate._create_clinical_contraindication_check()
        ]
        
        template.add_safety_gateway(
            gateway_id="admission_safety_gateway",
            gateway_name="Admission Safety Validation",
            safety_checks=admission_safety_checks,
            parallel_execution=True
        )
        
        # Step 4: Admission Authorization (Human Task)
        template.add_human_task(
            step_id="admission_authorization",
            step_name="Admission Authorization",
            required_roles=["attending_physician", "hospitalist"],
            timeout_minutes=30,
            escalation_minutes=15
        )
        
        # Step 5: Admission Confirmation
        admission_confirmation_activity = ClinicalActivity(
            activity_id="admission_confirmation",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=10,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.PATIENT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="confirm_admission",
            step_name="Confirm Patient Admission",
            activity=admission_confirmation_activity,
            timeout_minutes=10,
            required_roles=["system"]
        )
        
        # Step 6: Care Team Assignment
        care_team_assignment_activity = ClinicalActivity(
            activity_id="care_team_assignment",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=60,
            safety_critical=False,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.CONTEXT_SERVICE],
            real_data_only=True,
            fail_on_unavailable=False,
            audit_level="standard"
        )
        
        template.add_clinical_task(
            step_id="assign_care_team",
            step_name="Assign Care Team",
            activity=care_team_assignment_activity,
            timeout_minutes=15,
            required_roles=["charge_nurse", "case_manager"]
        )
        
        # Add sequence flows
        (template
            .add_sequence_flow("start_admission_request", "verify_patient_identity")
            .add_sequence_flow("verify_patient_identity", "admission_processing_gateway")
            .add_sequence_flow("admission_processing_gateway", "clinical_assessment")
            .add_sequence_flow("admission_processing_gateway", "verify_insurance")
            .add_sequence_flow("admission_processing_gateway", "assign_bed")
            .add_sequence_flow("admission_processing_gateway", "review_medical_history")
            .add_sequence_flow("clinical_assessment", "admission_safety_gateway")
            .add_sequence_flow("verify_insurance", "admission_safety_gateway")
            .add_sequence_flow("assign_bed", "admission_safety_gateway")
            .add_sequence_flow("review_medical_history", "admission_safety_gateway")
            .add_sequence_flow("admission_safety_gateway", "admission_authorization", "safety_checks_passed")
            .add_sequence_flow("admission_authorization", "confirm_admission", "authorized")
            .add_sequence_flow("confirm_admission", "assign_care_team")
            .add_sequence_flow("assign_care_team", "patient-admission-workflow-v1_end")
        )
        
        # Add compensation handlers
        (template
            .add_compensation_handler("identity_verification_failure", "handle_identity_failure")
            .add_compensation_handler("bed_assignment_failure", "release_bed_and_notify")
            .add_compensation_handler("admission_safety_failure", "cancel_admission_process")
            .add_compensation_handler("admission_cancellation", "full_admission_rollback")
        )
        
        # Add global safety checks
        global_identity_check = SafetyCheck(
            check_id="global_patient_identity",
            check_type=SafetyCheckType.CLINICAL_CONTEXT,
            description="Continuous patient identity verification throughout admission",
            required_data_sources=[DataSourceType.PATIENT_SERVICE],
            timeout_seconds=5,
            critical=True
        )
        template.add_global_safety_check(global_identity_check)
        
        # Add emergency stop conditions
        (template
            .add_emergency_stop_condition("patient_identity_mismatch")
            .add_emergency_stop_condition("critical_allergy_detected")
            .add_emergency_stop_condition("bed_capacity_exceeded")
            .add_emergency_stop_condition("insurance_authorization_denied")
        )
        
        # Set SLA targets
        template.set_sla_targets({
            "total_workflow_minutes": 120,  # 2 hours max
            "identity_verification_minutes": 10,
            "clinical_assessment_minutes": 60,
            "bed_assignment_minutes": 20,
            "safety_checks_seconds": 45,
            "admission_authorization_minutes": 30
        })
        
        return template.build()
    
    @staticmethod
    def _create_medication_review_check() -> SafetyCheck:
        """
        Create medication review safety check.
        """
        return SafetyCheck(
            check_id="current_medications_review",
            check_type=SafetyCheckType.CLINICAL_CONTEXT,
            description="Review current medications for admission compatibility",
            required_data_sources=[DataSourceType.MEDICATION_SERVICE, DataSourceType.FHIR_STORE],
            timeout_seconds=20,
            critical=True,
            retry_count=2
        )
    
    @staticmethod
    def _create_clinical_contraindication_check() -> SafetyCheck:
        """
        Create clinical contraindication safety check.
        """
        return SafetyCheck(
            check_id="clinical_contraindications",
            check_type=SafetyCheckType.CONTRAINDICATION,
            description="Check for clinical contraindications to admission",
            required_data_sources=[DataSourceType.CAE_SERVICE, DataSourceType.FHIR_STORE],
            timeout_seconds=15,
            critical=True,
            retry_count=2
        )
    
    @staticmethod
    def get_bpmn_xml() -> str:
        """
        Generate BPMN 2.0 XML for the patient admission workflow.
        """
        return """<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="PatientAdmissionWorkflow"
                  targetNamespace="http://clinical-synthesis-hub.com/workflows">
  
  <bpmn:process id="patient-admission-workflow-v1" name="Patient Admission Workflow" isExecutable="true">
    
    <!-- Start Event -->
    <bpmn:startEvent id="start_admission_request" name="Admission Request">
      <bpmn:outgoing>flow_to_identity_verification</bpmn:outgoing>
    </bpmn:startEvent>
    
    <!-- Patient Identity Verification -->
    <bpmn:serviceTask id="verify_patient_identity" name="Verify Patient Identity">
      <bpmn:incoming>flow_to_identity_verification</bpmn:incoming>
      <bpmn:outgoing>flow_to_processing_gateway</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Parallel Gateway for Admission Processing -->
    <bpmn:parallelGateway id="admission_processing_gateway" name="Parallel Admission Processing">
      <bpmn:incoming>flow_to_processing_gateway</bpmn:incoming>
      <bpmn:outgoing>flow_to_clinical_assessment</bpmn:outgoing>
      <bpmn:outgoing>flow_to_insurance_verification</bpmn:outgoing>
      <bpmn:outgoing>flow_to_bed_assignment</bpmn:outgoing>
      <bpmn:outgoing>flow_to_medical_history</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Parallel Processing Tasks -->
    <bpmn:userTask id="clinical_assessment" name="Initial Clinical Assessment">
      <bpmn:incoming>flow_to_clinical_assessment</bpmn:incoming>
      <bpmn:outgoing>flow_from_clinical_assessment</bpmn:outgoing>
    </bpmn:userTask>
    
    <bpmn:serviceTask id="verify_insurance" name="Verify Insurance Coverage">
      <bpmn:incoming>flow_to_insurance_verification</bpmn:incoming>
      <bpmn:outgoing>flow_from_insurance_verification</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="assign_bed" name="Assign Patient Bed">
      <bpmn:incoming>flow_to_bed_assignment</bpmn:incoming>
      <bpmn:outgoing>flow_from_bed_assignment</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="review_medical_history" name="Review Medical History">
      <bpmn:incoming>flow_to_medical_history</bpmn:incoming>
      <bpmn:outgoing>flow_from_medical_history</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Convergence Gateway -->
    <bpmn:parallelGateway id="processing_convergence_gateway" name="Processing Convergence">
      <bpmn:incoming>flow_from_clinical_assessment</bpmn:incoming>
      <bpmn:incoming>flow_from_insurance_verification</bpmn:incoming>
      <bpmn:incoming>flow_from_bed_assignment</bpmn:incoming>
      <bpmn:incoming>flow_from_medical_history</bpmn:incoming>
      <bpmn:outgoing>flow_to_safety_gateway</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Admission Safety Gateway -->
    <bpmn:parallelGateway id="admission_safety_gateway" name="Admission Safety Validation">
      <bpmn:incoming>flow_to_safety_gateway</bpmn:incoming>
      <bpmn:outgoing>flow_to_authorization</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Admission Authorization -->
    <bpmn:userTask id="admission_authorization" name="Admission Authorization">
      <bpmn:incoming>flow_to_authorization</bpmn:incoming>
      <bpmn:outgoing>flow_to_confirmation</bpmn:outgoing>
    </bpmn:userTask>
    
    <!-- Admission Confirmation -->
    <bpmn:serviceTask id="confirm_admission" name="Confirm Patient Admission">
      <bpmn:incoming>flow_to_confirmation</bpmn:incoming>
      <bpmn:outgoing>flow_to_care_team</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Care Team Assignment -->
    <bpmn:serviceTask id="assign_care_team" name="Assign Care Team">
      <bpmn:incoming>flow_to_care_team</bpmn:incoming>
      <bpmn:outgoing>flow_to_end</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- End Event -->
    <bpmn:endEvent id="patient-admission-workflow-v1_end" name="Admission Complete">
      <bpmn:incoming>flow_to_end</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow_to_identity_verification" sourceRef="start_admission_request" targetRef="verify_patient_identity"/>
    <bpmn:sequenceFlow id="flow_to_processing_gateway" sourceRef="verify_patient_identity" targetRef="admission_processing_gateway"/>
    <bpmn:sequenceFlow id="flow_to_clinical_assessment" sourceRef="admission_processing_gateway" targetRef="clinical_assessment"/>
    <bpmn:sequenceFlow id="flow_to_insurance_verification" sourceRef="admission_processing_gateway" targetRef="verify_insurance"/>
    <bpmn:sequenceFlow id="flow_to_bed_assignment" sourceRef="admission_processing_gateway" targetRef="assign_bed"/>
    <bpmn:sequenceFlow id="flow_to_medical_history" sourceRef="admission_processing_gateway" targetRef="review_medical_history"/>
    <bpmn:sequenceFlow id="flow_from_clinical_assessment" sourceRef="clinical_assessment" targetRef="processing_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_insurance_verification" sourceRef="verify_insurance" targetRef="processing_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_bed_assignment" sourceRef="assign_bed" targetRef="processing_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_medical_history" sourceRef="review_medical_history" targetRef="processing_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_to_safety_gateway" sourceRef="processing_convergence_gateway" targetRef="admission_safety_gateway"/>
    <bpmn:sequenceFlow id="flow_to_authorization" sourceRef="admission_safety_gateway" targetRef="admission_authorization"/>
    <bpmn:sequenceFlow id="flow_to_confirmation" sourceRef="admission_authorization" targetRef="confirm_admission"/>
    <bpmn:sequenceFlow id="flow_to_care_team" sourceRef="confirm_admission" targetRef="assign_care_team"/>
    <bpmn:sequenceFlow id="flow_to_end" sourceRef="assign_care_team" targetRef="patient-admission-workflow-v1_end"/>
    
  </bpmn:process>
  
</bpmn:definitions>"""
    
    @staticmethod
    def get_compensation_workflows() -> Dict[str, str]:
        """
        Get compensation workflow definitions.
        """
        return {
            "handle_identity_failure": """
                1. Log identity verification failure
                2. Request additional identification
                3. Escalate to registration supervisor
                4. Create security incident report
            """,
            "release_bed_and_notify": """
                1. Release assigned bed back to inventory
                2. Notify bed management system
                3. Update bed availability status
                4. Log bed assignment reversal
            """,
            "cancel_admission_process": """
                1. Stop all parallel admission processes
                2. Release allocated resources (bed, staff)
                3. Notify clinical team of cancellation
                4. Create admission cancellation record
                5. Update patient status
            """,
            "full_admission_rollback": """
                1. Cancel admission confirmation
                2. Release bed assignment
                3. Cancel care team assignments
                4. Reverse insurance authorizations
                5. Create comprehensive rollback audit trail
                6. Notify all involved parties
            """
        }
