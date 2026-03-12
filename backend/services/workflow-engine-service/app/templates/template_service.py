"""
Clinical Workflow Template Service.
Manages clinical workflow templates with BPMN 2.0 integration and safety mechanisms.
"""
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime

from app.models.clinical_workflow_templates import (
    ClinicalWorkflowTemplate, WorkflowTemplateType
)
from app.clinical.clinical_workflows.medication_ordering_workflow import MedicationOrderingWorkflowTemplate
from app.clinical.clinical_workflows.patient_admission_workflow import PatientAdmissionWorkflowTemplate
from app.clinical.clinical_workflows.patient_discharge_workflow import PatientDischargeWorkflowTemplate
from app.clinical.clinical_workflows.emergency_response_workflow import EmergencyResponseWorkflowTemplate
from app.clinical.clinical_workflows.clinical_assessment_workflow import ClinicalAssessmentWorkflowTemplate

logger = logging.getLogger(__name__)


class ClinicalWorkflowTemplateService:
    """
    Service for managing clinical workflow templates.
    """
    
    def __init__(self):
        self.templates: Dict[str, ClinicalWorkflowTemplate] = {}
        self.template_registry: Dict[WorkflowTemplateType, Any] = {
            WorkflowTemplateType.MEDICATION_ORDERING: MedicationOrderingWorkflowTemplate,
            WorkflowTemplateType.PATIENT_ADMISSION: PatientAdmissionWorkflowTemplate,
            WorkflowTemplateType.PATIENT_DISCHARGE: PatientDischargeWorkflowTemplate,
            WorkflowTemplateType.EMERGENCY_RESPONSE: EmergencyResponseWorkflowTemplate,
            WorkflowTemplateType.CLINICAL_ASSESSMENT: ClinicalAssessmentWorkflowTemplate
        }
        self._initialize_templates()
    
    def _initialize_templates(self):
        """
        Initialize all clinical workflow templates.
        """
        try:
            logger.info("🔄 Initializing clinical workflow templates...")
            
            # Load medication ordering workflow
            medication_template = MedicationOrderingWorkflowTemplate.create_template()
            self.templates[medication_template.template_id] = medication_template
            logger.info(f"✅ Loaded template: {medication_template.template_name}")
            
            # Load patient admission workflow
            admission_template = PatientAdmissionWorkflowTemplate.create_template()
            self.templates[admission_template.template_id] = admission_template
            logger.info(f"✅ Loaded template: {admission_template.template_name}")
            
            # Load patient discharge workflow
            discharge_template = PatientDischargeWorkflowTemplate.create_template()
            self.templates[discharge_template.template_id] = discharge_template
            logger.info(f"✅ Loaded template: {discharge_template.template_name}")

            # Load emergency response workflow
            emergency_template = EmergencyResponseWorkflowTemplate.create_template()
            self.templates[emergency_template.template_id] = emergency_template
            logger.info(f"✅ Loaded template: {emergency_template.template_name}")

            # Load clinical assessment workflow
            assessment_template = ClinicalAssessmentWorkflowTemplate.create_template()
            self.templates[assessment_template.template_id] = assessment_template
            logger.info(f"✅ Loaded template: {assessment_template.template_name}")

            logger.info(f"🎉 Successfully initialized {len(self.templates)} clinical workflow templates")
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize clinical workflow templates: {e}")
            raise
    
    def get_template(self, template_id: str) -> Optional[ClinicalWorkflowTemplate]:
        """
        Get a clinical workflow template by ID.
        """
        return self.templates.get(template_id)
    
    def get_template_by_type(self, template_type: WorkflowTemplateType) -> Optional[ClinicalWorkflowTemplate]:
        """
        Get a clinical workflow template by type.
        """
        for template in self.templates.values():
            if template.template_type == template_type:
                return template
        return None
    
    def list_templates(self) -> List[ClinicalWorkflowTemplate]:
        """
        List all available clinical workflow templates.
        """
        return list(self.templates.values())
    
    def get_template_summary(self) -> Dict[str, Any]:
        """
        Get summary information about all templates.
        """
        summary = {
            "total_templates": len(self.templates),
            "templates": []
        }
        
        for template in self.templates.values():
            template_info = {
                "template_id": template.template_id,
                "template_name": template.template_name,
                "template_type": template.template_type.value,
                "version": template.version,
                "description": template.description,
                "total_steps": len(template.steps),
                "safety_checks": len(template.global_safety_checks),
                "emergency_stops": len(template.emergency_stop_conditions),
                "max_execution_hours": template.max_execution_time_hours,
                "status": template.status,
                "created_at": template.created_at.isoformat()
            }
            summary["templates"].append(template_info)
        
        return summary
    
    def get_bpmn_xml(self, template_id: str) -> Optional[str]:
        """
        Get BPMN 2.0 XML for a template.
        """
        template = self.get_template(template_id)
        if not template:
            return None
        
        # Get BPMN XML from the appropriate template class
        template_class = self.template_registry.get(template.template_type)
        if template_class and hasattr(template_class, 'get_bpmn_xml'):
            return template_class.get_bpmn_xml()
        
        return None
    
    def get_compensation_workflows(self, template_id: str) -> Optional[Dict[str, str]]:
        """
        Get compensation workflow definitions for a template.
        """
        template = self.get_template(template_id)
        if not template:
            return None
        
        # Get compensation workflows from the appropriate template class
        template_class = self.template_registry.get(template.template_type)
        if template_class and hasattr(template_class, 'get_compensation_workflows'):
            return template_class.get_compensation_workflows()
        
        return {}
    
    def validate_template(self, template_id: str) -> Dict[str, Any]:
        """
        Validate a clinical workflow template.
        """
        template = self.get_template(template_id)
        if not template:
            return {
                "valid": False,
                "errors": [f"Template not found: {template_id}"]
            }
        
        errors = []
        warnings = []
        
        try:
            # Validate basic structure
            if not template.steps:
                errors.append("Template must have at least one step")
            
            if not template.start_event:
                errors.append("Template must have a start event")
            
            if not template.end_events:
                errors.append("Template must have at least one end event")
            
            # Validate step references
            step_ids = {step.step_id for step in template.steps}
            for step in template.steps:
                for next_step in step.next_steps:
                    if next_step not in step_ids and next_step not in template.end_events:
                        errors.append(f"Invalid next step reference in {step.step_id}: {next_step}")
            
            # Validate safety checks
            for step in template.steps:
                for safety_check in step.safety_checks:
                    if not safety_check.required_data_sources:
                        warnings.append(f"Safety check {safety_check.check_id} in step {step.step_id} has no required data sources")
            
            # Validate activities
            for step in template.steps:
                if step.activity and step.activity.real_data_only:
                    if not step.activity.approved_data_sources:
                        warnings.append(f"Activity {step.activity.activity_id} requires real data but has no approved data sources")
            
            # Validate SLA targets
            if not template.sla_targets:
                warnings.append("Template has no SLA targets defined")
            
            # Validate compensation handlers
            for step in template.steps:
                if step.compensation_handler and step.compensation_handler not in template.compensation_workflows:
                    warnings.append(f"Step {step.step_id} references undefined compensation handler: {step.compensation_handler}")
            
            return {
                "valid": len(errors) == 0,
                "errors": errors,
                "warnings": warnings,
                "validation_timestamp": datetime.utcnow().isoformat()
            }
            
        except Exception as e:
            return {
                "valid": False,
                "errors": [f"Validation failed: {str(e)}"],
                "warnings": warnings,
                "validation_timestamp": datetime.utcnow().isoformat()
            }
    
    def get_template_metrics(self, template_id: str) -> Optional[Dict[str, Any]]:
        """
        Get metrics and statistics for a template.
        """
        template = self.get_template(template_id)
        if not template:
            return None
        
        # Count different types of steps
        task_steps = sum(1 for step in template.steps if step.step_type == "task")
        gateway_steps = sum(1 for step in template.steps if step.step_type == "gateway")
        human_steps = sum(1 for step in template.steps if step.step_type == "task" and step.activity and step.activity.activity_type.value == "human")
        
        # Count safety features
        total_safety_checks = len(template.global_safety_checks)
        for step in template.steps:
            total_safety_checks += len(step.safety_checks)
        
        # Calculate complexity score
        complexity_score = (
            len(template.steps) * 1 +
            total_safety_checks * 2 +
            len(template.emergency_stop_conditions) * 3 +
            len(template.compensation_workflows) * 2
        )
        
        return {
            "template_id": template.template_id,
            "template_name": template.template_name,
            "total_steps": len(template.steps),
            "task_steps": task_steps,
            "gateway_steps": gateway_steps,
            "human_steps": human_steps,
            "total_safety_checks": total_safety_checks,
            "global_safety_checks": len(template.global_safety_checks),
            "emergency_stop_conditions": len(template.emergency_stop_conditions),
            "compensation_workflows": len(template.compensation_workflows),
            "complexity_score": complexity_score,
            "max_execution_hours": template.max_execution_time_hours,
            "phi_handling": template.phi_handling,
            "audit_level": template.audit_level,
            "retention_years": template.retention_years
        }
    
    def export_template_definition(self, template_id: str) -> Optional[Dict[str, Any]]:
        """
        Export complete template definition for external use.
        """
        template = self.get_template(template_id)
        if not template:
            return None
        
        # Convert template to dictionary format
        template_dict = {
            "template_id": template.template_id,
            "template_name": template.template_name,
            "template_type": template.template_type.value,
            "version": template.version,
            "description": template.description,
            "start_event": template.start_event,
            "end_events": template.end_events,
            "steps": [],
            "gateways": template.gateways,
            "timers": template.timers,
            "global_safety_checks": [],
            "emergency_stop_conditions": template.emergency_stop_conditions,
            "compensation_workflows": template.compensation_workflows,
            "required_approvals": template.required_approvals,
            "audit_level": template.audit_level,
            "retention_years": template.retention_years,
            "phi_handling": template.phi_handling,
            "max_execution_time_hours": template.max_execution_time_hours,
            "sla_targets": template.sla_targets,
            "created_by": template.created_by,
            "created_at": template.created_at.isoformat(),
            "updated_at": template.updated_at.isoformat(),
            "status": template.status
        }
        
        # Convert steps
        for step in template.steps:
            step_dict = {
                "step_id": step.step_id,
                "step_name": step.step_name,
                "step_type": step.step_type,
                "parallel_execution": step.parallel_execution,
                "timeout_minutes": step.timeout_minutes,
                "escalation_minutes": step.escalation_minutes,
                "required_roles": step.required_roles,
                "input_mapping": step.input_mapping,
                "output_mapping": step.output_mapping,
                "compensation_handler": step.compensation_handler,
                "next_steps": step.next_steps,
                "condition_expression": step.condition_expression,
                "safety_checks": [],
                "activity": None
            }
            
            # Convert safety checks
            for safety_check in step.safety_checks:
                safety_check_dict = {
                    "check_id": safety_check.check_id,
                    "check_type": safety_check.check_type.value,
                    "description": safety_check.description,
                    "required_data_sources": [ds.value for ds in safety_check.required_data_sources],
                    "timeout_seconds": safety_check.timeout_seconds,
                    "critical": safety_check.critical,
                    "retry_count": safety_check.retry_count,
                    "escalation_rules": safety_check.escalation_rules
                }
                step_dict["safety_checks"].append(safety_check_dict)
            
            # Convert activity
            if step.activity:
                step_dict["activity"] = {
                    "activity_id": step.activity.activity_id,
                    "activity_type": step.activity.activity_type.value,
                    "timeout_seconds": step.activity.timeout_seconds,
                    "safety_critical": step.activity.safety_critical,
                    "requires_clinical_context": step.activity.requires_clinical_context,
                    "audit_level": step.activity.audit_level,
                    "real_data_only": step.activity.real_data_only,
                    "fail_on_unavailable": step.activity.fail_on_unavailable,
                    "approved_data_sources": [ds.value for ds in step.activity.approved_data_sources] if step.activity.approved_data_sources else [],
                    "created_at": step.activity.created_at.isoformat()
                }
            
            template_dict["steps"].append(step_dict)
        
        # Convert global safety checks
        for safety_check in template.global_safety_checks:
            safety_check_dict = {
                "check_id": safety_check.check_id,
                "check_type": safety_check.check_type.value,
                "description": safety_check.description,
                "required_data_sources": [ds.value for ds in safety_check.required_data_sources],
                "timeout_seconds": safety_check.timeout_seconds,
                "critical": safety_check.critical,
                "retry_count": safety_check.retry_count,
                "escalation_rules": safety_check.escalation_rules
            }
            template_dict["global_safety_checks"].append(safety_check_dict)
        
        return template_dict


# Global template service instance
clinical_workflow_template_service = ClinicalWorkflowTemplateService()
