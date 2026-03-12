# Apollo Federation Gateway Flow

This document describes the complete flow for GraphQL requests through the system:

```
API Gateway > Auth > Apollo Federation Gateway > Microservices > Google Healthcare API
```

## Authentication Flow

1. Client sends a request with an authentication token to the API Gateway
2. API Gateway authenticates the request using the Auth Service
3. API Gateway forwards the request to the Apollo Federation Gateway with authentication headers
4. Apollo Federation Gateway extracts user information from the headers
5. Apollo Federation Gateway forwards the request to the appropriate microservice with the token and user info
6. Microservice validates the token and processes the request
7. Microservice communicates with Google Healthcare API
8. Response flows back through the chain to the client

## Headers Propagation

The following headers are propagated through the system:

- `Authorization`: The JWT token from the client
- `X-User-ID`: The user ID extracted from the token
- `X-User-Role`: The primary role of the user
- `X-User-Roles`: Comma-separated list of all roles assigned to the user
- `X-User-Permissions`: Comma-separated list of all permissions assigned to the user
- `X-User-Email`: The email of the user (if available)
- `X-User-Name`: The name of the user (if available)

## Federation Architecture

The Apollo Federation Gateway uses the `IntrospectAndCompose` approach to build a unified schema from all the microservices. Each microservice exposes a federation endpoint that:

1. Exposes the GraphQL schema with Federation directives
2. Allows introspection without authentication
3. Implements `__resolveReference` resolvers for entity types

## Microservices

Each microservice should implement:

1. A federation endpoint at `/api/federation` that exposes the schema with Federation directives
2. Authentication using the `HeaderAuthMiddleware` from the shared module
3. Connection to Google Healthcare API for FHIR data storage

## Running the System

To run the complete system, you need to start:

1. Auth Service
2. API Gateway
3. Apollo Federation Gateway
4. All microservices (Patient, Observation, Condition, Medication, Encounter)

## Testing

You can test the system using:

1. Apollo Sandbox at `http://localhost:4000/graphql` (direct access to Federation Gateway)
2. GraphQL endpoint at `http://localhost:8000/api/graphql` (through API Gateway)

## Troubleshooting

If you encounter issues:

1. Check the health endpoint at `http://localhost:4000/health` to see the status of all services
2. Verify that all microservices have implemented the federation endpoint correctly
3. Check the logs of the Apollo Federation Gateway for schema composition errors
4. Verify that authentication headers are being properly propagated

## Example GraphQL Query

```graphql
query {
  patients(limit: 10) {
    items {
      id
      name {
        family
        given
      }
      gender
      birthDate
    }
    total
  }
}
```

This query will:
1. Go through the API Gateway
2. Be authenticated by the Auth Service
3. Be forwarded to the Apollo Federation Gateway
4. Be resolved by the Patient service
5. Fetch data from Google Healthcare API
6. Return the result back through the chain
