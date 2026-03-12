"""
GraphDB Client for SPARQL Operations

This client provides a comprehensive interface for interacting with GraphDB,
executing SPARQL queries, and extracting OWL reasoning results for the
KB7 Neo4j Dual-Stream & Service Runtime Layer.

Features:
- SPARQL query execution with connection pooling
- OWL reasoning result extraction
- Drug interaction and contraindication queries
- Semantic relationship traversal
- Error handling and retry logic
- Performance monitoring and caching
"""

import asyncio
import aiohttp
import logging
from typing import Dict, Any, List, Optional, Union
from datetime import datetime, timedelta
from urllib.parse import urljoin, quote
import json
import time
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)


class SPARQLQueryType(Enum):
    """Types of SPARQL queries supported"""
    SELECT = "SELECT"
    CONSTRUCT = "CONSTRUCT"
    ASK = "ASK"
    DESCRIBE = "DESCRIBE"


@dataclass
class SPARQLResult:
    """Container for SPARQL query results"""
    query_type: SPARQLQueryType
    bindings: List[Dict[str, Any]]
    execution_time: float
    total_results: int
    query: str
    timestamp: datetime


@dataclass
class DrugInteraction:
    """Drug interaction data from GraphDB"""
    source_drug_uri: str
    target_drug_uri: str
    source_drug_label: str
    target_drug_label: str
    interaction_type: str
    severity: str
    mechanism: str
    evidence_level: str
    description: Optional[str] = None


@dataclass
class DrugContraindication:
    """Drug contraindication data from GraphDB"""
    drug_uri: str
    drug_label: str
    condition_uri: str
    condition_label: str
    contraindication_type: str
    severity: str
    evidence_level: str
    description: Optional[str] = None


