"""
Clinical Assertion Engine with Neo4j Knowledge Graph Integration

Main CAE Engine class that orchestrates clinical reasoning using real Neo4j
knowledge graph instead of mock data.
"""

import asyncio
import time
from typing import Dict, Any, List
import logging
import os
from .knowledge.knowledge_service import KnowledgeGraphService
from .reasoners.ddi_checker import DDIChecker
from .reasoners.allergy_checker_neo4j import AllergyChecker
from .reasoners.dose_validator_neo4j import DoseValidator
from .reasoners.contraindication_checker_neo4j import ContraindicationChecker
from .reasoners.base_checker import CheckerResult

logger = logging.getLogger(__name__)

class CAEEngine:
    """Clinical Assertion Engine with Neo4j Knowledge Graph Integration"""

    def __init__(self):
        # Use existing Neo4j client from knowledge-pipeline-service
        self.knowledge_service = KnowledgeGraphService()
        self.logger = logging.getLogger(__name__)

        # Initialize checkers with knowledge service
        self.checkers = {
            'ddi': DDIChecker(self.knowledge_service),
            'allergy': AllergyChecker(self.knowledge_service),
            'dose': DoseValidator(self.knowledge_service),
            'contraindication': ContraindicationChecker(self.knowledge_service)
        }

        # Performance tracking
        self.total_requests = 0
        self.total_execution_time = 0
        self.error_count = 0

    async def initialize(self):
        """Initialize the CAE Engine"""
        try:
            success = await self.knowledge_service.initialize()
            if success:
                self.logger.info("CAE Engine initialized with existing Neo4j knowledge graph client")
                return True
            else:
                self.logger.error("Failed to initialize CAE Engine - Neo4j connection failed")
                return False
        except Exception as e:
            self.logger.error(f"Failed to initialize CAE Engine: {e}")
            return False

    async def validate_safety(self, clinical_context: Dict[str, Any]) -> Dict[str, Any]:
        """Main safety validation using parallel checker execution"""
        start_time = time.time()
        self.total_requests += 1

        try:
            # Validate input
            if not self._validate_clinical_context(clinical_context):
                return self._create_error_response("Invalid clinical context", start_time)

            # Execute all checkers in parallel
            checker_tasks = [
                self._run_checker_with_timing(name, checker, clinical_context)
                for name, checker in self.checkers.items()
            ]

            checker_results = await asyncio.gather(*checker_tasks, return_exceptions=True)

            # Process results
            results = {}
            findings = []
            overall_status = "SAFE"

            for i, result in enumerate(checker_results):
                checker_name = list(self.checkers.keys())[i]

                if isinstance(result, Exception):
                    self.logger.error(f"Checker {checker_name} failed: {result}")
                    self.error_count += 1
                    results[checker_name] = {
                        'status': 'ERROR',
                        'error': str(result),
                        'execution_time_ms': 0
                    }
                    continue

                results[checker_name] = {
                    'status': result.status,
                    'findings': [finding.to_dict() for finding in result.findings],
                    'execution_time_ms': result.execution_time_ms
                }

                findings.extend(result.findings)

                # Update overall status
                if result.status == "UNSAFE":
                    overall_status = "UNSAFE"
                elif result.status == "WARNING" and overall_status == "SAFE":
                    overall_status = "WARNING"

            total_time_ms = (time.time() - start_time) * 1000
            self.total_execution_time += total_time_ms

            # Get cache statistics
            cache_stats = await self.knowledge_service.get_cache_stats()

            return {
                'overall_status': overall_status,
                'total_findings': len(findings),
                'findings': [finding.to_dict() for finding in findings],
                'checker_results': results,
                'performance': {
                    'total_execution_time_ms': total_time_ms,
                    'cache_stats': cache_stats,
                    'parallel_execution': True
                },
                'metadata': {
                    'engine_version': '2.0-neo4j',
                    'knowledge_source': 'Neo4j Knowledge Graph',
                    'timestamp': time.time(),
                    'request_id': f"cae_{int(time.time() * 1000)}"
                }
            }

        except Exception as e:
            self.logger.error(f"CAE Engine validation failed: {e}")
            self.error_count += 1
            return self._create_error_response(str(e), start_time)

    async def _run_checker_with_timing(self, name: str, checker, clinical_context: Dict[str, Any]) -> CheckerResult:
        """Run checker with execution timing"""
        start_time = time.time()

        try:
            result = await checker.check(clinical_context)
            result.execution_time_ms = (time.time() - start_time) * 1000
            
            self.logger.debug(f"Checker {name} completed in {result.execution_time_ms:.1f}ms")
            return result

        except Exception as e:
            self.logger.error(f"Checker {name} failed: {e}")
            raise

    def _validate_clinical_context(self, clinical_context: Dict[str, Any]) -> bool:
        """Validate clinical context structure"""
        required_fields = ['patient', 'medications']
        
        for field in required_fields:
            if field not in clinical_context:
                self.logger.warning(f"Missing required field: {field}")
                return False
        
        # Validate patient has ID
        patient = clinical_context.get('patient', {})
        if not patient.get('id'):
            self.logger.warning("Patient missing ID")
            return False
        
        return True

    def _create_error_response(self, error_message: str, start_time: float) -> Dict[str, Any]:
        """Create error response"""
        return {
            'overall_status': 'ERROR',
            'error': error_message,
            'total_findings': 0,
            'findings': [],
            'checker_results': {},
            'performance': {
                'total_execution_time_ms': (time.time() - start_time) * 1000
            },
            'metadata': {
                'engine_version': '2.0-neo4j',
                'knowledge_source': 'Neo4j Knowledge Graph',
                'timestamp': time.time()
            }
        }

    async def get_health_status(self) -> Dict[str, Any]:
        """Get CAE Engine health status"""
        try:
            # Test Neo4j connection using existing client
            connection_ok = await self.knowledge_service.client.test_connection()
            cache_stats = await self.knowledge_service.get_cache_stats()

            # Calculate performance metrics
            avg_execution_time = (self.total_execution_time / self.total_requests) if self.total_requests > 0 else 0
            error_rate = (self.error_count / self.total_requests * 100) if self.total_requests > 0 else 0

            return {
                'status': 'HEALTHY' if connection_ok else 'UNHEALTHY',
                'neo4j_connection': connection_ok,
                'cache_stats': cache_stats,
                'checkers': list(self.checkers.keys()),
                'performance_metrics': {
                    'total_requests': self.total_requests,
                    'average_execution_time_ms': avg_execution_time,
                    'error_rate_percent': error_rate,
                    'total_errors': self.error_count
                },
                'timestamp': time.time()
            }

        except Exception as e:
            return {
                'status': 'ERROR',
                'error': str(e),
                'timestamp': time.time()
            }

    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get detailed performance metrics"""
        cache_stats = await self.knowledge_service.get_cache_stats()

        return {
            'requests': {
                'total': self.total_requests,
                'errors': self.error_count,
                'success_rate': ((self.total_requests - self.error_count) / self.total_requests * 100) if self.total_requests > 0 else 0
            },
            'performance': {
                'total_execution_time_ms': self.total_execution_time,
                'average_execution_time_ms': (self.total_execution_time / self.total_requests) if self.total_requests > 0 else 0
            },
            'cache': cache_stats,
            'checkers': {
                'active': list(self.checkers.keys()),
                'count': len(self.checkers)
            }
        }

    async def reset_metrics(self):
        """Reset performance metrics"""
        self.total_requests = 0
        self.total_execution_time = 0
        self.error_count = 0
        await self.knowledge_service.cache.clear_all()
        self.logger.info("Performance metrics reset")

    async def close(self):
        """Close CAE Engine and cleanup resources"""
        try:
            await self.knowledge_service.close()
            self.logger.info("CAE Engine closed successfully")
        except Exception as e:
            self.logger.error(f"Error closing CAE Engine: {e}")
