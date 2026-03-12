# Unit Tests for Shared FHIR Models

This directory contains unit tests for the shared FHIR models used in the Clinical Synthesis Hub.

## Test Structure

- `test_base.py`: Tests for the base FHIR models
- `test_datatypes.py`: Tests for FHIR datatypes
- `test_resources.py`: Tests for FHIR resources
- `test_validators.py`: Tests for FHIR validators
- `run_tests.py`: Script to run all tests

## Running the Tests

To run all tests:

```bash
cd backend
python -m shared.models.tests.run_tests
```

To run a specific test module:

```bash
cd backend
python -m shared.models.tests.test_base
python -m shared.models.tests.test_datatypes
python -m shared.models.tests.test_resources
python -m shared.models.tests.test_validators
```

## Test Coverage

These tests cover:

1. **Base Models**:
   - Initialization
   - Conversion to/from FHIR dictionaries
   - Handling of extra fields

2. **Datatypes**:
   - Initialization of common datatypes
   - Conversion to dictionaries

3. **Resources**:
   - Initialization of Patient, Observation, and Condition resources
   - Conversion to/from FHIR dictionaries and models
   - Field validation

4. **Validators**:
   - Validation of valid and invalid resources
   - Error handling
