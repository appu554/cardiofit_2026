# Lab Service for Clinical Synthesis Hub

This service provides lab data management for the Clinical Synthesis Hub.

## Features

- Process HL7 ORU messages to extract lab data
- Store lab tests and panels
- Expose REST API for lab data
- Integration with FHIR service

## API Endpoints

### HL7 Endpoints

- `POST /api/hl7/process` - Process any HL7 message
- `POST /api/hl7/oru` - Process HL7 ORU message

### Lab Endpoints

- `POST /api/lab/tests` - Create a new lab test
- `GET /api/lab/tests/{test_id}` - Get a lab test by ID
- `GET /api/lab/tests` - Search for lab tests
- `POST /api/lab/panels` - Create a new lab panel
- `GET /api/lab/panels/{panel_id}` - Get a lab panel by ID
- `GET /api/lab/panels` - Search for lab panels
- `GET /api/lab/patient/{patient_id}/tests` - Get lab tests for a patient
- `GET /api/lab/patient/{patient_id}/panels` - Get lab panels for a patient

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

## Environment Variables

- `MONGODB_URL` - MongoDB connection URL (default: `mongodb://localhost:27017`)
- `MONGODB_DB_NAME` - MongoDB database name (default: `lab_service`)
- `FHIR_SERVICE_URL` - FHIR service URL (default: `http://localhost:8004/api`)
- `AUTH_SERVICE_URL` - Auth service URL (default: `http://localhost:8001/api`)

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
