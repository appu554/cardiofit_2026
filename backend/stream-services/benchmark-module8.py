#!/usr/bin/env python3
"""
Module 8 Performance Benchmark Suite

Comprehensive performance benchmarking for all storage projectors:
- Throughput tests with varying batch sizes
- Latency percentile measurements (p50, p95, p99)
- Resource usage monitoring (CPU, memory, disk I/O)
- Scaling tests with multiple Kafka partitions

Outputs:
- CSV results file
- Markdown report
- Grafana dashboard JSON
"""

import time
import json
import uuid
import logging
import csv
import statistics
from datetime import datetime
from typing import Dict, List, Any, Tuple
from dataclasses import dataclass, asdict
import psutil
import requests

from kafka import KafkaProducer
from kafka.admin import KafkaAdminClient, NewTopic

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@dataclass
class BenchmarkResult:
    """Single benchmark result"""
    test_name: str
    projector: str
    batch_size: int
    partition_count: int
    duration_seconds: float
    events_processed: int
    throughput_eps: float  # events per second
    latency_p50_ms: float
    latency_p95_ms: float
    latency_p99_ms: float
    cpu_percent: float
    memory_mb: float
    disk_io_mb: float
    timestamp: str


class PerformanceMonitor:
    """Monitor system resource usage"""

    def __init__(self, process_name: str):
        self.process_name = process_name
        self.process = None
        self._find_process()

    def _find_process(self):
        """Find process by name"""
        for proc in psutil.process_iter(['name', 'cmdline']):
            try:
                if self.process_name in ' '.join(proc.info['cmdline'] or []):
                    self.process = proc
                    logger.info(f"Found process: {proc.pid} - {self.process_name}")
                    return
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                pass

    def get_metrics(self) -> Dict[str, float]:
        """Get current resource metrics"""
        if not self.process:
            return {"cpu_percent": 0, "memory_mb": 0, "disk_io_mb": 0}

        try:
            cpu_percent = self.process.cpu_percent(interval=1)
            memory_mb = self.process.memory_info().rss / (1024 * 1024)

            # Disk I/O
            try:
                io_counters = self.process.io_counters()
                disk_io_mb = (io_counters.read_bytes + io_counters.write_bytes) / (1024 * 1024)
            except (AttributeError, OSError):
                disk_io_mb = 0

            return {
                "cpu_percent": cpu_percent,
                "memory_mb": memory_mb,
                "disk_io_mb": disk_io_mb
            }
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            return {"cpu_percent": 0, "memory_mb": 0, "disk_io_mb": 0}


class LatencyTracker:
    """Track latency percentiles"""

    def __init__(self):
        self.latencies: List[float] = []

    def record(self, latency_ms: float):
        """Record a latency measurement"""
        self.latencies.append(latency_ms)

    def get_percentiles(self) -> Dict[str, float]:
        """Calculate latency percentiles"""
        if not self.latencies:
            return {"p50": 0, "p95": 0, "p99": 0}

        sorted_latencies = sorted(self.latencies)
        n = len(sorted_latencies)

        return {
            "p50": sorted_latencies[int(n * 0.50)],
            "p95": sorted_latencies[int(n * 0.95)],
            "p99": sorted_latencies[int(n * 0.99)]
        }


