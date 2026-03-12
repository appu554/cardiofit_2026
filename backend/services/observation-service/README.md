# Observation Microservice

A GraphQL-based microservice for managing clinical observations with Apollo Federation support. This service follows the FHIR Observation resource model and can be integrated into a federated GraphQL architecture.

## Features

- **Federated GraphQL API** - Apollo Federation support for distributed GraphQL
- **FHIR Compliance** - Implements the FHIR Observation resource model
- **Real-time Updates** - Subscription support for observation changes
- **Scalable** - Built with FastAPI and MongoDB for horizontal scaling
- **Containerized** - Easy deployment with Docker and Kubernetes

## Prerequisites

- Python 3.12+
- MongoDB 6.0+
- Docker & Docker Compose (for containerized deployment)

## Observation Categories

The service supports the following observation categories:

- **Laboratory**: Lab test results (CBC, BMP, CMP, Lipid Panel, etc.)
- **Vital Signs**: Blood pressure, heart rate, respiratory rate, temperature, etc.
- **Physical Measurements**: Height, weight, BMI, etc.
- **Social History**: Smoking status, alcohol use, etc.
- **Imaging**: Simple imaging findings
- **Survey**: Questionnaire responses, assessment scores, etc.
- **Therapy**: Therapy-related observations
- **Activity**: Activity-related observations

## GraphQL API

The service provides a GraphQL API with the following key features:

- **Queries**
  - `observation(id: ID!)`: Get a single observation by ID
  - `observations(...)`: Search observations with filtering and pagination
  - `patientObservations(patientId: ID!, ...)`: Get observations for a specific patient
  - `observationsByCategory(category: String!, ...)`: Get observations by category

- **Mutations**
  - `createObservation(input: CreateObservationInput!)`: Create a new observation
  - `updateObservation(id: ID!, input: UpdateObservationInput!)`: Update an existing observation
  - `deleteObservation(id: ID!)`: Delete an observation (soft delete)

- **Subscriptions**
  - `observationCreated`: Subscribe to new observations
  - `observationUpdated`: Subscribe to observation updates
  - `observationDeleted`: Subscribe to observation deletions

### Apollo Federation

This service is designed to work with Apollo Federation and can be composed with other federated services. The `Observation` type is a federated entity that can be extended by other services.

Example of extending the Observation type in another service:

```graphql
extend type Observation @key(fields: "id") {
  id: ID! @external
  # Additional fields from other services
  relatedData: RelatedDataType @requires(fields: "id")
}
```
- `GET /api/laboratory/cbc` - Get CBC (Complete Blood Count) results
- `GET /api/laboratory/bmp` - Get BMP (Basic Metabolic Panel) results
- `GET /api/laboratory/cmp` - Get CMP (Comprehensive Metabolic Panel) results
- `GET /api/laboratory/lipid-panel` - Get Lipid Panel results
- `GET /api/laboratory/urinalysis` - Get Urinalysis results

### HL7 Endpoints

- `POST /api/hl7/process` - Process any HL7 message
- `POST /api/hl7/oru` - Process HL7 ORU message

## Running the Service

### Using Docker

```bash
docker build -t observation-service .
docker run -p 8007:8007 observation-service
```

### Using Python

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8007 --reload
```

## Environment Variables

- `MONGODB_URL` - MongoDB connection URL (default: `mongodb://localhost:27017`)
- `MONGODB_DB_NAME` - MongoDB database name (default: `observation_service`)
- `FHIR_SERVICE_URL` - FHIR service URL (default: `http://localhost:8004/api`)
- `AUTH_SERVICE_URL` - Auth service URL (default: `http://localhost:8001/api`)

## Examples

### Create an Observation

```bash
curl -X POST http://localhost:8007/api/observations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "status": "final",
    "category": "laboratory",
    "code": {
      "system": "http://loinc.org",
      "code": "718-7",
      "display": "Hemoglobin [Mass/volume] in Blood"
    },
    "subject": {
      "reference": "Patient/123"
    },
    "effective_datetime": "2023-06-15T08:00:00",
    "value_quantity": {
      "value": 14.5,
      "unit": "g/dL",
      "system": "http://unitsofmeasure.org",
      "code": "g/dL"
    }
  }'
```

### Get Observations for a Patient

```bash
curl -X GET http://localhost:8007/api/observations/patient/123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Process an HL7 ORU Message

```bash
curl -X POST http://localhost:8007/api/hl7/process \
  -H "Content-Type: application/json" \
## Integration with Other Services

The Observation Microservice integrates with the following services:

- **FHIR Service**: For storing and retrieving FHIR resources
- **Auth Service**: For authentication and authorization
- **GraphQL Gateway**: For providing a unified API to the frontend

## Development

### Local Development

1. Clone the repository
2. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```
3. Set up environment variables (copy `.env.example` to `.env` and update values)
4. Run the service:
   ```bash
   uvicorn app.main:app --reload
   ```

### Docker Compose

```bash
docker-compose up -d
```

This will start:
- Observation Service on port 8007
- MongoDB on port 27017

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Runtime environment (development, staging, production) | `development` |
| `MONGODB_URL` | MongoDB connection string | `mongodb://localhost:27017` |
| `MONGODB_DB` | MongoDB database name | `observation-service` |
| `USE_GOOGLE_HEALTHCARE_API` | Enable Google Healthcare API integration | `false` |
| `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID | - |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to Google Cloud credentials | - |

## Architecture

The Observation Microservice follows a layered architecture:

1. **API Layer**: REST endpoints for observation data access
2. **Service Layer**: Business logic for observation data processing
3. **Data Layer**: MongoDB storage for observation data
4. **Integration Layer**: Communication with other services
