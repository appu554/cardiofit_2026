#!/usr/bin/env python3
"""
Comprehensive Test Orchestrator for KB7 Terminology Phase 3.5
=============================================================

Master test runner that orchestrates all Phase 3.5 validation tests:
- Phase 3.5 success criteria validation
- Performance testing with realistic workloads
- Load testing with concurrent users
- Google FHIR integration validation
- Comprehensive reporting and CI/CD integration

Author: Claude Code Performance Engineer
Version: 1.0.0
"""

import asyncio
import json
import os
import sys
import time
import traceback
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
import logging
import subprocess
import argparse

from rich.console import Console
from rich.table import Table
from rich.progress import Progress, TaskID
from rich.panel import Panel
from rich.layout import Layout
from rich.live import Live
from rich.text import Text
import yaml

# Import our test modules
from phase35_validation import Phase35Validator
from performance_tests import PerformanceTestOrchestrator
from load_testing import LoadTestRunner
from google_fhir_benchmark import GoogleFHIRBenchmark

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('test_orchestrator.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

@dataclass
class TestSuiteConfig:
    """Configuration for the complete test suite"""
    run_phase35_validation: bool = True
    run_performance_tests: bool = True
    run_load_tests: bool = True
    run_google_fhir_tests: bool = True

    # Test parameters
    performance_workload: str = "realistic_mixed"
    performance_duration: int = 300
    load_test_scenario: str = "stress_test"
    google_fhir_iterations: int = 100

    # Parallel execution
    parallel_execution: bool = True
    max_concurrent_tests: int = 2

    # Reporting
    generate_html_report: bool = True
    generate_json_report: bool = True
    generate_junit_report: bool = True

    # CI/CD integration
    fail_on_criteria_failure: bool = True
    fail_on_performance_degradation: bool = True
    performance_threshold_p95_ms: float = 200.0

    # Output paths
    output_directory: str = "reports"
    config_file: Optional[str] = None

@dataclass
class TestResult:
    """Individual test result"""
    test_name: str
    test_type: str
    status: str  # 'PASS', 'FAIL', 'ERROR', 'SKIP'
    duration_seconds: float
    start_time: str
    end_time: str
    details: Dict[str, Any]
    error_message: Optional[str] = None
    recommendations: List[str] = None

@dataclass
class TestSuiteResult:
    """Complete test suite result"""
    suite_run_id: str
    start_time: str
    end_time: str
    total_duration_seconds: float
    test_results: List[TestResult]
    overall_status: str
    summary_statistics: Dict[str, Any]
    critical_failures: List[str]
    recommendations: List[str]
    environment_info: Dict[str, Any]

class TestOrchestrator:
    """Main test orchestration class"""

    def __init__(self, config: TestSuiteConfig):
        self.config = config
        self.console = Console()
        self.test_results: List[TestResult] = []

        # Ensure output directory exists
        os.makedirs(self.config.output_directory, exist_ok=True)

    async def run_complete_test_suite(self) -> TestSuiteResult:
        """Run the complete KB7 Phase 3.5 test suite"""
        suite_run_id = f"kb7_phase35_suite_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        start_time = datetime.now()

        self.console.print(Panel.fit(
            f"[bold blue]KB7 Terminology Phase 3.5 Complete Test Suite[/bold blue]\n"
            f"Suite ID: {suite_run_id}\n"
            f"Start Time: {start_time.isoformat()}",
            border_style="blue"
        ))

        # Pre-flight checks
        await self._run_preflight_checks()

        # Execute test phases
        if self.config.parallel_execution:
            await self._run_tests_parallel()
        else:
            await self._run_tests_sequential()

        end_time = datetime.now()
        total_duration = (end_time - start_time).total_seconds()

        # Compile results
        suite_result = self._compile_suite_results(
            suite_run_id, start_time, end_time, total_duration
        )

        # Generate reports
        await self._generate_reports(suite_result)

        # Print summary
        self._print_suite_summary(suite_result)

        return suite_result

    async def _run_preflight_checks(self):
        """Run preflight checks to ensure system readiness"""
        self.console.print("[yellow]Running preflight checks...[/yellow]")

        checks = [
            ("PostgreSQL Connection", self._check_postgresql),
            ("Redis Connection", self._check_redis),
            ("Neo4j Connection", self._check_neo4j),
            ("Query Router Health", self._check_query_router),
            ("FHIR Service Health", self._check_fhir_service),
        ]

        for check_name, check_func in checks:
            try:
                await check_func()
                self.console.print(f"✅ {check_name}")
            except Exception as e:
                self.console.print(f"❌ {check_name}: {e}")
                logger.warning(f"Preflight check failed: {check_name} - {e}")

    async def _check_postgresql(self):
        """Check PostgreSQL connectivity"""
        import psycopg2
        conn = psycopg2.connect(
            host='localhost',
            port=5433,
            database='terminology_db',
            user='kb7_user',
            password='kb7_password'
        )
        conn.close()

    async def _check_redis(self):
        """Check Redis connectivity"""
        import redis
        r = redis.Redis(host='localhost', port=6380, db=0)
        r.ping()
        r.close()

    async def _check_neo4j(self):
        """Check Neo4j connectivity"""
        from neo4j import GraphDatabase
        driver = GraphDatabase.driver(
            'bolt://localhost:7687',
            auth=('neo4j', 'kb7_neo4j')
        )
        with driver.session() as session:
            session.run("RETURN 1")
        driver.close()

    async def _check_query_router(self):
        """Check query router health"""
        import httpx
        async with httpx.AsyncClient() as client:
            response = await client.get("http://localhost:8090/health", timeout=10.0)
            if response.status_code != 200:
                raise Exception(f"Query router returned {response.status_code}")

    async def _check_fhir_service(self):
        """Check FHIR service health"""
        import httpx
        async with httpx.AsyncClient() as client:
            response = await client.get("http://localhost:8014/health", timeout=10.0)
            if response.status_code != 200:
                raise Exception(f"FHIR service returned {response.status_code}")

    async def _run_tests_sequential(self):
        """Run tests sequentially"""
        with Progress() as progress:
            total_tests = sum([
                self.config.run_phase35_validation,
                self.config.run_performance_tests,
                self.config.run_load_tests,
                self.config.run_google_fhir_tests
            ])

            main_task = progress.add_task("[green]Running test suite...", total=total_tests)

            if self.config.run_phase35_validation:
                progress.update(main_task, description="Phase 3.5 Validation...")
                await self._run_phase35_validation()
                progress.advance(main_task)

            if self.config.run_performance_tests:
                progress.update(main_task, description="Performance Testing...")
                await self._run_performance_tests()
                progress.advance(main_task)

            if self.config.run_load_tests:
                progress.update(main_task, description="Load Testing...")
                await self._run_load_tests()
                progress.advance(main_task)

            if self.config.run_google_fhir_tests:
                progress.update(main_task, description="Google FHIR Testing...")
                await self._run_google_fhir_tests()
                progress.advance(main_task)

    async def _run_tests_parallel(self):
        """Run tests in parallel where possible"""
        tasks = []

        if self.config.run_phase35_validation:
            tasks.append(self._run_phase35_validation())

        if self.config.run_performance_tests:
            tasks.append(self._run_performance_tests())

        if self.config.run_load_tests:
            tasks.append(self._run_load_tests())

        if self.config.run_google_fhir_tests:
            tasks.append(self._run_google_fhir_tests())

        # Run tasks with limited concurrency
        semaphore = asyncio.Semaphore(self.config.max_concurrent_tests)

        async def run_with_semaphore(task):
            async with semaphore:
                return await task

        await asyncio.gather(*[run_with_semaphore(task) for task in tasks], return_exceptions=True)

    async def _run_phase35_validation(self):
        """Run Phase 3.5 validation tests"""
        test_name = "Phase 3.5 Success Criteria Validation"
        start_time = datetime.now()

        try:
            self.console.print(f"[blue]Starting {test_name}...[/blue]")

            validator = Phase35Validator(self.config.config_file)
            report = await validator.run_full_validation()
            validator.close()

            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            # Determine status
            status = "PASS" if report.overall_status == "PASS" else "FAIL"

            # Save detailed report
            report_path = os.path.join(self.config.output_directory, "phase35_validation.json")
            with open(report_path, 'w') as f:
                json.dump(asdict(report), f, indent=2)

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="validation",
                status=status,
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={
                    'report_path': report_path,
                    'success_rate': report.performance_summary['success_rate_pct'],
                    'failed_criteria': [r.criterion for r in report.success_criteria_results if not r.passed]
                },
                recommendations=report.recommendations
            ))

            self.console.print(f"✅ {test_name} completed: {status}")

        except Exception as e:
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="validation",
                status="ERROR",
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={},
                error_message=str(e)
            ))

            self.console.print(f"❌ {test_name} failed: {e}")
            logger.error(f"Phase 3.5 validation failed: {e}\n{traceback.format_exc()}")

    async def _run_performance_tests(self):
        """Run performance tests"""
        test_name = "Performance Testing"
        start_time = datetime.now()

        try:
            self.console.print(f"[blue]Starting {test_name}...[/blue]")

            orchestrator = PerformanceTestOrchestrator(self.config.config_file)
            await orchestrator.initialize_all_testers()

            results = await orchestrator.run_workload_test(
                self.config.performance_workload,
                self.config.performance_duration,
                concurrent_users=10
            )

            orchestrator.cleanup()
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            # Determine status based on performance criteria
            max_p95 = max(r.p95_latency_ms for r in results)
            min_score = min(r.overall_score for r in results)

            status = "PASS" if max_p95 < self.config.performance_threshold_p95_ms and min_score > 70 else "FAIL"

            # Save detailed results
            results_path = os.path.join(self.config.output_directory, "performance_results.json")
            with open(results_path, 'w') as f:
                json.dump([asdict(r) for r in results], f, indent=2)

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="performance",
                status=status,
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={
                    'results_path': results_path,
                    'workload': self.config.performance_workload,
                    'max_p95_latency_ms': max_p95,
                    'min_score': min_score,
                    'total_tests': len(results)
                }
            ))

            self.console.print(f"✅ {test_name} completed: {status}")

        except Exception as e:
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="performance",
                status="ERROR",
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={},
                error_message=str(e)
            ))

            self.console.print(f"❌ {test_name} failed: {e}")
            logger.error(f"Performance testing failed: {e}\n{traceback.format_exc()}")

    async def _run_load_tests(self):
        """Run load tests"""
        test_name = "Load Testing"
        start_time = datetime.now()

        try:
            self.console.print(f"[blue]Starting {test_name}...[/blue]")

            runner = LoadTestRunner(self.config.config_file)
            result = await runner.run_load_test(self.config.load_test_scenario)
            runner.cleanup()

            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            # Determine status based on load test results
            status = "PASS" if (
                result.performance_summary['error_rate_pct'] < 5 and
                result.performance_summary['p95_latency_ms'] < 1000
            ) else "FAIL"

            # Save detailed results
            results_path = os.path.join(self.config.output_directory, "load_test_results.json")
            with open(results_path, 'w') as f:
                json.dump(asdict(result), f, indent=2)

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="load",
                status=status,
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={
                    'results_path': results_path,
                    'scenario': self.config.load_test_scenario,
                    'peak_users': result.peak_users,
                    'peak_rps': result.peak_rps,
                    'error_rate_pct': result.performance_summary['error_rate_pct']
                },
                recommendations=result.recommendations
            ))

            self.console.print(f"✅ {test_name} completed: {status}")

        except Exception as e:
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="load",
                status="ERROR",
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={},
                error_message=str(e)
            ))

            self.console.print(f"❌ {test_name} failed: {e}")
            logger.error(f"Load testing failed: {e}\n{traceback.format_exc()}")

    async def _run_google_fhir_tests(self):
        """Run Google FHIR tests"""
        test_name = "Google FHIR Integration"
        start_time = datetime.now()

        try:
            self.console.print(f"[blue]Starting {test_name}...[/blue]")

            benchmark = GoogleFHIRBenchmark(self.config.config_file)

            # Override iterations if specified
            benchmark.config['test_parameters']['iterations_per_operation'] = self.config.google_fhir_iterations

            result = await benchmark.run_comprehensive_benchmark()

            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            # Determine status based on Google FHIR results
            status = "PASS" if (
                result.performance_summary['overall_success_rate_pct'] > 90 and
                result.performance_summary['overall_p95_latency_ms'] < 5000
            ) else "FAIL"

            # Save detailed results
            results_path = os.path.join(self.config.output_directory, "google_fhir_results.json")
            with open(results_path, 'w') as f:
                json.dump(asdict(result), f, indent=2)

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="fhir",
                status=status,
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={
                    'results_path': results_path,
                    'fhir_store': f"{result.project_id}/{result.dataset_id}/{result.fhir_store_id}",
                    'success_rate_pct': result.performance_summary['overall_success_rate_pct'],
                    'p95_latency_ms': result.performance_summary['overall_p95_latency_ms']
                },
                recommendations=result.recommendations
            ))

            self.console.print(f"✅ {test_name} completed: {status}")

        except Exception as e:
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()

            self.test_results.append(TestResult(
                test_name=test_name,
                test_type="fhir",
                status="ERROR",
                duration_seconds=duration,
                start_time=start_time.isoformat(),
                end_time=end_time.isoformat(),
                details={},
                error_message=str(e)
            ))

            self.console.print(f"❌ {test_name} failed: {e}")
            logger.error(f"Google FHIR testing failed: {e}\n{traceback.format_exc()}")

    def _compile_suite_results(self, suite_run_id: str, start_time: datetime,
                              end_time: datetime, total_duration: float) -> TestSuiteResult:
        """Compile complete suite results"""

        # Calculate overall status
        statuses = [r.status for r in self.test_results]
        if "ERROR" in statuses:
            overall_status = "ERROR"
        elif "FAIL" in statuses:
            overall_status = "FAIL"
        else:
            overall_status = "PASS"

        # Summary statistics
        total_tests = len(self.test_results)
        passed_tests = len([r for r in self.test_results if r.status == "PASS"])
        failed_tests = len([r for r in self.test_results if r.status == "FAIL"])
        error_tests = len([r for r in self.test_results if r.status == "ERROR"])

        summary_statistics = {
            'total_tests': total_tests,
            'passed_tests': passed_tests,
            'failed_tests': failed_tests,
            'error_tests': error_tests,
            'success_rate_pct': (passed_tests / total_tests * 100) if total_tests > 0 else 0,
            'total_duration_seconds': total_duration,
            'average_test_duration_seconds': total_duration / total_tests if total_tests > 0 else 0
        }

        # Critical failures
        critical_failures = []
        for result in self.test_results:
            if result.status in ["FAIL", "ERROR"]:
                if result.test_type == "validation":
                    critical_failures.append(f"CRITICAL: Phase 3.5 validation failed - {result.error_message or 'Success criteria not met'}")
                elif result.test_type == "performance" and self.config.fail_on_performance_degradation:
                    critical_failures.append(f"HIGH: Performance degradation detected in {result.test_name}")
                else:
                    critical_failures.append(f"MEDIUM: {result.test_name} {result.status}")

        # Aggregate recommendations
        recommendations = []
        for result in self.test_results:
            if result.recommendations:
                recommendations.extend(result.recommendations)

        # Environment info
        environment_info = {
            'python_version': sys.version,
            'platform': sys.platform,
            'test_config': asdict(self.config),
            'timestamp': datetime.now().isoformat()
        }

        return TestSuiteResult(
            suite_run_id=suite_run_id,
            start_time=start_time.isoformat(),
            end_time=end_time.isoformat(),
            total_duration_seconds=total_duration,
            test_results=self.test_results,
            overall_status=overall_status,
            summary_statistics=summary_statistics,
            critical_failures=critical_failures,
            recommendations=recommendations,
            environment_info=environment_info
        )

    async def _generate_reports(self, suite_result: TestSuiteResult):
        """Generate various test reports"""

        # JSON Report
        if self.config.generate_json_report:
            json_path = os.path.join(self.config.output_directory, "test_suite_results.json")
            with open(json_path, 'w') as f:
                json.dump(asdict(suite_result), f, indent=2)
            self.console.print(f"📄 JSON report saved: {json_path}")

        # HTML Report
        if self.config.generate_html_report:
            html_path = await self._generate_html_report(suite_result)
            self.console.print(f"🌐 HTML report saved: {html_path}")

        # JUnit XML Report (for CI/CD)
        if self.config.generate_junit_report:
            junit_path = self._generate_junit_report(suite_result)
            self.console.print(f"📋 JUnit report saved: {junit_path}")

    async def _generate_html_report(self, suite_result: TestSuiteResult) -> str:
        """Generate HTML report"""
        html_template = """
<!DOCTYPE html>
<html>
<head>
    <title>KB7 Phase 3.5 Test Suite Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f4f4f4; padding: 20px; border-radius: 8px; }
        .status-pass { color: #28a745; }
        .status-fail { color: #dc3545; }
        .status-error { color: #fd7e14; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #f8f9fa; }
        .recommendations { background: #fff3cd; padding: 15px; border-radius: 5px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>KB7 Terminology Phase 3.5 Test Suite Report</h1>
        <p><strong>Suite ID:</strong> {suite_id}</p>
        <p><strong>Status:</strong> <span class="status-{status_class}">{overall_status}</span></p>
        <p><strong>Duration:</strong> {duration:.1f} seconds</p>
        <p><strong>Success Rate:</strong> {success_rate:.1f}%</p>
    </div>

    <h2>Test Results Summary</h2>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Type</th>
            <th>Status</th>
            <th>Duration (s)</th>
            <th>Details</th>
        </tr>
        {test_rows}
    </table>

    <h2>Critical Failures</h2>
    <div class="recommendations">
        {critical_failures}
    </div>

    <h2>Recommendations</h2>
    <div class="recommendations">
        {recommendations}
    </div>

    <h2>Environment Information</h2>
    <pre>{environment_info}</pre>
</body>
</html>
        """

        # Generate test rows
        test_rows = ""
        for result in suite_result.test_results:
            status_class = result.status.lower()
            details = f"Duration: {result.duration_seconds:.1f}s"
            if result.error_message:
                details += f"<br>Error: {result.error_message}"

            test_rows += f"""
            <tr>
                <td>{result.test_name}</td>
                <td>{result.test_type}</td>
                <td><span class="status-{status_class}">{result.status}</span></td>
                <td>{result.duration_seconds:.1f}</td>
                <td>{details}</td>
            </tr>
            """

        # Format lists
        critical_failures = "<br>".join(suite_result.critical_failures) if suite_result.critical_failures else "None"
        recommendations = "<br>".join(suite_result.recommendations[:10]) if suite_result.recommendations else "None"

        html_content = html_template.format(
            suite_id=suite_result.suite_run_id,
            overall_status=suite_result.overall_status,
            status_class=suite_result.overall_status.lower(),
            duration=suite_result.total_duration_seconds,
            success_rate=suite_result.summary_statistics['success_rate_pct'],
            test_rows=test_rows,
            critical_failures=critical_failures,
            recommendations=recommendations,
            environment_info=json.dumps(suite_result.environment_info, indent=2)
        )

        html_path = os.path.join(self.config.output_directory, "test_suite_report.html")
        with open(html_path, 'w') as f:
            f.write(html_content)

        return html_path

    def _generate_junit_report(self, suite_result: TestSuiteResult) -> str:
        """Generate JUnit XML report for CI/CD integration"""
        junit_template = """<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="KB7_Phase35_Suite" tests="{total_tests}" failures="{failures}" errors="{errors}" time="{total_time}">
    <testsuite name="KB7_Terminology_Phase35" tests="{total_tests}" failures="{failures}" errors="{errors}" time="{total_time}">
        {test_cases}
    </testsuite>
</testsuites>"""

        test_cases = ""
        for result in suite_result.test_results:
            if result.status == "PASS":
                test_cases += f'        <testcase classname="KB7.Phase35" name="{result.test_name}" time="{result.duration_seconds:.3f}"/>\n'
            elif result.status == "FAIL":
                test_cases += f"""        <testcase classname="KB7.Phase35" name="{result.test_name}" time="{result.duration_seconds:.3f}">
            <failure message="Test failed">{result.error_message or "Test criteria not met"}</failure>
        </testcase>
"""
            elif result.status == "ERROR":
                test_cases += f"""        <testcase classname="KB7.Phase35" name="{result.test_name}" time="{result.duration_seconds:.3f}">
            <error message="Test error">{result.error_message or "Unexpected error"}</error>
        </testcase>
"""

        failures = len([r for r in suite_result.test_results if r.status == "FAIL"])
        errors = len([r for r in suite_result.test_results if r.status == "ERROR"])

        junit_content = junit_template.format(
            total_tests=suite_result.summary_statistics['total_tests'],
            failures=failures,
            errors=errors,
            total_time=suite_result.total_duration_seconds,
            test_cases=test_cases
        )

        junit_path = os.path.join(self.config.output_directory, "junit_results.xml")
        with open(junit_path, 'w') as f:
            f.write(junit_content)

        return junit_path

    def _print_suite_summary(self, suite_result: TestSuiteResult):
        """Print comprehensive suite summary"""
        self.console.print("\n" + "="*80)
        self.console.print(f"[bold blue]KB7 Phase 3.5 Test Suite Complete[/bold blue]")
        self.console.print("="*80)

        # Status panel
        status_color = "green" if suite_result.overall_status == "PASS" else "red"
        self.console.print(Panel.fit(
            f"[bold {status_color}]Overall Status: {suite_result.overall_status}[/bold {status_color}]\n"
            f"Duration: {suite_result.total_duration_seconds:.1f} seconds\n"
            f"Success Rate: {suite_result.summary_statistics['success_rate_pct']:.1f}%",
            border_style=status_color
        ))

        # Test results table
        table = Table(title="Test Results Summary")
        table.add_column("Test Name", style="cyan")
        table.add_column("Type", style="magenta")
        table.add_column("Status", style="bold")
        table.add_column("Duration", style="yellow")

        for result in suite_result.test_results:
            status_style = "green" if result.status == "PASS" else "red"
            table.add_row(
                result.test_name,
                result.test_type,
                f"[{status_style}]{result.status}[/{status_style}]",
                f"{result.duration_seconds:.1f}s"
            )

        self.console.print(table)

        # Critical failures
        if suite_result.critical_failures:
            self.console.print("\n[bold red]Critical Failures:[/bold red]")
            for failure in suite_result.critical_failures:
                self.console.print(f"• [red]{failure}[/red]")

        # Top recommendations
        if suite_result.recommendations:
            self.console.print("\n[bold yellow]Top Recommendations:[/bold yellow]")
            for i, rec in enumerate(suite_result.recommendations[:5], 1):
                self.console.print(f"{i}. {rec}")

