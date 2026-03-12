#!/usr/bin/env python3
"""
Load Testing Suite for KB7 Terminology Phase 3.5
=================================================

Comprehensive load testing with concurrent users for:
- Stress testing with escalating load
- Soak testing for sustained load
- Spike testing for sudden load increases
- Capacity planning and breaking point analysis
- Resource utilization monitoring

Author: Claude Code Performance Engineer
Version: 1.0.0
"""

import asyncio
import time
import statistics
import random
import json
import os
import psutil
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple, NamedTuple
from dataclasses import dataclass, asdict
from concurrent.futures import ThreadPoolExecutor, as_completed
import logging
from collections import defaultdict
import threading

import httpx
import psycopg2
import redis
from neo4j import GraphDatabase
import numpy as np
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, TaskID
from rich.live import Live
from rich.layout import Layout
from rich.panel import Panel
from rich.text import Text
import yaml

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class LoadTestConfig:
    """Configuration for load testing scenarios"""
    test_name: str
    description: str
    max_users: int
    ramp_up_time_seconds: int
    test_duration_seconds: int
    ramp_down_time_seconds: int
    request_distribution: Dict[str, float]  # endpoint -> percentage
    think_time_ms: Tuple[int, int]  # min, max think time between requests
    spike_users: Optional[int] = None  # For spike testing
    spike_duration_seconds: Optional[int] = None

@dataclass
class LoadTestMetrics:
    """Metrics collected during load testing"""
    timestamp: str
    active_users: int
    requests_per_second: float
    mean_response_time_ms: float
    p95_response_time_ms: float
    p99_response_time_ms: float
    error_rate_pct: float
    cpu_usage_pct: float
    memory_usage_pct: float
    db_connections: int
    cache_hit_ratio: float
    network_io_mbps: float

@dataclass
class LoadTestResult:
    """Complete result of a load test run"""
    test_config: LoadTestConfig
    start_time: str
    end_time: str
    total_duration_seconds: float
    total_requests: int
    successful_requests: int
    failed_requests: int
    peak_rps: float
    peak_users: int
    breaking_point_users: Optional[int]
    metrics_timeline: List[LoadTestMetrics]
    error_summary: Dict[str, int]
    performance_summary: Dict[str, Any]
    recommendations: List[str]

