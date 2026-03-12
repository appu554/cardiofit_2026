# Shared FHIR Models for Clinical Synthesis Hub

This package provides shared Pydantic models for FHIR resources used across all microservices in the Clinical Synthesis Hub.

## Overview

The shared models package provides:

1. **Base Models**: Common base classes for all FHIR resources
2. **FHIR Datatypes**: Pydantic models for FHIR datatypes (e.g., CodeableConcept, Reference)
3. **FHIR Resources**: Pydantic models for FHIR resources (e.g., Patient, Observation)
4. **Validators**: Functions for validating FHIR resources

## Directory Structure

```
backend/shared/models/
├── __init__.py         # Package exports
├── base.py             # Base models and common functionality
├── datatypes/          # FHIR datatypes
│   ├── __init__.py
│   └── complex.py      # Complex datatypes
├── resources/          # FHIR resources
│   ├── __init__.py
│   ├── patient.py
│   ├── observation.py
│   ├── condition.py
│   ├── encounter.py
│   ├── medication.py
│   └── diagnostic.py
├── validators/         # Custom validators
│   ├── __init__.py
│   └── fhir.py
├── test_models.py      # Test script
└── README.md           # This file
```

## Usage

### Importing Models

```python
# Import specific models
from shared.models import Patient, Observation, Condition

# Import datatypes
from shared.models import CodeableConcept, Reference, Quantity

# Import validators
from shared.models import validate_fhir_resource
```

### Creating a Patient

```python
from shared.models import Patient

patient = Patient(
    id="patient-1",
    active=True,
    gender="male",
    birthDate="1970-01-01",
    name=[
        {
            "family": "Smith",
            "given": ["John"],
            "use": "official"
        }
    ],
    telecom=[
        {
            "system": "phone",
            "value": "555-123-4567",
            "use": "home"
        }
    ]
)

# Convert to FHIR JSON
fhir_json = patient.dict(exclude_none=True)

# Convert to FHIR model from fhir.resources
fhir_patient = patient.to_fhir_model()
```

### Creating an Observation

```python
from shared.models import Observation, CodeableConcept, Coding, Reference, Quantity

observation = Observation(
    id="observation-1",
    status="final",
    code=CodeableConcept(
        coding=[
            Coding(
                system="http://loinc.org",
                code="8867-4",
                display="Heart rate"
            )
        ],
        text="Heart rate"
    ),
    subject=Reference(
        reference="Patient/patient-1",
        display="Smith, John"
    ),
    effectiveDateTime="2023-07-01T12:00:00Z",
    valueQuantity=Quantity(
        value=80,
        unit="beats/minute",
        system="http://unitsofmeasure.org",
        code="/min"
    )
)
```

### Validating a FHIR Resource

```python
from shared.models import validate_fhir_resource

try:
    # Validate a FHIR resource
    validate_fhir_resource(patient.dict(exclude_none=True), "Patient")
    print("Patient is valid")
except Exception as e:
    print(f"Patient validation failed: {e}")
```

## Testing

Run the test script to verify that the models work correctly:

```bash
cd backend
python -m shared.models.test_models
```

## Integration with Microservices

To use these models in a microservice:

1. Make sure the `shared` module is in your Python path
2. Import the models you need
3. Use them in your API endpoints, services, and data access layers

Example:

```python
from fastapi import APIRouter, Depends
from shared.models import Patient

router = APIRouter()

@router.post("/patients", response_model=Patient)
async def create_patient(patient: Patient):
    # Save the patient to the database
    # ...
    return patient
```
