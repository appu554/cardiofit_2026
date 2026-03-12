# Neo4j Multi-KB Stream Manager - API Reference

## Classes

### MultiKBStreamManager

The main class for managing data streams across all CardioFit Knowledge Bases.

#### Constructor

```python
MultiKBStreamManager(config: Dict[str, Any])
```

**Parameters:**
- `config`: Configuration dictionary containing Neo4j connection details

**Configuration Options:**
```python
config = {
    'neo4j_uri': str,           # Neo4j connection URI (default: 'bolt://localhost:7687')
    'neo4j_user': str,          # Username (default: 'neo4j')
    'neo4j_password': str,      # Password (default: 'password')
}
```

#### Methods

##### initialize_all_streams()

```python
async def initialize_all_streams() -> bool
```

Initialize logical partitions for all knowledge bases, including indexes and constraints.

**Returns:**
- `bool`: Success status

**Example:**
```python
success = await manager.initialize_all_streams()
if success:
    print("All KB streams initialized successfully")
```

##### load_kb_data()

```python
async def load_kb_data(
    kb_id: Union[KnowledgeBase, str],
    stream_type: StreamType,
    entity_id: str,
    data: Dict[str, Any]
) -> bool
```

Load data into a specific knowledge base stream.

**Parameters:**
- `kb_id`: Knowledge Base identifier (enum or string)
- `stream_type`: Stream type (PATIENT, SEMANTIC, ANALYTICS, WORKFLOW)
- `entity_id`: Unique entity identifier
- `data`: Entity data dictionary

**Returns:**
- `bool`: Success status

**Example:**
```python
success = await manager.load_kb_data(
    KnowledgeBase.KB3_DRUG_CALCULATIONS,
    StreamType.PATIENT,
    "calc_rule_123",
    {
        'entity_type': 'CalculationRule',
        'drug_rxnorm': 'RX123456',
        'base_dose': 10,
        'unit': 'mg',
        'indication': 'hypertension'
    }
)
```

##### query_kb_stream()

```python
async def query_kb_stream(
    kb_id: Union[KnowledgeBase, str],
    stream_type: StreamType,
    query: str,
    params: Optional[Dict[str, Any]] = None
) -> List[Dict[str, Any]]
```

Query a specific knowledge base stream.

**Parameters:**
- `kb_id`: Knowledge Base identifier
- `stream_type`: Stream type to query
- `query`: Cypher query fragment (MATCH clause added automatically)
- `params`: Query parameters (optional)

**Returns:**
- `List[Dict[str, Any]]`: Query results

**Example:**
```python
results = await manager.query_kb_stream(
    KnowledgeBase.KB5_DRUG_INTERACTIONS,
    StreamType.PATIENT,
    "WHERE n.severity = $severity RETURN n.drug1_rxnorm, n.drug2_rxnorm",
    {'severity': 'severe'}
)
```

##### cross_kb_query()

```python
async def cross_kb_query(
    kb_list: List[Union[KnowledgeBase, str]],
    query: str,
    params: Optional[Dict[str, Any]] = None
) -> List[Dict[str, Any]]
```

Execute a query across multiple knowledge bases.

**Parameters:**
- `kb_list`: List of KB identifiers to include in query
- `query`: Complete Cypher query spanning multiple KBs
- `params`: Query parameters (optional)

**Returns:**
- `List[Dict[str, Any]]`: Combined query results

**Example:**
```python
results = await manager.cross_kb_query(
    [KnowledgeBase.KB1_PATIENT, KnowledgeBase.KB3_DRUG_CALCULATIONS],
    """
    MATCH (p:Patient:KB1_PatientStream {id: $patient_id})
    MATCH (c:CalculationRule:KB3_DrugCalculationStream)
    WHERE c.indication IN p.conditions
    RETURN p.name, c.drug_rxnorm, c.base_dose
    """,
    {'patient_id': 'patient_123'}
)
```

##### health_check_all_streams()

```python
async def health_check_all_streams() -> Dict[str, bool]
```

Check health status of all knowledge base streams.

