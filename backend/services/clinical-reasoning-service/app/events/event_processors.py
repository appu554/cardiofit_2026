"""
Workflow-Specific Event Processors

Specialized processors for different clinical workflows including
medication workflows, laboratory workflows, and clinical decision workflows.
Each processor handles workflow-specific logic, validation, and enrichment.
"""

import logging
import asyncio
from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Any, Callable
from abc import ABC, abstractmethod
from dataclasses import dataclass
from enum import Enum

from .clinical_event_envelope import (
    ClinicalEventEnvelope, EventType, EventSeverity, EventStatus,
    ClinicalContext, TemporalContext, ProvenanceContext, EventMetadata
)

logger = logging.getLogger(__name__)


class WorkflowType(Enum):
    """Types of clinical workflows"""
    MEDICATION = "medication"
    LABORATORY = "laboratory"
    CLINICAL_DECISION = "clinical_decision"
    PATIENT_ENCOUNTER = "patient_encounter"
    ADVERSE_EVENT = "adverse_event"


class ProcessingResult(Enum):
    """Results of event processing"""
    SUCCESS = "success"
    FAILED = "failed"
    REQUIRES_REVIEW = "requires_review"
    ESCALATED = "escalated"
    DEFERRED = "deferred"


@dataclass
class ProcessingOutcome:
    """Outcome of event processing"""
    result: ProcessingResult
    processed_envelope: Optional[ClinicalEventEnvelope]
    warnings: List[str]
    errors: List[str]
    recommendations: List[str]
    next_actions: List[Dict[str, Any]]
    processing_duration_ms: float
    metadata: Dict[str, Any]