class SystemMonitor:
    """Monitors system resources during load testing"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.monitoring = False
        self.metrics = []
        self.monitor_thread = None

        # Initialize database connections for monitoring
        try:
            self.redis_conn = redis.Redis(
                host=config['redis']['host'],
                port=config['redis']['port'],
                db=config['redis']['db'],
                decode_responses=True
            )
        except:
            self.redis_conn = None
            logger.warning("Could not connect to Redis for monitoring")

        try:
            self.pg_conn = psycopg2.connect(
                host=config['postgresql']['host'],
                port=config['postgresql']['port'],
                database=config['postgresql']['database'],
                user=config['postgresql']['user'],
                password=config['postgresql']['password']
            )
        except:
            self.pg_conn = None
            logger.warning("Could not connect to PostgreSQL for monitoring")

    def start_monitoring(self, interval_seconds: float = 1.0):
        """Start system monitoring in background thread"""
        self.monitoring = True
        self.monitor_thread = threading.Thread(
            target=self._monitoring_loop,
            args=(interval_seconds,),
            daemon=True
        )
        self.monitor_thread.start()

    def stop_monitoring(self):
        """Stop system monitoring"""
        self.monitoring = False
        if self.monitor_thread:
            self.monitor_thread.join(timeout=5.0)

    def _monitoring_loop(self, interval_seconds: float):
        """Main monitoring loop"""
        while self.monitoring:
            try:
                metrics = self._collect_metrics()
                self.metrics.append(metrics)
                time.sleep(interval_seconds)
            except Exception as e:
                logger.error(f"Error collecting metrics: {e}")
                time.sleep(interval_seconds)

    def _collect_metrics(self) -> Dict[str, Any]:
        """Collect system metrics at a point in time"""
        # CPU and Memory
        cpu_percent = psutil.cpu_percent(interval=0.1)
        memory = psutil.virtual_memory()

        # Network I/O
        net_io = psutil.net_io_counters()

        # Database connections
        db_connections = 0
        if self.pg_conn and not self.pg_conn.closed:
            try:
                with self.pg_conn.cursor() as cur:
                    cur.execute("SELECT count(*) FROM pg_stat_activity WHERE state = 'active'")
                    db_connections = cur.fetchone()[0]
            except:
                pass

        # Cache hit ratio
        cache_hit_ratio = 0.0
        if self.redis_conn:
            try:
                info = self.redis_conn.info('stats')
                hits = info.get('keyspace_hits', 0)
                misses = info.get('keyspace_misses', 0)
                total = hits + misses
                if total > 0:
                    cache_hit_ratio = (hits / total) * 100
            except:
                pass

        return {
            'timestamp': datetime.now().isoformat(),
            'cpu_usage_pct': cpu_percent,
            'memory_usage_pct': memory.percent,
            'memory_available_mb': memory.available / (1024 * 1024),
            'db_connections': db_connections,
            'cache_hit_ratio': cache_hit_ratio,
            'network_bytes_sent': net_io.bytes_sent,
            'network_bytes_recv': net_io.bytes_recv
        }

    def get_metrics_summary(self) -> Dict[str, Any]:
        """Get summary of collected metrics"""
        if not self.metrics:
            return {}

        cpu_values = [m['cpu_usage_pct'] for m in self.metrics]
        memory_values = [m['memory_usage_pct'] for m in self.metrics]
        db_conn_values = [m['db_connections'] for m in self.metrics]
        cache_values = [m['cache_hit_ratio'] for m in self.metrics if m['cache_hit_ratio'] > 0]

        return {
            'cpu_usage': {
                'mean': statistics.mean(cpu_values),
                'max': max(cpu_values),
                'p95': np.percentile(cpu_values, 95)
            },
            'memory_usage': {
                'mean': statistics.mean(memory_values),
                'max': max(memory_values),
                'p95': np.percentile(memory_values, 95)
            },
            'db_connections': {
                'mean': statistics.mean(db_conn_values),
                'max': max(db_conn_values),
                'p95': np.percentile(db_conn_values, 95)
            },
            'cache_hit_ratio': {
                'mean': statistics.mean(cache_values) if cache_values else 0,
                'min': min(cache_values) if cache_values else 0
            }
        }

class ConcurrentUser:
    """Simulates a single concurrent user"""

    def __init__(self, user_id: int, config: Dict[str, Any], endpoints: Dict[str, callable]):
        self.user_id = user_id
        self.config = config
        self.endpoints = endpoints
        self.request_count = 0
        self.latencies = []
        self.errors = []
        self.active = False

    async def run_user_session(self, duration_seconds: float, request_distribution: Dict[str, float],
                              think_time_range: Tuple[int, int]):
        """Run a user session for the specified duration"""
        self.active = True
        start_time = time.time()
        end_time = start_time + duration_seconds

        # Prepare endpoint selection based on distribution
        endpoint_choices = []
        for endpoint, weight in request_distribution.items():
            count = int(weight * 100)  # Convert percentage to count
            endpoint_choices.extend([endpoint] * count)

        while time.time() < end_time and self.active:
            try:
                # Select endpoint based on distribution
                endpoint_name = random.choice(endpoint_choices)
                endpoint_func = self.endpoints.get(endpoint_name)

                if endpoint_func:
                    start_request = time.perf_counter()
                    await endpoint_func()
                    end_request = time.perf_counter()

                    latency_ms = (end_request - start_request) * 1000
                    self.latencies.append(latency_ms)
                    self.request_count += 1

                # Think time between requests
                think_time = random.uniform(think_time_range[0], think_time_range[1]) / 1000
                await asyncio.sleep(think_time)

            except Exception as e:
                self.errors.append(str(e))
                # Short delay after error
                await asyncio.sleep(0.1)

        self.active = False

    def get_metrics(self) -> Dict[str, Any]:
        """Get user metrics"""
        if self.latencies:
            return {
                'user_id': self.user_id,
                'total_requests': self.request_count,
                'total_errors': len(self.errors),
                'mean_latency_ms': statistics.mean(self.latencies),
                'p95_latency_ms': np.percentile(self.latencies, 95),
                'max_latency_ms': max(self.latencies),
                'min_latency_ms': min(self.latencies)
            }
        else:
            return {
                'user_id': self.user_id,
                'total_requests': 0,
                'total_errors': len(self.errors),
                'mean_latency_ms': 0,
                'p95_latency_ms': 0,
                'max_latency_ms': 0,
                'min_latency_ms': 0
            }

class LoadTestRunner:
    """Main load testing orchestrator"""

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_config(config_path)
        self.console = Console()
        self.monitor = SystemMonitor(self.config)
        self.users = []
        self.user_metrics_timeline = []

        # Define test scenarios
        self.test_scenarios = {
            'stress_test': LoadTestConfig(
                test_name='stress_test',
                description='Gradually increase load to find breaking point',
                max_users=100,
                ramp_up_time_seconds=300,  # 5 minutes ramp up
                test_duration_seconds=600,  # 10 minutes sustained
                ramp_down_time_seconds=60,  # 1 minute ramp down
                request_distribution={
                    'concept_lookup': 0.4,
                    'concept_search': 0.2,
                    'valueset_expand': 0.15,
                    'fhir_lookup': 0.15,
                    'semantic_query': 0.1
                },
                think_time_ms=(100, 2000)
            ),
            'soak_test': LoadTestConfig(
                test_name='soak_test',
                description='Sustained load over extended period',
                max_users=50,
                ramp_up_time_seconds=120,
                test_duration_seconds=3600,  # 1 hour
                ramp_down_time_seconds=60,
                request_distribution={
                    'concept_lookup': 0.5,
                    'concept_search': 0.25,
                    'valueset_expand': 0.15,
                    'fhir_lookup': 0.1
                },
                think_time_ms=(500, 3000)
            ),
            'spike_test': LoadTestConfig(
                test_name='spike_test',
                description='Sudden spike in load',
                max_users=20,
                ramp_up_time_seconds=60,
                test_duration_seconds=300,
                ramp_down_time_seconds=60,
                request_distribution={
                    'concept_lookup': 0.6,
                    'fhir_lookup': 0.4
                },
                think_time_ms=(50, 500),
                spike_users=80,
                spike_duration_seconds=120
            ),
            'capacity_test': LoadTestConfig(
                test_name='capacity_test',
                description='Find maximum capacity',
                max_users=200,
                ramp_up_time_seconds=600,  # 10 minutes
                test_duration_seconds=300,  # 5 minutes sustained
                ramp_down_time_seconds=120,
                request_distribution={
                    'concept_lookup': 0.3,
                    'concept_search': 0.2,
                    'valueset_expand': 0.2,
                    'fhir_lookup': 0.2,
                    'semantic_query': 0.1
                },
                think_time_ms=(100, 1000)
            )
        }

    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load configuration"""
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
            'fhir_service': {
                'url': 'http://localhost:8014',
                'terminology_endpoint': '/terminology'
            },
            'query_router': {
                'url': 'http://localhost:8090'
            }
        }

        if config_path and os.path.exists(config_path):
            try:
                with open(config_path, 'r') as f:
                    loaded_config = yaml.safe_load(f)
                    default_config.update(loaded_config)
            except Exception as e:
                logger.warning(f"Could not load config: {e}")

        return default_config

    def _create_endpoint_functions(self) -> Dict[str, callable]:
        """Create endpoint functions for load testing"""

        async def concept_lookup():
            """PostgreSQL concept lookup"""
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.config['query_router']['url']}/query/concept",
                    params={
                        'system': 'http://snomed.info/sct',
                        'code': random.choice(['424144002', '38341003', '195967001'])
                    }
                )
                if response.status_code not in [200, 404]:
                    raise httpx.HTTPStatusError(f"Status {response.status_code}", request=response.request, response=response)

        async def concept_search():
            """PostgreSQL concept search"""
            async with httpx.AsyncClient(timeout=30.0) as client:
                terms = ['hypertension', 'diabetes', 'asthma', 'pneumonia', 'cardiac']
                response = await client.get(
                    f"{self.config['query_router']['url']}/query/search",
                    params={'term': random.choice(terms)}
                )
                if response.status_code not in [200, 404]:
                    raise httpx.HTTPStatusError(f"Status {response.status_code}", request=response.request, response=response)

        async def valueset_expand():
            """Value set expansion"""
            async with httpx.AsyncClient(timeout=30.0) as client:
                valuesets = ['cardiovascular-conditions', 'diabetes-medications', 'respiratory-conditions']
                response = await client.get(
                    f"{self.config['query_router']['url']}/query/valueset",
                    params={'id': random.choice(valuesets)}
                )
                if response.status_code not in [200, 404]:
                    raise httpx.HTTPStatusError(f"Status {response.status_code}", request=response.request, response=response)

        async def fhir_lookup():
            """FHIR CodeSystem lookup"""
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.config['fhir_service']['url']}{self.config['fhir_service']['terminology_endpoint']}/CodeSystem/$lookup",
                    params={
                        'system': 'http://snomed.info/sct',
                        'code': random.choice(['424144002', '38341003', '195967001'])
                    }
                )
                if response.status_code not in [200, 404]:
                    raise httpx.HTTPStatusError(f"Status {response.status_code}", request=response.request, response=response)

        async def semantic_query():
            """Neo4j semantic relationship query"""
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.post(
                    f"{self.config['query_router']['url']}/query/semantic",
                    json={
                        'concept_code': random.choice(['424144002', '38341003']),
                        'relationship_types': ['associated_with', 'treats']
                    }
                )
                if response.status_code not in [200, 404]:
                    raise httpx.HTTPStatusError(f"Status {response.status_code}", request=response.request, response=response)

        return {
            'concept_lookup': concept_lookup,
            'concept_search': concept_search,
            'valueset_expand': valueset_expand,
            'fhir_lookup': fhir_lookup,
            'semantic_query': semantic_query
        }

    async def run_load_test(self, test_scenario_name: str) -> LoadTestResult:
        """Run a specific load test scenario"""
        if test_scenario_name not in self.test_scenarios:
            raise ValueError(f"Unknown test scenario: {test_scenario_name}")

        test_config = self.test_scenarios[test_scenario_name]
        self.console.print(f"[bold green]Starting Load Test: {test_config.test_name}[/bold green]")
        self.console.print(f"Description: {test_config.description}")

        start_time = datetime.now()
        endpoint_functions = self._create_endpoint_functions()

        # Start system monitoring
        self.monitor.start_monitoring(interval_seconds=2.0)

        try:
            result = await self._execute_load_test(test_config, endpoint_functions)
            result.start_time = start_time.isoformat()
            result.end_time = datetime.now().isoformat()
            return result

        finally:
            # Stop monitoring
            self.monitor.stop_monitoring()

    async def _execute_load_test(self, config: LoadTestConfig, endpoints: Dict[str, callable]) -> LoadTestResult:
        """Execute the actual load test"""
        total_start_time = time.time()

        # Phase 1: Ramp Up
        self.console.print(f"[yellow]Phase 1: Ramping up to {config.max_users} users over {config.ramp_up_time_seconds}s[/yellow]")
        await self._ramp_up_users(config, endpoints)

        # Phase 2: Spike (if configured)
        if config.spike_users and config.spike_duration_seconds:
            self.console.print(f"[red]Spike: Adding {config.spike_users} users for {config.spike_duration_seconds}s[/red]")
            await self._execute_spike(config, endpoints)

        # Phase 3: Sustained Load
        self.console.print(f"[green]Phase 2: Sustained load for {config.test_duration_seconds}s[/green]")
        await self._sustain_load(config.test_duration_seconds)

        # Phase 4: Ramp Down
        self.console.print(f"[cyan]Phase 3: Ramping down over {config.ramp_down_time_seconds}s[/cyan]")
        await self._ramp_down_users(config.ramp_down_time_seconds)

        total_end_time = time.time()

        # Collect results
        return self._compile_results(config, total_start_time, total_end_time)

    async def _ramp_up_users(self, config: LoadTestConfig, endpoints: Dict[str, callable]):
        """Gradually ramp up users"""
        users_per_interval = max(1, config.max_users // 10)  # Add users in 10 intervals
        interval_duration = config.ramp_up_time_seconds / 10

        for i in range(10):
            # Add users for this interval
            for j in range(users_per_interval):
                if len(self.users) < config.max_users:
                    user_id = len(self.users)
                    user = ConcurrentUser(user_id, self.config, endpoints)
                    self.users.append(user)

                    # Start user session (will run until manually stopped)
                    asyncio.create_task(user.run_user_session(
                        float('inf'),  # Run indefinitely until stopped
                        config.request_distribution,
                        config.think_time_ms
                    ))

            # Wait for interval
            await asyncio.sleep(interval_duration)

            # Log progress
            active_users = len([u for u in self.users if u.active])
            self.console.print(f"Active users: {active_users}")

    async def _execute_spike(self, config: LoadTestConfig, endpoints: Dict[str, callable]):
        """Execute spike load"""
        spike_users = []

        # Add spike users quickly
        for i in range(config.spike_users):
            user_id = len(self.users) + len(spike_users)
            user = ConcurrentUser(user_id, self.config, endpoints)
            spike_users.append(user)

            # Start spike user session
            asyncio.create_task(user.run_user_session(
                config.spike_duration_seconds,
                config.request_distribution,
                (50, 200)  # Faster requests during spike
            ))

        # Wait for spike duration
        await asyncio.sleep(config.spike_duration_seconds)

        # Spike users will naturally stop after their duration

    async def _sustain_load(self, duration_seconds: float):
        """Maintain current load for specified duration"""
        with Progress() as progress:
            task = progress.add_task("[green]Sustaining load...", total=100)

            start_time = time.time()
            while time.time() - start_time < duration_seconds:
                elapsed = time.time() - start_time
                progress.update(task, completed=(elapsed / duration_seconds) * 100)

                # Log current metrics every 30 seconds
                if int(elapsed) % 30 == 0:
                    active_users = len([u for u in self.users if u.active])
                    total_requests = sum(u.request_count for u in self.users)
                    rps = total_requests / elapsed if elapsed > 0 else 0
                    self.console.print(f"Active: {active_users} users, Total requests: {total_requests}, RPS: {rps:.1f}")

                await asyncio.sleep(1)

    async def _ramp_down_users(self, ramp_down_seconds: float):
        """Gradually stop users"""
        if not self.users:
            return

        users_per_interval = max(1, len(self.users) // 5)  # Remove in 5 intervals
        interval_duration = ramp_down_seconds / 5

        active_users = [u for u in self.users if u.active]

        for i in range(5):
            # Stop users for this interval
            users_to_stop = active_users[i * users_per_interval:(i + 1) * users_per_interval]
            for user in users_to_stop:
                user.active = False

            await asyncio.sleep(interval_duration)

            remaining_active = len([u for u in self.users if u.active])
            self.console.print(f"Remaining active users: {remaining_active}")

        # Stop any remaining users
        for user in self.users:
            user.active = False

    def _compile_results(self, config: LoadTestConfig, start_time: float, end_time: float) -> LoadTestResult:
        """Compile test results"""
        total_duration = end_time - start_time

        # Aggregate user metrics
        total_requests = sum(u.request_count for u in self.users)
        successful_requests = total_requests
        failed_requests = sum(len(u.errors) for u in self.users)

        all_latencies = []
        for user in self.users:
            all_latencies.extend(user.latencies)

        # Calculate performance metrics
        if all_latencies:
            mean_latency = statistics.mean(all_latencies)
            p95_latency = np.percentile(all_latencies, 95)
            p99_latency = np.percentile(all_latencies, 99)
        else:
            mean_latency = p95_latency = p99_latency = 0

        peak_rps = total_requests / total_duration if total_duration > 0 else 0
        peak_users = config.max_users + (config.spike_users or 0)

        # Error summary
        error_summary = {}
        for user in self.users:
            for error in user.errors:
                error_type = type(error).__name__ if isinstance(error, Exception) else str(error)[:50]
                error_summary[error_type] = error_summary.get(error_type, 0) + 1

        # System metrics summary
        system_metrics = self.monitor.get_metrics_summary()

        # Performance summary
        performance_summary = {
            'peak_rps': peak_rps,
            'mean_latency_ms': mean_latency,
            'p95_latency_ms': p95_latency,
            'p99_latency_ms': p99_latency,
            'error_rate_pct': (failed_requests / total_requests * 100) if total_requests > 0 else 0,
            'system_metrics': system_metrics
        }

        # Generate recommendations
        recommendations = self._generate_recommendations(performance_summary, config)

        # Determine breaking point (simplified)
        breaking_point_users = None
        if performance_summary['error_rate_pct'] > 5 or performance_summary['p95_latency_ms'] > 1000:
            breaking_point_users = peak_users

        return LoadTestResult(
            test_config=config,
            start_time='',  # Will be set by caller
            end_time='',    # Will be set by caller
            total_duration_seconds=total_duration,
            total_requests=total_requests,
            successful_requests=successful_requests,
            failed_requests=failed_requests,
            peak_rps=peak_rps,
            peak_users=peak_users,
            breaking_point_users=breaking_point_users,
            metrics_timeline=[],  # Could be populated with detailed timeline
            error_summary=error_summary,
            performance_summary=performance_summary,
            recommendations=recommendations
        )

    def _generate_recommendations(self, performance_summary: Dict[str, Any],
                                config: LoadTestConfig) -> List[str]:
        """Generate performance recommendations"""
        recommendations = []

        # Latency recommendations
        if performance_summary['p95_latency_ms'] > 100:
            recommendations.append("HIGH: P95 latency exceeds 100ms - consider query optimization")

        if performance_summary['p95_latency_ms'] > 500:
            recommendations.append("CRITICAL: P95 latency exceeds 500ms - immediate optimization required")

        # Error rate recommendations
        if performance_summary['error_rate_pct'] > 1:
            recommendations.append(f"MEDIUM: Error rate {performance_summary['error_rate_pct']:.1f}% - investigate error causes")

        if performance_summary['error_rate_pct'] > 5:
            recommendations.append("HIGH: Error rate >5% - system reliability issues")

        # System resource recommendations
        system_metrics = performance_summary.get('system_metrics', {})

        cpu_max = system_metrics.get('cpu_usage', {}).get('max', 0)
        if cpu_max > 80:
            recommendations.append(f"HIGH: Peak CPU usage {cpu_max:.1f}% - consider horizontal scaling")

        memory_max = system_metrics.get('memory_usage', {}).get('max', 0)
        if memory_max > 85:
            recommendations.append(f"HIGH: Peak memory usage {memory_max:.1f}% - memory optimization needed")

        # Database recommendations
        db_max = system_metrics.get('db_connections', {}).get('max', 0)
        if db_max > 80:
            recommendations.append("MEDIUM: High database connection usage - consider connection pooling optimization")

        # Cache recommendations
        cache_hit = system_metrics.get('cache_hit_ratio', {}).get('mean', 0)
        if cache_hit < 80:
            recommendations.append(f"MEDIUM: Cache hit ratio {cache_hit:.1f}% - improve caching strategy")

        # Capacity recommendations
        if config.max_users < 50:
            recommendations.append("INFO: Test with higher user loads to find true capacity limits")

        if not recommendations:
            recommendations.append("GOOD: System performed well under test load")

        return recommendations

    def print_results(self, result: LoadTestResult):
        """Print formatted load test results"""
        self.console.print(f"\n[bold blue]Load Test Results: {result.test_config.test_name}[/bold blue]")

        # Summary table
        table = Table(title="Performance Summary")
        table.add_column("Metric", style="cyan")
        table.add_column("Value", style="magenta")

        table.add_row("Test Duration", f"{result.total_duration_seconds:.1f}s")
        table.add_row("Peak Users", str(result.peak_users))
        table.add_row("Total Requests", str(result.total_requests))
        table.add_row("Peak RPS", f"{result.peak_rps:.1f}")
        table.add_row("Success Rate", f"{(result.successful_requests/result.total_requests*100):.1f}%" if result.total_requests > 0 else "0%")
        table.add_row("Mean Latency", f"{result.performance_summary['mean_latency_ms']:.1f}ms")
        table.add_row("P95 Latency", f"{result.performance_summary['p95_latency_ms']:.1f}ms")
        table.add_row("P99 Latency", f"{result.performance_summary['p99_latency_ms']:.1f}ms")

        if result.breaking_point_users:
            table.add_row("Breaking Point", f"{result.breaking_point_users} users", style="red")

        self.console.print(table)

        # Error summary
        if result.error_summary:
            self.console.print("\n[bold red]Error Summary[/bold red]")
            error_table = Table()
            error_table.add_column("Error Type", style="red")
            error_table.add_column("Count", style="yellow")

            for error, count in result.error_summary.items():
                error_table.add_row(error, str(count))

            self.console.print(error_table)

        # Recommendations
        self.console.print("\n[bold]Recommendations[/bold]")
        for i, rec in enumerate(result.recommendations, 1):
            if "CRITICAL" in rec:
                color = "red"
            elif "HIGH" in rec:
                color = "red"
            elif "MEDIUM" in rec:
                color = "yellow"
            else:
                color = "green"

            self.console.print(f"{i}. [{color}]{rec}[/{color}]")

    def save_results(self, result: LoadTestResult, output_path: str):
        """Save load test results to JSON file"""
        result_dict = asdict(result)

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w') as f:
            json.dump(result_dict, f, indent=2)

        self.console.print(f"Results saved to: {output_path}")

    def cleanup(self):
        """Clean up resources"""
        # Stop all users
        for user in self.users:
            user.active = False

        self.monitor.stop_monitoring()

# CLI interface
async def main():
    """Main entry point for load testing"""
    import argparse

    parser = argparse.ArgumentParser(description="KB7 Terminology Load Testing")
    parser.add_argument("--scenario", default="stress_test",
                       choices=['stress_test', 'soak_test', 'spike_test', 'capacity_test'],
                       help="Load test scenario to run")
    parser.add_argument("--config", help="Path to configuration file")
    parser.add_argument("--output", help="Output file for results (JSON)")

    args = parser.parse_args()

    runner = None
    try:
        runner = LoadTestRunner(args.config)

        # Run the specified load test
        result = await runner.run_load_test(args.scenario)

        # Print results
        runner.print_results(result)

        # Save results if output specified
        if args.output:
            runner.save_results(result, args.output)

        # Determine exit code based on results
        if result.performance_summary['error_rate_pct'] > 5:
            return 1  # High error rate
        elif result.performance_summary['p95_latency_ms'] > 1000:
            return 1  # Unacceptable latency
        else:
            return 0  # Success

    except Exception as e:
        logger.error(f"Load test failed: {e}")
        return 1
    finally:
        if runner:
            runner.cleanup()

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)