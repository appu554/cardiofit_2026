"""
Query Router for KB7 Terminology Service
Routes queries to optimal data source based on pattern analysis
Integrates with existing PostgreSQL/Elasticsearch and new Neo4j/ClickHouse stores
"""

from typing import Dict, Optional, Any, List, Union
import asyncio
from enum import Enum
from datetime import datetime
import time
from loguru import logger
import json
import hashlib


class DataSource(Enum):
    """Available data sources for query routing"""
    POSTGRES = "postgres"
    ELASTICSEARCH = "elasticsearch"
    NEO4J_PATIENT = "neo4j_patient"
    NEO4J_SEMANTIC = "neo4j_semantic"
    CLICKHOUSE = "clickhouse"
    GRAPHDB = "graphdb"
    REDIS_CACHE = "redis_cache"


class QueryPattern(Enum):
    """Query patterns for routing decisions"""
    # Terminology patterns
    TERMINOLOGY_LOOKUP = "terminology_lookup"
    TERMINOLOGY_SEARCH = "terminology_search"
    TERMINOLOGY_VALIDATION = "terminology_validation"

    # Clinical patterns
    PATIENT_MEDICATIONS = "patient_medications"
    PATIENT_CONDITIONS = "patient_conditions"
    PATIENT_GRAPH = "patient_graph"

    # Semantic patterns
    DRUG_INTERACTIONS = "drug_interactions"
    CONTRAINDICATIONS = "contraindications"
    DRUG_ALTERNATIVES = "drug_alternatives"
    SUBSUMPTION_HIERARCHY = "subsumption_hierarchy"

    # Analytics patterns
    MEDICATION_SCORING = "medication_scoring"
    SAFETY_ANALYTICS = "safety_analytics"
    GUIDELINE_COMPLIANCE = "guideline_compliance"
    PERFORMANCE_METRICS = "performance_metrics"

    # Reasoning patterns
    CLINICAL_REASONING = "clinical_reasoning"
    SEMANTIC_INFERENCE = "semantic_inference"


class QueryRequest:
    """Encapsulates a query request with metadata"""

    def __init__(self, service_id: str, pattern: Union[QueryPattern, str],
                 params: Dict[str, Any], context: Optional[Dict] = None,
                 require_snapshot: bool = False):
        self.service_id = service_id
        self.pattern = QueryPattern(pattern) if isinstance(pattern, str) else pattern
        self.params = params
        self.context = context or {}
        self.require_snapshot = require_snapshot
        self.request_id = self._generate_request_id()
        self.timestamp = datetime.utcnow()
        self.timer = Timer()

    def _generate_request_id(self) -> str:
        """Generate unique request ID"""
        data = f"{self.service_id}{self.pattern.value}{datetime.utcnow().isoformat()}"
        return hashlib.md5(data.encode()).hexdigest()[:16]


class QueryResponse:
    """Encapsulates query response with metadata"""

    def __init__(self, data: Any, source: str, snapshot_id: Optional[str] = None,
                 latency: float = 0, cache_hit: bool = False):
        self.data = data
        self.source = source
        self.snapshot_id = snapshot_id
        self.latency = latency
        self.cache_hit = cache_hit
        self.timestamp = datetime.utcnow()


class Timer:
    """Simple timer for measuring query latency"""

    def __init__(self):
        self.start_time = time.perf_counter()

    def elapsed(self) -> float:
        """Get elapsed time in milliseconds"""
        return (time.perf_counter() - self.start_time) * 1000


