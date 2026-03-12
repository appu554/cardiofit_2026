"""
Request Analyzer - Multi-Dimensional Property Extraction

This module implements sophisticated request analysis for the Enhanced Orchestrator,
providing multi-dimensional property extraction and clinical context enrichment
as specified in the Enhanced Orchestrator design.

Key Features:
- Multi-dimensional medication property extraction (direct + derived)
- Patient clinical status analysis (demographics + conditions)
- Situational property inference (urgency + workflow context)
- Property enrichment pipeline with clinical intelligence
"""

import logging
from typing import Dict, List, Any, Optional, Set
from datetime import datetime
import time

from ..models.analyzed_request_models import (
    AnalyzedRequest, MedicationProperties, PatientProperties,
    SituationalProperties, EnrichedContext, RiskLevel, AgeGroup,
    OrganFunction, UrgencyLevel
)

logger = logging.getLogger(__name__)


# Data models are now imported from analyzed_request_models.py


class MedicationPropertyEnricher:
    """Enriches medication properties with derived clinical intelligence"""
    
    def __init__(self):
        # High-alert medication list (ISMP)
        self.high_alert_medications = {
            "insulin", "heparin", "warfarin", "chemotherapy", "opioids",
            "neuromuscular_blocking_agents", "concentrated_electrolytes"
        }
        
        # Narrow therapeutic index medications
        self.nti_medications = {
            "warfarin", "digoxin", "lithium", "phenytoin", "theophylline",
            "cyclosporine", "tacrolimus"
        }
        
        # REMS program medications
        self.rems_medications = {
            "clozapine", "isotretinoin", "thalidomide", "lenalidomide"
        }
    
    async def extract_properties(self, medication: Dict[str, Any]) -> MedicationProperties:
        """Extract comprehensive medication properties"""
        
        props = MedicationProperties(
            name=medication.get('name', '').lower(),
            rxnorm_code=medication.get('rxnorm_code'),
            ndc=medication.get('ndc'),
            therapeutic_class=medication.get('therapeutic_class'),
            pharmacologic_class=medication.get('pharmacologic_class')
        )
        
        # Derive clinical properties
        await self._derive_clinical_flags(props)
        await self._assess_risk_levels(props)
        await self._identify_special_considerations(props)
        
        return props
    
    async def _derive_clinical_flags(self, props: MedicationProperties):
        """Derive clinical flags from medication name/class"""
        med_name = props.name.lower()
        
        # High-alert medication check
        props.is_high_alert = any(alert_med in med_name for alert_med in self.high_alert_medications)
        
        # Narrow therapeutic index check
        props.is_narrow_therapeutic_index = any(nti_med in med_name for nti_med in self.nti_medications)
        
        # REMS program check
        props.has_rems_program = any(rems_med in med_name for rems_med in self.rems_medications)
        
        # Therapeutic monitoring requirement
        props.requires_therapeutic_monitoring = (
            props.is_narrow_therapeutic_index or 
            med_name in ["vancomycin", "gentamicin", "tobramycin", "amikacin"]
        )
    
    async def _assess_risk_levels(self, props: MedicationProperties):
        """Assess interaction potential and administration complexity"""
        
        # Interaction potential assessment
        if props.is_narrow_therapeutic_index or "warfarin" in props.name:
            props.interaction_potential = RiskLevel.HIGH
        elif props.is_high_alert:
            props.interaction_potential = RiskLevel.MODERATE
        else:
            props.interaction_potential = RiskLevel.LOW
        
        # Administration complexity assessment
        if props.therapeutic_class in ["chemotherapy", "immunosuppressant"]:
            props.administration_complexity = RiskLevel.HIGH
        elif props.is_high_alert:
            props.administration_complexity = RiskLevel.MODERATE
        else:
            props.administration_complexity = RiskLevel.LOW
    
    async def _identify_special_considerations(self, props: MedicationProperties):
        """Identify special population considerations"""
        
        if "warfarin" in props.name:
            props.special_population_considerations.extend([
                "elderly_bleeding_risk", "renal_impairment_adjustment", 
                "drug_interaction_monitoring"
            ])
        
        if props.therapeutic_class == "chemotherapy":
            props.special_population_considerations.extend([
                "neutropenia_monitoring", "organ_toxicity_screening",
                "fertility_counseling"
            ])


