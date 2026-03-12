"""
Priority Resolver - Multi-Match Resolution for Context Selection

This module implements the Priority Resolver component for the Enhanced Orchestrator,
handling complex scenarios where multiple rules match with sophisticated resolution
strategies including additive combination, hierarchical selection, and parallel execution.

Key Features:
- Additive combination for complementary contexts (renal + elderly)
- Hierarchical selection for subsumption (chemotherapy > high-alert)
- Parallel execution for independent safety dimensions
- Clinical priority-based conflict resolution
- Comprehensive resolution audit trails
"""

import logging
from typing import Dict, List, Any, Optional, Set, Tuple
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
import asyncio

from ..models.analyzed_request_models import AnalyzedRequest, RiskLevel, AgeGroup
from .context_selection_engine import ScoredRule, ContextRecipeSelection

logger = logging.getLogger(__name__)


class ResolutionStrategy(Enum):
    """Resolution strategies for multi-match scenarios"""
    SINGLE_MATCH = "single_match"
    ADDITIVE_COMBINATION = "additive_combination"
    HIERARCHICAL_SELECTION = "hierarchical_selection"
    PARALLEL_EXECUTION = "parallel_execution"
    CONFLICT_RESOLUTION = "conflict_resolution"


@dataclass
class RuleRelationship:
    """Analysis of relationships between matched rules"""
    are_complementary: bool = False
    has_hierarchy: bool = False
    are_independent: bool = False
    have_conflicts: bool = False
    
    # Detailed relationship analysis
    complementary_pairs: List[Tuple[str, str]] = field(default_factory=list)
    hierarchical_pairs: List[Tuple[str, str]] = field(default_factory=list)  # (parent, child)
    independent_groups: List[List[str]] = field(default_factory=list)
    conflicting_pairs: List[Tuple[str, str]] = field(default_factory=list)


@dataclass
class ResolvedContextRecipe:
    """Result of priority resolution with detailed reasoning"""
    primary_recipe: str
    secondary_recipes: List[str] = field(default_factory=list)
    resolution_strategy: ResolutionStrategy = ResolutionStrategy.SINGLE_MATCH
    confidence: float = 1.0
    
    # Resolution details
    selected_rules: List[ScoredRule] = field(default_factory=list)
    rejected_rules: List[ScoredRule] = field(default_factory=list)
    combination_rationale: str = ""
    
    # Audit information
    resolution_time_ms: float = 0.0
    rule_relationships: Optional[RuleRelationship] = None


class ConflictResolver:
    """Resolves conflicts between competing rules using clinical priorities"""
    
    def __init__(self):
        # Clinical priority order (higher number = higher priority)
        self.clinical_priority_order = {
            "life_threatening": 100,
            "organ_failure": 90,
            "emergency_situations": 85,
            "high_alert_medications": 80,
            "age_based_considerations": 70,
            "drug_interactions": 60,
            "monitoring_requirements": 50,
            "cost_formulary": 30,
            "convenience": 20
        }
        
        # Context recipe hierarchy (more specific > less specific)
        self.context_hierarchy = {
            # Specific combinations (highest priority)
            "anticoagulation_elderly_renal_context": 95,
            "chemotherapy_high_risk_context": 94,
            "insulin_emergency_context": 93,
            
            # Specialized contexts
            "anticoagulation_elderly_context": 85,
            "chemotherapy_neutropenia_context": 84,
            "opioid_high_alert_context": 83,
            
            # General high-alert contexts
            "high_alert_general_context": 75,
            "medication_safety_comprehensive_context": 70,
            
            # Standard contexts
            "medication_safety_enhanced_context": 60,
            "medication_safety_base_context": 50
        }
    
    async def resolve_conflicts(
        self, 
        conflicting_rules: List[ScoredRule],
        analyzed_request: AnalyzedRequest
    ) -> ScoredRule:
        """Resolve conflicts using clinical priority order"""
        
        logger.info(f"🔧 Resolving conflicts between {len(conflicting_rules)} rules")
        
        # Sort by clinical priority first, then by score
        prioritized_rules = sorted(
            conflicting_rules,
            key=lambda r: (
                self._get_clinical_priority(r, analyzed_request),
                self._get_context_hierarchy_score(r.rule.context_recipe),
                r.final_score
            ),
            reverse=True
        )
        
        selected_rule = prioritized_rules[0]
        
        logger.info(f"✅ Conflict resolved: Selected {selected_rule.rule.name} "
                   f"(clinical priority: {self._get_clinical_priority(selected_rule, analyzed_request)})")
        
        return selected_rule
    
    def _get_clinical_priority(self, rule: ScoredRule, analyzed_request: AnalyzedRequest) -> int:
        """Get clinical priority score for a rule"""
        
        priority_score = 0
        
        # Life-threatening conditions
        if analyzed_request.enriched_context.overall_risk_level == RiskLevel.CRITICAL:
            priority_score += self.clinical_priority_order["life_threatening"]
        
        # Organ failure considerations
        if (analyzed_request.patient_properties.renal_function.value in ["severe_impairment", "end_stage"] or
            analyzed_request.patient_properties.hepatic_function.value in ["severe_impairment", "end_stage"]):
            priority_score += self.clinical_priority_order["organ_failure"]
        
        # Emergency situations
        if analyzed_request.situational_properties.urgency.value in ["emergency", "stat"]:
            priority_score += self.clinical_priority_order["emergency_situations"]
        
        # High-alert medications
        if analyzed_request.medication_properties.is_high_alert:
            priority_score += self.clinical_priority_order["high_alert_medications"]
        
        # Age-based considerations
        if analyzed_request.patient_properties.age_group in [AgeGroup.ELDERLY, AgeGroup.NEONATE]:
            priority_score += self.clinical_priority_order["age_based_considerations"]
        
        return priority_score
    
    def _get_context_hierarchy_score(self, context_recipe: str) -> int:
        """Get hierarchy score for context recipe"""
        
        # Find best match in hierarchy
        for pattern, score in self.context_hierarchy.items():
            if pattern in context_recipe:
                return score
        
        return 0  # Default for unknown contexts


