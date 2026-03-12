# Data Transformation Layer for Clinical Synthesis Hub

This package provides a robust layer for transforming data between different formats, particularly between FHIR models and GraphQL types.

## Overview

The transformers package provides:

1. **Base Transformers**: Common base classes for all transformers
2. **FHIR to GraphQL Transformers**: Transformers for converting FHIR models to GraphQL types
3. **GraphQL to FHIR Transformers**: Transformers for converting GraphQL input types to FHIR models
4. **Utility Functions**: Functions for case conversion, type conversion, and other common tasks
5. **Error Handling**: Custom exceptions and validation for transformation errors

## Directory Structure

```
backend/shared/transformers/
├── __init__.py         # Package exports
├── base.py             # Base transformer classes
├── exceptions.py       # Custom exceptions
├── fhir_to_graphql/    # FHIR to GraphQL transformers
│   ├── __init__.py
│   ├── base.py         # Base FHIR to GraphQL transformer
│   ├── patient.py      # Patient transformer
│   ├── observation.py  # Observation transformer
│   └── condition.py    # Condition transformer
├── graphql_to_fhir/    # GraphQL to FHIR transformers
│   ├── __init__.py
│   ├── base.py         # Base GraphQL to FHIR transformer
│   ├── patient.py      # Patient input transformer
│   ├── observation.py  # Observation input transformer
│   └── condition.py    # Condition input transformer
├── utils/              # Utility functions
│   ├── __init__.py
│   ├── case_conversion.py  # Case conversion utilities
│   └── type_conversion.py  # Type conversion utilities
└── README.md           # This file
```

## Usage

### Basic Usage

```python
# Import transformers
from shared.transformers import PatientTransformer, PatientInputTransformer

# Import models
from shared.models import Patient
from app.graphql.types import PatientType, PatientInput

# Transform FHIR Patient to GraphQL PatientType
patient = Patient(id="patient-1", name=[{"family": "Smith", "given": ["John"]}])
patient_transformer = PatientTransformer()
patient_type = patient_transformer.transform(patient)

# Transform GraphQL PatientInput to FHIR Patient
patient_input = PatientInput(id="patient-2", name=[{"family": "Doe", "given": ["Jane"]}])
patient_input_transformer = PatientInputTransformer()
patient_model = patient_input_transformer.transform(patient_input)
```

### Using the Transformer Registry

```python
from shared.transformers import TransformerRegistry
from shared.models import Patient
from app.graphql.types import PatientType

# Register transformers (typically done at application startup)
from shared.transformers import PatientTransformer
TransformerRegistry.register(PatientTransformer)

# Transform using the registry
patient = Patient(id="patient-1", name=[{"family": "Smith", "given": ["John"]}])
patient_type = TransformerRegistry.transform(patient, PatientType)
```

### Error Handling

```python
from shared.transformers import TransformationError, ValidationError

try:
    patient_type = patient_transformer.transform(patient)
except ValidationError as e:
    # Handle validation errors
    print(f"Validation error: {e}")
    for error in e.validation_errors:
        print(f"  - {error}")
except TransformationError as e:
    # Handle other transformation errors
    print(f"Transformation error: {e}")
```

## Extending the Transformers

### Creating a New Transformer

To create a new transformer for a custom FHIR resource:

1. Create a new file in the appropriate directory (e.g., `fhir_to_graphql/custom_resource.py`)
2. Define a transformer class that extends the base transformer
3. Register the transformer with the registry

Example:

```python
from shared.models import CustomResource
from app.graphql.types import CustomResourceType
from shared.transformers.fhir_to_graphql.base import FHIRToGraphQLTransformer
from shared.transformers import TransformerRegistry

class CustomResourceTransformer(FHIRToGraphQLTransformer[CustomResource, CustomResourceType]):
    """Transformer for CustomResource to CustomResourceType."""
    
    source_type = CustomResource
    target_type = CustomResourceType
    
    def _transform_nested_objects(self, data):
        # Handle any special transformations for nested objects
        return data

# Register the transformer
TransformerRegistry.register(CustomResourceTransformer)
```
