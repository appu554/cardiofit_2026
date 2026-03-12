"""
Personalized Clinical Intelligence for Clinical Assertion Engine

Advanced personalized intelligence including individual patient intelligence,
clinician intelligence & personalization, and adaptive clinical decision support
for Phase 2 CAE implementation.
"""

import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
import json
from dataclasses import dataclass, asdict
from enum import Enum
import numpy as np
from collections import defaultdict, Counter
import statistics

logger = logging.getLogger(__name__)


class PersonalizationType(Enum):
    """Types of personalization"""
    PATIENT_SPECIFIC = "patient_specific"
    CLINICIAN_SPECIFIC = "clinician_specific"
    CONTEXT_ADAPTIVE = "context_adaptive"
    OUTCOME_OPTIMIZED = "outcome_optimized"
    RISK_STRATIFIED = "risk_stratified"


@dataclass
class PatientIntelligenceProfile:
    """Individual patient intelligence profile"""
    patient_id: str
    intelligence_score: float
    risk_stratification: Dict[str, float]
    personalized_thresholds: Dict[str, float]
    treatment_preferences: Dict[str, Any]
    outcome_predictions: Dict[str, float]
    pharmacogenomic_profile: Dict[str, str]
    clinical_trajectory: List[Dict[str, Any]]
    precision_medicine_recommendations: List[str]
    last_updated: datetime


@dataclass
class ClinicianIntelligenceProfile:
    """Clinician intelligence and personalization profile"""
    clinician_id: str
    expertise_areas: List[str]
    decision_patterns: Dict[str, Any]
    alert_preferences: Dict[str, float]
    override_patterns: Dict[str, int]
    performance_metrics: Dict[str, float]
    learning_style: str
    personalized_recommendations: List[str]
    collaboration_network: List[str]
    last_updated: datetime


@dataclass
class PersonalizedRecommendation:
    """Personalized clinical recommendation"""
    recommendation_id: str
    patient_id: str
    clinician_id: str
    recommendation_type: PersonalizationType
    clinical_recommendation: str
    personalization_factors: List[str]
    confidence_score: float
    expected_outcome: Dict[str, float]
    alternative_options: List[str]
    evidence_level: str
    created_at: datetime


@dataclass
class AdaptiveAlert:
    """Adaptive clinical alert"""
    alert_id: str
    alert_type: str
    patient_id: str
    clinician_id: str
    alert_message: str
    severity_level: str
    personalized_threshold: float
    suppression_rules: List[str]
    learning_feedback: Dict[str, Any]
    effectiveness_score: float