class CombinationEngine:
    """Combines complementary contexts and manages parallel execution"""
    
    def __init__(self):
        # Define complementary context patterns
        self.complementary_patterns = {
            ("elderly", "renal"): "additive_monitoring",
            ("high_alert", "emergency"): "enhanced_safety",
            ("pediatric", "weight_based"): "specialized_dosing",
            ("chemotherapy", "organ_impairment"): "toxicity_prevention"
        }
        
        # Define independent context dimensions
        self.independent_dimensions = {
            "safety_monitoring": ["drug_interaction", "allergy_screening"],
            "dosing_adjustment": ["renal_adjustment", "hepatic_adjustment"],
            "administration": ["iv_compatibility", "food_interactions"],
            "monitoring": ["therapeutic_monitoring", "safety_monitoring"]
        }
    
    async def combine_additive_contexts(
        self, 
        complementary_rules: List[ScoredRule],
        analyzed_request: AnalyzedRequest
    ) -> ResolvedContextRecipe:
        """Combine complementary contexts additively"""
        
        logger.info(f"🔗 Combining {len(complementary_rules)} complementary contexts")
        
        # Sort by score to determine primary context
        sorted_rules = sorted(complementary_rules, key=lambda r: r.final_score, reverse=True)
        primary_rule = sorted_rules[0]
        secondary_rules = sorted_rules[1:]
        
        # Generate combined context recipe name
        combined_recipe = await self._generate_combined_recipe_name(
            primary_rule.rule.context_recipe,
            [r.rule.context_recipe for r in secondary_rules]
        )
        
        # Calculate combined confidence
        combined_confidence = self._calculate_combined_confidence(complementary_rules)
        
        # Generate combination rationale
        rationale = self._generate_combination_rationale(complementary_rules, "additive")
        
        return ResolvedContextRecipe(
            primary_recipe=combined_recipe,
            secondary_recipes=[r.rule.context_recipe for r in secondary_rules],
            resolution_strategy=ResolutionStrategy.ADDITIVE_COMBINATION,
            confidence=combined_confidence,
            selected_rules=complementary_rules,
            combination_rationale=rationale
        )
    
    async def select_hierarchical_context(
        self, 
        hierarchical_rules: List[ScoredRule],
        analyzed_request: AnalyzedRequest
    ) -> ResolvedContextRecipe:
        """Select most specific context from hierarchical rules"""
        
        logger.info(f"🏗️ Selecting from {len(hierarchical_rules)} hierarchical contexts")
        
        # Find most specific context (highest in hierarchy)
        most_specific = max(
            hierarchical_rules,
            key=lambda r: self._get_specificity_score(r.rule.context_recipe, analyzed_request)
        )
        
        rationale = f"Selected most specific context: {most_specific.rule.name} " \
                   f"subsumes other matching contexts"
        
        return ResolvedContextRecipe(
            primary_recipe=most_specific.rule.context_recipe,
            resolution_strategy=ResolutionStrategy.HIERARCHICAL_SELECTION,
            confidence=most_specific.final_score,
            selected_rules=[most_specific],
            rejected_rules=[r for r in hierarchical_rules if r != most_specific],
            combination_rationale=rationale
        )
    
    async def execute_parallel_contexts(
        self, 
        independent_rules: List[ScoredRule],
        analyzed_request: AnalyzedRequest
    ) -> ResolvedContextRecipe:
        """Execute independent safety dimensions in parallel"""
        
        logger.info(f"⚡ Executing {len(independent_rules)} independent contexts in parallel")
        
        # Sort by score and select top contexts for parallel execution
        sorted_rules = sorted(independent_rules, key=lambda r: r.final_score, reverse=True)
        
        # Limit parallel execution to top 3 contexts for performance
        selected_rules = sorted_rules[:3]
        
        # Primary context is highest scoring
        primary_rule = selected_rules[0]
        secondary_contexts = [r.rule.context_recipe for r in selected_rules[1:]]
        
        # Calculate parallel confidence (average of selected rules)
        parallel_confidence = sum(r.final_score for r in selected_rules) / len(selected_rules)
        
        rationale = f"Parallel execution of {len(selected_rules)} independent safety dimensions"
        
        return ResolvedContextRecipe(
            primary_recipe=primary_rule.rule.context_recipe,
            secondary_recipes=secondary_contexts,
            resolution_strategy=ResolutionStrategy.PARALLEL_EXECUTION,
            confidence=parallel_confidence,
            selected_rules=selected_rules,
            combination_rationale=rationale
        )
    
    async def _generate_combined_recipe_name(
        self, 
        primary_recipe: str, 
        secondary_recipes: List[str]
    ) -> str:
        """Generate name for combined context recipe"""
        
        # For now, use primary recipe name with suffix
        # In production, this would generate actual combined recipes
        if len(secondary_recipes) > 0:
            return f"{primary_recipe}_combined"
        return primary_recipe
    
    def _calculate_combined_confidence(self, rules: List[ScoredRule]) -> float:
        """Calculate confidence for combined contexts"""
        
        # Use weighted average with decay for additional contexts
        if not rules:
            return 0.0
        
        total_weight = 0.0
        weighted_sum = 0.0
        
        for i, rule in enumerate(sorted(rules, key=lambda r: r.final_score, reverse=True)):
            weight = 1.0 / (1.0 + i * 0.3)  # Decay factor for additional rules
            weighted_sum += rule.final_score * weight
            total_weight += weight
        
        return weighted_sum / total_weight if total_weight > 0 else 0.0
    
    def _generate_combination_rationale(self, rules: List[ScoredRule], strategy: str) -> str:
        """Generate rationale for context combination"""
        
        rule_names = [r.rule.name for r in rules]
        
        if strategy == "additive":
            return f"Additive combination of complementary contexts: {', '.join(rule_names)}"
        elif strategy == "hierarchical":
            return f"Hierarchical selection from: {', '.join(rule_names)}"
        elif strategy == "parallel":
            return f"Parallel execution of independent contexts: {', '.join(rule_names)}"
        else:
            return f"Combined contexts: {', '.join(rule_names)}"
    
    def _get_specificity_score(self, context_recipe: str, analyzed_request: AnalyzedRequest) -> int:
        """Calculate specificity score for hierarchical selection"""
        
        specificity = 0
        
        # More specific contexts get higher scores
        if "elderly_renal" in context_recipe:
            specificity += 100
        elif "elderly" in context_recipe or "renal" in context_recipe:
            specificity += 80
        
        if "emergency" in context_recipe:
            specificity += 90
        
        if "high_risk" in context_recipe:
            specificity += 85
        
        if "pediatric" in context_recipe:
            specificity += 85
        
        # Base specificity for general contexts
        if "comprehensive" in context_recipe:
            specificity += 70
        elif "enhanced" in context_recipe:
            specificity += 60
        elif "base" in context_recipe:
            specificity += 50
        
        return specificity


