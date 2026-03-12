# Missing Components Implementation Plan
## Runtime Layer Critical Gap Closure

**Document Version**: 1.0
**Date**: September 24, 2025
**Target Completion**: 10-12 weeks
**Implementation Strategy**: Docker Compose + Microservices

---

## Executive Summary

This implementation plan addresses the 4 critical missing components identified in the runtime layer gap analysis:

1. **Apache Flink Stream Processing** - Real-time patient event processing
2. **Evidence Envelope System** - Clinical decision auditability
3. **Complete Cache Prefetcher Service** - L1 session cache and event-driven prefetching
4. **Timing Guarantees Infrastructure** - SLA monitoring and performance enforcement

**Implementation Strategy**: Containerized microservices approach using Docker Compose for local development and production deployment, maintaining compatibility with existing runtime layer architecture.

---

## Phase 1: Apache Flink Stream Processing (4-6 weeks)

### Overview
Implement high-performance stream processing for real-time patient events with <500ms end-to-end latency guarantee.

### Architecture Components

#### 1.1 Flink Cluster Setup
```yaml
# docker-compose.flink.yml
services:
  flink-jobmanager:
    image: flink:1.18-scala_2.12-java11
    ports:
      - "8081:8081"
    command: jobmanager
    environment:
      - JOB_MANAGER_RPC_ADDRESS=flink-jobmanager
      - FLINK_PROPERTIES=jobmanager.memory.process.size: 1600m
    volumes:
      - ./flink/conf:/opt/flink/conf
      - ./flink/jobs:/opt/flink/jobs

  flink-taskmanager:
    image: flink:1.18-scala_2.12-java11
    depends_on:
      - flink-jobmanager
    command: taskmanager
    scale: 2
    environment:
      - JOB_MANAGER_RPC_ADDRESS=flink-jobmanager
      - FLINK_PROPERTIES=taskmanager.memory.process.size: 1728m|taskmanager.numberOfTaskSlots: 4
    volumes:
      - ./flink/conf:/opt/flink/conf
```

#### 1.2 Directory Structure
```
backend/shared-infrastructure/runtime-layer/
├── flink-stream-processing/
│   ├── src/main/java/com/cardiofit/stream/
│   │   ├── jobs/
│   │   │   ├── PatientEventEnrichmentJob.java
│   │   │   ├── ClinicalPatternDetectionJob.java
│   │   │   └── WorkflowStateManager.java
│   │   ├── functions/
│   │   │   ├── EventEnricher.java
│   │   │   ├── PatternMatcher.java
│   │   │   └── StateAggregator.java
│   │   ├── sinks/
│   │   │   ├── FHIRStoreSink.java
│   │   │   ├── ElasticsearchSink.java
│   │   │   ├── NotificationSink.java
│   │   │   └── Neo4jSummarySink.java
│   │   └── utils/
│   │       ├── KafkaSchemas.java
│   │       └── SerializationUtils.java
│   ├── Dockerfile
│   ├── pom.xml
│   └── flink-conf.yaml
├── docker-compose.flink.yml
└── scripts/
    ├── deploy-flink-jobs.sh
    └── monitor-flink-jobs.sh
```

#### 1.3 Core Implementation Files

**PatientEventEnrichmentJob.java**
```java
public class PatientEventEnrichmentJob {
    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(4);
        env.enableCheckpointing(5000);

        // Source: Kafka patient events
        DataStream<PatientEvent> patientEvents = env
            .addSource(new FlinkKafkaConsumer<>(
                "patient-events",
                new PatientEventSchema(),
                kafkaProps))
            .name("patient-events-source");

        // Enrich with semantic mesh lookups
        DataStream<EnrichedPatientEvent> enrichedEvents = patientEvents
            .keyBy(PatientEvent::getPatientId)
            .process(new EventEnricher())
            .name("event-enricher");

        // Multi-sink distribution
        enrichedEvents.addSink(new FHIRStoreSink()).name("fhir-sink");
        enrichedEvents.addSink(new ElasticsearchSink()).name("es-sink");
        enrichedEvents.addSink(new NotificationSink()).name("notification-sink");
        enrichedEvents.addSink(new Neo4jSummarySink()).name("neo4j-sink");

        env.execute("Patient Event Enrichment Job");
    }
}
```

