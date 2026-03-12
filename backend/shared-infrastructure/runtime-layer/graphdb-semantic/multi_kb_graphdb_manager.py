"""
Multi-KB GraphDB Manager
Manages GraphDB connections and SPARQL queries across all Knowledge Bases
"""

import asyncio
import aiohttp
import json
from typing import Dict, Any, Optional, List, Union
from datetime import datetime, timedelta
from dataclasses import dataclass, asdict
from enum import Enum
import time
from urllib.parse import quote, urljoin
from loguru import logger

from ..config.multi_kb_config import MultiKBRuntimeConfig, KnowledgeBaseConfig


class SPARQLQueryType(Enum):
    """Types of SPARQL queries"""
    SELECT = "SELECT"
    CONSTRUCT = "CONSTRUCT"
    ASK = "ASK"
    DESCRIBE = "DESCRIBE"
    INSERT = "INSERT"
    DELETE = "DELETE"


@dataclass
class GraphDBRepository:
    """GraphDB repository configuration"""
    id: str
    kb_id: str
    title: str
    description: str
    type: str = "free"  # free, se, ee
    ruleset: str = "owl2-rl"
    baseurl: str = ""
    context: str = ""
    enabled: bool = True


@dataclass
class SPARQLQuery:
    """SPARQL query with metadata"""
    query: str
    query_type: SPARQLQueryType
    repository_id: str
    kb_id: str
    params: Dict[str, Any] = None
    timeout: int = 30
    infer: bool = True

    def __post_init__(self):
        if self.params is None:
            self.params = {}


@dataclass
class SPARQLResult:
    """SPARQL query result with metadata"""
    data: Union[List[Dict], Dict, bool, str]
    query_type: SPARQLQueryType
    execution_time_ms: float
    repository_id: str
    kb_id: str
    bindings_count: int = 0
    cache_hit: bool = False
    errors: List[str] = None

    def __post_init__(self):
        if self.errors is None:
            self.errors = []


