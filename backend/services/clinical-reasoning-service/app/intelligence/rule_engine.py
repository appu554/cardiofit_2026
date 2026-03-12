"""
Self-Improving Rule Engine for Clinical Assertion Engine

Dynamic rule learning system that adapts based on clinical outcomes,
override patterns, and real-world evidence to continuously improve
clinical decision accuracy.
"""

import logging
import json
import asyncio
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
from enum import Enum
import math

logger = logging.getLogger(__name__)


class RuleType(Enum):
    """Types of clinical rules"""
    INTERACTION = "interaction"
    DOSING = "dosing"
    CONTRAINDICATION = "contraindication"
    DUPLICATE_THERAPY = "duplicate_therapy"
    CLINICAL_CONTEXT = "clinical_context"


class RuleConfidence(Enum):
    """Rule confidence levels"""
    LEARNING = "learning"      # <0.3 - Still learning
    LOW = "low"               # 0.3-0.5 - Low confidence
    MODERATE = "moderate"     # 0.5-0.7 - Moderate confidence
    HIGH = "high"            # 0.7-0.9 - High confidence
    VALIDATED = "validated"   # >0.9 - Clinically validated


@dataclass
class ClinicalRule:
    """Dynamic clinical rule with learning capabilities"""
    rule_id: str
    rule_type: RuleType
    conditions: Dict[str, Any]  # Rule conditions (medications, patient factors)
    assertions: Dict[str, Any]  # Rule outputs (severity, recommendations)
    confidence_score: float
    evidence_count: int
    positive_outcomes: int
    negative_outcomes: int
    override_count: int
    last_updated: datetime
    created_at: datetime
    learning_rate: float = 0.1
    metadata: Dict[str, Any] = None


@dataclass
class LearningEvent:
    """Learning event from clinical outcomes"""
    event_id: str
    rule_id: str
    event_type: str  # outcome, override, validation
    outcome_positive: bool
    confidence_impact: float
    evidence_strength: float
    clinical_context: Dict[str, Any]
    timestamp: datetime