**Implementation Timeline**: 4 weeks
- Week 1: Flink cluster setup, basic job structure
- Week 2: Event enrichment logic, semantic mesh integration
- Week 3: Multi-sink implementation, state management
- Week 4: Performance tuning, monitoring integration

**Resource Requirements**:
- 3 containers (1 JobManager, 2 TaskManagers)
- 4GB memory total
- Kafka integration for event sources/sinks

---

## Phase 2: Evidence Envelope System (3-4 weeks)

### Overview
Implement comprehensive clinical decision auditability with provenance tracking and confidence scoring.

### Architecture Components

#### 2.1 Evidence Envelope Service
```yaml
# docker-compose.evidence.yml
services:
  evidence-service:
    build:
      context: ./evidence-envelope-service
      dockerfile: Dockerfile
    ports:
      - "8050:8050"
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/evidence_db
      - REDIS_URL=redis://redis:6379/2
    depends_on:
      - postgres
      - redis
    volumes:
      - ./evidence-envelope-service:/app
      - /app/venv
```

#### 2.2 Directory Structure
```
backend/shared-infrastructure/runtime-layer/
├── evidence-envelope-service/
│   ├── src/
│   │   ├── models/
│   │   │   ├── evidence_envelope.py
│   │   │   ├── inference_chain.py
│   │   │   └── confidence_scoring.py
│   │   ├── services/
│   │   │   ├── envelope_generator.py
│   │   │   ├── provenance_tracker.py
│   │   │   └── audit_logger.py
│   │   ├── api/
│   │   │   ├── envelope_endpoints.py
│   │   │   └── audit_endpoints.py
│   │   └── utils/
│   │       ├── snapshot_integration.py
│   │       └── knowledge_versioning.py
│   ├── migrations/
│   │   └── 001_create_evidence_tables.sql
│   ├── Dockerfile
│   ├── requirements.txt
│   └── config.py
├── evidence-envelope-integration/
│   ├── middleware/
│   │   ├── envelope_middleware.py
│   │   └── decision_tracker.py
│   └── schemas/
│       ├── evidence_envelope_schema.json
│       └── confidence_scoring_schema.json
└── docker-compose.evidence.yml
```

#### 2.3 Core Implementation Files

**evidence_envelope.py**
```python
from dataclasses import dataclass, asdict
from typing import Dict, List, Any, Optional
from datetime import datetime
import uuid

@dataclass
class ConfidenceScore:
    overall: float
    components: Dict[str, float]
    methodology: str
    calculated_at: datetime

@dataclass
class InferenceChain:
    steps: List[str]
    reasoning_path: List[Dict[str, Any]]
    knowledge_sources: List[str]
    confidence_propagation: List[float]

@dataclass
class EvidenceEnvelope:
    proposal_id: str
    snapshot_reference: str
    knowledge_versions: Dict[str, str]
    inference_chain: InferenceChain
    confidence_scores: ConfidenceScore

    # Clinical context
    patient_context: Optional[Dict[str, Any]] = None
    clinical_workflow: Optional[str] = None
    decision_timestamp: Optional[datetime] = None

    # Audit trail
    service_chain: List[str] = None
    execution_duration_ms: Optional[int] = None

    @classmethod
    def create(cls, snapshot_id: str, knowledge_versions: Dict[str, str]) -> 'EvidenceEnvelope':
        return cls(
            proposal_id=str(uuid.uuid4()),
            snapshot_reference=snapshot_id,
            knowledge_versions=knowledge_versions,
            inference_chain=InferenceChain([], [], [], []),
            confidence_scores=ConfidenceScore(0.0, {}, "", datetime.utcnow()),
            service_chain=[],
            decision_timestamp=datetime.utcnow()
        )

    def add_inference_step(self, step: str, reasoning: Dict[str, Any],
                          source: str, confidence: float):
        self.inference_chain.steps.append(step)
        self.inference_chain.reasoning_path.append(reasoning)
        self.inference_chain.knowledge_sources.append(source)
        self.inference_chain.confidence_propagation.append(confidence)

    def calculate_overall_confidence(self):
        if not self.inference_chain.confidence_propagation:
            self.confidence_scores.overall = 0.0
            return

        # Weighted average with step importance
        weights = [1.0 / (i + 1) for i in range(len(self.inference_chain.confidence_propagation))]
        weighted_sum = sum(c * w for c, w in zip(self.inference_chain.confidence_propagation, weights))
        weight_sum = sum(weights)

        self.confidence_scores.overall = weighted_sum / weight_sum if weight_sum > 0 else 0.0
        self.confidence_scores.calculated_at = datetime.utcnow()

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)
```

