"""
Adaptive Batching System for Kafka Producer

Implements intelligent batching with dynamic batch sizing, latency optimization,
and throughput balancing based on real-time metrics and device patterns.
"""

import asyncio
import logging
import time
from collections import defaultdict, deque
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional, Tuple, Callable
import json
import statistics

logger = logging.getLogger(__name__)


@dataclass
class BatchConfig:
    """Configuration for adaptive batching"""
    # Basic batching parameters
    min_batch_size: int = 1
    max_batch_size: int = 1000
    max_wait_time_ms: int = 30000  # 30 seconds
    
    # Adaptive parameters
    enable_adaptive_sizing: bool = True
    target_latency_ms: int = 100  # Target processing latency
    throughput_threshold_msgs_per_sec: int = 100
    
    # Device-specific batching
    enable_device_specific_batching: bool = True
    high_frequency_device_threshold: int = 10  # messages per minute
    
    # Performance tuning
    batch_size_adjustment_factor: float = 0.1  # 10% adjustment per iteration
    latency_tolerance_factor: float = 1.5  # 50% tolerance above target
    
    # Monitoring
    metrics_window_size: int = 100  # Number of batches to track for metrics
    performance_check_interval_ms: int = 5000  # 5 seconds


@dataclass
class BatchMetrics:
    """Metrics for batch performance tracking"""
    batch_sizes: deque = field(default_factory=lambda: deque(maxlen=100))
    processing_times: deque = field(default_factory=lambda: deque(maxlen=100))
    wait_times: deque = field(default_factory=lambda: deque(maxlen=100))
    throughput_samples: deque = field(default_factory=lambda: deque(maxlen=100))
    
    total_messages_processed: int = 0
    total_batches_processed: int = 0
    total_processing_time: float = 0.0
    
    last_adjustment_time: float = 0.0
    current_optimal_batch_size: int = 100


