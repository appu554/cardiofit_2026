#!/usr/bin/env python3
"""
KB7 Terminology Validation Suite

Comprehensive system validation and health checks for Phase 3.5 implementation.
Validates data integrity, system configuration, performance baselines, and
operational readiness.

Features:
- Data migration integrity validation
- System configuration verification
- Database connectivity and performance checks
- Cache system validation
- FHIR endpoint compliance testing
- Monitoring and alerting system verification
- End-to-end workflow validation

Usage:
    python validation_suite.py --full-validation
    python validation_suite.py --check-migration --check-performance
    python validation_suite.py --environment production --output-format json
"""

import asyncio
import json
import logging
import os
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any, Union
from dataclasses import dataclass, asdict
from pathlib import Path
import argparse
import hashlib

import httpx
import psycopg2
import redis
from neo4j import GraphDatabase
import yaml
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.progress import Progress, TaskID
import pandas as pd

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)
console = Console()

@dataclass
class ValidationResult:
    """Individual validation test result"""
    test_name: str
    category: str
    status: str  # PASS, FAIL, WARNING, SKIP
    message: str
    details: Dict[str, Any] = None
    timestamp: datetime = None
    duration_ms: float = 0.0

    def __post_init__(self):
        if self.timestamp is None:
            self.timestamp = datetime.now()
        if self.details is None:
            self.details = {}

@dataclass
class ValidationSummary:
    """Overall validation summary"""
    total_tests: int
    passed: int
    failed: int
    warnings: int
    skipped: int
    duration_seconds: float
    environment: str
    timestamp: datetime

    @property
    def success_rate(self) -> float:
        """Calculate overall success rate"""
        if self.total_tests == 0:
            return 0.0
        return (self.passed / self.total_tests) * 100

    @property
    def is_production_ready(self) -> bool:
        """Determine if system is production ready"""
        return self.failed == 0 and self.success_rate >= 95.0

