# Migration from Lab Microservice to Observation Microservice

This document explains the migration from the Lab Microservice to a more comprehensive Observation Microservice that handles all types of clinical observations.

## Overview

The Lab Microservice has been replaced with a more general Observation Microservice that handles all types of observations, including:

- Laboratory results
- Vital signs
- Physical measurements
- Social history
- Questionnaire responses
- Imaging findings

This change aligns better with the FHIR resource model where lab results are a type of Observation resource.

## Changes Made

1. **Created a New Observation Microservice**:
   - Set up a new service structure with FastAPI
   - Implemented models for different observation types
   - Created services for observation data management
   - Added endpoints for observation data access
   - Implemented HL7 message processing

2. **Removed Lab-Specific Microservice**:
   - Removed the Lab Microservice
   - Migrated lab functionality to the Observation Microservice

3. **Updated Integration Layer**:
   - Updated the FHIR service to route observation-related requests to the Observation Microservice
   - Updated the configuration to include the Observation Microservice URL

4. **Updated GraphQL Gateway**:
   - Updated the GraphQL gateway to communicate with the Observation Microservice
   - Implemented fallback to the FHIR service if the Observation Microservice is unavailable

## API Changes

### New Observation Microservice Endpoints

#### Core Observation Endpoints

- `POST /api/observations` - Create a new observation
- `GET /api/observations/{id}` - Get an observation by ID
- `PUT /api/observations/{id}` - Update an observation
- `DELETE /api/observations/{id}` - Delete an observation
- `GET /api/observations` - Search for observations
- `GET /api/observations/patient/{patient_id}` - Get observations for a patient
- `GET /api/observations/category/{category}` - Get observations by category

#### Vital Signs Endpoints

- `GET /api/vital-signs` - Get vital signs
- `GET /api/vital-signs/patient/{patient_id}` - Get vital signs for a patient
- `GET /api/vital-signs/blood-pressure` - Get blood pressure observations
- `GET /api/vital-signs/heart-rate` - Get heart rate observations
- `GET /api/vital-signs/respiratory-rate` - Get respiratory rate observations
- `GET /api/vital-signs/temperature` - Get temperature observations
- `GET /api/vital-signs/oxygen-saturation` - Get oxygen saturation observations

#### Laboratory Endpoints

- `GET /api/laboratory` - Get laboratory results
- `GET /api/laboratory/patient/{patient_id}` - Get laboratory results for a patient
- `GET /api/laboratory/cbc` - Get CBC (Complete Blood Count) results
- `GET /api/laboratory/bmp` - Get BMP (Basic Metabolic Panel) results
- `GET /api/laboratory/cmp` - Get CMP (Comprehensive Metabolic Panel) results
- `GET /api/laboratory/lipid-panel` - Get Lipid Panel results
- `GET /api/laboratory/urinalysis` - Get Urinalysis results

#### HL7 Endpoints

- `POST /api/hl7/process` - Process any HL7 message
- `POST /api/hl7/oru` - Process HL7 ORU message

### Updated FHIR Service Endpoints

- `/api/fhir/Patient/{id}/LabResults` - Now proxies to the Observation Microservice

## How to Use

### Processing HL7 ORU Messages

Send ORU messages to the Observation Microservice:

```bash
curl -X POST http://localhost:8005/api/hl7/process \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"message": "MSH|^~\\&|SENDING_APPLICATION|..."}'
```

### Getting Lab Results

Use the Observation Microservice directly:

```bash
curl -X GET http://localhost:8005/api/laboratory/patient/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Or use the FHIR service proxy:

```bash
curl -X GET http://localhost:8004/api/fhir/Patient/123/LabResults \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Getting Vital Signs

```bash
curl -X GET http://localhost:8005/api/vital-signs/patient/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Using GraphQL

The GraphQL gateway now communicates directly with the Observation Microservice:

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

## Benefits of the Observation Microservice Approach

1. **FHIR Alignment**: Better matches the FHIR resource model where lab results are a type of Observation.
2. **Broader Scope**: Handles all types of clinical observations, not just lab results.
3. **Future-proofing**: Easier to add new types of observations.
4. **Consistency**: All observations follow the same patterns for creation, retrieval, and search.
5. **Simplified Integration**: Other services only need to integrate with one microservice for all observation types.

## Running the Services

To run the services:

1. Start the FHIR service:
```bash
cd services/fhir-service
uvicorn app.main:app --host 0.0.0.0 --port 8004
```

2. Start the Observation Microservice:
```bash
cd services/observation-service
uvicorn app.main:app --host 0.0.0.0 --port 8005
```

3. Start the GraphQL gateway:
```bash
cd services/graphql-gateway
uvicorn app.main:app --host 0.0.0.0 --port 8006
```
