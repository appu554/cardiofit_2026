"""
Kafka monitoring and observability for Clinical Synthesis Hub
"""

import json
import logging
import time
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
from dataclasses import dataclass, asdict
import threading
from collections import defaultdict, deque

try:
    from confluent_kafka import Consumer
    try:
        from confluent_kafka.admin import AdminClient, ConfigResource, ResourceType
    except ImportError:
        AdminClient = None
        ConfigResource = None
        ResourceType = None
except ImportError:
    Consumer = None
    AdminClient = None
    ConfigResource = None
    ResourceType = None

from .config import kafka_config

logger = logging.getLogger(__name__)

@dataclass
class TopicMetrics:
    """Metrics for a Kafka topic"""
    topic_name: str
    partition_count: int
    replication_factor: int
    message_count: int = 0
    bytes_in_per_sec: float = 0.0
    bytes_out_per_sec: float = 0.0
    messages_in_per_sec: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

@dataclass
class ConsumerGroupMetrics:
    """Metrics for a consumer group"""
    group_id: str
    state: str
    members: int
    total_lag: int = 0
    max_lag: int = 0
    topics: List[str] = None
    
    def __post_init__(self):
        if self.topics is None:
            self.topics = []
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

@dataclass
class ProducerMetrics:
    """Metrics for a producer"""
    client_id: str
    messages_sent: int = 0
    messages_failed: int = 0
    bytes_sent: int = 0
    avg_latency_ms: float = 0.0
    error_rate: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

