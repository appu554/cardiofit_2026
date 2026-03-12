"""
Prometheus metrics for Module 8 projectors

Provides standardized metrics for all storage projector services.
"""

from prometheus_client import Counter, Histogram, Gauge


class ProjectorMetrics:
    """
    Standard metrics for all Module 8 projectors

    Metrics:
    - projector_messages_consumed_total: Total messages consumed from Kafka
    - projector_messages_processed_total: Total messages successfully processed
    - projector_messages_failed_total: Total messages that failed processing
    - projector_batch_size: Batch size histogram
    - projector_batch_flush_duration_seconds: Batch flush duration histogram
    - projector_consumer_lag: Current consumer lag
    """

    def __init__(self, projector_name: str):
        """
        Initialize metrics for projector

        Args:
            projector_name: Unique projector identifier
        """
        self.projector_name = projector_name

        # Counter: Messages consumed from Kafka
        self.messages_consumed = Counter(
            'projector_messages_consumed_total',
            'Total messages consumed from Kafka',
            ['projector'],
        ).labels(projector=projector_name)

        # Counter: Messages successfully processed
        self.messages_processed = Counter(
            'projector_messages_processed_total',
            'Total messages successfully processed',
            ['projector'],
        ).labels(projector=projector_name)

        # Counter: Messages failed
        self.messages_failed = Counter(
            'projector_messages_failed_total',
            'Total messages that failed processing',
            ['projector'],
        ).labels(projector=projector_name)

        # Histogram: Batch size
        self.batch_size = Histogram(
            'projector_batch_size',
            'Batch size distribution',
            ['projector'],
            buckets=[10, 25, 50, 100, 250, 500, 1000],
        ).labels(projector=projector_name)

        # Histogram: Batch flush duration
        self.batch_flush_duration = Histogram(
            'projector_batch_flush_duration_seconds',
            'Batch flush duration in seconds',
            ['projector'],
            buckets=[0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0],
        ).labels(projector=projector_name)

        # Gauge: Consumer lag
        self.consumer_lag = Gauge(
            'projector_consumer_lag',
            'Current consumer lag (messages behind high water mark)',
            ['projector'],
        ).labels(projector=projector_name)