class GraphDBClient:
    """
    Asynchronous GraphDB client for SPARQL operations

    Provides high-level methods for common clinical knowledge queries
    and low-level SPARQL execution capabilities.
    """

    def __init__(self,
                 graphdb_url: str,
                 repository: str = "kb7-terminology",
                 username: Optional[str] = None,
                 password: Optional[str] = None,
                 timeout: int = 30,
                 max_connections: int = 10,
                 cache_ttl: int = 300):
        """
        Initialize GraphDB client

        Args:
            graphdb_url: Base URL for GraphDB instance
            repository: Repository name
            username: Optional authentication username
            password: Optional authentication password
            timeout: Request timeout in seconds
            max_connections: Maximum concurrent connections
            cache_ttl: Cache time-to-live in seconds
        """
        self.base_url = graphdb_url.rstrip('/')
        self.repository = repository
        self.username = username
        self.password = password
        self.timeout = timeout
        self.cache_ttl = cache_ttl

        # Build repository URLs
        self.repo_url = f"{self.base_url}/repositories/{repository}"
        self.sparql_url = f"{self.repo_url}/sparql"

        # Session and connection management
        self.session: Optional[aiohttp.ClientSession] = None
        self.connector = aiohttp.TCPConnector(limit=max_connections)

        # Query cache
        self.query_cache: Dict[str, SPARQLResult] = {}

        # Performance metrics
        self.metrics = {
            'total_queries': 0,
            'cache_hits': 0,
            'average_response_time': 0.0,
            'errors': 0
        }

    async def __aenter__(self):
        """Async context manager entry"""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.close()

    async def connect(self):
        """Establish connection to GraphDB"""
        if not self.session:
            # Setup authentication if provided
            auth = None
            if self.username and self.password:
                auth = aiohttp.BasicAuth(self.username, self.password)

            # Create session with timeout and auth
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            self.session = aiohttp.ClientSession(
                connector=self.connector,
                timeout=timeout,
                auth=auth
            )

            # Test connection
            await self._test_connection()
            logger.info(f"Connected to GraphDB at {self.base_url}")

    async def close(self):
        """Close connection to GraphDB"""
        if self.session:
            await self.session.close()
            self.session = None
        await self.connector.close()
        logger.info("GraphDB connection closed")

    async def _test_connection(self):
        """Test GraphDB connection and repository access"""
        try:
            async with self.session.get(f"{self.base_url}/rest/info") as response:
                if response.status != 200:
                    raise ConnectionError(f"GraphDB not accessible: {response.status}")

            # Test repository access
            async with self.session.get(f"{self.repo_url}") as response:
                if response.status != 200:
                    raise ConnectionError(f"Repository '{self.repository}' not accessible: {response.status}")

        except Exception as e:
            raise ConnectionError(f"Failed to connect to GraphDB: {e}")

    def _get_cache_key(self, query: str, params: Optional[Dict] = None) -> str:
        """Generate cache key for query"""
        param_str = json.dumps(params or {}, sort_keys=True)
        return f"{hash(query)}_{hash(param_str)}"

    def _is_cache_valid(self, result: SPARQLResult) -> bool:
        """Check if cached result is still valid"""
        age = (datetime.utcnow() - result.timestamp).total_seconds()
        return age < self.cache_ttl

    async def execute_sparql(self,
                           query: str,
                           query_type: SPARQLQueryType = SPARQLQueryType.SELECT,
                           params: Optional[Dict[str, Any]] = None,
                           use_cache: bool = True) -> SPARQLResult:
        """
        Execute SPARQL query against GraphDB

        Args:
            query: SPARQL query string
            query_type: Type of SPARQL query
            params: Query parameters for substitution
            use_cache: Whether to use query caching

        Returns:
            SPARQLResult with query results and metadata
        """
        if not self.session:
            await self.connect()

        # Check cache first
        cache_key = self._get_cache_key(query, params)
        if use_cache and cache_key in self.query_cache:
            cached_result = self.query_cache[cache_key]
            if self._is_cache_valid(cached_result):
                self.metrics['cache_hits'] += 1
                logger.debug(f"Cache hit for query: {query[:100]}...")
                return cached_result

        # Substitute parameters in query
        formatted_query = query
        if params:
            for key, value in params.items():
                # Handle different parameter types
                if isinstance(value, str):
                    formatted_query = formatted_query.replace(f"${key}", f'"{value}"')
                elif isinstance(value, (int, float)):
                    formatted_query = formatted_query.replace(f"${key}", str(value))
                elif isinstance(value, list):
                    # Handle IN clauses
                    values = ', '.join([f'"{v}"' if isinstance(v, str) else str(v) for v in value])
                    formatted_query = formatted_query.replace(f"${key}", values)

        # Prepare request
        headers = {
            'Accept': 'application/sparql-results+json' if query_type == SPARQLQueryType.SELECT else 'application/ld+json',
            'Content-Type': 'application/sparql-query'
        }

        start_time = time.time()

        try:
            async with self.session.post(
                self.sparql_url,
                data=formatted_query,
                headers=headers
            ) as response:

                execution_time = time.time() - start_time

                if response.status != 200:
                    error_text = await response.text()
                    raise Exception(f"SPARQL query failed: {response.status} - {error_text}")

                response_data = await response.json()

                # Parse results based on query type
                if query_type == SPARQLQueryType.SELECT:
                    bindings = response_data.get('results', {}).get('bindings', [])
                    # Process bindings to extract values
                    processed_bindings = []
                    for binding in bindings:
                        processed_binding = {}
                        for var, data in binding.items():
                            processed_binding[var] = data.get('value', data.get('uri', str(data)))
                        processed_bindings.append(processed_binding)

                    result = SPARQLResult(
                        query_type=query_type,
                        bindings=processed_bindings,
                        execution_time=execution_time,
                        total_results=len(processed_bindings),
                        query=formatted_query,
                        timestamp=datetime.utcnow()
                    )
                else:
                    # Handle CONSTRUCT/DESCRIBE results
                    result = SPARQLResult(
                        query_type=query_type,
                        bindings=[response_data] if response_data else [],
                        execution_time=execution_time,
                        total_results=1 if response_data else 0,
                        query=formatted_query,
                        timestamp=datetime.utcnow()
                    )

                # Cache result
                if use_cache:
                    self.query_cache[cache_key] = result

                # Update metrics
                self.metrics['total_queries'] += 1
                self.metrics['average_response_time'] = (
                    (self.metrics['average_response_time'] * (self.metrics['total_queries'] - 1) + execution_time) /
                    self.metrics['total_queries']
                )

                logger.debug(f"SPARQL query executed in {execution_time:.3f}s, {result.total_results} results")
                return result

        except Exception as e:
            self.metrics['errors'] += 1
            logger.error(f"SPARQL query failed: {e}")
            raise

    async def get_drug_interactions(self,
                                  drug_uris: List[str],
                                  severity_filter: Optional[str] = None) -> List[DrugInteraction]:
        """
        Get drug interactions for given drug URIs

        Args:
            drug_uris: List of drug URIs to check for interactions
            severity_filter: Optional severity filter (e.g., 'major', 'moderate', 'minor')

        Returns:
            List of DrugInteraction objects
        """
        # Build SPARQL query for drug interactions
        query = """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX kb: <http://kb7.terminology.org/>

        SELECT ?sourceDrug ?targetDrug ?sourceLabel ?targetLabel
               ?interactionType ?severity ?mechanism ?evidenceLevel ?description
        WHERE {
            ?interaction a kb:DrugInteraction ;
                        kb:sourceDrug ?sourceDrug ;
                        kb:targetDrug ?targetDrug ;
                        kb:interactionType ?interactionType ;
                        kb:severity ?severity ;
                        kb:mechanism ?mechanism ;
                        kb:evidenceLevel ?evidenceLevel .

            ?sourceDrug rdfs:label ?sourceLabel .
            ?targetDrug rdfs:label ?targetLabel .

            OPTIONAL { ?interaction kb:description ?description }

            FILTER(?sourceDrug IN ($drug_list) || ?targetDrug IN ($drug_list))
        """

        if severity_filter:
            query += f'\n            FILTER(?severity = "{severity_filter}")'

        query += "\n        }"

        # Format drug list for query
        drug_list = ', '.join([f'<{uri}>' for uri in drug_uris])

        result = await self.execute_sparql(
            query,
            SPARQLQueryType.SELECT,
            {'drug_list': drug_list}
        )

        # Convert results to DrugInteraction objects
        interactions = []
        for binding in result.bindings:
            interaction = DrugInteraction(
                source_drug_uri=binding.get('sourceDrug', ''),
                target_drug_uri=binding.get('targetDrug', ''),
                source_drug_label=binding.get('sourceLabel', ''),
                target_drug_label=binding.get('targetLabel', ''),
                interaction_type=binding.get('interactionType', ''),
                severity=binding.get('severity', ''),
                mechanism=binding.get('mechanism', ''),
                evidence_level=binding.get('evidenceLevel', ''),
                description=binding.get('description')
            )
            interactions.append(interaction)

        logger.info(f"Found {len(interactions)} drug interactions for {len(drug_uris)} drugs")
        return interactions

    async def get_drug_contraindications(self,
                                       drug_uris: List[str],
                                       condition_uris: Optional[List[str]] = None) -> List[DrugContraindication]:
        """
        Get drug contraindications for given drugs and conditions

        Args:
            drug_uris: List of drug URIs
            condition_uris: Optional list of condition URIs to filter by

        Returns:
            List of DrugContraindication objects
        """
        query = """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX kb: <http://kb7.terminology.org/>

        SELECT ?drug ?condition ?drugLabel ?conditionLabel
               ?contraindicationType ?severity ?evidenceLevel ?description
        WHERE {
            ?contraindication a kb:Contraindication ;
                            kb:drug ?drug ;
                            kb:condition ?condition ;
                            kb:contraindicationType ?contraindicationType ;
                            kb:severity ?severity ;
                            kb:evidenceLevel ?evidenceLevel .

            ?drug rdfs:label ?drugLabel .
            ?condition rdfs:label ?conditionLabel .

            OPTIONAL { ?contraindication kb:description ?description }

            FILTER(?drug IN ($drug_list))
        """

        if condition_uris:
            condition_list = ', '.join([f'<{uri}>' for uri in condition_uris])
            query += f'\n            FILTER(?condition IN ({condition_list}))'

        query += "\n        }"

        drug_list = ', '.join([f'<{uri}>' for uri in drug_uris])

        result = await self.execute_sparql(
            query,
            SPARQLQueryType.SELECT,
            {'drug_list': drug_list}
        )

        # Convert results to DrugContraindication objects
        contraindications = []
        for binding in result.bindings:
            contraindication = DrugContraindication(
                drug_uri=binding.get('drug', ''),
                drug_label=binding.get('drugLabel', ''),
                condition_uri=binding.get('condition', ''),
                condition_label=binding.get('conditionLabel', ''),
                contraindication_type=binding.get('contraindicationType', ''),
                severity=binding.get('severity', ''),
                evidence_level=binding.get('evidenceLevel', ''),
                description=binding.get('description')
            )
            contraindications.append(contraindication)

        logger.info(f"Found {len(contraindications)} contraindications for {len(drug_uris)} drugs")
        return contraindications

    async def get_semantic_relationships(self,
                                       concept_uri: str,
                                       relationship_types: Optional[List[str]] = None,
                                       max_depth: int = 2) -> Dict[str, List[Dict[str, Any]]]:
        """
        Get semantic relationships for a concept

        Args:
            concept_uri: URI of the concept
            relationship_types: Optional list of relationship types to include
            max_depth: Maximum traversal depth

        Returns:
            Dictionary of relationship types to related concepts
        """
        # Build recursive query for semantic relationships
        query = f"""
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?relationType ?relatedConcept ?relatedLabel ?depth
        WHERE {{
            <{concept_uri}> ?relationType ?relatedConcept .
            ?relatedConcept rdfs:label ?relatedLabel .

            BIND(1 as ?depth)
        """

        if max_depth > 1:
            # Add recursive part for deeper relationships
            for depth in range(2, max_depth + 1):
                query += f"""
            UNION {{
                <{concept_uri}> ?relationType{depth-1} ?intermediate{depth-1} .
                ?intermediate{depth-1} ?relationType ?relatedConcept .
                ?relatedConcept rdfs:label ?relatedLabel .

                BIND({depth} as ?depth)
            }}
            """

        if relationship_types:
            rel_filter = ' || '.join([f'?relationType = <{rt}>' for rt in relationship_types])
            query += f'\n            FILTER({rel_filter})'

        query += "\n        }"

        result = await self.execute_sparql(query, SPARQLQueryType.SELECT)

        # Group results by relationship type
        relationships = {}
        for binding in result.bindings:
            rel_type = binding.get('relationType', '')
            if rel_type not in relationships:
                relationships[rel_type] = []

            relationships[rel_type].append({
                'uri': binding.get('relatedConcept', ''),
                'label': binding.get('relatedLabel', ''),
                'depth': int(binding.get('depth', 1))
            })

        logger.info(f"Found {len(relationships)} relationship types for concept {concept_uri}")
        return relationships

    async def get_subsumption_hierarchy(self,
                                      concept_uri: str,
                                      direction: str = "both") -> Dict[str, List[Dict[str, Any]]]:
        """
        Get subsumption hierarchy (parents/children) for a concept

        Args:
            concept_uri: URI of the concept
            direction: "up" for parents, "down" for children, "both" for both

        Returns:
            Dictionary with 'parents' and/or 'children' keys
        """
        hierarchy = {}

        if direction in ["up", "both"]:
            # Get parent concepts
            parent_query = f"""
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX owl: <http://www.w3.org/2002/07/owl#>

            SELECT ?parent ?parentLabel
            WHERE {{
                <{concept_uri}> rdfs:subClassOf* ?parent .
                ?parent rdfs:label ?parentLabel .
                FILTER(?parent != <{concept_uri}>)
            }}
            """

            result = await self.execute_sparql(parent_query, SPARQLQueryType.SELECT)
            hierarchy['parents'] = [
                {'uri': binding.get('parent', ''), 'label': binding.get('parentLabel', '')}
                for binding in result.bindings
            ]

        if direction in ["down", "both"]:
            # Get child concepts
            child_query = f"""
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX owl: <http://www.w3.org/2002/07/owl#>

            SELECT ?child ?childLabel
            WHERE {{
                ?child rdfs:subClassOf* <{concept_uri}> .
                ?child rdfs:label ?childLabel .
                FILTER(?child != <{concept_uri}>)
            }}
            """

            result = await self.execute_sparql(child_query, SPARQLQueryType.SELECT)
            hierarchy['children'] = [
                {'uri': binding.get('child', ''), 'label': binding.get('childLabel', '')}
                for binding in result.bindings
            ]

        return hierarchy

    async def health_check(self) -> Dict[str, Any]:
        """
        Perform health check on GraphDB connection

        Returns:
            Health status information
        """
        try:
            if not self.session:
                await self.connect()

            # Test basic connectivity
            start_time = time.time()
            test_query = "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
            result = await self.execute_sparql(test_query, SPARQLQueryType.SELECT, use_cache=False)
            response_time = time.time() - start_time

            # Get repository info
            async with self.session.get(f"{self.repo_url}") as response:
                repo_info = await response.json() if response.status == 200 else {}

            return {
                'status': 'healthy',
                'response_time': response_time,
                'repository': self.repository,
                'total_triples': int(result.bindings[0].get('count', 0)) if result.bindings else 0,
                'metrics': self.metrics,
                'repository_info': repo_info
            }

        except Exception as e:
            return {
                'status': 'unhealthy',
                'error': str(e),
                'repository': self.repository,
                'metrics': self.metrics
            }

    def get_metrics(self) -> Dict[str, Any]:
        """Get client performance metrics"""
        return {
            **self.metrics,
            'cache_size': len(self.query_cache),
            'cache_hit_rate': (
                self.metrics['cache_hits'] / max(self.metrics['total_queries'], 1) * 100
            )
        }

    async def clear_cache(self):
        """Clear query cache"""
        self.query_cache.clear()
        logger.info("Query cache cleared")


# CLI functionality for testing
async def main():
    """CLI interface for testing GraphDB client"""
    import argparse

    parser = argparse.ArgumentParser(description="GraphDB Client Test Interface")
    parser.add_argument('--url', default='http://localhost:7200', help='GraphDB URL')
    parser.add_argument('--repository', default='kb7-terminology', help='Repository name')
    parser.add_argument('--test', action='store_true', help='Run basic tests')
    parser.add_argument('--query', help='Execute custom SPARQL query')

    args = parser.parse_args()

    async with GraphDBClient(args.url, args.repository) as client:
        if args.test:
            print("Running GraphDB client tests...")

            # Test health check
            health = await client.health_check()
            print(f"Health status: {health['status']}")

            # Test basic query
            test_query = "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"
            result = await client.execute_sparql(test_query)
            print(f"Test query returned {result.total_results} results")

            # Show metrics
            metrics = client.get_metrics()
            print(f"Metrics: {metrics}")

        elif args.query:
            print(f"Executing query: {args.query}")
            result = await client.execute_sparql(args.query)
            print(f"Results: {result.total_results}")
            for binding in result.bindings[:5]:  # Show first 5 results
                print(f"  {binding}")


if __name__ == "__main__":
    asyncio.run(main())