class PatientClinicalAnalyzer:
    """Analyzes patient clinical status and demographics"""
    
    async def analyze_patient(self, patient_id: str) -> PatientProperties:
        """Analyze patient clinical status"""

        # TODO: Integrate with actual patient data sources
        # For now, return mock analysis structure

        props = PatientProperties(patient_id=patient_id)

        # Mock patient data - in real implementation, fetch from FHIR/database
        props.age_years = 65  # Mock data
        props.age_group = self._determine_age_group(props.age_years)
        props.care_setting = "inpatient"

        # Mock clinical conditions
        props.active_conditions = ["hypertension", "diabetes_type_2"]
        props.renal_function = OrganFunction.MILD_IMPAIRMENT
        props.hepatic_function = OrganFunction.NORMAL
        props.cardiac_function = OrganFunction.MILD_IMPAIRMENT

        return props
    
    def _determine_age_group(self, age_years: Optional[int]) -> Optional[AgeGroup]:
        """Determine age group from age in years"""
        if age_years is None:
            return None
        
        if age_years < 0.077:  # 28 days
            return AgeGroup.NEONATE
        elif age_years < 2:
            return AgeGroup.INFANT
        elif age_years < 12:
            return AgeGroup.CHILD
        elif age_years < 18:
            return AgeGroup.ADOLESCENT
        elif age_years < 65:
            return AgeGroup.ADULT
        else:
            return AgeGroup.ELDERLY


class SituationalAnalyzer:
    """Analyzes situational context and workflow properties"""
    
    async def analyze_situation(self, request: Any) -> SituationalProperties:
        """Analyze situational context"""

        props = SituationalProperties()

        # Extract urgency information
        urgency_str = getattr(request, 'urgency', 'routine')
        props.urgency = UrgencyLevel(urgency_str) if urgency_str in [u.value for u in UrgencyLevel] else UrgencyLevel.ROUTINE
        props.stat_order = props.urgency in [UrgencyLevel.EMERGENCY, UrgencyLevel.STAT]
        props.emergency_override = getattr(request, 'emergency_override', False)

        # Calculate time criticality score
        props.time_criticality_score = self._calculate_time_criticality(props)

        # Extract workflow context
        props.prescriber_specialty = getattr(request, 'prescriber_specialty', None)
        props.encounter_type = getattr(request, 'encounter_type', 'outpatient')

        return props
    
    def _calculate_time_criticality(self, props: SituationalProperties) -> float:
        """Calculate time criticality score (0.0 - 1.0)"""
        score = 0.0

        if props.urgency == UrgencyLevel.EMERGENCY:
            score += 0.8
        elif props.urgency == UrgencyLevel.STAT:
            score += 0.9
        elif props.urgency == UrgencyLevel.URGENT:
            score += 0.5
        elif props.urgency == UrgencyLevel.ROUTINE:
            score += 0.1

        if props.stat_order:
            score += 0.2

        if props.emergency_override:
            score += 0.3

        return min(score, 1.0)