class AdaptiveBatchManager:
    """Manages adaptive batching for Kafka producer with performance optimization"""
    
    def __init__(self, config: BatchConfig):
        self.config = config
        self.metrics = BatchMetrics()
        self.device_patterns: Dict[str, Dict[str, Any]] = defaultdict(dict)
        self.pending_messages: List[Tuple[Dict[str, Any], float]] = []  # (message, timestamp)
        self.batch_processor: Optional[Callable] = None
        
        # Adaptive sizing state
        self.current_batch_size = config.min_batch_size
        self.last_performance_check = time.time()
        
        # Background tasks
        self.batch_processing_task: Optional[asyncio.Task] = None
        self.performance_monitoring_task: Optional[asyncio.Task] = None
        self.is_running = False
        
        # Locks for thread safety
        self._message_lock = asyncio.Lock()
        self._metrics_lock = asyncio.Lock()
    
    async def initialize(self, batch_processor: Callable):
        """Initialize the adaptive batch manager"""
        self.batch_processor = batch_processor
        self.is_running = True
        
        # Start background tasks
        self.batch_processing_task = asyncio.create_task(self._batch_processing_loop())
        self.performance_monitoring_task = asyncio.create_task(self._performance_monitoring_loop())
        
        logger.info("Adaptive batch manager initialized")
    
    async def add_message(self, message: Dict[str, Any]) -> bool:
        """Add message to batch queue"""
        if not self.is_running:
            return False
        
        async with self._message_lock:
            current_time = time.time()
            self.pending_messages.append((message, current_time))
            
            # Update device patterns
            device_id = message.get('device_id', 'unknown')
            await self._update_device_pattern(device_id, current_time)
            
            # Check if we should trigger immediate batch processing
            if await self._should_process_batch_immediately():
                # Signal batch processing task
                pass  # The loop will pick it up
        
        return True
    
    async def _batch_processing_loop(self):
        """Main batch processing loop"""
        while self.is_running:
            try:
                await asyncio.sleep(0.1)  # Small delay to prevent busy waiting
                
                if await self._should_process_batch():
                    await self._process_current_batch()
                    
            except Exception as e:
                logger.error(f"Error in batch processing loop: {e}")
    
    async def _should_process_batch(self) -> bool:
        """Determine if current batch should be processed"""
        async with self._message_lock:
            if not self.pending_messages:
                return False
            
            batch_size = len(self.pending_messages)
            oldest_message_time = self.pending_messages[0][1]
            current_time = time.time()
            wait_time_ms = (current_time - oldest_message_time) * 1000
            
            # Process if batch size reached
            if batch_size >= self.current_batch_size:
                return True
            
            # Process if max wait time exceeded
            if wait_time_ms >= self.config.max_wait_time_ms:
                return True
            
            # Process if we have minimum batch size and target latency would be exceeded
            if (batch_size >= self.config.min_batch_size and 
                wait_time_ms >= self.config.target_latency_ms):
                return True
            
            return False
    
    async def _should_process_batch_immediately(self) -> bool:
        """Check if batch should be processed immediately (high priority scenarios)"""
        batch_size = len(self.pending_messages)
        
        # Process immediately if max batch size reached
        if batch_size >= self.config.max_batch_size:
            return True
        
        # Process immediately for high-frequency devices if batch is substantial
        if batch_size >= self.current_batch_size * 0.8:  # 80% of current optimal size
            return True
        
        return False
    
    async def _process_current_batch(self):
        """Process the current batch of messages"""
        if not self.batch_processor:
            logger.warning("No batch processor configured")
            return
        
        async with self._message_lock:
            if not self.pending_messages:
                return
            
            # Extract messages for processing
            batch_messages = []
            batch_timestamps = []
            
            batch_size = min(len(self.pending_messages), self.current_batch_size)
            
            for _ in range(batch_size):
                message, timestamp = self.pending_messages.pop(0)
                batch_messages.append(message)
                batch_timestamps.append(timestamp)
        
        if not batch_messages:
            return
        
        # Process the batch
        start_time = time.time()
        oldest_timestamp = min(batch_timestamps)
        wait_time = start_time - oldest_timestamp
        
        try:
            # Call the batch processor
            await self.batch_processor(batch_messages)
            
            processing_time = time.time() - start_time
            
            # Update metrics
            await self._update_batch_metrics(
                batch_size=len(batch_messages),
                processing_time=processing_time,
                wait_time=wait_time
            )
            
            logger.debug(
                f"Processed batch: size={len(batch_messages)}, "
                f"processing_time={processing_time:.3f}s, "
                f"wait_time={wait_time:.3f}s"
            )
            
        except Exception as e:
            logger.error(f"Batch processing failed: {e}")
            # Could implement retry logic here
    
    async def _update_device_pattern(self, device_id: str, timestamp: float):
        """Update device messaging patterns for adaptive batching"""
        if device_id not in self.device_patterns:
            self.device_patterns[device_id] = {
                'message_timestamps': deque(maxlen=60),  # Last 60 messages
                'frequency': 0.0,
                'last_seen': timestamp,
                'message_count': 0
            }
        
        pattern = self.device_patterns[device_id]
        pattern['message_timestamps'].append(timestamp)
        pattern['last_seen'] = timestamp
        pattern['message_count'] += 1
        
        # Calculate frequency (messages per minute)
        if len(pattern['message_timestamps']) >= 2:
            time_span = pattern['message_timestamps'][-1] - pattern['message_timestamps'][0]
            if time_span > 0:
                pattern['frequency'] = (len(pattern['message_timestamps']) - 1) / (time_span / 60)
    
    async def _update_batch_metrics(self, batch_size: int, processing_time: float, wait_time: float):
        """Update batch processing metrics"""
        async with self._metrics_lock:
            self.metrics.batch_sizes.append(batch_size)
            self.metrics.processing_times.append(processing_time)
            self.metrics.wait_times.append(wait_time)
            
            self.metrics.total_messages_processed += batch_size
            self.metrics.total_batches_processed += 1
            self.metrics.total_processing_time += processing_time
            
            # Calculate throughput (messages per second)
            if processing_time > 0:
                throughput = batch_size / processing_time
                self.metrics.throughput_samples.append(throughput)
    
    async def _performance_monitoring_loop(self):
        """Monitor performance and adjust batch size"""
        while self.is_running:
            try:
                await asyncio.sleep(self.config.performance_check_interval_ms / 1000)
                
                if self.config.enable_adaptive_sizing:
                    await self._adjust_batch_size()
                    
            except Exception as e:
                logger.error(f"Error in performance monitoring: {e}")
    
    async def _adjust_batch_size(self):
        """Adjust batch size based on performance metrics"""
        async with self._metrics_lock:
            if len(self.metrics.processing_times) < 10:  # Need enough samples
                return
            
            # Calculate average metrics
            avg_processing_time = statistics.mean(self.metrics.processing_times)
            avg_wait_time = statistics.mean(self.metrics.wait_times)
            avg_throughput = statistics.mean(self.metrics.throughput_samples) if self.metrics.throughput_samples else 0
            
            # Convert to milliseconds for comparison
            avg_processing_time_ms = avg_processing_time * 1000
            avg_wait_time_ms = avg_wait_time * 1000
            
            # Determine if adjustment is needed
            target_latency = self.config.target_latency_ms
            latency_tolerance = target_latency * self.config.latency_tolerance_factor
            
            adjustment_needed = False
            new_batch_size = self.current_batch_size
            
            # If processing time is too high, reduce batch size
            if avg_processing_time_ms > latency_tolerance:
                adjustment_factor = 1 - self.config.batch_size_adjustment_factor
                new_batch_size = int(self.current_batch_size * adjustment_factor)
                adjustment_needed = True
                logger.debug(f"Reducing batch size due to high processing time: {avg_processing_time_ms:.1f}ms")
            
            # If processing time is well below target and throughput is low, increase batch size
            elif (avg_processing_time_ms < target_latency * 0.5 and 
                  avg_throughput < self.config.throughput_threshold_msgs_per_sec):
                adjustment_factor = 1 + self.config.batch_size_adjustment_factor
                new_batch_size = int(self.current_batch_size * adjustment_factor)
                adjustment_needed = True
                logger.debug(f"Increasing batch size for better throughput: {avg_throughput:.1f} msg/s")
            
            # Apply constraints
            new_batch_size = max(self.config.min_batch_size, 
                               min(self.config.max_batch_size, new_batch_size))
            
            if adjustment_needed and new_batch_size != self.current_batch_size:
                old_size = self.current_batch_size
                self.current_batch_size = new_batch_size
                self.metrics.current_optimal_batch_size = new_batch_size
                self.metrics.last_adjustment_time = time.time()
                
                logger.info(
                    f"Adjusted batch size: {old_size} → {new_batch_size} "
                    f"(avg_processing: {avg_processing_time_ms:.1f}ms, "
                    f"avg_throughput: {avg_throughput:.1f} msg/s)"
                )
    
    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get current performance metrics"""
        async with self._metrics_lock:
            if not self.metrics.processing_times:
                return {"status": "no_data"}
            
            avg_batch_size = statistics.mean(self.metrics.batch_sizes) if self.metrics.batch_sizes else 0
            avg_processing_time = statistics.mean(self.metrics.processing_times) if self.metrics.processing_times else 0
            avg_wait_time = statistics.mean(self.metrics.wait_times) if self.metrics.wait_times else 0
            avg_throughput = statistics.mean(self.metrics.throughput_samples) if self.metrics.throughput_samples else 0
            
            return {
                "current_batch_size": self.current_batch_size,
                "pending_messages": len(self.pending_messages),
                "total_messages_processed": self.metrics.total_messages_processed,
                "total_batches_processed": self.metrics.total_batches_processed,
                "avg_batch_size": round(avg_batch_size, 1),
                "avg_processing_time_ms": round(avg_processing_time * 1000, 2),
                "avg_wait_time_ms": round(avg_wait_time * 1000, 2),
                "avg_throughput_msg_per_sec": round(avg_throughput, 1),
                "device_patterns_tracked": len(self.device_patterns),
                "last_adjustment_time": self.metrics.last_adjustment_time,
                "is_running": self.is_running
            }
    
    async def get_device_patterns(self) -> Dict[str, Any]:
        """Get device messaging patterns"""
        patterns = {}
        current_time = time.time()
        
        for device_id, pattern in self.device_patterns.items():
            patterns[device_id] = {
                "frequency_msg_per_min": round(pattern['frequency'], 2),
                "message_count": pattern['message_count'],
                "last_seen_seconds_ago": round(current_time - pattern['last_seen'], 1),
                "is_high_frequency": pattern['frequency'] > self.config.high_frequency_device_threshold
            }
        
        return patterns
    
    async def cleanup(self):
        """Cleanup batch manager resources"""
        self.is_running = False
        
        if self.batch_processing_task:
            self.batch_processing_task.cancel()
            try:
                await self.batch_processing_task
            except asyncio.CancelledError:
                pass
        
        if self.performance_monitoring_task:
            self.performance_monitoring_task.cancel()
            try:
                await self.performance_monitoring_task
            except asyncio.CancelledError:
                pass
        
        # Process any remaining messages
        if self.pending_messages:
            await self._process_current_batch()
        
        logger.info("Adaptive batch manager cleaned up")
