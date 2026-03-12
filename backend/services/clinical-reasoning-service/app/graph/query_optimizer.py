"""
Graph Query Optimizer for Clinical Assertion Engine

Optimizes SPARQL/Cypher queries for sub-100ms responses with intelligent query planning,
query rewriting, and performance monitoring.

Key Features:
- Intelligent query planning and optimization
- Query rewriting for performance
- Execution plan analysis
- Query performance monitoring
- Adaptive query optimization based on execution patterns
"""

import asyncio
import logging
import time
import re
from typing import Dict, List, Optional, Any, Tuple, Set
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from collections import defaultdict, Counter
from enum import Enum
import hashlib
import json

logger = logging.getLogger(__name__)


class QueryType(Enum):
    """Types of graph queries"""
    PATIENT_SIMILARITY = "patient_similarity"
    PATTERN_DISCOVERY = "pattern_discovery"
    RELATIONSHIP_TRAVERSAL = "relationship_traversal"
    POPULATION_CLUSTERING = "population_clustering"
    TEMPORAL_ANALYSIS = "temporal_analysis"
    SIMPLE_LOOKUP = "simple_lookup"
    COMPLEX_AGGREGATION = "complex_aggregation"


@dataclass
class QueryPlan:
    """Query execution plan"""
    query_id: str
    original_query: str
    optimized_query: str
    query_type: QueryType
    estimated_cost: float
    optimization_techniques: List[str]
    execution_order: List[str]
    index_hints: List[str]
    limit_clause: Optional[int]
    timeout_ms: int


@dataclass
class QueryPerformance:
    """Query performance metrics"""
    query_id: str
    query_hash: str
    execution_time_ms: float
    result_count: int
    cache_hit: bool
    optimization_applied: bool
    timestamp: datetime
    error: Optional[str] = None


@dataclass
class OptimizationRule:
    """Query optimization rule"""
    rule_id: str
    rule_name: str
    pattern: str
    replacement: str
    conditions: List[str]
    performance_gain: float
    applicability_score: float


