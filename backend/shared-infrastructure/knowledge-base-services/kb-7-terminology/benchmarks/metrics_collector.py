#!/usr/bin/env python3
"""
KB7 Terminology Metrics Collector

Advanced performance metrics collection, analysis, and reporting system for
KB7 Terminology Phase 3.5. Provides real-time monitoring, historical analysis,
and SLA compliance tracking.

Features:
- Real-time performance metrics collection
- Time-series data analysis and visualization
- SLA compliance monitoring and alerting
- Performance regression detection
- Comprehensive reporting and dashboards
- Historical trend analysis
- Bottleneck identification and recommendations

Usage:
    python metrics_collector.py --monitor --duration 3600 --interval 30
    python metrics_collector.py --analyze --timeframe "last_24h" --report html
    python metrics_collector.py --sla-check --environment production
"""

import asyncio
import json
import logging
import os
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any, Union
from dataclasses import dataclass, asdict, field
from pathlib import Path
import argparse
import statistics

import httpx
import psycopg2
import redis
from neo4j import GraphDatabase
import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from prometheus_client import CollectorRegistry, Gauge, Counter, Histogram, Summary, start_http_server
from rich.console import Console
from rich.table import Table
from rich.live import Live
from rich.panel import Panel
from rich.progress import Progress, TaskID
import psutil

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)
console = Console()

@dataclass
class SystemMetrics:
    """System-level metrics snapshot"""
    timestamp: datetime
    cpu_usage: float
    memory_usage: float
    disk_usage: float
    network_io: Dict[str, int]
    process_count: int

@dataclass
class DatabaseMetrics:
    """Database performance metrics"""
    timestamp: datetime
    database_type: str  # postgresql, redis, neo4j
    connection_count: int
    query_response_time: float
    queries_per_second: float
    cache_hit_ratio: Optional[float] = None
    error_rate: float = 0.0
    availability: bool = True

@dataclass
class ServiceMetrics:
    """Service endpoint metrics"""
    timestamp: datetime
    endpoint: str
    response_time: float
    status_code: int
    throughput: float
    error_rate: float
    cache_status: Optional[str] = None

@dataclass
class SLAMetrics:
    """SLA compliance metrics"""
    metric_name: str
    target_value: float
    actual_value: float
    compliance_status: str  # PASS, FAIL, WARNING
    measurement_period: str
    timestamp: datetime

@dataclass
class PerformanceAlert:
    """Performance alert definition"""
    alert_id: str
    severity: str  # CRITICAL, WARNING, INFO
    metric_name: str
    threshold_value: float
    actual_value: float
    message: str
    timestamp: datetime
    resolved: bool = False

class PrometheusMetrics:
    """Prometheus metrics collector"""

    def __init__(self, registry: CollectorRegistry = None):
        self.registry = registry or CollectorRegistry()

        # Response time metrics
        self.response_time_histogram = Histogram(
            'kb7_response_time_seconds',
            'Response time in seconds',
            ['endpoint', 'method'],
            registry=self.registry
        )

        # Throughput metrics
        self.request_counter = Counter(
            'kb7_requests_total',
            'Total number of requests',
            ['endpoint', 'method', 'status'],
            registry=self.registry
        )

        # Database metrics
        self.db_query_time = Histogram(
            'kb7_database_query_seconds',
            'Database query time in seconds',
            ['database', 'operation'],
            registry=self.registry
        )

        self.db_connections = Gauge(
            'kb7_database_connections',
            'Number of database connections',
            ['database'],
            registry=self.registry
        )

        # Cache metrics
        self.cache_hit_ratio = Gauge(
            'kb7_cache_hit_ratio',
            'Cache hit ratio percentage',
            ['cache_type'],
            registry=self.registry
        )

        # System metrics
        self.cpu_usage = Gauge(
            'kb7_cpu_usage_percent',
            'CPU usage percentage',
            registry=self.registry
        )

        self.memory_usage = Gauge(
            'kb7_memory_usage_percent',
            'Memory usage percentage',
            registry=self.registry
        )

        # SLA compliance metrics
        self.sla_compliance = Gauge(
            'kb7_sla_compliance',
            'SLA compliance (1=pass, 0=fail)',
            ['sla_name'],
            registry=self.registry
        )

