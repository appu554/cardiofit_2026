# CAE Clinical Intelligence - GraphDB Setup

This directory contains the RDF schema and sample data for the Clinical Assertion Engine (CAE) GraphDB integration.

## Files Overview

### 📋 Schema Files
- **`cae-clinical-schema.ttl`** - Complete RDF/OWL ontology defining the clinical intelligence schema
- **`cae-sample-data.ttl`** - Sample clinical data for testing and development

### 🔧 Import Tools
- **`import_to_graphdb.py`** - Python script to import schema and data into GraphDB
- **`README.md`** - This documentation file

## Quick Start

### 1. Prerequisites
- GraphDB running on `http://localhost:7200` (or update the URL in the script)
- Python 3.7+ with `requests` library
- GraphDB Desktop or GraphDB Free edition

### 2. Install GraphDB
Download and install GraphDB from: https://www.ontotext.com/products/graphdb/

### 3. Import Schema and Data
```bash
# Navigate to the graph directory
cd backend/services/clinical-reasoning-service/app/graph

# Install required Python packages
pip install requests

# Run the import script
python import_to_graphdb.py
```

### 4. Verify Import
The script will:
- ✅ Create a repository named `cae-clinical-intelligence`
- ✅ Import the clinical schema (classes, properties, relationships)
- ✅ Import sample clinical data (patients, medications, conditions)
- ✅ Run a test query to verify the import

## Schema Overview

### Core Clinical Entities
- **Patient** - Patient demographics and clinical context
- **Medication** - Medications with RxNorm codes and therapeutic classes
- **Condition** - Clinical conditions with SNOMED CT codes
- **Clinician** - Healthcare providers with specialties

### Dynamic Learning Relationships
- **DrugInteraction** - Drug interactions with confidence scores and learning capabilities
- **ClinicalAssertion** - CAE-generated assertions with provenance
- **ClinicalOutcome** - Clinical outcomes for learning
- **ClinicalOverride** - Clinician overrides for learning

### Advanced Features
- **Patient Similarity** - Graph-based patient similarity relationships
- **Temporal Patterns** - Time-based medication sequences
- **Context Vectors** - Patient similarity vectors for ML algorithms
- **Learning Relationships** - OVERRODE, EXPERIENCED, SIMILAR_TO, LEARNED_FROM

## Sample Data Included

### Patients
- `patient_001` - 65-year-old male with atrial fibrillation, hypertension
- `patient_002` - 58-year-old female with diabetes, hyperlipidemia  
- `patient_003` - 72-year-old male with hypertension, hyperlipidemia

### Medications
- Warfarin (anticoagulant)
- Aspirin (antiplatelet)
- Lisinopril (ACE inhibitor)
- Metformin (biguanide)
- Atorvastatin (statin)

### Drug Interactions
- Warfarin + Aspirin (critical interaction with learning data)

## SPARQL Query Examples

### 1. Find All Patients and Their Medications
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?patient ?patientId ?medication ?genericName WHERE {
    ?patient a cae:Patient ;
             cae:hasPatientId ?patientId ;
             cae:prescribedMedication ?medication .
    ?medication cae:hasGenericName ?genericName .
}
```

### 2. Find Drug Interactions with High Confidence
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?interaction ?severity ?confidence WHERE {
    ?interaction a cae:DrugInteraction ;
                 cae:hasInteractionSeverity ?severity ;
                 cae:hasConfidenceScore ?confidence .
    FILTER(?confidence > 0.8)
}
```

### 3. Find Similar Patients
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?patient1 ?patient2 WHERE {
    ?patient1 cae:similarTo ?patient2 .
}
```

### 4. Find Clinical Assertions by Type
```sparql
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

SELECT ?assertion ?type ?severity ?confidence WHERE {
    ?assertion a cae:ClinicalAssertion ;
               cae:hasAssertionType ?type ;
               cae:hasAssertionSeverity ?severity ;
               cae:hasAssertionConfidence ?confidence .
}
```

## Integration with CAE Service

### Update GraphDB Connection
In your CAE service configuration, update the GraphDB connection:

```python
# app/core/config.py
GRAPHDB_ENDPOINT = "http://localhost:7200"
GRAPHDB_REPOSITORY = "cae-clinical-intelligence"
```

### Schema Manager Integration
The `schema_manager.py` should connect to this repository:

```python
from app.graph.schema_manager import GraphSchemaManager

schema_manager = GraphSchemaManager(
    graphdb_endpoint="http://localhost:7200",
    repository="cae-clinical-intelligence"
)
```

## Troubleshooting

### Connection Issues
- Ensure GraphDB is running on the correct port
- Check firewall settings
- Verify repository name matches configuration

### Import Errors
- Check RDF syntax using online validators
- Ensure proper UTF-8 encoding
- Verify GraphDB has sufficient memory

### Query Issues
- Use GraphDB Workbench for interactive SPARQL queries
- Check namespace prefixes
- Validate query syntax

## Next Steps

1. **Connect CAE Service** - Update your CAE service to use this GraphDB repository
2. **Add Real Data** - Replace sample data with real clinical data sources
3. **Implement Learning** - Use the learning relationships for dynamic intelligence
4. **Performance Tuning** - Optimize queries and indexes for production use

## Production Considerations

- **Security** - Enable authentication and authorization in GraphDB
- **Backup** - Set up regular repository backups
- **Monitoring** - Monitor query performance and repository size
- **Scaling** - Consider GraphDB cluster for high availability
- **Data Privacy** - Ensure HIPAA compliance for patient data
