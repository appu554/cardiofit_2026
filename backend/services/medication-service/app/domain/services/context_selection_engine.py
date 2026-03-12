"""
Context Selection Engine - YAML-Based Rule Engine for Context Recipe Selection

This module implements the Context Selection Engine (renamed from ClinicalRuleEngine)
for the Enhanced Orchestrator, providing sophisticated rule-based context recipe
selection with YAML-based rules, advanced matching algorithms, and clinical rationale.

Key Features:
- YAML-based rule definitions with trigger conditions
- Sophisticated matching algorithms (all_of, any_of, none_of)
- Clinical priority scoring and evidence levels
- Performance-optimized rule evaluation
- Comprehensive audit trails and explanations
"""

import logging
import yaml
import os
from typing import Dict, List, Any, Optional, Set, Tuple
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
import asyncio
import time

from ..models.analyzed_request_models import (
    AnalyzedRequest, RiskLevel, AgeGroup, OrganFunction, UrgencyLevel
)

logger = logging.getLogger(__name__)


class RuleMatchType(Enum):
    """Rule matching types for trigger conditions"""
    ALL_OF = "all_of"      # AND conditions - all must be true
    ANY_OF = "any_of"      # OR conditions - at least one must be true
    NONE_OF = "none_of"    # NOT conditions - none must be true


@dataclass
class RuleTrigger:
    """Rule trigger conditions with different matching types"""
    all_of: List[str] = field(default_factory=list)
    any_of: List[str] = field(default_factory=list)
    none_of: List[str] = field(default_factory=list)


@dataclass
class RuleScoring:
    """Rule scoring configuration"""
    base_score: int = 50
    modifiers: List[Dict[str, Any]] = field(default_factory=list)


@dataclass
class ContextSelectionRule:
    """Context selection rule definition"""
    id: str
    name: str
    priority: int
    context_recipe: str
    triggers: RuleTrigger
    scoring: RuleScoring
    clinical_rationale: str
    evidence_level: str = "moderate"
    guideline_refs: List[str] = field(default_factory=list)
    created_at: datetime = field(default_factory=datetime.utcnow)
    is_active: bool = True


@dataclass
class RuleMatchResult:
    """Result of rule matching evaluation"""
    rule: ContextSelectionRule
    matches: bool
    matched_conditions: List[str] = field(default_factory=list)
    failed_conditions: List[str] = field(default_factory=list)
    match_score: float = 0.0
    evaluation_time_ms: float = 0.0


@dataclass
class ScoredRule:
    """Rule with calculated final score"""
    rule: ContextSelectionRule
    match_result: RuleMatchResult
    final_score: float
    clinical_priority_score: float
    specificity_score: float
    risk_assessment_score: float
    evidence_level_score: float
    scoring_rationale: str


@dataclass
class ContextRecipeSelection:
    """Final context recipe selection result"""
    context_recipe_id: str
    selected_rule: ScoredRule
    confidence_score: float
    clinical_rationale: str
    audit_trail: Dict[str, Any]
    selection_time_ms: float
    multiple_matches: bool = False
    matched_rules: List[ScoredRule] = field(default_factory=list)


