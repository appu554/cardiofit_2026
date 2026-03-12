"""
Medication Ordering Workflow Template.
Implements BPMN 2.0 compliant medication ordering with comprehensive safety checks.
"""
from typing import Dict, Any
from datetime import datetime

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplateBuilder, WorkflowTemplateType, CommonSafetyChecks
)
from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, DataSourceType
)


class MedicationOrderingWorkflowTemplate:
    """
    Medication ordering workflow with comprehensive safety mechanisms.
    
    Workflow Steps:
    1. Start Event
    2. Medication Harmonization (Sync)
    3. Safety Gateway (Parallel Safety Checks)
       - Drug Interaction Check
       - Allergy Check  
       - Dosage Validation
       - Provider Authorization
    4. Clinical Review (Human Task)
    5. Order Submission (Async)
    6. Confirmation & Audit (Sync)
    7. End Event
    
    Safety Features:
    - Real data validation at each step
    - Parallel safety checks for performance
    - Automatic compensation on failures
    - Clinical override capabilities
    - Comprehensive audit trail
    """
    
    @staticmethod
    def create_template() -> Dict[str, Any]:
        """
        Create the medication ordering workflow template.
        """
        builder = ClinicalWorkflowTemplateBuilder()
        
        # Create base template
        template = (builder
            .create_template(
                template_id="medication-ordering-workflow-v1",
                template_name="Medication Ordering Workflow",
                template_type=WorkflowTemplateType.MEDICATION_ORDERING,
                version="1.0.0",
                description="Clinical workflow for safe medication ordering with comprehensive safety checks"
            )
            .add_start_event("start_medication_order")
        )
        
        # Step 1: Medication Harmonization
        harmonization_activity = ClinicalActivity(
            activity_id="medication_harmonization",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=5,
            safety_critical=False,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.HARMONIZATION_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="detailed"
        )
        
        template.add_clinical_task(
            step_id="harmonize_medication",
            step_name="Harmonize Medication Data",
            activity=harmonization_activity,
            timeout_minutes=5,
            required_roles=["system"]
        )
        
        # Step 2: Safety Gateway with Parallel Safety Checks
        safety_checks = [
            CommonSafetyChecks.drug_interaction_check(),
            CommonSafetyChecks.allergy_check(),
            CommonSafetyChecks.dosage_validation(),
            CommonSafetyChecks.provider_authorization()
        ]
        
        template.add_safety_gateway(
            gateway_id="safety_validation_gateway",
            gateway_name="Parallel Safety Validation",
            safety_checks=safety_checks,
            parallel_execution=True
        )
        
        # Step 3: Clinical Review (Human Task)
        template.add_human_task(
            step_id="clinical_review",
            step_name="Clinical Review and Approval",
            required_roles=["doctor", "clinical_pharmacist"],
            timeout_minutes=120,  # 2 hours
            escalation_minutes=60  # 1 hour
        )
        
        # Step 4: Order Submission
        order_submission_activity = ClinicalActivity(
            activity_id="order_submission",
            activity_type=ClinicalActivityType.ASYNCHRONOUS,
            timeout_seconds=30,
            safety_critical=True,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.MEDICATION_SERVICE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="submit_order",
            step_name="Submit Medication Order",
            activity=order_submission_activity,
            timeout_minutes=10,
            required_roles=["system"]
        )
        
        # Step 5: Confirmation & Audit
        confirmation_activity = ClinicalActivity(
            activity_id="order_confirmation",
            activity_type=ClinicalActivityType.SYNCHRONOUS,
            timeout_seconds=5,
            safety_critical=False,
            requires_clinical_context=True,
            approved_data_sources=[DataSourceType.FHIR_STORE],
            real_data_only=True,
            fail_on_unavailable=True,
            audit_level="comprehensive"
        )
        
        template.add_clinical_task(
            step_id="confirm_order",
            step_name="Confirm Order and Create Audit Trail",
            activity=confirmation_activity,
            timeout_minutes=5,
            required_roles=["system"]
        )
        
        # Add sequence flows
        (template
            .add_sequence_flow("start_medication_order", "harmonize_medication")
            .add_sequence_flow("harmonize_medication", "safety_validation_gateway")
            .add_sequence_flow("safety_validation_gateway", "clinical_review", "safety_checks_passed")
            .add_sequence_flow("clinical_review", "submit_order", "approved")
            .add_sequence_flow("submit_order", "confirm_order")
            .add_sequence_flow("confirm_order", "medication-ordering-workflow-v1_end")
        )
        
        # Add compensation handlers
        (template
            .add_compensation_handler("harmonization_compensation", "reverse_harmonization")
            .add_compensation_handler("safety_failure_compensation", "handle_safety_failure")
            .add_compensation_handler("order_failure_compensation", "cancel_order_and_notify")
        )
        
        # Add global safety checks
        global_safety = CommonSafetyChecks.drug_interaction_check()
        global_safety.check_id = "global_drug_interaction"
        global_safety.description = "Global drug interaction monitoring throughout workflow"
        template.add_global_safety_check(global_safety)
        
        # Add emergency stop conditions
        (template
            .add_emergency_stop_condition("critical_drug_interaction_detected")
            .add_emergency_stop_condition("patient_allergy_conflict")
            .add_emergency_stop_condition("provider_authorization_revoked")
        )
        
        # Set SLA targets
        template.set_sla_targets({
            "total_workflow_minutes": 180,  # 3 hours max
            "safety_checks_seconds": 30,
            "clinical_review_minutes": 120,
            "order_submission_minutes": 10
        })
        
        return template.build()
    
    @staticmethod
    def get_bpmn_xml() -> str:
        """
        Generate BPMN 2.0 XML for the medication ordering workflow.
        """
        return """<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
                  id="MedicationOrderingWorkflow"
                  targetNamespace="http://clinical-synthesis-hub.com/workflows">
  
  <bpmn:process id="medication-ordering-workflow-v1" name="Medication Ordering Workflow" isExecutable="true">
    
    <!-- Start Event -->
    <bpmn:startEvent id="start_medication_order" name="Start Medication Order">
      <bpmn:outgoing>flow_to_harmonization</bpmn:outgoing>
    </bpmn:startEvent>
    
    <!-- Medication Harmonization Task -->
    <bpmn:serviceTask id="harmonize_medication" name="Harmonize Medication Data">
      <bpmn:incoming>flow_to_harmonization</bpmn:incoming>
      <bpmn:outgoing>flow_to_safety_gateway</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Safety Validation Gateway (Parallel) -->
    <bpmn:parallelGateway id="safety_validation_gateway" name="Safety Validation Gateway">
      <bpmn:incoming>flow_to_safety_gateway</bpmn:incoming>
      <bpmn:outgoing>flow_to_drug_interaction</bpmn:outgoing>
      <bpmn:outgoing>flow_to_allergy_check</bpmn:outgoing>
      <bpmn:outgoing>flow_to_dosage_validation</bpmn:outgoing>
      <bpmn:outgoing>flow_to_provider_auth</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Parallel Safety Checks -->
    <bpmn:serviceTask id="drug_interaction_check" name="Drug Interaction Check">
      <bpmn:incoming>flow_to_drug_interaction</bpmn:incoming>
      <bpmn:outgoing>flow_from_drug_interaction</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="allergy_check" name="Allergy Check">
      <bpmn:incoming>flow_to_allergy_check</bpmn:incoming>
      <bpmn:outgoing>flow_from_allergy_check</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="dosage_validation" name="Dosage Validation">
      <bpmn:incoming>flow_to_dosage_validation</bpmn:incoming>
      <bpmn:outgoing>flow_from_dosage_validation</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <bpmn:serviceTask id="provider_authorization" name="Provider Authorization">
      <bpmn:incoming>flow_to_provider_auth</bpmn:incoming>
      <bpmn:outgoing>flow_from_provider_auth</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Safety Convergence Gateway -->
    <bpmn:parallelGateway id="safety_convergence_gateway" name="Safety Convergence">
      <bpmn:incoming>flow_from_drug_interaction</bpmn:incoming>
      <bpmn:incoming>flow_from_allergy_check</bpmn:incoming>
      <bpmn:incoming>flow_from_dosage_validation</bpmn:incoming>
      <bpmn:incoming>flow_from_provider_auth</bpmn:incoming>
      <bpmn:outgoing>flow_to_clinical_review</bpmn:outgoing>
    </bpmn:parallelGateway>
    
    <!-- Clinical Review Human Task -->
    <bpmn:userTask id="clinical_review" name="Clinical Review and Approval">
      <bpmn:incoming>flow_to_clinical_review</bpmn:incoming>
      <bpmn:outgoing>flow_to_submit_order</bpmn:outgoing>
    </bpmn:userTask>
    
    <!-- Order Submission -->
    <bpmn:serviceTask id="submit_order" name="Submit Medication Order">
      <bpmn:incoming>flow_to_submit_order</bpmn:incoming>
      <bpmn:outgoing>flow_to_confirm_order</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- Order Confirmation -->
    <bpmn:serviceTask id="confirm_order" name="Confirm Order and Create Audit Trail">
      <bpmn:incoming>flow_to_confirm_order</bpmn:incoming>
      <bpmn:outgoing>flow_to_end</bpmn:outgoing>
    </bpmn:serviceTask>
    
    <!-- End Event -->
    <bpmn:endEvent id="medication-ordering-workflow-v1_end" name="Order Complete">
      <bpmn:incoming>flow_to_end</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="flow_to_harmonization" sourceRef="start_medication_order" targetRef="harmonize_medication"/>
    <bpmn:sequenceFlow id="flow_to_safety_gateway" sourceRef="harmonize_medication" targetRef="safety_validation_gateway"/>
    <bpmn:sequenceFlow id="flow_to_drug_interaction" sourceRef="safety_validation_gateway" targetRef="drug_interaction_check"/>
    <bpmn:sequenceFlow id="flow_to_allergy_check" sourceRef="safety_validation_gateway" targetRef="allergy_check"/>
    <bpmn:sequenceFlow id="flow_to_dosage_validation" sourceRef="safety_validation_gateway" targetRef="dosage_validation"/>
    <bpmn:sequenceFlow id="flow_to_provider_auth" sourceRef="safety_validation_gateway" targetRef="provider_authorization"/>
    <bpmn:sequenceFlow id="flow_from_drug_interaction" sourceRef="drug_interaction_check" targetRef="safety_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_allergy_check" sourceRef="allergy_check" targetRef="safety_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_dosage_validation" sourceRef="dosage_validation" targetRef="safety_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_from_provider_auth" sourceRef="provider_authorization" targetRef="safety_convergence_gateway"/>
    <bpmn:sequenceFlow id="flow_to_clinical_review" sourceRef="safety_convergence_gateway" targetRef="clinical_review"/>
    <bpmn:sequenceFlow id="flow_to_submit_order" sourceRef="clinical_review" targetRef="submit_order"/>
    <bpmn:sequenceFlow id="flow_to_confirm_order" sourceRef="submit_order" targetRef="confirm_order"/>
    <bpmn:sequenceFlow id="flow_to_end" sourceRef="confirm_order" targetRef="medication-ordering-workflow-v1_end"/>
    
  </bpmn:process>
  
</bpmn:definitions>"""
    
    @staticmethod
    def get_compensation_workflows() -> Dict[str, str]:
        """
        Get compensation workflow definitions.
        """
        return {
            "reverse_harmonization": """
                1. Log harmonization reversal
                2. Clear harmonized medication data
                3. Notify clinical team
                4. Create audit entry
            """,
            "handle_safety_failure": """
                1. Stop workflow execution
                2. Log safety failure details
                3. Send immediate alert to provider
                4. Create patient safety incident report
                5. Escalate to clinical supervisor
            """,
            "cancel_order_and_notify": """
                1. Cancel pending medication order
                2. Reverse any partial submissions
                3. Notify ordering provider
                4. Update patient record
                5. Create cancellation audit trail
            """
        }