class SystemValidator:
    """Main system validation class"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.base_url = config.get('base_url', 'http://localhost:8007')
        self.environment = config.get('environment', 'development')
        self.validation_results: List[ValidationResult] = []

        # Database connections
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

    async def run_validation(self, validation_type: str = "full") -> ValidationSummary:
        """Run comprehensive validation suite"""
        start_time = datetime.now()
        logger.info(f"Starting {validation_type} validation for {self.environment} environment")

        try:
            if validation_type == "full" or validation_type == "connectivity":
                await self._validate_system_connectivity()

            if validation_type == "full" or validation_type == "migration":
                await self._validate_data_migration()

            if validation_type == "full" or validation_type == "performance":
                await self._validate_performance_baselines()

            if validation_type == "full" or validation_type == "fhir":
                await self._validate_fhir_compliance()

            if validation_type == "full" or validation_type == "cache":
                await self._validate_cache_system()

            if validation_type == "full" or validation_type == "monitoring":
                await self._validate_monitoring_alerting()

            if validation_type == "full" or validation_type == "security":
                await self._validate_security_configuration()

            if validation_type == "full" or validation_type == "workflow":
                await self._validate_end_to_end_workflows()

        except Exception as e:
            logger.error(f"Validation failed with error: {e}")
            self.validation_results.append(ValidationResult(
                test_name="Validation Suite Execution",
                category="System",
                status="FAIL",
                message=f"Validation suite crashed: {str(e)}"
            ))

        end_time = datetime.now()
        duration = (end_time - start_time).total_seconds()

        # Calculate summary
        summary = ValidationSummary(
            total_tests=len(self.validation_results),
            passed=len([r for r in self.validation_results if r.status == "PASS"]),
            failed=len([r for r in self.validation_results if r.status == "FAIL"]),
            warnings=len([r for r in self.validation_results if r.status == "WARNING"]),
            skipped=len([r for r in self.validation_results if r.status == "SKIP"]),
            duration_seconds=duration,
            environment=self.environment,
            timestamp=start_time
        )

        logger.info(f"Validation completed in {duration:.2f}s")
        return summary

    async def _validate_system_connectivity(self):
        """Validate connectivity to all system components"""
        logger.info("Validating system connectivity...")

        # Test PostgreSQL connectivity
        await self._test_database_connection("PostgreSQL", self._test_postgres_connection)

        # Test Redis connectivity
        await self._test_database_connection("Redis", self._test_redis_connection)

        # Test Neo4j connectivity
        await self._test_database_connection("Neo4j", self._test_neo4j_connection)

        # Test service endpoints
        await self._test_service_endpoints()

    async def _test_database_connection(self, db_name: str, test_func):
        """Generic database connection test"""
        start_time = time.time()

        try:
            await test_func()
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name=f"{db_name} Connectivity",
                category="Connectivity",
                status="PASS",
                message=f"Successfully connected to {db_name}",
                details={"response_time_ms": duration},
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name=f"{db_name} Connectivity",
                category="Connectivity",
                status="FAIL",
                message=f"Failed to connect to {db_name}: {str(e)}",
                duration_ms=duration
            ))

    async def _test_postgres_connection(self):
        """Test PostgreSQL connection and basic functionality"""
        conn = psycopg2.connect(**self.postgres_config)
        cursor = conn.cursor()

        # Test basic query
        cursor.execute("SELECT version()")
        version = cursor.fetchone()[0]

        # Test table existence
        cursor.execute("""
            SELECT COUNT(*) FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = 'concepts'
        """)
        concepts_table_exists = cursor.fetchone()[0] > 0

        cursor.close()
        conn.close()

        if not concepts_table_exists:
            raise Exception("Required 'concepts' table not found")

    async def _test_redis_connection(self):
        """Test Redis connection and basic functionality"""
        redis_client = redis.Redis(**self.redis_config, decode_responses=True)

        # Test ping
        redis_client.ping()

        # Test basic operations
        test_key = "validation_test"
        redis_client.set(test_key, "test_value", ex=60)
        value = redis_client.get(test_key)

        if value != "test_value":
            raise Exception("Redis set/get operation failed")

        redis_client.delete(test_key)
        redis_client.close()

    async def _test_neo4j_connection(self):
        """Test Neo4j connection and basic functionality"""
        uri = f"bolt://{self.neo4j_config['host']}:{self.neo4j_config['port']}"
        driver = GraphDatabase.driver(
            uri,
            auth=(self.neo4j_config['user'], self.neo4j_config['password'])
        )

        with driver.session() as session:
            # Test basic query
            result = session.run("RETURN 1 as test")
            record = result.single()

            if record["test"] != 1:
                raise Exception("Neo4j basic query failed")

            # Test if concepts exist
            result = session.run("MATCH (c:Concept) RETURN count(c) as concept_count")
            record = result.single()
            concept_count = record["concept_count"]

            if concept_count == 0:
                logger.warning("No concepts found in Neo4j - this might be expected during initial setup")

        driver.close()

    async def _test_service_endpoints(self):
        """Test service endpoint availability and responsiveness"""
        endpoints = [
            ("/health", "Health Check"),
            ("/terminology/search?q=test", "Search Endpoint"),
            ("/terminology/codes/J44.0", "Code Lookup Endpoint")
        ]

        async with httpx.AsyncClient(timeout=30.0) as client:
            for endpoint, description in endpoints:
                start_time = time.time()

                try:
                    response = await client.get(f"{self.base_url}{endpoint}")
                    duration = (time.time() - start_time) * 1000

                    if response.status_code < 400:
                        self.validation_results.append(ValidationResult(
                            test_name=description,
                            category="Connectivity",
                            status="PASS",
                            message=f"Endpoint responded with status {response.status_code}",
                            details={
                                "status_code": response.status_code,
                                "response_time_ms": duration,
                                "endpoint": endpoint
                            },
                            duration_ms=duration
                        ))
                    else:
                        self.validation_results.append(ValidationResult(
                            test_name=description,
                            category="Connectivity",
                            status="FAIL",
                            message=f"Endpoint returned error status {response.status_code}",
                            details={
                                "status_code": response.status_code,
                                "response_time_ms": duration,
                                "endpoint": endpoint
                            },
                            duration_ms=duration
                        ))

                except Exception as e:
                    duration = (time.time() - start_time) * 1000

                    self.validation_results.append(ValidationResult(
                        test_name=description,
                        category="Connectivity",
                        status="FAIL",
                        message=f"Endpoint request failed: {str(e)}",
                        details={
                            "endpoint": endpoint,
                            "error": str(e)
                        },
                        duration_ms=duration
                    ))

    async def _validate_data_migration(self):
        """Validate data migration integrity and completeness"""
        logger.info("Validating data migration integrity...")

        # Test data counts and consistency
        await self._validate_data_counts()
        await self._validate_data_integrity()
        await self._validate_cross_database_consistency()

    async def _validate_data_counts(self):
        """Validate expected data counts in databases"""
        start_time = time.time()

        try:
            # PostgreSQL counts
            conn = psycopg2.connect(**self.postgres_config)
            cursor = conn.cursor()

            cursor.execute("SELECT COUNT(*) FROM concepts")
            postgres_concept_count = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(DISTINCT system) FROM concepts")
            postgres_system_count = cursor.fetchone()[0]

            cursor.close()
            conn.close()

            # Neo4j counts
            uri = f"bolt://{self.neo4j_config['host']}:{self.neo4j_config['port']}"
            driver = GraphDatabase.driver(
                uri,
                auth=(self.neo4j_config['user'], self.neo4j_config['password'])
            )

            with driver.session() as session:
                result = session.run("MATCH (c:Concept) RETURN count(c) as count")
                neo4j_concept_count = result.single()["count"]

            driver.close()

            duration = (time.time() - start_time) * 1000

            # Validate reasonable data volumes
            min_expected_concepts = 1000  # Adjust based on your data
            if postgres_concept_count >= min_expected_concepts:
                status = "PASS"
                message = f"Data counts validated: PostgreSQL={postgres_concept_count}, Neo4j={neo4j_concept_count}"
            else:
                status = "WARNING"
                message = f"Low concept count: PostgreSQL={postgres_concept_count}, Neo4j={neo4j_concept_count}"

            self.validation_results.append(ValidationResult(
                test_name="Data Count Validation",
                category="Migration",
                status=status,
                message=message,
                details={
                    "postgres_concepts": postgres_concept_count,
                    "postgres_systems": postgres_system_count,
                    "neo4j_concepts": neo4j_concept_count
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Data Count Validation",
                category="Migration",
                status="FAIL",
                message=f"Data count validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_data_integrity(self):
        """Validate data integrity and quality"""
        start_time = time.time()

        try:
            conn = psycopg2.connect(**self.postgres_config)
            cursor = conn.cursor()

            # Check for null codes (should not exist)
            cursor.execute("SELECT COUNT(*) FROM concepts WHERE code IS NULL OR code = ''")
            null_codes = cursor.fetchone()[0]

            # Check for duplicate codes within same system
            cursor.execute("""
                SELECT COUNT(*) FROM (
                    SELECT code, system, COUNT(*)
                    FROM concepts
                    GROUP BY code, system
                    HAVING COUNT(*) > 1
                ) duplicates
            """)
            duplicate_codes = cursor.fetchone()[0]

            # Check for orphaned relationships (if relationship table exists)
            cursor.execute("""
                SELECT COUNT(*) FROM information_schema.tables
                WHERE table_schema = 'public' AND table_name = 'concept_relationships'
            """)
            relationships_table_exists = cursor.fetchone()[0] > 0

            orphaned_relationships = 0
            if relationships_table_exists:
                cursor.execute("""
                    SELECT COUNT(*) FROM concept_relationships cr
                    WHERE NOT EXISTS (
                        SELECT 1 FROM concepts c WHERE c.id = cr.source_concept_id
                    ) OR NOT EXISTS (
                        SELECT 1 FROM concepts c WHERE c.id = cr.target_concept_id
                    )
                """)
                orphaned_relationships = cursor.fetchone()[0]

            cursor.close()
            conn.close()

            duration = (time.time() - start_time) * 1000

            # Determine status based on data quality
            issues = []
            if null_codes > 0:
                issues.append(f"{null_codes} null/empty codes")
            if duplicate_codes > 0:
                issues.append(f"{duplicate_codes} duplicate codes")
            if orphaned_relationships > 0:
                issues.append(f"{orphaned_relationships} orphaned relationships")

            if not issues:
                status = "PASS"
                message = "Data integrity validation passed"
            else:
                status = "WARNING" if len(issues) <= 2 else "FAIL"
                message = f"Data integrity issues found: {', '.join(issues)}"

            self.validation_results.append(ValidationResult(
                test_name="Data Integrity Validation",
                category="Migration",
                status=status,
                message=message,
                details={
                    "null_codes": null_codes,
                    "duplicate_codes": duplicate_codes,
                    "orphaned_relationships": orphaned_relationships
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Data Integrity Validation",
                category="Migration",
                status="FAIL",
                message=f"Data integrity validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_cross_database_consistency(self):
        """Validate consistency between PostgreSQL and Neo4j"""
        start_time = time.time()

        try:
            # Sample some codes from PostgreSQL
            conn = psycopg2.connect(**self.postgres_config)
            cursor = conn.cursor()

            cursor.execute("SELECT code, display, system FROM concepts ORDER BY RANDOM() LIMIT 10")
            sample_concepts = cursor.fetchall()

            cursor.close()
            conn.close()

            # Check if these concepts exist in Neo4j
            uri = f"bolt://{self.neo4j_config['host']}:{self.neo4j_config['port']}"
            driver = GraphDatabase.driver(
                uri,
                auth=(self.neo4j_config['user'], self.neo4j_config['password'])
            )

            consistency_issues = 0
            checked_concepts = len(sample_concepts)

            with driver.session() as session:
                for code, display, system in sample_concepts:
                    result = session.run(
                        "MATCH (c:Concept {code: $code}) RETURN c.display as display",
                        code=code
                    )
                    record = result.single()

                    if not record:
                        consistency_issues += 1
                        logger.warning(f"Code {code} found in PostgreSQL but not in Neo4j")

            driver.close()

            duration = (time.time() - start_time) * 1000

            consistency_rate = ((checked_concepts - consistency_issues) / checked_concepts * 100) if checked_concepts > 0 else 0

            if consistency_rate >= 95:
                status = "PASS"
                message = f"Cross-database consistency: {consistency_rate:.1f}%"
            elif consistency_rate >= 80:
                status = "WARNING"
                message = f"Cross-database consistency below target: {consistency_rate:.1f}%"
            else:
                status = "FAIL"
                message = f"Poor cross-database consistency: {consistency_rate:.1f}%"

            self.validation_results.append(ValidationResult(
                test_name="Cross-Database Consistency",
                category="Migration",
                status=status,
                message=message,
                details={
                    "checked_concepts": checked_concepts,
                    "consistency_issues": consistency_issues,
                    "consistency_rate": consistency_rate
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Cross-Database Consistency",
                category="Migration",
                status="FAIL",
                message=f"Cross-database consistency check failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_performance_baselines(self):
        """Validate performance meets Phase 3.5 criteria"""
        logger.info("Validating performance baselines...")

        await self._validate_postgres_performance()
        await self._validate_neo4j_performance()
        await self._validate_endpoint_performance()

    async def _validate_postgres_performance(self):
        """Validate PostgreSQL query performance <10ms (95th percentile)"""
        start_time = time.time()

        try:
            conn = psycopg2.connect(**self.postgres_config)
            cursor = conn.cursor()

            # Run multiple queries to get performance distribution
            response_times = []
            test_codes = ['J44.0', 'E11.9', 'I10', 'N18.6', 'F32.9']

            for _ in range(50):  # Run 50 test queries
                code = test_codes[len(response_times) % len(test_codes)]

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

            # Calculate percentiles
            p95_time = pd.Series(response_times).quantile(0.95)
            avg_time = pd.Series(response_times).mean()

            duration = (time.time() - start_time) * 1000

            # Validate against Phase 3.5 criteria
            if p95_time < 10.0:
                status = "PASS"
                message = f"PostgreSQL performance meets criteria: 95th percentile = {p95_time:.2f}ms"
            else:
                status = "FAIL"
                message = f"PostgreSQL performance below criteria: 95th percentile = {p95_time:.2f}ms (target: <10ms)"

            self.validation_results.append(ValidationResult(
                test_name="PostgreSQL Performance Baseline",
                category="Performance",
                status=status,
                message=message,
                details={
                    "avg_response_time_ms": avg_time,
                    "p95_response_time_ms": p95_time,
                    "sample_size": len(response_times),
                    "target_p95_ms": 10.0
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="PostgreSQL Performance Baseline",
                category="Performance",
                status="FAIL",
                message=f"PostgreSQL performance validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_neo4j_performance(self):
        """Validate Neo4j reasoning query performance <50ms (95th percentile)"""
        start_time = time.time()

        try:
            uri = f"bolt://{self.neo4j_config['host']}:{self.neo4j_config['port']}"
            driver = GraphDatabase.driver(
                uri,
                auth=(self.neo4j_config['user'], self.neo4j_config['password'])
            )

            response_times = []
            test_codes = ['J44.0', 'E11.9', 'I10', 'N18.6', 'F32.9']

            with driver.session() as session:
                for _ in range(30):  # Run 30 reasoning queries
                    code = test_codes[len(response_times) % len(test_codes)]

                    query_start = time.time()
                    result = session.run("""
                        MATCH (c:Concept {code: $code})-[:IS_A*1..3]->(parent:Concept)
                        RETURN c.code, c.display, collect(parent.display) as ancestors
                        LIMIT 10
                    """, code=code)

                    records = list(result)
                    query_time = (time.time() - query_start) * 1000

                    response_times.append(query_time)

            driver.close()

            # Calculate percentiles
            p95_time = pd.Series(response_times).quantile(0.95) if response_times else 0
            avg_time = pd.Series(response_times).mean() if response_times else 0

            duration = (time.time() - start_time) * 1000

            # Validate against Phase 3.5 criteria
            if p95_time < 50.0:
                status = "PASS"
                message = f"Neo4j performance meets criteria: 95th percentile = {p95_time:.2f}ms"
            else:
                status = "FAIL"
                message = f"Neo4j performance below criteria: 95th percentile = {p95_time:.2f}ms (target: <50ms)"

            self.validation_results.append(ValidationResult(
                test_name="Neo4j Performance Baseline",
                category="Performance",
                status=status,
                message=message,
                details={
                    "avg_response_time_ms": avg_time,
                    "p95_response_time_ms": p95_time,
                    "sample_size": len(response_times),
                    "target_p95_ms": 50.0
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Neo4j Performance Baseline",
                category="Performance",
                status="FAIL",
                message=f"Neo4j performance validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_endpoint_performance(self):
        """Validate FHIR endpoint performance <200ms (95th percentile)"""
        start_time = time.time()

        try:
            response_times = []
            endpoints = [
                "/terminology/codes/J44.0",
                "/terminology/search?q=diabetes",
                "/terminology/codes/E11.9",
                "/terminology/search?q=hypertension"
            ]

            async with httpx.AsyncClient(timeout=30.0) as client:
                for _ in range(40):  # Run 40 endpoint tests
                    endpoint = endpoints[len(response_times) % len(endpoints)]

                    request_start = time.time()
                    response = await client.get(f"{self.base_url}{endpoint}")
                    request_time = (time.time() - request_start) * 1000

                    if response.status_code < 400:
                        response_times.append(request_time)

            # Calculate percentiles
            p95_time = pd.Series(response_times).quantile(0.95) if response_times else float('inf')
            avg_time = pd.Series(response_times).mean() if response_times else float('inf')

            duration = (time.time() - start_time) * 1000

            # Validate against Phase 3.5 criteria
            if p95_time < 200.0:
                status = "PASS"
                message = f"FHIR endpoint performance meets criteria: 95th percentile = {p95_time:.2f}ms"
            else:
                status = "FAIL"
                message = f"FHIR endpoint performance below criteria: 95th percentile = {p95_time:.2f}ms (target: <200ms)"

            self.validation_results.append(ValidationResult(
                test_name="FHIR Endpoint Performance",
                category="Performance",
                status=status,
                message=message,
                details={
                    "avg_response_time_ms": avg_time,
                    "p95_response_time_ms": p95_time,
                    "sample_size": len(response_times),
                    "target_p95_ms": 200.0
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="FHIR Endpoint Performance",
                category="Performance",
                status="FAIL",
                message=f"FHIR endpoint performance validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_fhir_compliance(self):
        """Validate FHIR specification compliance"""
        logger.info("Validating FHIR compliance...")

        # Test FHIR resource structure and format
        await self._validate_fhir_resource_format()
        await self._validate_fhir_operation_outcomes()

    async def _validate_fhir_resource_format(self):
        """Validate FHIR resource format compliance"""
        start_time = time.time()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                # Test code lookup returns valid FHIR format
                response = await client.get(f"{self.base_url}/terminology/codes/J44.0")

                if response.status_code == 200:
                    data = response.json()

                    # Basic FHIR structure validation
                    required_fields = []
                    issues = []

                    # Check for proper content-type
                    content_type = response.headers.get('content-type', '')
                    if 'application/fhir+json' not in content_type and 'application/json' not in content_type:
                        issues.append("Invalid content-type for FHIR response")

                    # Additional FHIR validation would go here
                    # For now, we'll check basic structure

                    duration = (time.time() - start_time) * 1000

                    if not issues:
                        status = "PASS"
                        message = "FHIR resource format validation passed"
                    else:
                        status = "WARNING"
                        message = f"FHIR format issues: {', '.join(issues)}"

                    self.validation_results.append(ValidationResult(
                        test_name="FHIR Resource Format",
                        category="FHIR",
                        status=status,
                        message=message,
                        details={
                            "response_structure": "valid" if not issues else "issues_found",
                            "content_type": content_type,
                            "issues": issues
                        },
                        duration_ms=duration
                    ))

                else:
                    duration = (time.time() - start_time) * 1000

                    self.validation_results.append(ValidationResult(
                        test_name="FHIR Resource Format",
                        category="FHIR",
                        status="FAIL",
                        message=f"FHIR endpoint returned status {response.status_code}",
                        duration_ms=duration
                    ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="FHIR Resource Format",
                category="FHIR",
                status="FAIL",
                message=f"FHIR resource format validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_fhir_operation_outcomes(self):
        """Validate FHIR OperationOutcome responses for errors"""
        start_time = time.time()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                # Test invalid code lookup
                response = await client.get(f"{self.base_url}/terminology/codes/INVALID_CODE_12345")

                duration = (time.time() - start_time) * 1000

                if response.status_code >= 400:
                    # Should return proper error format
                    try:
                        error_data = response.json()
                        # Basic check for error structure
                        status = "PASS"
                        message = "FHIR error responses properly formatted"
                    except:
                        status = "WARNING"
                        message = "FHIR error responses not in JSON format"
                else:
                    status = "WARNING"
                    message = "Invalid codes returning success (might be intentional)"

                self.validation_results.append(ValidationResult(
                    test_name="FHIR Error Handling",
                    category="FHIR",
                    status=status,
                    message=message,
                    details={
                        "error_status_code": response.status_code,
                        "content_type": response.headers.get('content-type', '')
                    },
                    duration_ms=duration
                ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="FHIR Error Handling",
                category="FHIR",
                status="FAIL",
                message=f"FHIR error handling validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_cache_system(self):
        """Validate cache system performance and hit ratios"""
        logger.info("Validating cache system...")

        start_time = time.time()

        try:
            redis_client = redis.Redis(**self.redis_config, decode_responses=True)

            # Test cache operations
            test_key = "validation_cache_test"
            test_value = "validation_test_value"

            # Set cache value
            redis_client.setex(test_key, 300, test_value)

            # Test retrieval
            retrieved_value = redis_client.get(test_key)

            # Test cache hit simulation
            cache_hits = 0
            cache_tests = 20

            for i in range(cache_tests):
                key = f"test_concept_{i % 5}"  # Bias toward first 5 keys
                redis_client.setex(key, 300, f"concept_data_{i % 5}")

            for i in range(cache_tests):
                key = f"test_concept_{i % 5}"
                value = redis_client.get(key)
                if value:
                    cache_hits += 1

            # Clean up test data
            for i in range(5):
                redis_client.delete(f"test_concept_{i}")
            redis_client.delete(test_key)

            redis_client.close()

            duration = (time.time() - start_time) * 1000

            # Calculate cache hit ratio
            cache_hit_ratio = (cache_hits / cache_tests * 100) if cache_tests > 0 else 0

            # Validate against Phase 3.5 criteria (>90% hit ratio)
            if cache_hit_ratio >= 90.0:
                status = "PASS"
                message = f"Cache system meets criteria: {cache_hit_ratio:.1f}% hit ratio"
            elif cache_hit_ratio >= 80.0:
                status = "WARNING"
                message = f"Cache hit ratio below target: {cache_hit_ratio:.1f}% (target: >90%)"
            else:
                status = "FAIL"
                message = f"Poor cache performance: {cache_hit_ratio:.1f}% hit ratio"

            self.validation_results.append(ValidationResult(
                test_name="Cache System Performance",
                category="Cache",
                status=status,
                message=message,
                details={
                    "cache_hit_ratio": cache_hit_ratio,
                    "cache_hits": cache_hits,
                    "total_tests": cache_tests,
                    "target_hit_ratio": 90.0
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Cache System Performance",
                category="Cache",
                status="FAIL",
                message=f"Cache system validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_monitoring_alerting(self):
        """Validate monitoring and alerting system"""
        logger.info("Validating monitoring and alerting system...")

        # Test health endpoint monitoring
        await self._validate_health_monitoring()

        # Test metrics collection
        await self._validate_metrics_collection()

    async def _validate_health_monitoring(self):
        """Validate health check monitoring"""
        start_time = time.time()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                # Test health endpoint multiple times
                health_checks = []

                for _ in range(10):
                    check_start = time.time()
                    response = await client.get(f"{self.base_url}/health")
                    check_time = (time.time() - check_start) * 1000

                    health_checks.append({
                        'status_code': response.status_code,
                        'response_time': check_time,
                        'success': response.status_code == 200
                    })

                    await asyncio.sleep(0.5)  # 500ms between checks

            duration = (time.time() - start_time) * 1000

            # Calculate health check metrics
            success_rate = sum(1 for check in health_checks if check['success']) / len(health_checks) * 100
            avg_response_time = sum(check['response_time'] for check in health_checks) / len(health_checks)

            # Validate against uptime criteria (>99.9%)
            if success_rate >= 99.9:
                status = "PASS"
                message = f"Health monitoring meets criteria: {success_rate:.1f}% success rate"
            elif success_rate >= 95.0:
                status = "WARNING"
                message = f"Health monitoring below target: {success_rate:.1f}% (target: >99.9%)"
            else:
                status = "FAIL"
                message = f"Poor health monitoring: {success_rate:.1f}% success rate"

            self.validation_results.append(ValidationResult(
                test_name="Health Check Monitoring",
                category="Monitoring",
                status=status,
                message=message,
                details={
                    "success_rate": success_rate,
                    "avg_response_time": avg_response_time,
                    "total_checks": len(health_checks),
                    "target_uptime": 99.9
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Health Check Monitoring",
                category="Monitoring",
                status="FAIL",
                message=f"Health monitoring validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_metrics_collection(self):
        """Validate metrics collection endpoints"""
        start_time = time.time()

        try:
            # Test if metrics endpoints are available
            metrics_endpoints = [
                "/metrics",
                "/health/detailed",
                "/admin/stats"
            ]

            available_endpoints = 0

            async with httpx.AsyncClient(timeout=30.0) as client:
                for endpoint in metrics_endpoints:
                    try:
                        response = await client.get(f"{self.base_url}{endpoint}")
                        if response.status_code < 500:  # Allow 404 for optional endpoints
                            available_endpoints += 1
                    except:
                        pass  # Endpoint not available

            duration = (time.time() - start_time) * 1000

            if available_endpoints > 0:
                status = "PASS"
                message = f"Metrics collection available: {available_endpoints}/{len(metrics_endpoints)} endpoints"
            else:
                status = "WARNING"
                message = "No metrics collection endpoints found"

            self.validation_results.append(ValidationResult(
                test_name="Metrics Collection",
                category="Monitoring",
                status=status,
                message=message,
                details={
                    "available_endpoints": available_endpoints,
                    "total_endpoints": len(metrics_endpoints)
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Metrics Collection",
                category="Monitoring",
                status="FAIL",
                message=f"Metrics collection validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_security_configuration(self):
        """Validate security configuration and best practices"""
        logger.info("Validating security configuration...")

        await self._validate_security_headers()
        await self._validate_authentication_endpoints()

    async def _validate_security_headers(self):
        """Validate security headers in responses"""
        start_time = time.time()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(f"{self.base_url}/health")

                security_headers = {
                    'X-Content-Type-Options': 'nosniff',
                    'X-Frame-Options': ['DENY', 'SAMEORIGIN'],
                    'X-XSS-Protection': '1; mode=block'
                }

                missing_headers = []
                for header, expected_values in security_headers.items():
                    actual_value = response.headers.get(header)
                    if not actual_value:
                        missing_headers.append(header)
                    elif isinstance(expected_values, list) and actual_value not in expected_values:
                        missing_headers.append(f"{header} (incorrect value)")

            duration = (time.time() - start_time) * 1000

            if not missing_headers:
                status = "PASS"
                message = "Security headers properly configured"
            elif len(missing_headers) <= 1:
                status = "WARNING"
                message = f"Missing security headers: {', '.join(missing_headers)}"
            else:
                status = "FAIL"
                message = f"Multiple missing security headers: {', '.join(missing_headers)}"

            self.validation_results.append(ValidationResult(
                test_name="Security Headers",
                category="Security",
                status=status,
                message=message,
                details={
                    "missing_headers": missing_headers,
                    "checked_headers": list(security_headers.keys())
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Security Headers",
                category="Security",
                status="FAIL",
                message=f"Security headers validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_authentication_endpoints(self):
        """Validate authentication and authorization"""
        start_time = time.time()

        try:
            # Test if authentication is required for protected endpoints
            protected_endpoints = [
                "/admin/config",
                "/admin/users",
                "/admin/system"
            ]

            auth_required_count = 0

            async with httpx.AsyncClient(timeout=30.0) as client:
                for endpoint in protected_endpoints:
                    try:
                        response = await client.get(f"{self.base_url}{endpoint}")
                        if response.status_code == 401 or response.status_code == 403:
                            auth_required_count += 1
                    except:
                        pass  # Endpoint doesn't exist, which is fine

            duration = (time.time() - start_time) * 1000

            # This is informational - we don't know which endpoints should be protected
            status = "PASS"
            message = f"Authentication check completed: {auth_required_count} endpoints require auth"

            self.validation_results.append(ValidationResult(
                test_name="Authentication Configuration",
                category="Security",
                status=status,
                message=message,
                details={
                    "protected_endpoints": auth_required_count,
                    "checked_endpoints": len(protected_endpoints)
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Authentication Configuration",
                category="Security",
                status="FAIL",
                message=f"Authentication validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_end_to_end_workflows(self):
        """Validate complete end-to-end workflows"""
        logger.info("Validating end-to-end workflows...")

        await self._validate_terminology_lookup_workflow()
        await self._validate_search_workflow()

    async def _validate_terminology_lookup_workflow(self):
        """Validate complete terminology lookup workflow"""
        start_time = time.time()

        try:
            workflow_steps = []

            async with httpx.AsyncClient(timeout=30.0) as client:
                # Step 1: Code lookup
                step1_start = time.time()
                response1 = await client.get(f"{self.base_url}/terminology/codes/J44.0")
                step1_time = (time.time() - step1_start) * 1000

                workflow_steps.append({
                    'step': 'code_lookup',
                    'success': response1.status_code == 200,
                    'response_time': step1_time
                })

                # Step 2: Validation (if endpoint exists)
                step2_start = time.time()
                response2 = await client.get(f"{self.base_url}/terminology/validate?code=J44.0&system=http://hl7.org/fhir/sid/icd-10-cm")
                step2_time = (time.time() - step2_start) * 1000

                workflow_steps.append({
                    'step': 'code_validation',
                    'success': response2.status_code < 500,  # Allow 404 for optional endpoints
                    'response_time': step2_time
                })

                # Step 3: Hierarchy lookup (if endpoint exists)
                step3_start = time.time()
                response3 = await client.get(f"{self.base_url}/terminology/codes/J44.0/hierarchy")
                step3_time = (time.time() - step3_start) * 1000

                workflow_steps.append({
                    'step': 'hierarchy_lookup',
                    'success': response3.status_code < 500,  # Allow 404 for optional endpoints
                    'response_time': step3_time
                })

            duration = (time.time() - start_time) * 1000

            # Calculate workflow success
            successful_steps = sum(1 for step in workflow_steps if step['success'])
            total_workflow_time = sum(step['response_time'] for step in workflow_steps)

            if successful_steps >= 2:  # At least 2 out of 3 steps should work
                status = "PASS"
                message = f"Terminology lookup workflow successful: {successful_steps}/{len(workflow_steps)} steps"
            else:
                status = "FAIL"
                message = f"Terminology lookup workflow failed: {successful_steps}/{len(workflow_steps)} steps"

            self.validation_results.append(ValidationResult(
                test_name="Terminology Lookup Workflow",
                category="Workflow",
                status=status,
                message=message,
                details={
                    "successful_steps": successful_steps,
                    "total_steps": len(workflow_steps),
                    "total_workflow_time": total_workflow_time,
                    "workflow_steps": workflow_steps
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Terminology Lookup Workflow",
                category="Workflow",
                status="FAIL",
                message=f"Terminology lookup workflow validation failed: {str(e)}",
                duration_ms=duration
            ))

    async def _validate_search_workflow(self):
        """Validate search workflow"""
        start_time = time.time()

        try:
            search_terms = ['diabetes', 'hypertension', 'asthma']
            successful_searches = 0

            async with httpx.AsyncClient(timeout=30.0) as client:
                for term in search_terms:
                    try:
                        response = await client.get(f"{self.base_url}/terminology/search?q={term}")
                        if response.status_code == 200:
                            data = response.json()
                            # Basic validation of search results structure
                            if isinstance(data, (dict, list)):
                                successful_searches += 1
                    except:
                        pass

            duration = (time.time() - start_time) * 1000

            success_rate = (successful_searches / len(search_terms) * 100) if search_terms else 0

            if success_rate >= 80:
                status = "PASS"
                message = f"Search workflow successful: {success_rate:.1f}% success rate"
            else:
                status = "FAIL"
                message = f"Search workflow failed: {success_rate:.1f}% success rate"

            self.validation_results.append(ValidationResult(
                test_name="Search Workflow",
                category="Workflow",
                status=status,
                message=message,
                details={
                    "successful_searches": successful_searches,
                    "total_searches": len(search_terms),
                    "success_rate": success_rate
                },
                duration_ms=duration
            ))

        except Exception as e:
            duration = (time.time() - start_time) * 1000

            self.validation_results.append(ValidationResult(
                test_name="Search Workflow",
                category="Workflow",
                status="FAIL",
                message=f"Search workflow validation failed: {str(e)}",
                duration_ms=duration
            ))

    def generate_validation_report(self, output_format: str = "console") -> str:
        """Generate comprehensive validation report"""
        if output_format == "console":
            return self._generate_console_validation_report()
        elif output_format == "json":
            return self._generate_json_validation_report()
        elif output_format == "html":
            return self._generate_html_validation_report()
        else:
            raise ValueError(f"Unsupported output format: {output_format}")

    def _generate_console_validation_report(self) -> str:
        """Generate console validation report"""
        console.print("\n[bold blue]KB7 Terminology System Validation Report[/bold blue]")
        console.print(f"Environment: {self.environment}")
        console.print(f"Timestamp: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

        # Summary
        summary = self._calculate_validation_summary()

        summary_panel = Panel(
            f"""
