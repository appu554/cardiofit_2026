# Neo4j Multi-KB Stream Manager

A shared infrastructure component for managing data streams across all CardioFit Knowledge Bases using logical partitioning in Neo4j.

## Overview

The **Neo4j Multi-KB Stream Manager** serves as the central data orchestration layer for all eight knowledge bases in the CardioFit clinical platform. It implements a sophisticated logical partitioning strategy within a single Neo4j database to maintain data isolation while enabling powerful cross-knowledge-base queries essential for clinical decision support.

## Architecture

### Knowledge Base Support

The system manages eight distinct knowledge bases:

- **KB1**: Patient data and demographics
- **KB2**: Clinical practice guidelines
- **KB3**: Drug dosage and calculations
- **KB4**: Clinical safety protocols
- **KB5**: Medication interactions
- **KB6**: Evidence-based medicine
- **KB7**: Medical terminology/ontologies
- **KB8**: Clinical workflow processes

### Dual-Stream Pattern

Each knowledge base implements a dual-stream architecture:

- **Primary Stream**: Core operational data (patients, guidelines, drug rules)
- **Semantic Stream**: Relationships, classifications, and semantic connections

This separation allows for optimized queries while maintaining semantic integrity across the healthcare domain.

## Key Features

- **🔄 Logical Partitioning**: Uses Neo4j labels for KB isolation without separate databases
- **🔗 Cross-KB Queries**: Enables powerful queries spanning multiple knowledge bases
- **🏥 Healthcare Optimized**: Designed for HIPAA compliance and clinical workflows
- **📈 Scalable Architecture**: High-performance connection pooling for multi-service access
- **🔍 Comprehensive Monitoring**: Health checks and performance monitoring for all streams
- **🔄 Backward Compatibility**: Maintains compatibility with existing KB7-specific implementations

## Quick Start

### Installation

```bash
pip install -r requirements.txt
```

### Basic Usage

```python
from multi_kb_stream_manager import MultiKBStreamManager, KnowledgeBase, StreamType

# Initialize manager
config = {
    'neo4j_uri': 'bolt://localhost:7687',
    'neo4j_user': 'neo4j',
    'neo4j_password': 'password'
}

manager = MultiKBStreamManager(config)

# Initialize all knowledge base streams
await manager.initialize_all_streams()

# Load patient data into KB1
await manager.load_kb_data(
    KnowledgeBase.KB1_PATIENT,
    StreamType.PATIENT,
    "patient_123",
    {
        'entity_type': 'Patient',
        'name': 'John Doe',
        'mrn': 'MRN123456'
    }
)

# Query specific KB stream
results = await manager.query_kb_stream(
    KnowledgeBase.KB1_PATIENT,
    StreamType.PATIENT,
    "WHERE n.mrn = $mrn RETURN n",
    {'mrn': 'MRN123456'}
)
```

### Cross-KB Queries

```python
# Query across multiple knowledge bases
cross_kb_results = await manager.cross_kb_query(
    [KnowledgeBase.KB1_PATIENT, KnowledgeBase.KB5_DRUG_INTERACTIONS],
    """
    MATCH (p:Patient:KB1_PatientStream)-[:PRESCRIBED]->(m:Medication)
    MATCH (i:Interaction:KB5_InteractionStream)
    WHERE i.drug1_rxnorm = m.rxnorm AND i.severity = 'severe'
    RETURN p.name, m.name, i.description
    """
)
```

## Documentation

- [API Reference](./docs/API.md) - Complete API documentation
- [Architecture Guide](./docs/ARCHITECTURE.md) - System design and patterns
- [Usage Examples](./docs/EXAMPLES.md) - Practical implementation examples
- [Configuration Guide](./docs/CONFIGURATION.md) - Setup and deployment

## Requirements

- Python 3.8+
- Neo4j 5.12+ (Community or Enterprise)
- See `requirements.txt` for complete dependency list

## Health Monitoring

```python
# Check health of all knowledge base streams
health_status = await manager.health_check_all_streams()
print(f"KB1 Status: {health_status['kb1']['healthy']}")
print(f"KB1 Nodes: {health_status['kb1']['primary_nodes']}")
```

## Security and Compliance

- HIPAA-compliant data handling
- Comprehensive audit logging
- Data lineage tracking
- Role-based access patterns

## Performance

- Optimized for healthcare workloads
- Connection pooling (100 concurrent connections)
- KB-specific indexing strategies
- Efficient cross-KB query patterns

## Contributing

This is a shared infrastructure component. Changes should be:

1. Tested across all dependent knowledge bases
2. Backwards compatible with existing implementations
3. Documented with usage examples
4. Performance tested with realistic healthcare data volumes

## License

Internal CardioFit platform component.