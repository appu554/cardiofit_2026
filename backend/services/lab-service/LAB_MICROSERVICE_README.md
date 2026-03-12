# Lab Microservice for Clinical Synthesis Hub

This document explains the Lab Microservice architecture and how it integrates with the rest of the Clinical Synthesis Hub.

## Overview

The Lab Microservice is a dedicated service for managing laboratory data in the Clinical Synthesis Hub. It provides:

1. Processing of HL7 ORU (Observation Result) messages
2. Storage of lab tests and panels
3. REST API for lab data access
4. Integration with the FHIR service and GraphQL gateway

## Architecture

The Lab Microservice follows a layered architecture:

1. **API Layer**: REST endpoints for lab data access
2. **Service Layer**: Business logic for lab data processing
3. **Data Layer**: MongoDB storage for lab data
4. **Integration Layer**: Communication with other services

## Integration with Other Services

### FHIR Service Integration

The Lab Microservice integrates with the FHIR service through the FHIR Integration Layer. The FHIR service routes laboratory-related requests to the Lab Microservice.

When a client requests lab data through the FHIR API:
1. The FHIR service receives the request
2. The FHIR Integration Layer identifies it as a lab-related request
3. The request is routed to the Lab Microservice
4. The Lab Microservice processes the request and returns the data
5. The FHIR service maps the lab data to FHIR resources and returns them to the client

### GraphQL Gateway Integration

The GraphQL gateway directly communicates with the Lab Microservice for lab-related queries:

1. The GraphQL gateway receives a lab-related query
2. The gateway calls the Lab Microservice API
3. The Lab Microservice returns the lab data
4. The GraphQL gateway maps the lab data to GraphQL types and returns them to the client

## API Endpoints

### HL7 Endpoints

- `POST /api/hl7/process`: Process any HL7 message
- `POST /api/hl7/oru`: Process HL7 ORU message

### Lab Endpoints

- `POST /api/lab/tests`: Create a new lab test
- `GET /api/lab/tests/{test_id}`: Get a lab test by ID
- `GET /api/lab/tests`: Search for lab tests
- `POST /api/lab/panels`: Create a new lab panel
- `GET /api/lab/panels/{panel_id}`: Get a lab panel by ID
- `GET /api/lab/panels`: Search for lab panels
- `GET /api/lab/patient/{patient_id}/tests`: Get lab tests for a patient
- `GET /api/lab/patient/{patient_id}/panels`: Get lab panels for a patient

## Data Models

### Lab Test

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "test_code": "WBC",
  "test_name": "WHITE BLOOD CELL COUNT",
  "value": 8.5,
  "unit": "10*3/uL",
  "reference_range": "4.0-11.0",
  "interpretation": "normal",
  "status": "final",
  "category": "laboratory",
  "effective_date_time": "2023-06-15T08:00:00",
  "patient_id": "123e4567-e89b-12d3-a456-426614174001",
  "order_number": "LAB12345",
  "specimen_type": "BLOOD"
}
```

### Lab Panel

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174002",
  "panel_code": "CBC",
  "panel_name": "COMPLETE BLOOD COUNT",
  "tests": [
    {
      "test_code": "WBC",
      "test_name": "WHITE BLOOD CELL COUNT",
      "value": 8.5,
      "unit": "10*3/uL",
      "reference_range": "4.0-11.0",
      "interpretation": "normal",
      "status": "final",
      "category": "laboratory",
      "effective_date_time": "2023-06-15T08:00:00",
      "patient_id": "123e4567-e89b-12d3-a456-426614174001"
    }
  ],
  "effective_date_time": "2023-06-15T08:00:00",
  "patient_id": "123e4567-e89b-12d3-a456-426614174001",
  "order_number": "LAB12345",
  "specimen_type": "BLOOD"
}
```

## Running the Service

### Using Docker

```bash
docker build -t lab-service .
docker run -p 8005:8005 lab-service
```

### Using Python

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8005 --reload
```

## Testing

### Process an HL7 ORU Message

```bash
curl -X POST http://localhost:8005/api/hl7/process \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"message": "MSH|^~\\&|SENDING_APPLICATION|..."}'
```

### Get Lab Tests for a Patient

```bash
curl -X GET http://localhost:8005/api/lab/patient/123/tests \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Benefits of the Microservice Approach

1. **Separation of Concerns**: The Lab Microservice focuses solely on lab data, making it easier to maintain and extend.
2. **Scalability**: The Lab Microservice can be scaled independently of other services.
3. **Technology Independence**: The Lab Microservice can use technologies optimized for lab data processing.
4. **Team Autonomy**: Different teams can work on different microservices without interfering with each other.
5. **Resilience**: If the Lab Microservice fails, other services can continue to function.

## Future Enhancements

1. **Lab Result Notifications**: Implement a notification system for new lab results.
2. **Lab Result Trending**: Add support for trending lab results over time.
3. **Advanced Search**: Implement advanced search capabilities for lab data.
4. **Lab Order Management**: Add support for lab order management.
5. **Integration with External Lab Systems**: Add support for integration with external lab systems.
