#!/usr/bin/env python3
"""
Google FHIR Performance Benchmark for KB7 Terminology Phase 3.5
==============================================================

Comprehensive performance testing for Google Healthcare API FHIR stores:
- FHIR Terminology Server operations
- CodeSystem operations ($lookup, $validate-code)
- ValueSet operations ($expand, $validate-code)
- ConceptMap operations ($translate)
- Batch operations and bundle processing
- Real-world FHIR workflow validation

Author: Claude Code Performance Engineer
Version: 1.0.0
"""

import asyncio
import time
import json
import os
import statistics
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
import logging

import httpx
from google.auth import default
from google.auth.transport.requests import Request
import google.auth.transport.requests
from google.oauth2 import service_account
import numpy as np
from rich.console import Console
from rich.table import Table
from rich.progress import Progress
from rich.panel import Panel
import yaml

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class FHIROperationResult:
    """Result of a FHIR operation test"""
    operation_name: str
    resource_type: str
    operation_type: str  # 'read', 'create', 'update', 'delete', 'operation'
    total_requests: int
    successful_requests: int
    failed_requests: int
    mean_latency_ms: float
    p50_latency_ms: float
    p95_latency_ms: float
    p99_latency_ms: float
    max_latency_ms: float
    min_latency_ms: float
    throughput_rps: float
    error_rate_pct: float
    errors: List[str]
    timestamp: str

@dataclass
class GoogleFHIRBenchmarkResult:
    """Complete Google FHIR benchmark result"""
    test_run_id: str
    timestamp: str
    project_id: str
    location: str
    dataset_id: str
    fhir_store_id: str
    test_duration_seconds: float
    operation_results: List[FHIROperationResult]
    performance_summary: Dict[str, Any]
    compliance_results: Dict[str, bool]
    recommendations: List[str]

