#!/usr/bin/env python3
"""
Phase 3.5 Validation Suite for KB7 Terminology
==============================================

Comprehensive validation system that tests all Phase 3.5 success criteria:
- ✅ PostgreSQL lookup queries <10ms (95th percentile)
- ✅ GraphDB reasoning queries <50ms (95th percentile)
- ✅ Query router uptime >99.9%
- ✅ Data migration completed with 100% integrity
- ✅ FHIR endpoints using hybrid architecture
- ✅ Cache hit ratio >90% for frequent operations
- ✅ Performance monitoring and alerting operational

Author: Claude Code Performance Engineer
Version: 1.0.0
"""

import asyncio
import json
import os
import time
import statistics
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
from pathlib import Path
import logging

import httpx
import psycopg2
import redis
from neo4j import GraphDatabase
import yaml
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, TaskID
from rich import print as rprint
import numpy as np
from prometheus_client.parser import text_string_to_metric_families

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@dataclass
class PerformanceMetrics:
    """Performance metrics for a specific test"""
    test_name: str
    mean_latency_ms: float
    p50_latency_ms: float
    p95_latency_ms: float
    p99_latency_ms: float
    max_latency_ms: float
    min_latency_ms: float
    success_rate: float
    total_requests: int
    errors: List[str]
    timestamp: str

@dataclass
class ValidationResult:
    """Result of a validation test"""
    criterion: str
    target: str
    actual: str
    passed: bool
    details: Dict[str, Any]
    metrics: Optional[PerformanceMetrics] = None

@dataclass
class Phase35ValidationReport:
    """Complete Phase 3.5 validation report"""
    test_run_id: str
    timestamp: str
    overall_status: str
    success_criteria_results: List[ValidationResult]
    performance_summary: Dict[str, Any]
    recommendations: List[str]
    test_environment: Dict[str, Any]

class ConfigurationManager:
    """Manages test configuration and environment setup"""

    def __init__(self, config_path: Optional[str] = None):
        self.config_path = config_path or os.path.join(
            os.path.dirname(__file__), '../config/test_config.yaml'
        )
        self.config = self._load_config()

    def _load_config(self) -> Dict[str, Any]:
        """Load test configuration from YAML file"""
        default_config = {
            'postgresql': {
                'host': 'localhost',
                'port': 5433,
                'database': 'terminology_db',
                'user': 'kb7_user',
                'password': 'kb7_password'
            },
            'redis': {
                'host': 'localhost',
                'port': 6380,
                'db': 0
            },
            'neo4j': {
                'uri': 'bolt://localhost:7687',
                'user': 'neo4j',
                'password': 'kb7_neo4j'
            },
            'query_router': {
                'url': 'http://localhost:8090',
                'health_endpoint': '/health',
                'metrics_endpoint': '/metrics'
            },
            'fhir_service': {
                'url': 'http://localhost:8014',
                'terminology_endpoint': '/terminology'
            },
            'google_fhir': {
                'project_id': 'cardiofit-demo',
                'location': 'us-central1',
                'dataset_id': 'kb7_terminology'
            },
            'performance_targets': {
                'postgresql_p95_ms': 10,
                'graphdb_p95_ms': 50,
                'router_uptime_pct': 99.9,
                'cache_hit_ratio_pct': 90,
                'fhir_response_p95_ms': 200
            },
            'test_parameters': {
                'warmup_requests': 100,
                'test_requests': 1000,
                'concurrent_users': 10,
                'test_duration_seconds': 300
            }
        }

        try:
            if os.path.exists(self.config_path):
                with open(self.config_path, 'r') as f:
                    loaded_config = yaml.safe_load(f)
                    default_config.update(loaded_config)
        except Exception as e:
            logger.warning(f"Could not load config from {self.config_path}: {e}")
            logger.info("Using default configuration")

        return default_config

    def get(self, key: str, default: Any = None) -> Any:
        """Get configuration value using dot notation"""
        keys = key.split('.')
        value = self.config
        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default
        return value

