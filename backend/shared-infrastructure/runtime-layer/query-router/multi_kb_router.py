"""
Shared Multi-KB Query Router
Routes queries to optimal data sources across ALL CardioFit Knowledge Bases

Supports intelligent routing based on:
- Knowledge Base ID (kb-1, kb-2, kb-7, etc.)
- Query patterns (lookup, search, analytics, reasoning)
- Data source capabilities and performance
- Cross-KB query optimization
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
    """Available data sources for multi-KB routing"""
    POSTGRES = "postgres"
    ELASTICSEARCH = "elasticsearch"
    NEO4J_KB1 = "neo4j_kb1"
    NEO4J_KB2 = "neo4j_kb2"
    NEO4J_KB3 = "neo4j_kb3"
    NEO4J_KB5 = "neo4j_kb5"
    NEO4J_KB7 = "neo4j_kb7"
    NEO4J_SHARED = "neo4j_shared"
    CLICKHOUSE_KB1 = "clickhouse_kb1"
    CLICKHOUSE_KB3 = "clickhouse_kb3"
    CLICKHOUSE_KB6 = "clickhouse_kb6"
    CLICKHOUSE_KB7 = "clickhouse_kb7"
    GRAPHDB = "graphdb"
    REDIS_L2 = "redis_l2"
    REDIS_L3 = "redis_l3"


class QueryPattern(Enum):
    """Query patterns for intelligent routing"""
    # KB-specific patterns
    KB1_PATIENT_LOOKUP = "kb1_patient_lookup"
    KB2_GUIDELINE_SEARCH = "kb2_guideline_search"
    KB3_DRUG_CALCULATION = "kb3_drug_calculation"
    KB5_INTERACTION_CHECK = "kb5_interaction_check"
    KB7_TERMINOLOGY_LOOKUP = "kb7_terminology_lookup"
    KB7_TERMINOLOGY_SEARCH = "kb7_terminology_search"

    # Cross-KB patterns
    CROSS_KB_PATIENT_VIEW = "cross_kb_patient_view"
    CROSS_KB_DRUG_ANALYSIS = "cross_kb_drug_analysis"
    CROSS_KB_SEMANTIC_SEARCH = "cross_kb_semantic_search"

    # Analytics patterns
    PATIENT_ANALYTICS = "patient_analytics"
    DRUG_ANALYTICS = "drug_analytics"
    SAFETY_ANALYTICS = "safety_analytics"
    TERMINOLOGY_ANALYTICS = "terminology_analytics"

    # Reasoning patterns
    CLINICAL_REASONING = "clinical_reasoning"
    SEMANTIC_INFERENCE = "semantic_inference"


class MultiKBQueryRequest:
    """Encapsulates a multi-KB query request with metadata"""

    def __init__(self,
                 service_id: str,
                 kb_id: Optional[str],
                 pattern: Union[QueryPattern, str],
                 params: Dict[str, Any],
                 require_snapshot: bool = False,
                 cross_kb_scope: Optional[List[str]] = None,
                 priority: str = "normal"):
        self.service_id = service_id
        self.kb_id = kb_id  # None for cross-KB queries
        self.pattern = QueryPattern(pattern) if isinstance(pattern, str) else pattern
        self.params = params
        self.require_snapshot = require_snapshot
        self.cross_kb_scope = cross_kb_scope or []
        self.priority = priority
        self.timestamp = datetime.utcnow()
        self.request_id = self._generate_request_id()
        self.timer = time.time()

    def _generate_request_id(self) -> str:
        """Generate unique request ID"""
        data = f"{self.service_id}:{self.kb_id}:{self.pattern.value}:{self.timestamp}"
        return hashlib.md5(data.encode()).hexdigest()[:12]

    def elapsed_time(self) -> float:
        """Get elapsed time in milliseconds"""
        return (time.time() - self.timer) * 1000


class MultiKBQueryResponse:
    """Response from multi-KB query with metadata"""

    def __init__(self,
                 data: Any,
                 sources_used: List[str],
                 kb_sources: List[str],
                 latency: float,
                 snapshot_id: Optional[str] = None,
                 cache_status: str = "miss"):
        self.data = data
        self.sources_used = sources_used
        self.kb_sources = kb_sources
        self.latency = latency
        self.snapshot_id = snapshot_id
        self.cache_status = cache_status
        self.timestamp = datetime.utcnow()


class MultiKBQueryRouter:
    """
    Routes queries to optimal data sources across all CardioFit Knowledge Bases
    Provides intelligent routing, caching, and cross-KB query optimization
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Multi-KB Query Router

        Args:
            config: Configuration for all data stores and KBs
        """
        self.config = config
        self.kb_routing_rules = self._initialize_kb_routing_rules()
        self.cross_kb_rules = self._initialize_cross_kb_rules()
        self.fallback_rules = self._initialize_fallback_rules()

        # Data store clients (lazy initialized)
        self._postgres_client = None
        self._elasticsearch_client = None
        self._neo4j_manager = None
        self._clickhouse_clients = {}  # Per-KB ClickHouse clients
        self._graphdb_client = None
        self._redis_l2_client = None
        self._redis_l3_client = None
        self._snapshot_manager = None

        # Performance metrics
        self.metrics = {
            'total_queries': 0,
            'kb_query_counts': {},
            'cross_kb_queries': 0,
            'average_latency': 0.0,
            'cache_hit_rate': 0.0,
            'error_rate': 0.0
        }

        logger.info("Multi-KB Query Router initialized")

    def _initialize_kb_routing_rules(self) -> Dict[str, Dict[QueryPattern, DataSource]]:
        """Initialize KB-specific routing rules"""
        return {
            'kb1': {  # Patient Data
                QueryPattern.KB1_PATIENT_LOOKUP: DataSource.NEO4J_KB1,
            },
            'kb2': {  # Guidelines
                QueryPattern.KB2_GUIDELINE_SEARCH: DataSource.ELASTICSEARCH,
            },
            'kb3': {  # Drug Calculations
                QueryPattern.KB3_DRUG_CALCULATION: DataSource.CLICKHOUSE_KB3,
            },
            'kb5': {  # Drug Interactions
                QueryPattern.KB5_INTERACTION_CHECK: DataSource.NEO4J_KB5,
            },
            'kb7': {  # Terminology
                QueryPattern.KB7_TERMINOLOGY_LOOKUP: DataSource.POSTGRES,
                QueryPattern.KB7_TERMINOLOGY_SEARCH: DataSource.ELASTICSEARCH,
                QueryPattern.SEMANTIC_INFERENCE: DataSource.GRAPHDB,
            }
        }

    def _initialize_cross_kb_rules(self) -> Dict[QueryPattern, List[DataSource]]:
        """Initialize cross-KB routing rules"""
        return {
            QueryPattern.CROSS_KB_PATIENT_VIEW: [
                DataSource.NEO4J_KB1,     # Patient data
                DataSource.NEO4J_KB7,     # Terminology
                DataSource.NEO4J_SHARED   # Cross-KB relationships
            ],
            QueryPattern.CROSS_KB_DRUG_ANALYSIS: [
                DataSource.NEO4J_KB5,     # Interactions
                DataSource.CLICKHOUSE_KB3, # Calculations
                DataSource.NEO4J_KB7,     # Terminology
                DataSource.CLICKHOUSE_KB6  # Evidence
            ],
            QueryPattern.CROSS_KB_SEMANTIC_SEARCH: [
                DataSource.GRAPHDB,       # Semantic reasoning across KBs
                DataSource.NEO4J_SHARED,  # Shared semantic mesh
                DataSource.ELASTICSEARCH  # Text search
            ]
        }

    def _initialize_fallback_rules(self) -> Dict[DataSource, DataSource]:
        """Initialize fallback routing rules"""
        return {
            DataSource.NEO4J_KB7: DataSource.POSTGRES,  # KB7 Neo4j → PostgreSQL
            DataSource.CLICKHOUSE_KB7: DataSource.NEO4J_KB7,  # ClickHouse → Neo4j
            DataSource.ELASTICSEARCH: DataSource.POSTGRES,  # Elasticsearch → PostgreSQL
        }

    async def initialize_clients(self):
        """Lazy initialization of all data store clients"""
        if not self._neo4j_manager:
            from ..neo4j_dual_stream.multi_kb_stream_manager import MultiKBStreamManager
            self._neo4j_manager = MultiKBStreamManager(self.config['neo4j'])
            await self._neo4j_manager.initialize_all_streams()

        if not self._clickhouse_clients:
            from ..clickhouse_analytics.multi_kb_analytics import MultiKBAnalyticsManager

            # Initialize ClickHouse clients for each KB that needs analytics
            for kb_id in ['kb1', 'kb3', 'kb6', 'kb7']:
                if kb_id in self.config.get('clickhouse_databases', {}):
                    self._clickhouse_clients[kb_id] = MultiKBAnalyticsManager(
                        self.config['clickhouse_databases'][kb_id]
                    )

        # Initialize GraphDB client for semantic operations
        if not self._graphdb_client and 'graphdb' in self.config:
            from ..graphdb_semantic.multi_kb_graphdb_manager import MultiKBGraphDBManager
            self._graphdb_client = MultiKBGraphDBManager(self.config['graphdb'])
            init_success = await self._graphdb_client.initialize_connection()
            if init_success:
                logger.info("GraphDB client initialized successfully")
            else:
                logger.error("Failed to initialize GraphDB client")
                self._graphdb_client = None

        logger.info("All multi-KB data store clients initialized")

    async def route_query(self, request: MultiKBQueryRequest) -> MultiKBQueryResponse:
        """
        Route query to optimal data source(s)

        Args:
            request: Multi-KB query request

        Returns:
            Query response with metadata
        """
        start_time = time.time()

        try:
            # Initialize clients if needed
            await self.initialize_clients()

            # Update metrics
            self.metrics['total_queries'] += 1
            if request.kb_id:
                kb_count = self.metrics['kb_query_counts'].get(request.kb_id, 0)
                self.metrics['kb_query_counts'][request.kb_id] = kb_count + 1
            else:
                self.metrics['cross_kb_queries'] += 1

            # Create snapshot if required
            snapshot = None
            if request.require_snapshot:
                snapshot = await self._create_snapshot(request)

            # Determine routing strategy
            if request.kb_id and request.cross_kb_scope:
                # Cross-KB query
                result = await self._execute_cross_kb_query(request, snapshot)
            elif request.kb_id:
                # Single KB query
                result = await self._execute_single_kb_query(request, snapshot)
            else:
                # System-wide query
                result = await self._execute_system_query(request, snapshot)

            # Calculate latency
            latency = (time.time() - start_time) * 1000

            return MultiKBQueryResponse(
                data=result['data'],
                sources_used=result['sources'],
                kb_sources=result['kb_sources'],
                latency=latency,
                snapshot_id=snapshot.id if snapshot else None,
                cache_status=result.get('cache_status', 'miss')
            )

        except Exception as e:
            logger.error(f"Multi-KB query routing failed: {e}")
            return await self._handle_fallback(request, e)

    async def _execute_single_kb_query(self,
                                      request: MultiKBQueryRequest,
                                      snapshot=None) -> Dict[str, Any]:
        """Execute query on single KB"""

        kb_id = request.kb_id
        pattern = request.pattern

        # Get optimal source for this KB and pattern
        if kb_id in self.kb_routing_rules and pattern in self.kb_routing_rules[kb_id]:
            source = self.kb_routing_rules[kb_id][pattern]
        else:
            # Default to PostgreSQL for unknown patterns
            source = DataSource.POSTGRES

        # Execute query on determined source
        if source in [DataSource.NEO4J_KB1, DataSource.NEO4J_KB2, DataSource.NEO4J_KB3,
                     DataSource.NEO4J_KB5, DataSource.NEO4J_KB7]:
            result = await self._query_neo4j_kb(kb_id, request, snapshot)
        elif source.value.startswith('clickhouse'):
            result = await self._query_clickhouse_kb(kb_id, request, snapshot)
        elif source == DataSource.POSTGRES:
            result = await self._query_postgres(request)
        elif source == DataSource.ELASTICSEARCH:
            result = await self._query_elasticsearch(request)
        elif source == DataSource.GRAPHDB:
            result = await self._query_graphdb(kb_id, request)
        else:
            raise ValueError(f"Unknown data source: {source}")

        return {
            'data': result,
            'sources': [source.value],
            'kb_sources': [kb_id]
        }

    async def _execute_cross_kb_query(self,
                                     request: MultiKBQueryRequest,
                                     snapshot=None) -> Dict[str, Any]:
        """Execute query across multiple KBs"""

        pattern = request.pattern
        kb_scope = request.cross_kb_scope

        if pattern in self.cross_kb_rules:
            sources = self.cross_kb_rules[pattern]
        else:
            # Default cross-KB strategy: Neo4j shared + relevant KBs
            sources = [DataSource.NEO4J_SHARED] + [
                getattr(DataSource, f'NEO4J_{kb.upper()}')
                for kb in kb_scope
                if hasattr(DataSource, f'NEO4J_{kb.upper()}')
            ]

        # Execute queries on multiple sources in parallel
        tasks = []
        for source in sources:
            if source.value.startswith('neo4j'):
                tasks.append(self._query_neo4j_cross_kb(kb_scope, request, snapshot))
            elif source.value.startswith('clickhouse'):
                tasks.append(self._query_clickhouse_cross_kb(kb_scope, request, snapshot))
            elif source == DataSource.GRAPHDB:
                tasks.append(self._query_graphdb_cross_kb(kb_scope, request))

        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Combine results
        combined_data = []
        sources_used = []
        for i, result in enumerate(results):
            if not isinstance(result, Exception) and result:
                combined_data.extend(result if isinstance(result, list) else [result])
                sources_used.append(sources[i].value)

        return {
            'data': combined_data,
            'sources': sources_used,
            'kb_sources': kb_scope
        }

    async def _query_neo4j_kb(self, kb_id: str, request: MultiKBQueryRequest, snapshot=None):
        """Query specific KB in Neo4j"""
        from ..neo4j_dual_stream.multi_kb_stream_manager import KnowledgeBase, StreamType

        kb_enum = KnowledgeBase(kb_id)

        # Determine stream type based on query pattern
        if 'semantic' in request.pattern.value:
            stream_type = StreamType.SEMANTIC
        else:
            stream_type = StreamType.PATIENT

        # Execute query
        cypher_query = self._build_cypher_query(request)
        return await self._neo4j_manager.query_kb_stream(
            kb_enum, stream_type, cypher_query, request.params
        )

    async def _query_neo4j_cross_kb(self, kb_list: List[str], request: MultiKBQueryRequest, snapshot=None):
        """Query across multiple KBs in Neo4j"""
        from ..neo4j_dual_stream.multi_kb_stream_manager import KnowledgeBase

        kb_enums = [KnowledgeBase(kb) for kb in kb_list]
        cypher_query = self._build_cross_kb_cypher_query(request)

        return await self._neo4j_manager.cross_kb_query(
            kb_enums, cypher_query, request.params
        )

    async def _query_clickhouse_kb(self, kb_id: str, request: MultiKBQueryRequest, snapshot=None):
        """Query specific KB in ClickHouse"""
        if kb_id in self._clickhouse_clients:
            client = self._clickhouse_clients[kb_id]
            sql_query = self._build_sql_query(request)
            return await client.execute_query(sql_query, request.params)
        else:
            raise ValueError(f"No ClickHouse client for KB {kb_id}")

    def _build_cypher_query(self, request: MultiKBQueryRequest) -> str:
        """Build Cypher query based on request pattern"""
        pattern = request.pattern

        if pattern == QueryPattern.KB7_TERMINOLOGY_LOOKUP:
            return "WHERE n.code = $code AND n.system = $system RETURN n"
        elif pattern == QueryPattern.KB5_INTERACTION_CHECK:
            return "WHERE n.drug1_rxnorm IN $drug_codes OR n.drug2_rxnorm IN $drug_codes RETURN n"
        elif pattern == QueryPattern.KB1_PATIENT_LOOKUP:
            return "WHERE n.id = $patient_id RETURN n"
        else:
            return "RETURN n LIMIT 100"  # Default query

    def _build_cross_kb_cypher_query(self, request: MultiKBQueryRequest) -> str:
        """Build cross-KB Cypher query"""
        pattern = request.pattern

        if pattern == QueryPattern.CROSS_KB_PATIENT_VIEW:
            return """
                MATCH (p:Patient:KB1_PatientStream {id: $patient_id})
                OPTIONAL MATCH (p)-[r1]->(m:Medication:KB1_PatientStream)
                OPTIONAL MATCH (m)-[r2]->(t:Term:KB7_TerminologyStream)
                RETURN p, collect(m) as medications, collect(t) as terminology
            """
        elif pattern == QueryPattern.CROSS_KB_DRUG_ANALYSIS:
            return """
                MATCH (d:Drug:KB7_TerminologyStream {rxnorm: $drug_rxnorm})
                OPTIONAL MATCH (d)-[i:INTERACTS_WITH]-(d2:Drug:KB5_InteractionStream)
                OPTIONAL MATCH (d)-[calc:HAS_CALCULATION]-(c:Calculation:KB3_DrugCalculationStream)
                RETURN d, collect(i) as interactions, collect(c) as calculations
            """
        else:
            return "MATCH (n) RETURN n LIMIT 100"

    def _build_sql_query(self, request: MultiKBQueryRequest) -> str:
        """Build SQL query for ClickHouse"""
        pattern = request.pattern

        if pattern == QueryPattern.KB7_TERMINOLOGY_LOOKUP:
            return "SELECT * FROM terminology_analytics WHERE code = %(code)s"
        elif pattern == QueryPattern.KB3_DRUG_CALCULATION:
            return "SELECT * FROM drug_calculations WHERE drug_rxnorm = %(drug_rxnorm)s"
        else:
            return "SELECT * FROM analytics_summary LIMIT 100"

    async def _create_snapshot(self, request: MultiKBQueryRequest):
        """Create snapshot for consistency"""
        if not self._snapshot_manager:
            from ..snapshot_manager.manager import SnapshotManager
            self._snapshot_manager = SnapshotManager()

        return await self._snapshot_manager.create_snapshot(
            service_id=request.service_id,
            context={
                'kb_id': request.kb_id,
                'pattern': request.pattern.value,
                'cross_kb_scope': request.cross_kb_scope
            }
        )

    async def _handle_fallback(self, request: MultiKBQueryRequest, error: Exception):
        """Handle query failures with fallback"""
        logger.warning(f"Query failed, attempting fallback: {error}")

        # Simple fallback to PostgreSQL for KB7
        if request.kb_id == 'kb7':
            try:
                result = await self._query_postgres(request)
                return MultiKBQueryResponse(
                    data=result,
                    sources_used=['postgres_fallback'],
                    kb_sources=[request.kb_id],
                    latency=0.0,
                    cache_status='fallback'
                )
            except Exception as fallback_error:
                logger.error(f"Fallback also failed: {fallback_error}")

        # Return error response
        return MultiKBQueryResponse(
            data={'error': str(error)},
            sources_used=[],
            kb_sources=[],
            latency=0.0,
            cache_status='error'
        )

    async def _query_postgres(self, request: MultiKBQueryRequest):
        """Query PostgreSQL - placeholder implementation"""
        # This would connect to PostgreSQL and execute the query
        return {'message': 'PostgreSQL query result placeholder'}

    async def _query_elasticsearch(self, request: MultiKBQueryRequest):
        """Query Elasticsearch - placeholder implementation"""
        # This would connect to Elasticsearch and execute the search
        return {'message': 'Elasticsearch query result placeholder'}

    async def _query_graphdb(self, kb_id: str, request: MultiKBQueryRequest):
        """Query GraphDB for semantic operations"""
        if not self._graphdb_client:
            raise RuntimeError("GraphDB client not initialized")

        try:
            # Get repository ID for this KB
            repository_id = await self._graphdb_client.get_kb_repository_id(kb_id)
            if not repository_id:
                raise ValueError(f"No GraphDB repository found for KB {kb_id}")

            # Build SPARQL query based on pattern
            sparql_query_text = self._build_sparql_query(request, kb_id)

            # Create SPARQL query object
            from ..graphdb_semantic.multi_kb_graphdb_manager import SPARQLQuery, SPARQLQueryType

            # Determine query type from pattern
            query_type = self._get_sparql_query_type(request.pattern)

            sparql_query = SPARQLQuery(
                query=sparql_query_text,
                query_type=query_type,
                repository_id=repository_id,
                kb_id=kb_id,
                params=request.params,
                timeout=30,
                infer=True  # Enable reasoning
            )

            # Execute query
            result = await self._graphdb_client.execute_sparql_query(sparql_query)

            if result.errors:
                logger.error(f"GraphDB query errors: {result.errors}")
                return {'error': 'GraphDB query failed', 'details': result.errors}

            return {
                'data': result.data,
                'bindings_count': result.bindings_count,
                'execution_time_ms': result.execution_time_ms,
                'repository': repository_id
            }

        except Exception as e:
            logger.error(f"Error querying GraphDB for KB {kb_id}: {e}")
            return {'error': f'GraphDB query error: {str(e)}'}

    def _build_sparql_query(self, request: MultiKBQueryRequest, kb_id: str) -> str:
        """Build SPARQL query based on request pattern and KB"""
        pattern = request.pattern
        params = request.params or {}

        if pattern == QueryPattern.SEMANTIC_INFERENCE:
            # Semantic inference queries
            if kb_id == 'kb-7':  # Medical terminology
                search_term = params.get('search_term', '')
                return f"""
                PREFIX snomed: <http://snomed.info/id/>
                PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
                PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

                SELECT DISTINCT ?concept ?preferredLabel ?conceptId ?definition
                WHERE {{
                    ?concept skos:prefLabel ?preferredLabel .
                    ?concept snomed:conceptId ?conceptId .
                    OPTIONAL {{ ?concept skos:definition ?definition }}

                    FILTER (
                        CONTAINS(LCASE(?preferredLabel), LCASE("{search_term}"))
                    )
                }}
                ORDER BY STRLEN(?preferredLabel)
                LIMIT 20
                """
            elif kb_id == 'kb-5':  # Drug interactions
                drug_code = params.get('drug_code', '')
                return f"""
                PREFIX interaction: <http://cardiofit.ai/ontology/interaction/>
                PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

                SELECT DISTINCT ?interactingDrug ?interactionType ?severity ?description
                WHERE {{
                    ?interaction interaction:drug1 <{drug_code}> ;
                                interaction:drug2 ?interactingDrug ;
                                interaction:type ?interactionType ;
                                interaction:severity ?severity ;
                                rdfs:comment ?description .
                }}
                ORDER BY ?severity ?interactingDrug
                LIMIT 50
                """

        elif pattern == QueryPattern.CROSS_KB_SEMANTIC_SEARCH:
            # Cross-KB semantic search
            search_term = params.get('search_term', '')
            return f"""
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
            PREFIX owl: <http://www.w3.org/2002/07/owl#>

            SELECT DISTINCT ?concept ?label ?type ?kb ?description
            WHERE {{
                ?concept ?labelProp ?label .
                ?concept rdf:type ?type .
                ?concept <http://cardiofit.ai/ontology/kb> ?kb .
                OPTIONAL {{ ?concept rdfs:comment ?description }}

                FILTER (
                    ?labelProp IN (rdfs:label, skos:prefLabel, skos:altLabel) &&
                    (CONTAINS(LCASE(?label), LCASE("{search_term}")) ||
                     CONTAINS(LCASE(?description), LCASE("{search_term}")))
                )
            }}
            ORDER BY ?kb ?label
            LIMIT 100
            """

        # Default query for unknown patterns
        return """
        SELECT ?s ?p ?o
        WHERE { ?s ?p ?o }
        LIMIT 10
        """

    def _get_sparql_query_type(self, pattern: QueryPattern):
        """Determine SPARQL query type from query pattern"""
        from ..graphdb_semantic.multi_kb_graphdb_manager import SPARQLQueryType

        # Most semantic queries are SELECT queries
        if pattern in [QueryPattern.SEMANTIC_INFERENCE, QueryPattern.CROSS_KB_SEMANTIC_SEARCH]:
            return SPARQLQueryType.SELECT

        # Default to SELECT for unknown patterns
        return SPARQLQueryType.SELECT

    async def _query_graphdb_cross_kb(self, kb_list: List[str], request: MultiKBQueryRequest):
        """Execute GraphDB query across multiple knowledge bases"""
        if not self._graphdb_client:
            raise RuntimeError("GraphDB client not initialized")

        try:
            # Use the cross-KB semantic search method
            search_term = request.params.get('search_term', '')
            limit_per_kb = request.params.get('limit_per_kb', 20)

            result = await self._graphdb_client.execute_cross_kb_semantic_search(
                search_term=search_term,
                kb_ids=kb_list,
                limit=limit_per_kb
            )

            return {
                'data': result,
                'kb_sources': kb_list,
                'source_type': 'graphdb_cross_kb'
            }

        except Exception as e:
            logger.error(f"Error in cross-KB GraphDB query: {e}")
            return {'error': f'Cross-KB GraphDB query error: {str(e)}'}

    async def close(self) -> None:
        """Close all connections and cleanup resources"""
        logger.info("Closing Multi-KB Query Router...")

        # Close Neo4j connections
        if self._neo4j_manager:
            await self._neo4j_manager.close()
            self._neo4j_manager = None

        # Close ClickHouse connections
        for kb_id, client in self._clickhouse_clients.items():
            try:
                client.close_all_connections()
            except Exception as e:
                logger.error(f"Error closing ClickHouse client for {kb_id}: {e}")
        self._clickhouse_clients.clear()

        # Close GraphDB connections
        if self._graphdb_client:
            await self._graphdb_client.close()
            self._graphdb_client = None

        # Close other clients (PostgreSQL, Elasticsearch, Redis)
        # These would be implemented as needed

        logger.info("Multi-KB Query Router connections closed")

    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get router performance metrics"""
        return {
            **self.metrics,
            'timestamp': datetime.utcnow().isoformat()
        }

    async def close(self) -> None:
        """Close all connections"""
        if self._neo4j_manager:
            await self._neo4j_manager.close()

        for client in self._clickhouse_clients.values():
            if hasattr(client, 'close'):
                await client.close()

        logger.info("Multi-KB Query Router closed")


# Backward compatibility for KB7-specific usage
class QueryRouter(MultiKBQueryRouter):
    """
    Backward compatibility wrapper for KB7-specific usage
    Maps old single-KB interface to new multi-KB system
    """

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.kb_id = 'kb7'  # Default to KB7 for backward compatibility

    async def route_query(self, request) -> MultiKBQueryResponse:
        """Legacy method - convert old request format to new multi-KB format"""
        if hasattr(request, 'pattern') and hasattr(request, 'params'):
            # Convert old QueryRequest to new MultiKBQueryRequest
            multi_kb_request = MultiKBQueryRequest(
                service_id=getattr(request, 'service_id', 'unknown'),
                kb_id=self.kb_id,
                pattern=request.pattern,
                params=request.params,
                require_snapshot=getattr(request, 'require_snapshot', False)
            )
            return await super().route_query(multi_kb_request)
        else:
            return await super().route_query(request)