class PriorityResolver:
    """
    Handles complex scenarios where multiple rules match

    Implements the Priority Resolver component from the Enhanced Orchestrator design:
    - Additive combination for complementary contexts (renal + elderly)
    - Hierarchical selection for subsumption (chemotherapy > high-alert)
    - Parallel execution for independent dimensions
    - Clinical priority-based conflict resolution
    """

    def __init__(self):
        self.conflict_resolver = ConflictResolver()
        self.combination_engine = CombinationEngine()

        # Performance tracking
        self.resolution_stats = {
            'total_resolutions': 0,
            'strategy_counts': {
                'single_match': 0,
                'additive_combination': 0,
                'hierarchical_selection': 0,
                'parallel_execution': 0,
                'conflict_resolution': 0
            },
            'average_resolution_time_ms': 0.0
        }

        logger.info("Priority Resolver initialized with multi-match resolution strategies")

    async def resolve_multiple_matches(
        self,
        matched_rules: List[ScoredRule],
        analyzed_request: AnalyzedRequest
    ) -> ResolvedContextRecipe:
        """
        Resolve multiple matching rules using clinical intelligence

        Resolution Strategies:
        1. Additive: Merge complementary contexts (renal + elderly)
        2. Hierarchical: Select most specific (chemotherapy > high-alert)
        3. Parallel: Execute independent safety dimensions
        4. Conflict: Use clinical priority order
        """

        start_time = time.time()

        try:
            logger.info(f"🔧 Resolving {len(matched_rules)} matching rules")

            # Handle single match (no resolution needed)
            if len(matched_rules) == 1:
                return self._create_single_match_result(matched_rules[0])

            # Step 1: Analyze rule relationships
            rule_relationships = await self._analyze_rule_relationships(matched_rules)

            logger.info(f"📊 Rule relationships: Complementary={rule_relationships.are_complementary}, "
                       f"Hierarchical={rule_relationships.has_hierarchy}, "
                       f"Independent={rule_relationships.are_independent}, "
                       f"Conflicts={rule_relationships.have_conflicts}")

            # Step 2: Determine resolution strategy and execute
            resolution_result = None

            if rule_relationships.are_complementary:
                resolution_result = await self.combination_engine.combine_additive_contexts(
                    matched_rules, analyzed_request
                )
                self.resolution_stats['strategy_counts']['additive_combination'] += 1

            elif rule_relationships.has_hierarchy:
                resolution_result = await self.combination_engine.select_hierarchical_context(
                    matched_rules, analyzed_request
                )
                self.resolution_stats['strategy_counts']['hierarchical_selection'] += 1

            elif rule_relationships.are_independent:
                resolution_result = await self.combination_engine.execute_parallel_contexts(
                    matched_rules, analyzed_request
                )
                self.resolution_stats['strategy_counts']['parallel_execution'] += 1

            else:
                # Fallback to conflict resolution
                selected_rule = await self.conflict_resolver.resolve_conflicts(
                    matched_rules, analyzed_request
                )
                resolution_result = ResolvedContextRecipe(
                    primary_recipe=selected_rule.rule.context_recipe,
                    resolution_strategy=ResolutionStrategy.CONFLICT_RESOLUTION,
                    confidence=selected_rule.final_score,
                    selected_rules=[selected_rule],
                    rejected_rules=[r for r in matched_rules if r != selected_rule],
                    combination_rationale=f"Conflict resolution selected: {selected_rule.rule.name}"
                )
                self.resolution_stats['strategy_counts']['conflict_resolution'] += 1

            # Step 3: Finalize resolution result
            resolution_time = (time.time() - start_time) * 1000
            resolution_result.resolution_time_ms = resolution_time
            resolution_result.rule_relationships = rule_relationships

            # Update performance stats
            self._update_resolution_stats(resolution_time)

            logger.info(f"✅ Resolution completed using {resolution_result.resolution_strategy.value} "
                       f"strategy in {resolution_time:.1f}ms")

            return resolution_result

        except Exception as e:
            logger.error(f"❌ Priority resolution failed: {str(e)}")
            # Fallback to highest scoring rule
            best_rule = max(matched_rules, key=lambda r: r.final_score)
            return self._create_single_match_result(best_rule)

    async def _analyze_rule_relationships(self, matched_rules: List[ScoredRule]) -> RuleRelationship:
        """Analyze relationships between matched rules"""

        relationships = RuleRelationship()
        rule_contexts = [(rule.rule.id, rule.rule.context_recipe) for rule in matched_rules]

        # Analyze complementary relationships
        complementary_pairs = []
        for i, (id1, context1) in enumerate(rule_contexts):
            for j, (id2, context2) in enumerate(rule_contexts[i+1:], i+1):
                if self._are_complementary(context1, context2):
                    complementary_pairs.append((id1, id2))

        relationships.complementary_pairs = complementary_pairs
        relationships.are_complementary = len(complementary_pairs) > 0

        # Analyze hierarchical relationships
        hierarchical_pairs = []
        for i, (id1, context1) in enumerate(rule_contexts):
            for j, (id2, context2) in enumerate(rule_contexts):
                if i != j and self._is_hierarchical(context1, context2):
                    hierarchical_pairs.append((id1, id2))  # id1 is parent of id2

        relationships.hierarchical_pairs = hierarchical_pairs
        relationships.has_hierarchy = len(hierarchical_pairs) > 0

        # Analyze independence
        independent_groups = self._find_independent_groups(rule_contexts)
        relationships.independent_groups = independent_groups
        relationships.are_independent = len(independent_groups) > 1

        # Analyze conflicts (rules that compete for the same decision)
        conflicting_pairs = []
        for i, (id1, context1) in enumerate(rule_contexts):
            for j, (id2, context2) in enumerate(rule_contexts[i+1:], i+1):
                if self._are_conflicting(context1, context2):
                    conflicting_pairs.append((id1, id2))

        relationships.conflicting_pairs = conflicting_pairs
        relationships.have_conflicts = len(conflicting_pairs) > 0

        return relationships

    def _are_complementary(self, context1: str, context2: str) -> bool:
        """Check if two contexts are complementary (can be combined)"""

        # Define complementary patterns
        complementary_patterns = [
            ("elderly", "renal"),
            ("high_alert", "emergency"),
            ("pediatric", "weight_based"),
            ("chemotherapy", "neutropenia"),
            ("anticoagulant", "bleeding_risk")
        ]

        for pattern1, pattern2 in complementary_patterns:
            if ((pattern1 in context1 and pattern2 in context2) or
                (pattern2 in context1 and pattern1 in context2)):
                return True

        return False

    def _is_hierarchical(self, parent_context: str, child_context: str) -> bool:
        """Check if parent_context subsumes child_context"""

        # Define hierarchical relationships (parent -> child)
        hierarchical_patterns = [
            ("comprehensive", "enhanced"),
            ("enhanced", "base"),
            ("elderly_renal", "elderly"),
            ("elderly_renal", "renal"),
            ("high_risk", "standard"),
            ("emergency", "routine")
        ]

        for parent_pattern, child_pattern in hierarchical_patterns:
            if parent_pattern in parent_context and child_pattern in child_context:
                return True

        return False

    def _find_independent_groups(self, rule_contexts: List[Tuple[str, str]]) -> List[List[str]]:
        """Find groups of independent rules"""

        # Define independent dimensions
        independent_dimensions = {
            "safety": ["drug_interaction", "allergy", "contraindication"],
            "dosing": ["renal_adjustment", "hepatic_adjustment", "weight_based"],
            "monitoring": ["therapeutic_monitoring", "safety_monitoring", "lab_monitoring"],
            "administration": ["iv_compatibility", "food_interaction", "timing"]
        }

        groups = []
        for dimension, patterns in independent_dimensions.items():
            group = []
            for rule_id, context in rule_contexts:
                if any(pattern in context for pattern in patterns):
                    group.append(rule_id)
            if len(group) > 1:
                groups.append(group)

        return groups

    def _are_conflicting(self, context1: str, context2: str) -> bool:
        """Check if two contexts are conflicting (mutually exclusive)"""

        # Define conflicting patterns
        conflicting_patterns = [
            ("emergency", "routine"),
            ("high_dose", "low_dose"),
            ("iv_only", "oral_only"),
            ("immediate_release", "extended_release")
        ]

        for pattern1, pattern2 in conflicting_patterns:
            if ((pattern1 in context1 and pattern2 in context2) or
                (pattern2 in context1 and pattern1 in context2)):
                return True

        return False

    def _create_single_match_result(self, rule: ScoredRule) -> ResolvedContextRecipe:
        """Create result for single matching rule"""

        self.resolution_stats['strategy_counts']['single_match'] += 1

        return ResolvedContextRecipe(
            primary_recipe=rule.rule.context_recipe,
            resolution_strategy=ResolutionStrategy.SINGLE_MATCH,
            confidence=rule.final_score,
            selected_rules=[rule],
            combination_rationale=f"Single matching rule: {rule.rule.name}"
        )

    def _update_resolution_stats(self, resolution_time_ms: float):
        """Update resolution performance statistics"""

        self.resolution_stats['total_resolutions'] += 1

        # Update average resolution time
        current_avg = self.resolution_stats['average_resolution_time_ms']
        total_resolutions = self.resolution_stats['total_resolutions']

        self.resolution_stats['average_resolution_time_ms'] = (
            (current_avg * (total_resolutions - 1) + resolution_time_ms) / total_resolutions
        )

    def get_resolution_stats(self) -> Dict[str, Any]:
        """Get current resolution statistics"""
        return self.resolution_stats.copy()


# Import time module for performance tracking
import time