class KafkaMonitor:
    """Kafka monitoring and metrics collection"""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize Kafka monitor"""
        if AdminClient is None:
            logger.warning("AdminClient not available, monitoring will be limited")
            self.admin_client = None
        else:
            self.config = config or kafka_config.get_producer_config()
            self.admin_client = AdminClient(self.config)
        
        # Metrics storage
        self.topic_metrics: Dict[str, TopicMetrics] = {}
        self.consumer_group_metrics: Dict[str, ConsumerGroupMetrics] = {}
        self.producer_metrics: Dict[str, ProducerMetrics] = {}
        
        # Time series data (last 100 data points)
        self.metrics_history = defaultdict(lambda: deque(maxlen=100))
        
        # Monitoring state
        self.monitoring = False
        self.monitor_thread = None
        self.monitor_interval = 30  # seconds
        
        logger.info("KafkaMonitor initialized")
    
    def start_monitoring(self, interval: int = 30):
        """Start background monitoring"""
        if self.monitoring:
            logger.warning("Monitoring already started")
            return
        
        self.monitor_interval = interval
        self.monitoring = True
        self.monitor_thread = threading.Thread(target=self._monitor_loop, daemon=True)
        self.monitor_thread.start()
        
        logger.info("Started Kafka monitoring with %d second interval", interval)
    
    def stop_monitoring(self):
        """Stop background monitoring"""
        self.monitoring = False
        if self.monitor_thread:
            self.monitor_thread.join(timeout=5)
        logger.info("Stopped Kafka monitoring")
    
    def _monitor_loop(self):
        """Background monitoring loop"""
        while self.monitoring:
            try:
                self.collect_metrics()
                time.sleep(self.monitor_interval)
            except Exception as e:
                logger.error("Error in monitoring loop: %s", e)
                time.sleep(self.monitor_interval)
    
    def collect_metrics(self):
        """Collect all Kafka metrics"""
        try:
            self._collect_topic_metrics()
            self._collect_consumer_group_metrics()
            self._store_metrics_history()
        except Exception as e:
            logger.error("Error collecting metrics: %s", e)
    
    def _collect_topic_metrics(self):
        """Collect topic-level metrics"""
        if self.admin_client is None:
            logger.debug("AdminClient not available, skipping topic metrics")
            return

        try:
            # Get topic metadata
            metadata = self.admin_client.list_topics(timeout=10)

            for topic_name, topic_metadata in metadata.topics.items():
                if topic_metadata.error is not None:
                    continue

                metrics = TopicMetrics(
                    topic_name=topic_name,
                    partition_count=len(topic_metadata.partitions),
                    replication_factor=len(topic_metadata.partitions[0].replicas) if topic_metadata.partitions else 0
                )

                self.topic_metrics[topic_name] = metrics

        except Exception as e:
            logger.error("Error collecting topic metrics: %s", e)
    
    def _collect_consumer_group_metrics(self):
        """Collect consumer group metrics"""
        try:
            # This is a simplified implementation
            # In a full implementation, you'd use the AdminClient to get consumer group info
            # For now, we'll create placeholder metrics
            
            # Get list of consumer groups (this would require additional API calls)
            # For demonstration, we'll use known groups
            known_groups = [
                'clinical-synthesis-hub',
                'clinical-synthesis-hub-device-transformer',
                'clinical-synthesis-hub-fhir-loader',
                'clinical-synthesis-hub-read-model-projector'
            ]
            
            for group_id in known_groups:
                metrics = ConsumerGroupMetrics(
                    group_id=group_id,
                    state='Stable',  # Would get from API
                    members=1,       # Would get from API
                    total_lag=0,     # Would calculate from partition offsets
                    max_lag=0        # Would calculate from partition offsets
                )
                
                self.consumer_group_metrics[group_id] = metrics
                
        except Exception as e:
            logger.error("Error collecting consumer group metrics: %s", e)
    
    def _store_metrics_history(self):
        """Store current metrics in time series history"""
        timestamp = datetime.now().isoformat()
        
        # Store topic metrics
        for topic_name, metrics in self.topic_metrics.items():
            self.metrics_history[f"topic.{topic_name}.partitions"].append({
                'timestamp': timestamp,
                'value': metrics.partition_count
            })
        
        # Store consumer group metrics
        for group_id, metrics in self.consumer_group_metrics.items():
            self.metrics_history[f"consumer_group.{group_id}.lag"].append({
                'timestamp': timestamp,
                'value': metrics.total_lag
            })
    
    def get_topic_metrics(self, topic_name: Optional[str] = None) -> Dict[str, TopicMetrics]:
        """Get topic metrics"""
        if topic_name:
            return {topic_name: self.topic_metrics.get(topic_name)}
        return self.topic_metrics.copy()
    
    def get_consumer_group_metrics(self, group_id: Optional[str] = None) -> Dict[str, ConsumerGroupMetrics]:
        """Get consumer group metrics"""
        if group_id:
            return {group_id: self.consumer_group_metrics.get(group_id)}
        return self.consumer_group_metrics.copy()
    
    def get_producer_metrics(self, client_id: Optional[str] = None) -> Dict[str, ProducerMetrics]:
        """Get producer metrics"""
        if client_id:
            return {client_id: self.producer_metrics.get(client_id)}
        return self.producer_metrics.copy()
    
    def get_metrics_history(self, metric_name: str, hours: int = 1) -> List[Dict[str, Any]]:
        """Get historical metrics data"""
        cutoff_time = datetime.now() - timedelta(hours=hours)
        
        history = self.metrics_history.get(metric_name, [])
        return [
            point for point in history
            if datetime.fromisoformat(point['timestamp']) > cutoff_time
        ]
    
    def get_health_status(self) -> Dict[str, Any]:
        """Get overall Kafka cluster health status"""
        if self.admin_client is None:
            return {
                'status': 'limited',
                'error': 'AdminClient not available',
                'last_updated': datetime.now().isoformat()
            }

        try:
            # Test connectivity
            metadata = self.admin_client.list_topics(timeout=5)

            # Count topics and partitions
            total_topics = len(metadata.topics)
            total_partitions = sum(
                len(topic.partitions)
                for topic in metadata.topics.values()
                if topic.error is None
            )

            # Calculate consumer lag
            total_lag = sum(
                metrics.total_lag
                for metrics in self.consumer_group_metrics.values()
            )

            return {
                'status': 'healthy',
                'cluster_id': metadata.cluster_id,
                'broker_count': len(metadata.brokers),
                'topic_count': total_topics,
                'partition_count': total_partitions,
                'total_consumer_lag': total_lag,
                'last_updated': datetime.now().isoformat()
            }

        except Exception as e:
            return {
                'status': 'unhealthy',
                'error': str(e),
                'last_updated': datetime.now().isoformat()
            }
    
    def get_alerts(self) -> List[Dict[str, Any]]:
        """Get current alerts based on metrics"""
        alerts = []
        
        # Check consumer lag
        for group_id, metrics in self.consumer_group_metrics.items():
            if metrics.total_lag > 1000:  # Threshold
                alerts.append({
                    'type': 'high_consumer_lag',
                    'severity': 'warning' if metrics.total_lag < 5000 else 'critical',
                    'message': f"High consumer lag for group {group_id}: {metrics.total_lag}",
                    'group_id': group_id,
                    'lag': metrics.total_lag
                })
        
        # Check producer error rates
        for client_id, metrics in self.producer_metrics.items():
            if metrics.error_rate > 0.05:  # 5% error rate threshold
                alerts.append({
                    'type': 'high_error_rate',
                    'severity': 'warning' if metrics.error_rate < 0.1 else 'critical',
                    'message': f"High error rate for producer {client_id}: {metrics.error_rate:.2%}",
                    'client_id': client_id,
                    'error_rate': metrics.error_rate
                })
        
        return alerts
    
    def export_metrics(self) -> Dict[str, Any]:
        """Export all metrics in a structured format"""
        return {
            'timestamp': datetime.now().isoformat(),
            'topics': {name: metrics.to_dict() for name, metrics in self.topic_metrics.items()},
            'consumer_groups': {name: metrics.to_dict() for name, metrics in self.consumer_group_metrics.items()},
            'producers': {name: metrics.to_dict() for name, metrics in self.producer_metrics.items()},
            'health': self.get_health_status(),
            'alerts': self.get_alerts()
        }

# Global monitor instance
_monitor_instance = None

def get_kafka_monitor() -> KafkaMonitor:
    """Get global Kafka monitor instance"""
    global _monitor_instance
    if _monitor_instance is None:
        _monitor_instance = KafkaMonitor()
    return _monitor_instance
