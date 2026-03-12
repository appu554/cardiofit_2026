#!/usr/bin/env python3
"""
Cross-Service Integration Tests
Tests interactions between all KB services to validate end-to-end workflows
"""

import asyncio
import aiohttp
import json
import time
import logging
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from datetime import datetime, timedelta
import pytest
from concurrent.futures import ThreadPoolExecutor
import os
import sys

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class ServiceEndpoint:
    """Service endpoint configuration"""
    name: str
    base_url: str
    health_endpoint: str = "/health"
    auth_required: bool = False

@dataclass
class TestScenario:
    """Integration test scenario definition"""
    name: str
    description: str
    services: List[str]
    steps: List[Dict[str, Any]]
    expected_outcome: str
    timeout: int = 30

class CrossServiceTestRunner:
    """Comprehensive cross-service integration test runner"""
    
    def __init__(self):
        self.services = {
            "kb-drug-rules": ServiceEndpoint(
                name="KB-1 Drug Rules",
                base_url="http://localhost:8081",
                health_endpoint="/api/v1/health"
            ),
            "kb-clinical-context": ServiceEndpoint(
                name="KB-2 Clinical Context",
                base_url="http://localhost:8082",
                health_endpoint="/api/v1/health"
            ),
            "kb-guideline-evidence": ServiceEndpoint(
                name="KB-3 Guideline Evidence",
                base_url="http://localhost:8084",
                health_endpoint="/api/v1/health"
            ),
            "kb-patient-safety": ServiceEndpoint(
                name="KB-4 Patient Safety",
                base_url="http://localhost:8085",
                health_endpoint="/api/v1/health"
            ),
            "kb-drug-interactions": ServiceEndpoint(
                name="KB-5 Drug Interactions",
                base_url="http://localhost:8086",
                health_endpoint="/api/v1/health"
            ),
            "kb-formulary": ServiceEndpoint(
                name="KB-6 Formulary",
                base_url="http://localhost:8087",
                health_endpoint="/api/v1/health"
            ),
            "kb-terminology": ServiceEndpoint(
                name="KB-7 Terminology",
                base_url="http://localhost:8088",
                health_endpoint="/api/v1/health"
            )
        }
        
        self.scenarios = self._define_test_scenarios()
        self.session = None
        
    def _define_test_scenarios(self) -> List[TestScenario]:
        """Define comprehensive integration test scenarios"""
        return [
            TestScenario(
                name="medication_decision_support_workflow",
                description="Complete medication decision support workflow across all KB services",
                services=["kb-drug-rules", "kb-clinical-context", "kb-guideline-evidence", 
                         "kb-patient-safety", "kb-drug-interactions", "kb-formulary", "kb-terminology"],
                steps=[
                    {
                        "service": "kb-terminology",
                        "endpoint": "/api/v1/terminology/validate",
                        "method": "POST",
                        "payload": {
                            "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                            "code": "161",
                            "display": "Aspirin"
                        },
                        "expected_status": 200,
                        "store_response": "terminology_validation"
                    },
                    {
                        "service": "kb-clinical-context",
                        "endpoint": "/api/v1/context/phenotype-match",
                        "method": "POST",
                        "payload": {
                            "patient_id": "test-patient-123",
                            "conditions": ["I25.10", "E11.9"],  # CAD, Type 2 diabetes
                            "age": 65,
                            "gender": "male"
                        },
                        "expected_status": 200,
                        "store_response": "phenotype_match"
                    },
                    {
                        "service": "kb-guideline-evidence",
                        "endpoint": "/api/v1/recommendations/condition",
                        "method": "GET",
                        "params": {
                            "condition": "coronary artery disease",
                            "grades": ["A", "B"]
                        },
                        "expected_status": 200,
                        "store_response": "guidelines"
                    },
                    {
                        "service": "kb-drug-interactions",
                        "endpoint": "/api/v1/interactions/check",
                        "method": "POST",
                        "payload": {
                            "medications": [
                                {"rxcui": "161", "name": "Aspirin"},
                                {"rxcui": "29046", "name": "Lisinopril"}
                            ]
                        },
                        "expected_status": 200,
                        "store_response": "interactions"
                    },
                    {
                        "service": "kb-patient-safety",
                        "endpoint": "/api/v1/safety/contraindications",
                        "method": "POST",
                        "payload": {
                            "patient_profile": "{{phenotype_match}}",
                            "medication": {"rxcui": "161", "name": "Aspirin"},
                            "allergies": [],
                            "conditions": ["I25.10", "E11.9"]
                        },
                        "expected_status": 200,
                        "store_response": "safety_check"
                    },
                    {
                        "service": "kb-formulary",
                        "endpoint": "/api/v1/formulary/alternatives",
                        "method": "POST",
                        "payload": {
                            "medication": {"rxcui": "161", "name": "Aspirin"},
                            "formulary_id": "standard-formulary-2024",
                            "insurance_tier": "generic"
                        },
                        "expected_status": 200,
                        "store_response": "formulary_alternatives"
                    },
                    {
                        "service": "kb-drug-rules",
                        "endpoint": "/api/v1/dosing/calculate",
                        "method": "POST",
                        "payload": {
                            "medication": {"rxcui": "161", "name": "Aspirin"},
                            "patient": {
                                "age": 65,
                                "weight": 80.0,
                                "creatinine": 1.2,
                                "conditions": ["I25.10", "E11.9"]
                            },
                            "indication": "cardioprotection"
                        },
                        "expected_status": 200,
                        "store_response": "dosing_calculation"
                    }
                ],
                expected_outcome="Complete medication recommendation with dosing, safety checks, and alternatives",
                timeout=60
            ),
            
            TestScenario(
                name="terminology_consistency_check",
                description="Validate terminology consistency across all KB services",
                services=["kb-terminology", "kb-drug-rules", "kb-formulary"],
                steps=[
                    {
                        "service": "kb-terminology",
                        "endpoint": "/api/v1/terminology/lookup",
                        "method": "GET",
                        "params": {"system": "rxnorm", "code": "161"},
                        "expected_status": 200,
                        "store_response": "terminology_lookup"
                    },
                    {
                        "service": "kb-drug-rules",
                        "endpoint": "/api/v1/drugs/by-code",
                        "method": "GET",
                        "params": {"system": "rxnorm", "code": "161"},
                        "expected_status": 200,
                        "store_response": "drug_lookup"
                    },
                    {
                        "service": "kb-formulary",
                        "endpoint": "/api/v1/formulary/search",
                        "method": "GET",
                        "params": {"rxcui": "161"},
                        "expected_status": 200,
                        "store_response": "formulary_lookup"
                    }
                ],
                expected_outcome="Consistent terminology data across services",
                timeout=30
            ),
            
            TestScenario(
                name="clinical_workflow_integration",
                description="Test clinical workflow integration between context, guidelines, and safety",
                services=["kb-clinical-context", "kb-guideline-evidence", "kb-patient-safety"],
                steps=[
                    {
                        "service": "kb-clinical-context",
                        "endpoint": "/api/v1/context/population-cohort",
                        "method": "POST",
                        "payload": {
                            "criteria": {
                                "age_range": {"min": 50, "max": 80},
                                "conditions": ["I25.10"],
                                "phenotypes": ["cardiac_risk_high"]
                            }
                        },
                        "expected_status": 200,
                        "store_response": "cohort_definition"
                    },
                    {
                        "service": "kb-guideline-evidence",
                        "endpoint": "/api/v1/recommendations/population",
                        "method": "POST",
                        "payload": {
                            "population": "{{cohort_definition}}",
                            "condition": "coronary artery disease"
                        },
                        "expected_status": 200,
                        "store_response": "population_guidelines"
                    },
                    {
                        "service": "kb-patient-safety",
                        "endpoint": "/api/v1/safety/population-alerts",
                        "method": "POST",
                        "payload": {
                            "population": "{{cohort_definition}}",
                            "interventions": "{{population_guidelines}}"
                        },
                        "expected_status": 200,
                        "store_response": "population_safety"
                    }
                ],
                expected_outcome="Population-level clinical recommendations with safety considerations",
                timeout=45
            ),
            
            TestScenario(
                name="performance_stress_test",
                description="Test system performance under concurrent load across services",
                services=["kb-terminology", "kb-drug-rules", "kb-formulary"],
                steps=[
                    {
                        "service": "kb-terminology",
                        "endpoint": "/api/v1/terminology/batch-validate",
                        "method": "POST",
                        "payload": {
                            "validations": [
                                {"system": "rxnorm", "code": str(i), "display": f"Test Drug {i}"}
                                for i in range(100, 200)  # 100 items
                            ]
                        },
                        "expected_status": 200,
                        "concurrent_requests": 5,
                        "store_response": "batch_validation"
                    },
                    {
                        "service": "kb-drug-rules",
                        "endpoint": "/api/v1/dosing/batch-calculate",
                        "method": "POST",
                        "payload": {
                            "calculations": [
                                {
                                    "medication": {"rxcui": str(100 + i), "name": f"Test Drug {i}"},
                                    "patient": {"age": 50 + i, "weight": 70.0, "creatinine": 1.0}
                                }
                                for i in range(50)  # 50 items
                            ]
                        },
                        "expected_status": 200,
                        "concurrent_requests": 3,
                        "store_response": "batch_dosing"
                    }
                ],
                expected_outcome="System handles concurrent load without degradation",
                timeout=120
            )
        ]
    
    async def check_service_health(self, service_id: str) -> Dict[str, Any]:
        """Check health status of a service"""
        service = self.services[service_id]
        
        try:
            async with self.session.get(
                f"{service.base_url}{service.health_endpoint}",
                timeout=aiohttp.ClientTimeout(total=10)
            ) as response:
                health_data = await response.json()
                return {
                    "service": service.name,
                    "status": "healthy" if response.status == 200 else "unhealthy",
                    "response_time": None,  # Would need timing
                    "details": health_data
                }
        except Exception as e:
            return {
                "service": service.name,
                "status": "unreachable",
                "error": str(e),
                "details": None
            }
    
    async def execute_step(self, step: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a single test step"""
        service = self.services[step["service"]]
        method = step["method"].upper()
        endpoint = step["endpoint"]
        url = f"{service.base_url}{endpoint}"
        
        # Replace template variables in payload
        payload = step.get("payload", {})
        if payload:
            payload = self._replace_template_vars(payload, context)
        
        params = step.get("params", {})
        concurrent_requests = step.get("concurrent_requests", 1)
        
        start_time = time.time()
        
        try:
            if concurrent_requests == 1:
                # Single request
                if method == "GET":
                    async with self.session.get(url, params=params) as response:
                        result = await self._process_response(response, step)
                elif method == "POST":
                    async with self.session.post(url, json=payload, params=params) as response:
                        result = await self._process_response(response, step)
                else:
                    raise ValueError(f"Unsupported HTTP method: {method}")
            else:
                # Concurrent requests
                tasks = []
                for _ in range(concurrent_requests):
                    if method == "POST":
                        task = self.session.post(url, json=payload, params=params)
                    else:
                        task = self.session.get(url, params=params)
                    tasks.append(task)
                
                responses = await asyncio.gather(*tasks, return_exceptions=True)
                
                # Process the first successful response
                for response in responses:
                    if not isinstance(response, Exception):
                        async with response:
                            result = await self._process_response(response, step)
                            break
                else:
                    raise Exception("All concurrent requests failed")
            
            result["response_time"] = time.time() - start_time
            result["success"] = True
            
            # Store response in context if requested
            if step.get("store_response"):
                context[step["store_response"]] = result.get("data", {})
            
            return result
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
                "response_time": time.time() - start_time,
                "step": step
            }
    
    async def _process_response(self, response: aiohttp.ClientResponse, step: Dict[str, Any]) -> Dict[str, Any]:
        """Process HTTP response"""
        expected_status = step.get("expected_status", 200)
        
        result = {
            "status_code": response.status,
            "expected_status": expected_status,
            "headers": dict(response.headers)
        }
        
        try:
            data = await response.json()
            result["data"] = data
        except:
            text = await response.text()
            result["data"] = text
        
        if response.status != expected_status:
            result["error"] = f"Expected status {expected_status}, got {response.status}"
        
        return result
    
    def _replace_template_vars(self, obj: Any, context: Dict[str, Any]) -> Any:
        """Replace template variables in object with context values"""
        if isinstance(obj, dict):
            return {k: self._replace_template_vars(v, context) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [self._replace_template_vars(item, context) for item in obj]
        elif isinstance(obj, str) and obj.startswith("{{") and obj.endswith("}}"):
            var_name = obj[2:-2]
            return context.get(var_name, obj)
        else:
            return obj
    
    async def run_scenario(self, scenario: TestScenario) -> Dict[str, Any]:
        """Run a complete test scenario"""
        logger.info(f"Starting scenario: {scenario.name}")
        
        start_time = time.time()
        context = {}
        step_results = []
        
        try:
            # Check health of all required services
            health_checks = []
            for service_id in scenario.services:
                health_checks.append(self.check_service_health(service_id))
            
            health_results = await asyncio.gather(*health_checks)
            unhealthy_services = [h for h in health_results if h["status"] != "healthy"]
            
            if unhealthy_services:
                return {
                    "scenario": scenario.name,
                    "success": False,
                    "error": f"Unhealthy services: {[s['service'] for s in unhealthy_services]}",
                    "health_checks": health_results,
                    "execution_time": time.time() - start_time
                }
            
            # Execute all steps
            for i, step in enumerate(scenario.steps):
                logger.info(f"  Step {i+1}/{len(scenario.steps)}: {step.get('service')}{step.get('endpoint')}")
                
                step_result = await asyncio.wait_for(
                    self.execute_step(step, context),
                    timeout=scenario.timeout
                )
                
                step_results.append(step_result)
                
                if not step_result.get("success", False):
                    return {
                        "scenario": scenario.name,
                        "success": False,
                        "error": f"Step {i+1} failed: {step_result.get('error', 'Unknown error')}",
                        "step_results": step_results,
                        "health_checks": health_results,
                        "execution_time": time.time() - start_time
                    }
            
            # Scenario completed successfully
            return {
                "scenario": scenario.name,
                "success": True,
                "description": scenario.description,
                "expected_outcome": scenario.expected_outcome,
                "step_results": step_results,
                "health_checks": health_results,
                "execution_time": time.time() - start_time,
                "context": context
            }
            
        except asyncio.TimeoutError:
            return {
                "scenario": scenario.name,
                "success": False,
                "error": f"Scenario timeout after {scenario.timeout} seconds",
                "step_results": step_results,
                "execution_time": time.time() - start_time
            }
        except Exception as e:
            return {
                "scenario": scenario.name,
                "success": False,
                "error": f"Unexpected error: {str(e)}",
                "step_results": step_results,
                "execution_time": time.time() - start_time
            }
    
    async def run_all_scenarios(self) -> Dict[str, Any]:
        """Run all integration test scenarios"""
        logger.info("Starting cross-service integration tests")
        
        self.session = aiohttp.ClientSession()
        
        try:
            start_time = time.time()
            scenario_results = []
            
            for scenario in self.scenarios:
                result = await self.run_scenario(scenario)
                scenario_results.append(result)
                
                if result["success"]:
                    logger.info(f"✅ {scenario.name} - SUCCESS ({result['execution_time']:.2f}s)")
                else:
                    logger.error(f"❌ {scenario.name} - FAILED: {result.get('error', 'Unknown error')}")
            
            successful_scenarios = [r for r in scenario_results if r["success"]]
            failed_scenarios = [r for r in scenario_results if not r["success"]]
            
            summary = {
                "test_suite": "Cross-Service Integration Tests",
                "start_time": datetime.now().isoformat(),
                "total_execution_time": time.time() - start_time,
                "total_scenarios": len(self.scenarios),
                "successful_scenarios": len(successful_scenarios),
                "failed_scenarios": len(failed_scenarios),
                "success_rate": len(successful_scenarios) / len(self.scenarios) * 100,
                "scenario_results": scenario_results
            }
            
            return summary
            
        finally:
            await self.session.close()
    
    def generate_report(self, results: Dict[str, Any], output_file: str = None) -> str:
        """Generate comprehensive test report"""
        report = []
        
        report.append("# Cross-Service Integration Test Report")
        report.append(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        report.append("")
        
        # Summary
        report.append("## Summary")
        report.append(f"- **Total Scenarios**: {results['total_scenarios']}")
        report.append(f"- **Successful**: {results['successful_scenarios']}")
        report.append(f"- **Failed**: {results['failed_scenarios']}")
        report.append(f"- **Success Rate**: {results['success_rate']:.1f}%")
        report.append(f"- **Total Execution Time**: {results['total_execution_time']:.2f}s")
        report.append("")
        
        # Detailed Results
        report.append("## Detailed Results")
        
        for scenario_result in results["scenario_results"]:
            status = "✅ PASSED" if scenario_result["success"] else "❌ FAILED"
            report.append(f"### {scenario_result['scenario']} {status}")
            report.append(f"**Execution Time**: {scenario_result['execution_time']:.2f}s")
            
            if scenario_result["success"]:
                report.append(f"**Expected Outcome**: {scenario_result.get('expected_outcome', 'N/A')}")
                report.append(f"**Steps Completed**: {len(scenario_result.get('step_results', []))}")
            else:
                report.append(f"**Error**: {scenario_result.get('error', 'Unknown error')}")
            
            report.append("")
        
        # Health Status
        if results["scenario_results"]:
            first_result = results["scenario_results"][0]
            if "health_checks" in first_result:
                report.append("## Service Health Status")
                for health in first_result["health_checks"]:
                    status_icon = "✅" if health["status"] == "healthy" else "❌"
                    report.append(f"- {status_icon} **{health['service']}**: {health['status']}")
                report.append("")
        
        report_text = "\n".join(report)
        
        if output_file:
            with open(output_file, 'w') as f:
                f.write(report_text)
            logger.info(f"Report saved to {output_file}")
        
        return report_text

# CLI interface for running tests
async def main():
    """Main entry point for running integration tests"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Cross-service integration tests")
    parser.add_argument("--scenario", help="Run specific scenario by name")
    parser.add_argument("--report", help="Output report file path")
    parser.add_argument("--verbose", action="store_true", help="Verbose logging")
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    runner = CrossServiceTestRunner()
    
    if args.scenario:
        # Run specific scenario
        scenario = next((s for s in runner.scenarios if s.name == args.scenario), None)
        if not scenario:
            logger.error(f"Scenario '{args.scenario}' not found")
            available = [s.name for s in runner.scenarios]
            logger.info(f"Available scenarios: {', '.join(available)}")
            return 1
        
        runner.session = aiohttp.ClientSession()
        try:
            result = await runner.run_scenario(scenario)
            print(json.dumps(result, indent=2, default=str))
            return 0 if result["success"] else 1
        finally:
            await runner.session.close()
    else:
        # Run all scenarios
        results = await runner.run_all_scenarios()
        
        # Generate report
        report_file = args.report or f"integration_test_report_{int(time.time())}.md"
        runner.generate_report(results, report_file)
        
        # Print summary
        logger.info(f"Test Summary: {results['successful_scenarios']}/{results['total_scenarios']} scenarios passed")
        
        return 0 if results["failed_scenarios"] == 0 else 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)