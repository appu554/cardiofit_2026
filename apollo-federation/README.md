# Apollo Federation Server for Clinical Synthesis Hub

This project implements an Apollo Federation server that connects to your microservices and provides a unified GraphQL API. It follows the architecture:

```
API Gateway > Auth > Apollo Federation > Microservices > Google Healthcare API
```

## Features

- **Apollo Federation**: Combines GraphQL schemas from multiple microservices
- **Authentication Forwarding**: Passes authentication tokens to microservices
- **Direct FHIR Integration**: Connects to microservices that use Google Healthcare API
- **Scalable Architecture**: Each service can be deployed and scaled independently
- **Health Monitoring**: Provides health check endpoints for monitoring service status
- **Comprehensive Logging**: Detailed logging for debugging and monitoring
- **Error Handling**: Robust error handling with appropriate error responses

## Prerequisites

- Node.js 16+
- npm or yarn
- Docker and Docker Compose (optional, for containerized deployment)

## Getting Started

### 1. Install Dependencies

```bash
npm install
```

### 2. Configure Environment Variables

Copy the example environment file and update it with your settings:

```bash
cp .env.example .env
```

Edit the `.env` file to set your microservice URLs and other configuration.

### 3. Start the Federation Server

```bash
npm start
```

The server will be available at http://localhost:4000/graphql

You can also access the health check endpoint at http://localhost:4000/health and metrics at http://localhost:4000/metrics

### 4. Using Docker Compose (Optional)

To run the entire system using Docker Compose:

```bash
docker-compose up -d
```

This will start the Apollo Federation server and the Patient service.

## Architecture

### Apollo Federation Server

The main federation server (`index.js`) combines schemas from all microservices and handles authentication forwarding.

### Microservice Integration

Each microservice provides a GraphQL API that the federation server can consume. The federation server combines these APIs into a unified schema.

### Federation Endpoints

Each microservice provides a dedicated federation endpoint that bypasses authentication to allow the Federation Gateway to introspect the schema:

- Patient Service: `/api/federation`
- Observation Service: `/api/federation` (to be implemented)
- Condition Service: `/api/federation` (to be implemented)
- Medication Service: `/api/federation` (to be implemented)
- Encounter Service: `/api/federation` (to be implemented)

These endpoints are used only for schema introspection by the Federation Gateway. All actual GraphQL operations are routed through the authenticated `/api/graphql` endpoints.

### Authentication Flow

1. Client sends a request with an authentication token to the API Gateway
2. API Gateway forwards the request to the Apollo Federation server
3. Apollo Federation server extracts user information from the token
4. Apollo Federation server forwards the request to the appropriate microservice with the token and user info
5. Microservice validates the token and processes the request
6. Microservice communicates with Google Healthcare API
7. Response flows back through the chain to the client

## Adding More Services

To add more services to the federation:

1. Create a schema file in the `schemas` directory
2. Create a resolver file in the `resolvers` directory
3. Create a service file in the `services` directory
4. Add the service to the `serviceList` in `index.js`
5. Add the service to `docker-compose.yml` if using Docker

## Example Queries

### Get All Patients

```graphql
query GetPatients($page: Int) {
  patients(page: $page) {
    items {
      id
      resourceType
      name {
        family
        given
      }
      gender
      birthDate
    }
    total
    page
    count
  }
}
```

### Get Patient by ID

```graphql
query GetPatient($id: String!) {
  patient(id: $id) {
    id
    resourceType
    name {
      family
      given
    }
    gender
    birthDate
    generalPractitioner {
      reference
      display
    }
  }
}
```

### Create Patient

```graphql
mutation CreatePatient($patientData: PatientInput!) {
  createPatient(patientData: $patientData) {
    id
    resourceType
    name {
      family
      given
    }
  }
}
```

## Troubleshooting

### Service Discovery Issues

If the federation server can't discover your microservices:

1. Check that the microservice URLs in `.env` are correct
2. Ensure the microservices are running and accessible
3. Check that the microservices are exposing their GraphQL schemas correctly
4. Verify that the federation endpoints (`/api/federation`) are accessible and not requiring authentication
5. Check the logs for any connection errors
6. Use the `/health` endpoint to verify the status of all services

You can test a federation endpoint directly by visiting:
```
http://localhost:8003/api/federation/playground
```

### Authentication Issues

If you're having authentication problems:

1. Verify that the token is being passed correctly from the API Gateway
2. Check that the federation server is forwarding the token to microservices
3. Ensure the microservices are validating the token correctly
4. Verify that the JWT_SECRET is correctly set in the .env file
5. Check the logs for any authentication errors

### Schema Errors

If you're having issues with the GraphQL schema:

1. Ensure all services are using compatible versions of Apollo Federation
2. Check that entity types have proper @key directives
3. Verify that references between services are correctly defined
4. Use the Apollo Federation Rover CLI to validate your supergraph

## License

This project is licensed under the MIT License.
