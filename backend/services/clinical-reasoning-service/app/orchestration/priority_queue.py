"""
Priority Queue for Clinical Assertion Engine

Manages request queuing with priority-based processing and load balancing.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from enum import Enum
import heapq

from .request_router import ClinicalRequest, RequestPriority

logger = logging.getLogger(__name__)


@dataclass
class QueuedRequest:
    """Request in the priority queue"""
    priority_score: int
    created_at: datetime
    request: ClinicalRequest
    future: asyncio.Future
    
    def __lt__(self, other):
        """For heapq comparison (lower priority_score = higher priority)"""
        if self.priority_score != other.priority_score:
            return self.priority_score < other.priority_score
        # If same priority, older requests first
        return self.created_at < other.created_at


class QueueStats:
    """Queue statistics tracking"""
    def __init__(self):
        self.total_requests = 0
        self.processed_requests = 0
        self.queue_sizes = {p.value: 0 for p in RequestPriority}
        self.average_wait_times = {p.value: 0.0 for p in RequestPriority}
        self.timeout_count = 0
        self.current_queue_size = 0


class PriorityQueue:
    """
    Priority-based request queue with SLA enforcement
    
    Features:
    - Priority-based processing (CRITICAL -> HIGH -> NORMAL -> BATCH)
    - SLA timeout enforcement
    - Load balancing and backpressure
    - Queue statistics and monitoring
    - Circuit breaker integration
    """
    
    def __init__(self, max_queue_size: int = 1000, max_concurrent: int = 100):
        self.max_queue_size = max_queue_size
        self.max_concurrent = max_concurrent
        self.queue = []  # heapq
        self.processing_requests = set()
        self.stats = QueueStats()
        self.queue_lock = asyncio.Lock()
        self.processing_semaphore = asyncio.Semaphore(max_concurrent)
        
        # Priority scoring
        self.priority_scores = {
            RequestPriority.CRITICAL: 1,
            RequestPriority.HIGH: 2,
            RequestPriority.NORMAL: 3,
            RequestPriority.BATCH: 4
        }
        
        logger.info(f"Priority Queue initialized (max_queue_size={max_queue_size}, "
                   f"max_concurrent={max_concurrent})")
    
    async def enqueue_request(self, request: ClinicalRequest) -> asyncio.Future:
        """
        Enqueue a clinical request for processing
        
        Args:
            request: Clinical request to queue
            
        Returns:
            Future that will contain the processing result
        """
        async with self.queue_lock:
            # Check queue capacity
            if len(self.queue) >= self.max_queue_size:
                raise RuntimeError(f"Queue full (size: {len(self.queue)})")
            
            # Create future for result
            future = asyncio.Future()
            
            # Create queued request
            priority_score = self.priority_scores.get(request.priority, 999)
            queued_request = QueuedRequest(
                priority_score=priority_score,
                created_at=datetime.utcnow(),
                request=request,
                future=future
            )
            
            # Add to priority queue
            heapq.heappush(self.queue, queued_request)
            
            # Update statistics
            self.stats.total_requests += 1
            self.stats.queue_sizes[request.priority.value] += 1
            self.stats.current_queue_size = len(self.queue)
            
            logger.debug(f"Enqueued request for patient {request.patient_id} "
                        f"with priority {request.priority.value} "
                        f"(queue size: {len(self.queue)})")
            
            return future
    
    async def process_queue(self, processor_func):
        """
        Process requests from the queue
        
        Args:
            processor_func: Async function to process requests
        """
        logger.info("Starting queue processor")
        
        while True:
            try:
                # Get next request from queue
                queued_request = await self._get_next_request()
                
                if queued_request:
                    # Process request asynchronously
                    asyncio.create_task(
                        self._process_request(queued_request, processor_func)
                    )
                else:
                    # No requests available, wait briefly
                    await asyncio.sleep(0.01)
                    
            except Exception as e:
                logger.error(f"Error in queue processor: {e}")
                await asyncio.sleep(1)  # Brief pause on error
    
    async def _get_next_request(self) -> Optional[QueuedRequest]:
        """Get the next highest priority request from queue"""
        async with self.queue_lock:
            while self.queue:
                queued_request = heapq.heappop(self.queue)
                
                # Check if request has timed out
                if self._is_request_expired(queued_request):
                    self.stats.timeout_count += 1
                    queued_request.future.set_exception(
                        TimeoutError(f"Request timed out in queue after "
                                   f"{queued_request.request.timeout_ms}ms")
                    )
                    continue
                
                # Update queue size
                self.stats.current_queue_size = len(self.queue)
                return queued_request
            
            return None
    
    async def _process_request(self, queued_request: QueuedRequest, processor_func):
        """Process a single request with concurrency control"""
        
        # Acquire processing semaphore
        async with self.processing_semaphore:
            try:
                # Add to processing set
                self.processing_requests.add(queued_request.request.correlation_id)
                
                # Calculate wait time
                wait_time = (datetime.utcnow() - queued_request.created_at).total_seconds() * 1000
                
                # Check timeout again before processing
                if self._is_request_expired(queued_request):
                    self.stats.timeout_count += 1
                    queued_request.future.set_exception(
                        TimeoutError(f"Request timed out before processing")
                    )
                    return
                
                logger.debug(f"Processing request for patient {queued_request.request.patient_id} "
                           f"(waited {wait_time:.2f}ms)")
                
                # Process the request
                result = await processor_func(queued_request.request)
                
                # Set result
                queued_request.future.set_result(result)
                
                # Update statistics
                self._update_processing_stats(queued_request, wait_time)
                
            except Exception as e:
                logger.error(f"Error processing request: {e}")
                queued_request.future.set_exception(e)
                
            finally:
                # Remove from processing set
                self.processing_requests.discard(queued_request.request.correlation_id)
    
    def _is_request_expired(self, queued_request: QueuedRequest) -> bool:
        """Check if request has exceeded its timeout"""
        elapsed_ms = (datetime.utcnow() - queued_request.created_at).total_seconds() * 1000
        return elapsed_ms > queued_request.request.timeout_ms
    
    def _update_processing_stats(self, queued_request: QueuedRequest, wait_time_ms: float):
        """Update processing statistics"""
        self.stats.processed_requests += 1
        
        priority = queued_request.request.priority.value
        current_avg = self.stats.average_wait_times[priority]
        
        # Update average wait time for this priority
        if current_avg == 0:
            self.stats.average_wait_times[priority] = wait_time_ms
        else:
            # Simple moving average
            self.stats.average_wait_times[priority] = (current_avg + wait_time_ms) / 2
        
        # Decrement queue size for this priority
        if self.stats.queue_sizes[priority] > 0:
            self.stats.queue_sizes[priority] -= 1
    
    async def get_queue_status(self) -> Dict[str, Any]:
        """Get current queue status"""
        async with self.queue_lock:
            return {
                'current_queue_size': len(self.queue),
                'processing_count': len(self.processing_requests),
                'max_queue_size': self.max_queue_size,
                'max_concurrent': self.max_concurrent,
                'queue_utilization': len(self.queue) / self.max_queue_size,
                'processing_utilization': len(self.processing_requests) / self.max_concurrent,
                'priority_distribution': self.stats.queue_sizes.copy(),
                'average_wait_times': self.stats.average_wait_times.copy(),
                'total_requests': self.stats.total_requests,
                'processed_requests': self.stats.processed_requests,
                'timeout_count': self.stats.timeout_count
            }
    
    async def clear_expired_requests(self):
        """Clear expired requests from queue (maintenance task)"""
        async with self.queue_lock:
            valid_requests = []
            expired_count = 0
            
            while self.queue:
                queued_request = heapq.heappop(self.queue)
                
                if self._is_request_expired(queued_request):
                    expired_count += 1
                    self.stats.timeout_count += 1
                    queued_request.future.set_exception(
                        TimeoutError("Request expired during queue maintenance")
                    )
                else:
                    valid_requests.append(queued_request)
            
            # Rebuild heap with valid requests
            self.queue = valid_requests
            heapq.heapify(self.queue)
            
            if expired_count > 0:
                logger.info(f"Cleared {expired_count} expired requests from queue")
            
            self.stats.current_queue_size = len(self.queue)
    
    def get_stats(self) -> Dict[str, Any]:
        """Get queue statistics"""
        return {
            'total_requests': self.stats.total_requests,
            'processed_requests': self.stats.processed_requests,
            'current_queue_size': self.stats.current_queue_size,
            'timeout_count': self.stats.timeout_count,
            'queue_sizes_by_priority': self.stats.queue_sizes.copy(),
            'average_wait_times': self.stats.average_wait_times.copy(),
            'processing_count': len(self.processing_requests)
        }