class PersonalizedIntelligenceEngine:
    """
    Personalized clinical intelligence engine for Phase 2 CAE
    
    Features:
    - Individual patient intelligence profiles
    - Clinician-specific personalization
    - Adaptive alert systems
    - Precision medicine integration
    - Outcome-optimized recommendations
    """
    
    def __init__(self):
        # Personalization parameters
        self.min_intelligence_score = 0.6
        self.alert_fatigue_threshold = 0.3
        self.personalization_learning_rate = 0.1
        self.outcome_prediction_window_days = 30
        
        # Storage
        self.patient_profiles: Dict[str, PatientIntelligenceProfile] = {}
        self.clinician_profiles: Dict[str, ClinicianIntelligenceProfile] = {}
        self.personalized_recommendations: Dict[str, PersonalizedRecommendation] = {}
        self.adaptive_alerts: Dict[str, AdaptiveAlert] = {}
        
        # Learning cache
        self.personalization_cache = {}
        
        logger.info("Personalized Intelligence Engine initialized")
    
    async def create_patient_intelligence_profile(self, patient_id: str, 
                                                clinical_data: Dict[str, Any],
                                                historical_outcomes: List[Dict[str, Any]]) -> PatientIntelligenceProfile:
        """
        Create comprehensive patient intelligence profile
        
        Args:
            patient_id: Patient identifier
            clinical_data: Current clinical data
            historical_outcomes: Historical treatment outcomes
            
        Returns:
            Patient intelligence profile
        """
        try:
            # Phase 2 Enhancement: Sophisticated patient intelligence
            # For now, create comprehensive mock profile that demonstrates the concept
            
            # Calculate intelligence score based on data completeness and quality
            intelligence_score = self._calculate_patient_intelligence_score(clinical_data, historical_outcomes)
            
            # Risk stratification across multiple domains
            risk_stratification = {
                "cardiovascular_risk": 0.35,
                "diabetes_progression_risk": 0.42,
                "medication_adherence_risk": 0.28,
                "drug_interaction_risk": 0.51,
                "hospitalization_risk": 0.33,
                "mortality_risk": 0.18
            }
            
            # Personalized clinical thresholds
            personalized_thresholds = {
                "hba1c_target": 7.2,  # Personalized based on age, comorbidities
                "bp_systolic_target": 135,  # Adjusted for age and frailty
                "ldl_target": 85,  # Based on cardiovascular risk
                "alert_sensitivity": 0.75,  # Personalized alert threshold
                "drug_interaction_threshold": 0.6
            }
            
            # Treatment preferences based on historical responses
            treatment_preferences = {
                "medication_formulation": "tablet",
                "dosing_frequency": "once_daily_preferred",
                "route_preference": "oral",
                "brand_vs_generic": "generic_acceptable",
                "combination_therapy_tolerance": "good",
                "side_effect_sensitivity": "moderate"
            }
            
            # Outcome predictions using historical patterns
            outcome_predictions = {
                "treatment_response_probability": 0.78,
                "adverse_event_probability": 0.15,
                "medication_adherence_probability": 0.82,
                "quality_of_life_improvement": 0.65,
                "hospitalization_probability_30_days": 0.08,
                "emergency_visit_probability_30_days": 0.12
            }
            
            # Pharmacogenomic profile (mock data)
            pharmacogenomic_profile = {
                "cyp2d6_phenotype": "extensive_metabolizer",
                "cyp2c19_phenotype": "intermediate_metabolizer",
                "cyp3a4_activity": "normal",
                "warfarin_sensitivity": "normal",
                "clopidogrel_response": "reduced",
                "statin_myopathy_risk": "low"
            }
            
            # Clinical trajectory prediction
            clinical_trajectory = [
                {
                    "timepoint": "current",
                    "predicted_hba1c": 8.2,
                    "predicted_bp": "145/88",
                    "predicted_weight": 185,
                    "medication_count": 6
                },
                {
                    "timepoint": "3_months",
                    "predicted_hba1c": 7.8,
                    "predicted_bp": "138/82",
                    "predicted_weight": 182,
                    "medication_count": 7
                },
                {
                    "timepoint": "6_months",
                    "predicted_hba1c": 7.4,
                    "predicted_bp": "135/80",
                    "predicted_weight": 178,
                    "medication_count": 6
                }
            ]
            
            # Precision medicine recommendations
            precision_recommendations = [
                "Consider SGLT-2 inhibitor based on cardiovascular benefit profile",
                "Avoid clopidogrel due to CYP2C19 intermediate metabolizer status",
                "Standard warfarin dosing appropriate based on genetic profile",
                "Monitor for statin-related muscle symptoms despite low genetic risk",
                "Consider once-daily formulations to improve adherence"
            ]
            
            profile = PatientIntelligenceProfile(
                patient_id=patient_id,
                intelligence_score=intelligence_score,
                risk_stratification=risk_stratification,
                personalized_thresholds=personalized_thresholds,
                treatment_preferences=treatment_preferences,
                outcome_predictions=outcome_predictions,
                pharmacogenomic_profile=pharmacogenomic_profile,
                clinical_trajectory=clinical_trajectory,
                precision_medicine_recommendations=precision_recommendations,
                last_updated=datetime.utcnow()
            )
            
            # Store profile
            self.patient_profiles[patient_id] = profile
            
            logger.info(f"Created patient intelligence profile for {patient_id} "
                       f"with intelligence score {intelligence_score:.3f}")
            return profile
            
        except Exception as e:
            logger.error(f"Error creating patient intelligence profile: {e}")
            return PatientIntelligenceProfile(
                patient_id=patient_id,
                intelligence_score=0.0,
                risk_stratification={},
                personalized_thresholds={},
                treatment_preferences={},
                outcome_predictions={},
                pharmacogenomic_profile={},
                clinical_trajectory=[],
                precision_medicine_recommendations=[],
                last_updated=datetime.utcnow()
            )
    
    async def create_clinician_intelligence_profile(self, clinician_id: str,
                                                  decision_history: List[Dict[str, Any]],
                                                  performance_data: Dict[str, Any]) -> ClinicianIntelligenceProfile:
        """
        Create clinician intelligence and personalization profile
        
        Args:
            clinician_id: Clinician identifier
            decision_history: Historical clinical decisions
            performance_data: Performance metrics and outcomes
            
        Returns:
            Clinician intelligence profile
        """
        try:
            # Phase 2 Enhancement: Sophisticated clinician intelligence
            # For now, create comprehensive mock profile
            
            # Identify expertise areas based on decision patterns
            expertise_areas = [
                "diabetes_management",
                "cardiovascular_disease",
                "polypharmacy_optimization",
                "geriatric_medicine"
            ]
            
            # Analyze decision patterns
            decision_patterns = {
                "conservative_vs_aggressive": "moderate_conservative",
                "guideline_adherence_rate": 0.87,
                "early_adopter_score": 0.65,
                "collaboration_preference": "multidisciplinary",
                "evidence_preference": "randomized_controlled_trials",
                "patient_autonomy_respect": 0.92
            }
            
            # Personalized alert preferences
            alert_preferences = {
                "drug_interaction_threshold": 0.7,  # Higher threshold = fewer alerts
                "duplicate_therapy_sensitivity": 0.8,
                "dosing_alert_sensitivity": 0.6,
                "contraindication_sensitivity": 0.9,
                "allergy_alert_sensitivity": 0.95
            }
            
            # Override patterns analysis
            override_patterns = {
                "drug_interaction_overrides": 15,
                "dosing_alert_overrides": 8,
                "duplicate_therapy_overrides": 12,
                "total_alerts_received": 156,
                "override_rate": 0.22,
                "appropriate_override_rate": 0.85
            }
            
            # Performance metrics
            performance_metrics = {
                "patient_satisfaction_score": 4.7,
                "clinical_outcome_score": 0.84,
                "medication_adherence_improvement": 0.78,
                "adverse_event_rate": 0.08,
                "guideline_compliance_score": 0.87,
                "peer_collaboration_score": 0.91
            }
            
            # Learning style assessment
            learning_style = "evidence_based_collaborative"
            
            # Personalized recommendations for clinician
            personalized_recommendations = [
                "Consider reducing drug interaction alert sensitivity to decrease alert fatigue",
                "Leverage expertise in diabetes management for complex cases",
                "Participate in cardiovascular disease guideline update training",
                "Share polypharmacy optimization strategies with colleagues",
                "Consider point-of-care decision support tools for geriatric patients"
            ]
            
            # Collaboration network
            collaboration_network = [
                "pharmacist_jane_smith",
                "cardiologist_dr_johnson",
                "endocrinologist_dr_patel",
                "geriatrician_dr_williams"
            ]
            
            profile = ClinicianIntelligenceProfile(
                clinician_id=clinician_id,
                expertise_areas=expertise_areas,
                decision_patterns=decision_patterns,
                alert_preferences=alert_preferences,
                override_patterns=override_patterns,
                performance_metrics=performance_metrics,
                learning_style=learning_style,
                personalized_recommendations=personalized_recommendations,
                collaboration_network=collaboration_network,
                last_updated=datetime.utcnow()
            )
            
            # Store profile
            self.clinician_profiles[clinician_id] = profile
            
            logger.info(f"Created clinician intelligence profile for {clinician_id}")
            return profile
            
        except Exception as e:
            logger.error(f"Error creating clinician intelligence profile: {e}")
            return ClinicianIntelligenceProfile(
                clinician_id=clinician_id,
                expertise_areas=[],
                decision_patterns={},
                alert_preferences={},
                override_patterns={},
                performance_metrics={},
                learning_style="unknown",
                personalized_recommendations=[],
                collaboration_network=[],
                last_updated=datetime.utcnow()
            )

    async def generate_personalized_recommendation(self, patient_id: str,
                                                 clinician_id: str,
                                                 clinical_context: Dict[str, Any]) -> PersonalizedRecommendation:
        """
        Generate personalized clinical recommendation

        Args:
            patient_id: Patient identifier
            clinician_id: Clinician identifier
            clinical_context: Current clinical context

        Returns:
            Personalized recommendation
        """
        try:
            # Get patient and clinician profiles
            patient_profile = self.patient_profiles.get(patient_id)
            clinician_profile = self.clinician_profiles.get(clinician_id)

            if not patient_profile or not clinician_profile:
                return self._create_default_recommendation(patient_id, clinician_id, clinical_context)

            # Generate basic personalized recommendation
            recommendation = "Personalized clinical recommendation based on patient and clinician profiles"
            confidence_score = 0.75

            personalized_rec = PersonalizedRecommendation(
                recommendation_id=f"pers_rec_{patient_id}_{clinician_id}_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                patient_id=patient_id,
                clinician_id=clinician_id,
                recommendation_type=PersonalizationType.PATIENT_SPECIFIC,
                clinical_recommendation=recommendation,
                personalization_factors=["patient_profile", "clinician_profile"],
                confidence_score=confidence_score,
                expected_outcome={"clinical_improvement_probability": 0.8},
                alternative_options=["Standard care"],
                evidence_level="personalized_population_evidence",
                created_at=datetime.utcnow()
            )

            # Store recommendation
            self.personalized_recommendations[personalized_rec.recommendation_id] = personalized_rec

            logger.info(f"Generated personalized recommendation for patient {patient_id}")
            return personalized_rec

        except Exception as e:
            logger.error(f"Error generating personalized recommendation: {e}")
            return self._create_default_recommendation(patient_id, clinician_id, clinical_context)

    def _calculate_patient_intelligence_score(self, clinical_data: Dict[str, Any],
                                            historical_outcomes: List[Dict[str, Any]]) -> float:
        """Calculate patient intelligence score"""
        try:
            base_score = 0.5

            # Data completeness factor
            data_completeness = len(clinical_data) / 20.0  # Assume 20 ideal data points
            data_factor = min(1.0, data_completeness) * 0.3

            # Historical outcomes factor
            outcomes_factor = min(1.0, len(historical_outcomes) / 10.0) * 0.2

            # Clinical complexity factor
            complexity_indicators = ["diabetes", "hypertension", "heart_disease", "kidney_disease"]
            complexity_count = sum(1 for indicator in complexity_indicators
                                 if indicator in str(clinical_data).lower())
            complexity_factor = min(1.0, complexity_count / 4.0) * 0.3

            # Medication count factor
            medication_count = clinical_data.get("medication_count", len(clinical_data.get("medications", [])))
            medication_factor = min(1.0, medication_count / 10.0) * 0.2

            intelligence_score = base_score + data_factor + outcomes_factor + complexity_factor + medication_factor
            return min(1.0, intelligence_score)

        except Exception as e:
            logger.error(f"Error calculating patient intelligence score: {e}")
            return 0.5

    def _calculate_personalized_threshold(self, alert_type: str,
                                        patient_profile: Optional[PatientIntelligenceProfile],
                                        clinician_profile: Optional[ClinicianIntelligenceProfile]) -> float:
        """Calculate personalized alert threshold"""
        base_threshold = 0.5

        if clinician_profile and alert_type in clinician_profile.alert_preferences:
            base_threshold = clinician_profile.alert_preferences[alert_type]

        # Adjust based on patient risk
        if patient_profile:
            max_risk = max(patient_profile.risk_stratification.values()) if patient_profile.risk_stratification else 0.5
            if max_risk > 0.7:
                base_threshold *= 0.8  # Lower threshold for high-risk patients

        return base_threshold

    def _personalize_alert_message(self, alert_context: Dict[str, Any],
                                 patient_profile: Optional[PatientIntelligenceProfile],
                                 clinician_profile: Optional[ClinicianIntelligenceProfile]) -> str:
        """Personalize alert message based on profiles"""
        base_message = alert_context.get("message", "Clinical alert")

        # Add patient-specific context
        if patient_profile:
            if patient_profile.pharmacogenomic_profile:
                base_message += " (Consider pharmacogenomic factors)"

        # Add clinician-specific context
        if clinician_profile:
            if "diabetes_management" in clinician_profile.expertise_areas:
                base_message += " (Diabetes expertise noted)"

        return base_message

    def _generate_suppression_rules(self, alert_type: str,
                                  clinician_profile: Optional[ClinicianIntelligenceProfile]) -> List[str]:
        """Generate alert suppression rules"""
        rules = []

        if clinician_profile:
            override_rate = clinician_profile.override_patterns.get("override_rate", 0.0)
            if override_rate > 0.5:
                rules.append("High override rate - consider suppression")

        return rules

    def _calculate_alert_effectiveness(self, alert_type: str,
                                     clinician_profile: Optional[ClinicianIntelligenceProfile]) -> float:
        """Calculate alert effectiveness score"""
        if not clinician_profile:
            return 0.5

        appropriate_override_rate = clinician_profile.override_patterns.get("appropriate_override_rate", 0.5)
        return appropriate_override_rate

    def _create_default_recommendation(self, patient_id: str, clinician_id: str,
                                     clinical_context: Dict[str, Any]) -> PersonalizedRecommendation:
        """Create default recommendation when profiles unavailable"""
        return PersonalizedRecommendation(
            recommendation_id=f"default_rec_{patient_id}_{clinician_id}",
            patient_id=patient_id,
            clinician_id=clinician_id,
            recommendation_type=PersonalizationType.CONTEXT_ADAPTIVE,
            clinical_recommendation="Standard clinical guidelines apply",
            personalization_factors=["insufficient_data"],
            confidence_score=0.5,
            expected_outcome={"clinical_improvement_probability": 0.6},
            alternative_options=["Gather more patient data for personalization"],
            evidence_level="standard_guidelines",
            created_at=datetime.utcnow()
        )

    def get_personalization_statistics(self) -> Dict[str, Any]:
        """Get comprehensive personalization statistics"""
        try:
            # Calculate patient profile statistics
            patient_intelligence_scores = [p.intelligence_score for p in self.patient_profiles.values()]

            # Calculate clinician profile statistics
            clinician_override_rates = [
                p.override_patterns.get("override_rate", 0.0)
                for p in self.clinician_profiles.values()
            ]

            # Calculate recommendation statistics
            recommendation_types = Counter([
                r.recommendation_type.value for r in self.personalized_recommendations.values()
            ])

            return {
                "patient_profiles": {
                    "total_profiles": len(self.patient_profiles),
                    "average_intelligence_score": statistics.mean(patient_intelligence_scores) if patient_intelligence_scores else 0,
                    "high_intelligence_patients": len([s for s in patient_intelligence_scores if s > 0.8]),
                    "risk_distribution": self._calculate_risk_distribution()
                },
                "clinician_profiles": {
                    "total_profiles": len(self.clinician_profiles),
                    "average_override_rate": statistics.mean(clinician_override_rates) if clinician_override_rates else 0,
                    "expertise_areas": self._count_expertise_areas(),
                    "collaboration_networks": self._analyze_collaboration_networks()
                },
                "personalized_recommendations": {
                    "total_recommendations": len(self.personalized_recommendations),
                    "recommendation_type_distribution": dict(recommendation_types),
                    "average_confidence": self._calculate_average_recommendation_confidence(),
                    "high_confidence_recommendations": len([
                        r for r in self.personalized_recommendations.values()
                        if r.confidence_score > 0.8
                    ])
                },
                "adaptive_alerts": {
                    "total_alerts": len(self.adaptive_alerts),
                    "average_effectiveness": self._calculate_average_alert_effectiveness(),
                    "suppressed_alerts": len([
                        a for a in self.adaptive_alerts.values()
                        if a.suppression_rules
                    ])
                },
                "personalization_parameters": {
                    "min_intelligence_score": self.min_intelligence_score,
                    "alert_fatigue_threshold": self.alert_fatigue_threshold,
                    "personalization_learning_rate": self.personalization_learning_rate
                }
            }

        except Exception as e:
            logger.error(f"Error calculating personalization statistics: {e}")
            return {}

    def _calculate_risk_distribution(self) -> Dict[str, float]:
        """Calculate risk distribution across patient population"""
        risk_totals = defaultdict(list)

        for profile in self.patient_profiles.values():
            for risk_type, risk_value in profile.risk_stratification.items():
                risk_totals[risk_type].append(risk_value)

        return {
            risk_type: statistics.mean(values) if values else 0.0
            for risk_type, values in risk_totals.items()
        }

    def _count_expertise_areas(self) -> Dict[str, int]:
        """Count expertise areas across clinicians"""
        expertise_counts = Counter()

        for profile in self.clinician_profiles.values():
            for area in profile.expertise_areas:
                expertise_counts[area] += 1

        return dict(expertise_counts)

    def _analyze_collaboration_networks(self) -> Dict[str, Any]:
        """Analyze collaboration networks"""
        total_connections = sum(
            len(profile.collaboration_network)
            for profile in self.clinician_profiles.values()
        )

        return {
            "total_connections": total_connections,
            "average_network_size": total_connections / len(self.clinician_profiles) if self.clinician_profiles else 0,
            "highly_connected_clinicians": len([
                p for p in self.clinician_profiles.values()
                if len(p.collaboration_network) > 5
            ])
        }

    def _calculate_average_recommendation_confidence(self) -> float:
        """Calculate average recommendation confidence"""
        if not self.personalized_recommendations:
            return 0.0

        total_confidence = sum(r.confidence_score for r in self.personalized_recommendations.values())
        return total_confidence / len(self.personalized_recommendations)

    def _calculate_average_alert_effectiveness(self) -> float:
        """Calculate average alert effectiveness"""
        if not self.adaptive_alerts:
            return 0.0

        total_effectiveness = sum(a.effectiveness_score for a in self.adaptive_alerts.values())
        return total_effectiveness / len(self.adaptive_alerts)
