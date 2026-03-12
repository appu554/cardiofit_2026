"""
Graph-Powered Request Router for Clinical Assertion Engine

Enhanced request routing with graph intelligence capabilities including:
- Priority classification enhanced with patient similarity
- Context-aware routing based on graph patterns
- Dynamic strategy selection from graph intelligence
- Pattern-based request optimization
"""

import asyncio
import logging
from datetime import datetime, timedelta
from enum import Enum
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
import json
import hashlib
from collections import defaultdict, Counter

# Import base router
from .request_router import RequestRouter, RequestPriority, ClinicalRequest

# Import graph intelligence components
from ..graph.relationship_navigator import RelationshipNavigator, PatientSimilarity
from ..graph.pattern_discovery import PatternDiscoveryEngine, ClinicalPattern
from ..graph.population_clustering import PopulationClusteringEngine

logger = logging.getLogger(__name__)


class RoutingStrategy(Enum):
    """Dynamic routing strategies based on graph intelligence"""
    STANDARD = "standard"              # Standard priority-based routing
    SIMILARITY_ENHANCED = "similarity_enhanced"  # Enhanced with patient similarity
    PATTERN_OPTIMIZED = "pattern_optimized"     # Optimized based on discovered patterns
    POPULATION_AWARE = "population_aware"       # Population clustering-based routing
    ADAPTIVE = "adaptive"              # Dynamically adapts based on context


@dataclass
class GraphContext:
    """Graph intelligence context for routing decisions"""
    similar_patients: List[PatientSimilarity]
    relevant_patterns: List[ClinicalPattern]
    population_cluster: Optional[str]
    graph_complexity_score: float
    reasoning_path_suggestions: List[str]
    estimated_processing_time: int  # milliseconds


@dataclass
class EnhancedClinicalRequest(ClinicalRequest):
    """Clinical request enhanced with graph intelligence"""
    routing_strategy: RoutingStrategy
    graph_context: Optional[GraphContext]
    similarity_score: float
    pattern_matches: List[str]
    optimization_hints: Dict[str, Any]