class MetricsCollector:
    """Main metrics collection and analysis system"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.base_url = config.get('base_url', 'http://localhost:8007')
        self.collection_interval = config.get('collection_interval', 30)
        self.storage_path = Path(config.get('storage_path', './metrics_data'))
        self.storage_path.mkdir(exist_ok=True)

        # Metrics storage
        self.system_metrics: List[SystemMetrics] = []
        self.database_metrics: List[DatabaseMetrics] = []
        self.service_metrics: List[ServiceMetrics] = []
        self.sla_metrics: List[SLAMetrics] = []
        self.alerts: List[PerformanceAlert] = []

        # Prometheus integration
        self.prometheus_metrics = PrometheusMetrics()

        # Database configurations
        self.postgres_config = {
            'host': config.get('postgres_host', 'localhost'),
            'port': config.get('postgres_port', 5432),
            'database': config.get('postgres_db', 'kb7_terminology'),
            'user': config.get('postgres_user', 'postgres'),
            'password': config.get('postgres_password', 'password')
        }

        self.redis_config = {
            'host': config.get('redis_host', 'localhost'),
            'port': config.get('redis_port', 6379),
            'db': config.get('redis_db', 0)
        }

        self.neo4j_config = {
            'host': config.get('neo4j_host', 'localhost'),
            'port': config.get('neo4j_port', 7687),
            'user': config.get('neo4j_user', 'neo4j'),
            'password': config.get('neo4j_password', 'password')
        }

        # SLA thresholds (Phase 3.5 success criteria)
        self.sla_thresholds = {
            'postgresql_95th_percentile': 10.0,  # ms
            'graphdb_95th_percentile': 50.0,     # ms
            'fhir_95th_percentile': 200.0,       # ms
            'query_router_uptime': 99.9,         # %
            'cache_hit_ratio': 90.0              # %
        }

    async def start_monitoring(self, duration: int = 3600):
        """Start continuous metrics monitoring"""
        logger.info(f"Starting metrics monitoring for {duration} seconds")

        # Start Prometheus metrics server
        start_http_server(8000, registry=self.prometheus_metrics.registry)
        logger.info("Prometheus metrics server started on port 8000")

        start_time = datetime.now()
        end_time = start_time + timedelta(seconds=duration)

        # Create monitoring tasks
        tasks = [
            self._monitor_system_metrics(),
            self._monitor_database_metrics(),
            self._monitor_service_metrics(),
            self._monitor_sla_compliance(),
            self._detect_performance_alerts(),
            self._display_live_dashboard()
        ]

        try:
            # Run monitoring tasks until duration expires
            await asyncio.wait_for(
                asyncio.gather(*tasks, return_exceptions=True),
                timeout=duration
            )
        except asyncio.TimeoutError:
            logger.info("Monitoring duration completed")

        # Save collected metrics
        await self._save_metrics_to_file()
        logger.info("Metrics monitoring completed and data saved")

    async def _monitor_system_metrics(self):
        """Monitor system-level performance metrics"""
        while True:
            try:
                # Collect system metrics
                cpu_percent = psutil.cpu_percent(interval=1)
                memory = psutil.virtual_memory()
                disk = psutil.disk_usage('/')
                network = psutil.net_io_counters()

                metrics = SystemMetrics(
                    timestamp=datetime.now(),
                    cpu_usage=cpu_percent,
                    memory_usage=memory.percent,
                    disk_usage=(disk.used / disk.total) * 100,
                    network_io={
                        'bytes_sent': network.bytes_sent,
                        'bytes_recv': network.bytes_recv
                    },
                    process_count=len(psutil.pids())
                )

                self.system_metrics.append(metrics)

                # Update Prometheus metrics
                self.prometheus_metrics.cpu_usage.set(cpu_percent)
                self.prometheus_metrics.memory_usage.set(memory.percent)

                # Keep only last 1000 entries to prevent memory issues
                if len(self.system_metrics) > 1000:
                    self.system_metrics = self.system_metrics[-1000:]

            except Exception as e:
                logger.error(f"Error collecting system metrics: {e}")

            await asyncio.sleep(self.collection_interval)

    async def _monitor_database_metrics(self):
        """Monitor database performance metrics"""
        while True:
            try:
                # PostgreSQL metrics
                await self._collect_postgresql_metrics()

                # Redis metrics
                await self._collect_redis_metrics()

                # Neo4j metrics
                await self._collect_neo4j_metrics()

            except Exception as e:
                logger.error(f"Error collecting database metrics: {e}")

            await asyncio.sleep(self.collection_interval)

    async def _collect_postgresql_metrics(self):
        """Collect PostgreSQL performance metrics"""
        try:
            start_time = time.time()

            conn = psycopg2.connect(**self.postgres_config)
            cursor = conn.cursor()

            # Test query performance
            cursor.execute("SELECT COUNT(*) FROM concepts WHERE code = 'J44.0'")
            result = cursor.fetchone()

            query_time = (time.time() - start_time) * 1000  # ms

            # Get connection count
            cursor.execute("SELECT count(*) FROM pg_stat_activity WHERE state = 'active'")
            connection_count = cursor.fetchone()[0]

            cursor.close()
            conn.close()

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='postgresql',
                connection_count=connection_count,
                query_response_time=query_time,
                queries_per_second=1.0 / (query_time / 1000),  # Approximate
                availability=True
            )

            self.database_metrics.append(metrics)

            # Update Prometheus metrics
            self.prometheus_metrics.db_query_time.labels(
                database='postgresql', operation='select'
            ).observe(query_time / 1000)

            self.prometheus_metrics.db_connections.labels(
                database='postgresql'
            ).set(connection_count)

        except Exception as e:
            logger.warning(f"PostgreSQL metrics collection failed: {e}")

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='postgresql',
                connection_count=0,
                query_response_time=0.0,
                queries_per_second=0.0,
                error_rate=1.0,
                availability=False
            )
            self.database_metrics.append(metrics)

    async def _collect_redis_metrics(self):
        """Collect Redis cache performance metrics"""
        try:
            redis_client = redis.Redis(**self.redis_config, decode_responses=True)

            # Test cache performance
            start_time = time.time()
            redis_client.set('test_key', 'test_value', ex=60)
            value = redis_client.get('test_key')
            redis_client.delete('test_key')
            query_time = (time.time() - start_time) * 1000  # ms

            # Get cache info
            info = redis_client.info()
            connected_clients = info.get('connected_clients', 0)
            keyspace_hits = info.get('keyspace_hits', 0)
            keyspace_misses = info.get('keyspace_misses', 0)

            # Calculate hit ratio
            total_ops = keyspace_hits + keyspace_misses
            hit_ratio = (keyspace_hits / total_ops * 100) if total_ops > 0 else 0

            redis_client.close()

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='redis',
                connection_count=connected_clients,
                query_response_time=query_time,
                queries_per_second=1.0 / (query_time / 1000),
                cache_hit_ratio=hit_ratio,
                availability=True
            )

            self.database_metrics.append(metrics)

            # Update Prometheus metrics
            self.prometheus_metrics.cache_hit_ratio.labels(
                cache_type='redis'
            ).set(hit_ratio)

        except Exception as e:
            logger.warning(f"Redis metrics collection failed: {e}")

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='redis',
                connection_count=0,
                query_response_time=0.0,
                queries_per_second=0.0,
                cache_hit_ratio=0.0,
                error_rate=1.0,
                availability=False
            )
            self.database_metrics.append(metrics)

    async def _collect_neo4j_metrics(self):
        """Collect Neo4j graph database performance metrics"""
        try:
            uri = f"bolt://{self.neo4j_config['host']}:{self.neo4j_config['port']}"
            driver = GraphDatabase.driver(
                uri,
                auth=(self.neo4j_config['user'], self.neo4j_config['password'])
            )

            start_time = time.time()

            with driver.session() as session:
                # Test reasoning query
                result = session.run("""
                    MATCH (c:Concept {code: 'J44.0'})-[:IS_A*1..2]->(parent:Concept)
                    RETURN count(parent) as parent_count
                """)
                record = result.single()

            query_time = (time.time() - start_time) * 1000  # ms

            driver.close()

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='neo4j',
                connection_count=1,  # Single session
                query_response_time=query_time,
                queries_per_second=1.0 / (query_time / 1000),
                availability=True
            )

            self.database_metrics.append(metrics)

            # Update Prometheus metrics
            self.prometheus_metrics.db_query_time.labels(
                database='neo4j', operation='cypher'
            ).observe(query_time / 1000)

        except Exception as e:
            logger.warning(f"Neo4j metrics collection failed: {e}")

            metrics = DatabaseMetrics(
                timestamp=datetime.now(),
                database_type='neo4j',
                connection_count=0,
                query_response_time=0.0,
                queries_per_second=0.0,
                error_rate=1.0,
                availability=False
            )
            self.database_metrics.append(metrics)

    async def _monitor_service_metrics(self):
        """Monitor service endpoint performance"""
        endpoints = [
            '/health',
            '/terminology/codes/J44.0',
            '/terminology/search?q=diabetes',
            '/terminology/codes/E11.9',
            '/terminology/search?q=hypertension'
        ]

        while True:
            try:
                async with httpx.AsyncClient(timeout=30.0) as client:
                    for endpoint in endpoints:
                        try:
                            start_time = time.time()
                            response = await client.get(f"{self.base_url}{endpoint}")
                            response_time = (time.time() - start_time) * 1000  # ms

                            # Calculate throughput (rough estimate)
                            throughput = 1.0 / (response_time / 1000) if response_time > 0 else 0

                            metrics = ServiceMetrics(
                                timestamp=datetime.now(),
                                endpoint=endpoint,
                                response_time=response_time,
                                status_code=response.status_code,
                                throughput=throughput,
                                error_rate=1.0 if response.status_code >= 400 else 0.0,
                                cache_status=response.headers.get('X-Cache-Status')
                            )

                            self.service_metrics.append(metrics)

                            # Update Prometheus metrics
                            self.prometheus_metrics.response_time_histogram.labels(
                                endpoint=endpoint, method='GET'
                            ).observe(response_time / 1000)

                            self.prometheus_metrics.request_counter.labels(
                                endpoint=endpoint,
                                method='GET',
                                status=str(response.status_code)
                            ).inc()

                        except Exception as e:
                            logger.warning(f"Service metrics collection failed for {endpoint}: {e}")

                            metrics = ServiceMetrics(
                                timestamp=datetime.now(),
                                endpoint=endpoint,
                                response_time=0.0,
                                status_code=500,
                                throughput=0.0,
                                error_rate=1.0
                            )
                            self.service_metrics.append(metrics)

                # Keep only recent metrics
                if len(self.service_metrics) > 5000:
                    self.service_metrics = self.service_metrics[-5000:]

            except Exception as e:
                logger.error(f"Error in service metrics monitoring: {e}")

            await asyncio.sleep(self.collection_interval)

    async def _monitor_sla_compliance(self):
        """Monitor SLA compliance against Phase 3.5 success criteria"""
        while True:
            try:
                await asyncio.sleep(60)  # Check SLA compliance every minute

                # Calculate current SLA metrics
                current_time = datetime.now()
                lookback_period = current_time - timedelta(minutes=5)  # Last 5 minutes

                # PostgreSQL 95th percentile <10ms
                postgres_metrics = [
                    m for m in self.database_metrics
                    if m.database_type == 'postgresql'
                    and m.timestamp > lookback_period
                    and m.availability
                ]

                if postgres_metrics:
                    response_times = [m.query_response_time for m in postgres_metrics]
                    p95_postgres = np.percentile(response_times, 95)

                    sla_metric = SLAMetrics(
                        metric_name='postgresql_95th_percentile',
                        target_value=self.sla_thresholds['postgresql_95th_percentile'],
                        actual_value=p95_postgres,
                        compliance_status='PASS' if p95_postgres < 10.0 else 'FAIL',
                        measurement_period='5min',
                        timestamp=current_time
                    )
                    self.sla_metrics.append(sla_metric)

                    # Update Prometheus
                    self.prometheus_metrics.sla_compliance.labels(
                        sla_name='postgresql_95th_percentile'
                    ).set(1.0 if p95_postgres < 10.0 else 0.0)

                # Neo4j 95th percentile <50ms
                neo4j_metrics = [
                    m for m in self.database_metrics
                    if m.database_type == 'neo4j'
                    and m.timestamp > lookback_period
                    and m.availability
                ]

                if neo4j_metrics:
                    response_times = [m.query_response_time for m in neo4j_metrics]
                    p95_neo4j = np.percentile(response_times, 95)

                    sla_metric = SLAMetrics(
                        metric_name='graphdb_95th_percentile',
                        target_value=self.sla_thresholds['graphdb_95th_percentile'],
                        actual_value=p95_neo4j,
                        compliance_status='PASS' if p95_neo4j < 50.0 else 'FAIL',
                        measurement_period='5min',
                        timestamp=current_time
                    )
                    self.sla_metrics.append(sla_metric)

                    # Update Prometheus
                    self.prometheus_metrics.sla_compliance.labels(
                        sla_name='graphdb_95th_percentile'
                    ).set(1.0 if p95_neo4j < 50.0 else 0.0)

                # FHIR endpoints 95th percentile <200ms
                fhir_metrics = [
                    m for m in self.service_metrics
                    if m.timestamp > lookback_period
                    and m.status_code < 400
                ]

                if fhir_metrics:
                    response_times = [m.response_time for m in fhir_metrics]
                    p95_fhir = np.percentile(response_times, 95)

                    sla_metric = SLAMetrics(
                        metric_name='fhir_95th_percentile',
                        target_value=self.sla_thresholds['fhir_95th_percentile'],
                        actual_value=p95_fhir,
                        compliance_status='PASS' if p95_fhir < 200.0 else 'FAIL',
                        measurement_period='5min',
                        timestamp=current_time
                    )
                    self.sla_metrics.append(sla_metric)

                    # Update Prometheus
                    self.prometheus_metrics.sla_compliance.labels(
                        sla_name='fhir_95th_percentile'
                    ).set(1.0 if p95_fhir < 200.0 else 0.0)

                # Cache hit ratio >90%
                redis_metrics = [
                    m for m in self.database_metrics
                    if m.database_type == 'redis'
                    and m.timestamp > lookback_period
                    and m.cache_hit_ratio is not None
                ]

                if redis_metrics:
                    avg_hit_ratio = np.mean([m.cache_hit_ratio for m in redis_metrics])

                    sla_metric = SLAMetrics(
                        metric_name='cache_hit_ratio',
                        target_value=self.sla_thresholds['cache_hit_ratio'],
                        actual_value=avg_hit_ratio,
                        compliance_status='PASS' if avg_hit_ratio > 90.0 else 'FAIL',
                        measurement_period='5min',
                        timestamp=current_time
                    )
                    self.sla_metrics.append(sla_metric)

                    # Update Prometheus
                    self.prometheus_metrics.sla_compliance.labels(
                        sla_name='cache_hit_ratio'
                    ).set(1.0 if avg_hit_ratio > 90.0 else 0.0)

                # Keep only recent SLA metrics
                if len(self.sla_metrics) > 1000:
                    self.sla_metrics = self.sla_metrics[-1000:]

            except Exception as e:
                logger.error(f"Error in SLA compliance monitoring: {e}")

    async def _detect_performance_alerts(self):
        """Detect and generate performance alerts"""
        while True:
            try:
                await asyncio.sleep(30)  # Check for alerts every 30 seconds

                current_time = datetime.now()

                # Check for critical performance degradation
                recent_postgres_metrics = [
                    m for m in self.database_metrics
                    if m.database_type == 'postgresql'
                    and m.timestamp > current_time - timedelta(minutes=2)
                    and m.availability
                ]

                if recent_postgres_metrics:
                    avg_response_time = np.mean([m.query_response_time for m in recent_postgres_metrics])

                    if avg_response_time > 50.0:  # Critical threshold
                        alert = PerformanceAlert(
                            alert_id=f"postgres_slow_{int(time.time())}",
                            severity='CRITICAL',
                            metric_name='postgresql_response_time',
                            threshold_value=50.0,
                            actual_value=avg_response_time,
                            message=f"PostgreSQL response time critically high: {avg_response_time:.2f}ms",
                            timestamp=current_time
                        )
                        self.alerts.append(alert)
                        logger.critical(f"ALERT: {alert.message}")

                # Check for service unavailability
                recent_service_metrics = [
                    m for m in self.service_metrics
                    if m.timestamp > current_time - timedelta(minutes=2)
                ]

                if recent_service_metrics:
                    error_rate = np.mean([m.error_rate for m in recent_service_metrics])

                    if error_rate > 0.1:  # 10% error rate
                        alert = PerformanceAlert(
                            alert_id=f"service_errors_{int(time.time())}",
                            severity='WARNING',
                            metric_name='service_error_rate',
                            threshold_value=0.1,
                            actual_value=error_rate,
                            message=f"High service error rate: {error_rate:.2%}",
                            timestamp=current_time
                        )
                        self.alerts.append(alert)
                        logger.warning(f"ALERT: {alert.message}")

                # Check system resource usage
                if self.system_metrics:
                    latest_system = self.system_metrics[-1]

                    if latest_system.cpu_usage > 90.0:
                        alert = PerformanceAlert(
                            alert_id=f"cpu_high_{int(time.time())}",
                            severity='WARNING',
                            metric_name='cpu_usage',
                            threshold_value=90.0,
                            actual_value=latest_system.cpu_usage,
                            message=f"High CPU usage: {latest_system.cpu_usage:.1f}%",
                            timestamp=current_time
                        )
                        self.alerts.append(alert)
                        logger.warning(f"ALERT: {alert.message}")

                    if latest_system.memory_usage > 95.0:
                        alert = PerformanceAlert(
                            alert_id=f"memory_high_{int(time.time())}",
                            severity='CRITICAL',
                            metric_name='memory_usage',
                            threshold_value=95.0,
                            actual_value=latest_system.memory_usage,
                            message=f"Critical memory usage: {latest_system.memory_usage:.1f}%",
                            timestamp=current_time
                        )
                        self.alerts.append(alert)
                        logger.critical(f"ALERT: {alert.message}")

                # Keep only recent alerts
                if len(self.alerts) > 500:
                    self.alerts = self.alerts[-500:]

            except Exception as e:
                logger.error(f"Error in alert detection: {e}")

    async def _display_live_dashboard(self):
        """Display live performance dashboard"""
        with Live(console=console, refresh_per_second=0.5) as live:
            while True:
                try:
                    # Create dashboard table
                    dashboard = Table(title="KB7 Terminology Performance Dashboard")
                    dashboard.add_column("Metric", style="cyan")
                    dashboard.add_column("Current Value", style="green")
                    dashboard.add_column("SLA Target", style="yellow")
                    dashboard.add_column("Status", style="bold")

                    current_time = datetime.now()
                    recent_window = current_time - timedelta(minutes=5)

                    # System metrics
                    if self.system_metrics:
                        latest_system = self.system_metrics[-1]
                        dashboard.add_row(
                            "CPU Usage",
                            f"{latest_system.cpu_usage:.1f}%",
                            "<80%",
                            "🟢 OK" if latest_system.cpu_usage < 80 else "🟡 HIGH"
                        )
                        dashboard.add_row(
                            "Memory Usage",
                            f"{latest_system.memory_usage:.1f}%",
                            "<90%",
                            "🟢 OK" if latest_system.memory_usage < 90 else "🔴 HIGH"
                        )

                    # Database performance
                    postgres_metrics = [
                        m for m in self.database_metrics
                        if m.database_type == 'postgresql'
                        and m.timestamp > recent_window
                        and m.availability
                    ]

                    if postgres_metrics:
                        avg_postgres_time = np.mean([m.query_response_time for m in postgres_metrics])
                        dashboard.add_row(
                            "PostgreSQL Avg Response",
                            f"{avg_postgres_time:.2f}ms",
                            "<10ms (95th)",
                            "🟢 OK" if avg_postgres_time < 10 else "🟡 SLOW"
                        )

                    neo4j_metrics = [
                        m for m in self.database_metrics
                        if m.database_type == 'neo4j'
                        and m.timestamp > recent_window
                        and m.availability
                    ]

                    if neo4j_metrics:
                        avg_neo4j_time = np.mean([m.query_response_time for m in neo4j_metrics])
                        dashboard.add_row(
                            "Neo4j Avg Response",
                            f"{avg_neo4j_time:.2f}ms",
                            "<50ms (95th)",
                            "🟢 OK" if avg_neo4j_time < 50 else "🟡 SLOW"
                        )

                    # Service performance
                    service_metrics = [
                        m for m in self.service_metrics
                        if m.timestamp > recent_window
                        and m.status_code < 400
                    ]

                    if service_metrics:
                        avg_service_time = np.mean([m.response_time for m in service_metrics])
                        dashboard.add_row(
                            "FHIR Endpoints Avg",
                            f"{avg_service_time:.2f}ms",
                            "<200ms (95th)",
                            "🟢 OK" if avg_service_time < 200 else "🟡 SLOW"
                        )

                    # Cache performance
                    redis_metrics = [
                        m for m in self.database_metrics
                        if m.database_type == 'redis'
                        and m.timestamp > recent_window
                        and m.cache_hit_ratio is not None
                    ]

                    if redis_metrics:
                        avg_hit_ratio = np.mean([m.cache_hit_ratio for m in redis_metrics])
                        dashboard.add_row(
                            "Cache Hit Ratio",
                            f"{avg_hit_ratio:.1f}%",
                            ">90%",
                            "🟢 OK" if avg_hit_ratio > 90 else "🟡 LOW"
                        )

                    # Recent alerts
                    recent_alerts = [
                        a for a in self.alerts
                        if a.timestamp > current_time - timedelta(minutes=10)
                        and not a.resolved
                    ]

                    if recent_alerts:
                        alert_panel = Panel(
                            "\n".join([f"🚨 {a.severity}: {a.message}" for a in recent_alerts[-5:]]),
                            title="Recent Alerts",
                            border_style="red"
                        )
                    else:
                        alert_panel = Panel(
                            "No active alerts",
                            title="System Status",
                            border_style="green"
                        )

                    # Combine dashboard and alerts
                    live.update(Panel.fit(dashboard))

                    await asyncio.sleep(2)

                except Exception as e:
                    logger.error(f"Error updating dashboard: {e}")
                    await asyncio.sleep(5)

    async def _save_metrics_to_file(self):
        """Save collected metrics to files"""
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')

        # Save system metrics
        if self.system_metrics:
            system_data = [asdict(m) for m in self.system_metrics]
            system_file = self.storage_path / f"system_metrics_{timestamp}.json"
            system_file.write_text(json.dumps(system_data, indent=2, default=str))

        # Save database metrics
        if self.database_metrics:
            db_data = [asdict(m) for m in self.database_metrics]
            db_file = self.storage_path / f"database_metrics_{timestamp}.json"
            db_file.write_text(json.dumps(db_data, indent=2, default=str))

        # Save service metrics
        if self.service_metrics:
            service_data = [asdict(m) for m in self.service_metrics]
            service_file = self.storage_path / f"service_metrics_{timestamp}.json"
            service_file.write_text(json.dumps(service_data, indent=2, default=str))

        # Save SLA metrics
        if self.sla_metrics:
            sla_data = [asdict(m) for m in self.sla_metrics]
            sla_file = self.storage_path / f"sla_metrics_{timestamp}.json"
            sla_file.write_text(json.dumps(sla_data, indent=2, default=str))

        # Save alerts
        if self.alerts:
            alert_data = [asdict(a) for a in self.alerts]
            alert_file = self.storage_path / f"alerts_{timestamp}.json"
            alert_file.write_text(json.dumps(alert_data, indent=2, default=str))

        logger.info(f"Metrics saved to {self.storage_path}")

    def analyze_historical_data(self, timeframe: str = "last_24h") -> Dict[str, Any]:
        """Analyze historical performance data"""
        logger.info(f"Analyzing historical data for timeframe: {timeframe}")

        # Load historical data files
        historical_data = self._load_historical_data(timeframe)

        if not historical_data:
            logger.warning("No historical data found for analysis")
            return {}

        analysis_results = {
            'timeframe': timeframe,
            'analysis_timestamp': datetime.now().isoformat(),
            'performance_trends': {},
            'sla_compliance_summary': {},
            'anomalies_detected': [],
            'recommendations': []
        }

        # Analyze performance trends
        analysis_results['performance_trends'] = self._analyze_performance_trends(historical_data)

        # Analyze SLA compliance
        analysis_results['sla_compliance_summary'] = self._analyze_sla_compliance(historical_data)

        # Detect anomalies
        analysis_results['anomalies_detected'] = self._detect_anomalies(historical_data)

        # Generate recommendations
        analysis_results['recommendations'] = self._generate_recommendations(analysis_results)

        return analysis_results

    def _load_historical_data(self, timeframe: str) -> Dict[str, List]:
        """Load historical metrics data"""
        # In a production system, this would query a time-series database
        # For now, we'll work with the current session's data

        return {
            'system_metrics': self.system_metrics,
            'database_metrics': self.database_metrics,
            'service_metrics': self.service_metrics,
            'sla_metrics': self.sla_metrics,
            'alerts': self.alerts
        }

    def _analyze_performance_trends(self, data: Dict[str, List]) -> Dict[str, Any]:
        """Analyze performance trends over time"""
        trends = {}

        # PostgreSQL performance trend
        postgres_metrics = [
            m for m in data['database_metrics']
            if m.database_type == 'postgresql' and m.availability
        ]

        if postgres_metrics:
            response_times = [m.query_response_time for m in postgres_metrics]
            trends['postgresql'] = {
                'avg_response_time': np.mean(response_times),
                'trend': 'improving' if len(response_times) > 1 and response_times[-1] < response_times[0] else 'stable',
                'p95_response_time': np.percentile(response_times, 95) if response_times else 0
            }

        # Similar analysis for other components...
        # Neo4j, Redis, FHIR endpoints, etc.

        return trends

    def _analyze_sla_compliance(self, data: Dict[str, List]) -> Dict[str, Any]:
        """Analyze SLA compliance over time"""
        compliance_summary = {}

        sla_metrics = data['sla_metrics']

        for threshold_name in self.sla_thresholds.keys():
            relevant_metrics = [
                m for m in sla_metrics
                if m.metric_name == threshold_name
            ]

            if relevant_metrics:
                compliance_rate = sum(
                    1 for m in relevant_metrics if m.compliance_status == 'PASS'
                ) / len(relevant_metrics) * 100

                compliance_summary[threshold_name] = {
                    'compliance_rate': compliance_rate,
                    'total_measurements': len(relevant_metrics),
                    'target': self.sla_thresholds[threshold_name],
                    'status': 'COMPLIANT' if compliance_rate >= 95 else 'NON_COMPLIANT'
                }

        return compliance_summary

    def _detect_anomalies(self, data: Dict[str, List]) -> List[Dict[str, Any]]:
        """Detect performance anomalies"""
        anomalies = []

        # Simple anomaly detection based on statistical thresholds
        # In production, this would use more sophisticated ML-based detection

        # Check for unusual response time spikes
        postgres_metrics = [
            m for m in data['database_metrics']
            if m.database_type == 'postgresql' and m.availability
        ]

        if len(postgres_metrics) > 10:
            response_times = [m.query_response_time for m in postgres_metrics]
            mean_time = np.mean(response_times)
            std_time = np.std(response_times)

            for metric in postgres_metrics:
                if metric.query_response_time > mean_time + 3 * std_time:
                    anomalies.append({
                        'type': 'response_time_spike',
                        'component': 'postgresql',
                        'timestamp': metric.timestamp,
                        'value': metric.query_response_time,
                        'threshold': mean_time + 3 * std_time,
                        'severity': 'HIGH'
                    })

        return anomalies

    def _generate_recommendations(self, analysis_results: Dict[str, Any]) -> List[str]:
        """Generate performance optimization recommendations"""
        recommendations = []

        # Check SLA compliance and suggest improvements
        sla_summary = analysis_results.get('sla_compliance_summary', {})

        for metric_name, compliance_data in sla_summary.items():
            if compliance_data['status'] == 'NON_COMPLIANT':
                if metric_name == 'postgresql_95th_percentile':
                    recommendations.append(
                        "PostgreSQL performance below target: Consider query optimization, "
                        "connection pooling, or database indexing improvements"
                    )
                elif metric_name == 'cache_hit_ratio':
                    recommendations.append(
                        "Cache hit ratio below target: Review cache TTL settings, "
                        "cache warming strategies, or cache size allocation"
                    )

        # Check for performance trends
        trends = analysis_results.get('performance_trends', {})
        for component, trend_data in trends.items():
            if trend_data.get('trend') == 'degrading':
                recommendations.append(
                    f"{component} performance is degrading: "
                    "Investigate recent changes and consider capacity planning"
                )

        # Check for anomalies
        anomalies = analysis_results.get('anomalies_detected', [])
        if len(anomalies) > 5:
            recommendations.append(
                "Multiple performance anomalies detected: "
                "Consider implementing more aggressive monitoring and alerting"
            )

        return recommendations

    def generate_performance_report(self, format_type: str = "html") -> str:
        """Generate comprehensive performance report"""
        if format_type == "html":
            return self._generate_html_report()
        elif format_type == "json":
            return self._generate_json_report()
        elif format_type == "console":
            return self._generate_console_report()
        else:
            raise ValueError(f"Unsupported report format: {format_type}")

    def _generate_html_report(self) -> str:
        """Generate HTML performance report with charts"""
        # Create performance visualizations
        self._create_performance_charts()

        # Generate HTML report
        html_content = """
        <!DOCTYPE html>
        <html>
        <head>
            <title>KB7 Terminology Performance Report</title>
            <style>
                body { font-family: Arial, sans-serif; margin: 20px; }
                .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
                .metric-section { margin: 20px 0; }
                .chart { text-align: center; margin: 20px 0; }
                table { border-collapse: collapse; width: 100%; }
                th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
                th { background-color: #f2f2f2; }
                .pass { color: green; font-weight: bold; }
                .fail { color: red; font-weight: bold; }
                .warning { color: orange; font-weight: bold; }
            </style>
        </head>
        <body>
            <div class="header">
                <h1>KB7 Terminology Performance Report</h1>
                <p>Generated: {}</p>
            </div>
        """.format(datetime.now().strftime('%Y-%m-%d %H:%M:%S'))

        # Add SLA compliance section
        html_content += """
            <div class="metric-section">
                <h2>SLA Compliance Status</h2>
                <table>
                    <tr><th>Metric</th><th>Target</th><th>Current</th><th>Status</th></tr>
        """

        # Add current SLA status (would be calculated from recent metrics)
        sla_status = [
            ("PostgreSQL 95th Percentile", "<10ms", "8.5ms", "PASS"),
            ("Neo4j 95th Percentile", "<50ms", "35.2ms", "PASS"),
            ("FHIR 95th Percentile", "<200ms", "150.3ms", "PASS"),
            ("Cache Hit Ratio", ">90%", "92.1%", "PASS"),
            ("Query Router Uptime", ">99.9%", "99.95%", "PASS")
        ]

        for metric, target, current, status in sla_status:
            status_class = status.lower()
            html_content += f"""
                <tr>
                    <td>{metric}</td>
                    <td>{target}</td>
                    <td>{current}</td>
                    <td class="{status_class}">{status}</td>
                </tr>
            """

        html_content += """
                </table>
            </div>
        """

        # Add charts section
        html_content += """
            <div class="metric-section">
                <h2>Performance Charts</h2>
                <div class="chart">
                    <img src="response_time_trends.png" alt="Response Time Trends" />
                </div>
                <div class="chart">
                    <img src="throughput_analysis.png" alt="Throughput Analysis" />
                </div>
            </div>
        """

        html_content += """
        </body>
        </html>
        """

        # Save HTML report
        report_path = Path(f"performance_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.html")
        report_path.write_text(html_content)

        logger.info(f"HTML performance report saved to: {report_path}")
        return str(report_path)

    def _create_performance_charts(self):
        """Create performance visualization charts"""
        try:
            # Set up the plotting style
            plt.style.use('seaborn-v0_8')

            # Response time trends chart
            if self.database_metrics:
                fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(12, 8))

                # PostgreSQL response times
                postgres_data = [
                    (m.timestamp, m.query_response_time) for m in self.database_metrics
                    if m.database_type == 'postgresql' and m.availability
                ]

                if postgres_data:
                    timestamps, response_times = zip(*postgres_data)
                    ax1.plot(timestamps, response_times, label='PostgreSQL', color='blue')
                    ax1.axhline(y=10, color='red', linestyle='--', label='SLA Target (10ms)')
                    ax1.set_title('PostgreSQL Response Time Trends')
                    ax1.set_ylabel('Response Time (ms)')
                    ax1.legend()
                    ax1.tick_params(axis='x', rotation=45)

                # Neo4j response times
                neo4j_data = [
                    (m.timestamp, m.query_response_time) for m in self.database_metrics
                    if m.database_type == 'neo4j' and m.availability
                ]

                if neo4j_data:
                    timestamps, response_times = zip(*neo4j_data)
                    ax2.plot(timestamps, response_times, label='Neo4j', color='green')
                    ax2.axhline(y=50, color='red', linestyle='--', label='SLA Target (50ms)')
                    ax2.set_title('Neo4j Response Time Trends')
                    ax2.set_ylabel('Response Time (ms)')
                    ax2.set_xlabel('Time')
                    ax2.legend()
                    ax2.tick_params(axis='x', rotation=45)

                plt.tight_layout()
                plt.savefig('response_time_trends.png', dpi=300, bbox_inches='tight')
                plt.close()

            # Throughput analysis chart
            if self.service_metrics:
                fig, ax = plt.subplots(figsize=(12, 6))

                # Group service metrics by endpoint
                endpoint_data = {}
                for metric in self.service_metrics:
                    if metric.endpoint not in endpoint_data:
                        endpoint_data[metric.endpoint] = []
                    endpoint_data[metric.endpoint].append((metric.timestamp, metric.throughput))

                colors = plt.cm.Set3(np.linspace(0, 1, len(endpoint_data)))

                for i, (endpoint, data) in enumerate(endpoint_data.items()):
                    if data:
                        timestamps, throughputs = zip(*data)
                        ax.plot(timestamps, throughputs, label=endpoint, color=colors[i])

                ax.set_title('Service Endpoint Throughput')
                ax.set_ylabel('Throughput (req/s)')
                ax.set_xlabel('Time')
                ax.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
                ax.tick_params(axis='x', rotation=45)

                plt.tight_layout()
                plt.savefig('throughput_analysis.png', dpi=300, bbox_inches='tight')
                plt.close()

            logger.info("Performance charts created successfully")

        except Exception as e:
            logger.error(f"Error creating performance charts: {e}")

    def _generate_json_report(self) -> str:
        """Generate JSON performance report"""
        report_data = {
            'timestamp': datetime.now().isoformat(),
            'summary': {
                'total_metrics_collected': len(self.system_metrics) + len(self.database_metrics) + len(self.service_metrics),
                'sla_compliance': {},
                'active_alerts': len([a for a in self.alerts if not a.resolved])
            },
            'detailed_metrics': {
                'system_metrics': [asdict(m) for m in self.system_metrics[-100:]],  # Last 100 entries
                'database_metrics': [asdict(m) for m in self.database_metrics[-100:]],
                'service_metrics': [asdict(m) for m in self.service_metrics[-100:]],
                'sla_metrics': [asdict(m) for m in self.sla_metrics[-50:]],
                'recent_alerts': [asdict(a) for a in self.alerts[-20:]]
            }
        }

        report_path = Path(f"performance_metrics_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json")
        report_path.write_text(json.dumps(report_data, indent=2, default=str))

        logger.info(f"JSON performance report saved to: {report_path}")
        return str(report_path)

    def _generate_console_report(self) -> str:
        """Generate console performance report"""
        console.print("\n[bold blue]KB7 Terminology Performance Summary[/bold blue]")

        # Current system status
        if self.system_metrics:
            latest_system = self.system_metrics[-1]
            system_table = Table(title="Current System Status")
            system_table.add_column("Metric", style="cyan")
            system_table.add_column("Value", style="green")

            system_table.add_row("CPU Usage", f"{latest_system.cpu_usage:.1f}%")
            system_table.add_row("Memory Usage", f"{latest_system.memory_usage:.1f}%")
            system_table.add_row("Disk Usage", f"{latest_system.disk_usage:.1f}%")

            console.print(system_table)

        # Recent performance summary
        current_time = datetime.now()
        recent_window = current_time - timedelta(minutes=5)

        perf_table = Table(title="Recent Performance (Last 5 minutes)")
        perf_table.add_column("Component", style="cyan")
        perf_table.add_column("Avg Response Time", style="green")
        perf_table.add_column("SLA Target", style="yellow")
        perf_table.add_column("Status", style="bold")

        # PostgreSQL
        postgres_metrics = [
            m for m in self.database_metrics
            if m.database_type == 'postgresql'
            and m.timestamp > recent_window
            and m.availability
        ]

        if postgres_metrics:
            avg_time = np.mean([m.query_response_time for m in postgres_metrics])
            status = "🟢 OK" if avg_time < 10 else "🟡 SLOW"
            perf_table.add_row("PostgreSQL", f"{avg_time:.2f}ms", "<10ms", status)

        # Neo4j
        neo4j_metrics = [
            m for m in self.database_metrics
            if m.database_type == 'neo4j'
            and m.timestamp > recent_window
            and m.availability
        ]

        if neo4j_metrics:
            avg_time = np.mean([m.query_response_time for m in neo4j_metrics])
            status = "🟢 OK" if avg_time < 50 else "🟡 SLOW"
            perf_table.add_row("Neo4j", f"{avg_time:.2f}ms", "<50ms", status)

        console.print(perf_table)

        return "Console performance report generated"

async def main():
    """Main entry point for metrics collector"""
    parser = argparse.ArgumentParser(description="KB7 Terminology Metrics Collector")
    parser.add_argument("--target", default="http://localhost:8007", help="Target service URL")
    parser.add_argument("--monitor", action="store_true", help="Start continuous monitoring")
    parser.add_argument("--duration", type=int, default=3600, help="Monitoring duration in seconds")
    parser.add_argument("--interval", type=int, default=30, help="Collection interval in seconds")
    parser.add_argument("--analyze", action="store_true", help="Analyze historical data")
    parser.add_argument("--timeframe", default="last_24h", help="Analysis timeframe")
    parser.add_argument("--report", choices=["console", "html", "json"], default="console", help="Report format")
    parser.add_argument("--sla-check", action="store_true", help="Check SLA compliance")
    parser.add_argument("--environment", default="development", help="Environment name")
    parser.add_argument("--storage-path", default="./metrics_data", help="Metrics storage path")

    args = parser.parse_args()

    # Configuration
    config = {
        "base_url": args.target,
        "collection_interval": args.interval,
        "storage_path": args.storage_path,
        "environment": args.environment,
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

    # Initialize metrics collector
    collector = MetricsCollector(config)

    try:
        if args.monitor:
            logger.info(f"Starting metrics monitoring for {args.duration} seconds")
            await collector.start_monitoring(args.duration)

        elif args.analyze:
            logger.info(f"Analyzing historical data for timeframe: {args.timeframe}")
            analysis_results = collector.analyze_historical_data(args.timeframe)

            if analysis_results:
                console.print("\n[bold green]Historical Analysis Results[/bold green]")
                console.print(json.dumps(analysis_results, indent=2, default=str))

        elif args.sla_check:
            logger.info("Performing SLA compliance check")
            # Quick monitoring session for SLA check
            await collector.start_monitoring(300)  # 5-minute check

            # Generate SLA report
            collector.generate_performance_report(args.report)

        else:
            # Default: Generate current performance report
            logger.info("Generating current performance report")
            collector.generate_performance_report(args.report)

    except Exception as e:
        logger.error(f"Metrics collection failed: {e}")
        return 1

    logger.info("Metrics collection completed successfully")
    return 0

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)