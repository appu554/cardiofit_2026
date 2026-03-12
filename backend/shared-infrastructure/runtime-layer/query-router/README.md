# CardioFit Multi-KB Query Router

Production-ready intelligent routing system for all CardioFit Knowledge Bases with comprehensive performance optimization and fault tolerance.

## 🏗️ Architecture Overview

The Multi-KB Query Router serves as the intelligent traffic controller for all data access across CardioFit's 8+ Knowledge Bases, implementing a sophisticated polyglot persistence pattern with automatic data source selection.

```
┌─────────────────────────────────────────────────────────────┐
│                    Service Layer                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ Medication  │ │  Clinical   │ │  Patient Service    │   │
│  │  Service    │ │ Reasoning   │ │                     │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────┬───────────────────────────────┬─────────┘
                  │                               │
┌─────────────────▼───────────────────────────────▼─────────┐
│                Multi-KB Query Router                      │
│  ┌───────────────┐ ┌──────────────┐ ┌─────────────────┐  │
│  │ Cache         │ │ Performance  │ │ Fallback        │  │
│  │ Coordinator   │ │ Monitor      │ │ Handler         │  │
│  └───────────────┘ └──────────────┘ └─────────────────┘  │
└─────┬──────┬──────┬──────┬──────┬──────┬──────┬──────────┘
      │      │      │      │      │      │      │
   ┌──▼─┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼─┐ ┌──▼──┐
   │Neo4j│ │ CH │ │ PG │ │ ES │ │GDB │ │L2 │ │ L3  │
   │ KB1-│ │KB1-│ │    │ │    │ │    │ │   │ │Cache│
   │ KB7 │ │KB7 │ │    │ │    │ │    │ │   │ │     │
   └─────┘ └────┘ └────┘ └────┘ └────┘ └───┘ └─────┘
```

## 📊 Key Features

### 🎯 **Intelligent Routing**
- **Pattern-Based Selection**: Automatically routes queries to optimal data sources
- **Polyglot Persistence**: PostgreSQL (exact), Elasticsearch (search), Neo4j (graph), ClickHouse (analytics), GraphDB (semantic)
- **Cross-KB Orchestration**: Parallel execution across multiple Knowledge Bases

### ⚡ **Performance Optimization**
- **L2 Cache (Redis 6379)**: Proactively warmed by Cache Prefetcher from Kafka events
- **L3 Cache (Redis 6380)**: Router-managed complex query result caching
- **Circuit Breakers**: Automatic failure detection with intelligent fallback
- **Sub-millisecond Response**: For cache-hit scenarios

### 🛡️ **Comprehensive Resilience**
- **Multi-Level Fallback Chains**: Alternative data sources, degraded responses, cached results
- **Real-time Health Monitoring**: Performance metrics, error rates, latency tracking
- **Graceful Degradation**: Partial responses when some data sources fail

### 🔍 **GraphDB Semantic Integration**
- **Ontological Reasoning**: SPARQL queries for subsumption checks and concept relationships
- **Medical Terminology Translation**: Cross-system code translations using semantic inference
- **Advanced Clinical Logic**: Complex reasoning across medical ontologies

## 🚀 Quick Start

### Installation

```python
from backend.shared_infrastructure.runtime_layer.query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    QueryPattern
)
```

### Basic Configuration

```python
config = {
    'neo4j': {
        'uri': 'bolt://localhost:7687',
        'auth': ('neo4j', 'password')
    },
    'clickhouse': {
        'databases': {
            'kb1': {'database': 'kb1_patient_analytics'},
            'kb3': {'database': 'kb3_drug_calculations'},
            'kb7': {'database': 'kb7_terminology_analytics'}
        }
    },
    'postgres': {
        'host': 'localhost',
        'port': 5432,
        'database': 'cardiofit'
    },
    'graphdb': {
        'endpoint': 'http://localhost:7200',
        'repository': 'cardiofit-semantic'
    },
    'redis': {
        'l2': {'host': 'localhost', 'port': 6379},
        'l3': {'host': 'localhost', 'port': 6380}
    }
}

router = MultiKBQueryRouter(config)
await router.initialize_clients()
```

