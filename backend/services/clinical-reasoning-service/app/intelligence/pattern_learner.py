"""
Pattern Learner for Clinical Assertion Engine

Machine learning from clinical patterns to discover new relationships,
predict outcomes, and generate intelligent clinical recommendations
based on historical data and real-world evidence.
"""

import logging
import json
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
from enum import Enum
from collections import defaultdict, Counter
import math

logger = logging.getLogger(__name__)


class PatternType(Enum):
    """Types of clinical patterns"""
    MEDICATION_SEQUENCE = "medication_sequence"
    DRUG_INTERACTION = "drug_interaction"
    ADVERSE_EVENT = "adverse_event"
    THERAPEUTIC_RESPONSE = "therapeutic_response"
    PATIENT_SIMILARITY = "patient_similarity"
    TEMPORAL_CORRELATION = "temporal_correlation"


@dataclass
class ClinicalPattern:
    """Discovered clinical pattern"""
    pattern_id: str
    pattern_type: PatternType
    entities: List[str]
    frequency: int
    confidence: float
    support: float
    lift: float
    conviction: float
    clinical_significance: str
    discovered_at: datetime
    last_validated: datetime
    validation_score: float
    metadata: Dict[str, Any]


@dataclass
class PatternPrediction:
    """Prediction based on learned patterns"""
    prediction_id: str
    predicted_outcome: str
    confidence: float
    supporting_patterns: List[str]
    risk_factors: List[str]
    recommendations: List[str]
    explanation: str
    created_at: datetime


