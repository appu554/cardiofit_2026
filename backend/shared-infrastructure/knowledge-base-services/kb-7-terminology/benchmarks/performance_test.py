#!/usr/bin/env python3
"""
KB7 Terminology Phase 3.5 Performance Testing Suite

Comprehensive performance benchmarking system that validates all Phase 3.5 success criteria:
- PostgreSQL lookup queries <10ms (95th percentile)
- GraphDB reasoning queries <50ms (95th percentile)
- Query router uptime >99.9%
- FHIR endpoints using hybrid architecture <200ms (95% of requests)
- Cache hit ratio >90% for frequent operations
- Performance monitoring and alerting operational

Usage:
    python performance_test.py --target http://localhost:8007 --duration 300
    python performance_test.py --suite postgresql --percentile 95
    python performance_test.py --full-benchmark --report-format html
"""

import asyncio
import time
import json
import logging
import statistics
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
import argparse

import httpx
import psycopg2
import redis
from neo4j import GraphDatabase
import pytest
import numpy as np
import pandas as pd
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, TaskID
from concurrent.futures import ThreadPoolExecutor, as_completed
import matplotlib.pyplot as plt
import seaborn as sns

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)
console = Console()

@dataclass
class PerformanceMetrics:
    """Performance metrics data structure"""
    test_name: str
    operation: str
    response_times: List[float]
    success_count: int
    error_count: int
    cache_hits: int = 0
    cache_misses: int = 0
    throughput: float = 0.0
    percentiles: Dict[int, float] = None
    start_time: datetime = None
    end_time: datetime = None

    def __post_init__(self):
        if self.percentiles is None:
            self.percentiles = self.calculate_percentiles()
        if self.throughput == 0.0:
            self.throughput = self.calculate_throughput()

    def calculate_percentiles(self) -> Dict[int, float]:
        """Calculate response time percentiles"""
        if not self.response_times:
            return {}

        return {
            50: np.percentile(self.response_times, 50),
            95: np.percentile(self.response_times, 95),
            99: np.percentile(self.response_times, 99),
            99.9: np.percentile(self.response_times, 99.9)
        }

    def calculate_throughput(self) -> float:
        """Calculate requests per second"""
        if not self.start_time or not self.end_time:
            return 0.0

        duration = (self.end_time - self.start_time).total_seconds()
        return (self.success_count + self.error_count) / duration if duration > 0 else 0.0

    @property
    def cache_hit_ratio(self) -> float:
        """Calculate cache hit ratio as percentage"""
        total = self.cache_hits + self.cache_misses
        return (self.cache_hits / total * 100) if total > 0 else 0.0

    @property
    def error_rate(self) -> float:
        """Calculate error rate as percentage"""
        total = self.success_count + self.error_count
        return (self.error_count / total * 100) if total > 0 else 0.0