### Simple Query Example

```python
# Terminology lookup (routes to PostgreSQL)
request = MultiKBQueryRequest(
    service_id="medication-service",
    pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
    params={"code": "I10", "system": "ICD10"},
    kb_id="kb7"
)

response = await router.route_query(request)
print(f"Result: {response.data}")
print(f"Sources: {response.sources_used}")
print(f"Latency: {response.latency_ms}ms")
```

### Semantic Inference Example

```python
# Semantic reasoning (routes to GraphDB)
request = MultiKBQueryRequest(
    service_id="terminology-service",
    pattern=QueryPattern.KB7_SEMANTIC_INFERENCE,
    params={
        "concept_id": "123456",
        "target_system": "SNOMED-CT"
    },
    kb_id="kb7"
)

response = await router.route_query(request)
# Returns: ontological relationships, subsumptions, translations
```

### Cross-KB Query Example

```python
# Drug safety analysis across multiple KBs
request = MultiKBQueryRequest(
    service_id="clinical-reasoning",
    pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
    params={
        "drug_codes": ["rxnorm123", "rxnorm456"],
        "patient_id": "12345"
    },
    cross_kb_scope=["kb3", "kb5", "kb7"]
)

response = await router.route_query(request)
# Returns: interactions (KB5), calculations (KB3), terminology (KB7)
```

## 📋 Query Patterns

### Single KB Patterns

| Pattern | Data Source | Use Case |
|---------|-------------|----------|
| `KB1_PATIENT_LOOKUP` | Neo4j KB1 | Patient demographics and relationships |
| `KB2_GUIDELINE_SEARCH` | Elasticsearch | Clinical guideline text search |
| `KB3_DRUG_CALCULATION` | ClickHouse KB3 | High-performance dosing calculations |
| `KB5_INTERACTION_CHECK` | Neo4j KB5 | Drug interaction network analysis |
| `KB7_TERMINOLOGY_LOOKUP` | PostgreSQL | Exact medical code lookups |
| `KB7_TERMINOLOGY_SEARCH` | Elasticsearch | Fuzzy terminology search |
| **`KB7_SEMANTIC_INFERENCE`** | **GraphDB** | **Ontological reasoning and translations** |

### Cross-KB Patterns

| Pattern | Data Sources | Use Case |
|---------|-------------|----------|
| `CROSS_KB_PATIENT_VIEW` | Neo4j (patient_data + semantic_mesh streams) | Complete patient clinical profile |
| `CROSS_KB_DRUG_ANALYSIS` | Neo4j KB5 + ClickHouse KB3/KB6 + KB7 | Comprehensive medication safety |
| `CROSS_KB_SEMANTIC_SEARCH` | GraphDB + Neo4j Shared + Elasticsearch | Complex semantic queries |

## 🔧 Advanced Configuration

### Performance Tuning

```python
config = {
    # ... base config ...
    'monitoring': {
        'enabled': True,
        'slow_query_threshold_ms': 1000,
        'error_rate_threshold': 0.05
    },
    'caching': {
        'enabled': True,
        'l2_ttl': 300,  # 5 minutes
        'l3_ttl': 3600  # 1 hour
    },
    'fallback': {
        'enabled': True,
        'max_retries': 3,
        'circuit_breaker_threshold': 5
    }
}
```

### Health Monitoring

```python
# Get comprehensive health status
health = await router.get_health_status()
print(f"Router Status: {health['router_status']}")
print(f"Client Health: {health['client_health']}")

# Get performance metrics
metrics = await router.get_performance_metrics()
print(f"Avg Latency: {metrics['average_latency_ms']}ms")
print(f"Cache Hit Rate: {metrics['cache_hit_rate']:.2%}")
print(f"Error Rate: {metrics['error_rate']:.2%}")
```

## 🧪 Testing

Run the comprehensive test suite:

```bash
pytest test_multi_kb_router.py -v
```