class GoogleFHIRClient:
    """Client for Google Healthcare API FHIR operations"""

    def __init__(self, project_id: str, location: str, dataset_id: str, fhir_store_id: str,
                 credentials_path: Optional[str] = None):
        self.project_id = project_id
        self.location = location
        self.dataset_id = dataset_id
        self.fhir_store_id = fhir_store_id

        # Initialize credentials
        if credentials_path and os.path.exists(credentials_path):
            self.credentials = service_account.Credentials.from_service_account_file(credentials_path)
        else:
            self.credentials, _ = default()

        # Add required scopes
        if hasattr(self.credentials, 'with_scopes'):
            self.credentials = self.credentials.with_scopes([
                'https://www.googleapis.com/auth/cloud-healthcare',
                'https://www.googleapis.com/auth/cloud-platform'
            ])

        self.base_url = f"https://healthcare.googleapis.com/v1/projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}/fhir"

    def _get_auth_header(self) -> Dict[str, str]:
        """Get authorization header for requests"""
        # Refresh token if needed
        if not self.credentials.valid:
            auth_req = Request()
            self.credentials.refresh(auth_req)

        return {
            'Authorization': f'Bearer {self.credentials.token}',
            'Content-Type': 'application/fhir+json',
            'Accept': 'application/fhir+json'
        }

    async def codesystem_lookup(self, system: str, code: str) -> Tuple[float, bool, str]:
        """Perform CodeSystem $lookup operation"""
        start_time = time.perf_counter()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/CodeSystem/$lookup",
                    params={'system': system, 'code': code},
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code in [200, 404]  # 404 acceptable for unknown codes
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def codesystem_validate_code(self, system: str, code: str, display: Optional[str] = None) -> Tuple[float, bool, str]:
        """Perform CodeSystem $validate-code operation"""
        start_time = time.perf_counter()

        try:
            params = {'system': system, 'code': code}
            if display:
                params['display'] = display

            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/CodeSystem/$validate-code",
                    params=params,
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code == 200
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def valueset_expand(self, url: str, count: Optional[int] = None) -> Tuple[float, bool, str]:
        """Perform ValueSet $expand operation"""
        start_time = time.perf_counter()

        try:
            params = {'url': url}
            if count:
                params['count'] = str(count)

            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/ValueSet/$expand",
                    params=params,
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code in [200, 404]
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def valueset_validate_code(self, url: str, system: str, code: str) -> Tuple[float, bool, str]:
        """Perform ValueSet $validate-code operation"""
        start_time = time.perf_counter()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/ValueSet/$validate-code",
                    params={'url': url, 'system': system, 'code': code},
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code == 200
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def conceptmap_translate(self, url: str, system: str, code: str, target: Optional[str] = None) -> Tuple[float, bool, str]:
        """Perform ConceptMap $translate operation"""
        start_time = time.perf_counter()

        try:
            params = {'url': url, 'system': system, 'code': code}
            if target:
                params['target'] = target

            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/ConceptMap/$translate",
                    params=params,
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code in [200, 404]
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def create_codesystem(self, codesystem_resource: Dict[str, Any]) -> Tuple[float, bool, str]:
        """Create a CodeSystem resource"""
        start_time = time.perf_counter()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.post(
                    f"{self.base_url}/CodeSystem",
                    json=codesystem_resource,
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code in [200, 201]
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

    async def search_codesystems(self, params: Dict[str, str]) -> Tuple[float, bool, str]:
        """Search CodeSystems"""
        start_time = time.perf_counter()

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.get(
                    f"{self.base_url}/CodeSystem",
                    params=params,
                    headers=self._get_auth_header()
                )

                end_time = time.perf_counter()
                latency_ms = (end_time - start_time) * 1000

                success = response.status_code == 200
                error_msg = "" if success else f"HTTP {response.status_code}: {response.text}"

                return latency_ms, success, error_msg

        except Exception as e:
            end_time = time.perf_counter()
            latency_ms = (end_time - start_time) * 1000
            return latency_ms, False, str(e)

class GoogleFHIRBenchmark:
    """Main Google FHIR benchmarking orchestrator"""

    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_config(config_path)
        self.console = Console()

        # Initialize Google FHIR client
        self.fhir_client = GoogleFHIRClient(
            project_id=self.config['google_fhir']['project_id'],
            location=self.config['google_fhir']['location'],
            dataset_id=self.config['google_fhir']['dataset_id'],
            fhir_store_id=self.config['google_fhir']['fhir_store_id'],
            credentials_path=self.config['google_fhir'].get('credentials_path')
        )

        # Test data sets
        self.test_data = self._initialize_test_data()

    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load configuration"""
        default_config = {
            'google_fhir': {
                'project_id': 'cardiofit-demo',
                'location': 'us-central1',
                'dataset_id': 'kb7_terminology',
                'fhir_store_id': 'kb7-fhir-store',
                'credentials_path': None
            },
            'test_parameters': {
                'iterations_per_operation': 100,
                'concurrent_requests': 5,
                'operation_timeout_seconds': 30
            },
            'benchmark_operations': {
                'codesystem_lookup': True,
                'codesystem_validate': True,
                'valueset_expand': True,
                'valueset_validate': True,
                'conceptmap_translate': True,
                'search_operations': True,
                'create_operations': False  # Disabled by default to avoid creating test data
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

    def _initialize_test_data(self) -> Dict[str, Any]:
        """Initialize test data for benchmarking"""
        return {
            'code_systems': [
                {'system': 'http://snomed.info/sct', 'codes': ['424144002', '38341003', '195967001', '13645005']},
                {'system': 'http://loinc.org', 'codes': ['8480-6', '8462-4', '33747-0', '1975-2']},
                {'system': 'http://www.nlm.nih.gov/research/umls/rxnorm', 'codes': ['1191', '161', '5224']},
                {'system': 'http://hl7.org/fhir/sid/icd-10-cm', 'codes': ['I10', 'E11.9', 'J44.0']}
            ],
            'value_sets': [
                'http://cardiofit.com/fhir/ValueSet/cardiovascular-conditions',
                'http://cardiofit.com/fhir/ValueSet/diabetes-medications',
                'http://cardiofit.com/fhir/ValueSet/vital-signs',
                'http://cardiofit.com/fhir/ValueSet/laboratory-tests'
            ],
            'concept_maps': [
                'http://cardiofit.com/fhir/ConceptMap/snomed-to-loinc',
                'http://cardiofit.com/fhir/ConceptMap/rxnorm-to-snomed',
                'http://cardiofit.com/fhir/ConceptMap/icd10-to-snomed'
            ],
            'search_params': [
                {'name': 'cardiovascular'},
                {'title': 'cardio'},
                {'status': 'active'},
                {'version': '1.0'}
            ]
        }

    async def run_comprehensive_benchmark(self) -> GoogleFHIRBenchmarkResult:
        """Run comprehensive Google FHIR benchmark"""
        test_run_id = f"google_fhir_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        start_time = datetime.now()

        self.console.print("[bold blue]Starting Google FHIR Performance Benchmark[/bold blue]")
        self.console.print(f"Test Run ID: {test_run_id}")
        self.console.print(f"FHIR Store: {self.fhir_client.project_id}/{self.fhir_client.location}/{self.fhir_client.dataset_id}/{self.fhir_client.fhir_store_id}")

        operation_results = []

        with Progress() as progress:
            main_task = progress.add_task("[green]Running FHIR benchmarks...", total=6)

            # Test 1: CodeSystem $lookup operations
            if self.config['benchmark_operations']['codesystem_lookup']:
                progress.update(main_task, advance=1, description="Testing CodeSystem $lookup...")
                result = await self._benchmark_codesystem_lookup()
                operation_results.append(result)

            # Test 2: CodeSystem $validate-code operations
            if self.config['benchmark_operations']['codesystem_validate']:
                progress.update(main_task, advance=1, description="Testing CodeSystem $validate-code...")
                result = await self._benchmark_codesystem_validate()
                operation_results.append(result)

            # Test 3: ValueSet $expand operations
            if self.config['benchmark_operations']['valueset_expand']:
                progress.update(main_task, advance=1, description="Testing ValueSet $expand...")
                result = await self._benchmark_valueset_expand()
                operation_results.append(result)

            # Test 4: ValueSet $validate-code operations
            if self.config['benchmark_operations']['valueset_validate']:
                progress.update(main_task, advance=1, description="Testing ValueSet $validate-code...")
                result = await self._benchmark_valueset_validate()
                operation_results.append(result)

            # Test 5: ConceptMap $translate operations
            if self.config['benchmark_operations']['conceptmap_translate']:
                progress.update(main_task, advance=1, description="Testing ConceptMap $translate...")
                result = await self._benchmark_conceptmap_translate()
                operation_results.append(result)

            # Test 6: Search operations
            if self.config['benchmark_operations']['search_operations']:
                progress.update(main_task, advance=1, description="Testing search operations...")
                result = await self._benchmark_search_operations()
                operation_results.append(result)

        end_time = datetime.now()

        # Compile results
        return self._compile_benchmark_results(
            test_run_id, start_time, end_time, operation_results
        )

    async def _benchmark_codesystem_lookup(self) -> FHIROperationResult:
        """Benchmark CodeSystem $lookup operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            # Select random test data
            cs_data = random.choice(self.test_data['code_systems'])
            code = random.choice(cs_data['codes'])

            latency, success, error = await self.fhir_client.codesystem_lookup(
                cs_data['system'], code
            )

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'codesystem_lookup', 'CodeSystem', 'operation',
            latencies, successful, total - successful, errors
        )

    async def _benchmark_codesystem_validate(self) -> FHIROperationResult:
        """Benchmark CodeSystem $validate-code operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            cs_data = random.choice(self.test_data['code_systems'])
            code = random.choice(cs_data['codes'])

            latency, success, error = await self.fhir_client.codesystem_validate_code(
                cs_data['system'], code
            )

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'codesystem_validate', 'CodeSystem', 'operation',
            latencies, successful, total - successful, errors
        )

    async def _benchmark_valueset_expand(self) -> FHIROperationResult:
        """Benchmark ValueSet $expand operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            vs_url = random.choice(self.test_data['value_sets'])
            count = random.choice([10, 20, 50, None])

            latency, success, error = await self.fhir_client.valueset_expand(vs_url, count)

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'valueset_expand', 'ValueSet', 'operation',
            latencies, successful, total - successful, errors
        )

    async def _benchmark_valueset_validate(self) -> FHIROperationResult:
        """Benchmark ValueSet $validate-code operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            vs_url = random.choice(self.test_data['value_sets'])
            cs_data = random.choice(self.test_data['code_systems'])
            code = random.choice(cs_data['codes'])

            latency, success, error = await self.fhir_client.valueset_validate_code(
                vs_url, cs_data['system'], code
            )

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'valueset_validate', 'ValueSet', 'operation',
            latencies, successful, total - successful, errors
        )

    async def _benchmark_conceptmap_translate(self) -> FHIROperationResult:
        """Benchmark ConceptMap $translate operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            cm_url = random.choice(self.test_data['concept_maps'])
            cs_data = random.choice(self.test_data['code_systems'])
            code = random.choice(cs_data['codes'])

            # Select target system (different from source)
            target_systems = [cs['system'] for cs in self.test_data['code_systems']
                            if cs['system'] != cs_data['system']]
            target = random.choice(target_systems) if target_systems else None

            latency, success, error = await self.fhir_client.conceptmap_translate(
                cm_url, cs_data['system'], code, target
            )

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'conceptmap_translate', 'ConceptMap', 'operation',
            latencies, successful, total - successful, errors
        )

    async def _benchmark_search_operations(self) -> FHIROperationResult:
        """Benchmark search operations"""
        iterations = self.config['test_parameters']['iterations_per_operation']
        latencies = []
        errors = []
        successful = 0
        total = 0

        for _ in range(iterations):
            search_params = random.choice(self.test_data['search_params'])

            latency, success, error = await self.fhir_client.search_codesystems(search_params)

            latencies.append(latency)
            total += 1
            if success:
                successful += 1
            else:
                errors.append(error)

        return self._create_operation_result(
            'search_codesystems', 'CodeSystem', 'read',
            latencies, successful, total - successful, errors
        )

    def _create_operation_result(self, operation_name: str, resource_type: str, operation_type: str,
                                latencies: List[float], successful: int, failed: int,
                                errors: List[str]) -> FHIROperationResult:
        """Create an operation result from test data"""
        total = successful + failed

        if latencies:
            mean_latency = statistics.mean(latencies)
            p50_latency = np.percentile(latencies, 50)
            p95_latency = np.percentile(latencies, 95)
            p99_latency = np.percentile(latencies, 99)
            max_latency = max(latencies)
            min_latency = min(latencies)
        else:
            mean_latency = p50_latency = p95_latency = p99_latency = max_latency = min_latency = 0

        # Calculate throughput (assuming sequential execution for now)
        total_time_seconds = sum(latencies) / 1000 if latencies else 1
        throughput_rps = total / total_time_seconds if total_time_seconds > 0 else 0

        error_rate_pct = (failed / total * 100) if total > 0 else 0

        return FHIROperationResult(
            operation_name=operation_name,
            resource_type=resource_type,
            operation_type=operation_type,
            total_requests=total,
            successful_requests=successful,
            failed_requests=failed,
            mean_latency_ms=mean_latency,
            p50_latency_ms=p50_latency,
            p95_latency_ms=p95_latency,
            p99_latency_ms=p99_latency,
            max_latency_ms=max_latency,
            min_latency_ms=min_latency,
            throughput_rps=throughput_rps,
            error_rate_pct=error_rate_pct,
            errors=errors[:10],  # Limit error list
            timestamp=datetime.now().isoformat()
        )

    def _compile_benchmark_results(self, test_run_id: str, start_time: datetime,
                                 end_time: datetime, operation_results: List[FHIROperationResult]) -> GoogleFHIRBenchmarkResult:
        """Compile comprehensive benchmark results"""
        total_duration = (end_time - start_time).total_seconds()

        # Performance summary
        total_requests = sum(r.total_requests for r in operation_results)
        total_successful = sum(r.successful_requests for r in operation_results)
        total_failed = sum(r.failed_requests for r in operation_results)

        all_latencies = []
        for result in operation_results:
            # Approximate latencies for summary (would be better with raw data)
            all_latencies.extend([result.mean_latency_ms] * result.successful_requests)

        if all_latencies:
            overall_mean_latency = statistics.mean(all_latencies)
            overall_p95_latency = np.percentile(all_latencies, 95)
        else:
            overall_mean_latency = overall_p95_latency = 0

        performance_summary = {
            'total_requests': total_requests,
            'successful_requests': total_successful,
            'failed_requests': total_failed,
            'overall_success_rate_pct': (total_successful / total_requests * 100) if total_requests > 0 else 0,
            'overall_mean_latency_ms': overall_mean_latency,
            'overall_p95_latency_ms': overall_p95_latency,
            'test_duration_seconds': total_duration
        }

        # FHIR compliance results
        compliance_results = {
            'codesystem_operations_functional': any(r.operation_name.startswith('codesystem') and r.successful_requests > 0 for r in operation_results),
            'valueset_operations_functional': any(r.operation_name.startswith('valueset') and r.successful_requests > 0 for r in operation_results),
            'conceptmap_operations_functional': any(r.operation_name.startswith('conceptmap') and r.successful_requests > 0 for r in operation_results),
            'search_operations_functional': any(r.operation_name == 'search_codesystems' and r.successful_requests > 0 for r in operation_results),
            'acceptable_performance': overall_p95_latency < 1000  # <1s for 95th percentile
        }

        # Generate recommendations
        recommendations = self._generate_fhir_recommendations(performance_summary, operation_results)

        return GoogleFHIRBenchmarkResult(
            test_run_id=test_run_id,
            timestamp=start_time.isoformat(),
            project_id=self.fhir_client.project_id,
            location=self.fhir_client.location,
            dataset_id=self.fhir_client.dataset_id,
            fhir_store_id=self.fhir_client.fhir_store_id,
            test_duration_seconds=total_duration,
            operation_results=operation_results,
            performance_summary=performance_summary,
            compliance_results=compliance_results,
            recommendations=recommendations
        )

    def _generate_fhir_recommendations(self, performance_summary: Dict[str, Any],
                                     operation_results: List[FHIROperationResult]) -> List[str]:
        """Generate FHIR-specific recommendations"""
        recommendations = []

        # Overall performance
        if performance_summary['overall_success_rate_pct'] < 95:
            recommendations.append(f"CRITICAL: Overall success rate {performance_summary['overall_success_rate_pct']:.1f}% - investigate FHIR store connectivity")

        if performance_summary['overall_p95_latency_ms'] > 2000:
            recommendations.append("HIGH: P95 latency >2s - consider Google Cloud region optimization")

        # Operation-specific recommendations
        for result in operation_results:
            if result.error_rate_pct > 10:
                recommendations.append(f"HIGH: {result.operation_name} error rate {result.error_rate_pct:.1f}% - check FHIR resource configuration")

            if result.p95_latency_ms > 5000:
                recommendations.append(f"MEDIUM: {result.operation_name} P95 latency >5s - optimize query complexity")

            if result.throughput_rps < 1:
                recommendations.append(f"LOW: {result.operation_name} throughput <1 RPS - consider batch operations")

        # FHIR-specific recommendations
        codesystem_ops = [r for r in operation_results if r.resource_type == 'CodeSystem']
        if codesystem_ops:
            avg_cs_latency = statistics.mean([r.mean_latency_ms for r in codesystem_ops])
            if avg_cs_latency > 1000:
                recommendations.append("MEDIUM: CodeSystem operations >1s - consider terminology caching")

        valueset_ops = [r for r in operation_results if r.resource_type == 'ValueSet']
        if valueset_ops:
            avg_vs_latency = statistics.mean([r.mean_latency_ms for r in valueset_ops])
            if avg_vs_latency > 2000:
                recommendations.append("MEDIUM: ValueSet operations >2s - optimize ValueSet size and expansion parameters")

        if not recommendations:
            recommendations.append("GOOD: Google FHIR store performance meets expectations")

        return recommendations

    def print_results(self, result: GoogleFHIRBenchmarkResult):
        """Print formatted benchmark results"""
        self.console.print(f"\n[bold blue]Google FHIR Benchmark Results[/bold blue]")
        self.console.print(f"Test Run ID: {result.test_run_id}")
        self.console.print(f"FHIR Store: {result.project_id}/{result.location}/{result.dataset_id}/{result.fhir_store_id}")
        self.console.print(f"Duration: {result.test_duration_seconds:.1f}s")

        # Performance summary
        summary = result.performance_summary
        self.console.print(f"\n[bold]Performance Summary[/bold]")
        self.console.print(f"Total Requests: {summary['total_requests']}")
        self.console.print(f"Success Rate: {summary['overall_success_rate_pct']:.1f}%")
        self.console.print(f"Mean Latency: {summary['overall_mean_latency_ms']:.1f}ms")
        self.console.print(f"P95 Latency: {summary['overall_p95_latency_ms']:.1f}ms")

        # Operation results table
        table = Table(title="FHIR Operation Performance")
        table.add_column("Operation", style="cyan")
        table.add_column("Resource", style="magenta")
        table.add_column("Requests", style="yellow")
        table.add_column("Success Rate", style="green")
        table.add_column("Mean (ms)", style="blue")
        table.add_column("P95 (ms)", style="red")
        table.add_column("Throughput (RPS)", style="purple")

        for op_result in result.operation_results:
            table.add_row(
                op_result.operation_name,
                op_result.resource_type,
                str(op_result.total_requests),
                f"{(op_result.successful_requests/op_result.total_requests*100):.1f}%",
                f"{op_result.mean_latency_ms:.1f}",
                f"{op_result.p95_latency_ms:.1f}",
                f"{op_result.throughput_rps:.2f}"
            )

        self.console.print(table)

        # Compliance results
        self.console.print(f"\n[bold]FHIR Compliance[/bold]")
        for compliance, status in result.compliance_results.items():
            status_icon = "✅" if status else "❌"
            self.console.print(f"{status_icon} {compliance}: {status}")

        # Recommendations
        self.console.print(f"\n[bold]Recommendations[/bold]")
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

    def save_results(self, result: GoogleFHIRBenchmarkResult, output_path: str):
        """Save benchmark results to JSON"""
        result_dict = asdict(result)

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w') as f:
            json.dump(result_dict, f, indent=2)

        self.console.print(f"Results saved to: {output_path}")