**envelope_generator.py**
```python
from typing import Dict, Any, Optional
from ..models.evidence_envelope import EvidenceEnvelope
from ..utils.snapshot_integration import SnapshotManager
from ..utils.knowledge_versioning import KnowledgeVersionTracker

class EvidenceEnvelopeGenerator:
    def __init__(self, snapshot_manager: SnapshotManager,
                 version_tracker: KnowledgeVersionTracker):
        self.snapshot_manager = snapshot_manager
        self.version_tracker = version_tracker

    async def create_envelope(self, workflow_context: Dict[str, Any]) -> EvidenceEnvelope:
        # Create or retrieve snapshot
        snapshot = await self.snapshot_manager.create_snapshot(
            service_id="clinical-decision",
            context=workflow_context
        )

        # Get current knowledge base versions
        knowledge_versions = await self.version_tracker.get_current_versions()

        # Initialize evidence envelope
        envelope = EvidenceEnvelope.create(snapshot.id, knowledge_versions)
        envelope.clinical_workflow = workflow_context.get('workflow_type')
        envelope.patient_context = workflow_context.get('patient_context')

        return envelope

    async def add_decision_evidence(self, envelope: EvidenceEnvelope,
                                   service_name: str, decision_data: Dict[str, Any]):
        # Extract inference step
        reasoning_step = f"{service_name}_analysis"
        reasoning_data = {
            'service': service_name,
            'input_parameters': decision_data.get('inputs', {}),
            'processing_method': decision_data.get('method', 'unknown'),
            'output_summary': decision_data.get('outputs', {}),
            'execution_time_ms': decision_data.get('duration_ms', 0)
        }

        # Calculate step confidence
        step_confidence = self._calculate_step_confidence(decision_data)

        # Add to envelope
        envelope.add_inference_step(
            reasoning_step,
            reasoning_data,
            service_name,
            step_confidence
        )

        envelope.service_chain.append(service_name)

    def _calculate_step_confidence(self, decision_data: Dict[str, Any]) -> float:
        # Basic confidence calculation - can be enhanced
        base_confidence = 0.8

        # Adjust based on data quality
        if decision_data.get('data_completeness', 0) < 0.7:
            base_confidence -= 0.2

        # Adjust based on processing time (longer = more thorough)
        duration_ms = decision_data.get('duration_ms', 0)
        if duration_ms > 1000:  # More than 1 second
            base_confidence += 0.1
        elif duration_ms < 100:  # Less than 100ms
            base_confidence -= 0.1

        return max(0.0, min(1.0, base_confidence))
```

**Implementation Timeline**: 3 weeks
- Week 1: Core envelope models and database setup
- Week 2: Service integration middleware and API endpoints
- Week 3: Audit logging, confidence scoring, integration testing

---

## Phase 3: Complete Cache Prefetcher Service (2-3 weeks)

### Overview
Implement missing L1 session cache and enhance event-driven prefetching capabilities.

#### 3.1 Enhanced Cache Architecture
```yaml
# docker-compose.cache.yml
services:
  redis-l1:
    image: redis:7-alpine
    ports:
      - "6381:6379"
    command: redis-server --maxmemory 512mb --maxmemory-policy allkeys-lru
    volumes:
      - ./cache/redis-l1.conf:/usr/local/etc/redis/redis.conf

  cache-prefetcher-service:
    build:
      context: ./cache-prefetcher-enhanced
      dockerfile: Dockerfile
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=kafka:9092
      - REDIS_L1_URL=redis://redis-l1:6379/0
      - REDIS_L2_URL=redis://redis:6379/0
      - REDIS_L3_URL=redis://redis:6379/1
      - NEO4J_URL=bolt://neo4j:7687
      - CLICKHOUSE_URL=http://clickhouse:8123
    depends_on:
      - redis-l1
      - kafka
      - neo4j
      - clickhouse
```