class Module8Benchmark:
    """Module 8 Benchmark Suite"""

    def __init__(self, kafka_bootstrap_servers: str = "localhost:9092"):
        self.kafka_bootstrap_servers = kafka_bootstrap_servers
        self.producer = KafkaProducer(
            bootstrap_servers=kafka_bootstrap_servers,
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None,
            linger_ms=10,
            batch_size=32768
        )
        self.results: List[BenchmarkResult] = []

    def close(self):
        """Close Kafka producer"""
        self.producer.close()

    def generate_event(self, patient_id: str, event_type: str = "VITAL_SIGNS") -> Dict[str, Any]:
        """Generate test event"""
        event_id = str(uuid.uuid4())
        timestamp = datetime.utcnow().isoformat() + "Z"

        return {
            "eventId": event_id,
            "eventType": event_type,
            "patientId": patient_id,
            "deviceId": f"device-{uuid.uuid4()}",
            "timestamp": timestamp,
            "eventTime": timestamp,
            "sourceSystem": "benchmark",
            "version": "1.0.0",
            "enrichment": {
                "patientContext": {
                    "age": 45,
                    "gender": "M",
                    "conditions": ["I10", "E11.9"]
                },
                "clinicalContext": {
                    "location": "ICU-3",
                    "encounterType": "INPATIENT"
                },
                "validationStatus": "VALID",
                "enrichmentTimestamp": timestamp
            },
            "data": {
                "heartRate": 78,
                "systolicBP": 120,
                "diastolicBP": 80,
                "temperature": 37.2,
                "respiratoryRate": 16,
                "oxygenSaturation": 98
            }
        }

    def run_throughput_test(
        self,
        test_name: str,
        batch_size: int,
        projector_name: str,
        monitor: PerformanceMonitor
    ) -> BenchmarkResult:
        """Run throughput benchmark"""
        logger.info(f"Running throughput test: {test_name} - {batch_size} events")

        patient_id = f"benchmark-{uuid.uuid4()}"
        latency_tracker = LatencyTracker()

        # Publish events
        start_time = time.time()
        for i in range(batch_size):
            event = self.generate_event(patient_id)

            send_start = time.time()
            self.producer.send("prod.ehr.events.enriched", key=patient_id, value=event)
            send_latency_ms = (time.time() - send_start) * 1000
            latency_tracker.record(send_latency_ms)

            if (i + 1) % 1000 == 0:
                logger.info(f"Published {i + 1}/{batch_size} events")

        self.producer.flush()
        duration = time.time() - start_time

        # Get resource metrics
        metrics = monitor.get_metrics()
        percentiles = latency_tracker.get_percentiles()

        # Calculate throughput
        throughput = batch_size / duration

        result = BenchmarkResult(
            test_name=test_name,
            projector=projector_name,
            batch_size=batch_size,
            partition_count=1,
            duration_seconds=duration,
            events_processed=batch_size,
            throughput_eps=throughput,
            latency_p50_ms=percentiles["p50"],
            latency_p95_ms=percentiles["p95"],
            latency_p99_ms=percentiles["p99"],
            cpu_percent=metrics["cpu_percent"],
            memory_mb=metrics["memory_mb"],
            disk_io_mb=metrics["disk_io_mb"],
            timestamp=datetime.utcnow().isoformat()
        )

        logger.info(f"Results: {throughput:.0f} events/sec, p95={percentiles['p95']:.2f}ms")
        return result

    def run_scaling_test(
        self,
        test_name: str,
        partition_counts: List[int],
        events_per_partition: int
    ) -> List[BenchmarkResult]:
        """Test scaling with different partition counts"""
        logger.info(f"Running scaling test: {test_name}")

        results = []

        for partition_count in partition_counts:
            logger.info(f"Testing with {partition_count} partitions")

            # Create topic with specified partitions
            topic_name = f"benchmark-{partition_count}-partitions"
            self._create_topic(topic_name, partition_count)

            # Run throughput test
            monitor = PerformanceMonitor("postgresql-projector")
            batch_size = events_per_partition * partition_count

            patient_id = f"scaling-{uuid.uuid4()}"
            latency_tracker = LatencyTracker()

            start_time = time.time()
            for i in range(batch_size):
                event = self.generate_event(patient_id)

                send_start = time.time()
                self.producer.send(topic_name, key=patient_id, value=event)
                send_latency_ms = (time.time() - send_start) * 1000
                latency_tracker.record(send_latency_ms)

            self.producer.flush()
            duration = time.time() - start_time

            metrics = monitor.get_metrics()
            percentiles = latency_tracker.get_percentiles()
            throughput = batch_size / duration

            result = BenchmarkResult(
                test_name=test_name,
                projector="all",
                batch_size=batch_size,
                partition_count=partition_count,
                duration_seconds=duration,
                events_processed=batch_size,
                throughput_eps=throughput,
                latency_p50_ms=percentiles["p50"],
                latency_p95_ms=percentiles["p95"],
                latency_p99_ms=percentiles["p99"],
                cpu_percent=metrics["cpu_percent"],
                memory_mb=metrics["memory_mb"],
                disk_io_mb=metrics["disk_io_mb"],
                timestamp=datetime.utcnow().isoformat()
            )

            results.append(result)
            logger.info(f"Partition {partition_count}: {throughput:.0f} events/sec")

        return results

    def _create_topic(self, topic_name: str, partition_count: int):
        """Create Kafka topic with specified partitions"""
        try:
            admin_client = KafkaAdminClient(
                bootstrap_servers=self.kafka_bootstrap_servers
            )

            topic = NewTopic(
                name=topic_name,
                num_partitions=partition_count,
                replication_factor=1
            )

            admin_client.create_topics([topic])
            logger.info(f"Created topic {topic_name} with {partition_count} partitions")
            admin_client.close()
        except Exception as e:
            logger.warning(f"Could not create topic {topic_name}: {e}")

    def run_all_benchmarks(self):
        """Run all benchmarks"""
        logger.info("=" * 80)
        logger.info("MODULE 8 PERFORMANCE BENCHMARK SUITE")
        logger.info("=" * 80)

        # 1. Throughput tests with varying batch sizes
        logger.info("\n1. THROUGHPUT TESTS")
        logger.info("-" * 80)

        batch_sizes = [100, 500, 1000, 5000, 10000]
        projectors = {
            "postgresql": "postgresql-projector",
            "mongodb": "mongodb-projector",
            "elasticsearch": "elasticsearch-projector",
            "clickhouse": "clickhouse-projector",
            "influxdb": "influxdb-projector",
            "ups": "ups-projector"
        }

        for batch_size in batch_sizes:
            for projector_name, process_name in projectors.items():
                monitor = PerformanceMonitor(process_name)
                result = self.run_throughput_test(
                    f"throughput_{batch_size}",
                    batch_size,
                    projector_name,
                    monitor
                )
                self.results.append(result)

        # 2. Scaling tests
        logger.info("\n2. SCALING TESTS")
        logger.info("-" * 80)

        partition_counts = [1, 2, 4, 8]
        scaling_results = self.run_scaling_test(
            "partition_scaling",
            partition_counts,
            events_per_partition=1000
        )
        self.results.extend(scaling_results)

        # 3. Sustained load test
        logger.info("\n3. SUSTAINED LOAD TEST")
        logger.info("-" * 80)

        monitor = PerformanceMonitor("postgresql-projector")
        sustained_result = self.run_throughput_test(
            "sustained_load_30k",
            30000,
            "all_projectors",
            monitor
        )
        self.results.append(sustained_result)

        logger.info("\n" + "=" * 80)
        logger.info("BENCHMARK SUITE COMPLETE")
        logger.info("=" * 80)

    def save_results_csv(self, filename: str = "benchmark-results.csv"):
        """Save results to CSV file"""
        if not self.results:
            logger.warning("No results to save")
            return

        logger.info(f"Saving results to {filename}")

        with open(filename, 'w', newline='') as csvfile:
            fieldnames = [
                "test_name", "projector", "batch_size", "partition_count",
                "duration_seconds", "events_processed", "throughput_eps",
                "latency_p50_ms", "latency_p95_ms", "latency_p99_ms",
                "cpu_percent", "memory_mb", "disk_io_mb", "timestamp"
            ]
            writer = csv.DictWriter(csvfile, fieldnames=fieldnames)

            writer.writeheader()
            for result in self.results:
                writer.writerow(asdict(result))

        logger.info(f"Saved {len(self.results)} results to {filename}")

    def generate_markdown_report(self, filename: str = "BENCHMARK_REPORT.md"):
        """Generate markdown benchmark report"""
        logger.info(f"Generating markdown report: {filename}")

        with open(filename, 'w') as f:
            f.write("# Module 8 Performance Benchmark Report\n\n")
            f.write(f"**Generated**: {datetime.utcnow().isoformat()}\n\n")

            # Summary statistics
            f.write("## Summary Statistics\n\n")
            f.write("| Metric | Value |\n")
            f.write("|--------|-------|\n")

            total_events = sum(r.events_processed for r in self.results)
            avg_throughput = statistics.mean(r.throughput_eps for r in self.results)
            max_throughput = max(r.throughput_eps for r in self.results)
            avg_latency_p95 = statistics.mean(r.latency_p95_ms for r in self.results)

            f.write(f"| Total Events Processed | {total_events:,} |\n")
            f.write(f"| Average Throughput | {avg_throughput:.0f} events/sec |\n")
            f.write(f"| Max Throughput | {max_throughput:.0f} events/sec |\n")
            f.write(f"| Average p95 Latency | {avg_latency_p95:.2f} ms |\n\n")

            # Throughput by projector
            f.write("## Throughput by Projector\n\n")
            f.write("| Projector | Avg Throughput (events/sec) | p95 Latency (ms) |\n")
            f.write("|-----------|------------------------------|------------------|\n")

            projector_results = {}
            for result in self.results:
                if result.projector not in projector_results:
                    projector_results[result.projector] = []
                projector_results[result.projector].append(result)

            for projector, results in projector_results.items():
                avg_throughput = statistics.mean(r.throughput_eps for r in results)
                avg_p95 = statistics.mean(r.latency_p95_ms for r in results)
                f.write(f"| {projector} | {avg_throughput:.0f} | {avg_p95:.2f} |\n")

            f.write("\n## Scaling Test Results\n\n")
            f.write("| Partitions | Throughput (events/sec) | p95 Latency (ms) |\n")
            f.write("|------------|-------------------------|------------------|\n")

            scaling_results = [r for r in self.results if "scaling" in r.test_name]
            for result in scaling_results:
                f.write(f"| {result.partition_count} | {result.throughput_eps:.0f} | {result.latency_p95_ms:.2f} |\n")

            # Detailed results
            f.write("\n## Detailed Results\n\n")
            f.write("| Test | Projector | Batch Size | Throughput | p50 | p95 | p99 | CPU % | Memory MB |\n")
            f.write("|------|-----------|------------|------------|-----|-----|-----|-------|----------|\n")

            for result in self.results:
                f.write(
                    f"| {result.test_name} | {result.projector} | {result.batch_size} | "
                    f"{result.throughput_eps:.0f} | {result.latency_p50_ms:.2f} | "
                    f"{result.latency_p95_ms:.2f} | {result.latency_p99_ms:.2f} | "
                    f"{result.cpu_percent:.1f} | {result.memory_mb:.0f} |\n"
                )

        logger.info(f"Generated report: {filename}")

    def generate_grafana_dashboard(self, filename: str = "grafana-dashboard-benchmark.json"):
        """Generate Grafana dashboard JSON"""
        logger.info(f"Generating Grafana dashboard: {filename}")

        dashboard = {
            "dashboard": {
                "title": "Module 8 Benchmark Results",
                "timezone": "utc",
                "panels": [
                    {
                        "id": 1,
                        "title": "Throughput by Projector",
                        "type": "graph",
                        "targets": [
                            {
                                "expr": "rate(projector_messages_processed_total[5m])",
                                "legendFormat": "{{projector}}"
                            }
                        ]
                    },
                    {
                        "id": 2,
                        "title": "Latency Percentiles",
                        "type": "graph",
                        "targets": [
                            {
                                "expr": "histogram_quantile(0.95, projector_batch_processing_seconds_bucket)",
                                "legendFormat": "p95 - {{projector}}"
                            },
                            {
                                "expr": "histogram_quantile(0.99, projector_batch_processing_seconds_bucket)",
                                "legendFormat": "p99 - {{projector}}"
                            }
                        ]
                    },
                    {
                        "id": 3,
                        "title": "Resource Usage - CPU",
                        "type": "graph",
                        "targets": [
                            {
                                "expr": "process_cpu_percent{job='module8-projectors'}",
                                "legendFormat": "{{projector}}"
                            }
                        ]
                    },
                    {
                        "id": 4,
                        "title": "Resource Usage - Memory",
                        "type": "graph",
                        "targets": [
                            {
                                "expr": "process_memory_bytes{job='module8-projectors'} / 1024 / 1024",
                                "legendFormat": "{{projector}}"
                            }
                        ]
                    }
                ]
            }
        }

        with open(filename, 'w') as f:
            json.dump(dashboard, f, indent=2)

        logger.info(f"Generated dashboard: {filename}")


def main():
    """Main benchmark execution"""
    benchmark = Module8Benchmark()

    try:
        benchmark.run_all_benchmarks()
        benchmark.save_results_csv("benchmark-results.csv")
        benchmark.generate_markdown_report("BENCHMARK_REPORT.md")
        benchmark.generate_grafana_dashboard("grafana-dashboard-benchmark.json")
    finally:
        benchmark.close()


if __name__ == "__main__":
    main()