Key test areas:
- ✅ Single KB routing patterns
- ✅ Cross-KB query orchestration
- ✅ Cache coordination (L2/L3)
- ✅ Fallback chain execution
- ✅ Circuit breaker functionality
- ✅ Performance monitoring
- ✅ GraphDB semantic inference
- ✅ End-to-end workflows

## 📊 Performance Characteristics

### Latency Targets
- **Cache Hit**: < 5ms
- **Single KB Query**: < 100ms
- **Cross-KB Query**: < 500ms
- **Semantic Inference**: < 1000ms

### Throughput Capacity
- **Sustained**: 1000+ queries/second
- **Peak**: 5000+ queries/second (with cache)

### Availability Targets
- **Uptime**: 99.9%
- **Fallback Success Rate**: > 95%
- **Cache Hit Rate**: > 70%

## 🔍 Architecture Decisions

### Why Polyglot Persistence?
Each data model optimizes for specific query patterns:
- **PostgreSQL**: O(log n) exact lookups with B-tree indices
- **Neo4j**: O(1) graph traversals for relationships
- **ClickHouse**: Columnar analytics with 100x faster aggregations
- **Elasticsearch**: Full-text search with relevance ranking
- **GraphDB**: Semantic reasoning with SPARQL inferencing

### Why Dual Cache Layers?
- **L2 (Proactive)**: Event-driven warming reduces initial query latency by ~95%
- **L3 (Reactive)**: Complex result caching prevents expensive recomputation

### Why Circuit Breakers?
Prevents cascade failures and enables:
- **Fast Failure**: 5ms timeout vs 30s connection wait
- **Automatic Recovery**: Half-open state testing
- **System Protection**: Resource exhaustion prevention

## 🚨 Monitoring & Alerts

### Key Metrics
- **Latency Percentiles**: P50, P95, P99 tracking
- **Error Rates**: By pattern, KB, and data source
- **Cache Effectiveness**: Hit rates and eviction patterns
- **Circuit Breaker States**: Open/closed/half-open tracking

### Alert Conditions
- High latency (>2x threshold)
- Error rate >5%
- Cache hit rate <70%
- Circuit breaker open state

## 🔧 Operational Commands

### Circuit Breaker Management
```python
# Check circuit breaker status
health = await router.get_health_status()
breakers = health['circuit_breakers']

# Reset specific circuit breaker
await router.fallback_handler.reset_circuit_breaker(DataSource.POSTGRES)
```

### Cache Management
```python
# Get cache statistics
stats = await router.cache_coordinator.get_stats()

# Invalidate specific cache patterns
await router.cache_coordinator.invalidate_cache(pattern="kb7_terminology_lookup")
```

### Performance Analysis
```python
# Get slow queries
slow_queries = await router.performance_monitor.get_slow_queries(limit=10)

# Get error analysis
errors = await router.performance_monitor.get_error_analysis()
```

## 🔗 Integration Points

### With Cache Prefetcher
- Listens to Kafka `recipe_determined` events
- Proactively warms L2 cache with predicted data needs
- Coordinates via `cache_coordinator.coordinate_with_prefetcher()`

### With Neo4j Dual-Stream Manager
- Manages `patient_data` and `semantic_mesh` streams
- Provides partition-aware query routing
- Enables cross-stream relationship queries

### With Clinical Services
- **Medication Service**: Drug safety analysis and calculations
- **Clinical Reasoning**: Complex multi-KB clinical logic
- **Patient Service**: Comprehensive patient data assembly

## 📖 Documentation

- [Complete API Documentation](./QUERY_ROUTER_DOCUMENTATION.md)
- [Integration Examples](./INTEGRATION_EXAMPLES.md)
- [Performance Tuning Guide](./QUERY_ROUTER_DOCUMENTATION.md#performance-optimization)

## 🤝 Contributing

When extending the router:
1. Add new query patterns to `QueryPattern` enum
2. Update routing rules in `_initialize_kb_routing_rules()`
3. Implement data source query methods
4. Add comprehensive tests
5. Update fallback chains if needed

## 📝 License

Part of the CardioFit Clinical Synthesis Hub platform.