class RuleRepository:
    """Repository for loading and managing YAML-based context selection rules"""
    
    def __init__(self, rules_directory: str = "rules/context-selection"):
        self.rules_directory = rules_directory
        self.rules_cache: Dict[str, ContextSelectionRule] = {}
        self.cache_timestamp: Optional[datetime] = None
        self.cache_ttl_seconds = 300  # 5 minutes
        
        logger.info(f"Rule Repository initialized with directory: {rules_directory}")
    
    async def load_rules(self) -> List[ContextSelectionRule]:
        """Load all context selection rules from YAML files"""
        
        # Check cache validity
        if (self.cache_timestamp and 
            (datetime.utcnow() - self.cache_timestamp).total_seconds() < self.cache_ttl_seconds):
            logger.debug("Using cached rules")
            return list(self.rules_cache.values())
        
        logger.info("Loading rules from YAML files")
        rules = []
        
        try:
            # Create rules directory if it doesn't exist
            os.makedirs(self.rules_directory, exist_ok=True)
            
            # Load rules from YAML files
            for filename in os.listdir(self.rules_directory):
                if filename.endswith('.yaml') or filename.endswith('.yml'):
                    file_path = os.path.join(self.rules_directory, filename)
                    rule_set = await self._load_rule_file(file_path)
                    rules.extend(rule_set)
            
            # Update cache
            self.rules_cache = {rule.id: rule for rule in rules}
            self.cache_timestamp = datetime.utcnow()
            
            logger.info(f"Loaded {len(rules)} context selection rules")
            return rules
            
        except Exception as e:
            logger.error(f"Failed to load rules: {str(e)}")
            # Return cached rules if available
            if self.rules_cache:
                logger.warning("Using cached rules due to loading error")
                return list(self.rules_cache.values())
            return []
    
    async def _load_rule_file(self, file_path: str) -> List[ContextSelectionRule]:
        """Load rules from a single YAML file"""
        
        try:
            with open(file_path, 'r', encoding='utf-8') as file:
                data = yaml.safe_load(file)
            
            rules = []
            rule_set = data.get('ruleSet', {})
            
            for rule_data in rule_set.get('rules', []):
                rule = self._parse_rule_definition(rule_data, rule_set.get('metadata', {}))
                if rule:
                    rules.append(rule)
            
            logger.debug(f"Loaded {len(rules)} rules from {file_path}")
            return rules
            
        except Exception as e:
            logger.error(f"Failed to load rule file {file_path}: {str(e)}")
            return []
    
    def _parse_rule_definition(self, rule_data: Dict[str, Any], metadata: Dict[str, Any]) -> Optional[ContextSelectionRule]:
        """Parse rule definition from YAML data"""
        
        try:
            # Parse triggers
            trigger_data = rule_data.get('trigger', {})
            triggers = RuleTrigger(
                all_of=trigger_data.get('all_of', []),
                any_of=trigger_data.get('any_of', []),
                none_of=trigger_data.get('none_of', [])
            )
            
            # Parse scoring
            scoring_data = rule_data.get('scoring', {})
            scoring = RuleScoring(
                base_score=scoring_data.get('base_score', 50),
                modifiers=scoring_data.get('modifiers', [])
            )
            
            # Create rule
            rule = ContextSelectionRule(
                id=rule_data['id'],
                name=rule_data['name'],
                priority=rule_data.get('priority', 50),
                context_recipe=rule_data['context_recipe'],
                triggers=triggers,
                scoring=scoring,
                clinical_rationale=rule_data.get('clinical_rationale', ''),
                evidence_level=rule_data.get('evidence_level', 'moderate'),
                guideline_refs=rule_data.get('guideline_refs', [])
            )
            
            return rule
            
        except Exception as e:
            logger.error(f"Failed to parse rule definition: {str(e)}")
            return None
    
    async def get_candidate_rules(
        self, 
        medication_class: Optional[str] = None,
        patient_age_group: Optional[AgeGroup] = None,
        urgency: Optional[UrgencyLevel] = None
    ) -> List[ContextSelectionRule]:
        """Get candidate rules based on basic filtering for performance optimization"""
        
        all_rules = await self.load_rules()
        
        # For now, return all rules - in production, implement indexed filtering
        # TODO: Implement indexed filtering based on medication_class, age_group, urgency
        
        return [rule for rule in all_rules if rule.is_active]