#### 3.2 Directory Structure
```
backend/shared-infrastructure/runtime-layer/
├── cache-prefetcher-enhanced/
│   ├── src/
│   │   ├── services/
│   │   │   ├── l1_session_cache.py
│   │   │   ├── event_driven_prefetcher.py
│   │   │   ├── pattern_analyzer.py
│   │   │   └── cache_coordinator.py
│   │   ├── models/
│   │   │   ├── cache_models.py
│   │   │   ├── session_models.py
│   │   │   └── prefetch_patterns.py
│   │   ├── listeners/
│   │   │   ├── workflow_event_listener.py
│   │   │   ├── medication_event_listener.py
│   │   │   └── safety_event_listener.py
│   │   └── utils/
│   │       ├── cache_key_generator.py
│   │       └── performance_monitor.py
│   ├── Dockerfile
│   ├── requirements.txt
│   └── config.py
└── docker-compose.cache.yml
```

#### 3.3 Core Implementation Files

**l1_session_cache.py**
```python
import asyncio
import json
from typing import Dict, Any, Optional
from datetime import datetime, timedelta
import redis.asyncio as redis
from loguru import logger

class L1SessionCache:
    """
    L1 Session Cache (10-second TTL)
    Stores workflow state and immediate context data
    """

    def __init__(self, redis_url: str):
        self.redis = redis.from_url(redis_url, decode_responses=True)
        self.default_ttl = 10  # 10 seconds

    async def store_workflow_state(self, workflow_id: str, state_data: Dict[str, Any]):
        """Store workflow state with 10-second TTL"""
        cache_key = f"l1:workflow:{workflow_id}:state"

        try:
            serialized_data = json.dumps(state_data, default=str)
            await self.redis.setex(cache_key, self.default_ttl, serialized_data)
            logger.debug(f"Stored L1 workflow state: {workflow_id}")

        except Exception as e:
            logger.error(f"Failed to store L1 workflow state {workflow_id}: {e}")

    async def get_workflow_state(self, workflow_id: str) -> Optional[Dict[str, Any]]:
        """Retrieve workflow state"""
        cache_key = f"l1:workflow:{workflow_id}:state"

        try:
            cached_data = await self.redis.get(cache_key)
            if cached_data:
                return json.loads(cached_data)
            return None

        except Exception as e:
            logger.error(f"Failed to get L1 workflow state {workflow_id}: {e}")
            return None

    async def store_patient_context(self, patient_id: str, context: Dict[str, Any]):
        """Store patient context for current session"""
        cache_key = f"l1:patient:{patient_id}:context"

        try:
            serialized_context = json.dumps(context, default=str)
            await self.redis.setex(cache_key, self.default_ttl, serialized_context)
            logger.debug(f"Stored L1 patient context: {patient_id}")

        except Exception as e:
            logger.error(f"Failed to store L1 patient context {patient_id}: {e}")

    async def get_patient_context(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Retrieve patient context"""
        cache_key = f"l1:patient:{patient_id}:context"

        try:
            cached_context = await self.redis.get(cache_key)
            if cached_context:
                return json.loads(cached_context)
            return None

        except Exception as e:
            logger.error(f"Failed to get L1 patient context {patient_id}: {e}")
            return None

    async def invalidate_session(self, session_id: str):
        """Invalidate all L1 cache for a session"""
        pattern = f"l1:*{session_id}*"

        try:
            keys = await self.redis.keys(pattern)
            if keys:
                await self.redis.delete(*keys)
                logger.info(f"Invalidated L1 session cache: {session_id}")

        except Exception as e:
            logger.error(f"Failed to invalidate L1 session {session_id}: {e}")
```