class MultiKBGraphDBManager:
    """
    Manages GraphDB connections and operations across all Knowledge Bases

    Provides unified SPARQL query interface for semantic operations across
    all CardioFit knowledge bases with repository management and caching.
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Multi-KB GraphDB Manager

        Args:
            config: GraphDB configuration with connection details
        """
        self.config = config
        self.host = config.get('host', 'localhost')
        self.port = config.get('port', 7200)
        self.username = config.get('username', 'admin')
        self.password = config.get('password', 'admin')
        self.ssl = config.get('ssl', False)

        # Build base URLs
        protocol = 'https' if self.ssl else 'http'
        self.base_url = f"{protocol}://{self.host}:{self.port}"
        self.workbench_url = f"{self.base_url}/workbench"
        self.rest_url = f"{self.base_url}/rest"
        self.sparql_url = f"{self.base_url}/repositories"

        # Session and connection management
        self.session = None
        self.connected = False

        # KB-specific repositories
        self.repositories = {}
        self.kb_repository_map = {}

        # Performance metrics
        self.metrics = {
            'total_queries': 0,
            'queries_per_kb': {},
            'avg_query_time_ms': 0.0,
            'cache_hit_rate': 0.0,
            'connection_pool_size': 10,
            'active_connections': 0
        }

        # Health status
        self.health_status = {
            'connected': False,
            'repositories_healthy': {},
            'last_health_check': None,
            'uptime_start': datetime.utcnow()
        }

        logger.info(f"GraphDB Manager initialized for {self.base_url}")

    async def initialize_connection(self) -> bool:
        """
        Initialize connection to GraphDB server

        Returns:
            Success status
        """
        try:
            # Create aiohttp session with authentication
            auth = aiohttp.BasicAuth(self.username, self.password)
            connector = aiohttp.TCPConnector(
                limit=self.config.get('connection_pool_size', 10),
                ttl_dns_cache=300,
                use_dns_cache=True
            )

            timeout = aiohttp.ClientTimeout(total=30)
            self.session = aiohttp.ClientSession(
                auth=auth,
                connector=connector,
                timeout=timeout
            )

            # Test connection with health check
            health_check = await self._perform_health_check()

            if health_check:
                self.connected = True
                self.health_status['connected'] = True
                logger.info("GraphDB connection established successfully")

                # Initialize repositories for all KBs
                await self._initialize_kb_repositories()
                return True
            else:
                logger.error("GraphDB health check failed during initialization")
                return False

        except Exception as e:
            logger.error(f"Failed to initialize GraphDB connection: {e}")
            self.connected = False
            return False

    async def _initialize_kb_repositories(self) -> None:
        """Initialize repositories for all knowledge bases"""
        logger.info("Initializing KB-specific repositories...")

        # Define repository configurations for each KB
        kb_repos = {
            'kb-1': GraphDBRepository(
                id='kb1-patient-data',
                kb_id='kb-1',
                title='KB-1 Patient Data Repository',
                description='Patient clinical data and relationships',
                type='free',
                ruleset='owl2-rl'
            ),
            'kb-2': GraphDBRepository(
                id='kb2-clinical-guidelines',
                kb_id='kb-2',
                title='KB-2 Clinical Guidelines Repository',
                description='Evidence-based clinical practice guidelines',
                type='free',
                ruleset='owl2-rl'
            ),
            'kb-5': GraphDBRepository(
                id='kb5-drug-interactions',
                kb_id='kb-5',
                title='KB-5 Drug Interactions Repository',
                description='Pharmaceutical interaction knowledge graph',
                type='free',
                ruleset='owl2-rl'
            ),
            'kb-6': GraphDBRepository(
                id='kb6-evidence-base',
                kb_id='kb-6',
                title='KB-6 Evidence Base Repository',
                description='Clinical evidence and research outcomes',
                type='free',
                ruleset='owl2-rl'
            ),
            'kb-7': GraphDBRepository(
                id='kb7-terminology',
                kb_id='kb-7',
                title='KB-7 Medical Terminology Repository',
                description='SNOMED CT, ICD-10, and medical ontologies',
                type='free',
                ruleset='owl2-rl'
            )
        }

        # Create repositories and update mappings
        for kb_id, repo_config in kb_repos.items():
            try:
                # Check if repository exists, create if not
                exists = await self._check_repository_exists(repo_config.id)

                if not exists:
                    created = await self._create_repository(repo_config)
                    if created:
                        logger.info(f"Created repository {repo_config.id} for {kb_id}")
                    else:
                        logger.error(f"Failed to create repository {repo_config.id} for {kb_id}")
                        continue
                else:
                    logger.info(f"Repository {repo_config.id} already exists for {kb_id}")

                # Store repository configuration
                self.repositories[repo_config.id] = repo_config
                self.kb_repository_map[kb_id] = repo_config.id

                # Initialize health status
                self.health_status['repositories_healthy'][repo_config.id] = True

            except Exception as e:
                logger.error(f"Error initializing repository for {kb_id}: {e}")
                self.health_status['repositories_healthy'][kb_id] = False

    async def _check_repository_exists(self, repository_id: str) -> bool:
        """Check if a repository exists"""
        try:
            url = f"{self.rest_url}/repositories/{repository_id}"
            async with self.session.get(url) as response:
                return response.status == 200
        except Exception as e:
            logger.error(f"Error checking repository {repository_id}: {e}")
            return False

    async def _create_repository(self, repo_config: GraphDBRepository) -> bool:
        """Create a new GraphDB repository"""
        try:
            # Repository configuration template
            repo_template = {
                "id": repo_config.id,
                "title": repo_config.title,
                "type": f"graphdb:{repo_config.type}",
                "sesameType": "owlim:Sail",
                "owlim:base-URL": repo_config.baseurl,
                "owlim:defaultNS": f"{repo_config.baseurl}#",
                "owlim:entity-index-size": "10000000",
                "owlim:entity-id-size": "32",
                "owlim:imports": "",
                "owlim:repository-type": "file-repository",
                "owlim:ruleset": repo_config.ruleset,
                "owlim:storage-folder": "storage",
                "owlim:enable-context-index": "false",
                "owlim:enablePredicateList": "true",
                "owlim:in-memory-literal-properties": "true",
                "owlim:enable-literal-index": "true",
                "owlim:check-for-inconsistencies": "false",
                "owlim:disable-sameAs": "true",
                "owlim:query-timeout": "0",
                "owlim:query-limit-results": "0",
                "owlim:throw-QueryEvaluationException-on-timeout": "false",
                "owlim:read-only": "false"
            }

            url = f"{self.rest_url}/repositories"
            headers = {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }

            async with self.session.post(url, json=repo_template, headers=headers) as response:
                if response.status == 201:
                    logger.info(f"Successfully created repository {repo_config.id}")
                    return True
                else:
                    error_text = await response.text()
                    logger.error(f"Failed to create repository {repo_config.id}: {response.status} - {error_text}")
                    return False

        except Exception as e:
            logger.error(f"Error creating repository {repo_config.id}: {e}")
            return False

    async def execute_sparql_query(self, query: SPARQLQuery) -> SPARQLResult:
        """
        Execute SPARQL query on specified repository

        Args:
            query: SPARQL query with metadata

        Returns:
            Query result with metadata
        """
        if not self.connected:
            raise RuntimeError("GraphDB connection not initialized")

        start_time = time.time()

        try:
            # Build query URL
            repository_id = query.repository_id
            query_url = f"{self.sparql_url}/{repository_id}"

            # Set appropriate endpoint based on query type
            if query.query_type in [SPARQLQueryType.SELECT, SPARQLQueryType.CONSTRUCT,
                                  SPARQLQueryType.ASK, SPARQLQueryType.DESCRIBE]:
                query_url = f"{query_url}/sparql"
            else:  # INSERT/DELETE
                query_url = f"{query_url}/statements"

            # Prepare request parameters
            params = {
                'query': query.query,
                'infer': str(query.infer).lower(),
                'timeout': query.timeout
            }

            # Add query parameters if any
            if query.params:
                params.update(query.params)

            # Set appropriate headers
            headers = {'Accept': 'application/sparql-results+json'}
            if query.query_type == SPARQLQueryType.CONSTRUCT:
                headers['Accept'] = 'application/rdf+json'
            elif query.query_type == SPARQLQueryType.ASK:
                headers['Accept'] = 'text/boolean'

            # Execute query
            async with self.session.get(query_url, params=params, headers=headers) as response:
                execution_time = (time.time() - start_time) * 1000

                if response.status == 200:
                    result_data = await response.json()

                    # Process result based on query type
                    processed_data, bindings_count = self._process_sparql_result(
                        result_data, query.query_type
                    )

                    # Update metrics
                    self._update_query_metrics(query.kb_id, execution_time)

                    return SPARQLResult(
                        data=processed_data,
                        query_type=query.query_type,
                        execution_time_ms=execution_time,
                        repository_id=repository_id,
                        kb_id=query.kb_id,
                        bindings_count=bindings_count
                    )
                else:
                    error_text = await response.text()
                    logger.error(f"SPARQL query failed: {response.status} - {error_text}")

                    return SPARQLResult(
                        data={},
                        query_type=query.query_type,
                        execution_time_ms=execution_time,
                        repository_id=repository_id,
                        kb_id=query.kb_id,
                        errors=[f"HTTP {response.status}: {error_text}"]
                    )

        except Exception as e:
            execution_time = (time.time() - start_time) * 1000
            logger.error(f"Error executing SPARQL query: {e}")

            return SPARQLResult(
                data={},
                query_type=query.query_type,
                execution_time_ms=execution_time,
                repository_id=query.repository_id,
                kb_id=query.kb_id,
                errors=[str(e)]
            )

    def _process_sparql_result(self, result_data: Dict, query_type: SPARQLQueryType) -> tuple:
        """Process SPARQL result based on query type"""
        if query_type == SPARQLQueryType.SELECT:
            bindings = result_data.get('results', {}).get('bindings', [])
            return bindings, len(bindings)
        elif query_type == SPARQLQueryType.ASK:
            return result_data.get('boolean', False), 1
        elif query_type == SPARQLQueryType.CONSTRUCT:
            return result_data, len(result_data) if isinstance(result_data, list) else 1
        elif query_type == SPARQLQueryType.DESCRIBE:
            return result_data, len(result_data) if isinstance(result_data, list) else 1
        else:
            return result_data, 0

    def _update_query_metrics(self, kb_id: str, execution_time_ms: float) -> None:
        """Update query performance metrics"""
        self.metrics['total_queries'] += 1

        # Per-KB metrics
        if kb_id not in self.metrics['queries_per_kb']:
            self.metrics['queries_per_kb'][kb_id] = 0
        self.metrics['queries_per_kb'][kb_id] += 1

        # Update average query time
        total_queries = self.metrics['total_queries']
        current_avg = self.metrics['avg_query_time_ms']
        self.metrics['avg_query_time_ms'] = (
            (current_avg * (total_queries - 1) + execution_time_ms) / total_queries
        )

    async def get_kb_repository_id(self, kb_id: str) -> Optional[str]:
        """Get repository ID for a knowledge base"""
        return self.kb_repository_map.get(kb_id)

    async def execute_cross_kb_semantic_search(self, search_term: str,
                                             kb_ids: List[str],
                                             limit: int = 50) -> Dict[str, Any]:
        """
        Execute semantic search across multiple knowledge bases

        Args:
            search_term: Search term or concept
            kb_ids: List of KB IDs to search
            limit: Maximum results per KB

        Returns:
            Combined search results from all KBs
        """
        results = {}

        for kb_id in kb_ids:
            repository_id = await self.get_kb_repository_id(kb_id)
            if not repository_id:
                logger.warning(f"No repository found for KB {kb_id}")
                continue

            # Build semantic search query for this KB
            sparql_query = self._build_semantic_search_query(search_term, kb_id, limit)

            query = SPARQLQuery(
                query=sparql_query,
                query_type=SPARQLQueryType.SELECT,
                repository_id=repository_id,
                kb_id=kb_id,
                timeout=30
            )

            try:
                result = await self.execute_sparql_query(query)
                if not result.errors:
                    results[kb_id] = {
                        'data': result.data,
                        'bindings_count': result.bindings_count,
                        'execution_time_ms': result.execution_time_ms
                    }
                else:
                    results[kb_id] = {'errors': result.errors}

            except Exception as e:
                logger.error(f"Error searching KB {kb_id}: {e}")
                results[kb_id] = {'errors': [str(e)]}

        return results

    def _build_semantic_search_query(self, search_term: str, kb_id: str, limit: int) -> str:
        """Build SPARQL query for semantic search"""
        # Basic semantic search query - can be enhanced per KB type
        query = f"""
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>

        SELECT DISTINCT ?concept ?label ?type ?description
        WHERE {{
            ?concept ?labelProp ?label .
            ?concept rdf:type ?type .
            OPTIONAL {{ ?concept rdfs:comment ?description }}

            FILTER (
                ?labelProp IN (rdfs:label, skos:prefLabel, skos:altLabel) &&
                (CONTAINS(LCASE(?label), LCASE("{search_term}")) ||
                 CONTAINS(LCASE(?description), LCASE("{search_term}")))
            )
        }}
        ORDER BY ?label
        LIMIT {limit}
        """

        return query

    async def _perform_health_check(self) -> bool:
        """Perform comprehensive health check"""
        try:
            # Check server status
            url = f"{self.rest_url}/repositories"
            async with self.session.get(url) as response:
                server_healthy = response.status == 200

            if not server_healthy:
                return False

            # Check repository health
            for repo_id in self.repositories.keys():
                repo_healthy = await self._check_repository_health(repo_id)
                self.health_status['repositories_healthy'][repo_id] = repo_healthy

            self.health_status['last_health_check'] = datetime.utcnow()
            return True

        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return False

    async def _check_repository_health(self, repository_id: str) -> bool:
        """Check health of specific repository"""
        try:
            # Simple ASK query to test repository
            test_query = "ASK { ?s ?p ?o }"
            query_url = f"{self.sparql_url}/{repository_id}/sparql"
            params = {'query': test_query}

            async with self.session.get(query_url, params=params) as response:
                return response.status == 200

        except Exception as e:
            logger.error(f"Repository {repository_id} health check failed: {e}")
            return False

    async def get_health_status(self) -> Dict[str, Any]:
        """Get comprehensive health status"""
        # Perform fresh health check
        overall_healthy = await self._perform_health_check()

        return {
            'overall_healthy': overall_healthy,
            'connected': self.connected,
            'server_url': self.base_url,
            'repositories': {
                repo_id: {
                    'healthy': self.health_status['repositories_healthy'].get(repo_id, False),
                    'kb_id': config.kb_id,
                    'title': config.title
                }
                for repo_id, config in self.repositories.items()
            },
            'metrics': self.metrics,
            'last_health_check': self.health_status['last_health_check'].isoformat()
                if self.health_status['last_health_check'] else None,
            'uptime_seconds': (datetime.utcnow() - self.health_status['uptime_start']).total_seconds()
        }

    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get performance metrics"""
        return {
            'total_queries': self.metrics['total_queries'],
            'queries_per_kb': self.metrics['queries_per_kb'],
            'avg_query_time_ms': self.metrics['avg_query_time_ms'],
            'cache_hit_rate': self.metrics['cache_hit_rate'],
            'active_repositories': len(self.repositories),
            'connected_repositories': len([r for r in self.health_status['repositories_healthy'].values() if r])
        }

    async def close(self) -> None:
        """Close all connections and cleanup resources"""
        logger.info("Closing GraphDB connections...")

        if self.session:
            await self.session.close()
            self.session = None

        self.connected = False
        self.health_status['connected'] = False

        logger.info("GraphDB connections closed")


# Helper functions for backward compatibility
async def create_graphdb_manager(config: Dict[str, Any]) -> MultiKBGraphDBManager:
    """Create and initialize GraphDB manager"""
    manager = MultiKBGraphDBManager(config)
    success = await manager.initialize_connection()

    if not success:
        raise RuntimeError("Failed to initialize GraphDB manager")

    return manager


def build_terminology_search_query(search_term: str, limit: int = 20) -> str:
    """Build SPARQL query for medical terminology search (KB-7)"""
    return f"""
    PREFIX snomed: <http://snomed.info/id/>
    PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
    PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

    SELECT DISTINCT ?concept ?preferredLabel ?definition ?conceptId
    WHERE {{
        ?concept skos:prefLabel ?preferredLabel .
        ?concept snomed:conceptId ?conceptId .
        OPTIONAL {{ ?concept skos:definition ?definition }}

        FILTER (
            CONTAINS(LCASE(?preferredLabel), LCASE("{search_term}")) ||
            CONTAINS(LCASE(?definition), LCASE("{search_term}"))
        )

        FILTER (!CONTAINS(?preferredLabel, "(")) # Exclude parenthetical terms
    }}
    ORDER BY STRLEN(?preferredLabel)
    LIMIT {limit}
    """


def build_drug_interaction_query(drug_code: str) -> str:
    """Build SPARQL query for drug interactions (KB-5)"""
    return f"""
    PREFIX fhir: <http://hl7.org/fhir/>
    PREFIX interaction: <http://cardiofit.ai/ontology/interaction/>

    SELECT DISTINCT ?interactingDrug ?interactionType ?severity ?description
    WHERE {{
        ?interaction interaction:drug1 <{drug_code}> ;
                    interaction:drug2 ?interactingDrug ;
                    interaction:type ?interactionType ;
                    interaction:severity ?severity ;
                    rdfs:comment ?description .
    }}
    ORDER BY ?severity ?interactingDrug
    """