**Returns:**
- `Dict[str, bool]`: Health status for each KB

**Example:**
```python
health = await manager.health_check_all_streams()
for kb_name, status in health.items():
    print(f"{kb_name}: {'✓' if status['healthy'] else '✗'}")
    if status['healthy']:
        print(f"  Primary nodes: {status['primary_nodes']}")
        print(f"  Semantic nodes: {status['semantic_nodes']}")
```

##### close()

```python
async def close() -> None
```

Close the Neo4j driver connection.

**Example:**
```python
await manager.close()
```

## Enums

### KnowledgeBase

Enumeration of supported knowledge bases.

```python
class KnowledgeBase(Enum):
    KB1_PATIENT = "kb1"
    KB2_GUIDELINES = "kb2"
    KB3_DRUG_CALCULATIONS = "kb3"
    KB4_SAFETY_RULES = "kb4"
    KB5_DRUG_INTERACTIONS = "kb5"
    KB6_EVIDENCE = "kb6"
    KB7_TERMINOLOGY = "kb7"
    KB8_WORKFLOWS = "kb8"
```

### StreamType

Enumeration of stream types within each knowledge base.

```python
class StreamType(Enum):
    PATIENT = "PatientStream"      # Primary operational data
    SEMANTIC = "SemanticStream"    # Relationships and semantic data
    ANALYTICS = "AnalyticsStream"  # Analytical/statistical data
    WORKFLOW = "WorkflowStream"    # Process and workflow data
```

## Stream Labels

Each knowledge base uses specific Neo4j labels for logical partitioning:

### Primary Streams
- `KB1_PatientStream` - Patient data
- `KB2_GuidelineStream` - Clinical guidelines
- `KB3_DrugCalculationStream` - Drug calculations
- `KB4_SafetyStream` - Safety rules
- `KB5_InteractionStream` - Drug interactions
- `KB6_EvidenceStream` - Evidence base
- `KB7_TerminologyStream` - Medical terminology
- `KB8_WorkflowStream` - Clinical workflows

### Semantic Streams
- `KB1_SemanticStream` through `KB8_SemanticStream`

### Shared Streams
- `SharedSemanticMesh` - Cross-KB concept relationships
- `GlobalPatientStream` - Patient identity across KBs
- `CrossKBRelationshipStream` - Explicit KB-to-KB links

## Error Handling

All async methods handle exceptions gracefully and return appropriate default values:

- `initialize_all_streams()`: Returns `False` on failure
- `load_kb_data()`: Returns `False` on failure
- `query_kb_stream()`: Returns empty list `[]` on failure
- `cross_kb_query()`: Returns empty list `[]` on failure
- `health_check_all_streams()`: Returns error status for each KB on failure

**Example Error Handling:**
```python
try:
    results = await manager.query_kb_stream(
        KnowledgeBase.KB1_PATIENT,
        StreamType.PATIENT,
        "WHERE n.invalid_field = $value RETURN n",
        {'value': 'test'}
    )
    if not results:
        print("No results found or query failed")
except Exception as e:
    logger.error(f"Query execution failed: {e}")
```

## Legacy Compatibility

### Neo4jDualStreamManager

Backward compatibility wrapper for KB7-specific usage.

```python
from multi_kb_stream_manager import Neo4jDualStreamManager

# Legacy interface for existing KB7 implementations
legacy_manager = Neo4jDualStreamManager(config)

# Maps to new multi-KB interface internally
await legacy_manager.load_patient_data(patient_id, patient_data)
await legacy_manager.load_semantic_concept(concept_uri, concept_data)
health_ok = await legacy_manager.health_check()
```

## Performance Considerations

- **Connection Pooling**: 100 concurrent connections by default
- **Indexing**: KB-specific indexes for optimal query performance
- **Batch Operations**: Use transactions for multiple operations
- **Query Optimization**: Leverage specific stream labels in queries

## Security Notes

- All data includes `kb_source` metadata for audit trails
- Timestamps automatically added for data lineage
- Use parameterized queries to prevent injection attacks
- Consider using Neo4j Enterprise for additional security features