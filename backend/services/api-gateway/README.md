# API Gateway for Clinical Synthesis Hub

This API Gateway provides a centralized entry point for all microservices in the Clinical Synthesis Hub.

## Features

- **Centralized Authentication**: All requests are authenticated through the Auth Service
- **Role-Based Access Control (RBAC)**: Permissions are enforced based on user roles
- **Request Routing**: Requests are routed to the appropriate microservice
- **GraphQL Support**: GraphQL queries are supported for complex data requirements
- **Request Logging**: Comprehensive logging of requests and responses
- **Rate Limiting**: Protection against excessive requests

## Architecture

The API Gateway acts as a reverse proxy for all microservices:

```
Client → API Gateway → Microservices
```

1. Client sends a request to the API Gateway
2. API Gateway authenticates the request with the Auth Service
3. API Gateway checks if the user has the required permissions
4. API Gateway forwards the request to the appropriate microservice
5. Microservice processes the request and returns a response
6. API Gateway forwards the response back to the client

## Configuration

The API Gateway is configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| AUTH_SERVICE_URL | URL of the Auth Service | http://localhost:8001 |
| FHIR_SERVICE_URL | URL of the FHIR Service | http://localhost:8004 |
| PATIENT_SERVICE_URL | URL of the Patient Service | http://localhost:8003 |
| OBSERVATION_SERVICE_URL | URL of the Observation Service | http://localhost:8008 |
| MEDICATION_SERVICE_URL | URL of the Medication Service | http://localhost:8009 |
| CONDITION_SERVICE_URL | URL of the Condition Service | http://localhost:8010 |
| ENCOUNTER_SERVICE_URL | URL of the Encounter Service | http://localhost:8011 |
| TIMELINE_SERVICE_URL | URL of the Timeline Service | http://localhost:8012 |
| ENABLE_REQUEST_LOGGING | Enable request logging | True |
| LOG_REQUEST_BODY | Log request bodies | False |
| LOG_RESPONSE_BODY | Log response bodies | False |
| RATE_LIMIT_ENABLED | Enable rate limiting | False |
| RATE_LIMIT_REQUESTS | Maximum requests per window | 100 |
| RATE_LIMIT_WINDOW | Rate limit window in seconds | 60 |

## API Endpoints

### GraphQL

- **GET/POST /graphql**: GraphQL endpoint for complex queries

### Authentication

- **POST /api/auth/login**: Login with username and password
- **POST /api/auth/token**: Get a token using client credentials
- **POST /api/auth/authorize**: Get authorization URL for OAuth flow
- **POST /api/auth/callback**: Exchange authorization code for tokens
- **POST /api/auth/verify**: Verify a token

### Health Check

- **GET /health**: Check the health of the API Gateway

## Running the API Gateway

### Using Docker

```bash
docker build -t api-gateway .
docker run -p 8005:8000 api-gateway
```

### Using Python

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8005 --reload
```

## Development

### Adding a New Service

1. Add the service URL to `app/config.py`
2. Add the service route to `app/api/proxy.py`
3. Add any required permissions to `app/middleware/rbac.py`

### Testing

```bash
pytest
```
