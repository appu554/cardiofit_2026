# FHIR Microservice

This microservice provides a RESTful API for managing FHIR (Fast Healthcare Interoperability Resources) resources. It serves as the central integration layer for the Clinical Synthesis Hub, providing standardized interfaces to other microservices and handling authentication and authorization for FHIR resources.

## Features

- RESTful API for FHIR resources (Patient, Observation, Condition, MedicationRequest, DiagnosticReport, etc.)
- MongoDB Atlas integration for data persistence
- Authentication and authorization using Auth0
- Comprehensive error handling
- Pagination support for search operations
- Patient timeline aggregation

## Prerequisites

- Python 3.11 or higher
- MongoDB Atlas account
- Auth0 account (optional, for authentication)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/clinical-synthesis-hub.git
   cd clinical-synthesis-hub/services/fhir-service
   ```

2. Install dependencies:
   ```
   pip install -r requirements.txt
   ```

3. Create a `.env` file in the `services/fhir-service` directory with the following content:
   ```
   # MongoDB Configuration
   MONGODB_URI=mongodb+srv://admin:<your_password>@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0

   # Auth0 Configuration
   AUTH0_DOMAIN=your-auth0-domain.auth0.com
   AUTH0_API_AUDIENCE=your-api-audience

   # Service URLs
   FHIR_SERVICE_URL=http://localhost:8004
   PATIENT_SERVICE_URL=http://localhost:8003
   NOTES_SERVICE_URL=http://localhost:8005
   LABS_SERVICE_URL=http://localhost:8006
   MEDICATION_SERVICE_URL=http://localhost:8007
   IMAGING_SERVICE_URL=http://localhost:8008
   PROBLEM_LIST_SERVICE_URL=http://localhost:8009
   TIMELINE_SERVICE_URL=http://localhost:8010
   ```

   Replace `<your_password>` with your actual MongoDB Atlas password.

## Running the Service

### Local Development

```
cd services/fhir-service
uvicorn app.main:app --reload --port 8004
```

### Using Docker

```
docker-compose up fhir-service
```

## API Documentation

Once the service is running, you can access the API documentation at:

- Swagger UI: http://localhost:8004/api/docs
- ReDoc: http://localhost:8004/api/redoc

## API Endpoints

### Health Check

- `GET /health` - Check if the service is running

### Generic FHIR Resource Endpoints

- `POST /api/fhir/{resource_type}` - Create a new FHIR resource
- `GET /api/fhir/{resource_type}/{id}` - Get a FHIR resource by ID
- `PUT /api/fhir/{resource_type}/{id}` - Update a FHIR resource
- `DELETE /api/fhir/{resource_type}/{id}` - Delete a FHIR resource
- `GET /api/fhir/{resource_type}` - Search for FHIR resources

### Patient-Specific Endpoints

- `GET /api/fhir/Patient/{id}` - Get a patient by ID
- `GET /api/fhir/Patient` - Search for patients
- `GET /api/fhir/Patient/{id}/Observation` - Get observations for a patient
- `GET /api/fhir/Patient/{id}/Condition` - Get conditions for a patient
- `GET /api/fhir/Patient/{id}/MedicationRequest` - Get medication requests for a patient
- `GET /api/fhir/Patient/{id}/DiagnosticReport` - Get diagnostic reports for a patient
- `GET /api/fhir/Patient/{id}/Encounter` - Get encounters for a patient
- `GET /api/fhir/Patient/{id}/DocumentReference` - Get document references for a patient
- `GET /api/fhir/Patient/{id}/timeline` - Get a patient's timeline

## Testing

### Using Postman

1. Import the Postman collection from `FHIR_Microservice_Postman_Collection.json`
2. Set the `base_url` variable to `http://localhost:8004`
3. Set the `auth_token` variable to your Auth0 token
4. Run the requests

### Using Curl

```bash
# Health check
curl -X GET http://localhost:8004/health

# Create a patient
curl -X POST http://localhost:8004/api/fhir/Patient \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "Patient",
    "identifier": [
      {
        "system": "http://example.org/fhir/ids",
        "value": "123456"
      }
    ],
    "name": [
      {
        "family": "Smith",
        "given": ["John"]
      }
    ],
    "gender": "male",
    "birthDate": "1970-01-01"
  }'

# Get a patient
curl -X GET http://localhost:8004/api/fhir/Patient/PATIENT_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## MongoDB Atlas Integration

This service uses MongoDB Atlas for data persistence. Each FHIR resource type is stored in its own collection in the database. The connection to MongoDB Atlas is established using the `MONGODB_URI` environment variable.

### Database Structure

- Each FHIR resource type has its own collection (e.g., "Patient", "Observation", "Condition")
- MongoDB's document structure is well-suited for FHIR resources, which are JSON-based
- Indexes are created on commonly searched fields for better performance

## Architecture

The FHIR Microservice serves as the central integration layer for the Clinical Synthesis Hub. It provides a standardized interface for managing FHIR resources and routes requests to the appropriate resource-specific microservices.

### Components

- **API Layer**: Handles HTTP requests and responses
- **Service Layer**: Contains business logic for FHIR resources
- **Database Layer**: Manages data persistence using MongoDB Atlas
- **Integration Layer**: Routes requests to resource-specific microservices

### Integration with Other Microservices

The FHIR Microservice integrates with the following microservices:

- **Patient Service**: Manages patient demographics and identifiers
- **Observation Service**: Handles lab results, vital signs
- **Clinical Notes Service**: Manages clinical documents and notes
- **Medication Service**: Handles medications and prescriptions
- **Imaging Service**: Manages imaging studies and reports
- **Problem List Service**: Handles conditions/problems
- **Timeline Service**: Aggregates events across resources

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature-name`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin feature/your-feature-name`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