class SelfImprovingRuleEngine:
    """
    Self-improving rule engine with dynamic learning capabilities
    
    Features:
    - Dynamic rule creation from patterns
    - Confidence evolution based on outcomes
    - Override pattern learning
    - Rule validation and retirement
    - Performance-based rule optimization
    """
    
    def __init__(self, min_evidence_threshold: int = 5, confidence_threshold: float = 0.3):
        self.min_evidence_threshold = min_evidence_threshold
        self.confidence_threshold = confidence_threshold
        
        # Rule storage
        self.active_rules: Dict[str, ClinicalRule] = {}
        self.learning_rules: Dict[str, ClinicalRule] = {}
        self.retired_rules: Dict[str, ClinicalRule] = {}
        
        # Learning events
        self.learning_events: List[LearningEvent] = []
        
        # Performance metrics
        self.performance_metrics = {
            "total_rules": 0,
            "active_rules": 0,
            "learning_rules": 0,
            "retired_rules": 0,
            "average_confidence": 0.0,
            "learning_events_processed": 0,
            "rules_promoted": 0,
            "rules_retired": 0
        }
        
        logger.info("Self-Improving Rule Engine initialized")
    
    async def evaluate_rules(self, clinical_context: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Evaluate clinical context against dynamic rules
        
        Args:
            clinical_context: Patient and medication context
            
        Returns:
            List of rule-based assertions
        """
        try:
            assertions = []
            
            # Evaluate active rules
            for rule in self.active_rules.values():
                if await self._rule_matches_context(rule, clinical_context):
                    assertion = await self._generate_assertion_from_rule(rule, clinical_context)
                    if assertion:
                        assertions.append(assertion)
            
            # Evaluate learning rules (with lower confidence)
            for rule in self.learning_rules.values():
                if await self._rule_matches_context(rule, clinical_context):
                    assertion = await self._generate_assertion_from_rule(rule, clinical_context)
                    if assertion:
                        # Mark as learning rule
                        assertion["metadata"]["rule_status"] = "learning"
                        assertion["confidence"] *= 0.7  # Reduce confidence for learning rules
                        assertions.append(assertion)
            
            logger.info(f"Evaluated {len(self.active_rules)} active rules and {len(self.learning_rules)} learning rules")
            return assertions
            
        except Exception as e:
            logger.error(f"Error evaluating rules: {e}")
            return []
    
    async def learn_from_outcome(self, rule_id: str, outcome_positive: bool, 
                                clinical_context: Dict[str, Any], 
                                evidence_strength: float = 1.0):
        """
        Learn from clinical outcome to improve rule confidence
        
        Args:
            rule_id: Rule that generated the assertion
            outcome_positive: Whether outcome was positive
            clinical_context: Clinical context of the outcome
            evidence_strength: Strength of the evidence (0.0-1.0)
        """
        try:
            # Find the rule
            rule = self._find_rule(rule_id)
            if not rule:
                logger.warning(f"Rule {rule_id} not found for learning")
                return
            
            # Create learning event
            learning_event = LearningEvent(
                event_id=f"outcome_{rule_id}_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                rule_id=rule_id,
                event_type="outcome",
                outcome_positive=outcome_positive,
                confidence_impact=self._calculate_confidence_impact(outcome_positive, evidence_strength),
                evidence_strength=evidence_strength,
                clinical_context=clinical_context,
                timestamp=datetime.utcnow()
            )
            
            # Apply learning
            await self._apply_learning_event(rule, learning_event)
            
            # Store learning event
            self.learning_events.append(learning_event)
            
            # Update performance metrics
            self.performance_metrics["learning_events_processed"] += 1
            
            logger.info(f"Applied learning from outcome for rule {rule_id}: "
                       f"positive={outcome_positive}, new_confidence={rule.confidence_score:.3f}")
            
        except Exception as e:
            logger.error(f"Error learning from outcome: {e}")
    
    async def learn_from_override(self, rule_id: str, override_reason: str,
                                clinical_context: Dict[str, Any],
                                clinician_expertise: float = 1.0):
        """
        Learn from clinician override to adjust rule behavior
        
        Args:
            rule_id: Rule that was overridden
            override_reason: Reason for override
            clinical_context: Clinical context of override
            clinician_expertise: Expertise level of clinician (0.0-1.0)
        """
        try:
            rule = self._find_rule(rule_id)
            if not rule:
                logger.warning(f"Rule {rule_id} not found for override learning")
                return
            
            # Analyze override reason
            confidence_impact = self._analyze_override_impact(override_reason, clinician_expertise)
            
            # Create learning event
            learning_event = LearningEvent(
                event_id=f"override_{rule_id}_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}",
                rule_id=rule_id,
                event_type="override",
                outcome_positive=False,  # Override indicates rule was incorrect
                confidence_impact=confidence_impact,
                evidence_strength=clinician_expertise,
                clinical_context=clinical_context,
                timestamp=datetime.utcnow()
            )
            
            # Apply learning
            await self._apply_learning_event(rule, learning_event)
            
            # Update override count
            rule.override_count += 1
            
            # Store learning event
            self.learning_events.append(learning_event)
            
            logger.info(f"Applied learning from override for rule {rule_id}: "
                       f"reason={override_reason}, new_confidence={rule.confidence_score:.3f}")
            
        except Exception as e:
            logger.error(f"Error learning from override: {e}")
    
    async def create_rule_from_pattern(self, pattern_data: Dict[str, Any]) -> Optional[str]:
        """
        Create new rule from discovered pattern
        
        Args:
            pattern_data: Pattern discovery data
            
        Returns:
            Rule ID if created successfully
        """
        try:
            # Extract rule components from pattern
            rule_conditions = self._extract_rule_conditions(pattern_data)
            rule_assertions = self._extract_rule_assertions(pattern_data)
            
            if not rule_conditions or not rule_assertions:
                logger.warning("Insufficient data to create rule from pattern")
                return None
            
            # Create new rule
            rule_id = f"pattern_rule_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"
            
            new_rule = ClinicalRule(
                rule_id=rule_id,
                rule_type=RuleType(pattern_data.get("pattern_type", "interaction")),
                conditions=rule_conditions,
                assertions=rule_assertions,
                confidence_score=pattern_data.get("confidence_score", 0.5),
                evidence_count=pattern_data.get("support_count", 1),
                positive_outcomes=0,
                negative_outcomes=0,
                override_count=0,
                last_updated=datetime.utcnow(),
                created_at=datetime.utcnow(),
                metadata=pattern_data.get("metadata", {})
            )
            
            # Add to learning rules
            self.learning_rules[rule_id] = new_rule
            
            # Update metrics
            self.performance_metrics["total_rules"] += 1
            self.performance_metrics["learning_rules"] += 1
            
            logger.info(f"Created new rule {rule_id} from pattern with confidence {new_rule.confidence_score:.3f}")
            return rule_id
            
        except Exception as e:
            logger.error(f"Error creating rule from pattern: {e}")
            return None
    
    async def promote_learning_rules(self):
        """Promote learning rules to active status based on performance"""
        try:
            promoted_rules = []
            
            for rule_id, rule in list(self.learning_rules.items()):
                if await self._should_promote_rule(rule):
                    # Move to active rules
                    self.active_rules[rule_id] = rule
                    del self.learning_rules[rule_id]
                    
                    promoted_rules.append(rule_id)
                    
                    # Update metrics
                    self.performance_metrics["active_rules"] += 1
                    self.performance_metrics["learning_rules"] -= 1
                    self.performance_metrics["rules_promoted"] += 1
                    
                    logger.info(f"Promoted rule {rule_id} to active status "
                               f"(confidence: {rule.confidence_score:.3f})")
            
            return promoted_rules
            
        except Exception as e:
            logger.error(f"Error promoting learning rules: {e}")
            return []
    
    async def retire_poor_rules(self):
        """Retire rules with consistently poor performance"""
        try:
            retired_rules = []
            
            # Check active rules for retirement
            for rule_id, rule in list(self.active_rules.items()):
                if await self._should_retire_rule(rule):
                    # Move to retired rules
                    self.retired_rules[rule_id] = rule
                    del self.active_rules[rule_id]
                    
                    retired_rules.append(rule_id)
                    
                    # Update metrics
                    self.performance_metrics["active_rules"] -= 1
                    self.performance_metrics["retired_rules"] += 1
                    self.performance_metrics["rules_retired"] += 1
                    
                    logger.info(f"Retired rule {rule_id} due to poor performance "
                               f"(confidence: {rule.confidence_score:.3f})")
            
            # Check learning rules for retirement
            for rule_id, rule in list(self.learning_rules.items()):
                if await self._should_retire_rule(rule):
                    self.retired_rules[rule_id] = rule
                    del self.learning_rules[rule_id]
                    
                    retired_rules.append(rule_id)
                    
                    # Update metrics
                    self.performance_metrics["learning_rules"] -= 1
                    self.performance_metrics["retired_rules"] += 1
                    self.performance_metrics["rules_retired"] += 1
            
            return retired_rules
            
        except Exception as e:
            logger.error(f"Error retiring poor rules: {e}")
            return []
    
    async def _rule_matches_context(self, rule: ClinicalRule, context: Dict[str, Any]) -> bool:
        """Check if rule conditions match clinical context"""
        try:
            conditions = rule.conditions
            
            # Check medication conditions
            if "medications" in conditions:
                required_meds = set(conditions["medications"])
                context_meds = set(context.get("medication_ids", []))
                if not required_meds.issubset(context_meds):
                    return False
            
            # Check patient conditions
            if "patient_conditions" in conditions:
                required_conditions = set(conditions["patient_conditions"])
                context_conditions = set(context.get("condition_ids", []))
                if not required_conditions.issubset(context_conditions):
                    return False
            
            # Check age range
            if "age_range" in conditions:
                age_range = conditions["age_range"]
                patient_age = context.get("patient_context", {}).get("age", 0)
                if not (age_range["min"] <= patient_age <= age_range["max"]):
                    return False
            
            return True
            
        except Exception as e:
            logger.error(f"Error matching rule conditions: {e}")
            return False
    
    async def _generate_assertion_from_rule(self, rule: ClinicalRule, 
                                          context: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Generate clinical assertion from rule"""
        try:
            assertions = rule.assertions
            
            return {
                "type": rule.rule_type.value,
                "severity": assertions.get("severity", "moderate"),
                "title": assertions.get("title", f"Rule-based {rule.rule_type.value}"),
                "description": assertions.get("description", "Generated from learned rule"),
                "explanation": assertions.get("explanation", f"Based on rule {rule.rule_id}"),
                "confidence": rule.confidence_score,
                "evidence_sources": ["Dynamic Rule Engine", "Clinical Patterns"],
                "recommendations": assertions.get("recommendations", []),
                "metadata": {
                    "rule_id": rule.rule_id,
                    "rule_type": rule.rule_type.value,
                    "evidence_count": rule.evidence_count,
                    "rule_confidence": self._get_confidence_level(rule.confidence_score).value
                }
            }
            
        except Exception as e:
            logger.error(f"Error generating assertion from rule: {e}")
            return None
    
    def _find_rule(self, rule_id: str) -> Optional[ClinicalRule]:
        """Find rule by ID across all rule collections"""
        if rule_id in self.active_rules:
            return self.active_rules[rule_id]
        elif rule_id in self.learning_rules:
            return self.learning_rules[rule_id]
        elif rule_id in self.retired_rules:
            return self.retired_rules[rule_id]
        return None
    
    def _calculate_confidence_impact(self, outcome_positive: bool, evidence_strength: float) -> float:
        """Calculate confidence impact from outcome"""
        base_impact = 0.1 * evidence_strength
        return base_impact if outcome_positive else -base_impact
    
    def _analyze_override_impact(self, override_reason: str, clinician_expertise: float) -> float:
        """Analyze override impact on rule confidence"""
        # Different override reasons have different impacts
        reason_impacts = {
            "clinical_judgment": -0.05,
            "patient_specific": -0.03,
            "false_positive": -0.15,
            "inappropriate_alert": -0.20,
            "system_error": -0.25
        }
        
        base_impact = reason_impacts.get(override_reason.lower(), -0.10)
        return base_impact * clinician_expertise
    
    async def _apply_learning_event(self, rule: ClinicalRule, event: LearningEvent):
        """Apply learning event to update rule"""
        # Update confidence using exponential moving average
        old_confidence = rule.confidence_score
        confidence_delta = event.confidence_impact * rule.learning_rate
        rule.confidence_score = max(0.0, min(1.0, old_confidence + confidence_delta))
        
        # Update evidence counts
        rule.evidence_count += 1
        if event.outcome_positive:
            rule.positive_outcomes += 1
        else:
            rule.negative_outcomes += 1
        
        # Update timestamp
        rule.last_updated = event.timestamp
        
        # Adjust learning rate based on evidence
        if rule.evidence_count > 10:
            rule.learning_rate *= 0.95  # Reduce learning rate as evidence accumulates
    
    async def _should_promote_rule(self, rule: ClinicalRule) -> bool:
        """Determine if learning rule should be promoted to active"""
        return (
            rule.confidence_score >= self.confidence_threshold and
            rule.evidence_count >= self.min_evidence_threshold and
            rule.positive_outcomes > rule.negative_outcomes
        )
    
    async def _should_retire_rule(self, rule: ClinicalRule) -> bool:
        """Determine if rule should be retired"""
        return (
            rule.confidence_score < 0.1 or
            (rule.evidence_count >= self.min_evidence_threshold and 
             rule.negative_outcomes > rule.positive_outcomes * 2) or
            rule.override_count > rule.evidence_count * 0.5
        )
    
    def _extract_rule_conditions(self, pattern_data: Dict[str, Any]) -> Dict[str, Any]:
        """Extract rule conditions from pattern data"""
        conditions = {}
        
        if "entities_involved" in pattern_data:
            conditions["medications"] = pattern_data["entities_involved"]
        
        if "clinical_context" in pattern_data:
            context = pattern_data["clinical_context"]
            if "conditions" in context:
                conditions["patient_conditions"] = context["conditions"]
            if "age_range" in context:
                conditions["age_range"] = context["age_range"]
        
        return conditions
    
    def _extract_rule_assertions(self, pattern_data: Dict[str, Any]) -> Dict[str, Any]:
        """Extract rule assertions from pattern data"""
        return {
            "severity": pattern_data.get("clinical_significance", "moderate"),
            "title": pattern_data.get("description", "Pattern-based assertion"),
            "description": pattern_data.get("description", ""),
            "explanation": f"Based on pattern with {pattern_data.get('support_count', 0)} occurrences",
            "recommendations": ["Monitor closely", "Consider clinical context"]
        }
    
    def _get_confidence_level(self, confidence_score: float) -> RuleConfidence:
        """Get confidence level enum from score"""
        if confidence_score < 0.3:
            return RuleConfidence.LEARNING
        elif confidence_score < 0.5:
            return RuleConfidence.LOW
        elif confidence_score < 0.7:
            return RuleConfidence.MODERATE
        elif confidence_score < 0.9:
            return RuleConfidence.HIGH
        else:
            return RuleConfidence.VALIDATED
    
    def get_performance_metrics(self) -> Dict[str, Any]:
        """Get rule engine performance metrics"""
        # Update average confidence
        all_rules = list(self.active_rules.values()) + list(self.learning_rules.values())
        if all_rules:
            self.performance_metrics["average_confidence"] = sum(r.confidence_score for r in all_rules) / len(all_rules)
        
        return self.performance_metrics.copy()
    
    def export_rules(self) -> Dict[str, Any]:
        """Export all rules for persistence"""
        return {
            "active_rules": {rid: asdict(rule) for rid, rule in self.active_rules.items()},
            "learning_rules": {rid: asdict(rule) for rid, rule in self.learning_rules.items()},
            "retired_rules": {rid: asdict(rule) for rid, rule in self.retired_rules.items()},
            "performance_metrics": self.performance_metrics,
            "exported_at": datetime.utcnow().isoformat()
        }
