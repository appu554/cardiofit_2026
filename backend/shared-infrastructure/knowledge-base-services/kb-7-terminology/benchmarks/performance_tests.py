#!/usr/bin/env python3
"""
Performance Testing Suite for KB7 Terminology Phase 3.5
========================================================

Comprehensive performance testing with realistic workloads for:
- PostgreSQL terminology queries
- Neo4j semantic reasoning
- Query router performance
- FHIR endpoint validation
- Cache efficiency testing
- End-to-end workflow validation

Author: Claude Code Performance Engineer
Version: 1.0.0
"""

import asyncio
import time
import statistics
import random
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple, NamedTuple
from dataclasses import dataclass, asdict
from concurrent.futures import ThreadPoolExecutor, as_completed
import logging

import httpx
import psycopg2
import psycopg2.pool
import redis
from neo4j import GraphDatabase
import numpy as np
from faker import Faker
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, TaskID
from rich.live import Live
from rich.layout import Layout
from rich.panel import Panel
import yaml

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

fake = Faker()

class WorkloadProfile(NamedTuple):
    """Defines a specific workload profile for testing"""
    name: str
    query_type: str
    weight: float  # Percentage of total queries
    complexity: str  # 'simple', 'medium', 'complex'
    cache_friendly: bool

@dataclass
class PerformanceTestResult:
    """Result of a performance test run"""
    test_name: str
    workload_profile: str
    duration_seconds: float
    total_requests: int
    successful_requests: int
    failed_requests: int
    requests_per_second: float
    mean_latency_ms: float
    median_latency_ms: float
    p95_latency_ms: float
    p99_latency_ms: float
    max_latency_ms: float
    min_latency_ms: float
    error_rate_pct: float
    throughput_score: float
    latency_score: float
    overall_score: float
    errors: List[str]
    timestamp: str

@dataclass
class SystemMetrics:
    """System-level metrics during test execution"""
    cpu_usage_pct: float
    memory_usage_pct: float
    disk_io_read_mb: float
    disk_io_write_mb: float
    network_in_mb: float
    network_out_mb: float
    active_connections: int
    cache_hit_ratio: float
    timestamp: str

class RealisticDataGenerator:
    """Generates realistic test data for terminology queries"""

    def __init__(self):
        self.medical_terms = [
            "hypertension", "diabetes", "pneumonia", "asthma", "coronary",
            "myocardial", "pulmonary", "renal", "hepatic", "cardiac",
            "respiratory", "cardiovascular", "endocrine", "neurological",
            "gastrointestinal", "hematologic", "oncologic", "infectious"
        ]

        self.code_systems = [
            "http://snomed.info/sct",
            "http://loinc.org",
            "http://www.nlm.nih.gov/research/umls/rxnorm",
            "http://hl7.org/fhir/sid/icd-10-cm",
            "http://hl7.org/fhir/sid/cpt"
        ]

        self.sample_codes = {
            "http://snomed.info/sct": [
                "424144002", "38341003", "195967001", "13645005", "44054006",
                "233604007", "22298006", "840539006", "302509004", "386661006"
            ],
            "http://loinc.org": [
                "8480-6", "8462-4", "33747-0", "1975-2", "6690-2",
                "789-8", "718-7", "4548-4", "33743-4", "14647-2"
            ],
            "http://www.nlm.nih.gov/research/umls/rxnorm": [
                "1191", "161", "5224", "8124", "197361", "198039", "596976"
            ]
        }

        self.value_sets = [
            "cardiovascular-conditions",
            "diabetes-medications",
            "respiratory-conditions",
            "laboratory-tests",
            "vital-signs"
        ]

    def generate_concept_lookup_params(self) -> Dict[str, str]:
        """Generate parameters for concept lookup queries"""
        system = random.choice(self.code_systems)
        code = random.choice(self.sample_codes.get(system, ["unknown"]))
        return {"system": system, "code": code}

    def generate_search_params(self) -> Dict[str, str]:
        """Generate parameters for concept search queries"""
        term = random.choice(self.medical_terms)
        # Add some variation to search terms
        if random.random() < 0.3:
            term = f"{term}%"  # Prefix search
        elif random.random() < 0.5:
            term = f"%{term}%"  # Contains search
        return {"term": term}

    def generate_valueset_params(self) -> Dict[str, str]:
        """Generate parameters for value set operations"""
        return {"value_set": random.choice(self.value_sets)}

    def generate_mapping_params(self) -> Dict[str, str]:
        """Generate parameters for concept mapping queries"""
        source_system = random.choice(self.code_systems)
        target_system = random.choice([s for s in self.code_systems if s != source_system])
        source_code = random.choice(self.sample_codes.get(source_system, ["unknown"]))
        return {
            "source_system": source_system,
            "target_system": target_system,
            "source_code": source_code
        }