class PatternLearner:
    """
    Advanced pattern learning system for clinical intelligence
    
    Features:
    - Association rule mining for drug interactions
    - Sequential pattern mining for medication sequences
    - Temporal pattern analysis for outcome prediction
    - Patient similarity clustering
    - Anomaly detection for adverse events
    - Predictive modeling for clinical outcomes
    """
    
    def __init__(self, min_support: float = 0.1, min_confidence: float = 0.6):
        self.min_support = min_support
        self.min_confidence = min_confidence
        
        # Pattern storage
        self.discovered_patterns: Dict[str, ClinicalPattern] = {}
        self.pattern_predictions: Dict[str, PatternPrediction] = {}
        
        # Learning data
        self.clinical_transactions: List[Dict[str, Any]] = []
        self.patient_profiles: Dict[str, Dict[str, Any]] = {}
        self.outcome_history: List[Dict[str, Any]] = []
        
        # Pattern mining parameters
        self.max_pattern_length = 5
        self.temporal_window_hours = 72
        self.similarity_threshold = 0.8
        
        # Learning statistics
        self.learning_stats = {
            "patterns_discovered": 0,
            "predictions_made": 0,
            "validation_accuracy": 0.0,
            "last_learning_run": None
        }
        
        logger.info("Pattern Learner initialized")
    
    async def learn_from_clinical_data(self, clinical_data: List[Dict[str, Any]]) -> List[ClinicalPattern]:
        """
        Learn patterns from clinical data
        
        Args:
            clinical_data: List of clinical transactions/events
            
        Returns:
            List of discovered patterns
        """
        try:
            # Store clinical data
            self.clinical_transactions.extend(clinical_data)
            
            # Discover different types of patterns
            discovered_patterns = []
            
            # 1. Association rule mining for drug interactions
            interaction_patterns = await self._mine_interaction_patterns(clinical_data)
            discovered_patterns.extend(interaction_patterns)
            
            # 2. Sequential pattern mining for medication sequences
            sequence_patterns = await self._mine_sequence_patterns(clinical_data)
            discovered_patterns.extend(sequence_patterns)
            
            # 3. Temporal correlation analysis
            temporal_patterns = await self._mine_temporal_patterns(clinical_data)
            discovered_patterns.extend(temporal_patterns)
            
            # 4. Adverse event pattern detection
            adverse_patterns = await self._mine_adverse_event_patterns(clinical_data)
            discovered_patterns.extend(adverse_patterns)
            
            # Store discovered patterns
            for pattern in discovered_patterns:
                self.discovered_patterns[pattern.pattern_id] = pattern
            
            # Update statistics
            self.learning_stats["patterns_discovered"] += len(discovered_patterns)
            self.learning_stats["last_learning_run"] = datetime.utcnow()
            
            logger.info(f"Discovered {len(discovered_patterns)} new clinical patterns")
            return discovered_patterns
            
        except Exception as e:
            logger.error(f"Error learning from clinical data: {e}")
            return []
    
    async def predict_clinical_outcome(self, patient_context: Dict[str, Any],
                                     medication_list: List[str]) -> PatternPrediction:
        """
        Predict clinical outcome based on learned patterns
        
        Args:
            patient_context: Patient clinical context
            medication_list: List of medications
            
        Returns:
            Clinical outcome prediction
        """
        try:
            # Find relevant patterns
            relevant_patterns = self._find_relevant_patterns(patient_context, medication_list)
            
            if not relevant_patterns:
                return self._create_default_prediction(patient_context, medication_list)
            
            # Calculate outcome probabilities
            outcome_probabilities = self._calculate_outcome_probabilities(
                relevant_patterns, patient_context, medication_list
            )
            
            # Determine most likely outcome
            predicted_outcome = max(outcome_probabilities.items(), key=lambda x: x[1])
            
            # Generate risk factors and recommendations
            risk_factors = self._identify_risk_factors(relevant_patterns, patient_context)
            recommendations = self._generate_recommendations(relevant_patterns, predicted_outcome[0])
            
            # Create explanation
            explanation = self._create_prediction_explanation(
                relevant_patterns, predicted_outcome, risk_factors
            )
            
            # Create prediction
            prediction = PatternPrediction(
                prediction_id=f"pred_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                predicted_outcome=predicted_outcome[0],
                confidence=predicted_outcome[1],
                supporting_patterns=[p.pattern_id for p in relevant_patterns],
                risk_factors=risk_factors,
                recommendations=recommendations,
                explanation=explanation,
                created_at=datetime.utcnow()
            )
            
            # Store prediction
            self.pattern_predictions[prediction.prediction_id] = prediction
            
            # Update statistics
            self.learning_stats["predictions_made"] += 1
            
            logger.info(f"Generated prediction: {predicted_outcome[0]} "
                       f"(confidence: {predicted_outcome[1]:.3f})")
            
            return prediction
            
        except Exception as e:
            logger.error(f"Error predicting clinical outcome: {e}")
            return self._create_default_prediction(patient_context, medication_list)
    
    async def validate_patterns(self, validation_data: List[Dict[str, Any]]) -> Dict[str, float]:
        """
        Validate discovered patterns against new data
        
        Args:
            validation_data: New clinical data for validation
            
        Returns:
            Validation scores for each pattern
        """
        try:
            validation_scores = {}
            
            for pattern_id, pattern in self.discovered_patterns.items():
                score = await self._validate_single_pattern(pattern, validation_data)
                validation_scores[pattern_id] = score
                
                # Update pattern validation score
                pattern.last_validated = datetime.utcnow()
                pattern.validation_score = score
            
            # Calculate overall validation accuracy
            if validation_scores:
                overall_accuracy = sum(validation_scores.values()) / len(validation_scores)
                self.learning_stats["validation_accuracy"] = overall_accuracy
            
            logger.info(f"Validated {len(validation_scores)} patterns, "
                       f"average accuracy: {self.learning_stats['validation_accuracy']:.3f}")
            
            return validation_scores
            
        except Exception as e:
            logger.error(f"Error validating patterns: {e}")
            return {}
    
    async def _mine_interaction_patterns(self, clinical_data: List[Dict[str, Any]]) -> List[ClinicalPattern]:
        """Mine drug interaction patterns using association rules"""
        patterns = []
        
        try:
            # Extract medication combinations
            medication_combinations = []
            for transaction in clinical_data:
                medications = transaction.get("medications", [])
                if len(medications) >= 2:
                    medication_combinations.append(medications)
            
            if not medication_combinations:
                return patterns
            
            # Find frequent itemsets
            frequent_itemsets = self._find_frequent_itemsets(medication_combinations)
            
            # Generate association rules
            for itemset, support in frequent_itemsets.items():
                if len(itemset) >= 2:
                    # Generate all possible rules from this itemset
                    rules = self._generate_association_rules(itemset, support, medication_combinations)
                    
                    for rule in rules:
                        if rule["confidence"] >= self.min_confidence:
                            pattern = ClinicalPattern(
                                pattern_id=f"interaction_{rule['antecedent']}_{rule['consequent']}",
                                pattern_type=PatternType.DRUG_INTERACTION,
                                entities=list(itemset),
                                frequency=rule["frequency"],
                                confidence=rule["confidence"],
                                support=support,
                                lift=rule["lift"],
                                conviction=rule["conviction"],
                                clinical_significance=self._assess_interaction_significance(rule),
                                discovered_at=datetime.utcnow(),
                                last_validated=datetime.utcnow(),
                                validation_score=0.0,
                                metadata=rule
                            )
                            patterns.append(pattern)
            
            logger.info(f"Mined {len(patterns)} interaction patterns")
            return patterns
            
        except Exception as e:
            logger.error(f"Error mining interaction patterns: {e}")
            return []
    
    async def _mine_sequence_patterns(self, clinical_data: List[Dict[str, Any]]) -> List[ClinicalPattern]:
        """Mine sequential medication patterns"""
        patterns = []
        
        try:
            # Extract medication sequences
            sequences = []
            for transaction in clinical_data:
                if "medication_sequence" in transaction:
                    sequences.append(transaction["medication_sequence"])
            
            if not sequences:
                return patterns
            
            # Find frequent sequences
            frequent_sequences = self._find_frequent_sequences(sequences)
            
            for sequence, frequency in frequent_sequences.items():
                support = frequency / len(sequences)
                
                if support >= self.min_support:
                    pattern = ClinicalPattern(
                        pattern_id=f"sequence_{'_'.join(sequence)}",
                        pattern_type=PatternType.MEDICATION_SEQUENCE,
                        entities=list(sequence),
                        frequency=frequency,
                        confidence=support,  # For sequences, support serves as confidence
                        support=support,
                        lift=1.0,  # Not applicable for sequences
                        conviction=1.0,  # Not applicable for sequences
                        clinical_significance=self._assess_sequence_significance(sequence, frequency),
                        discovered_at=datetime.utcnow(),
                        last_validated=datetime.utcnow(),
                        validation_score=0.0,
                        metadata={"sequence": sequence, "frequency": frequency}
                    )
                    patterns.append(pattern)
            
            logger.info(f"Mined {len(patterns)} sequence patterns")
            return patterns
            
        except Exception as e:
            logger.error(f"Error mining sequence patterns: {e}")
            return []
    
    async def _mine_temporal_patterns(self, clinical_data: List[Dict[str, Any]]) -> List[ClinicalPattern]:
        """Mine temporal correlation patterns"""
        patterns = []
        
        try:
            # Group events by time windows
            temporal_groups = defaultdict(list)
            
            for transaction in clinical_data:
                timestamp = transaction.get("timestamp")
                if timestamp:
                    # Convert to time window (e.g., hour of day)
                    if isinstance(timestamp, str):
                        timestamp = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                    
                    time_key = timestamp.hour  # Group by hour of day
                    temporal_groups[time_key].append(transaction)
            
            # Analyze patterns within time windows
            for time_key, transactions in temporal_groups.items():
                if len(transactions) >= 5:  # Minimum transactions for pattern
                    # Find common elements in this time window
                    common_medications = self._find_common_elements(
                        [t.get("medications", []) for t in transactions]
                    )
                    
                    for medication, frequency in common_medications.items():
                        support = frequency / len(transactions)
                        
                        if support >= self.min_support:
                            pattern = ClinicalPattern(
                                pattern_id=f"temporal_{medication}_{time_key}",
                                pattern_type=PatternType.TEMPORAL_CORRELATION,
                                entities=[medication],
                                frequency=frequency,
                                confidence=support,
                                support=support,
                                lift=1.0,
                                conviction=1.0,
                                clinical_significance="moderate",
                                discovered_at=datetime.utcnow(),
                                last_validated=datetime.utcnow(),
                                validation_score=0.0,
                                metadata={
                                    "time_window": time_key,
                                    "total_transactions": len(transactions)
                                }
                            )
                            patterns.append(pattern)
            
            logger.info(f"Mined {len(patterns)} temporal patterns")
            return patterns
            
        except Exception as e:
            logger.error(f"Error mining temporal patterns: {e}")
            return []
    
    async def _mine_adverse_event_patterns(self, clinical_data: List[Dict[str, Any]]) -> List[ClinicalPattern]:
        """Mine adverse event patterns"""
        patterns = []
        
        try:
            # Filter for adverse events
            adverse_events = [
                t for t in clinical_data 
                if t.get("outcome_type") == "adverse_event"
            ]
            
            if not adverse_events:
                return patterns
            
            # Analyze medication combinations leading to adverse events
            adverse_combinations = []
            for event in adverse_events:
                medications = event.get("medications", [])
                if medications:
                    adverse_combinations.append(medications)
            
            # Find frequent adverse combinations
            frequent_adverse = self._find_frequent_itemsets(adverse_combinations)
            
            for itemset, frequency in frequent_adverse.items():
                support = frequency / len(adverse_combinations)
                
                if support >= self.min_support and len(itemset) >= 1:
                    pattern = ClinicalPattern(
                        pattern_id=f"adverse_{'_'.join(sorted(itemset))}",
                        pattern_type=PatternType.ADVERSE_EVENT,
                        entities=list(itemset),
                        frequency=frequency,
                        confidence=support,
                        support=support,
                        lift=1.0,
                        conviction=1.0,
                        clinical_significance="high",  # Adverse events are always significant
                        discovered_at=datetime.utcnow(),
                        last_validated=datetime.utcnow(),
                        validation_score=0.0,
                        metadata={
                            "adverse_event_frequency": frequency,
                            "total_adverse_events": len(adverse_events)
                        }
                    )
                    patterns.append(pattern)
            
            logger.info(f"Mined {len(patterns)} adverse event patterns")
            return patterns
            
        except Exception as e:
            logger.error(f"Error mining adverse event patterns: {e}")
            return []
    
    def _find_frequent_itemsets(self, transactions: List[List[str]]) -> Dict[frozenset, int]:
        """Find frequent itemsets using Apriori algorithm"""
        if not transactions:
            return {}
        
        # Count individual items
        item_counts = Counter()
        for transaction in transactions:
            for item in transaction:
                item_counts[item] += 1
        
        # Find frequent 1-itemsets
        min_support_count = int(self.min_support * len(transactions))
        frequent_itemsets = {}
        
        for item, count in item_counts.items():
            if count >= min_support_count:
                frequent_itemsets[frozenset([item])] = count
        
        # Generate larger itemsets
        k = 2
        while k <= self.max_pattern_length:
            candidates = self._generate_candidates(list(frequent_itemsets.keys()), k)
            
            if not candidates:
                break
            
            # Count candidate support
            candidate_counts = Counter()
            for transaction in transactions:
                transaction_set = set(transaction)
                for candidate in candidates:
                    if candidate.issubset(transaction_set):
                        candidate_counts[candidate] += 1
            
            # Keep frequent candidates
            new_frequent = {}
            for candidate, count in candidate_counts.items():
                if count >= min_support_count:
                    new_frequent[candidate] = count
            
            if not new_frequent:
                break
            
            frequent_itemsets.update(new_frequent)
            k += 1
        
        return frequent_itemsets
    
    def _generate_candidates(self, frequent_itemsets: List[frozenset], k: int) -> List[frozenset]:
        """Generate candidate itemsets of size k"""
        candidates = []
        
        for i in range(len(frequent_itemsets)):
            for j in range(i + 1, len(frequent_itemsets)):
                # Join two (k-1)-itemsets
                union = frequent_itemsets[i] | frequent_itemsets[j]
                if len(union) == k:
                    candidates.append(union)
        
        return candidates
    
    def _find_frequent_sequences(self, sequences: List[List[str]]) -> Dict[tuple, int]:
        """Find frequent sequential patterns"""
        sequence_counts = Counter()
        
        for sequence in sequences:
            # Generate all subsequences
            for length in range(1, min(len(sequence) + 1, self.max_pattern_length + 1)):
                for start in range(len(sequence) - length + 1):
                    subseq = tuple(sequence[start:start + length])
                    sequence_counts[subseq] += 1
        
        # Filter by minimum support
        min_support_count = int(self.min_support * len(sequences))
        frequent_sequences = {
            seq: count for seq, count in sequence_counts.items()
            if count >= min_support_count
        }
        
        return frequent_sequences
    
    def _find_common_elements(self, element_lists: List[List[str]]) -> Dict[str, int]:
        """Find common elements across lists"""
        element_counts = Counter()
        
        for element_list in element_lists:
            for element in set(element_list):  # Count each element once per list
                element_counts[element] += 1
        
        return dict(element_counts)
    
    def _generate_association_rules(self, itemset: frozenset, support: float,
                                  transactions: List[List[str]]) -> List[Dict[str, Any]]:
        """Generate association rules from frequent itemset"""
        rules = []
        
        if len(itemset) < 2:
            return rules
        
        # Generate all possible antecedent/consequent combinations
        for i in range(1, len(itemset)):
            for antecedent in self._get_combinations(itemset, i):
                consequent = itemset - antecedent
                
                # Calculate confidence
                antecedent_count = sum(
                    1 for t in transactions if antecedent.issubset(set(t))
                )
                
                if antecedent_count > 0:
                    confidence = support / (antecedent_count / len(transactions))
                    
                    # Calculate lift
                    consequent_count = sum(
                        1 for t in transactions if consequent.issubset(set(t))
                    )
                    consequent_support = consequent_count / len(transactions)
                    
                    lift = confidence / consequent_support if consequent_support > 0 else 0
                    
                    # Calculate conviction
                    conviction = (1 - consequent_support) / (1 - confidence) if confidence < 1 else float('inf')
                    
                    rules.append({
                        "antecedent": "_".join(sorted(antecedent)),
                        "consequent": "_".join(sorted(consequent)),
                        "confidence": confidence,
                        "lift": lift,
                        "conviction": conviction,
                        "frequency": int(support * len(transactions))
                    })
        
        return rules
    
    def _get_combinations(self, itemset: frozenset, size: int) -> List[frozenset]:
        """Get all combinations of given size from itemset"""
        from itertools import combinations
        items = list(itemset)
        return [frozenset(combo) for combo in combinations(items, size)]
    
    def _assess_interaction_significance(self, rule: Dict[str, Any]) -> str:
        """Assess clinical significance of interaction rule"""
        lift = rule.get("lift", 1.0)
        confidence = rule.get("confidence", 0.0)
        
        if lift > 2.0 and confidence > 0.8:
            return "high"
        elif lift > 1.5 and confidence > 0.6:
            return "moderate"
        else:
            return "low"
    
    def _assess_sequence_significance(self, sequence: tuple, frequency: int) -> str:
        """Assess clinical significance of sequence pattern"""
        if frequency > 50:
            return "high"
        elif frequency > 20:
            return "moderate"
        else:
            return "low"
    
    def _find_relevant_patterns(self, patient_context: Dict[str, Any],
                              medication_list: List[str]) -> List[ClinicalPattern]:
        """Find patterns relevant to current patient context"""
        relevant_patterns = []
        
        for pattern in self.discovered_patterns.values():
            # Check if pattern entities overlap with current medications
            pattern_entities = set(pattern.entities)
            medication_set = set(medication_list)
            
            if pattern_entities.intersection(medication_set):
                relevant_patterns.append(pattern)
        
        # Sort by confidence and validation score
        relevant_patterns.sort(
            key=lambda p: (p.confidence * p.validation_score), reverse=True
        )
        
        return relevant_patterns[:10]  # Top 10 most relevant patterns
    
    def _calculate_outcome_probabilities(self, patterns: List[ClinicalPattern],
                                       patient_context: Dict[str, Any],
                                       medication_list: List[str]) -> Dict[str, float]:
        """Calculate outcome probabilities based on patterns"""
        outcome_scores = defaultdict(float)
        
        for pattern in patterns:
            if pattern.pattern_type == PatternType.ADVERSE_EVENT:
                outcome_scores["adverse_event"] += pattern.confidence * pattern.validation_score
            elif pattern.pattern_type == PatternType.THERAPEUTIC_RESPONSE:
                outcome_scores["therapeutic_success"] += pattern.confidence * pattern.validation_score
            else:
                outcome_scores["neutral"] += pattern.confidence * pattern.validation_score
        
        # Normalize probabilities
        total_score = sum(outcome_scores.values())
        if total_score > 0:
            outcome_probabilities = {
                outcome: score / total_score 
                for outcome, score in outcome_scores.items()
            }
        else:
            outcome_probabilities = {"neutral": 1.0}
        
        return outcome_probabilities
    
    def _identify_risk_factors(self, patterns: List[ClinicalPattern],
                             patient_context: Dict[str, Any]) -> List[str]:
        """Identify risk factors from patterns"""
        risk_factors = []
        
        for pattern in patterns:
            if pattern.pattern_type == PatternType.ADVERSE_EVENT:
                risk_factors.extend(pattern.entities)
            elif pattern.clinical_significance == "high":
                risk_factors.extend(pattern.entities)
        
        return list(set(risk_factors))  # Remove duplicates
    
    def _generate_recommendations(self, patterns: List[ClinicalPattern],
                                predicted_outcome: str) -> List[str]:
        """Generate clinical recommendations based on patterns"""
        recommendations = []
        
        if predicted_outcome == "adverse_event":
            recommendations.extend([
                "Monitor patient closely for adverse effects",
                "Consider alternative medications",
                "Implement safety protocols"
            ])
        elif predicted_outcome == "therapeutic_success":
            recommendations.extend([
                "Continue current therapy",
                "Monitor therapeutic response",
                "Maintain current dosing"
            ])
        else:
            recommendations.extend([
                "Standard monitoring protocols",
                "Assess therapeutic response"
            ])
        
        return recommendations
    
    def _create_prediction_explanation(self, patterns: List[ClinicalPattern],
                                     predicted_outcome: Tuple[str, float],
                                     risk_factors: List[str]) -> str:
        """Create explanation for prediction"""
        explanation = f"Predicted outcome: {predicted_outcome[0]} "
        explanation += f"(confidence: {predicted_outcome[1]:.2f})\n\n"
        
        explanation += f"Based on {len(patterns)} relevant clinical patterns:\n"
        for pattern in patterns[:3]:  # Top 3 patterns
            explanation += f"- {pattern.pattern_type.value}: {', '.join(pattern.entities)} "
            explanation += f"(confidence: {pattern.confidence:.2f})\n"
        
        if risk_factors:
            explanation += f"\nIdentified risk factors: {', '.join(risk_factors)}"
        
        return explanation
    
    def _create_default_prediction(self, patient_context: Dict[str, Any],
                                 medication_list: List[str]) -> PatternPrediction:
        """Create default prediction when no patterns found"""
        return PatternPrediction(
            prediction_id=f"default_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
            predicted_outcome="neutral",
            confidence=0.5,
            supporting_patterns=[],
            risk_factors=[],
            recommendations=["Standard clinical monitoring"],
            explanation="No specific patterns found for this medication combination",
            created_at=datetime.utcnow()
        )
    
    async def _validate_single_pattern(self, pattern: ClinicalPattern,
                                     validation_data: List[Dict[str, Any]]) -> float:
        """Validate a single pattern against validation data"""
        try:
            # Count occurrences of pattern in validation data
            pattern_occurrences = 0
            total_applicable = 0
            
            for transaction in validation_data:
                medications = transaction.get("medications", [])
                
                # Check if pattern is applicable to this transaction
                if any(entity in medications for entity in pattern.entities):
                    total_applicable += 1
                    
                    # Check if pattern prediction is correct
                    if pattern.pattern_type == PatternType.ADVERSE_EVENT:
                        if transaction.get("outcome_type") == "adverse_event":
                            pattern_occurrences += 1
                    elif pattern.pattern_type == PatternType.DRUG_INTERACTION:
                        if set(pattern.entities).issubset(set(medications)):
                            pattern_occurrences += 1
            
            # Calculate validation score
            if total_applicable > 0:
                validation_score = pattern_occurrences / total_applicable
            else:
                validation_score = 0.0
            
            return validation_score
            
        except Exception as e:
            logger.error(f"Error validating pattern {pattern.pattern_id}: {e}")
            return 0.0
    
    def get_learning_statistics(self) -> Dict[str, Any]:
        """Get comprehensive learning statistics"""
        pattern_type_counts = Counter(p.pattern_type.value for p in self.discovered_patterns.values())
        
        return {
            "learning_stats": self.learning_stats,
            "pattern_counts": dict(pattern_type_counts),
            "total_patterns": len(self.discovered_patterns),
            "total_predictions": len(self.pattern_predictions),
            "clinical_transactions": len(self.clinical_transactions),
            "average_pattern_confidence": self._calculate_average_confidence(),
            "high_confidence_patterns": self._count_high_confidence_patterns()
        }
    
    def _calculate_average_confidence(self) -> float:
        """Calculate average confidence across all patterns"""
        if not self.discovered_patterns:
            return 0.0
        
        total_confidence = sum(p.confidence for p in self.discovered_patterns.values())
        return total_confidence / len(self.discovered_patterns)
    
    def _count_high_confidence_patterns(self) -> int:
        """Count patterns with high confidence (>0.8)"""
        return sum(1 for p in self.discovered_patterns.values() if p.confidence > 0.8)