class BaseEventProcessor(ABC):
    """Base class for workflow-specific event processors"""
    
    def __init__(self, workflow_type: WorkflowType):
        self.workflow_type = workflow_type
        self.processing_stats = {
            "total_processed": 0,
            "successful": 0,
            "failed": 0,
            "average_processing_time": 0.0
        }
        
        logger.info(f"Initialized {workflow_type.value} event processor")
    
    @abstractmethod
    async def process_event(self, envelope: ClinicalEventEnvelope) -> ProcessingOutcome:
        """Process a clinical event envelope"""
        pass
    
    @abstractmethod
    async def validate_event(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Validate event for workflow-specific requirements"""
        pass
    
    @abstractmethod
    async def enrich_event(self, envelope: ClinicalEventEnvelope) -> ClinicalEventEnvelope:
        """Enrich event with workflow-specific context"""
        pass
    
    async def _update_processing_stats(self, processing_time_ms: float, success: bool):
        """Update processing statistics"""
        self.processing_stats["total_processed"] += 1
        
        if success:
            self.processing_stats["successful"] += 1
        else:
            self.processing_stats["failed"] += 1
        
        # Update average processing time
        total = self.processing_stats["total_processed"]
        current_avg = self.processing_stats["average_processing_time"]
        self.processing_stats["average_processing_time"] = (
            (current_avg * (total - 1) + processing_time_ms) / total
        )
    
    def get_processing_stats(self) -> Dict[str, Any]:
        """Get processing statistics"""
        return self.processing_stats.copy()


class MedicationWorkflowProcessor(BaseEventProcessor):
    """
    Processor for medication-related workflows
    
    Handles medication orders, administration, reconciliation,
    and medication-related clinical decisions.
    """
    
    def __init__(self):
        super().__init__(WorkflowType.MEDICATION)
        
        # Medication-specific configuration
        self.high_risk_medications = {
            "warfarin", "heparin", "insulin", "digoxin", "lithium",
            "methotrexate", "phenytoin", "theophylline"
        }
        
        self.medication_interaction_cache = {}
        
    async def process_event(self, envelope: ClinicalEventEnvelope) -> ProcessingOutcome:
        """Process medication workflow event"""
        start_time = datetime.now()
        warnings = []
        errors = []
        recommendations = []
        next_actions = []
        
        try:
            # 1. Validate medication event
            validation_errors = await self.validate_event(envelope)
            if validation_errors:
                errors.extend(validation_errors)
                return ProcessingOutcome(
                    result=ProcessingResult.FAILED,
                    processed_envelope=None,
                    warnings=warnings,
                    errors=errors,
                    recommendations=recommendations,
                    next_actions=next_actions,
                    processing_duration_ms=0.0,
                    metadata={"validation_failed": True}
                )
            
            # 2. Enrich with medication-specific context
            enriched_envelope = await self.enrich_event(envelope)
            
            # 3. Check for high-risk medications
            high_risk_warnings = await self._check_high_risk_medications(enriched_envelope)
            warnings.extend(high_risk_warnings)
            
            # 4. Check for drug interactions
            interaction_warnings = await self._check_drug_interactions(enriched_envelope)
            warnings.extend(interaction_warnings)
            
            # 5. Check dosing appropriateness
            dosing_recommendations = await self._check_dosing_appropriateness(enriched_envelope)
            recommendations.extend(dosing_recommendations)
            
            # 6. Generate next actions
            next_actions = await self._generate_medication_actions(enriched_envelope, warnings)
            
            # 7. Update envelope status
            if warnings:
                enriched_envelope.update_status(EventStatus.COMPLETED, "medication_processor")
                enriched_envelope.metadata.event_severity = EventSeverity.MODERATE
            else:
                enriched_envelope.update_status(EventStatus.COMPLETED, "medication_processor")
            
            # Calculate processing time
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, True)
            
            result = ProcessingResult.REQUIRES_REVIEW if warnings else ProcessingResult.SUCCESS
            
            return ProcessingOutcome(
                result=result,
                processed_envelope=enriched_envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={"high_risk_detected": len(high_risk_warnings) > 0}
            )
            
        except Exception as e:
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, False)
            
            logger.error(f"Error processing medication event: {e}")
            errors.append(f"Processing error: {str(e)}")
            
            return ProcessingOutcome(
                result=ProcessingResult.FAILED,
                processed_envelope=envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={"processing_error": True}
            )
    
    async def validate_event(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Validate medication event"""
        errors = []
        
        # Check for required medication data
        event_data = envelope.event_data
        
        if "medication_id" not in event_data and "medication_name" not in event_data:
            errors.append("Medication ID or name is required")
        
        if "dosage" not in event_data:
            errors.append("Medication dosage is required")
        
        if "route" not in event_data:
            errors.append("Administration route is required")
        
        # Validate patient context
        if not envelope.clinical_context.patient_id:
            errors.append("Patient ID is required for medication events")
        
        # Check for active allergies
        if envelope.clinical_context.active_allergies:
            medication_name = event_data.get("medication_name", "").lower()
            for allergy in envelope.clinical_context.active_allergies:
                allergy_name = allergy.get("allergen", "").lower()
                if medication_name and allergy_name in medication_name:
                    errors.append(f"Patient has known allergy to {allergy_name}")
        
        return errors
    
    async def enrich_event(self, envelope: ClinicalEventEnvelope) -> ClinicalEventEnvelope:
        """Enrich medication event with additional context"""
        # Add medication-specific temporal context
        envelope.temporal_context.clinical_phase = "medication_administration"
        
        # Add medication-specific provenance
        envelope.add_provenance_entry(
            "medication_processor",
            "medication_workflow_enrichment",
            confidence=0.95
        )
        
        # Add medication class information
        medication_name = envelope.event_data.get("medication_name", "")
        medication_class = await self._get_medication_class(medication_name)
        
        if medication_class:
            envelope.event_data["medication_class"] = medication_class
            envelope.event_data["therapeutic_category"] = await self._get_therapeutic_category(medication_name)
        
        # Add dosing context
        await self._add_dosing_context(envelope)
        
        return envelope
    
    async def _check_high_risk_medications(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check for high-risk medications"""
        warnings = []
        
        medication_name = envelope.event_data.get("medication_name", "").lower()
        
        for high_risk_med in self.high_risk_medications:
            if high_risk_med in medication_name:
                warnings.append(f"High-risk medication detected: {high_risk_med}")
                
                # Add clinical warning to envelope
                envelope.add_clinical_warning(
                    warning_type="high_risk_medication",
                    severity="high",
                    description=f"High-risk medication: {high_risk_med}",
                    source="medication_processor"
                )
        
        return warnings
    
    async def _check_drug_interactions(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check for potential drug interactions"""
        warnings = []
        
        current_medication = envelope.event_data.get("medication_name", "")
        active_medications = envelope.clinical_context.active_medications
        
        # Simple interaction checking (in production, use comprehensive drug database)
        known_interactions = {
            "warfarin": ["aspirin", "ibuprofen", "naproxen"],
            "digoxin": ["furosemide", "spironolactone"],
            "lithium": ["furosemide", "lisinopril", "ibuprofen"]
        }
        
        current_med_lower = current_medication.lower()
        
        for interaction_med, interacting_drugs in known_interactions.items():
            if interaction_med in current_med_lower:
                for active_med in active_medications:
                    active_med_name = active_med.get("name", "").lower()
                    for interacting_drug in interacting_drugs:
                        if interacting_drug in active_med_name:
                            warning = f"Potential interaction: {current_medication} with {active_med['name']}"
                            warnings.append(warning)
                            
                            envelope.add_clinical_warning(
                                warning_type="drug_interaction",
                                severity="moderate",
                                description=warning,
                                source="medication_processor"
                            )
        
        return warnings
    
    async def _check_dosing_appropriateness(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check dosing appropriateness"""
        recommendations = []
        
        # Get patient demographics for dosing considerations
        demographics = envelope.clinical_context.patient_demographics
        age = demographics.get("age", 0)
        weight = demographics.get("weight", 0)
        
        dosage = envelope.event_data.get("dosage", "")
        medication_name = envelope.event_data.get("medication_name", "").lower()
        
        # Age-based dosing recommendations
        if age > 65:
            if "digoxin" in medication_name:
                recommendations.append("Consider reduced digoxin dose for elderly patient")
            elif "benzodiazepine" in medication_name:
                recommendations.append("Use caution with benzodiazepines in elderly patients")
        
        # Weight-based dosing recommendations
        if weight > 0 and "weight-based" in dosage.lower():
            recommendations.append("Verify weight-based dosing calculation")
        
        return recommendations
    
    async def _generate_medication_actions(self, envelope: ClinicalEventEnvelope, 
                                         warnings: List[str]) -> List[Dict[str, Any]]:
        """Generate next actions for medication workflow"""
        actions = []
        
        if warnings:
            actions.append({
                "action": "clinical_review",
                "priority": "high",
                "description": "Clinical review required due to medication warnings",
                "assigned_to": "pharmacist",
                "deadline": (datetime.now() + timedelta(hours=2)).isoformat()
            })
        
        # Standard medication actions
        actions.append({
            "action": "administration_verification",
            "priority": "normal",
            "description": "Verify medication administration",
            "assigned_to": "nurse",
            "deadline": (datetime.now() + timedelta(hours=1)).isoformat()
        })
        
        return actions
    
    async def _get_medication_class(self, medication_name: str) -> Optional[str]:
        """Get medication class (simplified implementation)"""
        medication_classes = {
            "warfarin": "anticoagulant",
            "aspirin": "antiplatelet",
            "metformin": "antidiabetic",
            "lisinopril": "ace_inhibitor",
            "furosemide": "diuretic"
        }
        
        for med, med_class in medication_classes.items():
            if med in medication_name.lower():
                return med_class
        
        return None
    
    async def _get_therapeutic_category(self, medication_name: str) -> Optional[str]:
        """Get therapeutic category (simplified implementation)"""
        therapeutic_categories = {
            "warfarin": "cardiovascular",
            "aspirin": "cardiovascular",
            "metformin": "endocrine",
            "lisinopril": "cardiovascular",
            "furosemide": "cardiovascular"
        }
        
        for med, category in therapeutic_categories.items():
            if med in medication_name.lower():
                return category
        
        return None
    
    async def _add_dosing_context(self, envelope: ClinicalEventEnvelope):
        """Add dosing context to envelope"""
        dosage = envelope.event_data.get("dosage", "")
        
        # Parse dosage information (simplified)
        if "mg" in dosage:
            envelope.event_data["dosage_unit"] = "mg"
        elif "ml" in dosage:
            envelope.event_data["dosage_unit"] = "ml"
        
        # Add frequency information
        frequency = envelope.event_data.get("frequency", "")
        if frequency:
            envelope.event_data["administration_frequency"] = frequency


class LaboratoryWorkflowProcessor(BaseEventProcessor):
    """
    Processor for laboratory-related workflows

    Handles laboratory orders, results, critical values,
    and laboratory-related clinical decisions.
    """

    def __init__(self):
        super().__init__(WorkflowType.LABORATORY)

        # Laboratory-specific configuration
        self.critical_value_ranges = {
            "glucose": {"critical_low": 40, "critical_high": 400},
            "potassium": {"critical_low": 2.5, "critical_high": 6.0},
            "sodium": {"critical_low": 120, "critical_high": 160},
            "creatinine": {"critical_low": 0.3, "critical_high": 10.0},
            "hemoglobin": {"critical_low": 5.0, "critical_high": 20.0}
        }

        self.panic_value_ranges = {
            "glucose": {"panic_low": 30, "panic_high": 500},
            "potassium": {"panic_low": 2.0, "panic_high": 7.0},
            "troponin": {"panic_high": 50.0}
        }

    async def process_event(self, envelope: ClinicalEventEnvelope) -> ProcessingOutcome:
        """Process laboratory workflow event"""
        start_time = datetime.now()
        warnings = []
        errors = []
        recommendations = []
        next_actions = []

        try:
            # 1. Validate laboratory event
            validation_errors = await self.validate_event(envelope)
            if validation_errors:
                errors.extend(validation_errors)
                return ProcessingOutcome(
                    result=ProcessingResult.FAILED,
                    processed_envelope=None,
                    warnings=warnings,
                    errors=errors,
                    recommendations=recommendations,
                    next_actions=next_actions,
                    processing_duration_ms=0.0,
                    metadata={"validation_failed": True}
                )

            # 2. Enrich with laboratory-specific context
            enriched_envelope = await self.enrich_event(envelope)

            # 3. Check for critical values
            critical_warnings = await self._check_critical_values(enriched_envelope)
            warnings.extend(critical_warnings)

            # 4. Check for panic values
            panic_warnings = await self._check_panic_values(enriched_envelope)
            warnings.extend(panic_warnings)

            # 5. Generate clinical recommendations
            lab_recommendations = await self._generate_lab_recommendations(enriched_envelope)
            recommendations.extend(lab_recommendations)

            # 6. Generate next actions
            next_actions = await self._generate_laboratory_actions(enriched_envelope, warnings)

            # 7. Update envelope status
            if panic_warnings:
                enriched_envelope.update_status(EventStatus.COMPLETED, "laboratory_processor")
                enriched_envelope.metadata.event_severity = EventSeverity.CRITICAL
                result = ProcessingResult.ESCALATED
            elif critical_warnings:
                enriched_envelope.update_status(EventStatus.COMPLETED, "laboratory_processor")
                enriched_envelope.metadata.event_severity = EventSeverity.HIGH
                result = ProcessingResult.REQUIRES_REVIEW
            else:
                enriched_envelope.update_status(EventStatus.COMPLETED, "laboratory_processor")
                result = ProcessingResult.SUCCESS

            # Calculate processing time
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, True)

            return ProcessingOutcome(
                result=result,
                processed_envelope=enriched_envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={
                    "critical_values_detected": len(critical_warnings) > 0,
                    "panic_values_detected": len(panic_warnings) > 0
                }
            )

        except Exception as e:
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, False)

            logger.error(f"Error processing laboratory event: {e}")
            errors.append(f"Processing error: {str(e)}")

            return ProcessingOutcome(
                result=ProcessingResult.FAILED,
                processed_envelope=envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={"processing_error": True}
            )

    async def validate_event(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Validate laboratory event"""
        errors = []

        event_data = envelope.event_data

        # Check for required laboratory data
        if "test_name" not in event_data:
            errors.append("Laboratory test name is required")

        if "result_value" not in event_data:
            errors.append("Laboratory result value is required")

        if "reference_range" not in event_data:
            errors.append("Reference range is required")

        # Validate patient context
        if not envelope.clinical_context.patient_id:
            errors.append("Patient ID is required for laboratory events")

        return errors

    async def enrich_event(self, envelope: ClinicalEventEnvelope) -> ClinicalEventEnvelope:
        """Enrich laboratory event with additional context"""
        # Add laboratory-specific temporal context
        envelope.temporal_context.clinical_phase = "laboratory_result"

        # Add laboratory-specific provenance
        envelope.add_provenance_entry(
            "laboratory_processor",
            "laboratory_workflow_enrichment",
            confidence=0.98
        )

        # Add test categorization
        test_name = envelope.event_data.get("test_name", "").lower()
        test_category = await self._get_test_category(test_name)

        if test_category:
            envelope.event_data["test_category"] = test_category

        # Add clinical significance
        await self._add_clinical_significance(envelope)

        return envelope

    async def _check_critical_values(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check for critical laboratory values"""
        warnings = []

        test_name = envelope.event_data.get("test_name", "").lower()
        result_value = envelope.event_data.get("result_value")

        if not isinstance(result_value, (int, float)):
            try:
                result_value = float(result_value)
            except (ValueError, TypeError):
                return warnings

        for test, ranges in self.critical_value_ranges.items():
            if test in test_name:
                if result_value < ranges.get("critical_low", float('-inf')):
                    warning = f"Critical low {test}: {result_value}"
                    warnings.append(warning)
                    envelope.add_clinical_warning(
                        warning_type="critical_lab_value",
                        severity="high",
                        description=warning,
                        source="laboratory_processor"
                    )
                elif result_value > ranges.get("critical_high", float('inf')):
                    warning = f"Critical high {test}: {result_value}"
                    warnings.append(warning)
                    envelope.add_clinical_warning(
                        warning_type="critical_lab_value",
                        severity="high",
                        description=warning,
                        source="laboratory_processor"
                    )

        return warnings

    async def _check_panic_values(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check for panic laboratory values"""
        warnings = []

        test_name = envelope.event_data.get("test_name", "").lower()
        result_value = envelope.event_data.get("result_value")

        if not isinstance(result_value, (int, float)):
            try:
                result_value = float(result_value)
            except (ValueError, TypeError):
                return warnings

        for test, ranges in self.panic_value_ranges.items():
            if test in test_name:
                if result_value < ranges.get("panic_low", float('-inf')):
                    warning = f"PANIC VALUE - Low {test}: {result_value}"
                    warnings.append(warning)
                    envelope.add_clinical_warning(
                        warning_type="panic_lab_value",
                        severity="critical",
                        description=warning,
                        source="laboratory_processor"
                    )
                elif result_value > ranges.get("panic_high", float('inf')):
                    warning = f"PANIC VALUE - High {test}: {result_value}"
                    warnings.append(warning)
                    envelope.add_clinical_warning(
                        warning_type="panic_lab_value",
                        severity="critical",
                        description=warning,
                        source="laboratory_processor"
                    )

        return warnings

    async def _generate_lab_recommendations(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Generate laboratory-based recommendations"""
        recommendations = []

        test_name = envelope.event_data.get("test_name", "").lower()
        result_value = envelope.event_data.get("result_value")

        # Generate test-specific recommendations
        if "glucose" in test_name and isinstance(result_value, (int, float)):
            if result_value > 200:
                recommendations.append("Consider diabetes management review")
            elif result_value < 70:
                recommendations.append("Monitor for hypoglycemia symptoms")

        if "creatinine" in test_name and isinstance(result_value, (int, float)):
            if result_value > 1.5:
                recommendations.append("Assess kidney function and medication dosing")

        return recommendations

    async def _generate_laboratory_actions(self, envelope: ClinicalEventEnvelope,
                                         warnings: List[str]) -> List[Dict[str, Any]]:
        """Generate next actions for laboratory workflow"""
        actions = []

        if any("PANIC" in warning for warning in warnings):
            actions.append({
                "action": "immediate_notification",
                "priority": "critical",
                "description": "Immediate physician notification for panic values",
                "assigned_to": "physician",
                "deadline": (datetime.now() + timedelta(minutes=15)).isoformat()
            })

        if warnings:
            actions.append({
                "action": "clinical_review",
                "priority": "high",
                "description": "Clinical review required for abnormal lab values",
                "assigned_to": "physician",
                "deadline": (datetime.now() + timedelta(hours=1)).isoformat()
            })

        return actions

    async def _get_test_category(self, test_name: str) -> Optional[str]:
        """Get laboratory test category"""
        test_categories = {
            "glucose": "chemistry",
            "potassium": "chemistry",
            "sodium": "chemistry",
            "creatinine": "chemistry",
            "hemoglobin": "hematology",
            "troponin": "cardiac_markers"
        }

        for test, category in test_categories.items():
            if test in test_name:
                return category

        return "general"

    async def _add_clinical_significance(self, envelope: ClinicalEventEnvelope):
        """Add clinical significance to laboratory result"""
        test_name = envelope.event_data.get("test_name", "").lower()

        # Add clinical significance based on test type
        if "troponin" in test_name:
            envelope.event_data["clinical_significance"] = "cardiac_injury_marker"
        elif "glucose" in test_name:
            envelope.event_data["clinical_significance"] = "metabolic_marker"
        elif "creatinine" in test_name:
            envelope.event_data["clinical_significance"] = "kidney_function_marker"


class ClinicalDecisionProcessor(BaseEventProcessor):
    """
    Processor for clinical decision workflows

    Handles clinical assertions, decision support alerts,
    and clinical reasoning events.
    """

    def __init__(self):
        super().__init__(WorkflowType.CLINICAL_DECISION)

        # Decision-specific configuration
        self.decision_confidence_threshold = 0.7
        self.escalation_severity_threshold = EventSeverity.HIGH

    async def process_event(self, envelope: ClinicalEventEnvelope) -> ProcessingOutcome:
        """Process clinical decision event"""
        start_time = datetime.now()
        warnings = []
        errors = []
        recommendations = []
        next_actions = []

        try:
            # 1. Validate clinical decision event
            validation_errors = await self.validate_event(envelope)
            if validation_errors:
                errors.extend(validation_errors)
                return ProcessingOutcome(
                    result=ProcessingResult.FAILED,
                    processed_envelope=None,
                    warnings=warnings,
                    errors=errors,
                    recommendations=recommendations,
                    next_actions=next_actions,
                    processing_duration_ms=0.0,
                    metadata={"validation_failed": True}
                )

            # 2. Enrich with decision-specific context
            enriched_envelope = await self.enrich_event(envelope)

            # 3. Assess decision confidence
            confidence_warnings = await self._assess_decision_confidence(enriched_envelope)
            warnings.extend(confidence_warnings)

            # 4. Check for conflicting decisions
            conflict_warnings = await self._check_decision_conflicts(enriched_envelope)
            warnings.extend(conflict_warnings)

            # 5. Generate clinical recommendations
            decision_recommendations = await self._generate_decision_recommendations(enriched_envelope)
            recommendations.extend(decision_recommendations)

            # 6. Generate next actions
            next_actions = await self._generate_decision_actions(enriched_envelope, warnings)

            # 7. Update envelope status
            confidence_score = envelope.event_data.get("confidence_score", 1.0)

            if confidence_score < self.decision_confidence_threshold:
                enriched_envelope.update_status(EventStatus.COMPLETED, "decision_processor")
                enriched_envelope.metadata.event_severity = EventSeverity.MODERATE
                result = ProcessingResult.REQUIRES_REVIEW
            elif enriched_envelope.metadata.event_severity.value in ["high", "critical"]:
                result = ProcessingResult.ESCALATED
            else:
                enriched_envelope.update_status(EventStatus.COMPLETED, "decision_processor")
                result = ProcessingResult.SUCCESS

            # Calculate processing time
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, True)

            return ProcessingOutcome(
                result=result,
                processed_envelope=enriched_envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={
                    "confidence_score": confidence_score,
                    "requires_escalation": result == ProcessingResult.ESCALATED
                }
            )

        except Exception as e:
            processing_time = (datetime.now() - start_time).total_seconds() * 1000
            await self._update_processing_stats(processing_time, False)

            logger.error(f"Error processing clinical decision event: {e}")
            errors.append(f"Processing error: {str(e)}")

            return ProcessingOutcome(
                result=ProcessingResult.FAILED,
                processed_envelope=envelope,
                warnings=warnings,
                errors=errors,
                recommendations=recommendations,
                next_actions=next_actions,
                processing_duration_ms=processing_time,
                metadata={"processing_error": True}
            )

    async def validate_event(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Validate clinical decision event"""
        errors = []

        event_data = envelope.event_data

        # Check for required decision data
        if "decision_type" not in event_data:
            errors.append("Decision type is required")

        if "confidence_score" not in event_data:
            errors.append("Confidence score is required")

        # Validate confidence score range
        confidence = event_data.get("confidence_score")
        if confidence is not None and not (0.0 <= confidence <= 1.0):
            errors.append("Confidence score must be between 0.0 and 1.0")

        return errors

    async def enrich_event(self, envelope: ClinicalEventEnvelope) -> ClinicalEventEnvelope:
        """Enrich clinical decision event with additional context"""
        # Add decision-specific temporal context
        envelope.temporal_context.clinical_phase = "clinical_decision"

        # Add decision-specific provenance
        envelope.add_provenance_entry(
            "decision_processor",
            "clinical_decision_enrichment",
            confidence=envelope.event_data.get("confidence_score", 0.8)
        )

        # Add decision categorization
        decision_type = envelope.event_data.get("decision_type", "")
        decision_category = await self._get_decision_category(decision_type)

        if decision_category:
            envelope.event_data["decision_category"] = decision_category

        return envelope

    async def _assess_decision_confidence(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Assess decision confidence and generate warnings"""
        warnings = []

        confidence_score = envelope.event_data.get("confidence_score", 1.0)

        if confidence_score < self.decision_confidence_threshold:
            warning = f"Low confidence decision: {confidence_score:.2f}"
            warnings.append(warning)

            envelope.add_clinical_warning(
                warning_type="low_confidence_decision",
                severity="moderate",
                description=warning,
                source="decision_processor"
            )

        return warnings

    async def _check_decision_conflicts(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Check for conflicting clinical decisions"""
        warnings = []

        # This would integrate with existing decision history
        # For now, implement basic conflict detection

        decision_type = envelope.event_data.get("decision_type", "")

        # Example: Check for conflicting medication decisions
        if "medication" in decision_type.lower():
            # Check against active medications for conflicts
            active_medications = envelope.clinical_context.active_medications

            for med in active_medications:
                if "contraindicated" in med.get("status", "").lower():
                    warning = f"Conflicting medication decision with {med.get('name', 'unknown')}"
                    warnings.append(warning)

                    envelope.add_clinical_warning(
                        warning_type="decision_conflict",
                        severity="high",
                        description=warning,
                        source="decision_processor"
                    )

        return warnings

    async def _generate_decision_recommendations(self, envelope: ClinicalEventEnvelope) -> List[str]:
        """Generate decision-based recommendations"""
        recommendations = []

        confidence_score = envelope.event_data.get("confidence_score", 1.0)
        decision_type = envelope.event_data.get("decision_type", "")

        if confidence_score < 0.8:
            recommendations.append("Consider additional clinical consultation")

        if "high_risk" in decision_type.lower():
            recommendations.append("Implement enhanced monitoring protocols")

        return recommendations

    async def _generate_decision_actions(self, envelope: ClinicalEventEnvelope,
                                       warnings: List[str]) -> List[Dict[str, Any]]:
        """Generate next actions for clinical decision workflow"""
        actions = []

        confidence_score = envelope.event_data.get("confidence_score", 1.0)

        if confidence_score < self.decision_confidence_threshold:
            actions.append({
                "action": "expert_consultation",
                "priority": "high",
                "description": "Expert consultation required for low confidence decision",
                "assigned_to": "specialist",
                "deadline": (datetime.now() + timedelta(hours=4)).isoformat()
            })

        if warnings:
            actions.append({
                "action": "decision_review",
                "priority": "high",
                "description": "Review required due to decision warnings",
                "assigned_to": "attending_physician",
                "deadline": (datetime.now() + timedelta(hours=2)).isoformat()
            })

        return actions

    async def _get_decision_category(self, decision_type: str) -> str:
        """Get decision category"""
        decision_categories = {
            "medication": "therapeutic",
            "diagnostic": "diagnostic",
            "procedure": "procedural",
            "discharge": "care_coordination",
            "referral": "care_coordination"
        }

        for keyword, category in decision_categories.items():
            if keyword in decision_type.lower():
                return category

        return "general"


class EventProcessorRegistry:
    """Registry for managing workflow-specific event processors"""

    def __init__(self):
        self.processors: Dict[WorkflowType, BaseEventProcessor] = {}
        self.default_processor: Optional[BaseEventProcessor] = None

        # Initialize default processors
        self._initialize_default_processors()

        logger.info("Event Processor Registry initialized")

    def _initialize_default_processors(self):
        """Initialize default processors for each workflow type"""
        self.register_processor(WorkflowType.MEDICATION, MedicationWorkflowProcessor())
        self.register_processor(WorkflowType.LABORATORY, LaboratoryWorkflowProcessor())
        self.register_processor(WorkflowType.CLINICAL_DECISION, ClinicalDecisionProcessor())

    def register_processor(self, workflow_type: WorkflowType, processor: BaseEventProcessor):
        """Register a processor for a specific workflow type"""
        self.processors[workflow_type] = processor
        logger.info(f"Registered processor for {workflow_type.value} workflow")

    def get_processor(self, workflow_type: WorkflowType) -> Optional[BaseEventProcessor]:
        """Get processor for a specific workflow type"""
        return self.processors.get(workflow_type)

    def set_default_processor(self, processor: BaseEventProcessor):
        """Set default processor for unregistered workflow types"""
        self.default_processor = processor

    async def process_event(self, envelope: ClinicalEventEnvelope) -> ProcessingOutcome:
        """Process event using appropriate workflow processor"""
        # Determine workflow type from event
        workflow_type = self._determine_workflow_type(envelope)

        # Get appropriate processor
        processor = self.get_processor(workflow_type)

        if not processor:
            if self.default_processor:
                processor = self.default_processor
            else:
                # Return failed outcome if no processor available
                return ProcessingOutcome(
                    result=ProcessingResult.FAILED,
                    processed_envelope=envelope,
                    warnings=[],
                    errors=[f"No processor available for workflow type: {workflow_type.value}"],
                    recommendations=[],
                    next_actions=[],
                    processing_duration_ms=0.0,
                    metadata={"no_processor": True}
                )

        # Process the event
        return await processor.process_event(envelope)

    def _determine_workflow_type(self, envelope: ClinicalEventEnvelope) -> WorkflowType:
        """Determine workflow type from event envelope"""
        event_type = envelope.metadata.event_type

        # Map event types to workflow types
        event_to_workflow_map = {
            EventType.MEDICATION_ORDER: WorkflowType.MEDICATION,
            EventType.LABORATORY_RESULT: WorkflowType.LABORATORY,
            EventType.CLINICAL_ASSERTION: WorkflowType.CLINICAL_DECISION,
            EventType.CLINICAL_DECISION: WorkflowType.CLINICAL_DECISION,
            EventType.ADVERSE_EVENT: WorkflowType.ADVERSE_EVENT,
            EventType.PATIENT_ENCOUNTER: WorkflowType.PATIENT_ENCOUNTER
        }

        return event_to_workflow_map.get(event_type, WorkflowType.CLINICAL_DECISION)

    def get_registry_stats(self) -> Dict[str, Any]:
        """Get registry statistics"""
        stats = {
            "registered_processors": len(self.processors),
            "workflow_types": [wt.value for wt in self.processors.keys()],
            "has_default_processor": self.default_processor is not None,
            "processor_stats": {}
        }

        # Get stats from each processor
        for workflow_type, processor in self.processors.items():
            stats["processor_stats"][workflow_type.value] = processor.get_processing_stats()

        return stats
