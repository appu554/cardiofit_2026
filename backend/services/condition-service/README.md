# Condition Microservice

This microservice provides a comprehensive API for managing FHIR Condition resources, including patient problems, diagnoses, and other clinical conditions. It follows the FHIR resource model and integrates with the FHIR server.

## Features

- Manage FHIR Condition resources
- Search and filter conditions by various criteria
- Patient-specific endpoints for condition data
- Integration with FHIR server

## API Endpoints

### Condition Endpoints

#### General Condition Endpoints
- `POST /api/conditions` - Create a new condition
- `GET /api/conditions/{id}` - Get a condition by ID
- `PUT /api/conditions/{id}` - Update a condition
- `DELETE /api/conditions/{id}` - Delete a condition
- `GET /api/conditions` - Search for conditions
- `GET /api/conditions/patient/{patient_id}` - Get conditions for a patient

#### Problem List Endpoints
- `POST /api/conditions/problems` - Create a new problem list item
- `GET /api/conditions/patient/{patient_id}/problems` - Get problem list items for a patient

#### Diagnosis Endpoints
- `POST /api/conditions/diagnoses` - Create a new encounter diagnosis
- `GET /api/conditions/patient/{patient_id}/diagnoses` - Get encounter diagnoses for a patient

#### Health Concern Endpoints
- `POST /api/conditions/health-concerns` - Create a new health concern
- `GET /api/conditions/patient/{patient_id}/health-concerns` - Get health concerns for a patient

### HL7 Endpoints (Future Implementation)

- `POST /api/hl7/process` - Process any HL7 message

## Running the Service

### Using Docker

```bash
docker build -t condition-service .
docker run -p 8019:8019 condition-service
```

### Using Python

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8019 --reload
```

## Environment Variables

- `MONGODB_URL` - MongoDB connection URL (default: MongoDB Atlas URL)
- `MONGODB_DB_NAME` - MongoDB database name (default: `clinical_synthesis_hub`)
- `FHIR_SERVICE_URL` - FHIR service URL (default: `http://localhost:8004`)
- `AUTH_SERVICE_URL` - Auth service URL (default: `http://localhost:8001/api`)

## Examples

### Create a Condition

```bash
curl -X POST http://localhost:8019/api/conditions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "clinicalStatus": {
      "coding": [
        {
          "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
          "code": "active",
          "display": "Active"
        }
      ]
    },
    "verificationStatus": {
      "coding": [
        {
          "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
          "code": "confirmed",
          "display": "Confirmed"
        }
      ]
    },
    "category": [
      {
        "coding": [
          {
            "system": "http://terminology.hl7.org/CodeSystem/condition-category",
            "code": "problem-list-item",
            "display": "Problem List Item"
          }
        ]
      }
    ],
    "code": {
      "coding": [
        {
          "system": "http://snomed.info/sct",
          "code": "73211009",
          "display": "Diabetes mellitus"
        }
      ],
      "text": "Diabetes mellitus"
    },
    "subject": {
      "reference": "Patient/123"
    },
    "onsetDateTime": "2023-01-15",
    "recordedDate": "2023-01-15"
  }'
```

### Get Conditions for a Patient

```bash
curl -X GET http://localhost:8019/api/conditions/patient/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Integration with Other Services

The Condition Microservice integrates with the following services:

- **FHIR Service**: For storing and retrieving FHIR resources
- **Auth Service**: For authentication and authorization
- **GraphQL Gateway**: For providing a unified API to the frontend

## Architecture

The Condition Microservice follows a layered architecture:

1. **API Layer**: REST endpoints for condition data access
2. **Service Layer**: Business logic for condition data processing
3. **Integration Layer**: Communication with the FHIR server
