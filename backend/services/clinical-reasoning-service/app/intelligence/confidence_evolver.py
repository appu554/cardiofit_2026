"""
Confidence Evolver for Clinical Assertion Engine

Advanced confidence scoring algorithms that evolve based on clinical outcomes,
population statistics, and real-world evidence to provide dynamic confidence
assessment for clinical assertions.
"""

import logging
import math
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)


class ConfidenceFactors(Enum):
    """Factors that influence confidence scoring"""
    EVIDENCE_STRENGTH = "evidence_strength"
    POPULATION_SIZE = "population_size"
    OUTCOME_CORRELATION = "outcome_correlation"
    CLINICIAN_AGREEMENT = "clinician_agreement"
    TEMPORAL_CONSISTENCY = "temporal_consistency"
    CROSS_VALIDATION = "cross_validation"
    EXTERNAL_VALIDATION = "external_validation"


@dataclass
class ConfidenceEvidence:
    """Evidence contributing to confidence calculation"""
    evidence_type: str
    strength: float  # 0.0 to 1.0
    weight: float   # Importance weight
    source: str
    timestamp: datetime
    metadata: Dict[str, Any] = None


@dataclass
class ConfidenceScore:
    """Comprehensive confidence score with breakdown"""
    overall_confidence: float
    base_confidence: float
    evidence_boost: float
    population_factor: float
    temporal_factor: float
    uncertainty_penalty: float
    confidence_interval: Tuple[float, float]
    evidence_breakdown: Dict[str, float]
    last_updated: datetime