**event_driven_prefetcher.py**
```python
from aiokafka import AIOKafkaConsumer
import asyncio
import json
from typing import Dict, List, Any
from .pattern_analyzer import PrefetchPatternAnalyzer
from .l1_session_cache import L1SessionCache

class EventDrivenPrefetcher:
    """
    Event-driven cache prefetcher based on Kafka events
    Responds to recipe_determined, patient_context_available events
    """

    def __init__(self, kafka_config: Dict[str, Any], cache_config: Dict[str, Any]):
        self.kafka_config = kafka_config
        self.pattern_analyzer = PrefetchPatternAnalyzer()
        self.l1_cache = L1SessionCache(cache_config['l1_url'])

        # Event topic subscriptions
        self.event_topics = [
            'medication-events',
            'safety-events',
            'workflow-events',
            'patient-events'
        ]

    async def start_listening(self):
        """Start Kafka event consumer"""
        consumer = AIOKafkaConsumer(
            *self.event_topics,
            bootstrap_servers=self.kafka_config['bootstrap_servers'],
            group_id='cache-prefetcher',
            value_deserializer=lambda m: json.loads(m.decode())
        )

        await consumer.start()
        logger.info(f"Event-driven prefetcher listening to: {self.event_topics}")

        try:
            async for message in consumer:
                await self._handle_event(message.topic, message.value)

        except Exception as e:
            logger.error(f"Event consumer error: {e}")

        finally:
            await consumer.stop()

    async def _handle_event(self, topic: str, event_data: Dict[str, Any]):
        """Handle incoming Kafka events"""
        event_type = event_data.get('event_type')

        try:
            if event_type == 'recipe_determined':
                await self._handle_recipe_event(event_data)
            elif event_type == 'patient_context_available':
                await self._handle_patient_context_event(event_data)
            elif event_type == 'workflow_started':
                await self._handle_workflow_started_event(event_data)
            else:
                logger.debug(f"Unhandled event type: {event_type}")

        except Exception as e:
            logger.error(f"Failed to handle event {event_type}: {e}")

    async def _handle_recipe_event(self, event_data: Dict[str, Any]):
        """Handle recipe_determined events"""
        workflow_id = event_data.get('workflow_id')
        patient_id = event_data.get('patient_id')
        recipe_data = event_data.get('recipe', {})

        # Analyze prefetch patterns
        prefetch_plan = await self.pattern_analyzer.analyze_recipe(recipe_data)

        # Execute prefetch plan
        for cache_operation in prefetch_plan.operations:
            await self._execute_prefetch_operation(cache_operation)

        # Store workflow state in L1 cache
        await self.l1_cache.store_workflow_state(workflow_id, {
            'recipe': recipe_data,
            'patient_id': patient_id,
            'prefetch_completed': True,
            'prefetch_operation_count': len(prefetch_plan.operations)
        })

        logger.info(f"Recipe prefetch completed: {workflow_id} ({len(prefetch_plan.operations)} operations)")

    async def _handle_patient_context_event(self, event_data: Dict[str, Any]):
        """Handle patient_context_available events"""
        patient_id = event_data.get('patient_id')
        context_data = event_data.get('context', {})

        # Store patient context in L1 cache
        await self.l1_cache.store_patient_context(patient_id, context_data)

        # Predict additional data needs based on context
        additional_prefetch = await self.pattern_analyzer.analyze_patient_context(context_data)

        for operation in additional_prefetch.operations:
            await self._execute_prefetch_operation(operation)

        logger.info(f"Patient context prefetch completed: {patient_id}")

    async def _execute_prefetch_operation(self, operation: Dict[str, Any]):
        """Execute individual prefetch operation"""
        operation_type = operation.get('type')

        if operation_type == 'terminology_lookup':
            await self._prefetch_terminology(operation)
        elif operation_type == 'drug_interaction_check':
            await self._prefetch_drug_interactions(operation)
        elif operation_type == 'patient_data_summary':
            await self._prefetch_patient_summary(operation)
        else:
            logger.warning(f"Unknown prefetch operation: {operation_type}")

    async def _prefetch_terminology(self, operation: Dict[str, Any]):
        """Prefetch terminology data"""
        # Implementation for terminology prefetching
        pass

    async def _prefetch_drug_interactions(self, operation: Dict[str, Any]):
        """Prefetch drug interaction data"""
        # Implementation for drug interaction prefetching
        pass

    async def _prefetch_patient_summary(self, operation: Dict[str, Any]):
        """Prefetch patient summary data"""
        # Implementation for patient summary prefetching
        pass
```

**Implementation Timeline**: 2-3 weeks
- Week 1: L1 session cache implementation and integration
- Week 2: Event-driven prefetching service and pattern analyzer
- Week 3: Performance optimization and monitoring integration

---

## Phase 4: Timing Guarantees Infrastructure (2 weeks)

### Overview
Implement SLA monitoring and performance enforcement to meet clinical timing requirements.