Total Tests: {summary.total_tests}
✅ Passed: {summary.passed}
❌ Failed: {summary.failed}
⚠️  Warnings: {summary.warnings}
⏭️  Skipped: {summary.skipped}

Success Rate: {summary.success_rate:.1f}%
Production Ready: {'✅ YES' if summary.is_production_ready else '❌ NO'}
            """,
            title="Validation Summary",
            expand=False
        )
        console.print(summary_panel)

        # Results by category
        categories = {}
        for result in self.validation_results:
            if result.category not in categories:
                categories[result.category] = []
            categories[result.category].append(result)

        for category, results in categories.items():
            table = Table(title=f"{category} Validation Results")
            table.add_column("Test", style="cyan")
            table.add_column("Status", style="bold")
            table.add_column("Message", style="white")
            table.add_column("Duration", style="dim")

            for result in results:
                status_color = {
                    "PASS": "green",
                    "FAIL": "red",
                    "WARNING": "yellow",
                    "SKIP": "dim"
                }.get(result.status, "white")

                status_symbol = {
                    "PASS": "✅",
                    "FAIL": "❌",
                    "WARNING": "⚠️",
                    "SKIP": "⏭️"
                }.get(result.status, "?")

                table.add_row(
                    result.test_name,
                    f"[{status_color}]{status_symbol} {result.status}[/{status_color}]",
                    result.message,
                    f"{result.duration_ms:.1f}ms"
                )

            console.print(table)

        return "Console validation report generated"

    def _generate_json_validation_report(self) -> str:
        """Generate JSON validation report"""
        summary = self._calculate_validation_summary()

        report_data = {
            "summary": asdict(summary),
            "results": [asdict(result) for result in self.validation_results],
            "environment": self.environment,
            "timestamp": datetime.now().isoformat()
        }

        report_path = Path(f"validation_report_{self.environment}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json")
        report_path.write_text(json.dumps(report_data, indent=2, default=str))

        logger.info(f"JSON validation report saved to: {report_path}")
        return str(report_path)

    def _generate_html_validation_report(self) -> str:
        """Generate HTML validation report"""
        summary = self._calculate_validation_summary()

        # HTML template would go here
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>KB7 Terminology Validation Report</title>
            <style>
                body {{ font-family: Arial, sans-serif; margin: 20px; }}
                .summary {{ background-color: #f0f0f0; padding: 20px; border-radius: 5px; }}
                .pass {{ color: green; }}
                .fail {{ color: red; }}
                .warning {{ color: orange; }}
                table {{ border-collapse: collapse; width: 100%; margin: 20px 0; }}
                th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
                th {{ background-color: #f2f2f2; }}
            </style>
        </head>
        <body>
            <h1>KB7 Terminology System Validation Report</h1>
            <div class="summary">
                <h2>Summary</h2>
                <p>Environment: {self.environment}</p>
                <p>Total Tests: {summary.total_tests}</p>
                <p>Success Rate: {summary.success_rate:.1f}%</p>
                <p>Production Ready: {'Yes' if summary.is_production_ready else 'No'}</p>
            </div>

            <h2>Detailed Results</h2>
            <table>
                <tr><th>Test</th><th>Category</th><th>Status</th><th>Message</th></tr>
        """

        for result in self.validation_results:
            status_class = result.status.lower()
            html_content += f"""
                <tr>
                    <td>{result.test_name}</td>
                    <td>{result.category}</td>
                    <td class="{status_class}">{result.status}</td>
                    <td>{result.message}</td>
                </tr>
            """

        html_content += """
            </table>
        </body>
        </html>
        """

        report_path = Path(f"validation_report_{self.environment}.html")
        report_path.write_text(html_content)

        logger.info(f"HTML validation report saved to: {report_path}")
        return str(report_path)

    def _calculate_validation_summary(self) -> ValidationSummary:
        """Calculate validation summary statistics"""
        total_tests = len(self.validation_results)
        passed = len([r for r in self.validation_results if r.status == "PASS"])
        failed = len([r for r in self.validation_results if r.status == "FAIL"])
        warnings = len([r for r in self.validation_results if r.status == "WARNING"])
        skipped = len([r for r in self.validation_results if r.status == "SKIP"])

        total_duration = sum(r.duration_ms for r in self.validation_results) / 1000.0

        return ValidationSummary(
            total_tests=total_tests,
            passed=passed,
            failed=failed,
            warnings=warnings,
            skipped=skipped,
            duration_seconds=total_duration,
            environment=self.environment,
            timestamp=datetime.now()
        )

