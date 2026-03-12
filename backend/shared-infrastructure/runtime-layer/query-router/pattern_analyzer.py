"""
Service-Aware Query Pattern Analyzer
Advanced routing logic with service-specific optimizations and ML prediction
Implements the Go-style QueryPatternAnalyzer pattern in Python
"""

from typing import Dict, List, Optional, Any
from enum import Enum
from dataclasses import dataclass
from abc import ABC, abstractmethod

from .multi_kb_query_router import MultiKBQueryRequest, QueryPattern, DataSource


class DataTier(Enum):
    """Data tier hierarchy for optimal routing"""
    TIER_DIRECT = "direct"           # Direct to Rust/Go engines
    TIER_NEO4J = "neo4j"            # Graph database tier
    TIER_CLICKHOUSE = "clickhouse"   # Analytics tier
    TIER_POSTGRES = "postgres"      # Relational tier
    TIER_ELASTICSEARCH = "elasticsearch"  # Search tier
    TIER_GRAPHDB = "graphdb"        # Semantic reasoning tier
    TIER_CACHE = "cache"            # Cache tier


class CacheStrategy(Enum):
    """Cache strategy specifications"""
    CACHE_NONE = "none"                 # No caching
    CACHE_L2_HOT = "l2_hot"            # L2 cache, hot data, frequent access
    CACHE_L3_REFERENCE = "l3_reference" # L3 cache, reference data, longer TTL
    CACHE_HYBRID = "hybrid"             # Both L2 and L3


@dataclass
class QueryPatternConfig:
    """Configuration for query pattern routing"""
    optimal_tier: DataTier
    fallback_tiers: List[DataTier]
    cache_strategy: CacheStrategy
    parallelizable: bool
    timeout_ms: int = 30000
    priority: str = "normal"
    require_consistency: bool = False