class GraphQueryOptimizer:
    """
    Intelligent graph query optimizer for sub-100ms performance
    
    This optimizer analyzes SPARQL queries and applies various optimization
    techniques to achieve sub-100ms response times for clinical intelligence queries.
    """
    
    def __init__(self):
        # Performance tracking
        self.query_performance_history: List[QueryPerformance] = []
        self.query_plans: Dict[str, QueryPlan] = {}
        self.optimization_rules: List[OptimizationRule] = []
        
        # Query analysis
        self.query_patterns = defaultdict(list)
        self.performance_baselines = {}
        
        # Optimization settings
        self.target_response_time_ms = 100
        self.max_result_limit = 1000
        self.query_timeout_ms = 5000
        
        # Initialize optimization rules
        self._initialize_optimization_rules()
        
        logger.info("Graph Query Optimizer initialized")
    
    def _initialize_optimization_rules(self):
        """Initialize query optimization rules"""
        self.optimization_rules = [
            # Limit optimization
            OptimizationRule(
                rule_id="limit_large_results",
                rule_name="Add LIMIT clause for large result sets",
                pattern=r"SELECT\s+.*?\s+WHERE",
                replacement=r"\g<0> LIMIT 1000",
                conditions=["no_limit_clause", "potential_large_result"],
                performance_gain=0.7,
                applicability_score=0.9
            ),
            
            # Index hint optimization
            OptimizationRule(
                rule_id="add_index_hints",
                rule_name="Add index hints for property lookups",
                pattern=r"\?(\w+)\s+(\w+:\w+)\s+\?(\w+)",
                replacement=r"?{} {} ?{} . HINT: INDEX(?{}, {})".format(r"\1", r"\2", r"\3", r"\1", r"\2"),
                conditions=["property_lookup", "no_index_hint"],
                performance_gain=0.5,
                applicability_score=0.8
            ),
            
            # Filter pushdown optimization
            OptimizationRule(
                rule_id="filter_pushdown",
                rule_name="Push filters closer to data source",
                pattern=r"(.*?)\s+FILTER\s*\((.*?)\)",
                replacement=r"FILTER(\2) . \1",
                conditions=["filter_at_end", "can_pushdown"],
                performance_gain=0.6,
                applicability_score=0.7
            ),
            
            # Optional clause optimization
            OptimizationRule(
                rule_id="optional_optimization",
                rule_name="Optimize OPTIONAL clauses",
                pattern=r"OPTIONAL\s*\{\s*(.*?)\s*\}",
                replacement=r"OPTIONAL { \1 } FILTER(bound(\1))",
                conditions=["optional_clause", "no_bound_filter"],
                performance_gain=0.4,
                applicability_score=0.6
            ),
            
            # Union optimization
            OptimizationRule(
                rule_id="union_to_values",
                rule_name="Convert simple UNIONs to VALUES",
                pattern=r"\{\s*\?(\w+)\s+(\w+:\w+)\s+(\w+:\w+)\s*\}\s*UNION\s*\{\s*\?\1\s+\2\s+(\w+:\w+)\s*\}",
                replacement=r"VALUES ?{} {{ {} {} }}".format(r"\1", r"\3", r"\4"),
                conditions=["simple_union", "same_predicate"],
                performance_gain=0.8,
                applicability_score=0.5
            )
        ]
    
    async def optimize_query(
        self, 
        query: str, 
        query_type: QueryType = QueryType.SIMPLE_LOOKUP,
        context: Optional[Dict[str, Any]] = None
    ) -> QueryPlan:
        """
        Optimize a SPARQL query for performance
        
        Args:
            query: Original SPARQL query
            query_type: Type of query for optimization strategy
            context: Additional context for optimization
            
        Returns:
            QueryPlan: Optimized query execution plan
        """
        try:
            # Generate query ID
            query_hash = hashlib.md5(query.encode()).hexdigest()
            query_id = f"{query_type.value}_{query_hash[:8]}"
            
            # Check if we have a cached plan
            if query_id in self.query_plans:
                cached_plan = self.query_plans[query_id]
                logger.debug(f"Using cached query plan for {query_id}")
                return cached_plan
            
            # Analyze query structure
            query_analysis = await self._analyze_query(query, query_type, context)
            
            # Apply optimization techniques
            optimized_query, techniques = await self._apply_optimizations(
                query, query_analysis, query_type
            )
            
            # Estimate query cost
            estimated_cost = await self._estimate_query_cost(optimized_query, query_analysis)
            
            # Generate execution plan
            execution_order = await self._generate_execution_order(optimized_query, query_analysis)
            
            # Generate index hints
            index_hints = await self._generate_index_hints(optimized_query, query_analysis)
            
            # Determine appropriate limit
            limit_clause = await self._determine_limit(query_type, context)
            
            # Create query plan
            query_plan = QueryPlan(
                query_id=query_id,
                original_query=query,
                optimized_query=optimized_query,
                query_type=query_type,
                estimated_cost=estimated_cost,
                optimization_techniques=techniques,
                execution_order=execution_order,
                index_hints=index_hints,
                limit_clause=limit_clause,
                timeout_ms=self._calculate_timeout(estimated_cost, query_type)
            )
            
            # Cache the plan
            self.query_plans[query_id] = query_plan
            
            logger.info(f"Optimized query {query_id} with {len(techniques)} techniques "
                       f"(estimated cost: {estimated_cost:.3f})")
            
            return query_plan
            
        except Exception as e:
            logger.error(f"Error optimizing query: {e}")
            # Return basic plan with original query
            return QueryPlan(
                query_id=f"fallback_{int(time.time())}",
                original_query=query,
                optimized_query=query,
                query_type=query_type,
                estimated_cost=1.0,
                optimization_techniques=[],
                execution_order=[],
                index_hints=[],
                limit_clause=None,
                timeout_ms=self.query_timeout_ms
            )
    
    async def _analyze_query(
        self, 
        query: str, 
        query_type: QueryType, 
        context: Optional[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Analyze query structure and characteristics"""
        try:
            analysis = {
                'query_type': query_type,
                'has_limit': 'LIMIT' in query.upper(),
                'has_optional': 'OPTIONAL' in query.upper(),
                'has_union': 'UNION' in query.upper(),
                'has_filter': 'FILTER' in query.upper(),
                'has_order_by': 'ORDER BY' in query.upper(),
                'has_group_by': 'GROUP BY' in query.upper(),
                'variable_count': len(re.findall(r'\?\w+', query)),
                'triple_count': len(re.findall(r'\?\w+\s+\w+:\w+\s+\?\w+', query)),
                'complexity_score': 0.0,
                'estimated_result_size': 'unknown',
                'optimization_opportunities': []
            }
            
            # Calculate complexity score
            complexity = 0.0
            complexity += analysis['variable_count'] * 0.1
            complexity += analysis['triple_count'] * 0.2
            if analysis['has_optional']:
                complexity += 0.3
            if analysis['has_union']:
                complexity += 0.4
            if analysis['has_group_by']:
                complexity += 0.5
            
            analysis['complexity_score'] = min(complexity, 1.0)
            
            # Identify optimization opportunities
            if not analysis['has_limit']:
                analysis['optimization_opportunities'].append('add_limit')
            if analysis['has_filter'] and not analysis['has_order_by']:
                analysis['optimization_opportunities'].append('filter_pushdown')
            if analysis['has_optional']:
                analysis['optimization_opportunities'].append('optional_optimization')
            if analysis['has_union']:
                analysis['optimization_opportunities'].append('union_optimization')
            
            return analysis
            
        except Exception as e:
            logger.warning(f"Error analyzing query: {e}")
            return {'complexity_score': 0.5, 'optimization_opportunities': []}
    
    async def _apply_optimizations(
        self, 
        query: str, 
        analysis: Dict[str, Any], 
        query_type: QueryType
    ) -> Tuple[str, List[str]]:
        """Apply optimization techniques to the query"""
        try:
            optimized_query = query
            applied_techniques = []
            
            # Apply optimization rules
            for rule in self.optimization_rules:
                if self._should_apply_rule(rule, analysis, query_type):
                    try:
                        # Apply the rule
                        new_query = re.sub(rule.pattern, rule.replacement, optimized_query, flags=re.IGNORECASE)
                        if new_query != optimized_query:
                            optimized_query = new_query
                            applied_techniques.append(rule.rule_name)
                            logger.debug(f"Applied optimization rule: {rule.rule_name}")
                    except Exception as e:
                        logger.warning(f"Error applying rule {rule.rule_name}: {e}")
            
            # Add query-type specific optimizations
            if query_type == QueryType.PATIENT_SIMILARITY:
                if 'LIMIT' not in optimized_query.upper():
                    optimized_query += " LIMIT 50"
                    applied_techniques.append("similarity_limit")
            
            elif query_type == QueryType.SIMPLE_LOOKUP:
                if 'LIMIT' not in optimized_query.upper():
                    optimized_query += " LIMIT 10"
                    applied_techniques.append("lookup_limit")
            
            return optimized_query, applied_techniques
            
        except Exception as e:
            logger.warning(f"Error applying optimizations: {e}")
            return query, []

    def _should_apply_rule(self, rule: OptimizationRule, analysis: Dict[str, Any], query_type: QueryType) -> bool:
        """Determine if an optimization rule should be applied"""
        try:
            # Check rule conditions
            for condition in rule.conditions:
                if condition == "no_limit_clause" and analysis.get('has_limit', False):
                    return False
                elif condition == "potential_large_result" and query_type == QueryType.SIMPLE_LOOKUP:
                    return False
                elif condition == "filter_at_end" and not analysis.get('has_filter', False):
                    return False
                elif condition == "optional_clause" and not analysis.get('has_optional', False):
                    return False
                elif condition == "simple_union" and not analysis.get('has_union', False):
                    return False

            # Check applicability score threshold
            return rule.applicability_score > 0.5

        except Exception as e:
            logger.warning(f"Error checking rule applicability: {e}")
            return False

    async def _estimate_query_cost(self, query: str, analysis: Dict[str, Any]) -> float:
        """Estimate query execution cost"""
        try:
            cost = 0.0

            # Base cost from complexity
            cost += analysis.get('complexity_score', 0.5)

            # Cost adjustments
            if analysis.get('has_union', False):
                cost += 0.3
            if analysis.get('has_optional', False):
                cost += 0.2
            if analysis.get('has_group_by', False):
                cost += 0.4
            if not analysis.get('has_limit', False):
                cost += 0.5

            # Triple count impact
            triple_count = analysis.get('triple_count', 1)
            cost += min(triple_count * 0.1, 0.5)

            return min(cost, 1.0)

        except Exception as e:
            logger.warning(f"Error estimating query cost: {e}")
            return 0.5

    async def _generate_execution_order(self, query: str, analysis: Dict[str, Any]) -> List[str]:
        """Generate optimal execution order for query components"""
        try:
            execution_order = []

            # Basic execution order based on query structure
            if 'WHERE' in query.upper():
                execution_order.append("filter_application")

            if analysis.get('has_optional', False):
                execution_order.append("optional_processing")

            if analysis.get('has_union', False):
                execution_order.append("union_processing")

            if analysis.get('has_group_by', False):
                execution_order.append("grouping")

            if analysis.get('has_order_by', False):
                execution_order.append("sorting")

            if analysis.get('has_limit', False):
                execution_order.append("limit_application")

            return execution_order

        except Exception as e:
            logger.warning(f"Error generating execution order: {e}")
            return ["standard_execution"]

    async def _generate_index_hints(self, query: str, analysis: Dict[str, Any]) -> List[str]:
        """Generate index hints for query optimization"""
        try:
            index_hints = []

            # Property-based index hints
            property_patterns = re.findall(r'(\w+:\w+)', query)
            for prop in set(property_patterns):
                index_hints.append(f"USE_INDEX({prop})")

            # Type-based index hints
            if 'rdf:type' in query or 'a ' in query:
                index_hints.append("USE_TYPE_INDEX")

            # Temporal index hints
            if any(temporal in query.lower() for temporal in ['time', 'date', 'timestamp']):
                index_hints.append("USE_TEMPORAL_INDEX")

            return index_hints

        except Exception as e:
            logger.warning(f"Error generating index hints: {e}")
            return []

    async def _determine_limit(self, query_type: QueryType, context: Optional[Dict[str, Any]]) -> Optional[int]:
        """Determine appropriate LIMIT clause"""
        try:
            # Default limits by query type
            type_limits = {
                QueryType.SIMPLE_LOOKUP: 10,
                QueryType.PATIENT_SIMILARITY: 50,
                QueryType.PATTERN_DISCOVERY: 100,
                QueryType.RELATIONSHIP_TRAVERSAL: 200,
                QueryType.POPULATION_CLUSTERING: 500,
                QueryType.TEMPORAL_ANALYSIS: 1000,
                QueryType.COMPLEX_AGGREGATION: None
            }

            base_limit = type_limits.get(query_type, 100)

            # Adjust based on context
            if context:
                if context.get('high_priority', False):
                    return min(base_limit or 100, 20) if base_limit else 20
                elif context.get('batch_processing', False):
                    return None  # No limit for batch processing

            return base_limit

        except Exception as e:
            logger.warning(f"Error determining limit: {e}")
            return 100

    def _calculate_timeout(self, estimated_cost: float, query_type: QueryType) -> int:
        """Calculate appropriate timeout based on estimated cost"""
        try:
            # Base timeout by query type
            base_timeouts = {
                QueryType.SIMPLE_LOOKUP: 50,
                QueryType.PATIENT_SIMILARITY: 200,
                QueryType.PATTERN_DISCOVERY: 500,
                QueryType.RELATIONSHIP_TRAVERSAL: 300,
                QueryType.POPULATION_CLUSTERING: 1000,
                QueryType.TEMPORAL_ANALYSIS: 800,
                QueryType.COMPLEX_AGGREGATION: 2000
            }

            base_timeout = base_timeouts.get(query_type, 500)

            # Adjust based on estimated cost
            cost_multiplier = 1.0 + estimated_cost
            adjusted_timeout = int(base_timeout * cost_multiplier)

            # Ensure within bounds
            return min(max(adjusted_timeout, 50), self.query_timeout_ms)

        except Exception as e:
            logger.warning(f"Error calculating timeout: {e}")
            return 500

    async def record_performance(
        self,
        query_plan: QueryPlan,
        execution_time_ms: float,
        result_count: int,
        cache_hit: bool = False,
        error: Optional[str] = None
    ):
        """Record query performance for learning"""
        try:
            performance = QueryPerformance(
                query_id=query_plan.query_id,
                query_hash=hashlib.md5(query_plan.optimized_query.encode()).hexdigest(),
                execution_time_ms=execution_time_ms,
                result_count=result_count,
                cache_hit=cache_hit,
                optimization_applied=len(query_plan.optimization_techniques) > 0,
                timestamp=datetime.now(timezone.utc),
                error=error
            )

            self.query_performance_history.append(performance)

            # Keep only recent performance data (last 10,000 records)
            if len(self.query_performance_history) > 10000:
                self.query_performance_history = self.query_performance_history[-10000:]

            # Update performance baselines
            await self._update_performance_baselines(performance)

            # Learn from performance
            await self._learn_from_performance(query_plan, performance)

            logger.debug(f"Recorded performance for {query_plan.query_id}: "
                        f"{execution_time_ms:.2f}ms, {result_count} results")

        except Exception as e:
            logger.warning(f"Error recording performance: {e}")

    async def _update_performance_baselines(self, performance: QueryPerformance):
        """Update performance baselines for query types"""
        try:
            query_hash = performance.query_hash

            if query_hash not in self.performance_baselines:
                self.performance_baselines[query_hash] = {
                    'execution_times': [],
                    'result_counts': [],
                    'success_rate': 0.0,
                    'optimization_effectiveness': 0.0
                }

            baseline = self.performance_baselines[query_hash]
            baseline['execution_times'].append(performance.execution_time_ms)
            baseline['result_counts'].append(performance.result_count)

            # Keep only recent data
            if len(baseline['execution_times']) > 100:
                baseline['execution_times'] = baseline['execution_times'][-100:]
                baseline['result_counts'] = baseline['result_counts'][-100:]

            # Calculate success rate
            recent_performances = [p for p in self.query_performance_history
                                 if p.query_hash == query_hash][-50:]
            successful = sum(1 for p in recent_performances if p.error is None)
            baseline['success_rate'] = successful / len(recent_performances) if recent_performances else 0.0

        except Exception as e:
            logger.warning(f"Error updating baselines: {e}")

    async def _learn_from_performance(self, query_plan: QueryPlan, performance: QueryPerformance):
        """Learn from query performance to improve future optimizations"""
        try:
            # If performance is poor, adjust optimization rules
            if performance.execution_time_ms > self.target_response_time_ms * 2:
                # Performance is poor - reduce applicability of used techniques
                for technique in query_plan.optimization_techniques:
                    for rule in self.optimization_rules:
                        if rule.rule_name == technique:
                            rule.applicability_score *= 0.95  # Reduce confidence
                            break

            elif performance.execution_time_ms < self.target_response_time_ms:
                # Performance is good - increase applicability of used techniques
                for technique in query_plan.optimization_techniques:
                    for rule in self.optimization_rules:
                        if rule.rule_name == technique:
                            rule.applicability_score = min(rule.applicability_score * 1.05, 1.0)
                            break

        except Exception as e:
            logger.warning(f"Error learning from performance: {e}")

    def get_optimization_stats(self) -> Dict[str, Any]:
        """Get query optimization statistics"""
        try:
            if not self.query_performance_history:
                return {'total_queries': 0}

            recent_performances = [p for p in self.query_performance_history
                                 if datetime.now(timezone.utc) - p.timestamp < timedelta(hours=24)]

            if not recent_performances:
                return {'total_queries': len(self.query_performance_history)}

            # Calculate statistics
            execution_times = [p.execution_time_ms for p in recent_performances if p.error is None]

            stats = {
                'total_queries': len(self.query_performance_history),
                'recent_queries_24h': len(recent_performances),
                'average_execution_time_ms': sum(execution_times) / len(execution_times) if execution_times else 0,
                'sub_100ms_rate': sum(1 for t in execution_times if t < 100) / len(execution_times) if execution_times else 0,
                'optimization_rate': sum(1 for p in recent_performances if p.optimization_applied) / len(recent_performances),
                'cache_hit_rate': sum(1 for p in recent_performances if p.cache_hit) / len(recent_performances),
                'error_rate': sum(1 for p in recent_performances if p.error is not None) / len(recent_performances),
                'cached_plans': len(self.query_plans),
                'optimization_rules': len(self.optimization_rules)
            }

            return stats

        except Exception as e:
            logger.warning(f"Error calculating optimization stats: {e}")
            return {'error': str(e)}