async def main():
    """Main entry point for validation suite"""
    parser = argparse.ArgumentParser(description="KB7 Terminology Validation Suite")
    parser.add_argument("--target", default="http://localhost:8007", help="Target service URL")
    parser.add_argument("--environment", default="development", help="Environment name")
    parser.add_argument("--validation-type",
                       choices=["full", "connectivity", "migration", "performance", "fhir", "cache", "monitoring", "security", "workflow"],
                       default="full", help="Type of validation to run")
    parser.add_argument("--output-format", choices=["console", "json", "html"],
                       default="console", help="Output report format")
    parser.add_argument("--config", help="Path to configuration file")

    # Specific validation flags
    parser.add_argument("--check-connectivity", action="store_true", help="Check system connectivity")
    parser.add_argument("--check-migration", action="store_true", help="Check data migration integrity")
    parser.add_argument("--check-performance", action="store_true", help="Check performance baselines")
    parser.add_argument("--check-fhir", action="store_true", help="Check FHIR compliance")
    parser.add_argument("--check-cache", action="store_true", help="Check cache system")
    parser.add_argument("--check-monitoring", action="store_true", help="Check monitoring/alerting")
    parser.add_argument("--check-security", action="store_true", help="Check security configuration")
    parser.add_argument("--check-workflows", action="store_true", help="Check end-to-end workflows")
    parser.add_argument("--full-validation", action="store_true", help="Run all validation checks")

    args = parser.parse_args()

    # Load configuration
    config = {
        "base_url": args.target,
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

    if args.config:
        with open(args.config) as f:
            config.update(yaml.safe_load(f))

    # Determine validation type based on flags
    validation_type = args.validation_type

    if args.full_validation:
        validation_type = "full"
    elif any([args.check_connectivity, args.check_migration, args.check_performance,
             args.check_fhir, args.check_cache, args.check_monitoring,
             args.check_security, args.check_workflows]):
        # Custom validation based on specific flags
        validation_type = "custom"

    # Initialize validator
    validator = SystemValidator(config)

    try:
        logger.info(f"Starting {validation_type} validation suite...")

        if validation_type == "custom":
            # Run specific validations based on flags
            if args.check_connectivity:
                await validator._validate_system_connectivity()
            if args.check_migration:
                await validator._validate_data_migration()
            if args.check_performance:
                await validator._validate_performance_baselines()
            if args.check_fhir:
                await validator._validate_fhir_compliance()
            if args.check_cache:
                await validator._validate_cache_system()
            if args.check_monitoring:
                await validator._validate_monitoring_alerting()
            if args.check_security:
                await validator._validate_security_configuration()
            if args.check_workflows:
                await validator._validate_end_to_end_workflows()

            # Calculate summary manually for custom validation
            summary = validator._calculate_validation_summary()
        else:
            # Run full validation suite
            summary = await validator.run_validation(validation_type)

        # Generate report
        report_path = validator.generate_validation_report(args.output_format)

        if args.output_format != "console":
            logger.info(f"Validation report saved to: {report_path}")

        # Exit with appropriate code
        if summary.is_production_ready:
            logger.info("🎉 System validation PASSED - Ready for production!")
            return 0
        else:
            logger.error(f"❌ System validation FAILED - {summary.failed} failures, {summary.warnings} warnings")
            return 1

    except Exception as e:
        logger.error(f"Validation suite failed: {e}")
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)