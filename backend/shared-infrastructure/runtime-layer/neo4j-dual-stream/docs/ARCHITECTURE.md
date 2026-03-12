# Architecture Guide - Neo4j Multi-KB Stream Manager

This document provides a comprehensive overview of the architectural decisions, design patterns, and technical implementation of the Neo4j Multi-KB Stream Manager.

## Table of Contents

- [System Overview](#system-overview)
- [Architectural Principles](#architectural-principles)
- [Logical Partitioning Strategy](#logical-partitioning-strategy)
- [Dual-Stream Architecture](#dual-stream-architecture)
- [Cross-KB Query Architecture](#cross-kb-query-architecture)
- [Performance Architecture](#performance-architecture)
- [Security Architecture](#security-architecture)
- [Scalability Patterns](#scalability-patterns)

## System Overview

### Purpose and Scope

The Neo4j Multi-KB Stream Manager serves as the **unified data orchestration layer** for the CardioFit clinical platform, managing eight distinct knowledge bases within a single Neo4j graph database. This design enables both **data isolation** for regulatory compliance and **cross-domain queries** for clinical decision support.

### Key Design Goals

1. **Clinical Safety**: Maintain strict data lineage and audit trails
2. **Regulatory Compliance**: Support HIPAA, FDA, and clinical audit requirements
3. **Performance**: Optimize for healthcare workload patterns
4. **Flexibility**: Enable both isolated and cross-KB analytical queries
5. **Maintainability**: Provide clear separation of concerns across clinical domains

## Architectural Principles

### 1. Logical Partitioning over Physical Separation

**Decision**: Use Neo4j labels for logical partitioning instead of separate databases

**Rationale**:
- Enables cross-KB queries without complex federation
- Reduces infrastructure complexity and operational overhead
- Maintains transaction consistency across related clinical data
- Leverages Neo4j's native graph capabilities for semantic relationships

**Trade-offs**:
- Requires careful access control and query optimization
- Single point of failure (mitigated by Neo4j clustering in production)
- Shared resource contention (managed through connection pooling)

### 2. Dual-Stream Pattern

**Decision**: Separate operational and semantic data streams within each KB

**Rationale**:
- **Operational Stream**: Fast access to clinical data (patients, medications, guidelines)
- **Semantic Stream**: Rich relationships and ontological connections
- Different access patterns require different optimization strategies
- Enables specialized indexing and query patterns

### 3. Shared Semantic Layer

**Decision**: Implement `SharedSemanticMesh` for cross-KB relationships

**Rationale**:
- Clinical concepts often span multiple knowledge domains
- Enables federated queries while preserving KB boundaries
- Supports clinical reasoning that requires multiple knowledge sources
- Maintains semantic consistency across the platform

## Logical Partitioning Strategy

### Neo4j Label Architecture

```
Node Labels Hierarchy:
├── Entity Type (Patient, Guideline, Interaction, etc.)
├── KB-Specific Stream (KB1_PatientStream, KB5_InteractionStream, etc.)
├── Generic Stream Type (PatientStream, SemanticStream)
└── Shared Streams (SharedSemanticMesh, GlobalPatientStream)
```

### Example Node Labeling

```cypher
-- Patient in KB1
(:Patient:KB1_PatientStream:PatientStream {id: "patient_123", kb_source: "kb1"})

-- Same patient in global context
(:Patient:GlobalPatientStream:PatientStream {global_patient_id: "patient_123"})

-- Drug interaction in KB5
(:Interaction:KB5_InteractionStream:PatientStream {
  id: "interaction_456",
  drug1_rxnorm: "RX123",
  drug2_rxnorm: "RX456",
  kb_source: "kb5"
})

-- Semantic relationship in shared mesh
(:Concept:SharedSemanticMesh:SemanticStream {
  global_uri: "http://snomed.info/sct/123456",
  references_kb1: true,
  references_kb5: true
})
```

### Query Targeting Strategies

#### KB-Specific Queries
```cypher
-- Targets only KB1 patient data
MATCH (p:Patient:KB1_PatientStream)
WHERE p.mrn = 'MRN123456'
RETURN p
```

#### Cross-KB Queries
```cypher
-- Spans multiple KBs using shared semantic mesh
MATCH (p:Patient:KB1_PatientStream)-[:HAS_CONDITION]->(c:Concept:SharedSemanticMesh)
MATCH (c)<-[:TREATS]-(d:Drug:KB3_DrugCalculationStream)
RETURN p.name, c.display_name, d.name
```

## Dual-Stream Architecture

### Stream Classification

| Stream Type | Purpose | Data Characteristics | Query Patterns |
|-------------|---------|---------------------|----------------|
| **Primary Stream** | Operational data | High-frequency access, transactional | Point queries, patient lookups |
| **Semantic Stream** | Relationships & ontologies | Lower frequency, analytical | Graph traversals, reasoning queries |
| **Analytics Stream** | Derived metrics | Read-heavy, aggregated | Reporting, dashboards |
| **Workflow Stream** | Process data | Time-series, state-based | Workflow tracking, auditing |

### Stream Selection Logic

```python
def get_stream_label(kb_id: KnowledgeBase, stream_type: StreamType) -> str:
    """Intelligent stream selection based on data type and usage pattern"""

    if stream_type == StreamType.PATIENT:
        # Primary operational data
        return self.kb_streams[kb_id]['primary']
    elif stream_type == StreamType.SEMANTIC:
        # Relationship and semantic data
        return self.kb_streams[kb_id]['semantic']
    else:
        # Default to primary for backward compatibility
        return self.kb_streams[kb_id]['primary']
```

### Data Routing Patterns

#### Write Path
```
Clinical Data → Stream Type Analysis → KB Classification → Label Assignment → Neo4j Write
```

#### Read Path
```
Query Intent → Stream Selection → KB Targeting → Label Filtering → Result Assembly
```

## Cross-KB Query Architecture

### Query Categories

#### 1. Federation Queries
Combine data from multiple KBs while preserving boundaries:

```cypher
MATCH (p:Patient:KB1_PatientStream {mrn: $patient_mrn})
MATCH (i:Interaction:KB5_InteractionStream)
WHERE any(med IN p.current_medications WHERE med IN [i.drug1_name, i.drug2_name])
RETURN p, collect(i) AS potential_interactions
```

#### 2. Semantic Queries
Leverage shared semantic mesh for conceptual relationships:

```cypher
MATCH (concept:Concept:SharedSemanticMesh)
WHERE concept.global_uri = $snomed_code
MATCH (concept)-[:REFERENCED_BY]->(kb_entity)
WHERE any(label IN labels(kb_entity) WHERE label ENDS WITH 'Stream')
RETURN concept, collect(distinct [labels(kb_entity)[0], kb_entity.kb_source]) AS kb_references
```

#### 3. Clinical Decision Support Queries
Multi-KB queries for clinical reasoning:

```cypher
-- Find treatment recommendations considering patient conditions,
-- drug interactions, and clinical guidelines
MATCH (p:Patient:KB1_PatientStream {mrn: $mrn})-[:HAS_CONDITION]->(condition)
MATCH (g:Guideline:KB2_GuidelineStream)
WHERE condition.snomed_code IN g.applicable_conditions

MATCH (p)-[:PRESCRIBED]->(current_med)
MATCH (g)-[:RECOMMENDS]->(recommended_med)
OPTIONAL MATCH (i:Interaction:KB5_InteractionStream)
WHERE i.drug1_name = current_med.name AND i.drug2_name = recommended_med.name

RETURN p.name, condition.display_name, g.title, recommended_med.name,
       CASE WHEN i IS NOT NULL THEN i.severity ELSE 'no_interaction' END AS interaction_risk
```

### Query Optimization Patterns

#### Label-First Filtering
```cypher
-- Efficient: Filter by specific labels first
MATCH (n:KB1_PatientStream)
WHERE n.mrn = $mrn

-- Inefficient: Generic label with property filtering
MATCH (n)
WHERE n.kb_source = 'kb1' AND n.mrn = $mrn
```

#### Stream-Specific Indexing
```cypher
-- KB-specific indexes for optimal performance
CREATE INDEX kb1_patient_mrn FOR (p:Patient:KB1_PatientStream) ON (p.mrn)
CREATE INDEX kb5_interaction_drugs FOR (i:Interaction:KB5_InteractionStream) ON (i.drug1_rxnorm, i.drug2_rxnorm)
```

## Performance Architecture

### Connection Management

#### Connection Pool Configuration
```python
# Optimized for healthcare workloads
AsyncGraphDatabase.driver(
    uri,
    auth=auth,
    max_connection_pool_size=100,    # High concurrency for multi-service access
    connection_acquisition_timeout=30,  # Healthcare SLA requirements
    max_transaction_retry_time=15    # Clinical data consistency requirements
)
```

#### Load Balancing Strategy
- **Read Replicas**: Route analytical queries to read replicas
- **Write Primary**: Clinical data updates go to primary instance
- **Stream Routing**: Different streams can target different cluster members

### Indexing Strategy

#### KB-Specific Indexes
```cypher
-- Patient identification indexes
CREATE INDEX kb1_patient_id FOR (p:Patient:KB1_PatientStream) ON (p.id)
CREATE INDEX kb1_patient_mrn FOR (p:Patient:KB1_PatientStream) ON (p.mrn)

-- Clinical lookup indexes
CREATE INDEX kb2_guideline_condition FOR (g:Guideline:KB2_GuidelineStream) ON (g.condition_category)
CREATE INDEX kb5_interaction_severity FOR (i:Interaction:KB5_InteractionStream) ON (i.severity)

-- Temporal indexes for audit trails
CREATE INDEX global_entity_updated FOR (e) ON (e.updated) WHERE e.kb_source IS NOT NULL
```

#### Composite Indexes for Cross-KB Queries
```cypher
-- Multi-property indexes for complex queries
CREATE INDEX kb5_drug_interaction_lookup FOR (i:Interaction:KB5_InteractionStream)
ON (i.drug1_rxnorm, i.drug2_rxnorm, i.severity)
```

### Query Performance Patterns

#### Query Plan Optimization
```cypher
-- Use EXPLAIN and PROFILE for query optimization
EXPLAIN MATCH (p:Patient:KB1_PatientStream)
WHERE p.mrn = $mrn
RETURN p

-- Leverage specific labels for better performance
PROFILE MATCH (p:Patient:KB1_PatientStream)-[:HAS_CONDITION]->(c)
RETURN p.name, collect(c.display_name)
```

#### Caching Strategy
- **Application Level**: Cache frequent KB health checks
- **Query Level**: Use Neo4j query caching for complex cross-KB queries
- **Result Level**: Cache clinical decision support results with TTL

## Security Architecture

### Data Lineage and Audit

#### Automatic Metadata Injection
```python
enhanced_data = {
    **original_data,
    'kb_source': kb_id.value,              # Source KB tracking
    'stream_type': stream_type.value,      # Stream classification
    'updated': datetime.utcnow().isoformat(),  # Timestamp for audit
    'operation_id': generate_operation_id(),   # Transaction tracking
    'user_context': get_current_user_context()  # User attribution
}
```

#### Audit Trail Queries
```cypher
-- Find all modifications to a specific patient
MATCH (p:Patient:KB1_PatientStream {mrn: $mrn})
RETURN p.updated, p.operation_id, p.user_context
ORDER BY p.updated DESC

-- Track cross-KB data relationships
MATCH (n)-[r:CROSS_KB_RELATION]->(m)
WHERE r.created > $start_date
RETURN n.kb_source, m.kb_source, r.relationship_type, r.created
```

### Access Control Patterns

#### Role-Based KB Access
```python
class KBAccessControl:
    def validate_kb_access(self, user_role: str, kb_id: KnowledgeBase) -> bool:
        """Validate user access to specific knowledge base"""
        access_matrix = {
            'clinician': [KB1_PATIENT, KB2_GUIDELINES, KB5_DRUG_INTERACTIONS],
            'pharmacist': [KB3_DRUG_CALCULATIONS, KB5_DRUG_INTERACTIONS, KB7_TERMINOLOGY],
            'administrator': list(KnowledgeBase),  # All KBs
            'researcher': [KB6_EVIDENCE, KB7_TERMINOLOGY]  # De-identified KBs only
        }
        return kb_id in access_matrix.get(user_role, [])
```

#### Query-Level Security
```cypher
-- Row-level security through query modification
MATCH (p:Patient:KB1_PatientStream)
WHERE p.organization_id = $user_organization_id
  AND p.consent_research = true  -- Consent-based filtering
RETURN p
```

## Scalability Patterns

### Horizontal Scaling

#### Neo4j Cluster Architecture
```yaml
neo4j_cluster:
  core_servers: 3              # Consensus and write coordination
  read_replicas: 5             # Scale read queries horizontally
  load_balancer:
    write_routing: core_servers
    read_routing: read_replicas
    analytics_routing: dedicated_analytics_replicas
```

#### Stream-Based Scaling
- **KB Partitioning**: Distribute KBs across different cluster members
- **Stream Partitioning**: Route different stream types to optimized nodes
- **Geographic Distribution**: Replicate critical KBs across data centers

### Vertical Scaling Considerations

#### Memory Optimization
```yaml
neo4j_config:
  dbms.memory.heap.initial_size: "4G"
  dbms.memory.heap.max_size: "8G"
  dbms.memory.pagecache.size: "16G"  # Large page cache for healthcare data
```

#### Storage Optimization
- **SSD Storage**: Critical for clinical query performance
- **Backup Strategy**: Continuous backups for healthcare regulatory requirements
- **Archival Strategy**: Move historical data to cold storage while maintaining access

### Monitoring and Observability

#### Health Check Architecture
```python
async def comprehensive_health_check(self) -> Dict[str, Any]:
    """Multi-dimensional health monitoring"""
    return {
        'database_connectivity': await self._check_db_connection(),
        'kb_stream_health': await self._check_all_kb_streams(),
        'query_performance': await self._benchmark_typical_queries(),
        'data_consistency': await self._validate_cross_kb_integrity(),
        'security_compliance': await self._audit_access_patterns()
    }
```

#### Performance Metrics
- **Query Response Times**: Track P95/P99 for clinical queries
- **Connection Pool Utilization**: Monitor for capacity planning
- **Cross-KB Query Complexity**: Identify optimization opportunities
- **Data Growth Rates**: Plan storage scaling proactively

This architecture enables the Neo4j Multi-KB Stream Manager to serve as a robust, scalable, and compliant foundation for clinical data management while supporting the complex analytical requirements of modern healthcare systems.