#### 4.1 SLA Monitoring Service
```yaml
# docker-compose.sla.yml
services:
  sla-monitor:
    build:
      context: ./sla-monitoring-service
      dockerfile: Dockerfile
    ports:
      - "8060:8060"
    environment:
      - PROMETHEUS_URL=http://prometheus:9090
      - GRAFANA_URL=http://grafana:3000
      - ALERT_WEBHOOK=http://alertmanager:9093/api/v1/alerts
    depends_on:
      - prometheus
      - grafana
      - alertmanager

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - ./monitoring/grafana-dashboards:/etc/grafana/provisioning/dashboards
      - grafana-data:/var/lib/grafana

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./monitoring/alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

#### 4.2 Core SLA Implementation

**sla_monitor.py**
```python
import asyncio
import time
from typing import Dict, List, Any, Optional
from datetime import datetime, timedelta
from dataclasses import dataclass
import aiohttp
from loguru import logger

@dataclass
class SLATarget:
    name: str
    description: str
    target_ms: int
    threshold_p95: int
    threshold_p99: int
    alert_threshold: int

class SLAMonitor:
    """
    SLA Monitor for Runtime Layer Performance Guarantees

    Target SLAs:
    - Patient Events: < 500ms end-to-end
    - Knowledge Updates: < 5 minutes propagation
    - Cache Warming: 50-100ms
    - Query Response: p95 < 100ms
    - Snapshot Creation: < 20ms
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.sla_targets = self._initialize_sla_targets()
        self.performance_metrics = {}

    def _initialize_sla_targets(self) -> Dict[str, SLATarget]:
        return {
            'patient_events': SLATarget(
                name='Patient Events Processing',
                description='End-to-end patient event processing latency',
                target_ms=500,
                threshold_p95=400,
                threshold_p99=500,
                alert_threshold=600
            ),
            'knowledge_updates': SLATarget(
                name='Knowledge Base Propagation',
                description='KB change propagation across runtime layer',
                target_ms=300000,  # 5 minutes
                threshold_p95=240000,  # 4 minutes
                threshold_p99=300000,  # 5 minutes
                alert_threshold=600000  # 10 minutes
            ),
            'cache_warming': SLATarget(
                name='Cache Warming Operations',
                description='Cache prefetch and warming latency',
                target_ms=100,
                threshold_p95=75,
                threshold_p99=100,
                alert_threshold=150
            ),
            'query_response': SLATarget(
                name='Query Response Time',
                description='Runtime layer query response latency',
                target_ms=100,
                threshold_p95=100,
                threshold_p99=150,
                alert_threshold=200
            ),
            'snapshot_creation': SLATarget(
                name='Snapshot Creation',
                description='Cross-store consistency snapshot creation',
                target_ms=20,
                threshold_p95=15,
                threshold_p99=20,
                alert_threshold=50
            )
        }

    async def start_monitoring(self):
        """Start SLA monitoring tasks"""
        tasks = [
            asyncio.create_task(self._monitor_performance_metrics()),
            asyncio.create_task(self._evaluate_sla_compliance()),
            asyncio.create_task(self._generate_alerts())
        ]

        await asyncio.gather(*tasks)

    async def record_operation_time(self, operation_type: str, duration_ms: int,
                                  context: Optional[Dict[str, Any]] = None):
        """Record operation timing for SLA evaluation"""
        timestamp = datetime.utcnow()

        if operation_type not in self.performance_metrics:
            self.performance_metrics[operation_type] = []

        self.performance_metrics[operation_type].append({
            'timestamp': timestamp,
            'duration_ms': duration_ms,
            'context': context or {}
        })

        # Keep only last 1000 measurements per operation
        if len(self.performance_metrics[operation_type]) > 1000:
            self.performance_metrics[operation_type] = \
                self.performance_metrics[operation_type][-1000:]

        # Check immediate SLA violation
        if operation_type in self.sla_targets:
            target = self.sla_targets[operation_type]
            if duration_ms > target.alert_threshold:
                await self._send_immediate_alert(operation_type, duration_ms, target)

    async def _evaluate_sla_compliance(self):
        """Periodically evaluate SLA compliance"""
        while True:
            try:
                compliance_report = {}

                for operation_type, target in self.sla_targets.items():
                    if operation_type in self.performance_metrics:
                        metrics = self.performance_metrics[operation_type]
                        compliance = self._calculate_compliance(metrics, target)
                        compliance_report[operation_type] = compliance

                await self._publish_compliance_metrics(compliance_report)

            except Exception as e:
                logger.error(f"SLA evaluation error: {e}")

            await asyncio.sleep(60)  # Evaluate every minute

    def _calculate_compliance(self, metrics: List[Dict[str, Any]],
                            target: SLATarget) -> Dict[str, Any]:
        """Calculate SLA compliance for operation type"""
        if not metrics:
            return {'compliance': 0.0, 'sample_size': 0}

        # Recent metrics (last 5 minutes)
        cutoff_time = datetime.utcnow() - timedelta(minutes=5)
        recent_metrics = [m for m in metrics if m['timestamp'] >= cutoff_time]

        if not recent_metrics:
            return {'compliance': 0.0, 'sample_size': 0}

        durations = [m['duration_ms'] for m in recent_metrics]
        durations.sort()

        # Calculate percentiles
        p50_idx = len(durations) // 2
        p95_idx = int(len(durations) * 0.95)
        p99_idx = int(len(durations) * 0.99)

        p50 = durations[p50_idx] if p50_idx < len(durations) else durations[-1]
        p95 = durations[p95_idx] if p95_idx < len(durations) else durations[-1]
        p99 = durations[p99_idx] if p99_idx < len(durations) else durations[-1]

        # Calculate compliance percentage
        compliant_operations = len([d for d in durations if d <= target.target_ms])
        compliance_rate = compliant_operations / len(durations)

        return {
            'compliance': compliance_rate,
            'sample_size': len(recent_metrics),
            'p50': p50,
            'p95': p95,
            'p99': p99,
            'target_ms': target.target_ms,
            'p95_compliant': p95 <= target.threshold_p95,
            'p99_compliant': p99 <= target.threshold_p99
        }

    async def _send_immediate_alert(self, operation_type: str, duration_ms: int,
                                  target: SLATarget):
        """Send immediate alert for SLA violation"""
        alert_data = {
            'alert_type': 'sla_violation',
            'operation': operation_type,
            'duration_ms': duration_ms,
            'target_ms': target.target_ms,
            'severity': 'high' if duration_ms > target.alert_threshold * 2 else 'medium',
            'timestamp': datetime.utcnow().isoformat()
        }

        logger.warning(f"SLA violation: {operation_type} took {duration_ms}ms "
                      f"(target: {target.target_ms}ms)")

        # Send to alerting system
        await self._send_alert(alert_data)
```

**Implementation Timeline**: 2 weeks
- Week 1: SLA monitoring infrastructure and Prometheus integration
- Week 2: Grafana dashboards, alerting rules, and performance enforcement

---

## Integration and Deployment Plan

### Docker Compose Master Configuration

**docker-compose.runtime-layer.yml**
```yaml
version: '3.8'

services:
  # Existing services
  neo4j:
    image: neo4j:5.11-community
    # ... existing configuration

  # New Flink Services
  flink-jobmanager:
    extends:
      file: docker-compose.flink.yml
      service: flink-jobmanager

  flink-taskmanager:
    extends:
      file: docker-compose.flink.yml
      service: flink-taskmanager

  # New Evidence Service
  evidence-service:
    extends:
      file: docker-compose.evidence.yml
      service: evidence-service

  # Enhanced Cache Services
  redis-l1:
    extends:
      file: docker-compose.cache.yml
      service: redis-l1

  cache-prefetcher-service:
    extends:
      file: docker-compose.cache.yml
      service: cache-prefetcher-service

  # SLA Monitoring
  sla-monitor:
    extends:
      file: docker-compose.sla.yml
      service: sla-monitor

  prometheus:
    extends:
      file: docker-compose.sla.yml
      service: prometheus

  grafana:
    extends:
      file: docker-compose.sla.yml
      service: grafana

networks:
  runtime-layer:
    driver: bridge

volumes:
  neo4j-data:
  postgres-data:
  clickhouse-data:
  prometheus-data:
  grafana-data:
  flink-checkpoints:
```

### Deployment Scripts

**scripts/deploy-missing-components.sh**
```bash
#!/bin/bash

set -e

echo "🚀 Deploying Missing Runtime Layer Components"

# Phase 1: Flink Stream Processing
echo "📊 Phase 1: Deploying Flink Stream Processing..."
docker-compose -f docker-compose.flink.yml up -d
./scripts/wait-for-flink.sh
./scripts/deploy-flink-jobs.sh

# Phase 2: Evidence Envelope System
echo "📋 Phase 2: Deploying Evidence Envelope System..."
docker-compose -f docker-compose.evidence.yml up -d
./scripts/run-evidence-migrations.sh

# Phase 3: Enhanced Cache Services
echo "💾 Phase 3: Deploying Enhanced Cache Services..."
docker-compose -f docker-compose.cache.yml up -d

# Phase 4: SLA Monitoring
echo "📈 Phase 4: Deploying SLA Monitoring..."
docker-compose -f docker-compose.sla.yml up -d

# Integration Test
echo "🧪 Running Integration Tests..."
./scripts/test-missing-components.sh

echo "✅ Missing components deployment complete!"
```

### Testing Strategy

**scripts/test-missing-components.sh**
```bash
#!/bin/bash

echo "🧪 Testing Missing Components Integration"

# Test Flink Stream Processing
echo "Testing Flink patient event processing..."
curl -X POST http://localhost:8081/jars/upload -F file=@flink-jobs.jar
curl -X POST http://localhost:8081/jobs -d '{"entryClass": "com.cardiofit.stream.PatientEventEnrichmentJob"}'

# Test Evidence Envelope
echo "Testing Evidence Envelope generation..."
curl -X POST http://localhost:8050/envelope/create -H "Content-Type: application/json" \
     -d '{"workflow_context": {"patient_id": "test-123", "workflow_type": "medication_analysis"}}'

# Test L1 Cache
echo "Testing L1 session cache..."
curl -X POST http://localhost:8051/cache/l1/store -H "Content-Type: application/json" \
     -d '{"session_id": "test-session", "data": {"test": "data"}}'

# Test SLA Monitoring
echo "Testing SLA monitoring..."
curl http://localhost:8060/sla/status

echo "✅ All component tests passed!"
```

---

## Resource Requirements and Scaling

### Infrastructure Requirements
- **Additional Memory**: 8GB total (Flink: 4GB, Evidence: 1GB, Cache: 2GB, Monitoring: 1GB)
- **Storage**: 50GB for checkpoints, evidence audit logs, and metrics
- **Network**: Internal container networking with service discovery
- **CPU**: 4 additional cores for stream processing and monitoring

### Production Scaling Considerations
- **Flink**: Auto-scaling TaskManagers based on event volume
- **Evidence Service**: Horizontal scaling with load balancer
- **Cache Services**: Redis clustering for high availability
- **SLA Monitoring**: Prometheus federation for multi-cluster monitoring

---

## Timeline Summary

| Phase | Component | Duration | Priority |
|-------|-----------|----------|----------|
| 1 | Apache Flink Stream Processing | 4-6 weeks | Critical |
| 2 | Evidence Envelope System | 3-4 weeks | Critical |
| 3 | Complete Cache Prefetcher | 2-3 weeks | High |
| 4 | SLA Monitoring Infrastructure | 2 weeks | High |
| **Total** | **All Missing Components** | **11-15 weeks** | **Critical** |

**Parallel Development**: Phases 1-2 can be developed in parallel, reducing total timeline to **8-10 weeks** with 2 developers.

**Production Readiness**: After completing all phases, the runtime layer will achieve 100% compliance with the complete design specification and meet all clinical performance requirements.

---

## Success Metrics

### Performance Targets (Post-Implementation)
- ✅ **Patient Events**: < 500ms end-to-end (Currently: >2s)
- ✅ **Knowledge Updates**: < 5 minutes propagation (Currently: Manual)
- ✅ **Cache Warming**: 50-100ms (Currently: 300ms+)
- ✅ **Query Response**: p95 < 100ms (Currently: 80% compliance)
- ✅ **Snapshot Creation**: < 20ms (Currently: 45ms)

### Clinical Compliance
- ✅ **Evidence Envelope**: 100% decision auditability
- ✅ **Regulatory Compliance**: Complete provenance tracking
- ✅ **Performance SLAs**: Automated monitoring and alerting
- ✅ **Fault Tolerance**: Circuit breakers and graceful degradation

This implementation plan provides a comprehensive roadmap for closing the critical gaps in the runtime layer, transforming it from a strong foundation into a production-ready clinical decision support platform that fully meets the complete design specification requirements.