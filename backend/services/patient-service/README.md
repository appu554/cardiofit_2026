# Patient Service

This service provides a RESTful API for managing patient resources in the Clinical Synthesis Hub. It implements the FHIR Patient resource and provides CRUD operations for patient data.

## Features

- FHIR-compliant Patient resource implementation
- MongoDB integration for data persistence
- Authentication and authorization using JWT tokens
- RESTful API for CRUD operations
- Search functionality for patients
- Shared FHIR models for consistent data representation
- Integration with API Gateway > Auth > FHIR flow
- Automatic enforcement of unique FHIR resource IDs
- Automatic detection and fixing of duplicate IDs

## Prerequisites

- Python 3.9 or higher
- MongoDB Atlas account or local MongoDB instance
- Access to the Clinical Synthesis Hub Auth Service

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/Cardiofit/clinical-synthesis-hub-full.git
   cd clinical-synthesis-hub-full/backend/services/patient-service
   ```

2. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

3. Set up environment variables:
   ```bash
   # MongoDB connection string
   export MONGODB_URL="mongodb+srv://username:password@cluster.mongodb.net/database?retryWrites=true&w=majority"
   export MONGODB_DB_NAME="clinical_synthesis_hub"

   # Service URLs
   export FHIR_SERVICE_URL="http://localhost:8014"
   export AUTH_SERVICE_URL="http://localhost:8001/api"
   ```

## Running the Service

Start the service using Uvicorn:

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8003 --reload
```

The service will be available at http://localhost:8003.

## API Endpoints

### FHIR Endpoints

- `GET /api/fhir/Patient`: Search for patients
- `POST /api/fhir/Patient`: Create a new patient
- `GET /api/fhir/Patient/{id}`: Get a patient by ID
- `PUT /api/fhir/Patient/{id}`: Update a patient
- `DELETE /api/fhir/Patient/{id}`: Delete a patient

### Other Endpoints

- `GET /health`: Health check endpoint
- `GET /me`: Get the authenticated user's information

## Authentication

The service uses JWT tokens for authentication. To access protected endpoints, include an `Authorization` header with a Bearer token:

```
Authorization: Bearer <token>
```

Tokens can be obtained from the Auth Service.

## MongoDB Integration

The service uses MongoDB for data persistence. Patient data is stored in the `patients` collection in the specified database.

To configure MongoDB:

1. Set the `MONGODB_URL` environment variable to your MongoDB connection string
2. Set the `MONGODB_DB_NAME` environment variable to your database name

## Shared FHIR Models

The service uses shared FHIR models from the `shared.models` package to ensure consistent data representation across all microservices. These models are based on the FHIR standard and provide validation for FHIR resources.

## Unique ID Enforcement

The service automatically enforces unique FHIR resource IDs to prevent conflicts:

- A unique index is created on the `id` field in the MongoDB collection
- When creating a new resource, if the provided ID already exists, a new UUID is generated
- When the service starts, it automatically detects and fixes any duplicate IDs in the database
- A utility script is provided to manually fix duplicate IDs if needed

### Fixing Duplicate IDs

If you need to manually fix duplicate IDs, you can run the provided script:

```bash
cd backend/services/patient-service
python scripts/fix_duplicate_ids.py
```

## API Gateway Integration

This service is designed to be accessed through the API Gateway, which routes requests to the FHIR service and then to this service. The correct flow is:

```
API Gateway > Auth > FHIR > Patient Service
```

You can test this flow using the Postman collection provided in the `postman` directory:

```
postman/clinical_synthesis_hub_api_gateway_fhir_flow.postman_collection.json
```

## Development

### Project Structure

- `app/`: Main application package
  - `api/`: API endpoints
  - `core/`: Core functionality (config, security)
  - `db/`: Database connection and models
  - `models/`: Pydantic models
  - `services/`: Business logic
  - `main.py`: Application entry point

### Adding New Features

1. Define models in `app/models/`
2. Implement business logic in `app/services/`
3. Add API endpoints in `app/api/`
4. Update tests

## Testing

### Unit Tests

Run unit tests using pytest:

```bash
pytest
```

### Integration Testing with Postman

You can test the service using the Postman collection provided in the `postman` directory:

```
postman/clinical_synthesis_hub_api_gateway_fhir_flow.postman_collection.json
```

This collection includes requests for all CRUD operations on Patient resources through the API Gateway > Auth > FHIR > Patient Service flow.

To use the collection:

1. Import it into Postman
2. Set up the environment variables:
   - `api_gateway_url`: URL of the API Gateway (e.g., `http://localhost:8005`)
   - `supabase_url`: URL of the Supabase instance
   - `supabase_key`: API key for Supabase
   - `doctor_token`: JWT token for a user with doctor role (obtained from the Login request)
   - `patient_id`: ID of a patient to test with

3. Run the requests in the collection to test the CRUD operations

## Troubleshooting

### Common Issues

- **Connection to MongoDB fails**: Check your MongoDB connection string and make sure the MongoDB Atlas IP whitelist includes your IP address.
- **Authentication fails**: Ensure the Auth Service is running and the JWT token is valid.
- **FHIR validation errors**: Check that your patient data conforms to the FHIR Patient resource schema.
- **Duplicate key errors**: If you see errors like `E11000 duplicate key error collection`, it means you have duplicate FHIR resource IDs. Run the `scripts/fix_duplicate_ids.py` script to fix this issue.
- **Database not connected errors**: If you see `Database not connected, cannot get collection patients`, check that MongoDB is running and accessible, and that the connection string is correct.

### Logs to Check

When troubleshooting, check the following log messages:

- `Database status: Connected` - Confirms the MongoDB connection is established
- `Successfully got patients collection` - Confirms the patients collection is accessible
- `FHIR service has a valid MongoDB collection` - Confirms the FHIR service can access the collection
- `Found X groups of duplicate Patient IDs` - Indicates duplicate IDs were detected and fixed

## License

This project is licensed under the MIT License - see the LICENSE file for details.
