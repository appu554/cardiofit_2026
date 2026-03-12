# Lab Microservice Migration

This document explains the migration of lab functionality from the FHIR service to a dedicated Lab Microservice.

## Overview

The lab functionality has been moved from the FHIR service to a dedicated Lab Microservice. This change follows a more modular architecture and allows for better separation of concerns.

## Changes Made

1. **Created a New Lab Microservice**:
   - Set up a new service structure with FastAPI
   - Implemented models for lab tests and panels
   - Created services for lab data management
   - Added endpoints for lab data access
   - Implemented HL7 ORU message processing

2. **Removed Lab-Specific Code from FHIR Service**:
   - Removed ORU message processing code
   - Removed ORU message model
   - Updated the FHIR service to proxy lab-related requests to the Lab Microservice

3. **Updated Integration Layer**:
   - Added routing for lab-related requests to the Lab Microservice
   - Updated the configuration to include the Lab Microservice URL

4. **Updated GraphQL Gateway**:
   - Added direct communication with the Lab Microservice for lab-related queries
   - Implemented fallback to the FHIR service if the Lab Microservice is unavailable

## API Changes

### New Lab Microservice Endpoints

- `POST /api/lab/tests` - Create a new lab test
- `GET /api/lab/tests/{test_id}` - Get a lab test by ID
- `GET /api/lab/tests` - Search for lab tests
- `POST /api/lab/panels` - Create a new lab panel
- `GET /api/lab/panels/{panel_id}` - Get a lab panel by ID
- `GET /api/lab/panels` - Search for lab panels
- `GET /api/lab/patient/{patient_id}/tests` - Get lab tests for a patient
- `GET /api/lab/patient/{patient_id}/panels` - Get lab panels for a patient
- `POST /api/hl7/process` - Process any HL7 message
- `POST /api/hl7/oru` - Process HL7 ORU message

### Updated FHIR Service Endpoints

- `/api/fhir/Patient/{id}/LabResults` - Now proxies to the Lab Microservice

## How to Use

### Processing HL7 ORU Messages

Send ORU messages to the Lab Microservice instead of the FHIR service:

```bash
curl -X POST http://localhost:8005/api/hl7/process \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"message": "MSH|^~\\&|SENDING_APPLICATION|..."}'
```

### Getting Lab Results

Use the Lab Microservice directly:

```bash
curl -X GET http://localhost:8005/api/lab/patient/123/tests \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Or use the FHIR service proxy (which will call the Lab Microservice):

```bash
curl -X GET http://localhost:8004/api/fhir/Patient/123/LabResults \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Using GraphQL

The GraphQL gateway now communicates directly with the Lab Microservice:

```graphql
query {
  patientLabResults(patientId: "123") {
    id
    code {
      coding {
        display
      }
    }
    valueQuantity {
      value
      unit
    }
  }
}
```

## Benefits of the Microservice Approach

1. **Separation of Concerns**: The Lab Microservice focuses solely on lab data, making it easier to maintain and extend.
2. **Scalability**: The Lab Microservice can be scaled independently of other services.
3. **Technology Independence**: The Lab Microservice can use technologies optimized for lab data processing.
4. **Team Autonomy**: Different teams can work on different microservices without interfering with each other.
5. **Resilience**: If the Lab Microservice fails, other services can continue to function.

## Running the Services

To run the services:

1. Start the FHIR service:
```bash
cd services/fhir-service
uvicorn app.main:app --host 0.0.0.0 --port 8004
```

2. Start the Lab Microservice:
```bash
cd services/lab-service
uvicorn app.main:app --host 0.0.0.0 --port 8005
```

3. Start the GraphQL gateway:
```bash
cd services/graphql-gateway
uvicorn app.main:app --host 0.0.0.0 --port 8006
```