class PostgreSQLPerformanceTester:
    """PostgreSQL-specific performance testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.data_generator = RealisticDataGenerator()
        self.connection_pool = None

    def initialize_connection_pool(self, min_connections: int = 5, max_connections: int = 20):
        """Initialize PostgreSQL connection pool"""
        try:
            self.connection_pool = psycopg2.pool.ThreadedConnectionPool(
                min_connections,
                max_connections,
                host=self.config['postgresql']['host'],
                port=self.config['postgresql']['port'],
                database=self.config['postgresql']['database'],
                user=self.config['postgresql']['user'],
                password=self.config['postgresql']['password']
            )
        except Exception as e:
            logger.error(f"Failed to initialize PostgreSQL connection pool: {e}")
            raise

    def execute_concept_lookup(self) -> float:
        """Execute concept lookup query"""
        params = self.data_generator.generate_concept_lookup_params()
        conn = self.connection_pool.getconn()

        try:
            start_time = time.perf_counter()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT concept_id, display_name, code_system, definition
                    FROM terminology_concepts
                    WHERE code = %s AND code_system = %s
                """, (params['code'], params['system']))
                result = cur.fetchall()
            end_time = time.perf_counter()
            return (end_time - start_time) * 1000  # Convert to ms
        finally:
            self.connection_pool.putconn(conn)

    def execute_concept_search(self) -> float:
        """Execute concept search query"""
        params = self.data_generator.generate_search_params()
        conn = self.connection_pool.getconn()

        try:
            start_time = time.perf_counter()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT concept_id, display_name, code_system, definition
                    FROM terminology_concepts
                    WHERE display_name ILIKE %s
                    ORDER BY
                        CASE
                            WHEN display_name ILIKE %s THEN 1
                            WHEN display_name ILIKE %s THEN 2
                            ELSE 3
                        END,
                        display_name
                    LIMIT 20
                """, (f"%{params['term']}%", f"{params['term']}%", f"%{params['term']}%"))
                result = cur.fetchall()
            end_time = time.perf_counter()
            return (end_time - start_time) * 1000
        finally:
            self.connection_pool.putconn(conn)

    def execute_valueset_expansion(self) -> float:
        """Execute value set expansion query"""
        params = self.data_generator.generate_valueset_params()
        conn = self.connection_pool.getconn()

        try:
            start_time = time.perf_counter()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT tc.concept_id, tc.display_name, tc.code, tc.code_system
                    FROM terminology_concepts tc
                    JOIN value_set_concepts vsc ON tc.concept_id = vsc.concept_id
                    JOIN value_sets vs ON vsc.value_set_id = vs.value_set_id
                    WHERE vs.value_set_id = %s
                    ORDER BY tc.display_name
                """, (params['value_set'],))
                result = cur.fetchall()
            end_time = time.perf_counter()
            return (end_time - start_time) * 1000
        finally:
            self.connection_pool.putconn(conn)

    def execute_concept_hierarchy(self) -> float:
        """Execute concept hierarchy query"""
        params = self.data_generator.generate_concept_lookup_params()
        conn = self.connection_pool.getconn()

        try:
            start_time = time.perf_counter()
            with conn.cursor() as cur:
                # Simulate hierarchy traversal with self-joins
                cur.execute("""
                    WITH RECURSIVE concept_hierarchy AS (
                        SELECT concept_id, parent_concept_id, display_name, 1 as level
                        FROM terminology_concepts
                        WHERE code = %s AND code_system = %s

                        UNION ALL

                        SELECT tc.concept_id, tc.parent_concept_id, tc.display_name, ch.level + 1
                        FROM terminology_concepts tc
                        JOIN concept_hierarchy ch ON tc.concept_id = ch.parent_concept_id
                        WHERE ch.level < 5
                    )
                    SELECT * FROM concept_hierarchy
                """, (params['code'], params['system']))
                result = cur.fetchall()
            end_time = time.perf_counter()
            return (end_time - start_time) * 1000
        finally:
            self.connection_pool.putconn(conn)

    def close_connection_pool(self):
        """Close connection pool"""
        if self.connection_pool:
            self.connection_pool.closeall()

