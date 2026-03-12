"""
Generic batch processor with size and time-based flushing

Thread-safe batch accumulator for projector services.
"""

import threading
import time
from typing import List, Callable, Any, Optional

import structlog


logger = structlog.get_logger(__name__)


class BatchProcessor:
    """
    Generic batch accumulator with configurable flushing

    Features:
    - Size-based flushing (flush when batch reaches max size)
    - Time-based flushing (flush after timeout even if batch not full)
    - Thread-safe operations
    - Metrics tracking
    """

    def __init__(
        self,
        batch_size: int,
        batch_timeout_seconds: float,
        flush_callback: Callable[[List[Any]], None],
    ):
        """
        Initialize batch processor

        Args:
            batch_size: Maximum number of items per batch
            batch_timeout_seconds: Maximum time to wait before flushing
            flush_callback: Function to call when batch is flushed
        """
        self.batch_size = batch_size
        self.batch_timeout_seconds = batch_timeout_seconds
        self.flush_callback = flush_callback

        # Batch storage
        self.batch: List[Any] = []
        self.batch_start_time = time.time()

        # Thread safety
        self.lock = threading.RLock()

        # Timer for timeout-based flushing
        self.timer: Optional[threading.Timer] = None

        logger.info(
            "Batch processor initialized",
            batch_size=batch_size,
            batch_timeout=batch_timeout_seconds,
        )

    def add(self, item: Any) -> None:
        """
        Add item to batch

        Args:
            item: Item to add to batch
        """
        with self.lock:
            self.batch.append(item)

            # Check if batch is full
            if len(self.batch) >= self.batch_size:
                self.flush()
            elif len(self.batch) == 1:
                # Start timer when first item is added
                self._start_timer()

    def flush(self) -> None:
        """Flush current batch"""
        with self.lock:
            if not self.batch:
                return

            # Cancel timer
            if self.timer:
                self.timer.cancel()
                self.timer = None

            # Get current batch
            current_batch = self.batch.copy()
            batch_size = len(current_batch)

            # Clear batch
            self.batch = []
            self.batch_start_time = time.time()

            logger.debug("Flushing batch", batch_size=batch_size)

        # Call flush callback outside lock to prevent deadlock
        try:
            self.flush_callback(current_batch)
        except Exception as e:
            logger.error("Flush callback failed", error=str(e), batch_size=batch_size)
            raise

    def _start_timer(self) -> None:
        """Start timeout timer for batch flushing"""
        if self.timer:
            self.timer.cancel()

        self.timer = threading.Timer(self.batch_timeout_seconds, self._timeout_flush)
        self.timer.daemon = True
        self.timer.start()

    def _timeout_flush(self) -> None:
        """Flush batch due to timeout"""
        with self.lock:
            if self.batch:
                logger.debug(
                    "Batch timeout reached, flushing",
                    batch_size=len(self.batch),
                    timeout=self.batch_timeout_seconds,
                )
                self.flush()

    def get_current_batch_size(self) -> int:
        """Get current batch size"""
        with self.lock:
            return len(self.batch)

    def get_batch_age(self) -> float:
        """Get age of current batch in seconds"""
        with self.lock:
            return time.time() - self.batch_start_time