class RuleMatcher:
    """Advanced rule matching engine with condition evaluation"""
    
    def __init__(self):
        self.condition_evaluators = {
            'medication': self._evaluate_medication_condition,
            'patient': self._evaluate_patient_condition,
            'situation': self._evaluate_situation_condition,
            'context': self._evaluate_context_condition
        }
    
    async def evaluate_rule(self, rule: ContextSelectionRule, analyzed_request: AnalyzedRequest) -> RuleMatchResult:
        """Evaluate if a rule matches the analyzed request"""
        
        start_time = time.time()
        
        try:
            matched_conditions = []
            failed_conditions = []
            
            # Evaluate ALL_OF conditions (AND logic)
            all_of_match = True
            for condition in rule.triggers.all_of:
                if await self._evaluate_condition(condition, analyzed_request):
                    matched_conditions.append(f"all_of: {condition}")
                else:
                    failed_conditions.append(f"all_of: {condition}")
                    all_of_match = False
            
            # Evaluate ANY_OF conditions (OR logic)
            any_of_match = True
            if rule.triggers.any_of:
                any_of_match = False
                for condition in rule.triggers.any_of:
                    if await self._evaluate_condition(condition, analyzed_request):
                        matched_conditions.append(f"any_of: {condition}")
                        any_of_match = True
                    else:
                        failed_conditions.append(f"any_of: {condition}")
            
            # Evaluate NONE_OF conditions (NOT logic)
            none_of_match = True
            for condition in rule.triggers.none_of:
                if await self._evaluate_condition(condition, analyzed_request):
                    failed_conditions.append(f"none_of: {condition}")
                    none_of_match = False
                else:
                    matched_conditions.append(f"none_of: {condition} (not present)")
            
            # Overall match result
            matches = all_of_match and any_of_match and none_of_match
            
            # Calculate match score
            match_score = self._calculate_match_score(
                len(matched_conditions), 
                len(matched_conditions) + len(failed_conditions)
            )
            
            evaluation_time = (time.time() - start_time) * 1000
            
            return RuleMatchResult(
                rule=rule,
                matches=matches,
                matched_conditions=matched_conditions,
                failed_conditions=failed_conditions,
                match_score=match_score,
                evaluation_time_ms=evaluation_time
            )
            
        except Exception as e:
            logger.error(f"Rule evaluation failed for {rule.id}: {str(e)}")
            return RuleMatchResult(
                rule=rule,
                matches=False,
                failed_conditions=[f"evaluation_error: {str(e)}"],
                evaluation_time_ms=(time.time() - start_time) * 1000
            )
    
    async def _evaluate_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate a single condition against the analyzed request"""
        
        try:
            # Parse condition format: "category.property operator value"
            # Example: "medication.therapeutic_class == anticoagulant"
            
            parts = condition.split()
            if len(parts) < 3:
                logger.warning(f"Invalid condition format: {condition}")
                return False
            
            property_path = parts[0]
            operator = parts[1]
            value = ' '.join(parts[2:]).strip('"\'')
            
            # Get property value from analyzed request
            actual_value = self._get_property_value(property_path, analyzed_request)
            
            # Evaluate condition based on operator
            return self._evaluate_operator(actual_value, operator, value)
            
        except Exception as e:
            logger.error(f"Condition evaluation failed for '{condition}': {str(e)}")
            return False
    
    def _get_property_value(self, property_path: str, analyzed_request: AnalyzedRequest) -> Any:
        """Get property value from analyzed request using dot notation"""
        
        try:
            parts = property_path.split('.')
            
            # Get root object
            if parts[0] == 'medication':
                obj = analyzed_request.medication_properties
            elif parts[0] == 'patient':
                obj = analyzed_request.patient_properties
            elif parts[0] == 'situation':
                obj = analyzed_request.situational_properties
            elif parts[0] == 'context':
                obj = analyzed_request.enriched_context
            else:
                return None
            
            # Navigate through property path
            for part in parts[1:]:
                if hasattr(obj, part):
                    obj = getattr(obj, part)
                elif isinstance(obj, dict) and part in obj:
                    obj = obj[part]
                else:
                    return None
            
            # Convert enum values to strings for comparison
            if hasattr(obj, 'value'):
                return obj.value
            
            return obj
            
        except Exception as e:
            logger.error(f"Property access failed for '{property_path}': {str(e)}")
            return None
    
    def _evaluate_operator(self, actual_value: Any, operator: str, expected_value: str) -> bool:
        """Evaluate condition using operator"""
        
        try:
            if operator == '==':
                return str(actual_value).lower() == expected_value.lower()
            elif operator == '!=':
                return str(actual_value).lower() != expected_value.lower()
            elif operator == 'in':
                # Handle list membership
                if isinstance(actual_value, list):
                    return expected_value.lower() in [str(v).lower() for v in actual_value]
                else:
                    return expected_value.lower() in str(actual_value).lower()
            elif operator == 'not_in':
                if isinstance(actual_value, list):
                    return expected_value.lower() not in [str(v).lower() for v in actual_value]
                else:
                    return expected_value.lower() not in str(actual_value).lower()
            elif operator == '>=':
                return float(actual_value) >= float(expected_value)
            elif operator == '<=':
                return float(actual_value) <= float(expected_value)
            elif operator == '>':
                return float(actual_value) > float(expected_value)
            elif operator == '<':
                return float(actual_value) < float(expected_value)
            else:
                logger.warning(f"Unknown operator: {operator}")
                return False
                
        except Exception as e:
            logger.error(f"Operator evaluation failed: {actual_value} {operator} {expected_value} - {str(e)}")
            return False
    
    def _calculate_match_score(self, matched_count: int, total_count: int) -> float:
        """Calculate match score based on matched vs total conditions"""
        if total_count == 0:
            return 1.0
        return matched_count / total_count
    
    # Condition evaluator methods (for future expansion)
    async def _evaluate_medication_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate medication-specific conditions"""
        # Implementation for complex medication conditions
        return await self._evaluate_condition(condition, analyzed_request)
    
    async def _evaluate_patient_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate patient-specific conditions"""
        # Implementation for complex patient conditions
        return await self._evaluate_condition(condition, analyzed_request)
    
    async def _evaluate_situation_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate situational conditions"""
        # Implementation for complex situational conditions
        return await self._evaluate_condition(condition, analyzed_request)
    
    async def _evaluate_context_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate enriched context conditions"""
        # Implementation for complex context conditions
        return await self._evaluate_condition(condition, analyzed_request)


class RuleScorer:
    """Advanced rule scoring engine with multi-dimensional scoring"""

    def __init__(self):
        self.clinical_priority_weights = {
            RiskLevel.CRITICAL: 1.0,
            RiskLevel.HIGH: 0.8,
            RiskLevel.MODERATE: 0.6,
            RiskLevel.LOW: 0.4
        }

        self.evidence_level_weights = {
            'high': 1.0,
            'moderate': 0.8,
            'low': 0.6,
            'expert_opinion': 0.4
        }

    async def calculate_score(
        self,
        rule: ContextSelectionRule,
        match_result: RuleMatchResult,
        analyzed_request: AnalyzedRequest
    ) -> ScoredRule:
        """Calculate comprehensive final score using decision matrix"""

        try:
            # Component 1: Clinical Priority Score (40% weight)
            clinical_score = await self._score_clinical_priority(rule, analyzed_request)

            # Component 2: Specificity Score (30% weight)
            specificity_score = await self._score_specificity(rule, match_result)

            # Component 3: Risk Assessment Score (20% weight)
            risk_score = await self._score_risk_assessment(rule, analyzed_request)

            # Component 4: Evidence Level Score (10% weight)
            evidence_score = await self._score_evidence_level(rule)

            # Apply weighted combination
            final_score = (
                clinical_score * 0.40 +
                specificity_score * 0.30 +
                risk_score * 0.20 +
                evidence_score * 0.10
            )

            # Apply rule-specific modifiers
            final_score = await self._apply_rule_modifiers(final_score, rule, analyzed_request)

            # Generate scoring rationale
            scoring_rationale = self._generate_scoring_rationale(
                clinical_score, specificity_score, risk_score, evidence_score, final_score
            )

            return ScoredRule(
                rule=rule,
                match_result=match_result,
                final_score=final_score,
                clinical_priority_score=clinical_score,
                specificity_score=specificity_score,
                risk_assessment_score=risk_score,
                evidence_level_score=evidence_score,
                scoring_rationale=scoring_rationale
            )

        except Exception as e:
            logger.error(f"Scoring failed for rule {rule.id}: {str(e)}")
            # Return minimal score on error
            return ScoredRule(
                rule=rule,
                match_result=match_result,
                final_score=0.0,
                clinical_priority_score=0.0,
                specificity_score=0.0,
                risk_assessment_score=0.0,
                evidence_level_score=0.0,
                scoring_rationale=f"Scoring error: {str(e)}"
            )

    async def _score_clinical_priority(self, rule: ContextSelectionRule, analyzed_request: AnalyzedRequest) -> float:
        """Score based on clinical priority (life-threatening > organ failure > age-based)"""

        base_score = rule.priority / 100.0  # Normalize to 0-1

        # Boost for high-risk scenarios
        if analyzed_request.enriched_context.overall_risk_level == RiskLevel.CRITICAL:
            base_score *= 1.2
        elif analyzed_request.enriched_context.overall_risk_level == RiskLevel.HIGH:
            base_score *= 1.1

        # Boost for emergency situations
        if analyzed_request.situational_properties.urgency in [UrgencyLevel.EMERGENCY, UrgencyLevel.STAT]:
            base_score *= 1.15

        return min(base_score, 1.0)

    async def _score_specificity(self, rule: ContextSelectionRule, match_result: RuleMatchResult) -> float:
        """Score based on rule specificity (matched conditions / total conditions)"""

        total_conditions = (
            len(rule.triggers.all_of) +
            len(rule.triggers.any_of) +
            len(rule.triggers.none_of)
        )

        if total_conditions == 0:
            return 0.5  # Default for rules with no conditions

        # Use match score from rule matcher
        return match_result.match_score

    async def _score_risk_assessment(self, rule: ContextSelectionRule, analyzed_request: AnalyzedRequest) -> float:
        """Score based on patient risk factors and medication risk"""

        risk_score = 0.5  # Base score

        # Patient risk factors
        if analyzed_request.patient_properties.age_group == AgeGroup.ELDERLY:
            risk_score += 0.2

        if analyzed_request.patient_properties.renal_function in [OrganFunction.MODERATE_IMPAIRMENT, OrganFunction.SEVERE_IMPAIRMENT]:
            risk_score += 0.2

        # Medication risk factors
        if analyzed_request.medication_properties.is_high_alert:
            risk_score += 0.2

        if analyzed_request.medication_properties.is_narrow_therapeutic_index:
            risk_score += 0.15

        return min(risk_score, 1.0)

    async def _score_evidence_level(self, rule: ContextSelectionRule) -> float:
        """Score based on evidence level and guideline strength"""

        return self.evidence_level_weights.get(rule.evidence_level, 0.5)

    async def _apply_rule_modifiers(
        self,
        base_score: float,
        rule: ContextSelectionRule,
        analyzed_request: AnalyzedRequest
    ) -> float:
        """Apply rule-specific scoring modifiers"""

        modified_score = base_score

        for modifier in rule.scoring.modifiers:
            condition = modifier.get('condition', '')
            add_score = modifier.get('add_score', 0) / 100.0  # Convert to 0-1 scale

            # Evaluate modifier condition
            if await self._evaluate_modifier_condition(condition, analyzed_request):
                modified_score += add_score

        return min(modified_score, 1.0)

    async def _evaluate_modifier_condition(self, condition: str, analyzed_request: AnalyzedRequest) -> bool:
        """Evaluate modifier condition (simplified for now)"""

        try:
            # Simple condition evaluation - can be expanded
            if 'patient.age >' in condition:
                age_threshold = int(condition.split('>')[-1].strip())
                return (analyzed_request.patient_properties.age_years or 0) > age_threshold

            if 'patient.age >=' in condition:
                age_threshold = int(condition.split('>=')[-1].strip())
                return (analyzed_request.patient_properties.age_years or 0) >= age_threshold

            # Add more condition types as needed
            return False

        except Exception as e:
            logger.error(f"Modifier condition evaluation failed: {condition} - {str(e)}")
            return False

    def _generate_scoring_rationale(
        self,
        clinical_score: float,
        specificity_score: float,
        risk_score: float,
        evidence_score: float,
        final_score: float
    ) -> str:
        """Generate human-readable scoring rationale"""

        return (
            f"Clinical Priority: {clinical_score:.2f} (40%), "
            f"Specificity: {specificity_score:.2f} (30%), "
            f"Risk Assessment: {risk_score:.2f} (20%), "
            f"Evidence Level: {evidence_score:.2f} (10%) "
            f"→ Final Score: {final_score:.2f}"
        )


class ContextSelectionEngine:
    """
    Main Context Selection Engine with YAML-based rules and advanced matching

    Implements sophisticated context recipe selection using:
    - YAML-based rule definitions with scoring
    - Trigger conditions (all_of, any_of, none_of)
    - Clinical rationale and evidence levels
    - Performance-optimized matching algorithms
    """

    def __init__(self, rules_directory: str = "rules/context-selection"):
        self.rule_repository = RuleRepository(rules_directory)
        self.rule_matcher = RuleMatcher()
        self.rule_scorer = RuleScorer()

        # Performance tracking
        self.performance_stats = {
            'total_selections': 0,
            'average_selection_time_ms': 0.0,
            'cache_hits': 0,
            'rule_evaluations': 0
        }

        logger.info("Context Selection Engine initialized with YAML-based rules")

    async def select_context_recipe(self, analyzed_request: AnalyzedRequest) -> ContextRecipeSelection:
        """
        Select optimal context recipe using rule-based intelligence

        Process:
        1. Initial filter phase (< 1ms) - indexed lookups
        2. Detailed evaluation phase (< 5ms) - parallel rule evaluation
        3. Score calculation with clinical weighting
        4. Final selection with audit trail
        """

        start_time = time.time()

        try:
            logger.info(f"🎯 Starting context recipe selection for {analyzed_request.analysis_id}")

            # Step 1: Fast filter using indexed properties
            candidate_rules = await self.rule_repository.get_candidate_rules(
                medication_class=analyzed_request.medication_properties.therapeutic_class,
                patient_age_group=analyzed_request.patient_properties.age_group,
                urgency=analyzed_request.situational_properties.urgency
            )

            logger.info(f"📋 Found {len(candidate_rules)} candidate rules")

            # Step 2: Detailed rule evaluation (parallel processing)
            matched_rules = []
            evaluation_tasks = []

            for rule in candidate_rules:
                task = self.rule_matcher.evaluate_rule(rule, analyzed_request)
                evaluation_tasks.append(task)

            # Execute evaluations in parallel
            match_results = await asyncio.gather(*evaluation_tasks)

            # Step 3: Score matching rules
            for match_result in match_results:
                if match_result.matches:
                    scored_rule = await self.rule_scorer.calculate_score(
                        match_result.rule, match_result, analyzed_request
                    )
                    matched_rules.append(scored_rule)

            logger.info(f"✅ {len(matched_rules)} rules matched and scored")

            # Step 4: Select best context recipe
            if not matched_rules:
                return await self._get_default_context_recipe(analyzed_request)

            # Sort by final score (highest first)
            matched_rules.sort(key=lambda r: r.final_score, reverse=True)
            best_rule = matched_rules[0]

            # Step 5: Generate selection result
            selection_time = (time.time() - start_time) * 1000

            selection_result = ContextRecipeSelection(
                context_recipe_id=best_rule.rule.context_recipe,
                selected_rule=best_rule,
                confidence_score=best_rule.final_score,
                clinical_rationale=best_rule.rule.clinical_rationale,
                audit_trail=self._generate_audit_trail(matched_rules, best_rule, analyzed_request),
                selection_time_ms=selection_time,
                multiple_matches=len(matched_rules) > 1,
                matched_rules=matched_rules
            )

            # Update performance stats
            self._update_performance_stats(selection_time, len(candidate_rules))

            logger.info(f"🎯 Context recipe selected: {best_rule.rule.context_recipe} "
                       f"(score: {best_rule.final_score:.2f}, time: {selection_time:.1f}ms)")

            return selection_result

        except Exception as e:
            logger.error(f"❌ Context recipe selection failed: {str(e)}")
            return await self._get_default_context_recipe(analyzed_request)

    async def _get_default_context_recipe(self, analyzed_request: AnalyzedRequest) -> ContextRecipeSelection:
        """Get default context recipe when no rules match"""

        # Determine default based on basic characteristics
        if analyzed_request.medication_properties.is_high_alert:
            default_recipe = "medication_safety_comprehensive_context_v2"
        elif analyzed_request.enriched_context.overall_risk_level in [RiskLevel.HIGH, RiskLevel.CRITICAL]:
            default_recipe = "medication_safety_enhanced_context_v2"
        else:
            default_recipe = "medication_safety_base_context_v2"

        logger.warning(f"Using default context recipe: {default_recipe}")

        return ContextRecipeSelection(
            context_recipe_id=default_recipe,
            selected_rule=None,
            confidence_score=0.5,
            clinical_rationale="Default selection - no matching rules found",
            audit_trail={"selection_type": "default", "reason": "no_matching_rules"},
            selection_time_ms=1.0,
            multiple_matches=False
        )

    def _generate_audit_trail(
        self,
        matched_rules: List[ScoredRule],
        selected_rule: ScoredRule,
        analyzed_request: AnalyzedRequest
    ) -> Dict[str, Any]:
        """Generate comprehensive audit trail for selection decision"""

        return {
            "selection_timestamp": datetime.utcnow().isoformat(),
            "analysis_id": analyzed_request.analysis_id,
            "total_rules_evaluated": self.performance_stats['rule_evaluations'],
            "matched_rules_count": len(matched_rules),
            "selected_rule": {
                "id": selected_rule.rule.id,
                "name": selected_rule.rule.name,
                "final_score": selected_rule.final_score,
                "scoring_breakdown": selected_rule.scoring_rationale
            },
            "alternative_rules": [
                {
                    "id": rule.rule.id,
                    "name": rule.rule.name,
                    "score": rule.final_score
                }
                for rule in matched_rules[1:6]  # Top 5 alternatives
            ],
            "selection_criteria": {
                "medication_class": analyzed_request.medication_properties.therapeutic_class,
                "risk_level": analyzed_request.enriched_context.overall_risk_level.value,
                "urgency": analyzed_request.situational_properties.urgency.value,
                "complexity_score": analyzed_request.enriched_context.complexity_score
            }
        }

    def _update_performance_stats(self, selection_time_ms: float, rules_evaluated: int):
        """Update performance statistics"""

        self.performance_stats['total_selections'] += 1
        self.performance_stats['rule_evaluations'] += rules_evaluated

        # Update average selection time
        current_avg = self.performance_stats['average_selection_time_ms']
        total_selections = self.performance_stats['total_selections']

        self.performance_stats['average_selection_time_ms'] = (
            (current_avg * (total_selections - 1) + selection_time_ms) / total_selections
        )

    def get_performance_stats(self) -> Dict[str, Any]:
        """Get current performance statistics"""
        return self.performance_stats.copy()