class RequestAnalyzer:
    """
    Sophisticated request analyzer with multi-dimensional property extraction
    
    Implements the Request Analyzer component from the Enhanced Orchestrator design:
    - Multi-dimensional medication property extraction
    - Patient clinical status analysis  
    - Situational property inference
    - Property enrichment pipeline
    """
    
    def __init__(self):
        self.medication_enricher = MedicationPropertyEnricher()
        self.patient_analyzer = PatientClinicalAnalyzer()
        self.situation_analyzer = SituationalAnalyzer()
        
        logger.info("Request Analyzer initialized with multi-dimensional analysis capabilities")
    
    async def analyze_request(self, request: Any) -> AnalyzedRequest:
        """
        Comprehensive request analysis with property extraction

        Returns AnalyzedRequest with:
        - medication_properties: Direct + derived properties
        - patient_properties: Demographics + clinical status
        - situational_properties: Urgency + workflow context
        - enriched_context: Inferred clinical context
        """

        start_time = time.time()
        logger.info(f"🔍 Starting multi-dimensional request analysis")

        try:
            # Step 1: Extract medication properties
            medication_props = await self.medication_enricher.extract_properties(
                request.medication
            )
            logger.info(f"📊 Medication analysis: {medication_props.therapeutic_class}, "
                       f"High-alert: {medication_props.is_high_alert}")

            # Step 2: Analyze patient clinical status
            patient_props = await self.patient_analyzer.analyze_patient(
                request.patient_id
            )
            logger.info(f"👤 Patient analysis: {patient_props.age_group.value if patient_props.age_group else 'unknown'}, "
                       f"Conditions: {len(patient_props.active_conditions)}")

            # Step 3: Assess situational context
            situation_props = await self.situation_analyzer.analyze_situation(request)
            logger.info(f"⏰ Situational analysis: {situation_props.urgency.value}, "
                       f"Criticality: {situation_props.time_criticality_score:.2f}")

            # Step 4: Enrich with derived properties
            enriched_context = await self._enrich_clinical_context(
                medication_props, patient_props, situation_props
            )
            logger.info(f"🧠 Context enrichment: Risk {enriched_context.overall_risk_level.value}, "
                       f"Complexity: {enriched_context.complexity_score:.2f}")

            # Step 5: Assess clinical rules requirement
            requires_clinical_rules = self._assess_clinical_rules_need(enriched_context)

            # Calculate analysis duration
            analysis_duration = (time.time() - start_time) * 1000  # Convert to milliseconds

            analyzed_request = AnalyzedRequest(
                original_request=request,
                request_id=getattr(request, 'id', None),
                medication_properties=medication_props,
                patient_properties=patient_props,
                situational_properties=situation_props,
                enriched_context=enriched_context,
                requires_clinical_rules=requires_clinical_rules,
                analysis_duration_ms=analysis_duration
            )

            logger.info(f"✅ Request analysis completed in {analysis_duration:.1f}ms - Clinical rules required: {requires_clinical_rules}")
            return analyzed_request

        except Exception as e:
            logger.error(f"❌ Request analysis failed: {str(e)}")
            raise
    
    async def _enrich_clinical_context(
        self, 
        medication_props: MedicationProperties,
        patient_props: PatientProperties,
        situation_props: SituationalProperties
    ) -> EnrichedContext:
        """Enrich clinical context with inferred properties"""
        
        context = EnrichedContext()
        
        # Calculate overall risk level
        context.overall_risk_level = self._calculate_overall_risk(
            medication_props, patient_props, situation_props
        )
        
        # Calculate complexity score
        context.complexity_score = self._calculate_complexity_score(
            medication_props, patient_props, situation_props
        )
        
        # Identify monitoring requirements
        context.monitoring_requirements = self._identify_monitoring_requirements(
            medication_props, patient_props
        )
        
        # Add clinical flags
        context.clinical_flags = self._generate_clinical_flags(
            medication_props, patient_props, situation_props
        )
        
        return context
    
    def _calculate_overall_risk(
        self, 
        med_props: MedicationProperties,
        patient_props: PatientProperties,
        situation_props: SituationalProperties
    ) -> RiskLevel:
        """Calculate overall clinical risk level"""
        
        risk_score = 0
        
        # Medication risk factors
        if med_props.is_high_alert:
            risk_score += 3
        if med_props.is_narrow_therapeutic_index:
            risk_score += 2
        if med_props.interaction_potential == RiskLevel.HIGH:
            risk_score += 2
        
        # Patient risk factors
        if patient_props.age_group == AgeGroup.ELDERLY:
            risk_score += 2
        if patient_props.renal_function in [OrganFunction.MODERATE_IMPAIRMENT, OrganFunction.SEVERE_IMPAIRMENT]:
            risk_score += 2
        if len(patient_props.active_conditions) > 3:
            risk_score += 1

        # Situational risk factors
        if situation_props.urgency in [UrgencyLevel.EMERGENCY, UrgencyLevel.STAT]:
            risk_score += 1
        
        # Convert score to risk level
        if risk_score >= 7:
            return RiskLevel.CRITICAL
        elif risk_score >= 5:
            return RiskLevel.HIGH
        elif risk_score >= 3:
            return RiskLevel.MODERATE
        else:
            return RiskLevel.LOW
    
    def _calculate_complexity_score(
        self, 
        med_props: MedicationProperties,
        patient_props: PatientProperties,
        situation_props: SituationalProperties
    ) -> float:
        """Calculate clinical complexity score (0.0 - 1.0)"""
        
        score = 0.0
        
        # Medication complexity
        if med_props.administration_complexity == RiskLevel.HIGH:
            score += 0.3
        elif med_props.administration_complexity == RiskLevel.MODERATE:
            score += 0.2
        
        # Patient complexity
        score += len(patient_props.active_conditions) * 0.05
        score += len(patient_props.current_medications) * 0.02
        
        # Age complexity
        if patient_props.age_group in [AgeGroup.ELDERLY, AgeGroup.NEONATE]:
            score += 0.2
        
        # Situational complexity
        score += situation_props.time_criticality_score * 0.2
        
        return min(score, 1.0)
    
    def _identify_monitoring_requirements(
        self, 
        med_props: MedicationProperties,
        patient_props: PatientProperties
    ) -> List[str]:
        """Identify required monitoring based on medication and patient factors"""
        
        monitoring = []
        
        if med_props.requires_therapeutic_monitoring:
            monitoring.append("therapeutic_drug_monitoring")
        
        if med_props.is_narrow_therapeutic_index:
            monitoring.append("frequent_lab_monitoring")
        
        if patient_props.renal_function != OrganFunction.NORMAL:
            monitoring.append("renal_function_monitoring")
        
        if patient_props.age_group == AgeGroup.ELDERLY:
            monitoring.append("enhanced_safety_monitoring")
        
        return monitoring
    
    def _generate_clinical_flags(
        self, 
        med_props: MedicationProperties,
        patient_props: PatientProperties,
        situation_props: SituationalProperties
    ) -> Set[str]:
        """Generate clinical flags for special attention"""
        
        flags = set()
        
        if med_props.is_high_alert:
            flags.add("high_alert_medication")
        
        if patient_props.age_group == AgeGroup.ELDERLY and med_props.is_narrow_therapeutic_index:
            flags.add("elderly_nti_combination")
        
        if situation_props.urgency in [UrgencyLevel.EMERGENCY, UrgencyLevel.STAT] and med_props.interaction_potential == RiskLevel.HIGH:
            flags.add("emergency_high_interaction_risk")
        
        if len(patient_props.active_conditions) > 5:
            flags.add("complex_comorbidities")
        
        return flags
    
    def _assess_clinical_rules_need(self, enriched_context: EnrichedContext) -> bool:
        """Assess if clinical rules engine processing is needed"""
        
        # Clinical rules needed for high-risk or complex scenarios
        return (
            enriched_context.overall_risk_level in [RiskLevel.HIGH, RiskLevel.CRITICAL] or
            enriched_context.complexity_score > 0.6 or
            len(enriched_context.clinical_flags) > 2 or
            len(enriched_context.monitoring_requirements) > 2
        )