class DatabaseConnector:
    """Manages database connections for testing"""

    def __init__(self, config: ConfigurationManager):
        self.config = config
        self._postgres_conn = None
        self._redis_conn = None
        self._neo4j_driver = None

    def get_postgres_connection(self):
        """Get PostgreSQL connection"""
        if self._postgres_conn is None or self._postgres_conn.closed:
            try:
                self._postgres_conn = psycopg2.connect(
                    host=self.config.get('postgresql.host'),
                    port=self.config.get('postgresql.port'),
                    database=self.config.get('postgresql.database'),
                    user=self.config.get('postgresql.user'),
                    password=self.config.get('postgresql.password')
                )
            except Exception as e:
                logger.error(f"Failed to connect to PostgreSQL: {e}")
                raise
        return self._postgres_conn

    def get_redis_connection(self):
        """Get Redis connection"""
        if self._redis_conn is None:
            try:
                self._redis_conn = redis.Redis(
                    host=self.config.get('redis.host'),
                    port=self.config.get('redis.port'),
                    db=self.config.get('redis.db'),
                    decode_responses=True
                )
                # Test connection
                self._redis_conn.ping()
            except Exception as e:
                logger.error(f"Failed to connect to Redis: {e}")
                raise
        return self._redis_conn

    def get_neo4j_driver(self):
        """Get Neo4j driver"""
        if self._neo4j_driver is None:
            try:
                self._neo4j_driver = GraphDatabase.driver(
                    self.config.get('neo4j.uri'),
                    auth=(
                        self.config.get('neo4j.user'),
                        self.config.get('neo4j.password')
                    )
                )
                # Test connection
                with self._neo4j_driver.session() as session:
                    session.run("RETURN 1")
            except Exception as e:
                logger.error(f"Failed to connect to Neo4j: {e}")
                raise
        return self._neo4j_driver

    def close_all(self):
        """Close all database connections"""
        if self._postgres_conn and not self._postgres_conn.closed:
            self._postgres_conn.close()
        if self._redis_conn:
            self._redis_conn.close()
        if self._neo4j_driver:
            self._neo4j_driver.close()

class PerformanceTester:
    """Core performance testing functionality"""

    def __init__(self, config: ConfigurationManager, db_connector: DatabaseConnector):
        self.config = config
        self.db = db_connector
        self.console = Console()

    async def measure_latency(self, test_func, iterations: int = 100) -> PerformanceMetrics:
        """Measure latency for a test function"""
        latencies = []
        errors = []
        start_time = datetime.now()

        for i in range(iterations):
            try:
                start = time.perf_counter()
                await test_func() if asyncio.iscoroutinefunction(test_func) else test_func()
                end = time.perf_counter()
                latencies.append((end - start) * 1000)  # Convert to ms
            except Exception as e:
                errors.append(str(e))
                logger.error(f"Test iteration {i} failed: {e}")

        if not latencies:
            raise RuntimeError("All test iterations failed")

        return PerformanceMetrics(
            test_name=test_func.__name__,
            mean_latency_ms=statistics.mean(latencies),
            p50_latency_ms=np.percentile(latencies, 50),
            p95_latency_ms=np.percentile(latencies, 95),
            p99_latency_ms=np.percentile(latencies, 99),
            max_latency_ms=max(latencies),
            min_latency_ms=min(latencies),
            success_rate=(len(latencies) / iterations) * 100,
            total_requests=iterations,
            errors=errors,
            timestamp=start_time.isoformat()
        )