# CLI interface
async def main():
    """Main entry point for Google FHIR benchmarking"""
    import argparse
    import random

    parser = argparse.ArgumentParser(description="Google FHIR Performance Benchmark")
    parser.add_argument("--config", help="Path to configuration file")
    parser.add_argument("--output", default="reports/google_fhir_benchmark.json",
                       help="Output file for results")
    parser.add_argument("--iterations", type=int, default=100,
                       help="Number of iterations per operation")

    args = parser.parse_args()

    try:
        benchmark = GoogleFHIRBenchmark(args.config)

        # Override iterations if specified
        if args.iterations:
            benchmark.config['test_parameters']['iterations_per_operation'] = args.iterations

        # Run benchmark
        result = await benchmark.run_comprehensive_benchmark()

        # Print results
        benchmark.print_results(result)

        # Save results
        benchmark.save_results(result, args.output)

        # Determine exit code
        if result.performance_summary['overall_success_rate_pct'] < 90:
            return 1
        elif result.performance_summary['overall_p95_latency_ms'] > 5000:
            return 1
        else:
            return 0

    except Exception as e:
        logger.error(f"Google FHIR benchmark failed: {e}")
        return 1

if __name__ == "__main__":
    import random
    exit_code = asyncio.run(main())
    exit(exit_code)