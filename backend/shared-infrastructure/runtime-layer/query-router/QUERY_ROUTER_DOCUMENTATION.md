# CardioFit Multi-KB Query Router Documentation

## Overview

The Multi-KB Query Router is the intelligent routing layer that directs queries across all CardioFit Knowledge Bases to their optimal data sources. It provides automatic data source selection, cross-KB query orchestration, and performance optimization through intelligent caching and fallback strategies.

## Table of Contents

1. [Architecture](#architecture)
2. [API Reference](#api-reference)
3. [Query Patterns](#query-patterns)
4. [Integration Guide](#integration-guide)
5. [Configuration](#configuration)
6. [Performance Optimization](#performance-optimization)
7. [Examples](#examples)

## Architecture

### Component Overview

```
   Client Request
         ↓
   MultiKBQueryRouter
         ↓
   Pattern Analysis
         ↓
   Route Selection
    ╱    |    ╲
Neo4j  ClickHouse  PostgreSQL
  ↓       ↓          ↓
Response Aggregation
         ↓
   Client Response
```

### Core Classes

#### `MultiKBQueryRouter`
Main routing engine that manages query distribution across all data sources.

**Responsibilities:**
- Query pattern classification
- Data source selection
- Cross-KB query orchestration
- Performance metrics tracking
- Fallback handling

#### `MultiKBQueryRequest`
Encapsulates incoming query requests with metadata.

**Properties:**
- `service_id`: Originating service identifier
- `kb_id`: Target Knowledge Base (None for cross-KB queries)
- `pattern`: Query pattern enumeration
- `params`: Query parameters dictionary
- `require_snapshot`: Consistency snapshot requirement
- `cross_kb_scope`: List of KBs for cross-KB queries
- `priority`: Query priority level (normal/high/low)

#### `MultiKBQueryResponse`
Structured response with query execution metadata.

**Properties:**
- `data`: Query result data
- `sources_used`: List of data sources queried
- `kb_sources`: List of Knowledge Bases accessed
- `latency`: Query execution time (ms)
- `snapshot_id`: Consistency snapshot identifier
- `cache_status`: Cache hit/miss/fallback status

## API Reference

### Core Methods

#### `initialize_clients()`
```python
async def initialize_clients(self) -> None
```
Lazy initialization of all data store clients (Neo4j, ClickHouse, GraphDB).

**Usage:**
```python
router = MultiKBQueryRouter(config)
await router.initialize_clients()
```

#### `route_query()`
```python
async def route_query(self, request: MultiKBQueryRequest) -> MultiKBQueryResponse
```
Main query routing method that determines optimal data sources and executes queries.

**Parameters:**
- `request`: MultiKBQueryRequest object containing query details

**Returns:**
- `MultiKBQueryResponse` with results and metadata

**Example:**
```python
request = MultiKBQueryRequest(
    service_id="clinical-reasoning",
    kb_id="kb7",
    pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
    params={"code": "I10", "system": "ICD10"}
)
response = await router.route_query(request)
```

#### `get_performance_metrics()`
```python
async def get_performance_metrics(self) -> Dict[str, Any]
```
Returns current performance metrics and statistics.

**Returns:**
```python
{
    'total_queries': 45231,
    'kb_query_counts': {
        'kb1': 12500,
        'kb7': 18700,
        'cross_kb': 8900
    },
    'average_latency': 127.5,
    'cache_hit_rate': 0.73,
    'error_rate': 0.002
}
```

## Query Patterns

### Single KB Patterns

| Pattern | Description | Primary Data Source | Use Case |
|---------|-------------|-------------------|----------|
| `KB1_PATIENT_LOOKUP` | Patient data retrieval | Neo4j KB1 | Get patient demographics, conditions |
| `KB2_GUIDELINE_SEARCH` | Clinical guideline search | Elasticsearch | Find relevant clinical guidelines |
| `KB3_DRUG_CALCULATION` | Drug dosing calculations | ClickHouse KB3 | Compute medication dosing |
| `KB5_INTERACTION_CHECK` | Drug interaction checking | Neo4j KB5 | Identify drug-drug interactions |
| `KB7_TERMINOLOGY_LOOKUP` | Medical term lookup | PostgreSQL | Exact terminology matching |
| `KB7_TERMINOLOGY_SEARCH` | Medical term search | Elasticsearch | Fuzzy terminology search |
| **`KB7_SEMANTIC_INFERENCE`** | **Ontology reasoning** | **GraphDB** | **Subsumption checks, class membership, translations** |

### Cross-KB Patterns

| Pattern | Description | Data Sources | Use Case |
|---------|-------------|-------------|----------|
| `CROSS_KB_PATIENT_VIEW` | Complete patient profile | **Neo4j** (`patient_data` stream + `semantic_mesh` stream) | Comprehensive patient overview |
| `CROSS_KB_DRUG_ANALYSIS` | Full drug safety analysis | Neo4j KB5 + ClickHouse KB3/KB6 + KB7 | Medication safety assessment |
| `CROSS_KB_SEMANTIC_SEARCH` | Semantic search across KBs | GraphDB + Neo4j Shared + Elasticsearch | Complex clinical queries |

### Analytics Patterns

| Pattern | Description | Data Source | Use Case |
|---------|-------------|-------------|----------|
| `PATIENT_ANALYTICS` | Patient population analytics | ClickHouse KB1 | Population health metrics |
| `DRUG_ANALYTICS` | Medication usage analytics | ClickHouse KB3 | Drug utilization analysis |
| `TERMINOLOGY_ANALYTICS` | Term usage patterns | ClickHouse KB7 | Terminology usage statistics |

## Integration Guide

### Basic Setup

1. **Import Required Classes:**
```python
from shared_infrastructure.runtime_layer.query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    QueryPattern
)
```

2. **Initialize Router:**
```python
config = {
    'neo4j': {
        'uri': 'bolt://localhost:7687',
        'auth': ('neo4j', 'password')
    },
    'clickhouse_databases': {
        'kb1': 'kb1_patient_analytics',
        'kb3': 'kb3_drug_calculations',
        'kb7': 'kb7_terminology_analytics'
    },
    'postgres': {
        'host': 'localhost',
        'port': 5432,
        'database': 'cardiofit'
    }
}

router = MultiKBQueryRouter(config)
await router.initialize_clients()
```

3. **Execute Queries:**
```python
# Single KB query
request = MultiKBQueryRequest(
    service_id="medication-service",
    kb_id="kb7",
    pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
    params={"code": "428.0", "system": "ICD9"}
)
response = await router.route_query(request)

# Cross-KB query
request = MultiKBQueryRequest(
    service_id="clinical-reasoning",
    kb_id=None,
    pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
    params={"patient_id": "12345", "drug_rxnorm": "197381"},
    cross_kb_scope=["kb3", "kb5", "kb7"]
)
response = await router.route_query(request)
```

### Advanced Features

#### Snapshot Consistency
```python
# Ensure consistent reads across multiple queries
request = MultiKBQueryRequest(
    service_id="critical-analysis",
    kb_id="kb1",
    pattern=QueryPattern.KB1_PATIENT_LOOKUP,
    params={"patient_id": "12345"},
    require_snapshot=True  # Ensures consistency
)
```

#### Priority Routing
```python
# High priority query bypasses cache
request = MultiKBQueryRequest(
    service_id="emergency-service",
    kb_id="kb5",
    pattern=QueryPattern.KB5_INTERACTION_CHECK,
    params={"drug_codes": ["123", "456"]},
    priority="high"  # Prioritized execution
)
```

## Configuration

### Full Configuration Schema

```python
RUNTIME_CONFIG = {
    # Data source connections
    'neo4j': {
        'uri': 'bolt://localhost:7687',
        'auth': ('neo4j', 'password'),
        'max_connection_lifetime': 3600,
        'max_connection_pool_size': 50
    },

    # ClickHouse analytics databases
    'clickhouse_databases': {
        'kb1': {
            'database': 'kb1_patient_analytics',
            'host': 'localhost',
            'port': 8123
        },
        'kb3': {
            'database': 'kb3_drug_calculations',
            'host': 'localhost',
            'port': 8123
        },
        'kb6': {
            'database': 'kb6_evidence_scores',
            'host': 'localhost',
            'port': 8123
        },
        'kb7': {
            'database': 'kb7_terminology_analytics',
            'host': 'localhost',
            'port': 8123
        }
    },

    # PostgreSQL configuration
    'postgres': {
        'host': 'localhost',
        'port': 5432,
        'database': 'cardiofit',
        'pool_size': 20
    },

    # Elasticsearch configuration
    'elasticsearch': {
        'hosts': ['http://localhost:9200'],
        'index_prefix': 'cardiofit_'
    },

    # Redis caching layers
    'redis_l2': {
        'host': 'localhost',
        'port': 6379,
        'db': 0,
        'ttl': 300  # 5 minutes
    },
    'redis_l3': {
        'host': 'localhost',
        'port': 6380,
        'db': 0,
        'ttl': 3600  # 1 hour
    },

    # GraphDB for semantic reasoning
    'graphdb': {
        'endpoint': 'http://localhost:7200',
        'repository': 'cardiofit-semantic'
    }
}
```

### Environment-Specific Configuration

```python
# Development
DEV_CONFIG = {
    **BASE_CONFIG,
    'debug': True,
    'cache_enabled': False,
    'fallback_enabled': True
}

# Production
PROD_CONFIG = {
    **BASE_CONFIG,
    'debug': False,
    'cache_enabled': True,
    'fallback_enabled': True,
    'performance_monitoring': True
}
```

## Performance Optimization

### Proactive Cache Warming

The router's caching works in tandem with the **Cache Prefetcher** service for optimal performance. The Prefetcher listens for upstream Kafka events (e.g., `recipe_determined`) and proactively warms the L2 Redis cache with the specific data a workflow is likely to need.

This means that by the time the Query Router receives a request, the required data is often already waiting in the L2 cache, resulting in sub-millisecond response times for the initial query. The router's L3 cache is then used for the results of more complex, cross-KB queries that it orchestrates itself.

### Caching Strategy

The router implements a multi-level caching strategy:

1. **L2 Cache (Redis 6379)**: Short-lived cache for frequently accessed data (5 min TTL)
   - **Proactively warmed** by Cache Prefetcher based on upstream events
   - Contains specific data items workflows are likely to request

2. **L3 Cache (Redis 6380)**: Longer-lived cache for complex query results (1 hour TTL)
   - Contains **router-orchestrated** cross-KB query results
   - Reduces latency for repeated complex analytical queries

```python
# Cache key structure
cache_key = f"{kb_id}:{pattern}:{hash(params)}"

# Example cache keys
"kb7:terminology_lookup:a3f2d8c9"  # Terminology lookup (L2 - proactively warmed)
"cross_kb:drug_analysis:b7e4f1a2"  # Cross-KB analysis (L3 - router cached)
```

### Query Optimization Techniques

1. **Parallel Execution**: Cross-KB queries execute data source queries in parallel
2. **Early Termination**: Stop execution if critical data source fails
3. **Adaptive Routing**: Route based on current system load and performance
4. **Fallback Chains**: Automatic fallback to alternative data sources

### Performance Monitoring

```python
# Get current performance metrics
metrics = await router.get_performance_metrics()

# Monitor specific KB performance
kb7_latency = metrics['kb_latency']['kb7']
if kb7_latency > 500:  # ms
    logger.warning(f"KB7 latency high: {kb7_latency}ms")
```

## Examples

### Example 1: Patient Medication Review

```python
async def review_patient_medications(patient_id: str):
    """Comprehensive medication review across multiple KBs"""

    router = MultiKBQueryRouter(config)

    # Get patient medications (KB1)
    medications_request = MultiKBQueryRequest(
        service_id="medication-review",
        kb_id="kb1",
        pattern=QueryPattern.KB1_PATIENT_LOOKUP,
        params={"patient_id": patient_id}
    )
    medications = await router.route_query(medications_request)

    # Check interactions (Cross-KB)
    drug_codes = [med['rxnorm'] for med in medications.data['medications']]
    interaction_request = MultiKBQueryRequest(
        service_id="medication-review",
        kb_id=None,
        pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
        params={"drug_codes": drug_codes},
        cross_kb_scope=["kb3", "kb5", "kb7"]
    )
    interactions = await router.route_query(interaction_request)

    return {
        'patient_id': patient_id,
        'medications': medications.data,
        'interactions': interactions.data,
        'sources_queried': interactions.sources_used
    }
```

### Example 2: Terminology Translation

```python
async def translate_terminology(code: str, from_system: str, to_system: str):
    """Translate medical codes between systems using the tiered terminology engine"""

    router = MultiKBQueryRouter(config)

    # 1. Attempt exact lookup in PostgreSQL
    lookup_request = MultiKBQueryRequest(
        service_id="terminology-service",
        kb_id="kb7",
        pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
        params={
            "code": code,
            "system": from_system
        }
    )
    term = await router.route_query(lookup_request)

    if not term.data:
        # 2. Fallback to fuzzy search in Elasticsearch
        search_request = MultiKBQueryRequest(
            service_id="terminology-service",
            kb_id="kb7",
            pattern=QueryPattern.KB7_TERMINOLOGY_SEARCH,
            params={
                "query": code,
                "system": from_system
            }
        )
        term = await router.route_query(search_request)

    # 3. Use the found concept to perform semantic reasoning in GraphDB
    if term.data and term.data.get('concept_id'):
        translation_request = MultiKBQueryRequest(
            service_id="terminology-service",
            kb_id="kb7",
            pattern=QueryPattern.KB7_SEMANTIC_INFERENCE,
            params={
                "concept_id": term.data['concept_id'],
                "target_system": to_system
            }
        )
        translation = await router.route_query(translation_request)
        return translation.data

    return None
```

### Example 3: Population Analytics

```python
async def analyze_drug_utilization(drug_class: str, time_range: dict):
    """Analyze drug utilization patterns across patient population"""

    router = MultiKBQueryRouter(config)

    # Analytics query to ClickHouse
    analytics_request = MultiKBQueryRequest(
        service_id="analytics-service",
        kb_id="kb3",
        pattern=QueryPattern.DRUG_ANALYTICS,
        params={
            "drug_class": drug_class,
            "start_date": time_range['start'],
            "end_date": time_range['end'],
            "aggregation": "daily"
        }
    )

    results = await router.route_query(analytics_request)

    return {
        'drug_class': drug_class,
        'time_range': time_range,
        'utilization_metrics': results.data,
        'query_latency': results.latency,
        'data_sources': results.sources_used
    }
```

## Troubleshooting

### Common Issues

1. **High Latency**
   - Check cache hit rates
   - Verify data source connectivity
   - Review query complexity

2. **Fallback Activation**
   - Monitor primary data source health
   - Check network connectivity
   - Review error logs

3. **Cross-KB Query Failures**
   - Ensure all required KBs are initialized
   - Verify KB access permissions
   - Check data consistency

### Debug Mode

```python
# Enable debug logging
import logging
logging.basicConfig(level=logging.DEBUG)

# Router with debug configuration
router = MultiKBQueryRouter({
    **config,
    'debug': True,
    'log_queries': True
})
```

## Support

For additional support or questions about the Query Router:
- Review logs in `/var/log/cardiofit/query-router/`
- Check metrics dashboard at `http://localhost:3000/d/query-router`
- Contact the Platform Team for architectural questions