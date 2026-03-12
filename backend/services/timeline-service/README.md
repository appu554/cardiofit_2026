# Timeline Microservice

This microservice provides a comprehensive API for aggregating and managing patient timeline data from various clinical data sources (Observations, Medications, Conditions, Encounters, Documents). It doesn't own a database but instead aggregates data from other microservices to construct a patient's longitudinal timeline.

## Features

- Aggregate events from various clinical data sources
- Filter timeline events by date range, event type, and resource type
- Sort timeline events chronologically
- Provide a unified view of a patient's clinical history

## API Endpoints

### Timeline Endpoints

- `GET /api/timeline/patients/{patient_id}` - Get a patient's timeline with optional query parameter filtering
- `POST /api/timeline/patients/{patient_id}/filter` - Filter a patient's timeline using a JSON body for complex filtering

## Running the Service

### Using Python

```bash
cd services/timeline-service
pip install -r requirements.txt
python run.py
```

### Using Docker

```bash
cd services/timeline-service
docker build -t timeline-service .
docker run -p 8010:8010 timeline-service
```

## Configuration

The microservice can be configured using environment variables:

- `PORT` - The port to run the service on (default: 8010)
- `FHIR_SERVICE_URL` - URL of the FHIR service (default: http://localhost:8004)
- `OBSERVATION_SERVICE_URL` - URL of the Observation service (default: http://localhost:8007)
- `CONDITION_SERVICE_URL` - URL of the Condition service (default: http://localhost:8019)
- `MEDICATION_SERVICE_URL` - URL of the Medication service (default: http://localhost:8018)
- `ENCOUNTER_SERVICE_URL` - URL of the Encounter service (default: http://localhost:8020)
- `DOCUMENT_SERVICE_URL` - URL of the Document service (default: http://localhost:8008)

## Testing

### Get a Patient's Timeline

```bash
curl -X GET "http://localhost:8010/api/timeline/patients/123" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Filter a Patient's Timeline

```bash
curl -X POST "http://localhost:8010/api/timeline/patients/123/filter" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "start_date": "2023-01-01T00:00:00Z",
    "end_date": "2023-12-31T23:59:59Z",
    "event_types": ["observation", "medication"],
    "resource_types": ["Observation", "MedicationRequest"]
  }'
```

## Architecture

The Timeline Service follows a layered architecture:

1. **API Layer**: REST endpoints for timeline data access
2. **Service Layer**: Business logic for aggregating and filtering timeline data
3. **Integration Layer**: Communication with other microservices

## Integration with Other Microservices

The Timeline Service integrates with the following microservices:

- **FHIR Service**: Central integration layer for accessing FHIR resources
- **Observation Service**: Provides observation data
- **Condition Service**: Provides condition/problem data
- **Medication Service**: Provides medication-related data
- **Encounter Service**: Provides encounter data
- **Document Service**: Provides clinical document data
