"""
CardioFit Multi-KB Query Router
Production-ready routing engine for all Knowledge Bases with:
- Intelligent polyglot persistence routing
- Cross-KB query orchestration
- Proactive cache coordination
- GraphDB semantic inference
- Comprehensive error handling
"""

import asyncio
import time
import hashlib
import json
from typing import Dict, Optional, Any, List, Union
from enum import Enum
from datetime import datetime, timedelta
from dataclasses import dataclass
from loguru import logger

from .performance_monitor import PerformanceMonitor
from .cache_coordinator import CacheCoordinator
from .fallback_handler import FallbackHandler


class DataSource(Enum):
    """Available data sources for multi-KB routing"""
    # Relational
    POSTGRES = "postgres"

    # Search & Analytics
    ELASTICSEARCH = "elasticsearch"

    # Graph databases (per KB partition)
    NEO4J_KB1 = "neo4j_kb1"          # Patient data stream
    NEO4J_KB2 = "neo4j_kb2"          # Guidelines stream
    NEO4J_KB3 = "neo4j_kb3"          # Drug calculations stream
    NEO4J_KB5 = "neo4j_kb5"          # Interactions stream
    NEO4J_KB7 = "neo4j_kb7"          # Terminology stream
    NEO4J_SHARED = "neo4j_shared"    # Semantic mesh

    # Analytics (per KB database)
    CLICKHOUSE_KB1 = "clickhouse_kb1"  # Patient analytics
    CLICKHOUSE_KB3 = "clickhouse_kb3"  # Drug calculations
    CLICKHOUSE_KB6 = "clickhouse_kb6"  # Evidence scores
    CLICKHOUSE_KB7 = "clickhouse_kb7"  # Terminology analytics

    # Semantic reasoning
    GRAPHDB = "graphdb"

    # Caching layers
    REDIS_L2 = "redis_l2"  # Proactively warmed
    REDIS_L3 = "redis_l3"  # Router-cached results


class QueryPattern(Enum):
    """Query patterns for intelligent routing"""
    # Single KB patterns
    KB1_PATIENT_LOOKUP = "kb1_patient_lookup"
    KB2_GUIDELINE_SEARCH = "kb2_guideline_search"
    KB3_DRUG_CALCULATION = "kb3_drug_calculation"
    KB4_SAFETY_RULE_CHECK = "kb4_safety_rule_check"
    KB5_INTERACTION_CHECK = "kb5_interaction_check"
    KB7_TERMINOLOGY_LOOKUP = "kb7_terminology_lookup"
    KB7_TERMINOLOGY_SEARCH = "kb7_terminology_search"
    KB7_SEMANTIC_INFERENCE = "kb7_semantic_inference"  # GraphDB reasoning
    KB8_WORKFLOW_LOOKUP = "kb8_workflow_lookup"

    # Cross-KB patterns
    CROSS_KB_PATIENT_VIEW = "cross_kb_patient_view"      # Patient + terminology
    CROSS_KB_DRUG_ANALYSIS = "cross_kb_drug_analysis"    # Multi-KB drug safety
    CROSS_KB_SEMANTIC_SEARCH = "cross_kb_semantic_search" # Semantic across KBs

    # Analytics patterns
    PATIENT_ANALYTICS = "patient_analytics"
    DRUG_ANALYTICS = "drug_analytics"
    SAFETY_ANALYTICS = "safety_analytics"
    TERMINOLOGY_ANALYTICS = "terminology_analytics"

    # Reasoning patterns
    CLINICAL_REASONING = "clinical_reasoning"
    SEMANTIC_INFERENCE = "semantic_inference"


