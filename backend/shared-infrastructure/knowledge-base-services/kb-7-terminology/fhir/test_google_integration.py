#!/usr/bin/env python3
"""
Google FHIR Integration Test Suite for KB7 Terminology Service

This script provides comprehensive testing of the Google Cloud Healthcare API
integration with the KB7 hybrid terminology architecture.
"""

import asyncio
import json
import logging
import os
import sys
from typing import Dict, Any, List
from datetime import datetime
import aiohttp
import aioredis
from pathlib import Path

# Add the parent directory to sys.path to import local modules
sys.path.append(str(Path(__file__).parent))

from google_config import load_google_fhir_config, validate_google_credentials
from google_fhir_terminology_client import create_google_fhir_client
from google_fhir_service import create_hybrid_service
from models import (
    CodeSystemLookupRequest,
    ValueSetExpandRequest,
    ConceptMapTranslateRequest,
    ValidateCodeRequest
)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class GoogleFHIRIntegrationTester:
    """Comprehensive test suite for Google FHIR integration."""

    def __init__(self):
        self.config = None
        self.google_client = None
        self.hybrid_service = None
        self.redis_client = None
        self.test_results = []

    async def __aenter__(self):
        """Initialize test environment."""
        await self.setup()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Cleanup test environment."""
        await self.cleanup()

    async def setup(self):
        """Set up test environment and clients."""
        logger.info("Setting up Google FHIR integration test environment...")

        # Load configuration
        try:
            self.config = load_google_fhir_config()
            logger.info(f"Configuration loaded for project: {self.config.project_id}")
        except Exception as e:
            logger.error(f"Failed to load configuration: {e}")
            raise

        # Validate credentials
        if not validate_google_credentials(self.config):
            logger.error("Google credentials validation failed")
            raise ValueError("Invalid Google Cloud credentials")

        # Initialize Redis client
        try:
            self.redis_client = await aioredis.from_url("redis://localhost:6379")
            await self.redis_client.ping()
            logger.info("Redis connection established")
        except Exception as e:
            logger.warning(f"Redis connection failed: {e}")
            self.redis_client = None

        # Initialize Google FHIR client
        try:
            self.google_client = await create_google_fhir_client(
                config=self.config,
                redis_url="redis://localhost:6379" if self.redis_client else None
            )
            logger.info("Google FHIR client initialized")
        except Exception as e:
            logger.error(f"Failed to initialize Google FHIR client: {e}")
            raise

        # Initialize hybrid service
        try:
            self.hybrid_service = await create_hybrid_service(
                google_config=self.config,
                query_router_url="http://localhost:8087",
                redis_url="redis://localhost:6379" if self.redis_client else None
            )
            logger.info("Hybrid service initialized")
        except Exception as e:
            logger.error(f"Failed to initialize hybrid service: {e}")
            raise

    async def cleanup(self):
        """Clean up test resources."""
        logger.info("Cleaning up test environment...")

        if self.hybrid_service:
            try:
                await self.hybrid_service.__aexit__(None, None, None)
            except Exception as e:
                logger.error(f"Error cleaning up hybrid service: {e}")

        if self.google_client:
            try:
                await self.google_client.__aexit__(None, None, None)
            except Exception as e:
                logger.error(f"Error cleaning up Google client: {e}")

        if self.redis_client:
            try:
                await self.redis_client.close()
            except Exception as e:
                logger.error(f"Error closing Redis connection: {e}")

    def add_test_result(self, test_name: str, success: bool,
                       duration_ms: int, details: Dict[str, Any] = None):
        """Add a test result to the results list."""
        result = {
            "test_name": test_name,
            "success": success,
            "duration_ms": duration_ms,
            "timestamp": datetime.now().isoformat(),
            "details": details or {}
        }
        self.test_results.append(result)

        status = "PASS" if success else "FAIL"
        logger.info(f"Test {test_name}: {status} ({duration_ms}ms)")

    async def test_configuration_validation(self):
        """Test configuration validation and credential checking."""
        test_name = "Configuration Validation"
        start_time = datetime.now()

        try:
            # Test configuration loading
            config = load_google_fhir_config()
            assert config.project_id == "cardiofit-905a8"
            assert config.location == "asia-south1"
            assert config.dataset_id == "clinical-synthesis-hub"
            assert config.fhir_store_id == "fhir-store"

            # Test credentials validation
            creds_valid = validate_google_credentials(config)
            assert creds_valid, "Credentials validation failed"

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "project_id": config.project_id,
                "credentials_valid": creds_valid
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_google_client_health_check(self):
        """Test Google FHIR client health check."""
        test_name = "Google Client Health Check"
        start_time = datetime.now()

        try:
            health_result = await self.google_client.health_check()

            assert health_result.get("status") == "healthy"
            assert "fhir_store" in health_result
            assert "latency_seconds" in health_result

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, health_result)

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_hybrid_service_health_check(self):
        """Test hybrid service health check."""
        test_name = "Hybrid Service Health Check"
        start_time = datetime.now()

        try:
            health_result = await self.hybrid_service.health_check()

            assert "components" in health_result
            assert "google_fhir" in health_result["components"]
            assert "local_router" in health_result["components"]

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, health_result)

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_codesystem_lookup_google(self):
        """Test CodeSystem $lookup operation via Google FHIR."""
        test_name = "CodeSystem Lookup (Google)"
        start_time = datetime.now()

        try:
            # Test with a standard SNOMED CT code
            request = CodeSystemLookupRequest(
                system_url="http://snomed.info/sct",
                code="73211009",  # Diabetes mellitus
                display_language="en"
            )

            result = await self.google_client.lookup_code(request)

            assert result.name is not None or result.display is not None

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "system": request.system_url,
                "code": request.code,
                "display": result.display
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_codesystem_lookup_hybrid(self):
        """Test CodeSystem $lookup operation via hybrid service."""
        test_name = "CodeSystem Lookup (Hybrid)"
        start_time = datetime.now()

        try:
            request = CodeSystemLookupRequest(
                system_url="http://snomed.info/sct",
                code="73211009",  # Diabetes mellitus
                display_language="en"
            )

            operation_result = await self.hybrid_service.lookup_code(
                request=request,
                prefer_source="google"
            )

            assert operation_result.success
            assert operation_result.data is not None
            assert operation_result.source in ["google", "local"]

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "source": operation_result.source,
                "latency_ms": operation_result.latency_ms,
                "fallback_used": operation_result.fallback_used
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_valueset_expand_google(self):
        """Test ValueSet $expand operation via Google FHIR."""
        test_name = "ValueSet Expand (Google)"
        start_time = datetime.now()

        try:
            request = ValueSetExpandRequest(
                url="http://hl7.org/fhir/ValueSet/administrative-gender",
                count=10
            )

            result = await self.google_client.expand_valueset(request)

            assert "expansion" in result or "entry" in result

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "valueset_url": request.url,
                "expansion_size": len(result.get("expansion", {}).get("contains", []))
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_validate_code_google(self):
        """Test $validate-code operation via Google FHIR."""
        test_name = "Validate Code (Google)"
        start_time = datetime.now()

        try:
            request = ValidateCodeRequest(
                code="male",
                system="http://hl7.org/fhir/administrative-gender",
                display="Male"
            )

            result = await self.google_client.validate_code(request)

            assert result.result is not None

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "code": request.code,
                "system": request.system,
                "valid": result.result
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_fallback_mechanism(self):
        """Test fallback mechanism from Google to local service."""
        test_name = "Fallback Mechanism"
        start_time = datetime.now()

        try:
            # Test with a code that might not exist in Google FHIR
            request = CodeSystemLookupRequest(
                system_url="http://example.com/custom-codes",
                code="CUSTOM_001",
                display_language="en"
            )

            operation_result = await self.hybrid_service.lookup_code(
                request=request,
                prefer_source="google"
            )

            # Should either succeed or gracefully handle the failure
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "success": operation_result.success,
                "source": operation_result.source,
                "fallback_used": operation_result.fallback_used,
                "error": operation_result.error
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_performance_benchmarks(self):
        """Test performance benchmarks for various operations."""
        test_name = "Performance Benchmarks"
        start_time = datetime.now()

        try:
            benchmarks = {}

            # Benchmark lookup operation
            lookup_times = []
            for i in range(5):
                loop_start = datetime.now()
                request = CodeSystemLookupRequest(
                    system_url="http://snomed.info/sct",
                    code="73211009"
                )
                await self.hybrid_service.lookup_code(request)
                lookup_times.append((datetime.now() - loop_start).total_seconds() * 1000)

            benchmarks["lookup_avg_ms"] = sum(lookup_times) / len(lookup_times)
            benchmarks["lookup_max_ms"] = max(lookup_times)

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, benchmarks)

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def test_statistics_collection(self):
        """Test statistics collection from hybrid service."""
        test_name = "Statistics Collection"
        start_time = datetime.now()

        try:
            stats = await self.hybrid_service.get_statistics()

            assert "hybrid_service" in stats
            assert "google_fhir" in stats
            assert "configuration" in stats

            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, True, duration, {
                "stats_keys": list(stats.keys()),
                "google_requests": stats.get("hybrid_service", {}).get("google_requests", 0),
                "local_requests": stats.get("hybrid_service", {}).get("local_requests", 0)
            })

        except Exception as e:
            duration = int((datetime.now() - start_time).total_seconds() * 1000)
            self.add_test_result(test_name, False, duration, {"error": str(e)})

    async def run_all_tests(self) -> Dict[str, Any]:
        """Run all integration tests."""
        logger.info("Starting Google FHIR integration test suite...")

        test_methods = [
            self.test_configuration_validation,
            self.test_google_client_health_check,
            self.test_hybrid_service_health_check,
            self.test_codesystem_lookup_google,
            self.test_codesystem_lookup_hybrid,
            self.test_valueset_expand_google,
            self.test_validate_code_google,
            self.test_fallback_mechanism,
            self.test_performance_benchmarks,
            self.test_statistics_collection
        ]

        for test_method in test_methods:
            try:
                await test_method()
            except Exception as e:
                logger.error(f"Test {test_method.__name__} failed with exception: {e}")

        # Calculate summary statistics
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result["success"])
        failed_tests = total_tests - passed_tests

        avg_duration = sum(result["duration_ms"] for result in self.test_results) / total_tests

        summary = {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "failed_tests": failed_tests,
            "success_rate": passed_tests / total_tests if total_tests > 0 else 0,
            "average_duration_ms": avg_duration,
            "timestamp": datetime.now().isoformat(),
            "test_results": self.test_results
        }

        logger.info(f"Test suite completed: {passed_tests}/{total_tests} tests passed")
        return summary


async def main():
    """Main test execution function."""
    # Check environment variables
    required_env_vars = [
        "GOOGLE_CLOUD_PROJECT_ID",
        "GOOGLE_CLOUD_DATASET_ID",
        "GOOGLE_CLOUD_FHIR_STORE_ID"
    ]

    missing_vars = [var for var in required_env_vars if not os.getenv(var)]
    if missing_vars:
        logger.error(f"Missing required environment variables: {missing_vars}")
        logger.error("Please set up your .env file with Google FHIR configuration")
        return 1

    try:
        async with GoogleFHIRIntegrationTester() as tester:
            results = await tester.run_all_tests()

            # Save results to file
            results_file = f"google_fhir_test_results_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
            with open(results_file, 'w') as f:
                json.dump(results, f, indent=2)

            logger.info(f"Test results saved to: {results_file}")

            # Return appropriate exit code
            return 0 if results["failed_tests"] == 0 else 1

    except Exception as e:
        logger.error(f"Test suite failed with error: {e}")
        return 1


if __name__ == "__main__":
    # Load environment variables from .env file
    from dotenv import load_dotenv
    load_dotenv(".env.google-fhir.example")

    exit_code = asyncio.run(main())
    sys.exit(exit_code)