class QueryRouter:
    """
    Routes queries to optimal data source based on pattern
    Integrates with all data stores and provides intelligent routing
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Query Router with all data store clients

        Args:
            config: Configuration for all data stores
        """
        self.config = config
        self.routing_rules = self._initialize_routing_rules()
        self.fallback_rules = self._initialize_fallback_rules()
        self.cache_patterns = self._initialize_cache_patterns()

        # Initialize data store clients (imported from existing modules)
        self._postgres_client = None
        self._elasticsearch_client = None
        self._neo4j_manager = None
        self._clickhouse_manager = None
        self._graphdb_client = None
        self._redis_cache = None
        self._snapshot_manager = None

        logger.info("Query Router initialized")

    async def initialize_clients(self):
        """Lazy initialization of data store clients"""
        if not self._postgres_client:
            from ..internal.database import database
            self._postgres_client = await database.get_connection()

        if not self._elasticsearch_client:
            from ..internal.elasticsearch import integration
            self._elasticsearch_client = integration.ElasticsearchIntegration(
                self.config['elasticsearch']
            )

        if not self._neo4j_manager:
            from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
            self._neo4j_manager = Neo4jDualStreamManager(self.config['neo4j'])
            await self._neo4j_manager.initialize_databases()

        if not self._clickhouse_manager:
            from ..clickhouse_runtime.manager import ClickHouseRuntimeManager
            self._clickhouse_manager = ClickHouseRuntimeManager(self.config['clickhouse'])

        if not self._snapshot_manager:
            from ..snapshot.manager import SnapshotManager
            self._snapshot_manager = SnapshotManager()

        logger.info("All data store clients initialized")

    def _initialize_routing_rules(self) -> Dict[QueryPattern, DataSource]:
        """Initialize pattern-based routing rules"""
        return {
            # Terminology patterns
            QueryPattern.TERMINOLOGY_LOOKUP: DataSource.POSTGRES,
            QueryPattern.TERMINOLOGY_SEARCH: DataSource.ELASTICSEARCH,
            QueryPattern.TERMINOLOGY_VALIDATION: DataSource.POSTGRES,

            # Patient patterns
            QueryPattern.PATIENT_MEDICATIONS: DataSource.NEO4J_PATIENT,
            QueryPattern.PATIENT_CONDITIONS: DataSource.NEO4J_PATIENT,
            QueryPattern.PATIENT_GRAPH: DataSource.NEO4J_PATIENT,

            # Semantic patterns
            QueryPattern.DRUG_INTERACTIONS: DataSource.NEO4J_SEMANTIC,
            QueryPattern.CONTRAINDICATIONS: DataSource.NEO4J_SEMANTIC,
            QueryPattern.DRUG_ALTERNATIVES: DataSource.NEO4J_SEMANTIC,
            QueryPattern.SUBSUMPTION_HIERARCHY: DataSource.NEO4J_SEMANTIC,

            # Analytics patterns
            QueryPattern.MEDICATION_SCORING: DataSource.CLICKHOUSE,
            QueryPattern.SAFETY_ANALYTICS: DataSource.CLICKHOUSE,
            QueryPattern.GUIDELINE_COMPLIANCE: DataSource.CLICKHOUSE,
            QueryPattern.PERFORMANCE_METRICS: DataSource.CLICKHOUSE,

            # Reasoning patterns
            QueryPattern.CLINICAL_REASONING: DataSource.GRAPHDB,
            QueryPattern.SEMANTIC_INFERENCE: DataSource.GRAPHDB,
        }

    def _initialize_fallback_rules(self) -> Dict[DataSource, DataSource]:
        """Initialize fallback data sources for resilience"""
        return {
            DataSource.NEO4J_SEMANTIC: DataSource.POSTGRES,
            DataSource.NEO4J_PATIENT: DataSource.POSTGRES,
            DataSource.CLICKHOUSE: DataSource.POSTGRES,
            DataSource.GRAPHDB: DataSource.NEO4J_SEMANTIC,
            DataSource.ELASTICSEARCH: DataSource.POSTGRES,
        }

    def _initialize_cache_patterns(self) -> List[QueryPattern]:
        """Patterns that should check cache first"""
        return [
            QueryPattern.TERMINOLOGY_LOOKUP,
            QueryPattern.DRUG_INTERACTIONS,
            QueryPattern.CONTRAINDICATIONS,
            QueryPattern.MEDICATION_SCORING,
        ]

    async def route_query(self, query_request: QueryRequest) -> QueryResponse:
        """
        Route query to optimal data source

        Args:
            query_request: Query request with pattern and parameters

        Returns:
            QueryResponse with data and metadata
        """
        await self.initialize_clients()

        # Create snapshot if needed
        snapshot = None
        if query_request.require_snapshot:
            snapshot = await self._snapshot_manager.create_snapshot(
                query_request.service_id,
                query_request.context
            )

        # Check cache first for cacheable patterns
        if query_request.pattern in self.cache_patterns:
            cache_response = await self._check_cache(query_request)
            if cache_response:
                return cache_response

        # Determine optimal source
        source = self._determine_source(query_request)

        # Execute query with fallback handling
        try:
            result = await self._execute_query(source, query_request, snapshot)

            # Record performance metrics
            if self._clickhouse_manager:
                self._clickhouse_manager.record_performance_metric(
                    query_type=query_request.pattern.value,
                    data_source=source.value,
                    response_time_ms=query_request.timer.elapsed(),
                    rows_returned=len(result) if isinstance(result, list) else 1
                )

            return QueryResponse(
                data=result,
                source=source.value,
                snapshot_id=snapshot.id if snapshot else None,
                latency=query_request.timer.elapsed()
            )

        except Exception as e:
            logger.error(f"Query failed for source {source}: {e}")
            return await self._handle_fallback(query_request, source, e)

    def _determine_source(self, request: QueryRequest) -> DataSource:
        """
        Determine optimal data source based on query pattern

        Args:
            request: Query request

        Returns:
            Optimal DataSource
        """
        # Use routing rules
        source = self.routing_rules.get(request.pattern, DataSource.POSTGRES)

        # Override based on context hints
        if request.context.get('force_source'):
            try:
                source = DataSource(request.context['force_source'])
            except ValueError:
                logger.warning(f"Invalid force_source: {request.context['force_source']}")

        return source

    async def _execute_query(self, source: DataSource, request: QueryRequest,
                            snapshot: Optional[Any] = None) -> Any:
        """
        Execute query against specified data source

        Args:
            source: Target data source
            request: Query request
            snapshot: Optional snapshot for consistency

        Returns:
            Query result
        """
        if source == DataSource.POSTGRES:
            return await self._query_postgres(request)
        elif source == DataSource.ELASTICSEARCH:
            return await self._query_elasticsearch(request)
        elif source == DataSource.NEO4J_PATIENT:
            return await self._query_neo4j_patient(request, snapshot)
        elif source == DataSource.NEO4J_SEMANTIC:
            return await self._query_neo4j_semantic(request, snapshot)
        elif source == DataSource.CLICKHOUSE:
            return await self._query_clickhouse(request, snapshot)
        elif source == DataSource.GRAPHDB:
            return await self._query_graphdb(request)
        else:
            raise ValueError(f"Unknown source: {source}")

    async def _query_postgres(self, request: QueryRequest) -> Any:
        """Query PostgreSQL for terminology data"""
        params = request.params

        if request.pattern == QueryPattern.TERMINOLOGY_LOOKUP:
            query = """
                SELECT concept_uuid, code, preferred_term, system, active
                FROM concepts
                WHERE code = %s AND system = %s
            """
            result = await self._postgres_client.fetchone(
                query, (params['code'], params['system'])
            )
            return dict(result) if result else None

        elif request.pattern == QueryPattern.TERMINOLOGY_VALIDATION:
            query = """
                SELECT COUNT(*) as valid
                FROM concepts
                WHERE code = ANY(%s) AND system = %s AND active = true
            """
            result = await self._postgres_client.fetchone(
                query, (params['codes'], params['system'])
            )
            return result['valid'] == len(params['codes'])

        return None

    async def _query_elasticsearch(self, request: QueryRequest) -> Any:
        """Query Elasticsearch for text search"""
        if request.pattern == QueryPattern.TERMINOLOGY_SEARCH:
            return await self._elasticsearch_client.search_concepts(
                query=request.params['query'],
                size=request.params.get('size', 10),
                filters=request.params.get('filters', {})
            )
        return None

    async def _query_neo4j_patient(self, request: QueryRequest,
                                   snapshot: Optional[Any] = None) -> Any:
        """Query Neo4j patient database"""
        if request.pattern == QueryPattern.PATIENT_MEDICATIONS:
            return await self._neo4j_manager.get_patient_medications(
                request.params['patient_id']
            )
        elif request.pattern == QueryPattern.PATIENT_CONDITIONS:
            async with self._neo4j_manager.driver.session(database="patient_data") as session:
                result = await session.run("""
                    MATCH (p:Patient {id: $patient_id})-[:HAS_CONDITION]->(c:Condition)
                    RETURN c.icd10 as code, c.name as name, c.onset_date as onset_date
                """, patient_id=request.params['patient_id'])

                conditions = []
                async for record in result:
                    conditions.append(dict(record))
                return conditions
        return None

    async def _query_neo4j_semantic(self, request: QueryRequest,
                                    snapshot: Optional[Any] = None) -> Any:
        """Query Neo4j semantic mesh database"""
        if request.pattern == QueryPattern.DRUG_INTERACTIONS:
            return await self._neo4j_manager.query_drug_interactions(
                request.params['drug_codes']
            )
        elif request.pattern == QueryPattern.CONTRAINDICATIONS:
            return await self._neo4j_manager.find_contraindications(
                request.params['drug_code'],
                request.params['condition_codes']
            )
        elif request.pattern == QueryPattern.DRUG_ALTERNATIVES:
            async with self._neo4j_manager.driver.session(database="semantic_mesh") as session:
                result = await session.run("""
                    MATCH (d1:Drug {rxnorm: $drug_code})-[:BELONGS_TO]->(dc:DrugClass)
                    <-[:BELONGS_TO]-(d2:Drug)
                    WHERE d1 <> d2
                    RETURN d2.rxnorm as rxnorm, d2.label as name
                    LIMIT 10
                """, drug_code=request.params['drug_code'])

                alternatives = []
                async for record in result:
                    alternatives.append(dict(record))
                return alternatives
        return None

    async def _query_clickhouse(self, request: QueryRequest,
                                snapshot: Optional[Any] = None) -> Any:
        """Query ClickHouse for analytics"""
        if request.pattern == QueryPattern.MEDICATION_SCORING:
            return await self._clickhouse_manager.calculate_medication_scores(
                drugs=request.params['drugs'],
                indication=request.params['indication'],
                patient_context=request.params.get('patient_context'),
                snapshot_id=snapshot.id if snapshot else None
            )
        elif request.pattern == QueryPattern.SAFETY_ANALYTICS:
            return await self._clickhouse_manager.calculate_safety_analytics(
                patient_id=request.params['patient_id'],
                medications=request.params['medications'],
                conditions=request.params['conditions']
            )
        elif request.pattern == QueryPattern.PERFORMANCE_METRICS:
            return self._clickhouse_manager.get_query_performance_stats(
                hours=request.params.get('hours', 24)
            )
        return None

    async def _query_graphdb(self, request: QueryRequest) -> Any:
        """Query GraphDB for semantic reasoning"""
        # Simplified - would implement SPARQL queries
        logger.info(f"GraphDB query for pattern: {request.pattern}")
        return {"reasoning": "result", "pattern": request.pattern.value}

    async def _check_cache(self, request: QueryRequest) -> Optional[QueryResponse]:
        """Check cache for query result"""
        if not self._redis_cache:
            return None

        cache_key = self._generate_cache_key(request)
        cached_data = await self._redis_cache.get(cache_key)

        if cached_data:
            logger.debug(f"Cache hit for key: {cache_key}")
            return QueryResponse(
                data=json.loads(cached_data),
                source=DataSource.REDIS_CACHE.value,
                cache_hit=True,
                latency=request.timer.elapsed()
            )
        return None

    def _generate_cache_key(self, request: QueryRequest) -> str:
        """Generate cache key for request"""
        key_data = f"{request.pattern.value}:{json.dumps(request.params, sort_keys=True)}"
        return f"query:{hashlib.md5(key_data.encode()).hexdigest()}"

    async def _handle_fallback(self, request: QueryRequest, failed_source: DataSource,
                              error: Exception) -> QueryResponse:
        """
        Handle fallback to alternative data source

        Args:
            request: Original query request
            failed_source: Source that failed
            error: Error that occurred

        Returns:
            QueryResponse from fallback source
        """
        fallback_source = self.fallback_rules.get(failed_source)

        if not fallback_source:
            logger.error(f"No fallback for {failed_source}: {error}")
            raise error

        logger.warning(f"Falling back from {failed_source} to {fallback_source}")

        try:
            result = await self._execute_query(fallback_source, request)
            return QueryResponse(
                data=result,
                source=f"{fallback_source.value} (fallback)",
                latency=request.timer.elapsed()
            )
        except Exception as fallback_error:
            logger.error(f"Fallback also failed: {fallback_error}")
            raise fallback_error

    async def health_check(self) -> Dict[str, Any]:
        """
        Check health of all data sources

        Returns:
            Health status for each data source
        """
        health = {
            'timestamp': datetime.utcnow().isoformat(),
            'sources': {}
        }

        # Check each data source
        if self._postgres_client:
            try:
                await self._postgres_client.fetchval("SELECT 1")
                health['sources']['postgres'] = 'healthy'
            except:
                health['sources']['postgres'] = 'unhealthy'

        if self._neo4j_manager:
            neo4j_health = await self._neo4j_manager.health_check()
            health['sources']['neo4j'] = neo4j_health['status']

        if self._clickhouse_manager:
            ch_health = self._clickhouse_manager.health_check()
            health['sources']['clickhouse'] = ch_health['status']

        # Overall health
        unhealthy = [k for k, v in health['sources'].items() if v == 'unhealthy']
        if not unhealthy:
            health['status'] = 'healthy'
        elif len(unhealthy) < len(health['sources']) / 2:
            health['status'] = 'degraded'
        else:
            health['status'] = 'unhealthy'

        return health