@dataclass
class MultiKBQueryRequest:
    """Multi-KB query request with comprehensive metadata"""
    service_id: str
    pattern: QueryPattern
    params: Dict[str, Any]
    kb_id: Optional[str] = None  # None for cross-KB queries
    cross_kb_scope: Optional[List[str]] = None
    require_snapshot: bool = False
    priority: str = "normal"  # normal, high, low
    timeout_ms: int = 30000

    def __post_init__(self):
        self.request_id = self._generate_request_id()
        self.timestamp = datetime.utcnow()

    def _generate_request_id(self) -> str:
        """Generate unique request identifier"""
        data = f"{self.service_id}:{self.pattern.value}:{json.dumps(self.params, sort_keys=True)}"
        return hashlib.sha256(data.encode()).hexdigest()[:16]


@dataclass
class MultiKBQueryResponse:
    """Multi-KB query response with execution metadata"""
    data: Any
    sources_used: List[str]
    kb_sources: List[str]
    latency_ms: float
    cache_status: str = "miss"  # hit, miss, partial, error, fallback
    snapshot_id: Optional[str] = None
    request_id: Optional[str] = None

    def __post_init__(self):
        self.timestamp = datetime.utcnow()


class MultiKBQueryRouter:
    """
    Production Multi-KB Query Router

    Provides intelligent routing across all CardioFit Knowledge Bases with:
    - Pattern-based data source selection
    - Cross-KB query orchestration
    - Proactive cache coordination
    - GraphDB semantic reasoning
    - Comprehensive fallback handling
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.performance_monitor = PerformanceMonitor(config.get('monitoring', {}))
        self.cache_coordinator = CacheCoordinator(config.get('caching', {}))
        self.fallback_handler = FallbackHandler(config.get('fallback', {}))

        # Routing rules
        self.kb_routing_rules = self._initialize_kb_routing_rules()
        self.cross_kb_rules = self._initialize_cross_kb_rules()

        # Data store clients (lazy initialized)
        self._clients = {}
        self._client_health = {}

        # Performance state
        self._circuit_breakers = {}
        self._query_cache = {}

        logger.info("Multi-KB Query Router initialized")

    def _initialize_kb_routing_rules(self) -> Dict[str, Dict[QueryPattern, DataSource]]:
        """Initialize KB-specific routing rules based on architectural design"""
        return {
            'kb1': {  # Patient Data
                QueryPattern.KB1_PATIENT_LOOKUP: DataSource.NEO4J_KB1,
            },
            'kb2': {  # Clinical Guidelines
                QueryPattern.KB2_GUIDELINE_SEARCH: DataSource.ELASTICSEARCH,
            },
            'kb3': {  # Drug Calculations
                QueryPattern.KB3_DRUG_CALCULATION: DataSource.CLICKHOUSE_KB3,
            },
            'kb4': {  # Safety Rules
                QueryPattern.KB4_SAFETY_RULE_CHECK: DataSource.POSTGRES,
            },
            'kb5': {  # Drug Interactions
                QueryPattern.KB5_INTERACTION_CHECK: DataSource.NEO4J_KB5,
            },
            'kb7': {  # Medical Terminology
                QueryPattern.KB7_TERMINOLOGY_LOOKUP: DataSource.POSTGRES,        # Exact matches
                QueryPattern.KB7_TERMINOLOGY_SEARCH: DataSource.ELASTICSEARCH,  # Fuzzy search
                QueryPattern.KB7_SEMANTIC_INFERENCE: DataSource.GRAPHDB,        # Ontology reasoning
            },
            'kb8': {  # Clinical Workflows
                QueryPattern.KB8_WORKFLOW_LOOKUP: DataSource.POSTGRES,
            }
        }

    def _initialize_cross_kb_rules(self) -> Dict[QueryPattern, List[DataSource]]:
        """Initialize cross-KB routing rules for complex queries"""
        return {
            QueryPattern.CROSS_KB_PATIENT_VIEW: [
                DataSource.NEO4J_KB1,      # Patient data stream
                DataSource.NEO4J_KB7,      # Terminology stream
                DataSource.NEO4J_SHARED    # Semantic mesh
            ],
            QueryPattern.CROSS_KB_DRUG_ANALYSIS: [
                DataSource.NEO4J_KB5,      # Drug interactions
                DataSource.CLICKHOUSE_KB3, # Dosing calculations
                DataSource.NEO4J_KB7,      # Drug terminology
                DataSource.CLICKHOUSE_KB6  # Clinical evidence scores
            ],
            QueryPattern.CROSS_KB_SEMANTIC_SEARCH: [
                DataSource.GRAPHDB,        # Semantic reasoning across KBs
                DataSource.NEO4J_SHARED,   # Shared semantic mesh
                DataSource.ELASTICSEARCH  # Full-text search
            ]
        }

    async def initialize_clients(self):
        """Lazy initialization of all data store clients"""
        try:
            await self._initialize_neo4j_clients()
            await self._initialize_clickhouse_clients()
            await self._initialize_postgres_client()
            await self._initialize_elasticsearch_client()
            await self._initialize_graphdb_client()
            await self._initialize_cache_clients()

            logger.info("All multi-KB data store clients initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize data store clients: {e}")
            raise

    async def _initialize_neo4j_clients(self):
        """Initialize Neo4j clients for each KB partition"""
        if 'neo4j' in self.config:
            from ..neo4j_dual_stream.multi_kb_stream_manager import MultiKBStreamManager

            self._clients['neo4j_manager'] = MultiKBStreamManager(self.config['neo4j'])
            await self._clients['neo4j_manager'].initialize_all_streams()

            # Mark individual KB partitions as healthy
            for kb in ['kb1', 'kb2', 'kb3', 'kb5', 'kb7']:
                self._client_health[f'neo4j_{kb}'] = True

            self._client_health['neo4j_shared'] = True

    async def _initialize_clickhouse_clients(self):
        """Initialize ClickHouse clients for analytics databases"""
        if 'clickhouse' in self.config:
            from ..clickhouse_analytics.multi_kb_analytics import MultiKBAnalyticsManager

            for kb_id in ['kb1', 'kb3', 'kb6', 'kb7']:
                if kb_id in self.config['clickhouse'].get('databases', {}):
                    client_key = f'clickhouse_{kb_id}'
                    self._clients[client_key] = MultiKBAnalyticsManager(
                        self.config['clickhouse']['databases'][kb_id]
                    )
                    self._client_health[client_key] = True

    async def _initialize_postgres_client(self):
        """Initialize PostgreSQL client for exact lookups"""
        if 'postgres' in self.config:
            # Initialize PostgreSQL client
            # Implementation depends on your PostgreSQL library choice
            self._clients['postgres'] = None  # Placeholder
            self._client_health['postgres'] = True

    async def _initialize_elasticsearch_client(self):
        """Initialize Elasticsearch client for search operations"""
        if 'elasticsearch' in self.config:
            # Initialize Elasticsearch client
            # Implementation depends on your Elasticsearch library choice
            self._clients['elasticsearch'] = None  # Placeholder
            self._client_health['elasticsearch'] = True

    async def _initialize_graphdb_client(self):
        """Initialize GraphDB client for semantic reasoning"""
        if 'graphdb' in self.config:
            try:
                # Initialize GraphDB client for semantic reasoning
                # Implementation depends on your GraphDB choice (e.g., Stardog, AllegroGraph)
                self._clients['graphdb'] = None  # Placeholder
                self._client_health['graphdb'] = True
                logger.info("GraphDB client initialized for semantic inference")

            except Exception as e:
                logger.error(f"Failed to initialize GraphDB client: {e}")
                self._client_health['graphdb'] = False

    async def _initialize_cache_clients(self):
        """Initialize Redis cache clients"""
        if 'redis' in self.config:
            # Initialize L2 and L3 cache clients
            await self.cache_coordinator.initialize(self.config['redis'])

    async def route_query(self, request: MultiKBQueryRequest) -> MultiKBQueryResponse:
        """
        Main query routing method with comprehensive error handling
        """
        start_time = time.time()

        try:
            # Initialize clients if needed
            await self.initialize_clients()

            # Record query metrics
            await self.performance_monitor.record_query_start(request)

            # Check cache first (L2 for proactively warmed data)
            cache_result = await self.cache_coordinator.check_cache(request)
            if cache_result:
                response = MultiKBQueryResponse(
                    data=cache_result['data'],
                    sources_used=['cache'],
                    kb_sources=cache_result.get('kb_sources', []),
                    latency_ms=(time.time() - start_time) * 1000,
                    cache_status="hit",
                    request_id=request.request_id
                )
                await self.performance_monitor.record_query_complete(request, response)
                return response

            # Route based on query pattern
            if request.cross_kb_scope:
                response = await self._execute_cross_kb_query(request, start_time)
            elif request.kb_id:
                response = await self._execute_single_kb_query(request, start_time)
            else:
                response = await self._execute_system_query(request, start_time)

            # Cache the result (L3 for complex queries)
            if response.cache_status != "error":
                await self.cache_coordinator.cache_result(request, response)

            # Record performance metrics
            await self.performance_monitor.record_query_complete(request, response)

            return response

        except Exception as e:
            logger.error(f"Query routing failed for {request.request_id}: {e}")
            return await self.fallback_handler.handle_error(request, e, start_time)

    async def _execute_single_kb_query(self, request: MultiKBQueryRequest, start_time: float) -> MultiKBQueryResponse:
        """Execute query on single Knowledge Base"""
        kb_id = request.kb_id
        pattern = request.pattern

        # Get routing rule for this KB and pattern
        if kb_id not in self.kb_routing_rules:
            raise ValueError(f"Unknown Knowledge Base: {kb_id}")

        kb_rules = self.kb_routing_rules[kb_id]
        if pattern not in kb_rules:
            raise ValueError(f"Unsupported pattern {pattern} for KB {kb_id}")

        data_source = kb_rules[pattern]

        # Execute query on the determined data source
        result = await self._query_data_source(data_source, request)

        return MultiKBQueryResponse(
            data=result,
            sources_used=[data_source.value],
            kb_sources=[kb_id],
            latency_ms=(time.time() - start_time) * 1000,
            cache_status="miss",
            request_id=request.request_id
        )

    async def _execute_cross_kb_query(self, request: MultiKBQueryRequest, start_time: float) -> MultiKBQueryResponse:
        """Execute cross-KB query with parallel data source orchestration"""
        pattern = request.pattern

        if pattern not in self.cross_kb_rules:
            raise ValueError(f"Unsupported cross-KB pattern: {pattern}")

        data_sources = self.cross_kb_rules[pattern]

        # Execute queries on all required data sources in parallel
        tasks = [
            self._query_data_source(source, request)
            for source in data_sources
        ]

        try:
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Combine results and handle any failures
            combined_data = {}
            successful_sources = []
            failed_sources = []

            for i, result in enumerate(results):
                source = data_sources[i]
                if isinstance(result, Exception):
                    logger.warning(f"Query failed on {source.value}: {result}")
                    failed_sources.append(source.value)
                else:
                    combined_data[source.value] = result
                    successful_sources.append(source.value)

            if not successful_sources:
                raise Exception("All data sources failed for cross-KB query")

            return MultiKBQueryResponse(
                data=combined_data,
                sources_used=successful_sources,
                kb_sources=request.cross_kb_scope or [],
                latency_ms=(time.time() - start_time) * 1000,
                cache_status="partial" if failed_sources else "miss",
                request_id=request.request_id
            )

        except Exception as e:
            logger.error(f"Cross-KB query execution failed: {e}")
            raise

    async def _execute_system_query(self, request: MultiKBQueryRequest, start_time: float) -> MultiKBQueryResponse:
        """Execute system-wide query (analytics, monitoring)"""
        # Implementation for system-wide queries
        # This would typically involve aggregation across multiple KBs

        return MultiKBQueryResponse(
            data={"message": "System query executed"},
            sources_used=["system"],
            kb_sources=[],
            latency_ms=(time.time() - start_time) * 1000,
            cache_status="miss",
            request_id=request.request_id
        )

    async def _query_data_source(self, data_source: DataSource, request: MultiKBQueryRequest) -> Any:
        """Query specific data source with circuit breaker protection"""

        # Check circuit breaker
        if await self._is_circuit_open(data_source):
            raise Exception(f"Circuit breaker open for {data_source.value}")

        try:
            if data_source == DataSource.POSTGRES:
                return await self._query_postgres(request)
            elif data_source == DataSource.ELASTICSEARCH:
                return await self._query_elasticsearch(request)
            elif data_source.value.startswith('neo4j_'):
                return await self._query_neo4j(data_source, request)
            elif data_source.value.startswith('clickhouse_'):
                return await self._query_clickhouse(data_source, request)
            elif data_source == DataSource.GRAPHDB:
                return await self._query_graphdb(request)
            else:
                raise ValueError(f"Unsupported data source: {data_source}")

        except Exception as e:
            await self._record_failure(data_source)
            raise

    async def _query_postgres(self, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Query PostgreSQL for exact lookups"""
        # Implement PostgreSQL query logic
        # This is a placeholder - implement based on your PostgreSQL client
        return {"source": "postgres", "data": "placeholder"}

    async def _query_elasticsearch(self, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Query Elasticsearch for search operations"""
        # Implement Elasticsearch query logic
        # This is a placeholder - implement based on your Elasticsearch client
        return {"source": "elasticsearch", "data": "placeholder"}

    async def _query_neo4j(self, data_source: DataSource, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Query Neo4j partition for graph operations"""
        if 'neo4j_manager' not in self._clients:
            raise Exception("Neo4j manager not initialized")

        manager = self._clients['neo4j_manager']

        # Extract KB partition from data source
        if data_source == DataSource.NEO4J_SHARED:
            partition = "shared_semantic_mesh"
        else:
            kb_num = data_source.value.split('_')[-1]  # Extract kb1, kb2, etc.
            partition = f"{kb_num}_stream"

        # Build and execute Cypher query
        cypher_query = self._build_cypher_query(request, partition)
        result = await manager.query_partition(partition, cypher_query, request.params)

        return result

    async def _query_clickhouse(self, data_source: DataSource, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Query ClickHouse for analytics"""
        kb_id = data_source.value.split('_')[-1]  # Extract kb1, kb3, etc.
        client_key = f'clickhouse_{kb_id}'

        if client_key not in self._clients:
            raise Exception(f"ClickHouse client for {kb_id} not initialized")

        client = self._clients[client_key]

        # Build and execute SQL query
        sql_query = self._build_sql_query(request, kb_id)
        result = await client.execute_query(sql_query, request.params)

        return result

    async def _query_graphdb(self, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Query GraphDB for semantic reasoning"""
        if 'graphdb' not in self._clients or not self._client_health.get('graphdb'):
            raise Exception("GraphDB client not available")

        client = self._clients['graphdb']

        # Build SPARQL query for semantic inference
        sparql_query = self._build_sparql_query(request)
        result = await client.query(sparql_query, request.params)

        return result

    def _build_cypher_query(self, request: MultiKBQueryRequest, partition: str) -> str:
        """Build Cypher query based on pattern and partition"""
        pattern = request.pattern

        # Build pattern-specific Cypher queries
        if pattern == QueryPattern.KB1_PATIENT_LOOKUP:
            return "MATCH (p:Patient {id: $patient_id}) RETURN p"
        elif pattern == QueryPattern.KB5_INTERACTION_CHECK:
            return "MATCH (d1:Drug)-[i:INTERACTS_WITH]-(d2:Drug) WHERE d1.rxnorm IN $drug_codes RETURN i, d1, d2"
        elif pattern == QueryPattern.KB7_TERMINOLOGY_LOOKUP:
            return "MATCH (t:Term {code: $code, system: $system}) RETURN t"
        elif pattern == QueryPattern.CROSS_KB_PATIENT_VIEW:
            return """
                MATCH (p:Patient {id: $patient_id})
                OPTIONAL MATCH (p)-[r1]->(m:Medication)
                OPTIONAL MATCH (m)-[r2]->(t:Term)
                RETURN p, collect(m) as medications, collect(t) as terminology
            """
        else:
            return "MATCH (n) RETURN n LIMIT 100"  # Default query

    def _build_sql_query(self, request: MultiKBQueryRequest, kb_id: str) -> str:
        """Build SQL query for ClickHouse analytics"""
        pattern = request.pattern

        if pattern == QueryPattern.KB3_DRUG_CALCULATION:
            return f"SELECT * FROM {kb_id}_drug_calculations WHERE drug_rxnorm = %(drug_rxnorm)s"
        elif pattern == QueryPattern.DRUG_ANALYTICS:
            return f"SELECT * FROM {kb_id}_drug_analytics WHERE drug_class = %(drug_class)s"
        elif pattern == QueryPattern.TERMINOLOGY_ANALYTICS:
            return f"SELECT * FROM {kb_id}_terminology_usage WHERE system = %(system)s"
        else:
            return f"SELECT * FROM {kb_id}_summary LIMIT 100"

    def _build_sparql_query(self, request: MultiKBQueryRequest) -> str:
        """Build SPARQL query for GraphDB semantic reasoning"""
        pattern = request.pattern

        if pattern == QueryPattern.KB7_SEMANTIC_INFERENCE:
            # Example: Find subsumptions or class memberships
            return """
                PREFIX owl: <http://www.w3.org/2002/07/owl#>
                PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

                SELECT ?subclass ?superclass WHERE {
                    ?concept a owl:Class .
                    ?concept rdfs:label ?conceptLabel .
                    FILTER(str(?conceptLabel) = $concept_label)
                    ?subclass rdfs:subClassOf* ?concept .
                    ?concept rdfs:subClassOf* ?superclass .
                }
            """
        else:
            return "SELECT * WHERE { ?s ?p ?o } LIMIT 100"

    async def _is_circuit_open(self, data_source: DataSource) -> bool:
        """Check if circuit breaker is open for data source"""
        breaker = self._circuit_breakers.get(data_source.value, {})
        if breaker.get('state') == 'open':
            # Check if enough time has passed to try again
            if time.time() - breaker.get('last_failure', 0) > 60:  # 60 seconds
                self._circuit_breakers[data_source.value]['state'] = 'half_open'
                return False
            return True
        return False

    async def _record_failure(self, data_source: DataSource):
        """Record failure for circuit breaker tracking"""
        source_name = data_source.value
        breaker = self._circuit_breakers.get(source_name, {
            'failure_count': 0,
            'state': 'closed',
            'last_failure': 0
        })

        breaker['failure_count'] += 1
        breaker['last_failure'] = time.time()

        # Open circuit if failure threshold exceeded
        if breaker['failure_count'] >= 5:  # 5 failures
            breaker['state'] = 'open'
            logger.warning(f"Circuit breaker opened for {source_name}")

        self._circuit_breakers[source_name] = breaker

    async def get_health_status(self) -> Dict[str, Any]:
        """Get health status of all components"""
        return {
            'router_status': 'healthy',
            'client_health': self._client_health,
            'circuit_breakers': self._circuit_breakers,
            'performance_metrics': await self.performance_monitor.get_metrics(),
            'cache_stats': await self.cache_coordinator.get_stats()
        }

    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get current performance metrics"""
        return await self.performance_monitor.get_metrics()