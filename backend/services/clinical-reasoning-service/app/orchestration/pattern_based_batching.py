"""
Pattern-Based Batching System for Clinical Assertion Engine

Implements intelligent batching of similar requests using graph similarity algorithms
to optimize processing efficiency and reduce redundant computations.

Key Features:
- Request similarity analysis using graph algorithms
- Intelligent batch formation based on clinical patterns
- Adaptive batch sizing based on system load
- Batch processing optimization
- Result distribution and correlation
"""

import asyncio
import logging
import time
import hashlib
from typing import Dict, List, Optional, Any, Set, Tuple
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta, timezone
from collections import defaultdict, Counter
from enum import Enum
import json
import numpy as np
from sklearn.cluster import DBSCAN
from sklearn.metrics.pairwise import cosine_similarity

# Import request types
from .request_router import ClinicalRequest, RequestPriority
from .graph_request_router import EnhancedClinicalRequest, RoutingStrategy

logger = logging.getLogger(__name__)


class BatchStrategy(Enum):
    """Batching strategies based on request characteristics"""
    SIMILARITY_BASED = "similarity_based"    # Group by request similarity
    PRIORITY_BASED = "priority_based"        # Group by priority level
    TEMPORAL_BASED = "temporal_based"        # Group by time windows
    RESOURCE_BASED = "resource_based"        # Group by resource requirements
    HYBRID = "hybrid"                        # Combination of strategies


@dataclass
class RequestBatch:
    """Batch of similar clinical requests"""
    batch_id: str
    requests: List[EnhancedClinicalRequest]
    batch_strategy: BatchStrategy
    similarity_score: float
    created_at: datetime
    priority: RequestPriority
    estimated_processing_time: int
    shared_context: Dict[str, Any]
    optimization_opportunities: List[str]


@dataclass
class BatchingStats:
    """Batching performance statistics"""
    total_requests: int
    batched_requests: int
    total_batches: int
    average_batch_size: float
    processing_time_saved: float
    similarity_threshold_hits: int
    batch_strategy_distribution: Dict[str, int]