class MLPredictor:
    """ML-based query pattern prediction (placeholder for actual ML integration)"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.model = None  # Placeholder for actual ML model

    async def predict_optimal_tier(self, request: MultiKBQueryRequest,
                                 historical_performance: Dict[str, float]) -> DataTier:
        """Predict optimal data tier based on request and historical performance"""

        # Simple rule-based prediction (replace with actual ML model)
        if request.pattern.value in ["terminology_lookup", "patient_lookup"]:
            if historical_performance.get("postgres", 0) < 100:  # < 100ms
                return DataTier.TIER_POSTGRES
            else:
                return DataTier.TIER_NEO4J

        elif "analytics" in request.pattern.value:
            return DataTier.TIER_CLICKHOUSE

        elif "semantic" in request.pattern.value:
            return DataTier.TIER_GRAPHDB

        return DataTier.TIER_POSTGRES  # Default


class ServiceAnalyzer(ABC):
    """Abstract base class for service-specific analyzers"""

    @abstractmethod
    async def analyze(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """Analyze request and return routing configuration"""
        pass


class MedicationServiceAnalyzer(ServiceAnalyzer):
    """Medication service specific routing logic"""

    async def analyze(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """Analyze medication service queries"""

        pattern_value = request.pattern.value

        if pattern_value == "drug_alternatives":
            # Graph traversal needed for drug alternatives
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_NEO4J,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L2_HOT,
                parallelizable=False,  # Graph traversal not easily parallelizable
                timeout_ms=5000
            )

        elif pattern_value == "dose_calculation":
            # Direct to Rust engine, no caching
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_DIRECT,
                fallback_tiers=[],
                cache_strategy=CacheStrategy.CACHE_NONE,
                parallelizable=False,  # Real-time calculation
                timeout_ms=2000
            )

        elif pattern_value == "scoring_matrix":
            # Pre-computed analytics
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_CLICKHOUSE,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L3_REFERENCE,
                parallelizable=True,  # Analytics can be parallelized
                timeout_ms=10000
            )

        elif pattern_value in ["drug_interactions", "contraindications"]:
            # Drug safety queries
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_NEO4J,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L2_HOT,
                parallelizable=True,  # Can check multiple drugs in parallel
                priority="high",  # Safety queries are high priority
                timeout_ms=3000
            )

        # Default medication service pattern
        return self._default_medication_pattern()

    def _default_medication_pattern(self) -> QueryPatternConfig:
        return QueryPatternConfig(
            optimal_tier=DataTier.TIER_POSTGRES,
            fallback_tiers=[DataTier.TIER_CACHE],
            cache_strategy=CacheStrategy.CACHE_L2_HOT,
            parallelizable=False
        )


class SafetyServiceAnalyzer(ServiceAnalyzer):
    """Safety service specific routing logic"""

    async def analyze(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """Analyze safety service queries"""

        pattern_value = request.pattern.value

        if pattern_value in ["safety_rules", "contraindication_check"]:
            # High priority safety checks
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_POSTGRES,  # Deterministic rules
                fallback_tiers=[DataTier.TIER_NEO4J],
                cache_strategy=CacheStrategy.CACHE_L2_HOT,
                parallelizable=True,
                priority="critical",
                require_consistency=True,
                timeout_ms=1000  # Very fast for safety
            )

        elif pattern_value == "safety_analytics":
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_CLICKHOUSE,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L3_REFERENCE,
                parallelizable=True,
                timeout_ms=15000
            )

        # Default safety pattern - prioritize speed and consistency
        return QueryPatternConfig(
            optimal_tier=DataTier.TIER_POSTGRES,
            fallback_tiers=[DataTier.TIER_CACHE],
            cache_strategy=CacheStrategy.CACHE_L2_HOT,
            parallelizable=False,
            priority="high",
            require_consistency=True
        )


class ScribeServiceAnalyzer(ServiceAnalyzer):
    """Scribe/Clinical documentation service analyzer"""

    async def analyze(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """Analyze scribe service queries"""

        pattern_value = request.pattern.value

        if pattern_value in ["clinical_notes_search", "template_search"]:
            # Text search heavy operations
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_ELASTICSEARCH,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L3_REFERENCE,
                parallelizable=True,
                timeout_ms=5000
            )

        elif pattern_value == "clinical_reasoning":
            # Complex reasoning requiring semantic analysis
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_GRAPHDB,
                fallback_tiers=[DataTier.TIER_NEO4J, DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_HYBRID,
                parallelizable=False,  # Complex reasoning not easily parallelized
                timeout_ms=20000  # Allow more time for complex reasoning
            )

        # Default scribe pattern
        return QueryPatternConfig(
            optimal_tier=DataTier.TIER_ELASTICSEARCH,
            fallback_tiers=[DataTier.TIER_POSTGRES],
            cache_strategy=CacheStrategy.CACHE_L2_HOT,
            parallelizable=True
        )


class QueryPatternAnalyzer:
    """
    Service-aware query pattern analyzer with ML prediction
    Implements the Go-style pattern analysis in Python
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.ml_predictor = MLPredictor(config.get('ml', {})) if config.get('ml_enabled') else None

        # Service-specific analyzers
        self.service_analyzers = {
            'medication': MedicationServiceAnalyzer(),
            'medication-service': MedicationServiceAnalyzer(),
            'safety': SafetyServiceAnalyzer(),
            'safety-gateway': SafetyServiceAnalyzer(),
            'scribe': ScribeServiceAnalyzer(),
            'clinical-reasoning': ScribeServiceAnalyzer(),
        }

        # Pattern cache for performance
        self.pattern_cache: Dict[str, QueryPatternConfig] = {}

    async def analyze(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """
        Main analysis method - routes to service-specific analyzers
        """

        # Check cache first
        cache_key = self._generate_cache_key(request)
        if cache_key in self.pattern_cache:
            return self.pattern_cache[cache_key]

        # Service-specific analysis
        service_id = request.service_id.lower()

        if service_id in self.service_analyzers:
            config = await self.service_analyzers[service_id].analyze(request)
        else:
            config = await self._default_pattern_analysis(request)

        # ML enhancement if available
        if self.ml_predictor and self.config.get('ml_enabled'):
            config = await self._enhance_with_ml(request, config)

        # Cache the result
        self.pattern_cache[cache_key] = config

        return config

    async def _default_pattern_analysis(self, request: MultiKBQueryRequest) -> QueryPatternConfig:
        """Default pattern analysis for unknown services"""

        pattern_value = request.pattern.value

        # Basic pattern recognition
        if "terminology" in pattern_value:
            if "lookup" in pattern_value:
                return QueryPatternConfig(
                    optimal_tier=DataTier.TIER_POSTGRES,
                    fallback_tiers=[DataTier.TIER_CACHE],
                    cache_strategy=CacheStrategy.CACHE_L2_HOT,
                    parallelizable=True
                )
            elif "search" in pattern_value:
                return QueryPatternConfig(
                    optimal_tier=DataTier.TIER_ELASTICSEARCH,
                    fallback_tiers=[DataTier.TIER_POSTGRES],
                    cache_strategy=CacheStrategy.CACHE_L3_REFERENCE,
                    parallelizable=True
                )

        elif "analytics" in pattern_value:
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_CLICKHOUSE,
                fallback_tiers=[DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_L3_REFERENCE,
                parallelizable=True
            )

        elif "semantic" in pattern_value or "reasoning" in pattern_value:
            return QueryPatternConfig(
                optimal_tier=DataTier.TIER_GRAPHDB,
                fallback_tiers=[DataTier.TIER_NEO4J, DataTier.TIER_POSTGRES],
                cache_strategy=CacheStrategy.CACHE_HYBRID,
                parallelizable=False
            )

        # Fallback default
        return QueryPatternConfig(
            optimal_tier=DataTier.TIER_POSTGRES,
            fallback_tiers=[DataTier.TIER_CACHE],
            cache_strategy=CacheStrategy.CACHE_L2_HOT,
            parallelizable=False
        )

    async def _enhance_with_ml(self, request: MultiKBQueryRequest,
                             config: QueryPatternConfig) -> QueryPatternConfig:
        """Enhance configuration with ML predictions"""

        # Get historical performance data (placeholder)
        historical_performance = await self._get_historical_performance(request)

        # ML prediction for optimal tier
        predicted_tier = await self.ml_predictor.predict_optimal_tier(
            request, historical_performance
        )

        # Use ML prediction if confidence is high
        if predicted_tier != config.optimal_tier:
            # Create enhanced config with ML suggestion
            return QueryPatternConfig(
                optimal_tier=predicted_tier,
                fallback_tiers=[config.optimal_tier] + config.fallback_tiers,
                cache_strategy=config.cache_strategy,
                parallelizable=config.parallelizable,
                timeout_ms=config.timeout_ms,
                priority=config.priority
            )

        return config

    async def _get_historical_performance(self, request: MultiKBQueryRequest) -> Dict[str, float]:
        """Get historical performance data for similar requests"""
        # Placeholder for actual performance data retrieval
        return {
            "postgres": 95.0,
            "neo4j": 150.0,
            "clickhouse": 80.0,
            "elasticsearch": 120.0,
            "graphdb": 250.0
        }

    def _generate_cache_key(self, request: MultiKBQueryRequest) -> str:
        """Generate cache key for pattern analysis"""
        return f"{request.service_id}:{request.pattern.value}:{len(request.params)}"

    def get_tier_mapping(self) -> Dict[DataTier, DataSource]:
        """Map data tiers to actual data sources"""
        return {
            DataTier.TIER_POSTGRES: DataSource.POSTGRES,
            DataTier.TIER_NEO4J: DataSource.NEO4J_KB7,  # Default to KB7
            DataTier.TIER_CLICKHOUSE: DataSource.CLICKHOUSE_KB7,
            DataTier.TIER_ELASTICSEARCH: DataSource.ELASTICSEARCH,
            DataTier.TIER_GRAPHDB: DataSource.GRAPHDB,
            DataTier.TIER_CACHE: DataSource.REDIS_L2,
            # TIER_DIRECT would need special handling
        }

    async def get_analyzer_stats(self) -> Dict[str, Any]:
        """Get analyzer statistics and performance metrics"""
        return {
            'cache_size': len(self.pattern_cache),
            'service_analyzers': list(self.service_analyzers.keys()),
            'ml_enabled': self.ml_predictor is not None,
            'total_patterns_cached': len(self.pattern_cache)
        }