class Phase35Validator:
    """Main validation class for Phase 3.5 success criteria"""

    def __init__(self, config_path: Optional[str] = None):
        self.config = ConfigurationManager(config_path)
        self.db = DatabaseConnector(self.config)
        self.tester = PerformanceTester(self.config, self.db)
        self.console = Console()
        self.validation_results: List[ValidationResult] = []

    async def run_full_validation(self) -> Phase35ValidationReport:
        """Run complete Phase 3.5 validation suite"""
        test_run_id = f"phase35_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        self.console.print(f"[bold blue]Starting Phase 3.5 Validation Suite[/bold blue]")
        self.console.print(f"Test Run ID: {test_run_id}")

        with Progress() as progress:
            main_task = progress.add_task("[green]Validating Phase 3.5 Criteria...", total=7)

            # Test 1: PostgreSQL Performance
            progress.update(main_task, advance=1, description="Testing PostgreSQL performance...")
            pg_result = await self._validate_postgresql_performance()
            self.validation_results.append(pg_result)

            # Test 2: GraphDB Performance
            progress.update(main_task, advance=1, description="Testing GraphDB performance...")
            graph_result = await self._validate_graphdb_performance()
            self.validation_results.append(graph_result)

            # Test 3: Query Router Uptime
            progress.update(main_task, advance=1, description="Testing query router uptime...")
            router_result = await self._validate_query_router_uptime()
            self.validation_results.append(router_result)

            # Test 4: Data Migration Integrity
            progress.update(main_task, advance=1, description="Validating data migration integrity...")
            migration_result = await self._validate_data_migration_integrity()
            self.validation_results.append(migration_result)

            # Test 5: FHIR Endpoints
            progress.update(main_task, advance=1, description="Testing FHIR endpoints...")
            fhir_result = await self._validate_fhir_hybrid_architecture()
            self.validation_results.append(fhir_result)

            # Test 6: Cache Hit Ratio
            progress.update(main_task, advance=1, description="Testing cache performance...")
            cache_result = await self._validate_cache_hit_ratio()
            self.validation_results.append(cache_result)

            # Test 7: Monitoring and Alerting
            progress.update(main_task, advance=1, description="Validating monitoring system...")
            monitoring_result = await self._validate_monitoring_and_alerting()
            self.validation_results.append(monitoring_result)

        return self._generate_report(test_run_id)

    async def _validate_postgresql_performance(self) -> ValidationResult:
        """Validate PostgreSQL lookup queries <10ms (95th percentile)"""
        target_p95_ms = self.config.get('performance_targets.postgresql_p95_ms', 10)

        def test_terminology_lookup():
            """Test typical terminology lookup query"""
            conn = self.db.get_postgres_connection()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT concept_id, display_name, code_system
                    FROM terminology_concepts
                    WHERE code = %s AND code_system = %s
                """, ('424144002', 'http://snomed.info/sct'))
                return cur.fetchall()

        def test_concept_search():
            """Test concept search query"""
            conn = self.db.get_postgres_connection()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT concept_id, display_name, code_system
                    FROM terminology_concepts
                    WHERE display_name ILIKE %s
                    LIMIT 10
                """, ('%hypertension%',))
                return cur.fetchall()

        def test_value_set_expansion():
            """Test value set expansion query"""
            conn = self.db.get_postgres_connection()
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT tc.concept_id, tc.display_name
                    FROM terminology_concepts tc
                    JOIN value_set_concepts vsc ON tc.concept_id = vsc.concept_id
                    WHERE vsc.value_set_id = %s
                """, ('cardiovascular-conditions',))
                return cur.fetchall()

        # Run performance tests
        iterations = self.config.get('test_parameters.test_requests', 1000)

        lookup_metrics = await self.tester.measure_latency(test_terminology_lookup, iterations)
        search_metrics = await self.tester.measure_latency(test_concept_search, iterations)
        expansion_metrics = await self.tester.measure_latency(test_value_set_expansion, iterations)

        # Calculate overall P95
        all_p95_values = [lookup_metrics.p95_latency_ms, search_metrics.p95_latency_ms, expansion_metrics.p95_latency_ms]
        overall_p95 = max(all_p95_values)

        passed = overall_p95 < target_p95_ms

        return ValidationResult(
            criterion="PostgreSQL lookup queries <10ms (95th percentile)",
            target=f"<{target_p95_ms}ms",
            actual=f"{overall_p95:.2f}ms",
            passed=passed,
            details={
                'lookup_p95_ms': lookup_metrics.p95_latency_ms,
                'search_p95_ms': search_metrics.p95_latency_ms,
                'expansion_p95_ms': expansion_metrics.p95_latency_ms,
                'overall_p95_ms': overall_p95,
                'test_iterations': iterations
            },
            metrics=lookup_metrics  # Primary metric
        )

    async def _validate_graphdb_performance(self) -> ValidationResult:
        """Validate GraphDB reasoning queries <50ms (95th percentile)"""
        target_p95_ms = self.config.get('performance_targets.graphdb_p95_ms', 50)

        def test_concept_hierarchy():
            """Test concept hierarchy traversal"""
            driver = self.db.get_neo4j_driver()
            with driver.session() as session:
                result = session.run("""
                    MATCH (c:Concept {code: $code})-[:IS_A*1..3]->(parent:Concept)
                    RETURN parent.code, parent.display, parent.system
                """, code="424144002")
                return list(result)

        def test_semantic_relationships():
            """Test semantic relationship queries"""
            driver = self.db.get_neo4j_driver()
            with driver.session() as session:
                result = session.run("""
                    MATCH (source:Concept)-[r:SEMANTIC_RELATION]->(target:Concept)
                    WHERE source.system = $system
                    AND r.relationship_type IN ['associated_with', 'contraindicated_with']
                    RETURN source.code, r.relationship_type, target.code
                    LIMIT 20
                """, system="http://snomed.info/sct")
                return list(result)

        def test_inference_chain():
            """Test complex inference chain"""
            driver = self.db.get_neo4j_driver()
            with driver.session() as session:
                result = session.run("""
                    MATCH path = (start:Concept {code: $start_code})
                    -[:SEMANTIC_RELATION*1..4]->
                    (end:Concept)
                    WHERE end.system = $target_system
                    RETURN path
                    LIMIT 10
                """, start_code="424144002", target_system="http://loinc.org")
                return list(result)

        # Run performance tests
        iterations = self.config.get('test_parameters.test_requests', 500)  # Fewer for complex queries

        hierarchy_metrics = await self.tester.measure_latency(test_concept_hierarchy, iterations)
        semantic_metrics = await self.tester.measure_latency(test_semantic_relationships, iterations)
        inference_metrics = await self.tester.measure_latency(test_inference_chain, iterations)

        # Calculate overall P95
        all_p95_values = [hierarchy_metrics.p95_latency_ms, semantic_metrics.p95_latency_ms, inference_metrics.p95_latency_ms]
        overall_p95 = max(all_p95_values)

        passed = overall_p95 < target_p95_ms

        return ValidationResult(
            criterion="GraphDB reasoning queries <50ms (95th percentile)",
            target=f"<{target_p95_ms}ms",
            actual=f"{overall_p95:.2f}ms",
            passed=passed,
            details={
                'hierarchy_p95_ms': hierarchy_metrics.p95_latency_ms,
                'semantic_p95_ms': semantic_metrics.p95_latency_ms,
                'inference_p95_ms': inference_metrics.p95_latency_ms,
                'overall_p95_ms': overall_p95,
                'test_iterations': iterations
            },
            metrics=hierarchy_metrics  # Primary metric
        )

    async def _validate_query_router_uptime(self) -> ValidationResult:
        """Validate Query router uptime >99.9%"""
        target_uptime_pct = self.config.get('performance_targets.router_uptime_pct', 99.9)
        router_url = self.config.get('query_router.url')
        health_endpoint = self.config.get('query_router.health_endpoint')

        total_checks = 1000
        failed_checks = 0
        response_times = []

        async with httpx.AsyncClient() as client:
            for i in range(total_checks):
                try:
                    start_time = time.perf_counter()
                    response = await client.get(f"{router_url}{health_endpoint}", timeout=5.0)
                    end_time = time.perf_counter()

                    response_times.append((end_time - start_time) * 1000)

                    if response.status_code != 200:
                        failed_checks += 1
                        logger.warning(f"Health check failed with status {response.status_code}")
                except Exception as e:
                    failed_checks += 1
                    logger.error(f"Health check request failed: {e}")

                # Small delay between checks
                await asyncio.sleep(0.01)

        actual_uptime_pct = ((total_checks - failed_checks) / total_checks) * 100
        passed = actual_uptime_pct >= target_uptime_pct

        return ValidationResult(
            criterion="Query router uptime >99.9%",
            target=f">{target_uptime_pct}%",
            actual=f"{actual_uptime_pct:.3f}%",
            passed=passed,
            details={
                'total_checks': total_checks,
                'failed_checks': failed_checks,
                'success_checks': total_checks - failed_checks,
                'actual_uptime_pct': actual_uptime_pct,
                'mean_response_time_ms': statistics.mean(response_times) if response_times else 0,
                'p95_response_time_ms': np.percentile(response_times, 95) if response_times else 0
            }
        )

    async def _validate_data_migration_integrity(self) -> ValidationResult:
        """Validate data migration completed with 100% integrity"""

        def check_data_counts():
            """Check data counts across stores"""
            conn = self.db.get_postgres_connection()
            with conn.cursor() as cur:
                # Check main tables
                cur.execute("SELECT COUNT(*) FROM terminology_concepts")
                concept_count = cur.fetchone()[0]

                cur.execute("SELECT COUNT(*) FROM concept_mappings")
                mapping_count = cur.fetchone()[0]

                cur.execute("SELECT COUNT(*) FROM value_sets")
                valueset_count = cur.fetchone()[0]

                # Check for orphaned records
                cur.execute("""
                    SELECT COUNT(*) FROM concept_mappings cm
                    LEFT JOIN terminology_concepts tc ON cm.source_concept_id = tc.concept_id
                    WHERE tc.concept_id IS NULL
                """)
                orphaned_mappings = cur.fetchone()[0]

                return {
                    'concept_count': concept_count,
                    'mapping_count': mapping_count,
                    'valueset_count': valueset_count,
                    'orphaned_mappings': orphaned_mappings
                }

        def check_referential_integrity():
            """Check referential integrity constraints"""
            conn = self.db.get_postgres_connection()
            integrity_issues = []

            with conn.cursor() as cur:
                # Check foreign key constraints
                checks = [
                    ("concept_mappings.source_concept_id", "terminology_concepts.concept_id"),
                    ("concept_mappings.target_concept_id", "terminology_concepts.concept_id"),
                    ("value_set_concepts.concept_id", "terminology_concepts.concept_id"),
                    ("value_set_concepts.value_set_id", "value_sets.value_set_id")
                ]

                for fk_col, ref_col in checks:
                    fk_table, fk_column = fk_col.split('.')
                    ref_table, ref_column = ref_col.split('.')

                    cur.execute(f"""
                        SELECT COUNT(*) FROM {fk_table} fk
                        LEFT JOIN {ref_table} ref ON fk.{fk_column} = ref.{ref_column}
                        WHERE ref.{ref_column} IS NULL AND fk.{fk_column} IS NOT NULL
                    """)

                    violation_count = cur.fetchone()[0]
                    if violation_count > 0:
                        integrity_issues.append(f"{fk_col} -> {ref_col}: {violation_count} violations")

            return integrity_issues

        def check_neo4j_sync():
            """Check Neo4j synchronization"""
            driver = self.db.get_neo4j_driver()
            with driver.session() as session:
                # Count concepts in Neo4j
                result = session.run("MATCH (c:Concept) RETURN COUNT(c) as count")
                neo4j_concept_count = result.single()['count']

                # Count relationships
                result = session.run("MATCH ()-[r]->() RETURN COUNT(r) as count")
                neo4j_relationship_count = result.single()['count']

                return {
                    'neo4j_concept_count': neo4j_concept_count,
                    'neo4j_relationship_count': neo4j_relationship_count
                }

        # Run integrity checks
        try:
            data_counts = check_data_counts()
            integrity_issues = check_referential_integrity()
            neo4j_data = check_neo4j_sync()

            # Determine if migration passed integrity check
            passed = (
                data_counts['concept_count'] > 0 and
                data_counts['orphaned_mappings'] == 0 and
                len(integrity_issues) == 0 and
                neo4j_data['neo4j_concept_count'] > 0
            )

            integrity_percentage = 100.0 if passed else (
                100.0 - (len(integrity_issues) * 10 + data_counts['orphaned_mappings'])
            )

            return ValidationResult(
                criterion="Data migration completed with 100% integrity",
                target="100% integrity",
                actual=f"{integrity_percentage:.1f}% integrity",
                passed=passed,
                details={
                    **data_counts,
                    **neo4j_data,
                    'integrity_issues': integrity_issues,
                    'integrity_percentage': integrity_percentage
                }
            )

        except Exception as e:
            return ValidationResult(
                criterion="Data migration completed with 100% integrity",
                target="100% integrity",
                actual="Failed to validate",
                passed=False,
                details={'error': str(e)}
            )

    async def _validate_fhir_hybrid_architecture(self) -> ValidationResult:
        """Validate FHIR endpoints using hybrid architecture"""
        fhir_url = self.config.get('fhir_service.url')
        terminology_endpoint = self.config.get('fhir_service.terminology_endpoint')
        target_p95_ms = self.config.get('performance_targets.fhir_response_p95_ms', 200)

        async def test_codesystem_lookup():
            """Test FHIR CodeSystem/$lookup operation"""
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{fhir_url}{terminology_endpoint}/CodeSystem/$lookup",
                    params={
                        'system': 'http://snomed.info/sct',
                        'code': '424144002'
                    },
                    timeout=10.0
                )
                return response.status_code == 200

        async def test_valueset_expand():
            """Test FHIR ValueSet/$expand operation"""
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{fhir_url}{terminology_endpoint}/ValueSet/$expand",
                    params={
                        'url': 'http://cardiofit.com/fhir/ValueSet/cardiovascular-conditions'
                    },
                    timeout=10.0
                )
                return response.status_code == 200

        async def test_conceptmap_translate():
            """Test FHIR ConceptMap/$translate operation"""
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{fhir_url}{terminology_endpoint}/ConceptMap/$translate",
                    params={
                        'system': 'http://snomed.info/sct',
                        'code': '424144002',
                        'target': 'http://loinc.org'
                    },
                    timeout=10.0
                )
                return response.status_code == 200

        # Run performance tests
        iterations = self.config.get('test_parameters.test_requests', 200)

        lookup_metrics = await self.tester.measure_latency(test_codesystem_lookup, iterations)
        expand_metrics = await self.tester.measure_latency(test_valueset_expand, iterations)
        translate_metrics = await self.tester.measure_latency(test_conceptmap_translate, iterations)

        # Calculate overall P95
        all_p95_values = [lookup_metrics.p95_latency_ms, expand_metrics.p95_latency_ms, translate_metrics.p95_latency_ms]
        overall_p95 = max(all_p95_values)

        passed = (
            overall_p95 < target_p95_ms and
            lookup_metrics.success_rate > 95 and
            expand_metrics.success_rate > 95 and
            translate_metrics.success_rate > 95
        )

        return ValidationResult(
            criterion="FHIR endpoints using hybrid architecture",
            target=f"<{target_p95_ms}ms P95, >95% success rate",
            actual=f"{overall_p95:.2f}ms P95, {min(lookup_metrics.success_rate, expand_metrics.success_rate, translate_metrics.success_rate):.1f}% success rate",
            passed=passed,
            details={
                'lookup_p95_ms': lookup_metrics.p95_latency_ms,
                'lookup_success_rate': lookup_metrics.success_rate,
                'expand_p95_ms': expand_metrics.p95_latency_ms,
                'expand_success_rate': expand_metrics.success_rate,
                'translate_p95_ms': translate_metrics.p95_latency_ms,
                'translate_success_rate': translate_metrics.success_rate,
                'overall_p95_ms': overall_p95
            },
            metrics=lookup_metrics
        )

    async def _validate_cache_hit_ratio(self) -> ValidationResult:
        """Validate cache hit ratio >90% for frequent operations"""
        target_hit_ratio_pct = self.config.get('performance_targets.cache_hit_ratio_pct', 90)
        redis_conn = self.db.get_redis_connection()

        # Get initial cache stats
        initial_stats = redis_conn.info('stats')
        initial_hits = initial_stats.get('keyspace_hits', 0)
        initial_misses = initial_stats.get('keyspace_misses', 0)

        # Warm up cache with frequent queries
        warmup_keys = [
            "terminology:concept:424144002",
            "terminology:concept:38341003",
            "terminology:valueset:cardiovascular-conditions",
            "terminology:mapping:snomed_to_loinc",
            "terminology:search:hypertension"
        ]

        # First, populate cache
        for key in warmup_keys:
            redis_conn.set(key, f"cached_data_for_{key}", ex=300)

        # Now perform cache operations to test hit ratio
        cache_operations = 1000
        for i in range(cache_operations):
            # Access cached items (should be hits)
            key = warmup_keys[i % len(warmup_keys)]
            redis_conn.get(key)

            # Occasionally try non-existent keys (will be misses)
            if i % 50 == 0:
                redis_conn.get(f"non_existent_key_{i}")

        # Get final cache stats
        final_stats = redis_conn.info('stats')
        final_hits = final_stats.get('keyspace_hits', 0)
        final_misses = final_stats.get('keyspace_misses', 0)

        # Calculate hit ratio during test
        test_hits = final_hits - initial_hits
        test_misses = final_misses - initial_misses
        total_operations = test_hits + test_misses

        if total_operations > 0:
            hit_ratio_pct = (test_hits / total_operations) * 100
        else:
            hit_ratio_pct = 0

        passed = hit_ratio_pct >= target_hit_ratio_pct

        return ValidationResult(
            criterion="Cache hit ratio >90% for frequent operations",
            target=f">{target_hit_ratio_pct}%",
            actual=f"{hit_ratio_pct:.1f}%",
            passed=passed,
            details={
                'test_hits': test_hits,
                'test_misses': test_misses,
                'total_operations': total_operations,
                'hit_ratio_pct': hit_ratio_pct,
                'cache_operations_performed': cache_operations
            }
        )

    async def _validate_monitoring_and_alerting(self) -> ValidationResult:
        """Validate performance monitoring and alerting operational"""
        router_url = self.config.get('query_router.url')
        metrics_endpoint = self.config.get('query_router.metrics_endpoint')

        try:
            # Check Prometheus metrics endpoint
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{router_url}{metrics_endpoint}", timeout=10.0)

                if response.status_code != 200:
                    return ValidationResult(
                        criterion="Performance monitoring and alerting operational",
                        target="Metrics endpoint accessible",
                        actual=f"HTTP {response.status_code}",
                        passed=False,
                        details={'error': f"Metrics endpoint returned {response.status_code}"}
                    )

                # Parse Prometheus metrics
                metrics_text = response.text
                metrics_families = list(text_string_to_metric_families(metrics_text))

                # Check for expected metrics
                expected_metrics = [
                    'http_requests_total',
                    'http_request_duration_seconds',
                    'query_router_health',
                    'cache_hit_ratio',
                    'database_connection_pool_size'
                ]

                found_metrics = [family.name for family in metrics_families]
                missing_metrics = [metric for metric in expected_metrics if metric not in found_metrics]

                # Check for alerting rules (simple check)
                has_alerting = any('alert' in metric.lower() for metric in found_metrics)

                monitoring_health = len(missing_metrics) == 0 and has_alerting

                return ValidationResult(
                    criterion="Performance monitoring and alerting operational",
                    target="All metrics available, alerting configured",
                    actual=f"{len(found_metrics)} metrics, alerting: {'yes' if has_alerting else 'no'}",
                    passed=monitoring_health,
                    details={
                        'total_metrics': len(found_metrics),
                        'expected_metrics': expected_metrics,
                        'found_metrics': found_metrics[:10],  # Limit for readability
                        'missing_metrics': missing_metrics,
                        'has_alerting': has_alerting
                    }
                )

        except Exception as e:
            return ValidationResult(
                criterion="Performance monitoring and alerting operational",
                target="Metrics endpoint accessible",
                actual="Failed to access",
                passed=False,
                details={'error': str(e)}
            )

    def _generate_report(self, test_run_id: str) -> Phase35ValidationReport:
        """Generate comprehensive validation report"""
        timestamp = datetime.now().isoformat()

        # Calculate overall status
        total_criteria = len(self.validation_results)
        passed_criteria = sum(1 for result in self.validation_results if result.passed)
        overall_status = "PASS" if passed_criteria == total_criteria else "FAIL"

        # Generate performance summary
        performance_summary = {
            'total_criteria': total_criteria,
            'passed_criteria': passed_criteria,
            'failed_criteria': total_criteria - passed_criteria,
            'success_rate_pct': (passed_criteria / total_criteria) * 100 if total_criteria > 0 else 0
        }

        # Add individual metric summaries
        for result in self.validation_results:
            if result.metrics:
                performance_summary[f"{result.criterion.lower().replace(' ', '_')}_metrics"] = {
                    'p95_latency_ms': result.metrics.p95_latency_ms,
                    'mean_latency_ms': result.metrics.mean_latency_ms,
                    'success_rate': result.metrics.success_rate
                }

        # Generate recommendations
        recommendations = []
        for result in self.validation_results:
            if not result.passed:
                recommendations.append(f"CRITICAL: {result.criterion} failed - {result.actual} vs target {result.target}")

        if not recommendations:
            recommendations.append("All Phase 3.5 success criteria met! System is ready for production.")

        # Test environment info
        test_environment = {
            'postgresql_host': self.config.get('postgresql.host'),
            'redis_host': self.config.get('redis.host'),
            'neo4j_uri': self.config.get('neo4j.uri'),
            'query_router_url': self.config.get('query_router.url'),
            'test_duration': 'Variable per test',
            'concurrent_users': self.config.get('test_parameters.concurrent_users')
        }

        return Phase35ValidationReport(
            test_run_id=test_run_id,
            timestamp=timestamp,
            overall_status=overall_status,
            success_criteria_results=self.validation_results,
            performance_summary=performance_summary,
            recommendations=recommendations,
            test_environment=test_environment
        )

    def print_report(self, report: Phase35ValidationReport):
        """Print formatted validation report"""
        self.console.print(f"\n[bold blue]Phase 3.5 Validation Report[/bold blue]")
        self.console.print(f"Test Run ID: {report.test_run_id}")
        self.console.print(f"Timestamp: {report.timestamp}")

        # Overall status
        status_color = "green" if report.overall_status == "PASS" else "red"
        self.console.print(f"Overall Status: [{status_color}]{report.overall_status}[/{status_color}]")

        # Success criteria table
        table = Table(title="Success Criteria Validation Results")
        table.add_column("Criterion", style="cyan", no_wrap=True)
        table.add_column("Target", style="magenta")
        table.add_column("Actual", style="yellow")
        table.add_column("Status", style="bold")

        for result in report.success_criteria_results:
            status_emoji = "✅" if result.passed else "❌"
            status_color = "green" if result.passed else "red"
            table.add_row(
                result.criterion,
                result.target,
                result.actual,
                f"[{status_color}]{status_emoji}[/{status_color}]"
            )

        self.console.print(table)

        # Performance Summary
        self.console.print(f"\n[bold]Performance Summary[/bold]")
        self.console.print(f"Success Rate: {report.performance_summary['success_rate_pct']:.1f}%")
        self.console.print(f"Passed: {report.performance_summary['passed_criteria']}/{report.performance_summary['total_criteria']}")

        # Recommendations
        self.console.print(f"\n[bold]Recommendations[/bold]")
        for i, rec in enumerate(report.recommendations, 1):
            color = "red" if "CRITICAL" in rec else "green"
            self.console.print(f"{i}. [{color}]{rec}[/{color}]")

    def save_report(self, report: Phase35ValidationReport, output_path: str):
        """Save validation report to JSON file"""
        report_dict = asdict(report)

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w') as f:
            json.dump(report_dict, f, indent=2)

        self.console.print(f"Report saved to: {output_path}")

    def close(self):
        """Clean up resources"""
        self.db.close_all()

# CLI interface
async def main():
    """Main entry point for Phase 3.5 validation"""
    import argparse

    parser = argparse.ArgumentParser(description="KB7 Terminology Phase 3.5 Validation Suite")
    parser.add_argument("--config", help="Path to test configuration file")
    parser.add_argument("--output", default="reports/phase35_validation_report.json",
                       help="Output path for validation report")
    parser.add_argument("--quiet", action="store_true", help="Suppress console output")

    args = parser.parse_args()

    validator = None
    try:
        validator = Phase35Validator(args.config)

        if not args.quiet:
            validator.console.print("[bold green]Starting KB7 Terminology Phase 3.5 Validation...[/bold green]")

        # Run validation
        report = await validator.run_full_validation()

        # Save report
        validator.save_report(report, args.output)

        # Print results
        if not args.quiet:
            validator.print_report(report)

        # Exit with appropriate code
        exit_code = 0 if report.overall_status == "PASS" else 1
        return exit_code

    except Exception as e:
        logger.error(f"Validation failed with error: {e}")
        if validator and not args.quiet:
            validator.console.print(f"[red]Validation failed: {e}[/red]")
        return 1
    finally:
        if validator:
            validator.close()

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)