class Neo4jPerformanceTester:
    """Neo4j-specific performance testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.data_generator = RealisticDataGenerator()
        self.driver = None

    def initialize_driver(self):
        """Initialize Neo4j driver"""
        try:
            self.driver = GraphDatabase.driver(
                self.config['neo4j']['uri'],
                auth=(self.config['neo4j']['user'], self.config['neo4j']['password']),
                max_connection_lifetime=3600,
                max_connection_pool_size=50,
                connection_acquisition_timeout=120
            )
        except Exception as e:
            logger.error(f"Failed to initialize Neo4j driver: {e}")
            raise

    def execute_concept_hierarchy_traversal(self) -> float:
        """Execute concept hierarchy traversal in Neo4j"""
        params = self.data_generator.generate_concept_lookup_params()

        start_time = time.perf_counter()
        with self.driver.session() as session:
            result = session.run("""
                MATCH (c:Concept {code: $code, system: $system})
                OPTIONAL MATCH path = (c)-[:IS_A*1..3]->(parent:Concept)
                RETURN c.code, c.display, collect(parent.code) as ancestors
            """, code=params['code'], system=params['system'])
            records = list(result)
        end_time = time.perf_counter()

        return (end_time - start_time) * 1000

    def execute_semantic_relationships(self) -> float:
        """Execute semantic relationship queries"""
        params = self.data_generator.generate_concept_lookup_params()

        start_time = time.perf_counter()
        with self.driver.session() as session:
            result = session.run("""
                MATCH (source:Concept {code: $code, system: $system})
                MATCH (source)-[r:SEMANTIC_RELATION]->(target:Concept)
                WHERE r.relationship_type IN ['associated_with', 'contraindicated_with', 'treats']
                RETURN source.code, r.relationship_type, target.code, target.display
                LIMIT 20
            """, code=params['code'], system=params['system'])
            records = list(result)
        end_time = time.perf_counter()

        return (end_time - start_time) * 1000

    def execute_inference_chain(self) -> float:
        """Execute complex inference chain query"""
        params = self.data_generator.generate_mapping_params()

        start_time = time.perf_counter()
        with self.driver.session() as session:
            result = session.run("""
                MATCH (start:Concept {code: $start_code, system: $source_system})
                MATCH path = (start)-[:SEMANTIC_RELATION*1..4]->(end:Concept)
                WHERE end.system = $target_system
                RETURN path, length(path) as path_length
                ORDER BY path_length
                LIMIT 10
            """,
            start_code=params['source_code'],
            source_system=params['source_system'],
            target_system=params['target_system'])
            records = list(result)
        end_time = time.perf_counter()

        return (end_time - start_time) * 1000

    def execute_concept_clustering(self) -> float:
        """Execute concept clustering query"""
        start_time = time.perf_counter()
        with self.driver.session() as session:
            result = session.run("""
                MATCH (c:Concept)-[:SEMANTIC_RELATION]->(related:Concept)
                WHERE c.system = 'http://snomed.info/sct'
                WITH c, count(related) as relationship_count
                WHERE relationship_count > 5
                RETURN c.code, c.display, relationship_count
                ORDER BY relationship_count DESC
                LIMIT 20
            """)
            records = list(result)
        end_time = time.perf_counter()

        return (end_time - start_time) * 1000

    def close_driver(self):
        """Close Neo4j driver"""
        if self.driver:
            self.driver.close()

class FHIREndpointTester:
    """FHIR endpoint performance testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.data_generator = RealisticDataGenerator()
        self.base_url = config['fhir_service']['url']
        self.terminology_endpoint = config['fhir_service']['terminology_endpoint']

    async def execute_codesystem_lookup(self) -> float:
        """Test FHIR CodeSystem/$lookup operation"""
        params = self.data_generator.generate_concept_lookup_params()

        async with httpx.AsyncClient(timeout=30.0) as client:
            start_time = time.perf_counter()
            response = await client.get(
                f"{self.base_url}{self.terminology_endpoint}/CodeSystem/$lookup",
                params={
                    'system': params['system'],
                    'code': params['code']
                }
            )
            end_time = time.perf_counter()

            if response.status_code not in [200, 404]:  # 404 is acceptable for test data
                raise httpx.HTTPStatusError(f"Unexpected status: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

    async def execute_valueset_expand(self) -> float:
        """Test FHIR ValueSet/$expand operation"""
        params = self.data_generator.generate_valueset_params()

        async with httpx.AsyncClient(timeout=30.0) as client:
            start_time = time.perf_counter()
            response = await client.get(
                f"{self.base_url}{self.terminology_endpoint}/ValueSet/$expand",
                params={
                    'url': f'http://cardiofit.com/fhir/ValueSet/{params["value_set"]}'
                }
            )
            end_time = time.perf_counter()

            if response.status_code not in [200, 404]:
                raise httpx.HTTPStatusError(f"Unexpected status: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

    async def execute_conceptmap_translate(self) -> float:
        """Test FHIR ConceptMap/$translate operation"""
        params = self.data_generator.generate_mapping_params()

        async with httpx.AsyncClient(timeout=30.0) as client:
            start_time = time.perf_counter()
            response = await client.get(
                f"{self.base_url}{self.terminology_endpoint}/ConceptMap/$translate",
                params={
                    'system': params['source_system'],
                    'code': params['source_code'],
                    'target': params['target_system']
                }
            )
            end_time = time.perf_counter()

            if response.status_code not in [200, 404]:
                raise httpx.HTTPStatusError(f"Unexpected status: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

    async def execute_terminology_search(self) -> float:
        """Test FHIR terminology search"""
        params = self.data_generator.generate_search_params()

        async with httpx.AsyncClient(timeout=30.0) as client:
            start_time = time.perf_counter()
            response = await client.get(
                f"{self.base_url}{self.terminology_endpoint}/CodeSystem",
                params={
                    'name:contains': params['term'],
                    '_count': 20
                }
            )
            end_time = time.perf_counter()

            if response.status_code not in [200, 404]:
                raise httpx.HTTPStatusError(f"Unexpected status: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

class QueryRouterTester:
    """Query router performance testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.data_generator = RealisticDataGenerator()
        self.base_url = config['query_router']['url']

    async def execute_hybrid_query(self) -> float:
        """Test hybrid query routing"""
        params = self.data_generator.generate_concept_lookup_params()

        async with httpx.AsyncClient(timeout=30.0) as client:
            start_time = time.perf_counter()
            response = await client.post(
                f"{self.base_url}/query/hybrid",
                json={
                    'query_type': 'concept_lookup',
                    'parameters': params,
                    'include_semantic': True,
                    'use_cache': True
                }
            )
            end_time = time.perf_counter()

            if response.status_code not in [200, 404]:
                raise httpx.HTTPStatusError(f"Unexpected status: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

    async def execute_router_health_check(self) -> float:
        """Test router health check endpoint"""
        async with httpx.AsyncClient(timeout=10.0) as client:
            start_time = time.perf_counter()
            response = await client.get(f"{self.base_url}/health")
            end_time = time.perf_counter()

            if response.status_code != 200:
                raise httpx.HTTPStatusError(f"Health check failed: {response.status_code}", request=response.request, response=response)

            return (end_time - start_time) * 1000

class PerformanceTestOrchestrator:
    """Orchestrates comprehensive performance testing"""

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_config(config_path)
        self.console = Console()

        # Initialize testers
        self.pg_tester = PostgreSQLPerformanceTester(self.config)
        self.neo4j_tester = Neo4jPerformanceTester(self.config)
        self.fhir_tester = FHIREndpointTester(self.config)
        self.router_tester = QueryRouterTester(self.config)

        # Define workload profiles
        self.workload_profiles = {
            'realistic_mixed': [
                WorkloadProfile('concept_lookup', 'postgresql', 0.35, 'simple', True),
                WorkloadProfile('concept_search', 'postgresql', 0.25, 'medium', False),
                WorkloadProfile('valueset_expansion', 'postgresql', 0.15, 'medium', True),
                WorkloadProfile('semantic_relationships', 'neo4j', 0.15, 'complex', False),
                WorkloadProfile('fhir_operations', 'fhir', 0.10, 'medium', True),
            ],
            'heavy_reasoning': [
                WorkloadProfile('inference_chain', 'neo4j', 0.40, 'complex', False),
                WorkloadProfile('concept_clustering', 'neo4j', 0.30, 'complex', False),
                WorkloadProfile('hierarchy_traversal', 'neo4j', 0.20, 'medium', True),
                WorkloadProfile('concept_lookup', 'postgresql', 0.10, 'simple', True),
            ],
            'fhir_heavy': [
                WorkloadProfile('codesystem_lookup', 'fhir', 0.30, 'medium', True),
                WorkloadProfile('valueset_expand', 'fhir', 0.25, 'medium', True),
                WorkloadProfile('conceptmap_translate', 'fhir', 0.20, 'complex', False),
                WorkloadProfile('terminology_search', 'fhir', 0.15, 'medium', False),
                WorkloadProfile('hybrid_query', 'router', 0.10, 'complex', True),
            ],
            'cache_optimized': [
                WorkloadProfile('concept_lookup', 'postgresql', 0.60, 'simple', True),
                WorkloadProfile('valueset_expansion', 'postgresql', 0.20, 'medium', True),
                WorkloadProfile('hierarchy_traversal', 'neo4j', 0.15, 'medium', True),
                WorkloadProfile('codesystem_lookup', 'fhir', 0.05, 'medium', True),
            ]
        }

    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load test configuration"""
        default_config = {
            'postgresql': {
                'host': 'localhost',
                'port': 5433,
                'database': 'terminology_db',
                'user': 'kb7_user',
                'password': 'kb7_password'
            },
            'neo4j': {
                'uri': 'bolt://localhost:7687',
                'user': 'neo4j',
                'password': 'kb7_neo4j'
            },
            'fhir_service': {
                'url': 'http://localhost:8014',
                'terminology_endpoint': '/terminology'
            },
            'query_router': {
                'url': 'http://localhost:8090'
            },
            'test_parameters': {
                'duration_seconds': 300,
                'concurrent_users': 10,
                'warmup_requests': 100
            }
        }

        if config_path and os.path.exists(config_path):
            try:
                with open(config_path, 'r') as f:
                    loaded_config = yaml.safe_load(f)
                    default_config.update(loaded_config)
            except Exception as e:
                logger.warning(f"Could not load config from {config_path}: {e}")

        return default_config

    async def initialize_all_testers(self):
        """Initialize all performance testers"""
        self.pg_tester.initialize_connection_pool()
        self.neo4j_tester.initialize_driver()

        # Warm up connections
        await self._warmup_connections()

    async def _warmup_connections(self):
        """Warm up all database connections"""
        warmup_requests = self.config['test_parameters']['warmup_requests']

        with Progress() as progress:
            warmup_task = progress.add_task("[yellow]Warming up connections...", total=warmup_requests)

            for i in range(warmup_requests):
                try:
                    # Quick warmup queries
                    self.pg_tester.execute_concept_lookup()
                    await self.fhir_tester.execute_codesystem_lookup()
                    await asyncio.sleep(0.01)  # Small delay
                    progress.update(warmup_task, advance=1)
                except Exception as e:
                    logger.warning(f"Warmup request {i} failed: {e}")

    async def run_workload_test(self, workload_name: str, duration_seconds: int = 300,
                               concurrent_users: int = 10) -> List[PerformanceTestResult]:
        """Run a specific workload test"""
        if workload_name not in self.workload_profiles:
            raise ValueError(f"Unknown workload: {workload_name}")

        workload = self.workload_profiles[workload_name]
        results = []

        self.console.print(f"[bold green]Running workload: {workload_name}[/bold green]")
        self.console.print(f"Duration: {duration_seconds}s, Concurrent users: {concurrent_users}")

        # Run each workload profile
        for profile in workload:
            self.console.print(f"Testing {profile.name} ({profile.query_type})...")

            result = await self._execute_profile_test(
                profile, duration_seconds, concurrent_users
            )
            results.append(result)

        return results

    async def _execute_profile_test(self, profile: WorkloadProfile,
                                   duration_seconds: int,
                                   concurrent_users: int) -> PerformanceTestResult:
        """Execute test for a specific workload profile"""
        start_time = time.time()
        end_time = start_time + duration_seconds

        latencies = []
        errors = []
        total_requests = 0
        successful_requests = 0

        # Create semaphore to limit concurrent requests
        semaphore = asyncio.Semaphore(concurrent_users)

        async def execute_single_request():
            async with semaphore:
                nonlocal total_requests, successful_requests
                total_requests += 1

                try:
                    if profile.query_type == 'postgresql':
                        latency = await self._execute_postgresql_query(profile.name)
                    elif profile.query_type == 'neo4j':
                        latency = await self._execute_neo4j_query(profile.name)
                    elif profile.query_type == 'fhir':
                        latency = await self._execute_fhir_query(profile.name)
                    elif profile.query_type == 'router':
                        latency = await self._execute_router_query(profile.name)
                    else:
                        raise ValueError(f"Unknown query type: {profile.query_type}")

                    latencies.append(latency)
                    successful_requests += 1

                except Exception as e:
                    errors.append(str(e))

        # Execute requests for the specified duration
        tasks = []
        while time.time() < end_time:
            task = asyncio.create_task(execute_single_request())
            tasks.append(task)

            # Small delay to prevent overwhelming the system
            await asyncio.sleep(0.001)

        # Wait for all remaining tasks to complete
        await asyncio.gather(*tasks, return_exceptions=True)

        # Calculate metrics
        actual_duration = time.time() - start_time
        failed_requests = total_requests - successful_requests

        if latencies:
            mean_latency = statistics.mean(latencies)
            median_latency = statistics.median(latencies)
            p95_latency = np.percentile(latencies, 95)
            p99_latency = np.percentile(latencies, 99)
            max_latency = max(latencies)
            min_latency = min(latencies)
        else:
            mean_latency = median_latency = p95_latency = p99_latency = max_latency = min_latency = 0

        requests_per_second = total_requests / actual_duration if actual_duration > 0 else 0
        error_rate_pct = (failed_requests / total_requests * 100) if total_requests > 0 else 0

        # Calculate performance scores
        throughput_score = min(100, (requests_per_second / 100) * 100)  # Normalize to 100 RPS = 100 points
        latency_score = max(0, 100 - (p95_latency / 10))  # 10ms = 90 points, 100ms = 0 points
        overall_score = (throughput_score * 0.6 + latency_score * 0.4) * (1 - error_rate_pct / 100)

        return PerformanceTestResult(
            test_name=profile.name,
            workload_profile=profile.query_type,
            duration_seconds=actual_duration,
            total_requests=total_requests,
            successful_requests=successful_requests,
            failed_requests=failed_requests,
            requests_per_second=requests_per_second,
            mean_latency_ms=mean_latency,
            median_latency_ms=median_latency,
            p95_latency_ms=p95_latency,
            p99_latency_ms=p99_latency,
            max_latency_ms=max_latency,
            min_latency_ms=min_latency,
            error_rate_pct=error_rate_pct,
            throughput_score=throughput_score,
            latency_score=latency_score,
            overall_score=overall_score,
            errors=errors[:10],  # Limit error list
            timestamp=datetime.now().isoformat()
        )

    async def _execute_postgresql_query(self, query_name: str) -> float:
        """Execute PostgreSQL query based on name"""
        if query_name == 'concept_lookup':
            return self.pg_tester.execute_concept_lookup()
        elif query_name == 'concept_search':
            return self.pg_tester.execute_concept_search()
        elif query_name == 'valueset_expansion':
            return self.pg_tester.execute_valueset_expansion()
        elif query_name == 'concept_hierarchy':
            return self.pg_tester.execute_concept_hierarchy()
        else:
            raise ValueError(f"Unknown PostgreSQL query: {query_name}")

    async def _execute_neo4j_query(self, query_name: str) -> float:
        """Execute Neo4j query based on name"""
        if query_name == 'hierarchy_traversal':
            return self.neo4j_tester.execute_concept_hierarchy_traversal()
        elif query_name == 'semantic_relationships':
            return self.neo4j_tester.execute_semantic_relationships()
        elif query_name == 'inference_chain':
            return self.neo4j_tester.execute_inference_chain()
        elif query_name == 'concept_clustering':
            return self.neo4j_tester.execute_concept_clustering()
        else:
            raise ValueError(f"Unknown Neo4j query: {query_name}")

    async def _execute_fhir_query(self, query_name: str) -> float:
        """Execute FHIR query based on name"""
        if query_name == 'codesystem_lookup' or query_name == 'fhir_operations':
            return await self.fhir_tester.execute_codesystem_lookup()
        elif query_name == 'valueset_expand':
            return await self.fhir_tester.execute_valueset_expand()
        elif query_name == 'conceptmap_translate':
            return await self.fhir_tester.execute_conceptmap_translate()
        elif query_name == 'terminology_search':
            return await self.fhir_tester.execute_terminology_search()
        else:
            raise ValueError(f"Unknown FHIR query: {query_name}")

    async def _execute_router_query(self, query_name: str) -> float:
        """Execute router query based on name"""
        if query_name == 'hybrid_query':
            return await self.router_tester.execute_hybrid_query()
        elif query_name == 'health_check':
            return await self.router_tester.execute_router_health_check()
        else:
            raise ValueError(f"Unknown router query: {query_name}")

    def print_results(self, results: List[PerformanceTestResult]):
        """Print formatted test results"""
        table = Table(title="Performance Test Results")
        table.add_column("Test", style="cyan")
        table.add_column("RPS", style="magenta")
        table.add_column("Mean (ms)", style="yellow")
        table.add_column("P95 (ms)", style="red")
        table.add_column("P99 (ms)", style="red")
        table.add_column("Error %", style="red")
        table.add_column("Score", style="green")

        for result in results:
            table.add_row(
                result.test_name,
                f"{result.requests_per_second:.1f}",
                f"{result.mean_latency_ms:.1f}",
                f"{result.p95_latency_ms:.1f}",
                f"{result.p99_latency_ms:.1f}",
                f"{result.error_rate_pct:.1f}%",
                f"{result.overall_score:.1f}"
            )

        self.console.print(table)

    def cleanup(self):
        """Clean up all resources"""
        self.pg_tester.close_connection_pool()
        self.neo4j_tester.close_driver()

# CLI interface
async def main():
    """Main entry point for performance testing"""
    import argparse

    parser = argparse.ArgumentParser(description="KB7 Terminology Performance Testing")
    parser.add_argument("--workload", default="realistic_mixed",
                       choices=['realistic_mixed', 'heavy_reasoning', 'fhir_heavy', 'cache_optimized'],
                       help="Workload profile to test")
    parser.add_argument("--duration", type=int, default=300, help="Test duration in seconds")
    parser.add_argument("--users", type=int, default=10, help="Number of concurrent users")
    parser.add_argument("--config", help="Path to configuration file")
    parser.add_argument("--output", help="Output file for results (JSON)")

    args = parser.parse_args()

    orchestrator = None
    try:
        orchestrator = PerformanceTestOrchestrator(args.config)

        # Initialize all testers
        await orchestrator.initialize_all_testers()

        # Run the specified workload test
        results = await orchestrator.run_workload_test(
            args.workload, args.duration, args.users
        )

        # Print results
        orchestrator.print_results(results)

        # Save results if output specified
        if args.output:
            import json
            results_dict = [asdict(result) for result in results]
            with open(args.output, 'w') as f:
                json.dump(results_dict, f, indent=2)
            print(f"Results saved to {args.output}")

    except Exception as e:
        logger.error(f"Performance test failed: {e}")
        return 1
    finally:
        if orchestrator:
            orchestrator.cleanup()

    return 0

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)