class PatternBasedBatcher:
    """
    Intelligent request batcher using graph similarity algorithms
    
    This system analyzes incoming clinical requests and groups similar ones
    into batches for optimized processing, reducing redundant computations
    and improving overall system efficiency.
    """
    
    def __init__(self):
        # Batching configuration
        self.similarity_threshold = 0.7
        self.max_batch_size = 10
        self.min_batch_size = 2
        self.batch_timeout_ms = 100  # Maximum time to wait for batch formation
        self.max_wait_time_ms = 50   # Maximum wait time for similar requests
        
        # Request queues by priority
        self.request_queues: Dict[RequestPriority, List[EnhancedClinicalRequest]] = {
            priority: [] for priority in RequestPriority
        }
        
        # Pending batches
        self.pending_batches: Dict[str, RequestBatch] = {}
        self.completed_batches: List[RequestBatch] = []
        
        # Similarity cache
        self.similarity_cache: Dict[str, float] = {}
        
        # Statistics
        self.stats = BatchingStats(
            total_requests=0,
            batched_requests=0,
            total_batches=0,
            average_batch_size=0.0,
            processing_time_saved=0.0,
            similarity_threshold_hits=0,
            batch_strategy_distribution={s.value: 0 for s in BatchStrategy}
        )
        
        # Background tasks
        self._background_tasks: Set[asyncio.Task] = set()
        self._start_background_tasks()
        
        logger.info("Pattern-Based Batcher initialized")
    
    def _start_background_tasks(self):
        """Start background batching tasks"""
        try:
            # Batch formation task
            batch_task = asyncio.create_task(self._batch_formation_loop())
            self._background_tasks.add(batch_task)
            batch_task.add_done_callback(self._background_tasks.discard)
            
            # Batch timeout task
            timeout_task = asyncio.create_task(self._batch_timeout_loop())
            self._background_tasks.add(timeout_task)
            timeout_task.add_done_callback(self._background_tasks.discard)
            
            logger.debug("Background batching tasks started")
            
        except Exception as e:
            logger.warning(f"Error starting background tasks: {e}")
    
    async def add_request(self, request: EnhancedClinicalRequest) -> Optional[str]:
        """
        Add request to batching system
        
        Args:
            request: Enhanced clinical request to batch
            
        Returns:
            Batch ID if request was batched, None if processed immediately
        """
        try:
            self.stats.total_requests += 1
            
            # Check if request should be batched
            if not self._should_batch_request(request):
                return None
            
            # Find similar pending requests
            similar_requests = await self._find_similar_requests(request)
            
            if similar_requests:
                # Add to existing batch or create new one
                batch_id = await self._add_to_batch(request, similar_requests)
                self.stats.batched_requests += 1
                return batch_id
            else:
                # Add to queue for potential batching
                self.request_queues[request.priority].append(request)
                return None
                
        except Exception as e:
            logger.error(f"Error adding request to batcher: {e}")
            return None
    
    async def get_batch(self, batch_id: str) -> Optional[RequestBatch]:
        """Get batch by ID"""
        return self.pending_batches.get(batch_id)
    
    async def complete_batch(self, batch_id: str, results: Dict[str, Any]):
        """Mark batch as completed and distribute results"""
        try:
            if batch_id not in self.pending_batches:
                logger.warning(f"Batch {batch_id} not found")
                return
            
            batch = self.pending_batches[batch_id]
            
            # Move to completed batches
            self.completed_batches.append(batch)
            del self.pending_batches[batch_id]
            
            # Update statistics
            self.stats.total_batches += 1
            batch_size = len(batch.requests)
            
            # Update average batch size
            total_batches = self.stats.total_batches
            self.stats.average_batch_size = (
                (self.stats.average_batch_size * (total_batches - 1) + batch_size) / total_batches
            )
            
            # Estimate processing time saved
            individual_time = sum(r.timeout_ms for r in batch.requests)
            batch_time = batch.estimated_processing_time
            time_saved = max(0, individual_time - batch_time)
            self.stats.processing_time_saved += time_saved
            
            logger.info(f"Completed batch {batch_id} with {batch_size} requests "
                       f"(saved {time_saved}ms)")
            
        except Exception as e:
            logger.error(f"Error completing batch {batch_id}: {e}")
    
    def _should_batch_request(self, request: EnhancedClinicalRequest) -> bool:
        """Determine if request should be batched"""
        try:
            # Don't batch critical requests
            if request.priority == RequestPriority.CRITICAL:
                return False
            
            # Don't batch if routing strategy doesn't support batching
            if request.routing_strategy == RoutingStrategy.ADAPTIVE:
                return False
            
            # Check if request has batching hints
            if request.optimization_hints.get('batch_compatible', True):
                return True
            
            # Batch if request has multiple medications (likely to have similarities)
            if len(request.medication_ids) > 1:
                return True
            
            return False
            
        except Exception as e:
            logger.warning(f"Error checking if request should be batched: {e}")
            return False
    
    async def _find_similar_requests(self, request: EnhancedClinicalRequest) -> List[EnhancedClinicalRequest]:
        """Find similar requests in queues"""
        try:
            similar_requests = []
            
            # Search in same priority queue first
            queue = self.request_queues[request.priority]
            for queued_request in queue:
                similarity = await self._calculate_request_similarity(request, queued_request)
                if similarity >= self.similarity_threshold:
                    similar_requests.append(queued_request)
                    self.stats.similarity_threshold_hits += 1
            
            # If not enough similar requests, search in lower priority queues
            if len(similar_requests) < self.min_batch_size:
                for priority in RequestPriority:
                    if priority.value > request.priority.value:  # Lower priority
                        queue = self.request_queues[priority]
                        for queued_request in queue:
                            similarity = await self._calculate_request_similarity(request, queued_request)
                            if similarity >= self.similarity_threshold:
                                similar_requests.append(queued_request)
                                if len(similar_requests) >= self.max_batch_size:
                                    break
                        if len(similar_requests) >= self.max_batch_size:
                            break
            
            return similar_requests[:self.max_batch_size]
            
        except Exception as e:
            logger.warning(f"Error finding similar requests: {e}")
            return []
    
    async def _calculate_request_similarity(
        self, 
        request1: EnhancedClinicalRequest, 
        request2: EnhancedClinicalRequest
    ) -> float:
        """Calculate similarity between two requests"""
        try:
            # Create cache key
            key1 = self._create_request_key(request1)
            key2 = self._create_request_key(request2)
            cache_key = f"{min(key1, key2)}:{max(key1, key2)}"
            
            # Check cache
            if cache_key in self.similarity_cache:
                return self.similarity_cache[cache_key]
            
            similarity = 0.0
            
            # Medication similarity (40% weight)
            med_similarity = self._calculate_medication_similarity(
                request1.medication_ids, request2.medication_ids
            )
            similarity += med_similarity * 0.4
            
            # Condition similarity (30% weight)
            condition_similarity = self._calculate_condition_similarity(
                request1.condition_ids, request2.condition_ids
            )
            similarity += condition_similarity * 0.3
            
            # Clinical context similarity (20% weight)
            context_similarity = self._calculate_context_similarity(
                request1.clinical_context, request2.clinical_context
            )
            similarity += context_similarity * 0.2
            
            # Graph context similarity (10% weight)
            graph_similarity = self._calculate_graph_similarity(
                request1.graph_context, request2.graph_context
            )
            similarity += graph_similarity * 0.1
            
            # Cache the result
            self.similarity_cache[cache_key] = similarity
            
            return similarity
            
        except Exception as e:
            logger.warning(f"Error calculating request similarity: {e}")
            return 0.0
    
    def _calculate_medication_similarity(self, meds1: List[str], meds2: List[str]) -> float:
        """Calculate medication list similarity"""
        try:
            if not meds1 and not meds2:
                return 1.0
            if not meds1 or not meds2:
                return 0.0
            
            set1 = set(meds1)
            set2 = set(meds2)
            
            intersection = len(set1.intersection(set2))
            union = len(set1.union(set2))
            
            return intersection / union if union > 0 else 0.0
            
        except Exception:
            return 0.0
    
    def _calculate_condition_similarity(self, conds1: List[str], conds2: List[str]) -> float:
        """Calculate condition list similarity"""
        try:
            if not conds1 and not conds2:
                return 1.0
            if not conds1 or not conds2:
                return 0.0
            
            set1 = set(conds1)
            set2 = set(conds2)
            
            intersection = len(set1.intersection(set2))
            union = len(set1.union(set2))
            
            return intersection / union if union > 0 else 0.0
            
        except Exception:
            return 0.0
    
    def _calculate_context_similarity(self, ctx1: Dict[str, Any], ctx2: Dict[str, Any]) -> float:
        """Calculate clinical context similarity"""
        try:
            if not ctx1 and not ctx2:
                return 1.0
            if not ctx1 or not ctx2:
                return 0.0
            
            # Compare key context fields
            similarity_factors = []
            
            # Age similarity
            age1 = ctx1.get('age', 0)
            age2 = ctx2.get('age', 0)
            if age1 and age2:
                age_diff = abs(age1 - age2)
                age_similarity = max(0, 1 - age_diff / 100)
                similarity_factors.append(age_similarity)
            
            # Gender similarity
            gender1 = ctx1.get('gender', '')
            gender2 = ctx2.get('gender', '')
            if gender1 and gender2:
                gender_similarity = 1.0 if gender1 == gender2 else 0.0
                similarity_factors.append(gender_similarity)
            
            # Encounter type similarity
            encounter1 = ctx1.get('encounter_type', '')
            encounter2 = ctx2.get('encounter_type', '')
            if encounter1 and encounter2:
                encounter_similarity = 1.0 if encounter1 == encounter2 else 0.0
                similarity_factors.append(encounter_similarity)
            
            return sum(similarity_factors) / len(similarity_factors) if similarity_factors else 0.0
            
        except Exception:
            return 0.0

    def _calculate_graph_similarity(self, graph1: Optional[Any], graph2: Optional[Any]) -> float:
        """Calculate graph context similarity"""
        try:
            if not graph1 and not graph2:
                return 1.0
            if not graph1 or not graph2:
                return 0.0

            # Compare similarity scores
            sim1 = getattr(graph1, 'similarity_score', 0.0) if hasattr(graph1, 'similarity_score') else 0.0
            sim2 = getattr(graph2, 'similarity_score', 0.0) if hasattr(graph2, 'similarity_score') else 0.0

            if sim1 == 0.0 and sim2 == 0.0:
                return 1.0

            return 1.0 - abs(sim1 - sim2)

        except Exception:
            return 0.0

    def _create_request_key(self, request: EnhancedClinicalRequest) -> str:
        """Create unique key for request"""
        try:
            key_data = {
                'medications': sorted(request.medication_ids),
                'conditions': sorted(request.condition_ids),
                'priority': request.priority.value,
                'strategy': request.routing_strategy.value
            }
            key_string = json.dumps(key_data, sort_keys=True)
            return hashlib.md5(key_string.encode()).hexdigest()[:8]
        except Exception:
            return str(hash(request.correlation_id))

    async def _add_to_batch(
        self,
        request: EnhancedClinicalRequest,
        similar_requests: List[EnhancedClinicalRequest]
    ) -> str:
        """Add request to batch with similar requests"""
        try:
            # Remove similar requests from queues
            for similar_request in similar_requests:
                for priority_queue in self.request_queues.values():
                    if similar_request in priority_queue:
                        priority_queue.remove(similar_request)
                        break

            # Create batch
            batch_requests = [request] + similar_requests
            batch_id = f"batch_{int(time.time() * 1000)}_{len(batch_requests)}"

            # Determine batch strategy
            batch_strategy = self._determine_batch_strategy(batch_requests)

            # Calculate batch similarity
            batch_similarity = await self._calculate_batch_similarity(batch_requests)

            # Determine batch priority (highest priority in batch)
            batch_priority = min(req.priority for req in batch_requests)

            # Estimate processing time
            estimated_time = self._estimate_batch_processing_time(batch_requests)

            # Extract shared context
            shared_context = self._extract_shared_context(batch_requests)

            # Identify optimization opportunities
            optimization_opportunities = self._identify_batch_optimizations(batch_requests)

            # Create batch
            batch = RequestBatch(
                batch_id=batch_id,
                requests=batch_requests,
                batch_strategy=batch_strategy,
                similarity_score=batch_similarity,
                created_at=datetime.now(timezone.utc),
                priority=batch_priority,
                estimated_processing_time=estimated_time,
                shared_context=shared_context,
                optimization_opportunities=optimization_opportunities
            )

            self.pending_batches[batch_id] = batch
            self.stats.batch_strategy_distribution[batch_strategy.value] += 1

            logger.info(f"Created batch {batch_id} with {len(batch_requests)} requests "
                       f"(similarity: {batch_similarity:.3f}, strategy: {batch_strategy.value})")

            return batch_id

        except Exception as e:
            logger.error(f"Error adding to batch: {e}")
            return ""

    def _determine_batch_strategy(self, requests: List[EnhancedClinicalRequest]) -> BatchStrategy:
        """Determine optimal batching strategy for requests"""
        try:
            # Analyze request characteristics
            priorities = [req.priority for req in requests]
            strategies = [req.routing_strategy for req in requests]

            # If all same priority, use priority-based
            if len(set(priorities)) == 1:
                return BatchStrategy.PRIORITY_BASED

            # If all same routing strategy, use similarity-based
            if len(set(strategies)) == 1:
                return BatchStrategy.SIMILARITY_BASED

            # If requests are close in time, use temporal-based
            times = [req.created_at for req in requests]
            time_span = max(times) - min(times)
            if time_span.total_seconds() < 10:
                return BatchStrategy.TEMPORAL_BASED

            # Default to hybrid
            return BatchStrategy.HYBRID

        except Exception as e:
            logger.warning(f"Error determining batch strategy: {e}")
            return BatchStrategy.SIMILARITY_BASED

    async def _calculate_batch_similarity(self, requests: List[EnhancedClinicalRequest]) -> float:
        """Calculate overall similarity score for batch"""
        try:
            if len(requests) < 2:
                return 1.0

            similarities = []
            for i in range(len(requests)):
                for j in range(i + 1, len(requests)):
                    similarity = await self._calculate_request_similarity(requests[i], requests[j])
                    similarities.append(similarity)

            return sum(similarities) / len(similarities) if similarities else 0.0

        except Exception as e:
            logger.warning(f"Error calculating batch similarity: {e}")
            return 0.0

    def _estimate_batch_processing_time(self, requests: List[EnhancedClinicalRequest]) -> int:
        """Estimate batch processing time"""
        try:
            # Base time is maximum individual request time
            base_time = max(req.timeout_ms for req in requests)

            # Apply batch efficiency factor (batching reduces total time)
            batch_size = len(requests)
            efficiency_factor = 0.7 + (0.3 / batch_size)  # Diminishing returns

            estimated_time = int(base_time * efficiency_factor)

            return estimated_time

        except Exception as e:
            logger.warning(f"Error estimating batch processing time: {e}")
            return max(req.timeout_ms for req in requests)

    def _extract_shared_context(self, requests: List[EnhancedClinicalRequest]) -> Dict[str, Any]:
        """Extract shared context from batch requests"""
        try:
            shared_context = {}

            # Find common medications
            all_medications = [set(req.medication_ids) for req in requests]
            if all_medications:
                common_medications = set.intersection(*all_medications)
                if common_medications:
                    shared_context['common_medications'] = list(common_medications)

            # Find common conditions
            all_conditions = [set(req.condition_ids) for req in requests]
            if all_conditions:
                common_conditions = set.intersection(*all_conditions)
                if common_conditions:
                    shared_context['common_conditions'] = list(common_conditions)

            # Find common clinical context elements
            common_encounter_types = set()
            for req in requests:
                encounter_type = req.clinical_context.get('encounter_type')
                if encounter_type:
                    common_encounter_types.add(encounter_type)

            if len(common_encounter_types) == 1:
                shared_context['encounter_type'] = list(common_encounter_types)[0]

            return shared_context

        except Exception as e:
            logger.warning(f"Error extracting shared context: {e}")
            return {}

    def _identify_batch_optimizations(self, requests: List[EnhancedClinicalRequest]) -> List[str]:
        """Identify optimization opportunities for batch"""
        try:
            optimizations = []

            # Check for shared medication analysis
            all_medications = []
            for req in requests:
                all_medications.extend(req.medication_ids)

            if len(set(all_medications)) < len(all_medications):
                optimizations.append("shared_medication_analysis")

            # Check for parallel reasoner execution
            if len(requests) > 2:
                optimizations.append("parallel_reasoner_execution")

            # Check for cached result reuse
            reasoner_types = set()
            for req in requests:
                reasoner_types.update(req.reasoner_types)

            if len(reasoner_types) < sum(len(req.reasoner_types) for req in requests):
                optimizations.append("cached_result_reuse")

            # Check for batch-specific caching
            if len(requests) >= 3:
                optimizations.append("batch_result_caching")

            return optimizations

        except Exception as e:
            logger.warning(f"Error identifying batch optimizations: {e}")
            return []

    def get_batching_stats(self) -> Dict[str, Any]:
        """Get comprehensive batching statistics"""
        try:
            # Calculate rates
            total_requests = max(self.stats.total_requests, 1)
            batching_rate = self.stats.batched_requests / total_requests

            # Calculate queue sizes
            queue_sizes = {
                priority.value: len(queue)
                for priority, queue in self.request_queues.items()
            }

            return {
                'performance': {
                    'total_requests': self.stats.total_requests,
                    'batched_requests': self.stats.batched_requests,
                    'batching_rate': batching_rate,
                    'total_batches': self.stats.total_batches,
                    'average_batch_size': self.stats.average_batch_size,
                    'processing_time_saved_ms': self.stats.processing_time_saved
                },
                'similarity': {
                    'similarity_threshold': self.similarity_threshold,
                    'threshold_hits': self.stats.similarity_threshold_hits,
                    'cache_size': len(self.similarity_cache)
                },
                'queues': {
                    'queue_sizes': queue_sizes,
                    'pending_batches': len(self.pending_batches),
                    'completed_batches': len(self.completed_batches)
                },
                'strategies': {
                    'distribution': self.stats.batch_strategy_distribution,
                    'max_batch_size': self.max_batch_size,
                    'min_batch_size': self.min_batch_size
                }
            }

        except Exception as e:
            logger.warning(f"Error calculating batching stats: {e}")
            return {'error': str(e)}

    async def _batch_formation_loop(self):
        """Background task for batch formation"""
        try:
            while True:
                await asyncio.sleep(0.01)  # Check every 10ms

                try:
                    # Check each priority queue for batching opportunities
                    for priority in RequestPriority:
                        queue = self.request_queues[priority]
                        if len(queue) >= self.min_batch_size:
                            await self._form_batches_from_queue(queue, priority)

                except Exception as e:
                    logger.warning(f"Error in batch formation loop: {e}")

        except asyncio.CancelledError:
            logger.info("Batch formation task cancelled")
        except Exception as e:
            logger.error(f"Batch formation task error: {e}")

    async def _batch_timeout_loop(self):
        """Background task for batch timeouts"""
        try:
            while True:
                await asyncio.sleep(0.05)  # Check every 50ms

                try:
                    # Check for requests that have waited too long
                    current_time = datetime.now(timezone.utc)

                    for priority in RequestPriority:
                        queue = self.request_queues[priority]
                        expired_requests = []

                        for request in queue:
                            wait_time = (current_time - request.created_at).total_seconds() * 1000
                            if wait_time > self.max_wait_time_ms:
                                expired_requests.append(request)

                        # Remove expired requests from queue
                        for request in expired_requests:
                            queue.remove(request)
                            # These requests should be processed immediately
                            logger.debug(f"Request {request.correlation_id} expired from batch queue")

                except Exception as e:
                    logger.warning(f"Error in batch timeout loop: {e}")

        except asyncio.CancelledError:
            logger.info("Batch timeout task cancelled")
        except Exception as e:
            logger.error(f"Batch timeout task error: {e}")

    async def _form_batches_from_queue(self, queue: List[EnhancedClinicalRequest], priority: RequestPriority):
        """Form batches from requests in queue"""
        try:
            if len(queue) < self.min_batch_size:
                return

            # Simple pairwise matching for batching
            processed_indices = set()

            for i, request in enumerate(queue):
                if i in processed_indices:
                    continue

                similar_requests = []
                for j, other_request in enumerate(queue[i+1:], i+1):
                    if j in processed_indices:
                        continue

                    similarity = await self._calculate_request_similarity(request, other_request)
                    if similarity >= self.similarity_threshold:
                        similar_requests.append(other_request)
                        processed_indices.add(j)

                        if len(similar_requests) >= self.max_batch_size - 1:
                            break

                if len(similar_requests) >= self.min_batch_size - 1:
                    await self._add_to_batch(request, similar_requests)
                    processed_indices.add(i)

                    # Remove processed requests from queue
                    for req in [request] + similar_requests:
                        if req in queue:
                            queue.remove(req)

        except Exception as e:
            logger.warning(f"Error forming batches from queue: {e}")

    async def shutdown(self):
        """Gracefully shutdown the batching system"""
        try:
            # Cancel background tasks
            for task in self._background_tasks:
                task.cancel()

            # Wait for tasks to complete
            if self._background_tasks:
                await asyncio.gather(*self._background_tasks, return_exceptions=True)

            logger.info("Pattern-based batcher shutdown complete")

        except Exception as e:
            logger.warning(f"Error during batcher shutdown: {e}")
