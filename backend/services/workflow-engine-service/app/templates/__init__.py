"""
Clinical Workflow Templates module.
"""

from .medication_ordering_workflow import MedicationOrderingWorkflowTemplate
from .patient_admission_workflow import PatientAdmissionWorkflowTemplate
from .patient_discharge_workflow import PatientDischargeWorkflowTemplate

__all__ = [
    'MedicationOrderingWorkflowTemplate',
    'PatientAdmissionWorkflowTemplate', 
    'PatientDischargeWorkflowTemplate'
]