class GraphPoweredRequestRouter(RequestRouter):
    """
    Graph-powered request router with intelligent routing capabilities
    
    This router extends the base RequestRouter with graph intelligence to:
    1. Enhance priority classification using patient similarity
    2. Route requests based on discovered clinical patterns
    3. Dynamically select optimal reasoning strategies
    4. Optimize performance using graph-based insights
    """
    
    def __init__(self, graphdb_endpoint: str = "http://localhost:7200"):
        super().__init__()
        
        # Initialize graph intelligence components
        self.relationship_navigator = RelationshipNavigator(graphdb_endpoint)
        self.pattern_discovery = PatternDiscoveryEngine(graphdb_endpoint)
        self.population_clustering = PopulationClusteringEngine(graphdb_endpoint)
        
        # Graph-enhanced routing statistics
        self.graph_stats = {
            'similarity_enhanced_requests': 0,
            'pattern_optimized_requests': 0,
            'population_aware_requests': 0,
            'adaptive_routing_decisions': 0,
            'graph_intelligence_cache_hits': 0,
            'average_similarity_scores': [],
            'pattern_match_rates': [],
            'routing_strategy_distribution': {s.value: 0 for s in RoutingStrategy}
        }
        
        # Intelligent caching for graph queries
        self.graph_cache = {}
        self.cache_ttl = 300  # 5 minutes
        
        # Pattern-based optimization
        self.routing_patterns = defaultdict(list)
        self.performance_patterns = defaultdict(list)
        
        logger.info("Graph-Powered Request Router initialized")
    
    async def route_request(self, raw_request: Dict[str, Any]) -> EnhancedClinicalRequest:
        """
        Enhanced request routing with graph intelligence
        
        Args:
            raw_request: Raw gRPC request data
            
        Returns:
            EnhancedClinicalRequest: Enhanced request with graph intelligence context
        """
        try:
            # Start with base routing
            base_request = await super().route_request(raw_request)
            
            # Determine optimal routing strategy
            routing_strategy = await self._determine_routing_strategy(base_request)
            
            # Build graph intelligence context
            graph_context = await self._build_graph_context(base_request, routing_strategy)
            
            # Enhance priority classification with graph intelligence
            enhanced_priority = await self._enhance_priority_with_graph_intelligence(
                base_request, graph_context
            )
            
            # Generate optimization hints
            optimization_hints = await self._generate_optimization_hints(
                base_request, graph_context
            )
            
            # Create enhanced clinical request
            enhanced_request = EnhancedClinicalRequest(
                # Base request fields
                patient_id=base_request.patient_id,
                correlation_id=base_request.correlation_id,
                reasoner_types=base_request.reasoner_types,
                medication_ids=base_request.medication_ids,
                condition_ids=base_request.condition_ids,
                allergy_ids=base_request.allergy_ids,
                priority=enhanced_priority,
                clinical_context=base_request.clinical_context,
                temporal_context=base_request.temporal_context,
                timeout_ms=base_request.timeout_ms,
                created_at=base_request.created_at,
                
                # Enhanced fields
                routing_strategy=routing_strategy,
                graph_context=graph_context,
                similarity_score=self._calculate_similarity_score(graph_context),
                pattern_matches=self._extract_pattern_matches(graph_context),
                optimization_hints=optimization_hints
            )
            
            # Update graph-enhanced statistics
            await self._update_graph_stats(enhanced_request)
            
            # Learn from routing decision for future optimization
            await self._learn_from_routing_decision(enhanced_request)
            
            logger.info(f"Enhanced routing for patient {enhanced_request.patient_id} "
                       f"using {routing_strategy.value} strategy "
                       f"(similarity: {enhanced_request.similarity_score:.3f})")
            
            return enhanced_request
            
        except Exception as e:
            logger.error(f"Error in graph-powered routing: {e}")
            # Fallback to base routing
            base_request = await super().route_request(raw_request)
            return EnhancedClinicalRequest(
                **asdict(base_request),
                routing_strategy=RoutingStrategy.STANDARD,
                graph_context=None,
                similarity_score=0.0,
                pattern_matches=[],
                optimization_hints={}
            )

    async def _determine_routing_strategy(self, request: ClinicalRequest) -> RoutingStrategy:
        """
        Determine optimal routing strategy based on request characteristics
        """
        try:
            # Check cache for similar requests
            cache_key = self._generate_cache_key(request)
            if cache_key in self.graph_cache:
                cached_data = self.graph_cache[cache_key]
                if datetime.now(timezone.utc) - cached_data['timestamp'] < timedelta(seconds=self.cache_ttl):
                    self.graph_stats['graph_intelligence_cache_hits'] += 1
                    return cached_data['strategy']

            # Analyze request complexity
            complexity_score = self._calculate_request_complexity(request)

            # Determine strategy based on complexity and context
            if complexity_score > 0.8:
                strategy = RoutingStrategy.ADAPTIVE
            elif len(request.medication_ids) > 5:
                strategy = RoutingStrategy.PATTERN_OPTIMIZED
            elif request.priority in [RequestPriority.CRITICAL, RequestPriority.HIGH]:
                strategy = RoutingStrategy.SIMILARITY_ENHANCED
            elif len(request.condition_ids) > 3:
                strategy = RoutingStrategy.POPULATION_AWARE
            else:
                strategy = RoutingStrategy.STANDARD

            # Cache the decision
            self.graph_cache[cache_key] = {
                'strategy': strategy,
                'timestamp': datetime.now(timezone.utc)
            }

            return strategy

        except Exception as e:
            logger.warning(f"Error determining routing strategy: {e}")
            return RoutingStrategy.STANDARD

    async def _build_graph_context(self, request: ClinicalRequest, strategy: RoutingStrategy) -> Optional[GraphContext]:
        """
        Build graph intelligence context based on routing strategy
        """
        try:
            if strategy == RoutingStrategy.STANDARD:
                return None

            # Initialize context components
            similar_patients = []
            relevant_patterns = []
            population_cluster = None
            reasoning_path_suggestions = []

            # Build context based on strategy
            if strategy in [RoutingStrategy.SIMILARITY_ENHANCED, RoutingStrategy.ADAPTIVE]:
                similar_patients = await self._find_similar_patients(request)

            if strategy in [RoutingStrategy.PATTERN_OPTIMIZED, RoutingStrategy.ADAPTIVE]:
                relevant_patterns = await self._discover_relevant_patterns(request)

            if strategy in [RoutingStrategy.POPULATION_AWARE, RoutingStrategy.ADAPTIVE]:
                population_cluster = await self._identify_population_cluster(request)

            # Generate reasoning path suggestions
            reasoning_path_suggestions = await self._generate_reasoning_paths(
                request, similar_patients, relevant_patterns
            )

            # Calculate graph complexity score
            graph_complexity_score = self._calculate_graph_complexity(
                similar_patients, relevant_patterns, population_cluster
            )

            # Estimate processing time based on graph context
            estimated_processing_time = self._estimate_processing_time(
                request, graph_complexity_score, strategy
            )

            return GraphContext(
                similar_patients=similar_patients,
                relevant_patterns=relevant_patterns,
                population_cluster=population_cluster,
                graph_complexity_score=graph_complexity_score,
                reasoning_path_suggestions=reasoning_path_suggestions,
                estimated_processing_time=estimated_processing_time
            )

        except Exception as e:
            logger.warning(f"Error building graph context: {e}")
            return None

    async def _find_similar_patients(self, request: ClinicalRequest) -> List[PatientSimilarity]:
        """Find similar patients using graph intelligence"""
        try:
            # Build patient context for similarity search
            patient_context = {
                'patient_id': request.patient_id,
                'medications': request.medication_ids,
                'conditions': request.condition_ids,
                'allergies': request.allergy_ids,
                'clinical_context': request.clinical_context
            }

            # Use relationship navigator to find similar patients
            similar_patients = await self.relationship_navigator.find_similar_patients(
                patient_context, limit=10, similarity_threshold=0.6
            )

            return similar_patients

        except Exception as e:
            logger.warning(f"Error finding similar patients: {e}")
            return []

    async def _discover_relevant_patterns(self, request: ClinicalRequest) -> List[ClinicalPattern]:
        """Discover relevant clinical patterns for the request"""
        try:
            # Search for patterns involving the medications and conditions
            search_context = {
                'medications': request.medication_ids,
                'conditions': request.condition_ids,
                'temporal_context': request.temporal_context
            }

            # Use pattern discovery engine
            patterns = await self.pattern_discovery.discover_patterns_for_context(
                search_context, limit=5
            )

            return patterns

        except Exception as e:
            logger.warning(f"Error discovering patterns: {e}")
            return []

    async def _identify_population_cluster(self, request: ClinicalRequest) -> Optional[str]:
        """Identify population cluster for the patient"""
        try:
            # Build patient profile for clustering
            patient_profile = {
                'patient_id': request.patient_id,
                'medications': request.medication_ids,
                'conditions': request.condition_ids,
                'demographics': request.clinical_context.get('demographics', {}),
                'clinical_context': request.clinical_context
            }

            # Use population clustering to identify cluster
            cluster_info = await self.population_clustering.identify_patient_cluster(
                patient_profile
            )

            return cluster_info.get('cluster_id') if cluster_info else None

        except Exception as e:
            logger.warning(f"Error identifying population cluster: {e}")
            return None

    async def _generate_reasoning_paths(
        self,
        request: ClinicalRequest,
        similar_patients: List[PatientSimilarity],
        relevant_patterns: List[ClinicalPattern]
    ) -> List[str]:
        """Generate optimal reasoning path suggestions"""
        try:
            reasoning_paths = []

            # Standard reasoning path
            reasoning_paths.append("standard_clinical_reasoning")

            # Add similarity-based paths
            if similar_patients:
                reasoning_paths.append("similarity_enhanced_reasoning")
                if any(p.similarity_score > 0.8 for p in similar_patients):
                    reasoning_paths.append("high_similarity_fast_track")

            # Add pattern-based paths
            if relevant_patterns:
                reasoning_paths.append("pattern_optimized_reasoning")
                if any(p.confidence_score > 0.9 for p in relevant_patterns):
                    reasoning_paths.append("high_confidence_pattern_path")

            # Add complex reasoning for high-risk scenarios
            if request.priority == RequestPriority.CRITICAL:
                reasoning_paths.append("critical_care_comprehensive")

            return reasoning_paths

        except Exception as e:
            logger.warning(f"Error generating reasoning paths: {e}")
            return ["standard_clinical_reasoning"]

    async def _enhance_priority_with_graph_intelligence(
        self,
        request: ClinicalRequest,
        graph_context: Optional[GraphContext]
    ) -> RequestPriority:
        """Enhance priority classification using graph intelligence"""
        try:
            base_priority = request.priority

            if not graph_context:
                return base_priority

            # Check for high-risk patterns
            high_risk_patterns = [
                p for p in graph_context.relevant_patterns
                if 'high_risk' in p.clinical_significance.lower() or p.confidence_score > 0.9
            ]

            # Check for similar patients with adverse outcomes
            high_risk_similarities = [
                p for p in graph_context.similar_patients
                if 'adverse' in str(p.clinical_context).lower() and p.similarity_score > 0.8
            ]

            # Escalate priority if high-risk patterns or similarities found
            if high_risk_patterns or high_risk_similarities:
                if base_priority == RequestPriority.NORMAL:
                    return RequestPriority.HIGH
                elif base_priority == RequestPriority.HIGH:
                    return RequestPriority.CRITICAL

            # Consider graph complexity for processing time
            if graph_context.graph_complexity_score > 0.8:
                if base_priority == RequestPriority.NORMAL:
                    return RequestPriority.HIGH

            return base_priority

        except Exception as e:
            logger.warning(f"Error enhancing priority: {e}")
            return request.priority

    async def _generate_optimization_hints(
        self,
        request: ClinicalRequest,
        graph_context: Optional[GraphContext]
    ) -> Dict[str, Any]:
        """Generate optimization hints for request processing"""
        try:
            hints = {
                'use_parallel_processing': len(request.medication_ids) > 3,
                'enable_caching': True,
                'priority_reasoners': [],
                'skip_reasoners': [],
                'batch_compatible': False,
                'estimated_complexity': 'medium'
            }

            if not graph_context:
                return hints

            # Optimize based on similar patients
            if graph_context.similar_patients:
                hints['use_similarity_cache'] = True
                hints['similar_patient_count'] = len(graph_context.similar_patients)

                # If high similarity, can potentially skip some reasoners
                high_sim_patients = [p for p in graph_context.similar_patients if p.similarity_score > 0.9]
                if high_sim_patients:
                    hints['high_similarity_optimization'] = True

            # Optimize based on patterns
            if graph_context.relevant_patterns:
                hints['pattern_optimization'] = True
                hints['relevant_pattern_count'] = len(graph_context.relevant_patterns)

                # Prioritize reasoners based on patterns
                for pattern in graph_context.relevant_patterns:
                    if 'interaction' in pattern.pattern_type.lower():
                        hints['priority_reasoners'].append('medication_interaction')
                    elif 'dosing' in pattern.pattern_type.lower():
                        hints['priority_reasoners'].append('dosing_calculator')

            # Set complexity based on graph context
            if graph_context.graph_complexity_score > 0.8:
                hints['estimated_complexity'] = 'high'
                hints['use_parallel_processing'] = True
            elif graph_context.graph_complexity_score < 0.3:
                hints['estimated_complexity'] = 'low'
                hints['batch_compatible'] = True

            # Reasoning path optimization
            if graph_context.reasoning_path_suggestions:
                hints['suggested_reasoning_paths'] = graph_context.reasoning_path_suggestions

            return hints

        except Exception as e:
            logger.warning(f"Error generating optimization hints: {e}")
            return {'use_parallel_processing': True, 'enable_caching': True}

    def _calculate_request_complexity(self, request: ClinicalRequest) -> float:
        """Calculate request complexity score"""
        try:
            complexity = 0.0

            # Medication complexity
            complexity += min(len(request.medication_ids) * 0.1, 0.4)

            # Condition complexity
            complexity += min(len(request.condition_ids) * 0.15, 0.3)

            # Allergy complexity
            complexity += min(len(request.allergy_ids) * 0.1, 0.2)

            # Clinical context complexity
            context_items = len(request.clinical_context)
            complexity += min(context_items * 0.05, 0.1)

            return min(complexity, 1.0)

        except Exception as e:
            logger.warning(f"Error calculating complexity: {e}")
            return 0.5

    def _calculate_graph_complexity(
        self,
        similar_patients: List[PatientSimilarity],
        relevant_patterns: List[ClinicalPattern],
        population_cluster: Optional[str]
    ) -> float:
        """Calculate graph complexity score"""
        try:
            complexity = 0.0

            # Similar patients complexity
            if similar_patients:
                complexity += min(len(similar_patients) * 0.1, 0.4)
                avg_similarity = sum(p.similarity_score for p in similar_patients) / len(similar_patients)
                complexity += avg_similarity * 0.2

            # Pattern complexity
            if relevant_patterns:
                complexity += min(len(relevant_patterns) * 0.15, 0.3)
                avg_confidence = sum(p.confidence_score for p in relevant_patterns) / len(relevant_patterns)
                complexity += avg_confidence * 0.1

            # Population cluster complexity
            if population_cluster:
                complexity += 0.1

            return min(complexity, 1.0)

        except Exception as e:
            logger.warning(f"Error calculating graph complexity: {e}")
            return 0.5

    def _estimate_processing_time(
        self,
        request: ClinicalRequest,
        graph_complexity: float,
        strategy: RoutingStrategy
    ) -> int:
        """Estimate processing time in milliseconds"""
        try:
            base_time = request.timeout_ms * 0.7  # Conservative estimate

            # Adjust based on graph complexity
            complexity_multiplier = 1.0 + (graph_complexity * 0.5)

            # Adjust based on strategy
            strategy_multipliers = {
                RoutingStrategy.STANDARD: 1.0,
                RoutingStrategy.SIMILARITY_ENHANCED: 1.2,
                RoutingStrategy.PATTERN_OPTIMIZED: 1.1,
                RoutingStrategy.POPULATION_AWARE: 1.3,
                RoutingStrategy.ADAPTIVE: 1.4
            }

            strategy_multiplier = strategy_multipliers.get(strategy, 1.0)

            estimated_time = int(base_time * complexity_multiplier * strategy_multiplier)
            return min(estimated_time, request.timeout_ms)

        except Exception as e:
            logger.warning(f"Error estimating processing time: {e}")
            return request.timeout_ms

    def _calculate_similarity_score(self, graph_context: Optional[GraphContext]) -> float:
        """Calculate overall similarity score from graph context"""
        if not graph_context or not graph_context.similar_patients:
            return 0.0

        return max(p.similarity_score for p in graph_context.similar_patients)

    def _extract_pattern_matches(self, graph_context: Optional[GraphContext]) -> List[str]:
        """Extract pattern match identifiers"""
        if not graph_context or not graph_context.relevant_patterns:
            return []

        return [p.pattern_id for p in graph_context.relevant_patterns]

    def _generate_cache_key(self, request: ClinicalRequest) -> str:
        """Generate cache key for request"""
        key_data = {
            'medications': sorted(request.medication_ids),
            'conditions': sorted(request.condition_ids),
            'allergies': sorted(request.allergy_ids),
            'priority': request.priority.value
        }

        key_string = json.dumps(key_data, sort_keys=True)
        return hashlib.md5(key_string.encode()).hexdigest()

    async def _update_graph_stats(self, request: EnhancedClinicalRequest):
        """Update graph-enhanced routing statistics"""
        try:
            # Update strategy distribution
            self.graph_stats['routing_strategy_distribution'][request.routing_strategy.value] += 1

            # Update similarity scores
            if request.similarity_score > 0:
                self.graph_stats['average_similarity_scores'].append(request.similarity_score)
                # Keep only last 1000 scores
                if len(self.graph_stats['average_similarity_scores']) > 1000:
                    self.graph_stats['average_similarity_scores'] = \
                        self.graph_stats['average_similarity_scores'][-1000:]

            # Update pattern match rates
            pattern_match_rate = len(request.pattern_matches) / max(len(request.medication_ids), 1)
            self.graph_stats['pattern_match_rates'].append(pattern_match_rate)
            if len(self.graph_stats['pattern_match_rates']) > 1000:
                self.graph_stats['pattern_match_rates'] = \
                    self.graph_stats['pattern_match_rates'][-1000:]

            # Update strategy-specific counters
            if request.routing_strategy == RoutingStrategy.SIMILARITY_ENHANCED:
                self.graph_stats['similarity_enhanced_requests'] += 1
            elif request.routing_strategy == RoutingStrategy.PATTERN_OPTIMIZED:
                self.graph_stats['pattern_optimized_requests'] += 1
            elif request.routing_strategy == RoutingStrategy.POPULATION_AWARE:
                self.graph_stats['population_aware_requests'] += 1
            elif request.routing_strategy == RoutingStrategy.ADAPTIVE:
                self.graph_stats['adaptive_routing_decisions'] += 1

        except Exception as e:
            logger.warning(f"Error updating graph stats: {e}")

    async def _learn_from_routing_decision(self, request: EnhancedClinicalRequest):
        """Learn from routing decision for future optimization"""
        try:
            # Store routing pattern for learning
            pattern_key = f"{request.routing_strategy.value}_{len(request.medication_ids)}_{request.priority.value}"

            routing_data = {
                'timestamp': request.created_at,
                'similarity_score': request.similarity_score,
                'pattern_matches': len(request.pattern_matches),
                'graph_complexity': request.graph_context.graph_complexity_score if request.graph_context else 0.0,
                'estimated_time': request.graph_context.estimated_processing_time if request.graph_context else 0
            }

            self.routing_patterns[pattern_key].append(routing_data)

            # Keep only recent patterns (last 30 days)
            cutoff_date = datetime.now(timezone.utc) - timedelta(days=30)
            self.routing_patterns[pattern_key] = [
                data for data in self.routing_patterns[pattern_key]
                if data['timestamp'] > cutoff_date
            ]

        except Exception as e:
            logger.warning(f"Error learning from routing decision: {e}")

    def get_enhanced_stats(self) -> Dict[str, Any]:
        """Get enhanced routing statistics including graph intelligence metrics"""
        base_stats = super().get_stats()

        # Calculate averages
        avg_similarity = 0.0
        if self.graph_stats['average_similarity_scores']:
            avg_similarity = sum(self.graph_stats['average_similarity_scores']) / \
                           len(self.graph_stats['average_similarity_scores'])

        avg_pattern_match_rate = 0.0
        if self.graph_stats['pattern_match_rates']:
            avg_pattern_match_rate = sum(self.graph_stats['pattern_match_rates']) / \
                                   len(self.graph_stats['pattern_match_rates'])

        enhanced_stats = {
            **base_stats,
            'graph_intelligence': {
                **self.graph_stats,
                'average_similarity_score': avg_similarity,
                'average_pattern_match_rate': avg_pattern_match_rate,
                'cache_hit_rate': self.graph_stats['graph_intelligence_cache_hits'] / max(base_stats['total_requests'], 1),
                'total_routing_patterns': len(self.routing_patterns)
            }
        }

        return enhanced_stats