# CLI interface
def main():
    """Main entry point for test orchestrator"""
    parser = argparse.ArgumentParser(description="KB7 Terminology Phase 3.5 Complete Test Suite")

    # Test selection
    parser.add_argument("--skip-validation", action="store_true", help="Skip Phase 3.5 validation tests")
    parser.add_argument("--skip-performance", action="store_true", help="Skip performance tests")
    parser.add_argument("--skip-load", action="store_true", help="Skip load tests")
    parser.add_argument("--skip-google-fhir", action="store_true", help="Skip Google FHIR tests")

    # Test parameters
    parser.add_argument("--performance-workload", default="realistic_mixed",
                       choices=['realistic_mixed', 'heavy_reasoning', 'fhir_heavy', 'cache_optimized'],
                       help="Performance test workload")
    parser.add_argument("--load-scenario", default="stress_test",
                       choices=['stress_test', 'soak_test', 'spike_test', 'capacity_test'],
                       help="Load test scenario")
    parser.add_argument("--google-fhir-iterations", type=int, default=100,
                       help="Google FHIR test iterations")

    # Execution control
    parser.add_argument("--sequential", action="store_true", help="Run tests sequentially")
    parser.add_argument("--max-concurrent", type=int, default=2, help="Max concurrent tests")

    # Reporting
    parser.add_argument("--output-dir", default="reports", help="Output directory for reports")
    parser.add_argument("--no-html", action="store_true", help="Skip HTML report generation")
    parser.add_argument("--no-junit", action="store_true", help="Skip JUnit report generation")

    # CI/CD integration
    parser.add_argument("--fail-fast", action="store_true", help="Fail on first test failure")
    parser.add_argument("--performance-threshold", type=float, default=200.0,
                       help="Performance threshold (P95 latency in ms)")

    # Configuration
    parser.add_argument("--config", help="Path to configuration file")

    args = parser.parse_args()

    # Create test suite configuration
    config = TestSuiteConfig(
        run_phase35_validation=not args.skip_validation,
        run_performance_tests=not args.skip_performance,
        run_load_tests=not args.skip_load,
        run_google_fhir_tests=not args.skip_google_fhir,
        performance_workload=args.performance_workload,
        load_test_scenario=args.load_scenario,
        google_fhir_iterations=args.google_fhir_iterations,
        parallel_execution=not args.sequential,
        max_concurrent_tests=args.max_concurrent,
        output_directory=args.output_dir,
        generate_html_report=not args.no_html,
        generate_junit_report=not args.no_junit,
        performance_threshold_p95_ms=args.performance_threshold,
        config_file=args.config
    )

    async def run_suite():
        orchestrator = TestOrchestrator(config)
        return await orchestrator.run_complete_test_suite()

    try:
        # Run the test suite
        suite_result = asyncio.run(run_suite())

        # Determine exit code
        if suite_result.overall_status == "PASS":
            return 0
        elif config.fail_on_criteria_failure and suite_result.overall_status in ["FAIL", "ERROR"]:
            return 1
        else:
            return 0

    except KeyboardInterrupt:
        print("\nTest suite interrupted by user")
        return 130
    except Exception as e:
        print(f"Test suite failed with error: {e}")
        logger.error(f"Test suite error: {e}\n{traceback.format_exc()}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)