class ConfidenceEvolver:
    """
    Advanced confidence evolution system for clinical assertions
    
    Features:
    - Bayesian confidence updating
    - Population-based confidence adjustment
    - Temporal confidence decay/reinforcement
    - Evidence-weighted confidence scoring
    - Uncertainty quantification
    - Cross-validation confidence boosting
    """
    
    def __init__(self):
        # Confidence evolution parameters
        self.base_confidence_threshold = 0.5
        self.evidence_decay_rate = 0.95  # Daily decay for old evidence
        self.population_boost_threshold = 100  # Minimum population for boost
        self.temporal_window_days = 90  # Relevance window for evidence
        
        # Evidence weights by type
        self.evidence_weights = {
            "clinical_trial": 1.0,
            "real_world_evidence": 0.8,
            "expert_consensus": 0.7,
            "case_series": 0.6,
            "case_report": 0.4,
            "in_vitro": 0.3,
            "theoretical": 0.2
        }
        
        # Confidence tracking
        self.confidence_history: Dict[str, List[ConfidenceScore]] = {}
        self.evidence_repository: Dict[str, List[ConfidenceEvidence]] = {}
        
        logger.info("Confidence Evolver initialized")
    
    async def evolve_confidence(self, assertion_id: str, 
                              new_evidence: List[ConfidenceEvidence],
                              population_data: Dict[str, Any] = None) -> ConfidenceScore:
        """
        Evolve confidence score based on new evidence and population data
        
        Args:
            assertion_id: Unique assertion identifier
            new_evidence: List of new evidence to incorporate
            population_data: Population statistics for context
            
        Returns:
            Updated confidence score
        """
        try:
            # Add new evidence to repository
            if assertion_id not in self.evidence_repository:
                self.evidence_repository[assertion_id] = []
            
            self.evidence_repository[assertion_id].extend(new_evidence)
            
            # Get current evidence (within temporal window)
            current_evidence = self._get_current_evidence(assertion_id)
            
            # Calculate base confidence from evidence
            base_confidence = await self._calculate_base_confidence(current_evidence)
            
            # Apply population-based adjustments
            population_factor = self._calculate_population_factor(population_data)
            
            # Apply temporal factors
            temporal_factor = self._calculate_temporal_factor(current_evidence)
            
            # Calculate evidence boost
            evidence_boost = self._calculate_evidence_boost(current_evidence)
            
            # Calculate uncertainty penalty
            uncertainty_penalty = self._calculate_uncertainty_penalty(current_evidence)
            
            # Combine factors using weighted geometric mean
            overall_confidence = self._combine_confidence_factors(
                base_confidence, population_factor, temporal_factor, 
                evidence_boost, uncertainty_penalty
            )
            
            # Calculate confidence interval
            confidence_interval = self._calculate_confidence_interval(
                overall_confidence, current_evidence
            )
            
            # Create evidence breakdown
            evidence_breakdown = self._create_evidence_breakdown(current_evidence)
            
            # Create confidence score
            confidence_score = ConfidenceScore(
                overall_confidence=overall_confidence,
                base_confidence=base_confidence,
                evidence_boost=evidence_boost,
                population_factor=population_factor,
                temporal_factor=temporal_factor,
                uncertainty_penalty=uncertainty_penalty,
                confidence_interval=confidence_interval,
                evidence_breakdown=evidence_breakdown,
                last_updated=datetime.utcnow()
            )
            
            # Store in history
            if assertion_id not in self.confidence_history:
                self.confidence_history[assertion_id] = []
            
            self.confidence_history[assertion_id].append(confidence_score)
            
            # Keep only last 100 scores
            if len(self.confidence_history[assertion_id]) > 100:
                self.confidence_history[assertion_id] = self.confidence_history[assertion_id][-100:]
            
            logger.info(f"Evolved confidence for {assertion_id}: {overall_confidence:.3f} "
                       f"(base: {base_confidence:.3f}, evidence: {len(current_evidence)})")
            
            return confidence_score
            
        except Exception as e:
            logger.error(f"Error evolving confidence: {e}")
            # Return default confidence score
            return ConfidenceScore(
                overall_confidence=0.5,
                base_confidence=0.5,
                evidence_boost=0.0,
                population_factor=1.0,
                temporal_factor=1.0,
                uncertainty_penalty=0.0,
                confidence_interval=(0.3, 0.7),
                evidence_breakdown={},
                last_updated=datetime.utcnow()
            )
    
    async def update_from_outcome(self, assertion_id: str, outcome_positive: bool,
                                outcome_strength: float = 1.0,
                                clinical_context: Dict[str, Any] = None):
        """
        Update confidence based on clinical outcome
        
        Args:
            assertion_id: Assertion that led to outcome
            outcome_positive: Whether outcome was positive
            outcome_strength: Strength of the outcome evidence
            clinical_context: Context of the outcome
        """
        try:
            # Create outcome evidence
            outcome_evidence = ConfidenceEvidence(
                evidence_type="clinical_outcome",
                strength=outcome_strength if outcome_positive else -outcome_strength,
                weight=1.0,  # Outcomes have high weight
                source="clinical_practice",
                timestamp=datetime.utcnow(),
                metadata={
                    "outcome_positive": outcome_positive,
                    "clinical_context": clinical_context or {}
                }
            )
            
            # Evolve confidence with outcome evidence
            await self.evolve_confidence(assertion_id, [outcome_evidence])
            
            logger.info(f"Updated confidence from outcome for {assertion_id}: "
                       f"positive={outcome_positive}, strength={outcome_strength}")
            
        except Exception as e:
            logger.error(f"Error updating confidence from outcome: {e}")
    
    async def cross_validate_confidence(self, assertion_id: str,
                                      validation_results: List[Dict[str, Any]]) -> float:
        """
        Cross-validate confidence using multiple validation approaches
        
        Args:
            assertion_id: Assertion to validate
            validation_results: Results from different validation methods
            
        Returns:
            Cross-validated confidence boost factor
        """
        try:
            if not validation_results:
                return 1.0
            
            # Calculate validation scores
            validation_scores = []
            for result in validation_results:
                method = result.get("method", "unknown")
                accuracy = result.get("accuracy", 0.5)
                sample_size = result.get("sample_size", 1)
                
                # Weight by sample size and method reliability
                method_weight = {
                    "holdout_validation": 1.0,
                    "cross_validation": 0.9,
                    "bootstrap": 0.8,
                    "expert_review": 0.7,
                    "peer_review": 0.6
                }.get(method, 0.5)
                
                # Weight by sample size (logarithmic scaling)
                size_weight = min(1.0, math.log10(sample_size + 1) / 3.0)
                
                weighted_score = accuracy * method_weight * size_weight
                validation_scores.append(weighted_score)
            
            # Calculate overall validation boost
            if validation_scores:
                avg_validation = sum(validation_scores) / len(validation_scores)
                # Convert to boost factor (1.0 = no change, >1.0 = boost, <1.0 = penalty)
                validation_boost = 0.8 + (avg_validation * 0.4)  # Range: 0.8 to 1.2
            else:
                validation_boost = 1.0
            
            # Create validation evidence
            validation_evidence = ConfidenceEvidence(
                evidence_type="cross_validation",
                strength=avg_validation,
                weight=0.8,
                source="validation_framework",
                timestamp=datetime.utcnow(),
                metadata={
                    "validation_methods": [r.get("method") for r in validation_results],
                    "validation_scores": validation_scores,
                    "boost_factor": validation_boost
                }
            )
            
            # Add to evidence repository
            if assertion_id not in self.evidence_repository:
                self.evidence_repository[assertion_id] = []
            
            self.evidence_repository[assertion_id].append(validation_evidence)
            
            logger.info(f"Cross-validated confidence for {assertion_id}: "
                       f"boost_factor={validation_boost:.3f}")
            
            return validation_boost
            
        except Exception as e:
            logger.error(f"Error in cross-validation: {e}")
            return 1.0
    
    def _get_current_evidence(self, assertion_id: str) -> List[ConfidenceEvidence]:
        """Get evidence within temporal window"""
        if assertion_id not in self.evidence_repository:
            return []
        
        cutoff_date = datetime.utcnow() - timedelta(days=self.temporal_window_days)
        current_evidence = []
        
        for evidence in self.evidence_repository[assertion_id]:
            # Apply temporal decay
            age_days = (datetime.utcnow() - evidence.timestamp).days
            decay_factor = self.evidence_decay_rate ** age_days
            
            if evidence.timestamp >= cutoff_date and decay_factor > 0.1:
                # Create decayed evidence
                decayed_evidence = ConfidenceEvidence(
                    evidence_type=evidence.evidence_type,
                    strength=evidence.strength * decay_factor,
                    weight=evidence.weight,
                    source=evidence.source,
                    timestamp=evidence.timestamp,
                    metadata=evidence.metadata
                )
                current_evidence.append(decayed_evidence)
        
        return current_evidence
    
    async def _calculate_base_confidence(self, evidence: List[ConfidenceEvidence]) -> float:
        """Calculate base confidence from evidence using Bayesian updating"""
        if not evidence:
            return self.base_confidence_threshold
        
        # Start with prior
        prior_confidence = self.base_confidence_threshold
        
        # Bayesian updating for each piece of evidence
        current_confidence = prior_confidence
        
        for ev in evidence:
            # Get evidence weight
            evidence_weight = self.evidence_weights.get(ev.evidence_type, 0.5)
            
            # Calculate likelihood ratio
            if ev.strength > 0:
                # Positive evidence
                likelihood_ratio = 1 + (ev.strength * evidence_weight * ev.weight)
            else:
                # Negative evidence
                likelihood_ratio = 1 / (1 + abs(ev.strength) * evidence_weight * ev.weight)
            
            # Bayesian update
            odds = current_confidence / (1 - current_confidence)
            updated_odds = odds * likelihood_ratio
            current_confidence = updated_odds / (1 + updated_odds)
            
            # Ensure bounds
            current_confidence = max(0.01, min(0.99, current_confidence))
        
        return current_confidence
    
    def _calculate_population_factor(self, population_data: Dict[str, Any]) -> float:
        """Calculate population-based confidence adjustment"""
        if not population_data:
            return 1.0
        
        population_size = population_data.get("population_size", 0)
        
        if population_size < 10:
            return 0.8  # Small population penalty
        elif population_size < self.population_boost_threshold:
            return 0.9  # Moderate population
        else:
            # Large population boost (logarithmic scaling)
            boost = 1.0 + (math.log10(population_size / self.population_boost_threshold) * 0.1)
            return min(1.3, boost)  # Cap at 30% boost
    
    def _calculate_temporal_factor(self, evidence: List[ConfidenceEvidence]) -> float:
        """Calculate temporal consistency factor"""
        if len(evidence) < 2:
            return 1.0
        
        # Sort evidence by timestamp
        sorted_evidence = sorted(evidence, key=lambda x: x.timestamp)
        
        # Calculate temporal consistency
        consistency_scores = []
        for i in range(1, len(sorted_evidence)):
            prev_strength = sorted_evidence[i-1].strength
            curr_strength = sorted_evidence[i].strength
            
            # Check if evidence is consistent (same sign)
            if (prev_strength > 0 and curr_strength > 0) or (prev_strength < 0 and curr_strength < 0):
                consistency_scores.append(1.0)
            else:
                consistency_scores.append(0.0)
        
        if consistency_scores:
            consistency = sum(consistency_scores) / len(consistency_scores)
            return 0.8 + (consistency * 0.4)  # Range: 0.8 to 1.2
        
        return 1.0
    
    def _calculate_evidence_boost(self, evidence: List[ConfidenceEvidence]) -> float:
        """Calculate evidence quantity and quality boost"""
        if not evidence:
            return 0.0
        
        # Quantity boost (logarithmic)
        quantity_boost = math.log10(len(evidence) + 1) * 0.1
        
        # Quality boost (average evidence strength)
        quality_scores = [abs(ev.strength) * ev.weight for ev in evidence]
        quality_boost = (sum(quality_scores) / len(quality_scores)) * 0.1 if quality_scores else 0.0
        
        return min(0.3, quantity_boost + quality_boost)  # Cap at 30% boost
    
    def _calculate_uncertainty_penalty(self, evidence: List[ConfidenceEvidence]) -> float:
        """Calculate uncertainty penalty based on evidence variability"""
        if len(evidence) < 2:
            return 0.0
        
        # Calculate variance in evidence strengths
        strengths = [ev.strength for ev in evidence]
        mean_strength = sum(strengths) / len(strengths)
        variance = sum((s - mean_strength) ** 2 for s in strengths) / len(strengths)
        
        # Convert variance to penalty (higher variance = higher penalty)
        uncertainty_penalty = min(0.2, variance * 0.1)  # Cap at 20% penalty
        
        return uncertainty_penalty
    
    def _combine_confidence_factors(self, base_confidence: float, population_factor: float,
                                  temporal_factor: float, evidence_boost: float,
                                  uncertainty_penalty: float) -> float:
        """Combine confidence factors using weighted geometric mean"""
        
        # Apply multiplicative factors
        adjusted_confidence = base_confidence * population_factor * temporal_factor
        
        # Apply additive boost
        adjusted_confidence += evidence_boost
        
        # Apply penalty
        adjusted_confidence -= uncertainty_penalty
        
        # Ensure bounds
        return max(0.01, min(0.99, adjusted_confidence))
    
    def _calculate_confidence_interval(self, confidence: float, 
                                     evidence: List[ConfidenceEvidence]) -> Tuple[float, float]:
        """Calculate confidence interval based on evidence uncertainty"""
        if not evidence:
            return (max(0.0, confidence - 0.2), min(1.0, confidence + 0.2))
        
        # Calculate standard error based on evidence count and variability
        n = len(evidence)
        strengths = [ev.strength for ev in evidence]
        
        if len(strengths) > 1:
            std_dev = np.std(strengths)
            standard_error = std_dev / math.sqrt(n)
        else:
            standard_error = 0.1  # Default uncertainty
        
        # 95% confidence interval (approximately 2 standard errors)
        margin_of_error = 2 * standard_error
        
        lower_bound = max(0.0, confidence - margin_of_error)
        upper_bound = min(1.0, confidence + margin_of_error)
        
        return (lower_bound, upper_bound)
    
    def _create_evidence_breakdown(self, evidence: List[ConfidenceEvidence]) -> Dict[str, float]:
        """Create breakdown of evidence contributions"""
        breakdown = {}
        
        for ev in evidence:
            if ev.evidence_type not in breakdown:
                breakdown[ev.evidence_type] = 0.0
            
            contribution = abs(ev.strength) * ev.weight
            breakdown[ev.evidence_type] += contribution
        
        # Normalize to percentages
        total_contribution = sum(breakdown.values())
        if total_contribution > 0:
            breakdown = {k: (v / total_contribution) * 100 for k, v in breakdown.items()}
        
        return breakdown
    
    def get_confidence_trend(self, assertion_id: str, days: int = 30) -> List[Dict[str, Any]]:
        """Get confidence trend over time"""
        if assertion_id not in self.confidence_history:
            return []
        
        cutoff_date = datetime.utcnow() - timedelta(days=days)
        
        trend_data = []
        for score in self.confidence_history[assertion_id]:
            if score.last_updated >= cutoff_date:
                trend_data.append({
                    "timestamp": score.last_updated.isoformat(),
                    "confidence": score.overall_confidence,
                    "evidence_count": len(score.evidence_breakdown),
                    "confidence_interval": score.confidence_interval
                })
        
        return sorted(trend_data, key=lambda x: x["timestamp"])
    
    def get_confidence_statistics(self) -> Dict[str, Any]:
        """Get overall confidence evolution statistics"""
        total_assertions = len(self.confidence_history)
        total_evidence = sum(len(evidence) for evidence in self.evidence_repository.values())
        
        # Calculate average confidence
        all_scores = []
        for scores in self.confidence_history.values():
            if scores:
                all_scores.append(scores[-1].overall_confidence)
        
        avg_confidence = sum(all_scores) / len(all_scores) if all_scores else 0.0
        
        return {
            "total_assertions": total_assertions,
            "total_evidence_pieces": total_evidence,
            "average_confidence": round(avg_confidence, 3),
            "confidence_distribution": self._get_confidence_distribution(all_scores),
            "evidence_types": self._get_evidence_type_distribution()
        }
    
    def _get_confidence_distribution(self, scores: List[float]) -> Dict[str, int]:
        """Get distribution of confidence scores"""
        distribution = {
            "very_low": 0,    # 0.0-0.2
            "low": 0,         # 0.2-0.4
            "moderate": 0,    # 0.4-0.6
            "high": 0,        # 0.6-0.8
            "very_high": 0    # 0.8-1.0
        }
        
        for score in scores:
            if score < 0.2:
                distribution["very_low"] += 1
            elif score < 0.4:
                distribution["low"] += 1
            elif score < 0.6:
                distribution["moderate"] += 1
            elif score < 0.8:
                distribution["high"] += 1
            else:
                distribution["very_high"] += 1
        
        return distribution
    
    def _get_evidence_type_distribution(self) -> Dict[str, int]:
        """Get distribution of evidence types"""
        type_counts = {}
        
        for evidence_list in self.evidence_repository.values():
            for evidence in evidence_list:
                evidence_type = evidence.evidence_type
                type_counts[evidence_type] = type_counts.get(evidence_type, 0) + 1
        
        return type_counts