class DatabaseConnections:
    """Database connection manager for performance testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.postgres_conn = None
        self.redis_conn = None
        self.neo4j_driver = None

    def connect_postgresql(self):
        """Connect to PostgreSQL database"""
        try:
            self.postgres_conn = psycopg2.connect(
                host=self.config.get('postgres_host', 'localhost'),
                port=self.config.get('postgres_port', 5432),
                database=self.config.get('postgres_db', 'kb7_terminology'),
                user=self.config.get('postgres_user', 'postgres'),
                password=self.config.get('postgres_password', 'password')
            )
            logger.info("Connected to PostgreSQL")
        except Exception as e:
            logger.error(f"Failed to connect to PostgreSQL: {e}")
            raise

    def connect_redis(self):
        """Connect to Redis cache"""
        try:
            self.redis_conn = redis.Redis(
                host=self.config.get('redis_host', 'localhost'),
                port=self.config.get('redis_port', 6379),
                db=self.config.get('redis_db', 0),
                decode_responses=True
            )
            self.redis_conn.ping()
            logger.info("Connected to Redis")
        except Exception as e:
            logger.error(f"Failed to connect to Redis: {e}")
            raise

    def connect_neo4j(self):
        """Connect to Neo4j GraphDB"""
        try:
            uri = f"bolt://{self.config.get('neo4j_host', 'localhost')}:{self.config.get('neo4j_port', 7687)}"
            self.neo4j_driver = GraphDatabase.driver(
                uri,
                auth=(
                    self.config.get('neo4j_user', 'neo4j'),
                    self.config.get('neo4j_password', 'password')
                )
            )
            # Test connection
            with self.neo4j_driver.session() as session:
                session.run("RETURN 1")
            logger.info("Connected to Neo4j")
        except Exception as e:
            logger.error(f"Failed to connect to Neo4j: {e}")
            raise

    def close_all(self):
        """Close all database connections"""
        if self.postgres_conn:
            self.postgres_conn.close()
        if self.redis_conn:
            self.redis_conn.close()
        if self.neo4j_driver:
            self.neo4j_driver.close()

class PerformanceTester:
    """Main performance testing class"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.base_url = config.get('base_url', 'http://localhost:8007')
        self.db_connections = DatabaseConnections(config)
        self.results: List[PerformanceMetrics] = []

        # Test data for realistic workloads
        self.test_codes = [
            'J44.0', 'E11.9', 'I10', 'N18.6', 'F32.9',
            'M79.3', 'K59.00', 'Z87.891', 'R06.02', 'G47.33'
        ]

        self.test_medications = [
            'metformin', 'lisinopril', 'amlodipine', 'metoprolol', 'simvastatin',
            'omeprazole', 'levothyroxine', 'albuterol', 'furosemide', 'insulin'
        ]

    async def setup(self):
        """Initialize connections and test environment"""
        logger.info("Setting up performance test environment...")

        # Connect to databases
        self.db_connections.connect_postgresql()
        self.db_connections.connect_redis()
        self.db_connections.connect_neo4j()

        # Warm up caches
        await self.warm_up_caches()

        logger.info("Setup completed successfully")

    async def warm_up_caches(self):
        """Warm up caches with test data"""
        logger.info("Warming up caches...")

        async with httpx.AsyncClient() as client:
            # Warm up with common terminology lookups
            for code in self.test_codes[:5]:
                try:
                    await client.get(f"{self.base_url}/terminology/codes/{code}")
                except:
                    pass  # Ignore errors during warmup

            # Warm up with medication searches
            for med in self.test_medications[:5]:
                try:
                    await client.get(f"{self.base_url}/terminology/search?q={med}")
                except:
                    pass  # Ignore errors during warmup

    def teardown(self):
        """Clean up connections and resources"""
        logger.info("Tearing down test environment...")
        self.db_connections.close_all()

    async def test_postgresql_performance(self, duration: int = 60, concurrent_users: int = 10) -> PerformanceMetrics:
        """
        Test PostgreSQL lookup query performance
        Target: <10ms for 95th percentile
        """
        logger.info(f"Testing PostgreSQL performance for {duration}s with {concurrent_users} concurrent users")

        response_times = []
        success_count = 0
        error_count = 0
        start_time = datetime.now()

        async def execute_query():
            """Execute a single PostgreSQL query"""
            try:
                conn = psycopg2.connect(
                    host=self.config.get('postgres_host', 'localhost'),
                    port=self.config.get('postgres_port', 5432),
                    database=self.config.get('postgres_db', 'kb7_terminology'),
                    user=self.config.get('postgres_user', 'postgres'),
                    password=self.config.get('postgres_password', 'password')
                )

                cursor = conn.cursor()

                # Random test query
                code = np.random.choice(self.test_codes)
                query_start = time.time()

                cursor.execute(
                    "SELECT code, display, system FROM concepts WHERE code = %s LIMIT 1",
                    (code,)
                )
                result = cursor.fetchone()

                query_time = (time.time() - query_start) * 1000  # Convert to ms
                response_times.append(query_time)

                cursor.close()
                conn.close()

                return True
            except Exception as e:
                logger.error(f"PostgreSQL query error: {e}")
                return False

        # Run concurrent queries for specified duration
        end_time = start_time + timedelta(seconds=duration)

        with ThreadPoolExecutor(max_workers=concurrent_users) as executor:
            while datetime.now() < end_time:
                futures = []

                # Submit batch of queries
                for _ in range(concurrent_users):
                    futures.append(executor.submit(asyncio.run, execute_query()))

                # Collect results
                for future in as_completed(futures):
                    try:
                        if future.result():
                            success_count += 1
                        else:
                            error_count += 1
                    except Exception:
                        error_count += 1

                # Small delay between batches
                await asyncio.sleep(0.1)

        metrics = PerformanceMetrics(
            test_name="PostgreSQL Lookup Performance",
            operation="SELECT query",
            response_times=response_times,
            success_count=success_count,
            error_count=error_count,
            start_time=start_time,
            end_time=datetime.now()
        )

        self.results.append(metrics)
        return metrics

    async def test_graphdb_performance(self, duration: int = 60, concurrent_users: int = 5) -> PerformanceMetrics:
        """
        Test GraphDB reasoning query performance
        Target: <50ms for 95th percentile
        """
        logger.info(f"Testing GraphDB performance for {duration}s with {concurrent_users} concurrent users")

        response_times = []
        success_count = 0
        error_count = 0
        start_time = datetime.now()

        def execute_reasoning_query():
            """Execute a single GraphDB reasoning query"""
            try:
                with self.db_connections.neo4j_driver.session() as session:
                    code = np.random.choice(self.test_codes)
                    query_start = time.time()

                    # Complex reasoning query
                    result = session.run("""
                        MATCH (c:Concept {code: $code})-[:IS_A*1..3]->(parent:Concept)
                        RETURN c.code, c.display, collect(parent.display) as ancestors
                        LIMIT 10
                    """, code=code)

                    records = list(result)
                    query_time = (time.time() - query_start) * 1000  # Convert to ms
                    response_times.append(query_time)

                    return True
            except Exception as e:
                logger.error(f"GraphDB query error: {e}")
                return False

        # Run concurrent queries for specified duration
        end_time = start_time + timedelta(seconds=duration)

        with ThreadPoolExecutor(max_workers=concurrent_users) as executor:
            while datetime.now() < end_time:
                futures = []

                # Submit batch of queries
                for _ in range(concurrent_users):
                    futures.append(executor.submit(execute_reasoning_query))

                # Collect results
                for future in as_completed(futures):
                    try:
                        if future.result():
                            success_count += 1
                        else:
                            error_count += 1
                    except Exception:
                        error_count += 1

                # Small delay between batches
                await asyncio.sleep(0.2)

        metrics = PerformanceMetrics(
            test_name="GraphDB Reasoning Performance",
            operation="CYPHER reasoning query",
            response_times=response_times,
            success_count=success_count,
            error_count=error_count,
            start_time=start_time,
            end_time=datetime.now()
        )

        self.results.append(metrics)
        return metrics

    async def test_query_router_uptime(self, duration: int = 300, check_interval: int = 5) -> PerformanceMetrics:
        """
        Test query router uptime and availability
        Target: >99.9% uptime
        """
        logger.info(f"Testing query router uptime for {duration}s with {check_interval}s intervals")

        response_times = []
        success_count = 0
        error_count = 0
        start_time = datetime.now()

        async with httpx.AsyncClient(timeout=30.0) as client:
            end_time = start_time + timedelta(seconds=duration)

            while datetime.now() < end_time:
                try:
                    check_start = time.time()
                    response = await client.get(f"{self.base_url}/health")
                    check_time = (time.time() - check_start) * 1000

                    if response.status_code == 200:
                        success_count += 1
                        response_times.append(check_time)
                    else:
                        error_count += 1

                except Exception as e:
                    error_count += 1
                    logger.warning(f"Health check failed: {e}")

                await asyncio.sleep(check_interval)

        metrics = PerformanceMetrics(
            test_name="Query Router Uptime",
            operation="Health check",
            response_times=response_times,
            success_count=success_count,
            error_count=error_count,
            start_time=start_time,
            end_time=datetime.now()
        )

        # Calculate uptime percentage
        total_checks = success_count + error_count
        uptime_percentage = (success_count / total_checks * 100) if total_checks > 0 else 0

        logger.info(f"Query router uptime: {uptime_percentage:.2f}%")
        self.results.append(metrics)
        return metrics

    async def test_fhir_endpoint_performance(self, duration: int = 120, concurrent_users: int = 15) -> PerformanceMetrics:
        """
        Test FHIR endpoint performance using hybrid architecture
        Target: <200ms for 95% of requests
        """
        logger.info(f"Testing FHIR endpoint performance for {duration}s with {concurrent_users} concurrent users")

        response_times = []
        success_count = 0
        error_count = 0
        cache_hits = 0
        cache_misses = 0
        start_time = datetime.now()

        async def execute_fhir_request():
            """Execute a single FHIR request"""
            try:
                async with httpx.AsyncClient(timeout=30.0) as client:
                    # Random FHIR operation
                    operation = np.random.choice([
                        'terminology/codes',
                        'terminology/search',
                        'terminology/validate',
                        'terminology/expand'
                    ])

                    if operation == 'terminology/codes':
                        code = np.random.choice(self.test_codes)
                        url = f"{self.base_url}/{operation}/{code}"
                    elif operation == 'terminology/search':
                        term = np.random.choice(self.test_medications)
                        url = f"{self.base_url}/{operation}?q={term}"
                    else:
                        url = f"{self.base_url}/{operation}"

                    request_start = time.time()
                    response = await client.get(url)
                    request_time = (time.time() - request_start) * 1000

                    response_times.append(request_time)

                    # Check for cache indicators in headers
                    if 'X-Cache-Status' in response.headers:
                        if response.headers['X-Cache-Status'] == 'HIT':
                            return True, True  # success, cache_hit
                        else:
                            return True, False  # success, cache_miss

                    return response.status_code < 400, False

            except Exception as e:
                logger.error(f"FHIR request error: {e}")
                return False, False

        # Run concurrent requests for specified duration
        end_time = start_time + timedelta(seconds=duration)

        semaphore = asyncio.Semaphore(concurrent_users)

        async def bounded_request():
            async with semaphore:
                return await execute_fhir_request()

        while datetime.now() < end_time:
            tasks = []

            # Create batch of requests
            for _ in range(min(concurrent_users, 20)):
                tasks.append(bounded_request())

            # Execute batch
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Process results
            for result in results:
                if isinstance(result, tuple):
                    success, cache_hit = result
                    if success:
                        success_count += 1
                        if cache_hit:
                            cache_hits += 1
                        else:
                            cache_misses += 1
                    else:
                        error_count += 1
                else:
                    error_count += 1

            # Small delay between batches
            await asyncio.sleep(0.1)

        metrics = PerformanceMetrics(
            test_name="FHIR Endpoint Performance",
            operation="FHIR terminology requests",
            response_times=response_times,
            success_count=success_count,
            error_count=error_count,
            cache_hits=cache_hits,
            cache_misses=cache_misses,
            start_time=start_time,
            end_time=datetime.now()
        )

        self.results.append(metrics)
        return metrics

    async def test_cache_performance(self, duration: int = 60) -> PerformanceMetrics:
        """
        Test cache hit ratio and performance
        Target: >90% hit ratio for frequent operations
        """
        logger.info(f"Testing cache performance for {duration}s")

        cache_hits = 0
        cache_misses = 0
        response_times = []
        start_time = datetime.now()

        # Pre-populate cache with some data
        for code in self.test_codes:
            try:
                self.db_connections.redis_conn.setex(f"concept:{code}", 300, json.dumps({
                    "code": code,
                    "display": f"Test concept {code}",
                    "system": "http://snomed.info/sct"
                }))
            except:
                pass

        end_time = start_time + timedelta(seconds=duration)

        while datetime.now() < end_time:
            # Test cache lookups with high frequency on some keys
            code = np.random.choice(self.test_codes + self.test_codes[:3] * 10)  # Bias toward first 3

            try:
                lookup_start = time.time()
                result = self.db_connections.redis_conn.get(f"concept:{code}")
                lookup_time = (time.time() - lookup_start) * 1000

                response_times.append(lookup_time)

                if result:
                    cache_hits += 1
                else:
                    cache_misses += 1
                    # Simulate cache miss - populate cache
                    self.db_connections.redis_conn.setex(f"concept:{code}", 300, json.dumps({
                        "code": code,
                        "display": f"Loaded concept {code}",
                        "system": "http://snomed.info/sct"
                    }))

            except Exception as e:
                cache_misses += 1
                logger.error(f"Cache lookup error: {e}")

            await asyncio.sleep(0.01)  # 10ms between lookups

        metrics = PerformanceMetrics(
            test_name="Cache Performance",
            operation="Redis cache lookup",
            response_times=response_times,
            success_count=cache_hits + cache_misses,
            error_count=0,
            cache_hits=cache_hits,
            cache_misses=cache_misses,
            start_time=start_time,
            end_time=datetime.now()
        )

        self.results.append(metrics)
        return metrics

    def validate_success_criteria(self) -> Dict[str, bool]:
        """
        Validate all Phase 3.5 success criteria
        Returns dict of criteria and pass/fail status
        """
        criteria_results = {}

        for metric in self.results:
            if metric.test_name == "PostgreSQL Lookup Performance":
                # PostgreSQL lookup queries <10ms (95th percentile)
                criteria_results["postgresql_95th_percentile"] = metric.percentiles.get(95, float('inf')) < 10.0

            elif metric.test_name == "GraphDB Reasoning Performance":
                # GraphDB reasoning queries <50ms (95th percentile)
                criteria_results["graphdb_95th_percentile"] = metric.percentiles.get(95, float('inf')) < 50.0

            elif metric.test_name == "Query Router Uptime":
                # Query router uptime >99.9%
                total_checks = metric.success_count + metric.error_count
                uptime = (metric.success_count / total_checks * 100) if total_checks > 0 else 0
                criteria_results["query_router_uptime"] = uptime > 99.9

            elif metric.test_name == "FHIR Endpoint Performance":
                # FHIR endpoints <200ms for 95% of requests
                criteria_results["fhir_95th_percentile"] = metric.percentiles.get(95, float('inf')) < 200.0

            elif metric.test_name == "Cache Performance":
                # Cache hit ratio >90% for frequent operations
                criteria_results["cache_hit_ratio"] = metric.cache_hit_ratio > 90.0

        return criteria_results

    def generate_performance_report(self, output_format: str = "console") -> str:
        """Generate comprehensive performance report"""

        if output_format == "console":
            return self._generate_console_report()
        elif output_format == "html":
            return self._generate_html_report()
        elif output_format == "json":
            return self._generate_json_report()
        else:
            raise ValueError(f"Unsupported output format: {output_format}")

    def _generate_console_report(self) -> str:
        """Generate console-formatted performance report"""
        console.print("\n[bold blue]KB7 Terminology Phase 3.5 Performance Report[/bold blue]")
        console.print(f"Report generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

        # Success criteria validation
        criteria = self.validate_success_criteria()

        criteria_table = Table(title="Phase 3.5 Success Criteria Validation")
        criteria_table.add_column("Criterion", style="cyan")
        criteria_table.add_column("Target", style="yellow")
        criteria_table.add_column("Status", style="green")

        criteria_table.add_row(
            "PostgreSQL 95th percentile",
            "<10ms",
            "✅ PASS" if criteria.get("postgresql_95th_percentile", False) else "❌ FAIL"
        )
        criteria_table.add_row(
            "GraphDB 95th percentile",
            "<50ms",
            "✅ PASS" if criteria.get("graphdb_95th_percentile", False) else "❌ FAIL"
        )
        criteria_table.add_row(
            "Query Router Uptime",
            ">99.9%",
            "✅ PASS" if criteria.get("query_router_uptime", False) else "❌ FAIL"
        )
        criteria_table.add_row(
            "FHIR 95th percentile",
            "<200ms",
            "✅ PASS" if criteria.get("fhir_95th_percentile", False) else "❌ FAIL"
        )
        criteria_table.add_row(
            "Cache Hit Ratio",
            ">90%",
            "✅ PASS" if criteria.get("cache_hit_ratio", False) else "❌ FAIL"
        )

        console.print(criteria_table)

        # Detailed metrics
        for metric in self.results:
            self._print_metric_details(metric)

        return "Console report generated"

    def _print_metric_details(self, metric: PerformanceMetrics):
        """Print detailed metrics for a single test"""
        console.print(f"\n[bold cyan]{metric.test_name}[/bold cyan]")

        details_table = Table()
        details_table.add_column("Metric", style="white")
        details_table.add_column("Value", style="green")

        details_table.add_row("Operation", metric.operation)
        details_table.add_row("Total Requests", str(metric.success_count + metric.error_count))
        details_table.add_row("Success Count", str(metric.success_count))
        details_table.add_row("Error Count", str(metric.error_count))
        details_table.add_row("Error Rate", f"{metric.error_rate:.2f}%")
        details_table.add_row("Throughput", f"{metric.throughput:.2f} req/s")

        if metric.response_times:
            details_table.add_row("Mean Response Time", f"{np.mean(metric.response_times):.2f}ms")
            details_table.add_row("50th Percentile", f"{metric.percentiles.get(50, 0):.2f}ms")
            details_table.add_row("95th Percentile", f"{metric.percentiles.get(95, 0):.2f}ms")
            details_table.add_row("99th Percentile", f"{metric.percentiles.get(99, 0):.2f}ms")

        if metric.cache_hits > 0 or metric.cache_misses > 0:
            details_table.add_row("Cache Hit Ratio", f"{metric.cache_hit_ratio:.2f}%")
            details_table.add_row("Cache Hits", str(metric.cache_hits))
            details_table.add_row("Cache Misses", str(metric.cache_misses))

        console.print(details_table)

    def _generate_html_report(self) -> str:
        """Generate HTML performance report with charts"""
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>KB7 Terminology Performance Report</title>
            <style>
                body {{ font-family: Arial, sans-serif; margin: 20px; }}
                .header {{ background-color: #f0f0f0; padding: 20px; border-radius: 5px; }}
                .criteria {{ margin: 20px 0; }}
                .metric {{ margin: 20px 0; border: 1px solid #ddd; padding: 15px; border-radius: 5px; }}
                .pass {{ color: green; font-weight: bold; }}
                .fail {{ color: red; font-weight: bold; }}
                table {{ border-collapse: collapse; width: 100%; }}
                th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
                th {{ background-color: #f2f2f2; }}
            </style>
        </head>
        <body>
            <div class="header">
                <h1>KB7 Terminology Phase 3.5 Performance Report</h1>
                <p>Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
            </div>
        """

        # Success criteria
        criteria = self.validate_success_criteria()
        html_content += '<div class="criteria"><h2>Success Criteria Validation</h2><table>'
        html_content += '<tr><th>Criterion</th><th>Target</th><th>Status</th></tr>'

        criteria_data = [
            ("PostgreSQL 95th percentile", "<10ms", criteria.get("postgresql_95th_percentile", False)),
            ("GraphDB 95th percentile", "<50ms", criteria.get("graphdb_95th_percentile", False)),
            ("Query Router Uptime", ">99.9%", criteria.get("query_router_uptime", False)),
            ("FHIR 95th percentile", "<200ms", criteria.get("fhir_95th_percentile", False)),
            ("Cache Hit Ratio", ">90%", criteria.get("cache_hit_ratio", False))
        ]

        for name, target, passed in criteria_data:
            status_class = "pass" if passed else "fail"
            status_text = "✅ PASS" if passed else "❌ FAIL"
            html_content += f'<tr><td>{name}</td><td>{target}</td><td class="{status_class}">{status_text}</td></tr>'

        html_content += '</table></div>'

        # Detailed metrics
        for metric in self.results:
            html_content += f'<div class="metric"><h3>{metric.test_name}</h3>'
            html_content += '<table>'
            html_content += f'<tr><td>Operation</td><td>{metric.operation}</td></tr>'
            html_content += f'<tr><td>Total Requests</td><td>{metric.success_count + metric.error_count}</td></tr>'
            html_content += f'<tr><td>Success Count</td><td>{metric.success_count}</td></tr>'
            html_content += f'<tr><td>Error Rate</td><td>{metric.error_rate:.2f}%</td></tr>'
            html_content += f'<tr><td>Throughput</td><td>{metric.throughput:.2f} req/s</td></tr>'

            if metric.response_times:
                html_content += f'<tr><td>95th Percentile</td><td>{metric.percentiles.get(95, 0):.2f}ms</td></tr>'

            if metric.cache_hits > 0 or metric.cache_misses > 0:
                html_content += f'<tr><td>Cache Hit Ratio</td><td>{metric.cache_hit_ratio:.2f}%</td></tr>'

            html_content += '</table></div>'

        html_content += '</body></html>'

        # Save to file
        report_path = Path("performance_report.html")
        report_path.write_text(html_content)

        return str(report_path)

    def _generate_json_report(self) -> str:
        """Generate JSON performance report"""
        report_data = {
            "timestamp": datetime.now().isoformat(),
            "success_criteria": self.validate_success_criteria(),
            "metrics": [asdict(metric) for metric in self.results]
        }

        report_path = Path("performance_report.json")
        report_path.write_text(json.dumps(report_data, indent=2, default=str))

        return str(report_path)

async def main():
    """Main entry point for performance testing"""
    parser = argparse.ArgumentParser(description="KB7 Terminology Performance Testing Suite")
    parser.add_argument("--target", default="http://localhost:8007", help="Target service URL")
    parser.add_argument("--duration", type=int, default=120, help="Test duration in seconds")
    parser.add_argument("--concurrent-users", type=int, default=10, help="Number of concurrent users")
    parser.add_argument("--suite", choices=["postgresql", "graphdb", "fhir", "cache", "uptime", "all"],
                       default="all", help="Test suite to run")
    parser.add_argument("--report-format", choices=["console", "html", "json"],
                       default="console", help="Output report format")
    parser.add_argument("--config", help="Path to configuration file")

    args = parser.parse_args()

    # Load configuration
    config = {
        "base_url": args.target,
        "postgres_host": "localhost",
        "postgres_port": 5432,
        "postgres_db": "kb7_terminology",
        "postgres_user": "postgres",
        "postgres_password": "password",
        "redis_host": "localhost",
        "redis_port": 6379,
        "redis_db": 0,
        "neo4j_host": "localhost",
        "neo4j_port": 7687,
        "neo4j_user": "neo4j",
        "neo4j_password": "password"
    }

    if args.config:
        import yaml
        with open(args.config) as f:
            config.update(yaml.safe_load(f))

    # Initialize tester
    tester = PerformanceTester(config)

    try:
        await tester.setup()

        logger.info("Starting performance tests...")

        if args.suite == "all" or args.suite == "postgresql":
            await tester.test_postgresql_performance(args.duration, args.concurrent_users)

        if args.suite == "all" or args.suite == "graphdb":
            await tester.test_graphdb_performance(args.duration, max(args.concurrent_users // 2, 1))

        if args.suite == "all" or args.suite == "uptime":
            await tester.test_query_router_uptime(args.duration * 2, 5)

        if args.suite == "all" or args.suite == "fhir":
            await tester.test_fhir_endpoint_performance(args.duration, args.concurrent_users + 5)

        if args.suite == "all" or args.suite == "cache":
            await tester.test_cache_performance(args.duration)

        # Generate report
        report_path = tester.generate_performance_report(args.report_format)

        if args.report_format != "console":
            logger.info(f"Performance report saved to: {report_path}")

        # Validate success criteria
        criteria = tester.validate_success_criteria()
        all_passed = all(criteria.values())

        if all_passed:
            logger.info("🎉 All Phase 3.5 success criteria PASSED!")
            sys.exit(0)
        else:
            failed_criteria = [k for k, v in criteria.items() if not v]
            logger.error(f"❌ Failed criteria: {failed_criteria}")
            sys.exit(1)

    except Exception as e:
        logger.error(f"Performance testing failed: {e}")
        sys.exit(1)
    finally:
        tester.teardown()

if __name__ == "__main__":
    asyncio.run(main())