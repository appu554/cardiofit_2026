"""
Parallel Executor for Clinical Assertion Engine

Executes multiple clinical reasoners concurrently with dependency management,
failure isolation, and performance optimization.
"""

import asyncio
import logging
from datetime import datetime
from typing import Dict, List, Optional, Any, Callable
from dataclasses import dataclass
from enum import Enum

from .request_router import ClinicalRequest

logger = logging.getLogger(__name__)


class ReasonerStatus(Enum):
    """Status of individual reasoner execution"""
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    TIMEOUT = "timeout"


@dataclass
class ReasonerResult:
    """Result from individual reasoner execution"""
    reasoner_type: str
    status: ReasonerStatus
    assertions: List[Dict[str, Any]]
    execution_time_ms: float
    error_message: Optional[str] = None
    confidence_score: Optional[float] = None
    metadata: Optional[Dict[str, Any]] = None


@dataclass
class ExecutionPlan:
    """Execution plan for reasoners with dependencies"""
    reasoner_type: str
    dependencies: List[str]
    timeout_ms: int
    executor_func: Callable
    priority: int = 0


class ParallelExecutor:
    """
    High-performance parallel executor for clinical reasoners
    
    Features:
    - Concurrent execution with asyncio.gather()
    - Dependency management between reasoners
    - Failure isolation and partial results
    - Timeout management per reasoner
    - Performance monitoring and optimization
    """
    
    def __init__(self):
        self.execution_stats = {
            'total_executions': 0,
            'successful_executions': 0,
            'failed_executions': 0,
            'average_execution_time': 0.0,
            'reasoner_performance': {}
        }
        self.reasoner_registry = {}
        logger.info("Parallel Executor initialized")
    
    def register_reasoner(self, reasoner_type: str, executor_func: Callable, 
                         dependencies: List[str] = None, timeout_ms: int = 1000):
        """Register a reasoner with its execution function and dependencies"""
        self.reasoner_registry[reasoner_type] = ExecutionPlan(
            reasoner_type=reasoner_type,
            dependencies=dependencies or [],
            timeout_ms=timeout_ms,
            executor_func=executor_func
        )
        logger.info(f"Registered reasoner: {reasoner_type}")
    
    async def execute_reasoners(self, request: ClinicalRequest, 
                              reasoner_instances: Dict[str, Any]) -> Dict[str, ReasonerResult]:
        """
        Execute multiple reasoners in parallel with dependency management
        
        Args:
            request: Clinical request with context
            reasoner_instances: Dictionary of reasoner instances
            
        Returns:
            Dictionary of reasoner results
        """
        start_time = datetime.utcnow()
        
        try:
            # Handle dictionary vs object types for request attributes
            if isinstance(request, dict):
                reasoner_types = request.get('reasoner_types', [])
                patient_id = request.get('patient_id', 'unknown')
            else:
                reasoner_types = request.reasoner_types
                patient_id = request.patient_id
            
            # Build execution plan
            execution_plan = self._build_execution_plan(reasoner_types)
            
            # Execute reasoners in dependency order
            results = await self._execute_with_dependencies(
                execution_plan, request, reasoner_instances
            )
            
            # Update statistics
            execution_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            self._update_stats(results, execution_time)
            
            logger.info(f"Executed {len(results)} reasoners in {execution_time:.2f}ms "
                       f"for patient {patient_id}")
            
            return results
            
        except Exception as e:
            logger.error(f"Error in parallel execution: {e}")
            self.execution_stats['failed_executions'] += 1
            raise
    
    def _build_execution_plan(self, reasoner_types: List[str]) -> List[List[str]]:
        """
        Build execution plan respecting dependencies
        
        Returns list of execution waves where each wave can run in parallel
        """
        # For now, implement simple parallel execution
        # TODO: Implement proper dependency resolution
        
        # Group reasoners by dependency level
        independent_reasoners = []
        dependent_reasoners = []
        
        for reasoner_type in reasoner_types:
            if reasoner_type in self.reasoner_registry:
                plan = self.reasoner_registry[reasoner_type]
                if not plan.dependencies:
                    independent_reasoners.append(reasoner_type)
                else:
                    dependent_reasoners.append(reasoner_type)
            else:
                # Default reasoners that can run independently
                independent_reasoners.append(reasoner_type)
        
        execution_waves = []
        if independent_reasoners:
            execution_waves.append(independent_reasoners)
        if dependent_reasoners:
            execution_waves.append(dependent_reasoners)
        
        return execution_waves
    
    async def _execute_with_dependencies(self, execution_waves: List[List[str]], 
                                       request: ClinicalRequest,
                                       reasoner_instances: Dict[str, Any]) -> Dict[str, ReasonerResult]:
        """Execute reasoners in waves respecting dependencies"""
        all_results = {}
        
        for wave_index, wave_reasoners in enumerate(execution_waves):
            logger.info(f"Executing wave {wave_index + 1} with reasoners: {wave_reasoners}")
            
            # Create tasks for this wave
            tasks = []
            for reasoner_type in wave_reasoners:
                task = self._execute_single_reasoner(
                    reasoner_type, request, reasoner_instances, all_results
                )
                tasks.append(task)
            
            # Execute wave in parallel
            wave_results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Process wave results
            for i, result in enumerate(wave_results):
                reasoner_type = wave_reasoners[i]
                if isinstance(result, Exception):
                    logger.error(f"Reasoner {reasoner_type} failed: {result}")
                    all_results[reasoner_type] = ReasonerResult(
                        reasoner_type=reasoner_type,
                        status=ReasonerStatus.FAILED,
                        assertions=[],
                        execution_time_ms=0.0,
                        error_message=str(result)
                    )
                else:
                    all_results[reasoner_type] = result
        
        return all_results
    
    async def _execute_single_reasoner(self, reasoner_type: str, request: ClinicalRequest,
                                     reasoner_instances: Dict[str, Any],
                                     previous_results: Dict[str, ReasonerResult]) -> ReasonerResult:
        """Execute a single reasoner with timeout and error handling"""
        start_time = datetime.utcnow()
        
        try:
            # Get reasoner instance
            reasoner = reasoner_instances.get(reasoner_type)
            if not reasoner:
                raise ValueError(f"Reasoner {reasoner_type} not available")
            
            # Determine timeout - more generous for individual reasoners
            timeout_ms = max(100, request.timeout_ms // len(request.reasoner_types))  # At least 100ms per reasoner
            
            # Execute reasoner with timeout
            result = await asyncio.wait_for(
                self._call_reasoner(reasoner_type, reasoner, request, previous_results),
                timeout=timeout_ms / 1000.0
            )
            
            execution_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            
            return ReasonerResult(
                reasoner_type=reasoner_type,
                status=ReasonerStatus.COMPLETED,
                assertions=result.get('assertions', []),
                execution_time_ms=execution_time,
                confidence_score=result.get('confidence_score'),
                metadata=result.get('metadata', {})
            )
            
        except asyncio.TimeoutError:
            execution_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            logger.warning(f"Reasoner {reasoner_type} timed out after {execution_time:.2f}ms")
            
            return ReasonerResult(
                reasoner_type=reasoner_type,
                status=ReasonerStatus.TIMEOUT,
                assertions=[],
                execution_time_ms=execution_time,
                error_message=f"Timeout after {timeout_ms}ms"
            )
            
        except Exception as e:
            execution_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            logger.error(f"Reasoner {reasoner_type} failed: {e}")
            
            return ReasonerResult(
                reasoner_type=reasoner_type,
                status=ReasonerStatus.FAILED,
                assertions=[],
                execution_time_ms=execution_time,
                error_message=str(e)
            )
    
    async def _call_reasoner(self, reasoner_type: str, reasoner: Any,
                           request: ClinicalRequest,
                           previous_results: Dict[str, ReasonerResult]) -> Dict[str, Any]:
        """Call the appropriate reasoner method based on type"""

        # Ensure clinical_context is always a dictionary
        if isinstance(request.clinical_context, dict):
            clinical_context = request.clinical_context
        elif hasattr(request.clinical_context, '__dict__'):
            # Convert object to dict if it has __dict__ attribute
            clinical_context = vars(request.clinical_context)
        else:
            clinical_context = {}

        # Use the full clinical_context as the patient_context
        patient_context = clinical_context

        # Debug logging to identify the issue
        logger.info(f"DEBUG: clinical_context type: {type(request.clinical_context)}, value: {request.clinical_context}")
        logger.info(f"DEBUG: extracted patient_context: {patient_context}")
        logger.info(f"DEBUG: request.medication_ids type: {type(request.medication_ids)}, value: {request.medication_ids}")
        logger.info(f"DEBUG: request.condition_ids type: {type(request.condition_ids)}, value: {request.condition_ids}")
        logger.info(f"DEBUG: request.allergy_ids type: {type(request.allergy_ids)}, value: {request.allergy_ids}")

        if reasoner_type == "interaction":
            if hasattr(reasoner, 'check_interactions'):
                return await reasoner.check_interactions(
                    patient_id=request.patient_id,
                    medication_ids=request.medication_ids,
                    patient_context=patient_context
                )
        
        elif reasoner_type == "dosing":
            if hasattr(reasoner, 'calculate_dosing'):
                # For dosing, we need to process each medication individually
                dosing_results = []
                for medication_id in request.medication_ids:
                    try:
                        dosing_result = await reasoner.calculate_dosing(
                            patient_id=request.patient_id,
                            medication_id=medication_id,
                            patient_parameters=patient_context,
                            indication=patient_context.get('indication') if patient_context else None
                        )
                        dosing_results.append({
                            'medication_id': medication_id,
                            'dosing': dosing_result
                        })
                    except Exception as e:
                        dosing_results.append({
                            'medication_id': medication_id,
                            'error': str(e)
                        })
                return {'dosing_recommendations': dosing_results}
        
        elif reasoner_type == "contraindication":
            if hasattr(reasoner, 'check_contraindications'):
                return await reasoner.check_contraindications(
                    patient_id=request.patient_id,
                    medication_ids=request.medication_ids,
                    condition_ids=request.condition_ids,
                    allergy_ids=request.allergy_ids,
                    patient_context=patient_context
                )

        elif reasoner_type == "duplicate_therapy":
            if hasattr(reasoner, 'check_duplicate_therapy'):
                return await reasoner.check_duplicate_therapy(
                    patient_id=request.patient_id,
                    medication_ids=request.medication_ids,
                    patient_context=patient_context
                )

        elif reasoner_type == "clinical_context":
            if hasattr(reasoner, 'check_clinical_context'):
                return await reasoner.check_clinical_context(
                    patient_id=request.patient_id,
                    medication_ids=request.medication_ids,
                    patient_context=patient_context
                )

        # Default fallback
        return {
            'assertions': [],
            'confidence_score': 0.0,
            'metadata': {'reasoner_type': reasoner_type, 'status': 'not_implemented'}
        }
    
    def _update_stats(self, results: Dict[str, ReasonerResult], total_execution_time: float):
        """Update execution statistics"""
        self.execution_stats['total_executions'] += 1
        
        successful_count = sum(1 for r in results.values() if r.status == ReasonerStatus.COMPLETED)
        if successful_count == len(results):
            self.execution_stats['successful_executions'] += 1
        else:
            self.execution_stats['failed_executions'] += 1
        
        # Update average execution time
        current_avg = self.execution_stats['average_execution_time']
        total_execs = self.execution_stats['total_executions']
        self.execution_stats['average_execution_time'] = (
            (current_avg * (total_execs - 1) + total_execution_time) / total_execs
        )
        
        # Update per-reasoner performance
        for reasoner_type, result in results.items():
            if reasoner_type not in self.execution_stats['reasoner_performance']:
                self.execution_stats['reasoner_performance'][reasoner_type] = {
                    'total_calls': 0,
                    'successful_calls': 0,
                    'average_time': 0.0
                }
            
            perf = self.execution_stats['reasoner_performance'][reasoner_type]
            perf['total_calls'] += 1
            
            if result.status == ReasonerStatus.COMPLETED:
                perf['successful_calls'] += 1
            
            # Update average time
            current_avg = perf['average_time']
            total_calls = perf['total_calls']
            perf['average_time'] = (
                (current_avg * (total_calls - 1) + result.execution_time_ms) / total_calls
            )
    
    def get_stats(self) -> Dict[str, Any]:
        """Get execution statistics"""
        return self.execution_stats.copy()
